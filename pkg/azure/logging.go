package azure

import (
	"log"
	"os"
	"strings"
)

var DebugLogging = parseBoolEnv("AZURE_OPENAI_PROXY_DEBUG")

func debugf(format string, args ...interface{}) {
	if !DebugLogging {
		return
	}
	log.Printf(format, args...)
}

func parseBoolEnv(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
