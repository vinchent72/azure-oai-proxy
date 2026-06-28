package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleResponsesCompatibilityLeavesFullAgentBodiesUntouched(t *testing.T) {
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
		t.Fatalf("expected full-agent request to pass through without early handling")
	}
	if context.Request.URL.Path != "/v1/responses" {
		t.Fatalf("unexpected request path rewrite: %q", context.Request.URL.Path)
	}

	restoredBody, err := io.ReadAll(context.Request.Body)
	if err != nil {
		t.Fatalf("failed reading restored request body: %v", err)
	}
	if string(restoredBody) != body {
		t.Fatalf("expected body to be preserved exactly, got %q", string(restoredBody))
	}
}
