package azure

import "testing"

func TestResolveModelAPIProfile(t *testing.T) {
	t.Run("claude uses anthropic", func(t *testing.T) {
		profile := ResolveModelAPIProfile("claude-sonnet-4.5")
		if !profile.UsesAnthropicMessages {
			t.Fatalf("expected Claude profile to use Anthropic Messages")
		}
		if profile.PrefersResponsesAPI {
			t.Fatalf("did not expect Claude profile to prefer Responses API")
		}
		if profile.ChatOnly {
			t.Fatalf("did not expect Claude profile to be chat-only")
		}
	})

	t.Run("reasoning model prefers responses", func(t *testing.T) {
		profile := ResolveModelAPIProfile("gpt-5.1-codex")
		if !profile.PrefersResponsesAPI {
			t.Fatalf("expected Codex profile to prefer Responses API")
		}
		if profile.UsesAnthropicMessages {
			t.Fatalf("did not expect Codex profile to use Anthropic Messages")
		}
		if profile.CompatibilityMode != CompatibilityModeFullAgent {
			t.Fatalf("expected Codex profile to stay in full agent mode")
		}
	})

	t.Run("deepseek is chat only", func(t *testing.T) {
		profile := ResolveModelAPIProfile("DeepSeek-V4-Flash")
		if !profile.ChatOnly {
			t.Fatalf("expected DeepSeek profile to be chat-only")
		}
		if profile.CompatibilityMode != CompatibilityModePlainChat {
			t.Fatalf("expected DeepSeek profile to use plain chat compatibility mode")
		}
	})

	t.Run("grok uses filtered agent mode", func(t *testing.T) {
		profile := ResolveModelAPIProfile("grok-4.3")
		if profile.CompatibilityMode != CompatibilityModeFilteredAgent {
			t.Fatalf("expected Grok profile to use filtered agent mode")
		}
		if !profile.BlockedResponseTools["image_generation"] {
			t.Fatalf("expected Grok profile to block image_generation")
		}
	})
}

func TestSelectTargetAPI(t *testing.T) {
	testCases := []struct {
		name       string
		model      string
		requestURL string
		want       TargetAPI
	}{
		{
			name:       "claude chat goes anthropic",
			model:      "claude-sonnet-4.5",
			requestURL: "/v1/chat/completions",
			want:       TargetAPIAnthropicMessage,
		},
		{
			name:       "codex chat goes responses",
			model:      "gpt-5.1-codex",
			requestURL: "/v1/chat/completions",
			want:       TargetAPIResponses,
		},
		{
			name:       "deepseek responses downgrade to chat",
			model:      "DeepSeek-V4-Flash",
			requestURL: "/v1/responses",
			want:       TargetAPIChatCompletions,
		},
		{
			name:       "gpt responses stay responses",
			model:      "gpt-5.4-mini",
			requestURL: "/v1/responses",
			want:       TargetAPIResponses,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := SelectTargetAPI(testCase.model, testCase.requestURL)
			if got != testCase.want {
				t.Fatalf("SelectTargetAPI(%q, %q) = %q, want %q", testCase.model, testCase.requestURL, got, testCase.want)
			}
		})
	}
}
