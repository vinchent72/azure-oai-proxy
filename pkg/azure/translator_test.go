package azure

import (
	"encoding/json"
	"testing"
)

func TestTranslateResponsesToChatRequestNormalizesResponsesInput(t *testing.T) {
	payload := []byte(`{
		"model":"DeepSeek-V4-Flash",
		"instructions":"base system prompt",
		"stream":true,
		"max_output_tokens":512,
		"parallel_tool_calls":true,
		"tool_choice":"auto",
		"tools":[{"type":"function","name":"lookup"}],
		"input":[
			{
				"type":"message",
				"role":"developer",
				"content":[{"type":"input_text","text":"developer guardrails"}]
			},
			{
				"type":"message",
				"role":"user",
				"content":[{"type":"input_text","text":"hello from user"}]
			}
		]
	}`)

	translated, err := TranslateResponsesToChatRequest(payload)
	if err != nil {
		t.Fatalf("TranslateResponsesToChatRequest returned error: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(translated, &got); err != nil {
		t.Fatalf("unmarshal translated payload: %v", err)
	}

	if got["model"] != "DeepSeek-V4-Flash" {
		t.Fatalf("unexpected model: %#v", got["model"])
	}
	if got["max_tokens"] != float64(512) {
		t.Fatalf("unexpected max_tokens: %#v", got["max_tokens"])
	}
	if got["parallel_tool_calls"] != true {
		t.Fatalf("unexpected parallel_tool_calls: %#v", got["parallel_tool_calls"])
	}
	if got["tool_choice"] != "auto" {
		t.Fatalf("unexpected tool_choice: %#v", got["tool_choice"])
	}

	messages, ok := got["messages"].([]interface{})
	if !ok {
		t.Fatalf("messages is not an array: %#v", got["messages"])
	}
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	assertMessage := func(index int, role, content string) {
		t.Helper()
		msg, ok := messages[index].(map[string]interface{})
		if !ok {
			t.Fatalf("message %d is not an object: %#v", index, messages[index])
		}
		if msg["role"] != role {
			t.Fatalf("message %d role = %#v, want %q", index, msg["role"], role)
		}
		if msg["content"] != content {
			t.Fatalf("message %d content = %#v, want %q", index, msg["content"], content)
		}
	}

	assertMessage(0, "system", "base system prompt")
	assertMessage(1, "system", "developer guardrails")
	assertMessage(2, "user", "hello from user")
}

func TestTranslateResponsesToChatRequestNormalizesRootMessages(t *testing.T) {
	payload := []byte(`{
		"model":"DeepSeek-V4-Flash",
		"messages":[
			{"role":"developer","content":"developer note"},
			{"role":"assistant","content":[{"type":"output_text","text":"prior answer"}]}
		],
		"input":"next question"
	}`)

	translated, err := TranslateResponsesToChatRequest(payload)
	if err != nil {
		t.Fatalf("TranslateResponsesToChatRequest returned error: %v", err)
	}

	var got struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(translated, &got); err != nil {
		t.Fatalf("unmarshal translated payload: %v", err)
	}

	if len(got.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(got.Messages))
	}

	if got.Messages[0].Role != "system" || got.Messages[0].Content != "developer note" {
		t.Fatalf("unexpected first message: %#v", got.Messages[0])
	}
	if got.Messages[1].Role != "assistant" || got.Messages[1].Content != "prior answer" {
		t.Fatalf("unexpected second message: %#v", got.Messages[1])
	}
	if got.Messages[2].Role != "user" || got.Messages[2].Content != "next question" {
		t.Fatalf("unexpected third message: %#v", got.Messages[2])
	}
}
