# Azure OpenAI Proxy

[![Go Report Card](https://goreportcard.com/badge/github.com/Gyarbij/azure-oai-proxy)](https://goreportcard.com/report/github.com/Gyarbij/azure-oai-proxy)
[![Main v Dev Commits](https://shields.git.vg/github/commits-difference/Gyarbij/azure-oai-proxy?base=main&head=dev)](https://github.com/gyarbij/azure-oai-proxy)
[![Taal](https://shields.git.vg/github/languages/top/Gyarbij/azure-oai-proxy)](https://github.com/gyarbij/azure-oai-proxy)
[![GHCR Build](https://shields.git.vg/github/actions/workflow/status/gyarbij/azure-oai-proxy/ghcr-docker-publish.yml)](https://github.com/gyarbij/azure-oai-proxy)
[![License](https://shields.git.vg/github/license/Gyarbij/azure-oai-proxy?style=for-the-badge&color=blue)](https://github.com/gyarbij/azure-oai-proxy/blob/main/LICENSE)

## Introduction

Azure OAI Proxy is a lightweight, high-performance proxy server that enables seamless integration with **Microsoft Azure AI Foundry** (serverless deployments). It translates OpenAI API format requests into Azure Foundry endpoints, allowing applications built for OpenAI's API to work seamlessly with Azure's managed AI services.

The proxy provides full support for the complete Microsoft Foundry model catalog, including GPT-5.x, reasoning models (O-series), Claude models, embeddings, audio, video, and specialized models—all through a unified serverless architecture.

## Key Features

-   ✅ **Microsoft Foundry Integration**: Seamlessly routes requests to Azure AI Foundry (serverless) endpoints.
-   ✅ **Comprehensive Model Support**: 200+ model variants including GPT-5.x, O-series, Claude, embeddings, audio, video, and specialized models.
-   🧠 **Advanced Reasoning Models**: Full support for O1, O3, O4 series and reasoning models through Responses API.
-   📡 **Streaming Support**: Real-time streaming for all model types with proper format conversion.
-   🗺️ **Intelligent Model Mapping**: Automatically maps OpenAI model names to Foundry deployments with comprehensive built-in model list.
-   🌐 **Multi-API Support**: Handles chat completions, embeddings, image/video generation, audio, and more.
-   🚦 **Error Handling**: Meaningful error messages and detailed logging for debugging.
-   ⚙️ **Simple Configuration**: Easy setup with environment variables for region, API version, and custom mappings.
-   🔐 **Bearer Token Authentication**: Standard Azure Foundry authentication for all requests.

## Use Cases

This proxy is particularly useful for:

-   Running applications like Open WebUI, LangChain, or other OpenAI-compatible tools with Azure Foundry deployments.
-   Using Microsoft's latest reasoning models (O3, O4) through OpenAI-compatible interfaces.
-   Leveraging Claude models on Azure Foundry in applications built for OpenAI API.
-   Testing Azure Foundry capabilities without modifying existing OpenAI-based applications.
-   Cost-effective managed deployments: Foundry serverless handles scaling automatically.
-   Multi-model workloads: Switch between GPT, Claude, Phi, and specialized models seamlessly.

## Architecture

Azure OAI Proxy uses a simple routing model:

```
OpenAI API Request → Proxy → Microsoft Azure Foundry
(http://localhost:11437)      (https://{deployment}.{region}.models.ai.azure.com)
```

**Request Flow:**
1. Receive request in OpenAI format (e.g., `/v1/chat/completions`)
2. Extract model name from request body
3. Look up Foundry deployment name in model mapper
4. Route to Foundry endpoint: `https://{deployment}.{region}.models.ai.azure.com/{endpoint}?api-version=2024-08-01-preview`
5. Convert authentication (api-key → Bearer token)
6. Forward request and convert response back to OpenAI format

## Supported APIs

The latest version of the Azure OpenAI service supports the following APIs:

| Path                               | Status | Notes |
| :--------------------------------- | :----- | :---- |
| /v1/chat/completions               | ✅     | Auto-routes to Responses API for reasoning models |
| /v1/completions                    | ✅     |       |
| /v1/embeddings                     | ✅     |       |
| /v1/images/generations             | ✅     |       |
| /v1/fine_tunes                     | ✅     |       |
| /v1/files                          | ✅     |       |
| /v1/models                         | ✅     |       |
| /v1/responses                      | ✅     | **New** - Azure Responses API support |
| /v1/responses/:response_id         | ✅     | **New** - Retrieve, delete, cancel operations |
| /v1/responses/:response_id/input_items | ✅ | **New** - List input items |
| /deployments                       | ✅     |       |
| /v1/audio/speech                   | ✅     |       |
| /v1/audio/transcriptions            | ✅     |       |
| /v1/audio/translations             | ✅     |       |
| /v1/models/:model_id/capabilities | ✅     |       |

## Model Support

The proxy supports **200+ model variants** across all Microsoft Foundry categories:

### GPT Series (Chat Completions API)
- **GPT-5.5 series** (Latest): gpt-5.5, gpt-5.5-mini, gpt-5.5-nano, gpt-5.5-chat
- **GPT-5.4 series**: gpt-5.4, gpt-5.4-mini, gpt-5.4-nano, gpt-5.4-chat
- **GPT-5.3 series**: gpt-5.3, gpt-5.3-mini, gpt-5.3-nano, gpt-5.3-chat
- **GPT-5.x series**: gpt-5.2, gpt-5.1, gpt-5 (all variants)
- **GPT-4.1 series**: gpt-4.1, gpt-4.1-mini, gpt-4.1-nano
- **GPT-4o series**: gpt-4o, gpt-4o-mini (with date variants)
- **GPT-4 series**: gpt-4, gpt-4-turbo, gpt-4-32k (all variants)
- **GPT-3.5 series**: gpt-3.5-turbo, gpt-3.5-turbo-16k (all variants)

### Reasoning Models (Responses API)
- **O-Series**: o1, o1-preview, o1-mini, o3, o3-mini, o3-pro, o3-deep-research, o4, o4-mini
- **Specialized**: codex-mini, gpt-5.x-codex variants, computer-use-preview, gpt-5-pro

### Claude Models (Anthropic Messages API)
- **Latest**: claude-opus-4.5, claude-sonnet-4.5, claude-haiku-4.5
- **Opus 4.1**: claude-opus-4.1
- ℹ️ Automatically converted from OpenAI chat format to Anthropic Messages API

### Other Models
- **Embeddings**: text-embedding-3-small, text-embedding-3-large, text-embedding-ada-002
- **Image Generation**: dall-e-2, dall-e-3, gpt-image-1, gpt-image-1-mini
- **Video**: sora, sora-2 (with date variants)
- **Audio**: gpt-4o-audio-preview, gpt-4o-realtime-preview, gpt-audio, gpt-realtime, whisper
- **Speech-to-Text**: gpt-4o-transcribe, gpt-4o-transcribe-diarize
- **Text-to-Speech**: tts, tts-hd, gpt-4o-mini-tts
- **Open Source**: phi-3, phi-3-mini, phi-3-small, phi-3-medium, phi-4, gpt-oss-120b, gpt-oss-20b

*For a complete list, see the [model mapper](pkg/azure/proxy.go#L180) in the code.*

## Configuration

### Environment Variables

| Parameter                       | Description                                                    | Default Value    | Required |
| :------------------------------ | :------------------------------------------------------------- | :--------------- | :------- |
| AZURE_FOUNDRY_REGION            | Azure Foundry region (e.g., westus, eastus, northcentralus)   | westus           | No       |
| ANTHROPIC_APIVERSION            | Anthropic API version (for Claude models)                      | 2023-06-01       | No       |
| AZURE_OPENAI_PROXY_ADDRESS      | Proxy server listening address                                 | 0.0.0.0:11437    | No       |
| AZURE_OPENAI_MODEL_MAPPER       | Comma-separated list of model=deployment pairs (optional overrides) |                  | No       |

### How It Works

1. **Region**: Set `AZURE_FOUNDRY_REGION` to your Foundry region. Default is `westus`.
2. **Models**: The proxy includes 200+ built-in model mappings. Use `AZURE_OPENAI_MODEL_MAPPER` to override for custom deployments.
3. **Authentication**: Pass your Foundry API key as Bearer token or `api-key` header. The proxy converts it automatically.
4. **API Version**: `ANTHROPIC_APIVERSION` is used only for Claude models.

### Example Custom Model Mappings

If you have custom deployment names, override them:

```
AZURE_OPENAI_MODEL_MAPPER=my-gpt=my-gpt-deployment,my-claude=my-claude-deployment
```

## Usage

### Quick Start with Docker Compose

1. **Using the provided compose.yaml:**

```sh
docker compose up -d
```

2. **Custom region:**

```sh
AZURE_FOUNDRY_REGION=eastus docker compose up -d
```

3. **Or create a .env file:**

```
AZURE_FOUNDRY_REGION=westus
ANTHROPIC_APIVERSION=2023-06-01
```

Then run:
```sh
docker compose up -d
```

### Docker Command

```sh
docker run -d -p 11437:11437 \
  -e AZURE_FOUNDRY_REGION=westus \
  ghcr.io/gyarbij/azure-oai-proxy:latest
```

### Configuration in Docker Compose

See [compose.yaml](compose.yaml) for a pre-configured example with all supported environment variables documented.

## API Examples

All examples use standard OpenAI API format. The proxy automatically routes to the appropriate Foundry endpoint.

### GPT Model (Chat Completions)

```bash
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-foundry-api-key" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Reasoning Model (O-Series)

The proxy automatically routes O-series models through the Responses API while maintaining OpenAI format compatibility.

```bash
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-foundry-api-key" \
  -d '{
    "model": "o3-mini",
    "messages": [{"role": "user", "content": "Solve this math problem: 2+2"}],
    "max_tokens": 2000
  }'
```

### Claude Model (Anthropic Messages API)

The proxy automatically converts from OpenAI format to Anthropic Messages API internally.

```bash
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-foundry-api-key" \
  -d '{
    "model": "claude-sonnet-4.5",
    "messages": [{"role": "user", "content": "Explain quantum computing"}],
    "max_tokens": 1000
  }'
```

### Embeddings

```bash
curl http://localhost:11437/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-foundry-api-key" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "Hello, world!"
  }'
```

### Image Generation

```bash
curl http://localhost:11437/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-foundry-api-key" \
  -d '{
    "model": "dall-e-3",
    "prompt": "A beautiful sunset",
    "n": 1,
    "size": "1024x1024"
  }'
```

### Streaming

Add `"stream": true` to enable streaming responses:

```bash
curl http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-foundry-api-key" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Count to 10"}],
    "stream": true
  }'
```

## Security Recommendations

1. **Always use TLS/SSL** in production. Configure a reverse proxy (nginx, Caddy) with SSL termination in front of the proxy.
2. **Protect your API key**: Never expose your Foundry API key in client-side code.
3. **Firewall**: Restrict access to the proxy port to trusted networks only.
4. **Rate limiting**: Consider adding rate limiting middleware for production deployments.

## Troubleshooting

### Check Logs

```sh
docker logs $(docker ps --filter "ancestor=ghcr.io/gyarbij/azure-oai-proxy:latest" -q)
```

### Verify Foundry Connectivity

Ensure you can reach your Foundry deployment:

```bash
curl -H "Authorization: Bearer your-api-key" \
  "https://your-deployment.westus.models.ai.azure.com/chat/completions?api-version=2024-08-01-preview" \
  -X POST -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"test"}]}'
```

### Common Issues

| Issue | Solution |
|-------|----------|
| "Model not found" | Ensure the model is available in your Foundry region. Check the model mapper in proxy.go. |
| "Authentication failed" | Verify your API key is correct and has permissions for the Foundry region. |
| "Connection refused" | Check that the Foundry endpoint is correct (region, deployment name). |
| "Claude responses fail" | Claude must be deployed separately in Azure Foundry. Check model mapper configuration. |
- Response is converted back to OpenAI chat completion format
- System messages are extracted and passed as the `system` parameter
- Headers are automatically adjusted (`x-api-key`, `anthropic-version: 2023-06-01`)
- **Note**: Uses `ANTHROPIC_APIVERSION` environment variable (default: `2023-06-01`)

**Example with custom deployment name:**
If your Claude deployment has a different name (e.g., `Claude-Sonnet-45-20251001`), use the model mapper:
```bash
AZURE_OPENAI_MODEL_MAPPER=claude-sonnet-4.5=Claude-Sonnet-45-20251001
```

#### Phi Models (Azure Foundry)
```sh
curl http://localhost:11437/v1/chat/completions \
 -H "Content-Type: application/json" \
 -H "Authorization: Bearer your-azure-api-key" \
 -d '{
  "model": "phi-4",
  "messages": [{"role": "user", "content": "What is machine learning?"}]
 }'
```

#### Reasoning Models (Automatically routed to Responses API)
```sh
curl http://localhost:11437/v1/chat/completions \
 -H "Content-Type: application/json" \
 -H "Authorization: Bearer your-azure-api-key" \
 -d '{
  "model": "o3-pro",
  "messages": [{"role": "user", "content": "Solve this complex reasoning problem..."}],
  "stream": true
 }'
```

#### Direct Responses API Access
```sh
curl http://localhost:11437/v1/responses \
 -H "Content-Type: application/json" \
 -H "Authorization: Bearer your-azure-api-key" \
 -d '{
  "model": "o3-pro",
  "input": "What are the implications of quantum computing?",
  "stream": false
 }'
```

For serverless deployments, use the model name as defined in your `AZURE_AI_STUDIO_DEPLOYMENTS` configuration.

## Model Mapping Mechanism (Used for Custom deployment names)

These are the default mappings for the most common models, if your Azure OpenAI deployment uses different names, you can set the `AZURE_OPENAI_MODEL_MAPPER` environment variable to define custom mappings. The proxy also includes a comprehensive **failsafe list** to handle a wide variety of model names:

### Reasoning Models (O-series)
| OpenAI Model                 | Azure OpenAI Model           |
| :--------------------------- | :--------------------------- |
| `"o1"`                       | `"o1"`                       |
| `"o1-preview"`               | `"o1-preview"`               |
| `"o1-mini"`                  | `"o1-mini"`                  |
| `"o1-mini-2024-09-12"`       | `"o1-mini-2024-09-12"`       |
| `"o3"`                       | `"o3"`                       |
| `"o3-mini"`                  | `"o3-mini"`                  |
| `"o3-pro"`                   | `"o3-pro"`                   |
| `"o3-pro-2025-06-10"`        | `"o3-pro-2025-06-10"`        |
| `"o4"`                       | `"o4"`                       |
| `"o4-mini"`                  | `"o4-mini"`                  |

### Claude Models (Azure Foundry)
| OpenAI Model                 | Azure OpenAI Model           |
| :--------------------------- | :--------------------------- |
| `"claude-opus-4.5"`          | `"claude-opus-4.5"`          |
| `"claude-opus-4-5"`          | `"claude-opus-4.5"`          |
| `"claude-sonnet-4.5"`        | `"claude-sonnet-4.5"`        |
| `"claude-sonnet-4-5"`        | `"claude-sonnet-4.5"`        |
| `"claude-haiku-4.5"`         | `"claude-haiku-4.5"`         |
| `"claude-haiku-4-5"`         | `"claude-haiku-4.5"`         |
| `"claude-opus-4.1"`          | `"claude-opus-4.1"`          |
| `"claude-opus-4-1"`          | `"claude-opus-4.1"`          |

### GPT Models
| OpenAI Model                 | Azure OpenAI Model           |
| :--------------------------- | :--------------------------- |
| `"gpt-4o"`                   | `"gpt-4o"`                   |
| `"gpt-4o-2024-05-13"`        | `"gpt-4o-2024-05-13"`        |
| `"gpt-4o-2024-08-06"`        | `"gpt-4o-2024-08-06"`        |
| `"gpt-4o-2024-11-20"`        | `"gpt-4o-2024-11-20"`        |
| `"gpt-4o-mini"`              | `"gpt-4o-mini"`              |
| `"gpt-4o-mini-2024-07-18"`   | `"gpt-4o-mini-2024-07-18"`   |
| `"gpt-4"`                    | `"gpt-4-0613"`               |
| `"gpt-4-turbo"`              | `"gpt-4-turbo"`              |
| `"gpt-4-turbo-2024-04-09"`   | `"gpt-4-turbo-2024-04-09"`   |
| `"gpt-3.5-turbo"`            | `"gpt-35-turbo-0613"`        |
| `"gpt-3.5-turbo-16k"`        | `"gpt-35-turbo-16k-0613"`    |

### Phi Models (Azure Foundry)
| OpenAI Model                 | Azure OpenAI Model           |
| :--------------------------- | :--------------------------- |
| `"phi-3"`                    | `"phi-3"`                    |
| `"phi-3-mini"`               | `"phi-3-mini"`               |
| `"phi-3-small"`              | `"phi-3-small"`              |
| `"phi-3-medium"`             | `"phi-3-medium"`             |
| `"phi-4"`                    | `"phi-4"`                    |

### Other Models
| OpenAI Model                 | Azure OpenAI Model           |
| :--------------------------- | :--------------------------- |
| `"text-embedding-3-small"`   | `"text-embedding-3-small-1"` |
| `"text-embedding-3-large"`   | `"text-embedding-3-large-1"` |
| `"dall-e-2"`                 | `"dall-e-2-2.0"`             |
| `"dall-e-3"`                 | `"dall-e-3-3.0"`             |
| `"tts"`                      | `"tts-001"`                  |
| `"tts-hd"`                   | `"tts-hd-001"`               |
| `"whisper"`                  | `"whisper-001"`              |

For custom fine-tuned models, the model name can be passed directly. For models with deployment names different from the model names, custom mapping relationships can be defined, such as:

| Model Name        | Deployment Name          |
| :---------------- | :----------------------- |
| gpt-3.5-turbo     | gpt-35-turbo-upgrade     |
| gpt-3.5-turbo-0301 | gpt-35-turbo-0301-fine-tuned |

## Reasoning Models & Responses API

### Automatic Detection
The proxy automatically detects when you're using reasoning models (O1, O3, O4 series) and:

1. **Routes to Responses API**: Automatically converts `/v1/chat/completions` requests to use Azure's `/openai/v1/responses` endpoint
2. **Converts Request Format**: Transforms OpenAI chat messages to Responses API input format
3. **Handles Streaming**: Converts Responses API SSE events to OpenAI-compatible streaming format
4. **Maintains Compatibility**: Your client code doesn't need to change - use standard OpenAI format

### Supported Reasoning Models
- **O1 Family**: `o1`, `o1-preview`, `o1-mini`, `o1-mini-2024-09-12`
- **O3 Family**: `o3`, `o3-pro`, `o3-mini`, `o3-pro-2025-06-10` 
- **O4 Family**: `o4`, `o4-mini`

### Response API Features
When using reasoning models, you get access to:
- **Advanced Reasoning**: Enhanced problem-solving capabilities
- **Reasoning Traces**: Detailed reasoning process (when available)
- **Background Processing**: Support for long-running reasoning tasks
- **Chain of Thought**: Structured reasoning outputs

## Important Notes

-   Always use HTTPS in production environments for secure communication.
-   Regularly update the proxy to ensure compatibility with the latest Azure OpenAI API changes.
-   Monitor your Azure OpenAI usage and costs, especially when using this proxy in high-traffic scenarios.
-   Reasoning models may have higher latency due to their advanced processing capabilities.
-   Some reasoning models may have usage limits or require special access permissions.

## Troubleshooting

### Claude Models

**✅ NEW: Native Anthropic Messages API Support**
- Claude models now use the **Anthropic Messages API** (`/anthropic/v1/messages`)
- Automatic conversion from OpenAI chat completions format
- Automatic response conversion back to OpenAI format
- No configuration changes needed - use standard OpenAI format

**Error: "This model is not supported by Responses API"**
- **Fixed**: Claude models now correctly use Anthropic Messages API (not Responses API or standard Chat Completions)
- **Solution**: Update to the latest version - the proxy now automatically routes Claude to the correct endpoint

**Error: "Unknown model: claude-sonnet-4-5" or similar**
- **Cause**: The deployment name in Azure doesn't match the model name you're using
- **Solution**: Use `AZURE_OPENAI_MODEL_MAPPER` to map the model name to your actual Azure deployment name:
  ```bash
  # If your deployment is named something like "Claude-Sonnet-45-20251001"
  AZURE_OPENAI_MODEL_MAPPER=claude-sonnet-4.5=Claude-Sonnet-45-20251001
  ```
- **Tip**: Check your Azure Foundry portal to see the exact deployment name

**Deployment Requirements**:
1. Claude models must be deployed in your Azure Foundry account (East US2 or Sweden Central)
2. They require Global Standard deployment
3. The endpoint format is `https://your-resource.services.ai.azure.com`
4. Uses `x-api-key` header and `anthropic-version: 2023-06-01`

### General 404 Errors

**Error: "Resource not found" (404)**
- **Check deployment exists**: Verify the model is deployed in your Azure account
- **Check deployment name**: Use the detailed logging to see what deployment name is being used
- **Use model mapper**: Map model names to your actual deployment names if they differ

## Recently Updated
-   **2025-12-14 (Latest)** Added native Anthropic Messages API support for Claude models:
    - Claude models now use `/anthropic/v1/messages` endpoint (correct format for Azure Foundry)
    - Automatic bidirectional conversion between OpenAI and Anthropic formats
    - System messages extracted and handled correctly
    - Headers automatically adjusted (`x-api-key`, `anthropic-version`)
    - Seamless integration - use standard OpenAI chat completions format
-   **2025-12-14** Added comprehensive Azure OpenAI in Microsoft Foundry support including:
    - GPT-5.2 series (gpt-5.2, gpt-5.2-chat) - NEW preview models
    - GPT-5.1 series (gpt-5.1, gpt-5.1-chat, gpt-5.1-codex variants)
    - GPT-5 series (gpt-5, gpt-5-mini, gpt-5-nano, gpt-5-chat, gpt-5-codex, gpt-5-pro)
    - GPT-4.1 series (gpt-4.1, gpt-4.1-mini, gpt-4.1-nano)
    - Claude 4.x models (Opus 4.5, Sonnet 4.5, Haiku 4.5, Opus 4.1)
    - Complete O-series reasoning models (o1, o3, o4 variants, o3-deep-research)
    - Codex models (codex-mini, gpt-5.1-codex variants)
    - Audio models (gpt-4o audio/realtime/transcribe, gpt-realtime, gpt-audio variants)
    - Image generation (gpt-image-1, gpt-image-1-mini)
    - Video generation (sora, sora-2)
    - Open-weight models (gpt-oss-120b, gpt-oss-20b)
    - Specialized models (computer-use-preview)
    - Updated API versions to 2024-08-01-preview (general and Responses API - supports all Azure Foundry models)
-   **2025-08-03 (v1.0.8)** Added comprehensive support for Azure OpenAI Responses API with automatic reasoning model detection and streaming conversion.
-   2025-01-24 Added support for Azure OpenAI API version 2024-12-01-preview and new model fetching mechanism.
-   2024-07-25 Implemented support for Azure AI Studio deployments with support for Meta LLama 3.1, Mistral-2407 (mistral large 2), and other open models including from Cohere AI.
-   2024-07-18 Added support for `gpt-4o-mini`.
-   2024-06-23 Implemented dynamic model fetching for `/v1/models` endpoint, replacing hardcoded model list.
-   2024-06-23 Unified token handling mechanism across the application, improving consistency and security.
-   2024-06-23 Added support for audio-related endpoints: `/v1/audio/speech`, `/v1/audio/transcriptions`, and `/v1/audio/translations`.
-   2024-06-23 Implemented flexible environment variable handling for configuration (AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_API_KEY, AZURE_OPENAI_TOKEN).
-   2024-06-23 Added support for model capabilities endpoint `/v1/models/:model_id/capabilities`.
-   2024-06-23 Improved cross-origin resource sharing (CORS) handling with OPTIONS requests.
-   2024-06-23 Enhanced proxy functionality to better handle various Azure OpenAI API endpoints.
-   2024-06-23 Implemented fallback model mapping for unsupported models.
-   2024-06-22 Added support for image generation `/v1/images/generations`, fine-tuning operations `/v1/fine_tunes`, and file management `/v1/files`.
-   2024-06-22 Implemented better error handling and logging for API requests.
-   2024-06-22 Improved handling of rate limiting and streaming responses.
-   2024-06-22 Updated model mappings to include the latest models (gpt-4-turbo, gpt-4-vision-preview, dall-e-3).
-   2024-06-23 Added support for deployments management (/deployments).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.

## Disclaimer

This project is not officially associated with or endorsed by Microsoft Azure or OpenAI. Use at your own discretion and ensure compliance with all relevant terms of service.