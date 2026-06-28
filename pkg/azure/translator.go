package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// IsChatOnlyModel determines if a model lacks native Responses API capability
func IsChatOnlyModel(model string) bool {
	return ResolveModelAPIProfile(model).ChatOnly
}

// TranslateResponsesToChatRequest transforms a Responses API body into a standard Chat Completion body
func TranslateResponsesToChatRequest(resBodyBytes []byte) ([]byte, error) {
	var src map[string]interface{}
	if err := json.Unmarshal(resBodyBytes, &src); err != nil {
		log.Printf("[DEBUG-ERROR] Failed to unmarshal client body: %v\n", err)
		return nil, err
	}

	log.Printf("[DEBUG-INBOUND] Responses payload summary: %s\n", summarizeResponsesPayload(src))

	model, _ := src["model"].(string)
	dst := map[string]interface{}{
		"model": model,
	}

	var messages []map[string]interface{}

	// 1. Extract instructions if present
	if inst, ok := src["instructions"].(string); ok && inst != "" {
		appendTranslatedMessage(&messages, "system", inst)
	}

	// 2. Preserve root-level messages if present
	if msgsRaw, ok := src["messages"].([]interface{}); ok {
		log.Printf("[DEBUG-CONTEXT] Found %d root-level history messages in client payload\n", len(msgsRaw))
		for _, m := range msgsRaw {
			if msgMap, ok := m.(map[string]interface{}); ok {
				appendTranslatedMessage(&messages, getString(msgMap["role"]), msgMap["content"])
			}
		}
	}

	// 3. Extract standard standalone input
	if inputRaw, exists := src["input"]; exists {
		if inputStr, ok := inputRaw.(string); ok {
			appendTranslatedMessage(&messages, "user", inputStr)
		} else {
			if inputMsgs, ok := inputRaw.([]interface{}); ok {
				for _, item := range inputMsgs {
					inputMsg, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					appendTranslatedMessage(&messages, getString(inputMsg["role"]), inputMsg["content"])
				}
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

	if translatedTools := translateResponsesTools(src["tools"]); len(translatedTools) > 0 {
		dst["tools"] = translatedTools
		if toolChoice := translateToolChoice(src["tool_choice"]); toolChoice != nil {
			dst["tool_choice"] = toolChoice
		}
		if parallelToolCalls, ok := src["parallel_tool_calls"].(bool); ok {
			dst["parallel_tool_calls"] = parallelToolCalls
		}
	}

	translatedBytes, _ := json.Marshal(dst)
	log.Printf("[DEBUG-OUTBOUND] Chat payload summary: %s\n", summarizeChatPayload(dst))
	return translatedBytes, nil
}

func appendTranslatedMessage(messages *[]map[string]interface{}, role string, content interface{}) {
	normalizedRole := normalizeChatRole(role)
	if normalizedRole == "" {
		return
	}

	text := extractMessageText(content)
	if text == "" {
		return
	}

	*messages = append(*messages, map[string]interface{}{
		"role":    normalizedRole,
		"content": text,
	})
}

func translateResponsesTools(rawTools interface{}) []map[string]interface{} {
	tools, ok := rawTools.([]interface{})
	if !ok {
		return nil
	}

	translated := make([]map[string]interface{}, 0, len(tools))
	dropped := 0

	for _, item := range tools {
		tool, ok := item.(map[string]interface{})
		if !ok {
			dropped++
			continue
		}

		translatedTool, ok := translateResponsesTool(tool)
		if !ok {
			dropped++
			continue
		}

		translated = append(translated, translatedTool)
	}

	if dropped > 0 {
		log.Printf("[DEBUG-TOOLS] Dropped %d unsupported tool definitions during chat translation\n", dropped)
	}

	return translated
}

func translateResponsesTool(tool map[string]interface{}) (map[string]interface{}, bool) {
	name := strings.TrimSpace(getString(tool["name"]))
	if name == "" {
		name = deriveToolName(tool)
	}
	if name == "" {
		return nil, false
	}

	parameters, ok := tool["parameters"].(map[string]interface{})
	if !ok || parameters == nil {
		parameters = map[string]interface{}{
			"type":                 "object",
			"properties":           map[string]interface{}{},
			"additionalProperties": true,
		}
	}

	description := strings.TrimSpace(getString(tool["description"]))
	if description == "" {
		description = fmt.Sprintf("Proxy-translated tool for %s.", name)
	}

	if originalType := strings.TrimSpace(getString(tool["type"])); originalType != "" && originalType != "function" {
		description = fmt.Sprintf("%s Original Responses tool type: %s.", description, originalType)
	}

	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        name,
			"description": description,
			"parameters":  parameters,
		},
	}, true
}

func deriveToolName(tool map[string]interface{}) string {
	switch strings.TrimSpace(getString(tool["type"])) {
	case "web_search":
		return "web_search"
	case "image_generation":
		return "image_generation"
	default:
		return ""
	}
}

func translateToolChoice(rawChoice interface{}) interface{} {
	switch typed := rawChoice.(type) {
	case string:
		choice := strings.TrimSpace(typed)
		if choice == "" {
			return nil
		}
		return choice
	case map[string]interface{}:
		name := strings.TrimSpace(getString(typed["name"]))
		if name == "" {
			if functionMap, ok := typed["function"].(map[string]interface{}); ok {
				name = strings.TrimSpace(getString(functionMap["name"]))
			}
		}
		if name == "" {
			return nil
		}
		return map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name": name,
			},
		}
	default:
		return nil
	}
}

func normalizeChatRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "developer", "system":
		return "system"
	case "user":
		return "user"
	case "assistant":
		return "assistant"
	case "tool":
		return "tool"
	default:
		return ""
	}
}

func extractMessageText(content interface{}) string {
	switch typed := content.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []interface{}:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			text := extractContentPartText(item)
			if text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n\n")
	case map[string]interface{}:
		return strings.TrimSpace(getString(typed["text"]))
	default:
		return ""
	}
}

func extractContentPartText(part interface{}) string {
	switch typed := part.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]interface{}:
		if text := strings.TrimSpace(getString(typed["text"])); text != "" {
			return text
		}
		if nested := extractMessageText(typed["content"]); nested != "" {
			return nested
		}
		return ""
	default:
		return ""
	}
}

func summarizeResponsesPayload(src map[string]interface{}) string {
	messageCount := countArrayEntries(src["messages"])
	inputCount := countArrayEntries(src["input"])
	toolCount := countArrayEntries(src["tools"])
	fields := make([]string, 0, len(src))
	for key := range src {
		if key == "instructions" || key == "input" || key == "messages" || key == "tools" {
			continue
		}
		fields = append(fields, key)
	}
	sort.Strings(fields)

	return fmt.Sprintf(
		"model=%q instructions_chars=%d input_items=%d root_messages=%d tools=%d stream=%t extra_fields=%v",
		getString(src["model"]),
		len(getString(src["instructions"])),
		inputCount,
		messageCount,
		toolCount,
		getBool(src["stream"]),
		fields,
	)
}

func summarizeChatPayload(src map[string]interface{}) string {
	return fmt.Sprintf(
		"model=%q messages=%d tools=%d stream=%t max_tokens=%d",
		getString(src["model"]),
		countArrayEntries(src["messages"]),
		countArrayEntries(src["tools"]),
		getBool(src["stream"]),
		getInt(src["max_tokens"]),
	)
}

func countArrayEntries(value interface{}) int {
	switch typed := value.(type) {
	case []interface{}:
		return len(typed)
	case []map[string]interface{}:
		return len(typed)
	default:
		return 0
	}
}

func getString(value interface{}) string {
	text, _ := value.(string)
	return text
}

func getBool(value interface{}) bool {
	flag, _ := value.(bool)
	return flag
}

func getInt(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

// ResponseTranslationWriter captures backend Chat Completion responses and reformats them
type ResponseTranslationWriter struct {
	gin.ResponseWriter
	bodyBuffer   *bytes.Buffer
	isStream     bool
	modelName    string
	hasStarted   bool
	isCompleted  bool
	terminalSent bool
	lastChunkID  string
	lastCreated  float64
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
		return w.bodyBuffer.Write(b)
	}

	statusCode := w.ResponseWriter.Status()
	contentType := strings.ToLower(w.Header().Get("Content-Type"))
	if !w.hasStarted {
		if statusCode >= 400 || (contentType != "" && !strings.Contains(contentType, "text/event-stream")) {
			log.Printf("[DEBUG-STREAM-PASSTHROUGH] Passing through upstream status=%d content_type=%q without SSE rewrite\n", statusCode, contentType)
			return w.ResponseWriter.Write(b)
		}

		log.Printf("[DEBUG-STREAM] Stream connection initiated down to client (isStream=true)\n")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")
		if !w.ResponseWriter.Written() {
			w.WriteHeader(200)
		}
		w.hasStarted = true
	}

	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		log.Printf("[DEBUG-BACKEND-CHUNK] Raw block from model: %s\n", line)

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		dataContent := strings.TrimPrefix(line, "data: ")
		if dataContent == "[DONE]" {
			log.Printf("[DEBUG-STREAM] Caught [DONE] delimiter.\n")
			w.sendTerminalStreamChunk()
			w.ResponseWriter.Write([]byte("data: [DONE]\n\n"))
			continue
		}

		var chatChunk map[string]interface{}
		if err := json.Unmarshal([]byte(dataContent), &chatChunk); err == nil {
			if id, ok := chatChunk["id"].(string); ok {
				w.lastChunkID = id
			}
			if created, ok := chatChunk["created"].(float64); ok {
				w.lastCreated = created
			}

			choices, _ := chatChunk["choices"].([]interface{})
			deltaText := ""
			finishReason := interface{}(nil)
			hasDelta := false

			if len(choices) > 0 {
				choice := choices[0].(map[string]interface{})
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					if content, ok := delta["content"].(string); ok {
						deltaText = content
						hasDelta = true
					}
				}
				if fr, ok := choice["finish_reason"]; ok && fr != nil {
					finishReason = fr
				}
			}

			if len(choices) == 0 {
				log.Printf("[DEBUG-STREAM-USAGE] Ignoring usage-only chunk after translation\n")
				continue
			}

			if w.isCompleted && finishReason == nil && !hasDelta {
				log.Printf("[DEBUG-STREAM-SKIP] Ignoring trailing empty chunk after completion\n")
				continue
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
				w.isCompleted = true
				w.terminalSent = true
				log.Printf("[DEBUG-STREAM-FINISH] DeepSeek declared natural execution end condition: %v\n", finishReason)
			} else {
				respChunk["status"] = "in_progress"
			}

			chunkBytes, _ := json.Marshal(respChunk)
			log.Printf("[DEBUG-CLIENT-CHUNK] Writing SSE chunk frame: %s\n", string(chunkBytes))
			w.ResponseWriter.Write([]byte("event: response.chunk\n"))
			w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n\n", string(chunkBytes))))
		}
	}
	return len(b), nil
}

func (w *ResponseTranslationWriter) sendTerminalStreamChunk() {
	if w.terminalSent {
		log.Printf("[DEBUG-CLIENT-TERMINAL] Skipping synthetic completed chunk because a terminal event was already sent\n")
		return
	}

	if w.lastChunkID == "" {
		w.lastChunkID = "chatcmpl-proxy-final-id"
	}
	if w.lastCreated == 0 {
		w.lastCreated = 1782607675
	}

	finalChunk := map[string]interface{}{
		"id":            w.lastChunkID,
		"object":        "response.chunk",
		"created_at":    w.lastCreated,
		"model":         w.modelName,
		"output_text":   "",
		"status":        "completed",
		"finish_reason": "stop",
	}

	chunkBytes, _ := json.Marshal(finalChunk)
	log.Printf("[DEBUG-CLIENT-TERMINAL] Writing synthetic completed token barrier chunk: %s\n", string(chunkBytes))
	w.ResponseWriter.Write([]byte("event: response.chunk\n"))
	w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n\n", string(chunkBytes))))
	w.isCompleted = true
	w.terminalSent = true
}

func (w *ResponseTranslationWriter) FlushResponse() {
	if w.isStream {
		return
	}

	log.Printf("[DEBUG-UNARY] Compiling non-streaming backend string data...\n")
	var chatResponse map[string]interface{}
	if err := json.Unmarshal(w.bodyBuffer.Bytes(), &chatResponse); err != nil {
		log.Printf("[DEBUG-UNARY-FALLBACK] Failed parsing unary json, delivering raw payload stream bytes.\n")
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
	log.Printf("[DEBUG-UNARY-RESPONSE] Returning clean payload back to curl:\n%s\n", string(finalBytes))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(finalBytes)))
	w.ResponseWriter.Write(finalBytes)
}
