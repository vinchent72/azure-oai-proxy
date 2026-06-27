package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	Address           = "0.0.0.0:11437"
	ProxyMode         = "azure"
	ModelsRegistry    = make(map[string]ModelConfig)
	ProvidersRegistry = make(map[string]ProviderConfig)
)

// Structural configuration aligning directly with copilot-api conventions
type ModelConfig struct {
	DisplayName     string `json:"display_name"`
	Provider        string `json:"provider"`
	ProviderModel   string `json:"provider_model"`
	ReasoningEffort string `json:"reasoning_effort"`
}

type ProviderConfig struct {
	Name               string `json:"name"`
	BaseURL            string `json:"base_url"`
	WireAPI            string `json:"wire_api"`
	APIVersion         string `json:"api_version"`
	EnvKey             string `json:"env_key"`
	RequiresOpenAIAuth bool   `json:"requires_openai_auth"`
}

type OpenAIRequest struct {
	Model string `json:"model"`
}

type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID              string       `json:"id"`
	Object          string       `json:"object"`
	CreatedAt       int64        `json:"created_at"`
	Capabilities    Capabilities `json:"capabilities"`
	LifecycleStatus string       `json:"lifecycle_status"`
	Status          string       `json:"status"`
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

	log.Printf("Starting Cloud Architecture Gateway Proxy...")
	log.Printf("Listening Address: %s | Mode: %s", Address, ProxyMode)

	// 1. Initialize Model Configuration Presets matching your client requirements
	ModelsRegistry["foundry-gpt-5-mini"] = ModelConfig{
		DisplayName:   "Foundry GPT‑5 Mini",
		Provider:      "azure_oai_proxy",
		ProviderModel: "gpt-5-mini",
	}
	ModelsRegistry["foundry-gpt-5.4-mini"] = ModelConfig{
		DisplayName:   "Foundry GPT‑5.4 Mini",
		Provider:      "azure_oai_proxy",
		ProviderModel: "gpt-5.4-mini",
	}
	ModelsRegistry["foundry-gpt-5.4"] = ModelConfig{
		DisplayName:   "Foundry GPT‑5.4 (xhigh)",
		Provider:      "azure_oai_proxy",
		ProviderModel: "gpt-5.4",
	}

	// 2. Initialize Provider Endpoint Configuration Profiles
	foundryBaseURL := os.Getenv("FOUNDRY_PROVIDER_BASE_URL")
	if foundryBaseURL == "" {
		// Fallback default
		foundryBaseURL = "https://code-ai-proxy.livelymeadow-ec98615a.westus3.inference.ai.azure.com"
	}

	ProvidersRegistry["azure_oai_proxy"] = ProviderConfig{
		Name:               "Azure OpenAI Proxy",
		BaseURL:            foundryBaseURL,
		WireAPI:            "responses",
		APIVersion:         "2024-10-01-preview",
		RequiresOpenAIAuth: false,
	}
}

func main() {
	router := gin.Default()

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	if ProxyMode == "azure" {
		router.GET("/v1/models", handleGetModels)
		router.OPTIONS("/v1/*path", handleOptions)

		// Create a catch-all group for your dynamic proxy routes
		v1Group := router.Group("/v1")
		{
			v1Group.Any("/*path", handleDynamicRoutingProxy)
		}
		router.Any("/deployments/*path", handleDynamicRoutingProxy)
	} else {
		// Fallback block if you switch back to open-source or native paths
		router.Any("*path", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Standard raw pass-through is deactivated"})
		})
	}

	if err := router.Run(Address); err != nil {
		log.Fatalf("Proxy engine failed: %v", err)
	}
}

func handleGetModels(c *gin.Context) {
	models := make([]Model, 0, len(ModelsRegistry))
	for modelID := range ModelsRegistry {
		models = append(models, Model{
			ID:              modelID,
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

func handleDynamicRoutingProxy(c *gin.Context) {
	if c.Request.Method == http.MethodOptions {
		handleOptions(c)
		return
	}

	// 1. Peek inside the payload to see what model is being requested
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read data payload"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req OpenAIRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil || req.Model == "" {
		// Fallback if model structure is empty
		for k := range ModelsRegistry {
			req.Model = k
			break
		}
	}

	// 2. Resolve the requested model rules from the registry
	modelSpec, modelExists := ModelsRegistry[req.Model]
	if !modelExists {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Model '%s' is unmapped in proxy config", req.Model)})
		return
	}

	providerSpec, providerExists := ProvidersRegistry[modelSpec.Provider]
	if !providerExists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Provider configurations for '%s' missing", modelSpec.Provider)})
		return
	}

	// 3. Construct target routing information dynamically
	parsedBase, err := url.Parse(providerSpec.BaseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Malformed provider upstream host location"})
		return
	}

	// Modify paths to support custom interfaces (like your /v1/responses logic)
	targetPath := c.Request.URL.Path
	if providerSpec.WireAPI != "" && !strings.Contains(targetPath, providerSpec.WireAPI) {
		targetPath = "/v1/" + providerSpec.WireAPI
	}

	targetURLStr := fmt.Sprintf("%s://%s%s", parsedBase.Scheme, parsedBase.Host, targetPath)

	// Stitch API version strings into query params safely
	if providerSpec.APIVersion != "" {
		if c.Request.URL.RawQuery != "" {
			targetURLStr += "?" + c.Request.URL.RawQuery + "&api-version=" + providerSpec.APIVersion
		} else {
			targetURLStr += "?api-version=" + providerSpec.APIVersion
		}
	}

	target, err := url.Parse(targetURLStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed assembling downstream target path"})
		return
	}

	// 4. Instantiate and trigger the HTTP reverse proxy flow
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		req.URL.RawQuery = target.RawQuery

		// Override model parameter to whatever backend provider identifier is expected
		// (e.g. swapping 'foundry-gpt-5-mini' back to 'gpt-5-mini' before hitting Foundry)
		if len(bodyBytes) > 0 {
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &bodyMap); err == nil {
				bodyMap["model"] = modelSpec.ProviderModel
				modifiedBody, _ := json.Marshal(bodyMap)
				req.Body = io.NopCloser(bytes.NewBuffer(modifiedBody))
				req.ContentLength = int64(len(modifiedBody))
				req.Header.Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))
			}
		}

		// Inject your Authorization/Entra context here
	}

	proxy.ServeHTTP(c.Writer, c.Request)

	if c.Writer.Header().Get("Content-Type") == "text/event-stream" {
		_, _ = c.Writer.Write([]byte("\n"))
	}
}