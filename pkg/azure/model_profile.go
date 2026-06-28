package azure

import "strings"

type TargetAPI string

const (
	TargetAPIChatCompletions  TargetAPI = "chat_completions"
	TargetAPIResponses        TargetAPI = "responses"
	TargetAPIAnthropicMessage TargetAPI = "anthropic_messages"
)

type ModelAPIProfile struct {
	Model                 string
	UsesAnthropicMessages bool
	PrefersResponsesAPI   bool
	ChatOnly              bool
}

func ResolveModelAPIProfile(model string) ModelAPIProfile {
	modelLower := strings.ToLower(strings.TrimSpace(model))

	return ModelAPIProfile{
		Model:                 model,
		UsesAnthropicMessages: hasAnyPrefix(modelLower, claudeModelPrefixes),
		PrefersResponsesAPI:   hasAnyPrefix(modelLower, responsesModelPrefixes),
		ChatOnly:              hasAnySubstring(modelLower, chatOnlyModelMarkers),
	}
}

func SelectTargetAPI(model string, requestPath string) TargetAPI {
	profile := ResolveModelAPIProfile(model)

	switch {
	case strings.HasPrefix(requestPath, "/v1/chat/completions"):
		if profile.UsesAnthropicMessages {
			return TargetAPIAnthropicMessage
		}
		if profile.PrefersResponsesAPI {
			return TargetAPIResponses
		}
		return TargetAPIChatCompletions
	case strings.HasPrefix(requestPath, "/v1/responses"):
		if profile.ChatOnly {
			return TargetAPIChatCompletions
		}
		return TargetAPIResponses
	default:
		return TargetAPIChatCompletions
	}
}

var claudeModelPrefixes = []string{
	"claude-opus", "claude-sonnet", "claude-haiku",
	"claude-3", "claude-4",
}

var responsesModelPrefixes = []string{
	"o1", "o1-preview", "o1-mini",
	"o3", "o3-mini", "o3-pro", "o3-deep-research",
	"o4", "o4-mini",
	"codex-mini",
	"gpt-5.1-codex", "gpt-5-codex",
	"gpt-5-pro",
	"computer-use-preview",
}

var chatOnlyModelMarkers = []string{
	"deepseek", "llama", "qwen",
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func hasAnySubstring(value string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}
