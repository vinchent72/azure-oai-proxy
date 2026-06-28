# Microsoft Foundry Proxy

A lightweight proxy that presents an OpenAI-compatible surface and forwards requests to Microsoft Foundry. It can also run in a pass-through OpenAI mode when you want to relay requests to `OPENAI_API_ENDPOINT` instead.

## What It Does

- Accepts OpenAI-style requests such as `/v1/chat/completions`, `/v1/responses`, `/v1/embeddings`, and `/v1/images/generations`.
- Maps model names to Microsoft Foundry deployment names with a built-in mapper plus optional overrides.
- Routes either to a workspace-level Foundry endpoint or to serverless per-deployment endpoints.
- Converts some requests and responses between Chat Completions, Responses, and Anthropic Messages formats when needed.
- Exposes `/healthz` and `/v1/models` for basic diagnostics.

## Running

### Prerequisites

- Go 1.24 or newer.

### Start Locally

```bash
go run .
```

Or build a binary first:

```bash
go build -o azure-oai-proxy .
./azure-oai-proxy
```

By default the proxy listens on `0.0.0.0:11437`.

## Configuration

The service reads configuration from environment variables at startup.

- `AZURE_OPENAI_PROXY_ADDRESS` sets the listen address. Default: `0.0.0.0:11437`.
- `AZURE_OPENAI_PROXY_MODE` selects the proxy mode. Default: `azure`.
- `AZURE_OPENAI_MODEL_MAPPER` adds or overrides `source=target` model mappings. Example: `gpt-5.4=deployment-name,gpt-5.4-mini=mini-deploy`.
- `AZURE_FOUNDRY_ENDPOINT` sets a workspace-level Foundry endpoint. When present, workspace routing is used.
- `AZURE_FOUNDRY_REGION` sets the serverless region used for per-deployment routing. Default: `westus`.
- `FOUNDRY_API_KEY` is the server-side key used when forwarding to Foundry.
- `AUTH_TOKENS` is a comma-separated allowlist of client tokens. Clients can send one of these in `Authorization: Bearer <token>` or `api-key`.
- `ANTHROPIC_APIVERSION` sets the Anthropic API version used for Claude routing. Default: `2023-06-01`.
- `AZURE_OPENAI_PROXY_DEBUG` enables verbose compatibility and stream translation logs when set to `true`, `1`, `yes`, or `on`.
- `OPENAI_API_ENDPOINT` overrides the upstream OpenAI endpoint when `AZURE_OPENAI_PROXY_MODE=openai`.

## Routes

- `GET /healthz` returns `{"status":"healthy"}`.
- `GET /v1/models` returns the currently known model names from the built-in mapper plus any overrides from `AZURE_OPENAI_MODEL_MAPPER`.
- `OPTIONS /v1/*` returns CORS preflight headers.
- `ANY /v1/*` forwards OpenAI-compatible requests.
- `ANY /deployments/*` forwards deployment-scoped requests.

## Azure Mode

When `AZURE_OPENAI_PROXY_MODE=azure`, the proxy uses Microsoft Foundry as the upstream.

- `POST /v1/chat/completions` is forwarded to Foundry chat, Responses, or Anthropic Messages depending on the model.
- `POST /v1/responses` is sanitized before forwarding. Chat-only models such as DeepSeek, Llama, and Qwen are translated to `/v1/chat/completions`, while partially compatible models can have unsupported tools removed.
- `POST /v1/completions`, `/v1/embeddings`, `/v1/images/generations`, `/v1/audio/*`, and `/v1/files` are routed through Foundry.
- Claude-family models are converted to Anthropic Messages API format before forwarding.
- Reasoning and Codex-style models are routed through the Responses API path.
- If `AZURE_FOUNDRY_ENDPOINT` is set, workspace routing is used; otherwise the proxy uses serverless deployment hostnames of the form `<deployment>.<region>.models.ai.azure.com`.

## OpenAI Mode

When `AZURE_OPENAI_PROXY_MODE=openai`, the proxy acts as a thin reverse proxy to `OPENAI_API_ENDPOINT` and preserves the requested path and query string.

## Example

```bash
curl -sS -X POST http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <CLIENT_TOKEN_FROM_AUTH_TOKENS>" \
  -d '{
    "model": "gpt-5.4",
    "messages": [
      { "role": "user", "content": "Hello from the proxy!" }
    ],
    "max_tokens": 200
  }'
```

List known models:

```bash
curl -sS http://localhost:11437/v1/models
```

## Model Mapping

Model resolution lives in `pkg/azure/proxy.go`. The built-in mapper covers a broad set of OpenAI, Claude, image, audio, and Foundry model names.

- Exact mappings win first.
- Version suffixes such as `-2026-06-10` are stripped when needed.
- Environment overrides from `AZURE_OPENAI_MODEL_MAPPER` take precedence over built-ins.
- Unknown model names fall back to the original value, which lets custom deployment names pass through.

## Notes

- Requests must include a token from `AUTH_TOKENS` when that allowlist is configured.
- `FOUNDRY_API_KEY` is only used on the server side and is never exposed to clients.
- The proxy logs request routing decisions and upstream errors to stdout.
- Verbose chunk-level translation logs are disabled by default and can be enabled with `AZURE_OPENAI_PROXY_DEBUG`.

## Repository Map

- `main.go` wires up the HTTP server, `/healthz`, and proxy routing.
- `pkg/azure/proxy.go` contains Foundry routing, model mapping, and response conversion logic.
- `pkg/azure/translator.go` handles Responses-to-chat translation for chat-only models.
- `pkg/openai/proxy.go` contains the OpenAI pass-through reverse proxy.
