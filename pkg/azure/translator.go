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

	// 1. Extract instructions if present (map to system message context)
	if inst, ok := src["instructions"].(string); ok && inst != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": inst,
		})
	}

	// 2. If Codex CLI passes a root-level "messages" array, preserve all of them
	if msgsRaw, ok := src["messages"].([]interface{}); ok {
		for _, m := range msgsRaw {
			if msgMap, ok := m.(map[string]interface{}); ok {
				messages = append(messages, msgMap)
			}
		}
	}

	// 3. Extract standard standalone "input" string if present
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

	// Map remaining configuration options safely
	if temp, ok := src["temperature"].(float64); ok {
		dst["temperature"] = temp
	}
	if topP, ok := src["top_p"].(float64); ok {
		dst["top_p"] = topP
	}
	// Force backend stream to always ensure uniform chunked transfer encoding pipeline for Codex CLI
	dst["stream"] = true

	if maxTokens, ok := src["max_output_tokens"].(float64); ok {
		dst["max_tokens"] = int(maxTokens)
	} else if maxTok, ok := src["max_tokens"].(float64); ok {
		dst["max_tokens"] = int(maxTok)
	}

	return json.Marshal(dst)
}

// ResponseTranslationWriter captures backend Chat Completion chunks and reformats them to Responses API schema
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
	// Set appropriate chunked streaming headers on the very first incoming data package frame
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
			if w.isStream {
				w.ResponseWriter.Write([]byte("data: [DONE]\n\n"))
			}
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

			// Capture data internally if the client requested a standard single block response
			if !w.isStream {
				w.bodyBuffer.WriteString(deltaText)
				continue
			}

			// Active Stream Formatting (Real-time updates)
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

	// For non-streaming requests, deliver the collected text chunk wrapped as a fast chunk-transfer frame.
	outputText := w.bodyBuffer.String()
	
	responsesAPIObject := map[string]interface{}{
		"id":          "chatcmpl-proxy-generated-id",
		"object":      "response",
		"created_at":  1782607675,
		"model":       w.modelName,
		"status":      "completed",
		"output_text": outputText,
	}
	responsesAPIObject["output"] = []map[string]interface{}{
		{
			"type":          "message",
			"role":          "assistant",
			"finish_reason": "stop",
			"content": []map[string]interface{}{
				{
					"type": "output_text",
					"text": outputText,
				},
			},
		},
	}

	finalBytes, _ := json.Marshal(responsesAPIObject)
	
	// Deliver both structural elements to finalize standard calls via chunk streams
	w.ResponseWriter.Write([]byte("event: response.chunk\n"))
	w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n\n", string(finalBytes))))
	w.ResponseWriter.Write([]byte("data: [DONE]\n\n"))
}