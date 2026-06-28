package azure

import "strings"

type TargetAPI string
type CompatibilityMode string

const (
	TargetAPIChatCompletions  TargetAPI = "chat_completions"
	TargetAPIResponses        TargetAPI = "responses"
	TargetAPIAnthropicMessage TargetAPI = "anthropic_messages"

	CompatibilityModeFullAgent     CompatibilityMode = "full_agent"
	CompatibilityModeFilteredAgent CompatibilityMode = "filtered_agent"
	CompatibilityModePlainChat     CompatibilityMode = "plain_chat"
)

type ModelAPIProfile struct {
	Model                 string
	UsesAnthropicMessages bool
	PrefersResponsesAPI   bool
	ChatOnly              bool
	CompatibilityMode     CompatibilityMode
	BlockedResponseTools  map[string]bool
}

func (p ModelAPIProfile) UsesAnthropicMessagesForChat() bool {
	return p.UsesAnthropicMessages
}

func (p ModelAPIProfile) RoutesChatCompletionsToResponses() bool {
	return p.PrefersResponsesAPI
}

func (p ModelAPIProfile) RoutesResponsesToChatCompletions() bool {
	return p.ChatOnly
}

func (p ModelAPIProfile) DropsAllResponseTools() bool {
	return p.CompatibilityMode == CompatibilityModePlainChat
}

func (p ModelAPIProfile) BlocksResponseTool(toolName string) bool {
	return toolName != "" && p.BlockedResponseTools[toolName]
}

func ResolveModelAPIProfile(model string) ModelAPIProfile {
	modelLower := strings.ToLower(strings.TrimSpace(model))

	profile := ModelAPIProfile{
		Model:                 model,
		UsesAnthropicMessages: hasAnyPrefix(modelLower, claudeModelPrefixes),
		PrefersResponsesAPI:   hasAnyPrefix(modelLower, responsesModelPrefixes),
		ChatOnly:              hasAnySubstring(modelLower, chatOnlyModelMarkers),
		CompatibilityMode:     CompatibilityModeFullAgent,
		BlockedResponseTools:  make(map[string]bool),
	}

	if profile.ChatOnly {
		profile.CompatibilityMode = CompatibilityModePlainChat
	}

	if hasAnyPrefix(modelLower, filteredAgentModelPrefixes) {
		profile.CompatibilityMode = CompatibilityModeFilteredAgent
	}

	for _, toolName := range blockedResponseToolsByPrefix {
		if hasAnyPrefix(modelLower, toolName.prefixes) {
			for _, blockedTool := range toolName.tools {
				profile.BlockedResponseTools[blockedTool] = true
			}
		}
	}

	return profile
}

func SelectTargetAPI(model string, requestPath string) TargetAPI {
	return selectTargetAPIForProfile(ResolveModelAPIProfile(model), requestPath)
}

func selectTargetAPIForProfile(profile ModelAPIProfile, requestPath string) TargetAPI {
	switch {
	case strings.HasPrefix(requestPath, "/v1/chat/completions"):
		if profile.UsesAnthropicMessagesForChat() {
			return TargetAPIAnthropicMessage
		}
		if profile.RoutesChatCompletionsToResponses() {
			return TargetAPIResponses
		}
		return TargetAPIChatCompletions
	case strings.HasPrefix(requestPath, "/v1/responses"):
		if profile.RoutesResponsesToChatCompletions() {
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

var filteredAgentModelPrefixes = []string{
	"gpt-5-mini",
	"grok-",
}

type blockedResponseToolsRule struct {
	prefixes []string
	tools    []string
}

var blockedResponseToolsByPrefix = []blockedResponseToolsRule{
	{
		prefixes: []string{"gpt-5-mini"},
		tools:    []string{"tool_search"},
	},
	{
		prefixes: []string{"grok-"},
		tools:    []string{"image_generation"},
	},
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
