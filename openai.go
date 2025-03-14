package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type APIKeyMissingError struct{}
type JSONParseError struct{ Err error }
type OpenAIRequestError struct{ Err error }
type UnsupportedModelError struct{ Model string }

func (e APIKeyMissingError) Error() string {
	return "CFOR_API_KEY or OPENAI_API_KEY environment variable must be set"
}

func (e OpenAIRequestError) Error() string {
	return fmt.Sprintf("OpenAI request failed: %v", e.Err)
}

func (e JSONParseError) Error() string {
	return fmt.Sprintf("JSON unmarshal failed: %v", e.Err)
}

func (e UnsupportedModelError) Error() string {
	return fmt.Sprintf("Unsupported model: %s", e.Model)
}

func (e UnsupportedModelError) Is(target error) bool {
	return target == e
}

// OpenAI client configuration
const (
	timeout          = 10 * time.Second
	temperature      = 0.3
	topP             = 1.0
	presencePenalty  = 0.0
	frequencyPenalty = 0.0
)

// Prompts
const (
	systemPrompt       = "You are a helpful system admin who provides users with commands to execute inside terminal, when asked."
	jsonResponsePrompt = "Return your response as a valid JSON object."
	mainPrompt         = "What is the command for"
	guidelinePrompt    = `Follow the below guidelines.

## **General Rules**
- **Do**:
  - Provide variations of the command in the order of increasing complexity
  - Append very short, minimal *inline comments* for each command
- **Do not**:
  - Add newlines for comments.
  - Provide any remarks.

`
)

func newClient() (*openai.Client, error) {
	// CFOR_API_KEY takes precedence
	apiKey := os.Getenv("CFOR_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	// If both are missing, return an error
	if apiKey == "" {
		return nil, &APIKeyMissingError{}
	}

	return openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(timeout),
	), nil
}

type ChatResult[T any] struct {
	Message T
	Cost    Cost
}

func GenerateSchema[T any]() any {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

func chatStructured[T any](model, prompt string, schema openai.ResponseFormatJSONSchemaJSONSchemaParam) (ChatResult[T], error) {
	client, err := newClient()
	if err != nil {
		return ChatResult[T]{}, err
	}

	resp, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Model:            openai.F(model),
		Temperature:      openai.Float(temperature),
		TopP:             openai.Float(topP),
		PresencePenalty:  openai.Float(presencePenalty),
		FrequencyPenalty: openai.Float(frequencyPenalty),
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt + jsonResponsePrompt),
			openai.UserMessage(prompt),
		}),
		ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
			openai.ResponseFormatJSONSchemaParam{
				Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
				JSONSchema: openai.F(schema),
			}),
	})
	if err != nil {
		return ChatResult[T]{}, &OpenAIRequestError{Err: err}
	}

	content := resp.Choices[0].Message.Content
	var result T
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return ChatResult[T]{}, &JSONParseError{Err: err}
	}

	return ChatResult[T]{
		Message: result,
		Cost:    EstimateCost(model, resp.Usage),
	}, nil
}

type CmdEntry struct {
	Cmd     string `json:"cmd"`
	Comment string `json:"comment"`
}

type Cmds struct {
	Cmds []CmdEntry `json:"cmds"`
}

var StructuredCmdsSchema = GenerateSchema[Cmds]()

func GenerateCmds(question string) (ChatResult[Cmds], error) {
	model := os.Getenv("CFOR_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	if !IsSupportedModel(model) {
		return ChatResult[Cmds]{}, UnsupportedModelError{Model: model}
	}

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("cmds"),
		Description: openai.F("A list of commands and associated comments to execute."),
		Schema:      openai.F(StructuredCmdsSchema),
		Strict:      openai.Bool(true),
	}

	prompt := guidelinePrompt
	prompt += fmt.Sprintf("%s %s?", mainPrompt, question)
	result, err := chatStructured[Cmds](model, prompt, schemaParam)
	if err != nil {
		return ChatResult[Cmds]{}, err
	}

	return result, nil
}
