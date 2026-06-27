# Microsoft Foundry Proxy

A lightweight OpenAI-compatible proxy that forwards OpenAI-style requests to Microsoft Foundry (Azure) endpoints. It lets applications written for the OpenAI API run against Microsoft Foundry deployments with minimal or no client changes by translating paths, headers, and payloads between the two APIs.

**Key points**
- Listens for OpenAI-compatible requests and translates them to Microsoft Foundry (Responses API or per-deployment endpoints).
- Default listening address: `0.0.0.0:11437` (configurable via env).
- Exposes a small set of OpenAI-compatible endpoints (see Supported Endpoints).
- Model mapping and provider behavior are implemented in `pkg/azure/proxy.go` and bootstrapped in `main.go`.

**Status**
- Proxy mode: `azure` (default).
- Wire API: `responses` (default provider behavior).

**Table Of Contents**
- Overview
- Quickstart
- Configuration
- Supported Endpoints
- Example Requests
- Model Mapping
- Health & Diagnostics
- Contributing

**Overview**
This proxy accepts OpenAI-style requests (for example, `/v1/chat/completions`) and forwards them to Microsoft Foundry endpoints while performing URL, header, and payload translation. It supports workspace-level routing and serverless per-deployment routing, streaming conversion, Anthropic (Claude) compatibility, and model name mapping.

**Quickstart**

**Prerequisites**
- Go 1.24+ (see `go.mod`).

**Build & Run (local)**
- Run directly: `go run .`
- Build a binary and run: `go build -o azure-oai-proxy .` then `./azure-oai-proxy`

The proxy listens on `0.0.0.0:11437` by default (override with `AZURE_OPENAI_PROXY_ADDRESS`).

**Configuration (environment variables)**
The proxy reads configuration from environment variables at startup. Important vars used by the code (see `main.go` and `pkg/azure/proxy.go`):

- `AZURE_OPENAI_PROXY_ADDRESS` — listen address (default: `0.0.0.0:11437`).
- `AZURE_OPENAI_PROXY_MODE` — proxy mode (default: `azure`).
- `AZURE_OPENAI_MODEL_MAPPER` — comma-separated `source=target` pairs to override built-in model mappings (example: `gpt-5.4=deployment-name,gpt-5.4-mini=mini-deploy`).
- `AZURE_FOUNDRY_REGION` — region used for serverless per-deployment routing (default: `westus`).
- `AZURE_FOUNDRY_ENDPOINT` — workspace-level Foundry endpoint (if set, workspace routing is used).
- `FOUNDRY_API_KEY` — server-side API key used to authenticate outgoing requests to Foundry.
- `AUTH_TOKENS` — comma-separated list of client tokens allowed to call this proxy; requests must present one of these tokens (via `Authorization: Bearer <token>` or `api-key` header).
- `OPENAI_API_ENDPOINT` — override target OpenAI endpoint when running in OpenAI pass-through mode (optional).
- `ANTHROPIC_APIVERSION` — Anthropic (Claude) API version used for Claude conversions (optional).

**Supported Endpoints**
The proxy exposes OpenAI-compatible endpoints and translates them to Foundry routes. Main endpoints:

- `GET /healthz` — health check (`{"status":"healthy"}`).
- `GET /v1/models` — list known model names (populated from the built-in mapper and `AZURE_OPENAI_MODEL_MAPPER`).
- `OPTIONS /v1/*` — CORS preflight.
- `ANY /v1/*` — catch-all proxy for OpenAI-style API paths (examples: `/v1/chat/completions`, `/v1/completions`, `/v1/embeddings`, `/v1/images/generations`, `/v1/responses`, audio endpoints).
- `ANY /deployments/*` — pass-through for deployment-scoped provider operations.

The proxy maps the `model` field (or `deployments/<name>` path) to a Foundry deployment name.

**Example Requests**
The proxy accepts common OpenAI headers (`Authorization: Bearer <token>` or `api-key`).

Chat completion example (curl):

```
curl -sS -X POST http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <CLIENT_TOKEN_FROM_AUTH_TOKENS>" \
  -d '{"model": "gpt-5.4","messages": [{"role":"user","content":"Hello from the proxy!"}],"max_tokens":200}'
```

List models:

`curl -sS http://localhost:11437/v1/models`

**Model Mapping**
Model mappings are defined in `pkg/azure/proxy.go` (see `FoundryModelMapper` and `initializeModelMapper()`). The repo includes a large set of built-in mappings.

Override or extend mappings with the `AZURE_OPENAI_MODEL_MAPPER` environment variable. Format: comma-separated `source=target` pairs (examples):

`AZURE_OPENAI_MODEL_MAPPER="gpt-5.4=deployment-123,gpt-5.4-mini=mini-deploy"`

Note: `pkg/azure` lower-cases and trims mapping keys during loading.

**Health & Diagnostics**
- `GET /healthz` — returns `200 OK` with `{"status":"healthy"}`.
- Basic logging is printed to stdout on startup and for each proxied request (see `pkg/azure/proxy.go`).

Common troubleshooting steps:
- Ensure `FOUNDRY_API_KEY` is set for outgoing Foundry authentication.
- Verify `AUTH_TOKENS` contains the client token you send to the proxy.
- Set `AZURE_FOUNDRY_ENDPOINT` to use workspace-level routing, or `AZURE_FOUNDRY_REGION` for per-deployment serverless routing.

**Contributing**
Contributions, bug reports and enhancements are welcome. If you modify model mappings or provider behavior, please document the changes and include examples. Key files to inspect when making changes: `main.go:1`, `pkg/azure/proxy.go:1`, and `pkg/openai/proxy.go:1`.
