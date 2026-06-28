package azure

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSanitizeResponsesRequestFilteredAgentDropsUnsupportedTools(t *testing.T) {
	payload := []byte(`{
		"model":"grok-4.3",
		"tool_choice":{"type":"function","name":"exec_command"},
		"parallel_tool_calls":true,
		"tools":[
			{"type":"image_generation"},
			{"type":"function","name":"exec_command","parameters":{"type":"object","properties":{"cmd":{"type":"string"}}}}
		],
		"input":"hello"
	}`)

	sanitized, report, err := SanitizeResponsesRequest(payload)
	if err != nil {
		t.Fatalf("SanitizeResponsesRequest returned error: %v", err)
	}

	if report.Mode != CompatibilityModeFilteredAgent {
		t.Fatalf("unexpected compatibility mode: %q", report.Mode)
	}
	if len(report.DroppedTools) != 1 || report.DroppedTools[0] != "image_generation" {
		t.Fatalf("unexpected dropped tools: %#v", report.DroppedTools)
	}
	if report.DroppedChoice {
		t.Fatalf("did not expect tool_choice to be dropped")
	}

	var got map[string]interface{}
	if err := json.Unmarshal(sanitized, &got); err != nil {
		t.Fatalf("unmarshal sanitized payload: %v", err)
	}

	tools, ok := got["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatalf("expected 1 remaining tool, got %#v", got["tools"])
	}
}

func TestSanitizeResponsesRequestGpt5MiniDropsToolSearch(t *testing.T) {
	payload := []byte(`{
		"model":"gpt-5-mini-2025-08-07",
		"tool_choice":{"type":"function","name":"tool_search"},
		"parallel_tool_calls":true,
		"tools":[
			{"type":"function","name":"tool_search","parameters":{"type":"object","properties":{"query":{"type":"string"}}}},
			{"type":"function","name":"exec_command","parameters":{"type":"object","properties":{"cmd":{"type":"string"}}}}
		],
		"input":"hello"
	}`)

	sanitized, report, err := SanitizeResponsesRequest(payload)
	if err != nil {
		t.Fatalf("SanitizeResponsesRequest returned error: %v", err)
	}

	if report.Mode != CompatibilityModeFilteredAgent {
		t.Fatalf("unexpected compatibility mode: %q", report.Mode)
	}
	if len(report.DroppedTools) != 1 || report.DroppedTools[0] != "tool_search" {
		t.Fatalf("unexpected dropped tools: %#v", report.DroppedTools)
	}
	if !report.DroppedChoice {
		t.Fatalf("expected tool_choice to be dropped")
	}

	var got map[string]interface{}
	if err := json.Unmarshal(sanitized, &got); err != nil {
		t.Fatalf("unmarshal sanitized payload: %v", err)
	}

	tools, ok := got["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatalf("expected 1 remaining tool, got %#v", got["tools"])
	}

	tool, ok := tools[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected remaining tool to be an object, got %#v", tools[0])
	}
	if tool["name"] != "exec_command" {
		t.Fatalf("unexpected remaining tool: %#v", tool)
	}

	if _, exists := got["tool_choice"]; exists {
		t.Fatalf("expected tool_choice to be removed, got %#v", got["tool_choice"])
	}
	if _, exists := got["parallel_tool_calls"]; exists {
		t.Fatalf("expected parallel_tool_calls to be removed when tool_choice is dropped, got %#v", got["parallel_tool_calls"])
	}
}

func TestSanitizeResponsesRequestPlainChatDropsToolsAndChoice(t *testing.T) {
	payload := []byte(`{
		"model":"DeepSeek-V4-Flash",
		"tool_choice":{"type":"function","name":"exec_command"},
		"parallel_tool_calls":true,
		"tools":[
			{"type":"function","name":"exec_command","parameters":{"type":"object","properties":{"cmd":{"type":"string"}}}},
			{"type":"web_search"}
		],
		"input":"hello"
	}`)

	sanitized, report, err := SanitizeResponsesRequest(payload)
	if err != nil {
		t.Fatalf("SanitizeResponsesRequest returned error: %v", err)
	}

	if report.Mode != CompatibilityModePlainChat {
		t.Fatalf("unexpected compatibility mode: %q", report.Mode)
	}
	if len(report.DroppedTools) != 2 {
		t.Fatalf("expected 2 dropped tools, got %#v", report.DroppedTools)
	}
	if !report.DroppedChoice {
		t.Fatalf("expected tool_choice to be dropped")
	}

	var got map[string]interface{}
	if err := json.Unmarshal(sanitized, &got); err != nil {
		t.Fatalf("unmarshal sanitized payload: %v", err)
	}

	if _, exists := got["tools"]; exists {
		t.Fatalf("expected tools to be removed, got %#v", got["tools"])
	}
	if _, exists := got["tool_choice"]; exists {
		t.Fatalf("expected tool_choice to be removed, got %#v", got["tool_choice"])
	}
	if _, exists := got["parallel_tool_calls"]; exists {
		t.Fatalf("expected parallel_tool_calls to be removed, got %#v", got["parallel_tool_calls"])
	}
}

func TestSanitizeResponsesRequestFullAgentLeavesOpenAIModelsUntouched(t *testing.T) {
	payload := []byte(`{
		"model":"gpt-5.4-mini",
		"tool_choice":{"type":"function","name":"exec_command"},
		"parallel_tool_calls":true,
		"tools":[
			{"type":"image_generation"},
			{"type":"function","name":"exec_command","parameters":{"type":"object","properties":{"cmd":{"type":"string"}}}}
		],
		"input":[
			{"role":"user","content":[{"type":"input_text","text":"hello"}]}
		]
	}`)

	sanitized, report, err := SanitizeResponsesRequest(payload)
	if err != nil {
		t.Fatalf("SanitizeResponsesRequest returned error: %v", err)
	}

	if report.Mode != CompatibilityModeFullAgent {
		t.Fatalf("unexpected compatibility mode: %q", report.Mode)
	}
	if len(report.DroppedTools) != 0 {
		t.Fatalf("did not expect dropped tools, got %#v", report.DroppedTools)
	}
	if report.DroppedChoice {
		t.Fatalf("did not expect tool_choice to be dropped")
	}

	var want map[string]interface{}
	if err := json.Unmarshal(payload, &want); err != nil {
		t.Fatalf("unmarshal original payload: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(sanitized, &got); err != nil {
		t.Fatalf("unmarshal sanitized payload: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected sanitizer to leave full-agent payload unchanged, got %#v want %#v", got, want)
	}
}
