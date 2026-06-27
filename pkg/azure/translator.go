package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// IsChatOnlyModel determines if a model lacks native Responses API capability
func IsChatOnlyModel(model string) bool {
	m := strings.ToLower(model)
	// DeepSeek and standard third-party models on Foundry typically use standard chat/completions
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

	// 1. Build the messages array
	var messages []map[string]interface{}

	// Extract instructions if present (map to system message)
	if inst, ok := src["instructions"].(string); ok && inst != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": inst,
		})
	}

	// Extract input field
	if inputRaw, exists := src["input"]; exists {
		if inputStr, ok := inputRaw.String(); ok {
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": inputStr,
			})
		} else {
			// If input is already an array of structured messages, pass them along
			inputBytes, _ := json.Marshal(inputRaw)
			var inputMsgs []map[string]interface{}
			if err := json.Unmarshal(inputBytes, &inputMsgs); err == nil {
				messages = append(messages, inputMsgs...)
			}
		}
	}

	dst["messages"] = messages

	// 2. Map structural configurations
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
	}

	return json.Marshal(dst)
}

// ResponseTranslationWriter captures backend Chat Completion chunks and reformats them to Responses API schema
type ResponseTranslationWriter struct {
	gin.ResponseWriter
	bodyBuffer *bytes.Buffer
	isStream   bool
	modelName  string
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
		// Buffer unary payload completely to process at completion
		return w.bodyBuffer.Write(b)
	}

	// Handle Streaming Translation (SSE data lines)
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		if line == "" {
			w.ResponseWriter.Write([]byte("\n"))
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			w.ResponseWriter.Write([]byte(line + "\n"))
			continue
		}

		dataContent := strings.TrimPrefix(line, "data: ")
		if dataContent == "[DONE]" {
			w.ResponseWriter.Write([]byte("data: [DONE]\n"))
			continue
		}

		// Convert standard Chat Completion chunk to valid Responses API stream chunk
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
			w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n", string(chunkBytes))))
		}
	}
	return len(b), nil
}

func (w *ResponseTranslationWriter) FlushResponse() {
	if w.isStream {
		return
	}

	// Process Unary (Standard) Response mapping
	var chatResponse map[string]interface{}
	if err := json.Unmarshal(w.bodyBuffer.Bytes(), &chatResponse); err != nil {
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

	// Mirror OpenAI/Azure Responses API contract layout exactly
	responsesAPIObject := map[string]interface{}{
		"id":          chatResponse["id"],
		"object":      "response",
		"created_at":  chatResponse["created"],
		"model":       w.modelName,
		"status":      "completed",
		"output_text": outputText,
		"usage":       chatResponse["usage"],
	}
	if finishReason != "" {
		responsesAPIObject["output"] = []map[string]interface{}{
			{
				"type": "message",
				"role": "assistant",
				"finish_reason": finishReason,
				"content": []map[string]interface{}{
					{
						"type": "output_text",
						"text": outputText,
					},
				},
			},
		}
	}

	finalBytes, _ := json.Marshal(responsesAPIObject)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(finalBytes)))
	w.ResponseWriter.Write(finalBytes)
}