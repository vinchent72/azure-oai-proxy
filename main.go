package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vinchent72/azure-oai-proxy/pkg/azure"
	"github.com/vinchent72/azure-oai-proxy/pkg/openai"
	"github.com/joho/godotenv"
)

var (
	Address   = "0.0.0.0:11437"
	ProxyMode = "azure"
)

type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID              string       `json:"id"`
	Object          string       `json:"object"`
	LifecycleStatus string       `json:"lifecycle_status"`
	Status          string       `json:"status"`
	Capabilities    Capabilities `json:"capabilities"`
}

type Capabilities struct {
	Completion     bool `json:"completion"`
	ChatCompletion bool `json:"chat_completion"`
	Inference      bool `json:"inference"`
	Embeddings     bool `json:"embeddings"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	gin.SetMode(gin.ReleaseMode)
	if v := os.Getenv("AZURE_OPENAI_PROXY_ADDRESS"); v != "" {
		Address = v
	}
	if v := os.Getenv("AZURE_OPENAI_PROXY_MODE"); v != "" {
		ProxyMode = v
	}

	log.Printf("Starting Proxy Service on: %s (Mode: %s)", Address, ProxyMode)

	// Load native mappings cleanly from the environment directly into the package mapper
	if v := os.Getenv("AZURE_OPENAI_MODEL_MAPPER"); v != "" {
		for _, pair := range strings.Split(v, ",") {
			info := strings.Split(pair, "=")
			if len(info) == 2 {
				azure.FoundryModelMapper[info[0]] = info[1]
				log.Printf("Registered Native Model Route: %s -> %s", info[0], info[1])
			}
		}
	}
}

func main() {
	router := gin.Default()

	// Global Health Endpoint
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	if ProxyMode == "azure" {
		// 1. Register explicit, unambiguous routes first
		router.GET("/v1/models", handleGetModels)

		// 2. Use NoRoute as a safe interceptor to handle dynamic proxies without tree collisions
		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path

			// Intercept standard OPTIONS requests
			if c.Request.Method == http.MethodOptions {
				handleOptions(c)
				return
			}

			// Route standard v1 or workspace deployments pathways directly to Azure handler
			if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/deployments/") {
				handleAzureProxy(c)
				return
			}

			// Fallback for completely unmatched paths
			c.JSON(http.StatusNotFound, gin.H{"error": "Resource path not found"})
		})
	} else {
		// OpenAI standard proxy routing fallback
		router.NoRoute(handleOpenAIProxy)
	}

	if err := router.Run(Address); err != nil {
		log.Fatalf("Proxy engine failure: %v", err)
	}
}

func handleGetModels(c *gin.Context) {
	// Dynamically expose native keys directly to Codex CLI
	models := make([]Model, 0, len(azure.FoundryModelMapper))
	for modelName := range azure.FoundryModelMapper {
		models = append(models, Model{
			ID:              modelName,
			Object:          "model",
			LifecycleStatus: "active",
			Status:          "ready",
			Capabilities: Capabilities{
				Completion:     true,
				ChatCompletion: true,
				Inference:      true,
				Embeddings:     true,
			},
		})
	}
	c.JSON(http.StatusOK, ModelList{Object: "list", Data: models})
}

func handleOptions(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, api-key")
	c.Status(http.StatusOK)
}

func handleAzureProxy(c *gin.Context) {
	if c.Request.Method == http.MethodOptions {
		handleOptions(c)
		return
	}

	// Pass context straight to your underlying pkg/azure package
	server := azure.NewOpenAIReverseProxy()
	server.ServeHTTP(c.Writer, c.Request)

	if c.Writer.Header().Get("Content-Type") == "text/event-stream" {
		_, _ = c.Writer.Write([]byte("\n"))
	}
}

func handleOpenAIProxy(c *gin.Context) {
	server := openai.NewOpenAIReverseProxy()
	server.ServeHTTP(c.Writer, c.Request)
}