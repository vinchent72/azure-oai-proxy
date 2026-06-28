package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleResponsesCompatibilitySanitizesGpt5MiniToolSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := `{
		"model":"gpt-5-mini-2025-08-07",
		"tool_choice":{"type":"function","name":"tool_search"},
		"parallel_tool_calls":true,
		"tools":[
			{"type":"function","name":"tool_search","parameters":{"type":"object","properties":{"query":{"type":"string"}}}},
			{"type":"function","name":"exec_command","parameters":{"type":"object","properties":{"cmd":{"type":"string"}}}}
		],
		"input":"hello"
	}`

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	context.Request = request

	handled := handleResponsesCompatibility(context)
	if handled {
		t.Fatalf("expected filtered-agent request to continue through proxy handling")
	}
	if context.Request.URL.Path != "/v1/responses" {
		t.Fatalf("unexpected request path rewrite: %q", context.Request.URL.Path)
	}

	restoredBody, err := io.ReadAll(context.Request.Body)
	if err != nil {
		t.Fatalf("failed reading restored request body: %v", err)
	}

	got := string(restoredBody)
	if strings.Contains(got, `"name":"tool_search"`) {
		t.Fatalf("expected tool_search to be removed, got %q", got)
	}
	if strings.Contains(got, `"tool_choice"`) {
		t.Fatalf("expected tool_choice to be removed, got %q", got)
	}
	if strings.Contains(got, `"parallel_tool_calls"`) {
		t.Fatalf("expected parallel_tool_calls to be removed, got %q", got)
	}
	if !strings.Contains(got, `"name":"exec_command"`) {
		t.Fatalf("expected exec_command tool to remain, got %q", got)
	}
}
