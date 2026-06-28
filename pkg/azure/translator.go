package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// IsChatOnlyModel determines if a model lacks native Responses API capability
func IsChatOnlyModel(model string) bool {
	m := strings.ToLower(model)
	if strings.Contains(m, "deepseek") || strings.Contains(m, "llama") || strings.Contains(m, "qwen") {
		return true
	}
	return false
}

// TranslateResponsesToChatRequest transforms a Responses API body into a standard Chat Completion body
func TranslateResponsesToChatRequest(resBodyBytes []byte) ([]byte, error) {
	var src map[string]interface{}
	if err := json.Unmarshal(resBodyBytes, &src); err != nil {
		return nil, err
	}

	model, _ := src["model"].(string)
	dst := map[string]interface{}{
		"model": model,
	}

	var messages []map[string]interface{}

	// 1. Extract instructions if present
	if inst, ok := src["instructions"].(string); ok && inst != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": inst,
		})
	}

	// 2. Preserve root-level messages if present
	if msgsRaw, ok := src["messages"].([]interface{}); ok {
		for _, m := range msgsRaw {
			if msgMap, ok := m.(map[string]interface{}); ok {
				messages = append(messages, msgMap)
			}
		}
	}

	// 3. Extract standard standalone input
	if inputRaw, exists := src["input"]; exists {
		if inputStr, ok := inputRaw.(string); ok {
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": inputStr,
			})
		} else {
			inputBytes, _ := json.Marshal(inputRaw)
			var inputMsgs []map[string]interface{}
			if err := json.Unmarshal(inputBytes, &inputMsgs); err == nil {
				messages = append(messages, inputMsgs...)
			}
		}
	}

	dst["messages"] = messages

	if temp, ok := src["temperature"].(float64); ok {
		dst["temperature"] = temp
	}
	if topP, ok := src["top_p"].(float64); ok {
		dst["top_p"] = topP
	}
	if stream, ok := src["stream"].(bool); ok {
		dst["stream"] = stream
	}

	if maxTokens, ok := src["max_output_tokens"].(float64); ok {
		dst["max_tokens"] = int(maxTokens)
	} else if maxTok, ok := src["max_tokens"].(float64); ok {
		dst["max_tokens"] = int(maxTok)
	}

	return json.Marshal(dst)
}

// ResponseTranslationWriter captures backend Chat Completion responses and reformats them
type ResponseTranslationWriter struct {
	gin.ResponseWriter
	bodyBuffer *bytes.Buffer
	isStream   bool
	modelName  string
	hasStarted bool
}

func NewResponseTranslationWriter(w gin.ResponseWriter, isStream bool, model string) *ResponseTranslationWriter {
	return &ResponseTranslationWriter{
		ResponseWriter: w,
		bodyBuffer:     bytes.NewBuffer(nil),
		isStream:       isStream,
		modelName:      model,
	}
}

func (w *ResponseTranslationWriter) Write(b []byte) (int, error) {
	if !w.isStream {
		// For standard unary calls, buffer the raw JSON backend response completely
		return w.bodyBuffer.Write(b)
	}

	// Dynamic Streaming Handler (For stream: true)
	if !w.hasStarted {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(200)
		w.hasStarted = true
	}

	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		dataContent := strings.TrimPrefix(line, "data: ")
		if dataContent == "[DONE]" {
			w.ResponseWriter.Write([]byte("data: [DONE]\n\n"))
			continue
		}

		var chatChunk map[string]interface{}
		if err := json.Unmarshal([]byte(dataContent), &chatChunk); err == nil {
			choices, _ := chatChunk["choices"].([]interface{})
			deltaText := ""
			finishReason := interface{}(nil)

			if len(choices) > 0 {
				choice := choices[0].(map[string]interface{})
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					if content, ok := delta["content"].(string); ok {
						deltaText = content
					}
				}
				if fr, ok := choice["finish_reason"]; ok && fr != nil {
					finishReason = fr
				}
			}

			respChunk := map[string]interface{}{
				"id":          chatChunk["id"],
				"object":      "response.chunk",
				"created_at":  chatChunk["created"],
				"model":       w.modelName,
				"output_text": deltaText,
			}
			if finishReason != nil {
				respChunk["status"] = "completed"
				respChunk["finish_reason"] = finishReason
			} else {
				respChunk["status"] = "in_progress"
			}

			chunkBytes, _ := json.Marshal(respChunk)
			w.ResponseWriter.Write([]byte("event: response.chunk\n"))
			w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n\n", string(chunkBytes))))
		}
	}
	return len(b), nil
}

func (w *ResponseTranslationWriter) FlushResponse() {
	if w.isStream {
		return
	}

	var chatResponse map[string]interface{}
	if err := json.Unmarshal(w.bodyBuffer.Bytes(), &chatResponse); err != nil {
		// If parsing fails, fall back to writing the raw backend bytes
		w.ResponseWriter.Write(w.bodyBuffer.Bytes())
		return
	}

	choices, _ := chatResponse["choices"].([]interface{})
	outputText := ""
	finishReason := "stop"

	if len(choices) > 0 {
		choice := choices[0].(map[string]interface{})
		if msg, ok := choice["message"].(map[string]interface{}); ok {
			outputText, _ = msg["content"].(string)
		}
		if fr, ok := choice["finish_reason"].(string); ok && fr != "" {
			finishReason = fr
		}
	}

	// CRITICAL SCHEMA MATCH FOR CODEX CLI UNARY RESPONSES
	responsesAPIObject := map[string]interface{}{
		"id":          chatResponse["id"],
		"object":      "response",
		"created_at":  chatResponse["created"],
		"model":       w.modelName,
		"status":      "completed",
		"output_text": outputText,
		"usage":       chatResponse["usage"],
	}
	responsesAPIObject["output"] = []map[string]interface{}{
		{
			"type":          "message",
			"role":          "assistant",
			"finish_reason": finishReason,
			"content": []map[string]interface{}{
				{
					"type": "output_text",
					"text": outputText,
				},
			},
		},
	}

	finalBytes, _ := json.Marshal(responsesAPIObject)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(finalBytes)))
	w.ResponseWriter.Write(finalBytes)
}