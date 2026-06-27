package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vinchent72/azure-oai-proxy/pkg/azure"
	"github.com/vinchent72/azure-oai-proxy/pkg/openai"
	"github.com/joho/godotenv"
	"github.com/tidwall/gjson"
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

	if v := os.Getenv("AZURE_OPENAI_MODEL_MAPPER"); v != "" {
		for _, pair := range strings.Split(v, ",") {
			info := strings.Split(pair, "=")
			if len(info) == 2 {
				azure.FoundryModelMapper[strings.ToLower(strings.TrimSpace(info[0]))] = strings.TrimSpace(info[1])
				log.Printf("Registered Native Model Route: %s -> %s", info[0], info[1])
			}
		}
	}
}

func main() {
	router := gin.Default()

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	if ProxyMode == "azure" {
		router.GET("/v1/models", handleGetModels)

		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path

			if c.Request.Method == http.MethodOptions {
				handleOptions(c)
				return
			}

			// Core Interceptor for /v1/responses targeting Chat-Only backends (e.g., DeepSeek)
			if path == "/v1/responses" && c.Request.Body != nil {
				bodyBytes, err := io.ReadAll(c.Request.Body)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
					return
				}

				modelName := gjson.GetBytes(bodyBytes, "model").String()
				isStream := gjson.GetBytes(bodyBytes, "stream").Bool()

				if azure.IsChatOnlyModel(modelName) {
					log.Printf("[Autodetect] Model %s is chat-only. Translating payload...", modelName)
					
					translatedBody, err := azure.TranslateResponsesToChatRequest(bodyBytes)
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to map responses schema to chat completion"})
						return
					}

					// Mutate Request properties for underlying package compatibility
					c.Request.URL.Path = "/v1/chat/completions"
					c.Request.Body = io.NopCloser(bytes.NewBuffer(translatedBody))
					c.Request.ContentLength = int64(len(translatedBody))
					c.Request.Header.Set("Content-Length", string(len(translatedBody)))

					// Instantiate our dynamic response transformation wrapper
					translationWriter := azure.NewResponseTranslationWriter(c.Writer, isStream, modelName)
					c.Writer = translationWriter

					// Forward execution
					handleAzureProxy(c)

					// Finalize transformation for unary non-streaming bodies
					translationWriter.FlushResponse()
					return
				}

				// If it supports Responses natively, restore body data unmodified
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/deployments/") {
				handleAzureProxy(c)
				return
			}

			c.JSON(http.StatusNotFound, gin.H{"error": "Resource path not found"})
		})
	} else {
		router.NoRoute(handleOpenAIProxy)
	}

	if err := router.Run(Address); err != nil {
		log.Fatalf("Proxy engine failure: %v", err)
	}
}

func handleGetModels(c *gin.Context) {
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