package azure

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestTranslateResponsesToChatRequestConvertsTools(t *testing.T) {
	payload := []byte(`{
		"model":"DeepSeek-V4-Flash",
		"tool_choice":{"type":"function","name":"exec_command"},
		"parallel_tool_calls":true,
		"tools":[
			{
				"type":"function",
				"name":"exec_command",
				"description":"Run a shell command",
				"parameters":{"type":"object","properties":{"cmd":{"type":"string"}},"required":["cmd"]}
			},
			{
				"type":"web_search",
				"external_web_access":false
			}
		],
		"input":"hello"
	}`)

	translated, err := TranslateResponsesToChatRequest(payload)
	if err != nil {
		t.Fatalf("TranslateResponsesToChatRequest returned error: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(translated, &got); err != nil {
		t.Fatalf("unmarshal translated payload: %v", err)
	}

	if got["parallel_tool_calls"] != true {
		t.Fatalf("unexpected parallel_tool_calls: %#v", got["parallel_tool_calls"])
	}

	toolChoice, ok := got["tool_choice"].(map[string]interface{})
	if !ok {
		t.Fatalf("tool_choice is not an object: %#v", got["tool_choice"])
	}
	if toolChoice["type"] != "function" {
		t.Fatalf("unexpected tool_choice type: %#v", toolChoice["type"])
	}
	functionChoice, ok := toolChoice["function"].(map[string]interface{})
	if !ok || functionChoice["name"] != "exec_command" {
		t.Fatalf("unexpected tool_choice function: %#v", toolChoice["function"])
	}

	tools, ok := got["tools"].([]interface{})
	if !ok {
		t.Fatalf("tools is not an array: %#v", got["tools"])
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 translated tools, got %d", len(tools))
	}

	firstTool, ok := tools[0].(map[string]interface{})
	if !ok {
		t.Fatalf("first tool is not an object: %#v", tools[0])
	}
	if firstTool["type"] != "function" {
		t.Fatalf("unexpected first tool type: %#v", firstTool["type"])
	}
	firstFunction, ok := firstTool["function"].(map[string]interface{})
	if !ok || firstFunction["name"] != "exec_command" {
		t.Fatalf("unexpected first tool function: %#v", firstTool["function"])
	}

	secondTool, ok := tools[1].(map[string]interface{})
	if !ok {
		t.Fatalf("second tool is not an object: %#v", tools[1])
	}
	secondFunction, ok := secondTool["function"].(map[string]interface{})
	if !ok || secondFunction["name"] != "web_search" {
		t.Fatalf("unexpected second tool function: %#v", secondTool["function"])
	}
}

func TestResponseTranslationWriterPassesThroughStreamingErrors(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Writer.WriteHeader(429)
	context.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")

	writer := NewResponseTranslationWriter(context.Writer, true, "DeepSeek-V4-Flash")
	body := []byte(`{"error":{"code":"RateLimitReached"}}`)

	written, err := writer.Write(body)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if written != len(body) {
		t.Fatalf("Write returned %d, want %d", written, len(body))
	}

	if recorder.Code != 429 {
		t.Fatalf("unexpected status code: got %d want 429", recorder.Code)
	}
	if recorder.Body.String() != string(body) {
		t.Fatalf("unexpected body: %#v", recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %#v", contentType)
	}
}

func TestResponseTranslationWriterSkipsUsageOnlyTailAndDuplicateTerminal(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	writer := NewResponseTranslationWriter(context.Writer, true, "DeepSeek-V4-Flash")
	stream := strings.Join([]string{
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"DeepSeek-V4-Flash","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"DeepSeek-V4-Flash","choices":[{"index":0,"delta":{"content":null},"finish_reason":"stop"}]}`,
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"DeepSeek-V4-Flash","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`,
		`data: [DONE]`,
		"",
	}, "\n")

	if _, err := writer.Write([]byte(stream)); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	body := recorder.Body.String()
	if strings.Count(body, `"status":"completed"`) != 1 {
		t.Fatalf("expected exactly one completed chunk, got body: %s", body)
	}
	if strings.Count(body, `"status":"in_progress"`) != 1 {
		t.Fatalf("expected exactly one in-progress chunk, got body: %s", body)
	}
	if strings.Count(body, `data: [DONE]`) != 1 {
		t.Fatalf("expected [DONE] marker once, got body: %s", body)
	}
}
