package main

import (
	"errors"
	"fmt"
	"os"
)

type APIKeyMissingError struct{}
type CostFileNotFoundError struct{}
type InjectError struct{ Char rune }
type JSONParseError struct{ Err error }
type OpenAIRequestError struct{ Err error }
type QuitError struct{}
type RerunError struct{}
type UnsupportedModelError struct{ Model string }

func (e APIKeyMissingError) Error() string {
	return "CFOR_OPEN_API_KEY or OPENAI_API_KEY environment variable must be set"
}

func (e CostFileNotFoundError) Error() string {
	return "Cost file not found"
}

func (e InjectError) Error() string {
	return fmt.Sprintf("failed to inject character: %c", e.Char)
}

func (e JSONParseError) Error() string {
	return fmt.Sprintf("JSON unmarshal failed: %v", e.Err)
}

func (e OpenAIRequestError) Error() string {
	return fmt.Sprintf("OpenAI request failed: %v", e.Err)
}

func (q QuitError) Error() string {
	return "quitting"
}

func (q RerunError) Error() string {
	return "rerunning"
}

func (e UnsupportedModelError) Error() string {
	return fmt.Sprintf("Unsupported model: %s", e.Model)
}

func (e UnsupportedModelError) Is(target error) bool {
	return target == e
}

func HandleQuitError(err error) {
	if errors.Is(err, QuitError{}) {
		os.Exit(0)
	}
}
