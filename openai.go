package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"slices"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAI client configuration
const (
	timeout          = 10 * time.Second
	temperature      = 0.1
	topP             = 1.0
	presencePenalty  = 0.0
	frequencyPenalty = 0.0
	maxTokens        = 2048
)

// Prompts
const (
	systemPrompt       = "You are a helpful system admin who provides users with commands to execute inside terminal, when asked."
	jsonResponsePrompt = "Return your response as a valid JSON object."
	mainPrompt         = "what is the command for"
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
	// CFOR_OPENAI_API_KEY takes precedence
	apiKey := os.Getenv("CFOR_OPENAI_API_KEY")
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
		MaxTokens:        openai.Int(maxTokens),
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
	model := os.Getenv("CFOR_OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
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
	prompt += fmt.Sprintf("For the **%s** operation system, %s %s?", runtime.GOOS, mainPrompt, question)
	result, err := chatStructured[Cmds](model, prompt, schemaParam)
	if err != nil {
		return ChatResult[Cmds]{}, err
	}

	return result, nil
}

const (
	OpenAIModelGPT4oMini openai.ChatModel = openai.ChatModelGPT4oMini
	OpenAIModelGPT4o     openai.ChatModel = openai.ChatModelGPT4o
)

func IsSupportedModel(model openai.ChatModel) bool {
	return slices.Contains(OpenAISupportedModels, model)
}

// https://openai.com/api/pricing/
const (
	// GPT-4o Mini
	OpenAIModelGPT4oMiniInputCostPerToken       Cost = 2.50 * 1e-6
	OpenAIModelGPT4oMiniCachedInputCostPerToken Cost = 1.25 * 1e-6
	OpenAIModelGPT4oMiniOutputCostPerToken      Cost = 10.00 * 1e-6
	// GPT-4o
	OpenAIModelGPT4oInputCostPerToken       Cost = 0.150 * 1e-6
	OpenAIModelGPT4oCachedInputCostPerToken Cost = 0.075 * 1e-6
	OpenAIModelGPT4oOutputCostPerToken      Cost = 0.670 * 1e-6
)

type CostPerToken struct {
	Input       Cost
	CachedInput Cost
	Output      Cost
}

var OpenAIModelCosts = map[openai.ChatModel]CostPerToken{
	OpenAIModelGPT4oMini: {
		Input:       OpenAIModelGPT4oMiniInputCostPerToken,
		CachedInput: OpenAIModelGPT4oMiniCachedInputCostPerToken,
		Output:      OpenAIModelGPT4oMiniOutputCostPerToken,
	},
	OpenAIModelGPT4o: {
		Input:       OpenAIModelGPT4oInputCostPerToken,
		CachedInput: OpenAIModelGPT4oCachedInputCostPerToken,
		Output:      OpenAIModelGPT4oOutputCostPerToken,
	},
}

var OpenAISupportedModels = []openai.ChatModel{
	OpenAIModelGPT4oMini,
	OpenAIModelGPT4o,
}

func EstimateCost(model openai.ChatModel, usage openai.CompletionUsage) Cost {
	cost := OpenAIModelCosts[model]
	estimatedCost := float64(cost.Input)*float64(usage.PromptTokens) +
		float64(cost.CachedInput)*float64(usage.PromptTokensDetails.CachedTokens) +
		float64(cost.Output)*float64(usage.CompletionTokens)
	return Cost(estimatedCost)
}
