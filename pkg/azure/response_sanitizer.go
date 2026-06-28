package azure

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type ResponsesSanitizationReport struct {
	Model         string
	Mode          CompatibilityMode
	DroppedTools  []string
	DroppedChoice bool
}

func SanitizeResponsesRequest(bodyBytes []byte) ([]byte, ResponsesSanitizationReport, error) {
	var src map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &src); err != nil {
		return nil, ResponsesSanitizationReport{}, err
	}

	modelName, _ := src["model"].(string)
	profile := ResolveModelAPIProfile(modelName)
	report := ResponsesSanitizationReport{
		Model: modelName,
		Mode:  profile.CompatibilityMode,
	}

	filteredTools, droppedTools := sanitizeResponsesTools(src["tools"], profile)
	if len(droppedTools) > 0 || profile.CompatibilityMode == CompatibilityModePlainChat {
		report.DroppedTools = droppedTools
		if len(filteredTools) > 0 {
			src["tools"] = filteredTools
		} else {
			delete(src, "tools")
		}
	}

	if shouldDropToolChoice(src["tool_choice"], filteredTools, profile) {
		delete(src, "tool_choice")
		delete(src, "parallel_tool_calls")
		report.DroppedChoice = true
	}

	sanitizedBytes, err := json.Marshal(src)
	if err != nil {
		return nil, ResponsesSanitizationReport{}, err
	}

	logResponsesSanitization(report)
	return sanitizedBytes, report, nil
}

func sanitizeResponsesTools(rawTools interface{}, profile ModelAPIProfile) ([]interface{}, []string) {
	tools, ok := rawTools.([]interface{})
	if !ok {
		return nil, nil
	}

	filtered := make([]interface{}, 0, len(tools))
	dropped := make([]string, 0)

	for _, item := range tools {
		tool, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		toolName := responseToolName(tool)
		toolType := strings.TrimSpace(getString(tool["type"]))

		if profile.CompatibilityMode == CompatibilityModePlainChat {
			if toolName == "" {
				toolName = toolType
			}
			if toolName != "" {
				dropped = append(dropped, toolName)
			}
			continue
		}

		if toolName != "" && profile.BlockedResponseTools[toolName] {
			dropped = append(dropped, toolName)
			continue
		}

		filtered = append(filtered, item)
	}

	return filtered, dropped
}

func responseToolName(tool map[string]interface{}) string {
	if name := strings.TrimSpace(getString(tool["name"])); name != "" {
		return name
	}
	return strings.TrimSpace(getString(tool["type"]))
}

func shouldDropToolChoice(rawChoice interface{}, filteredTools []interface{}, profile ModelAPIProfile) bool {
	if rawChoice == nil {
		return false
	}
	if profile.CompatibilityMode == CompatibilityModePlainChat || len(filteredTools) == 0 {
		return true
	}

	chosenToolName := extractToolChoiceName(rawChoice)
	if chosenToolName == "" {
		return false
	}

	for _, item := range filteredTools {
		tool, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if responseToolName(tool) == chosenToolName {
			return false
		}
	}

	return true
}

func extractToolChoiceName(rawChoice interface{}) string {
	switch typed := rawChoice.(type) {
	case string:
		switch strings.TrimSpace(typed) {
		case "", "auto", "none", "required":
			return ""
		default:
			return strings.TrimSpace(typed)
		}
	case map[string]interface{}:
		if name := strings.TrimSpace(getString(typed["name"])); name != "" {
			return name
		}
		if functionMap, ok := typed["function"].(map[string]interface{}); ok {
			return strings.TrimSpace(getString(functionMap["name"]))
		}
		return ""
	default:
		return ""
	}
}

func logResponsesSanitization(report ResponsesSanitizationReport) {
	if len(report.DroppedTools) == 0 && !report.DroppedChoice {
		return
	}

	log.Printf(
		"[DEBUG-SANITIZE] model=%q mode=%s dropped_tools=%v dropped_tool_choice=%t\n",
		report.Model,
		report.Mode,
		report.DroppedTools,
		report.DroppedChoice,
	)
}

func (r ResponsesSanitizationReport) String() string {
	return fmt.Sprintf("model=%q mode=%s dropped_tools=%v dropped_tool_choice=%t", r.Model, r.Mode, r.DroppedTools, r.DroppedChoice)
}
