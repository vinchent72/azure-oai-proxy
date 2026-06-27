# Microsoft Foundry Proxy

This repository is forked from **Gyarbij/azure-oai-proxy** and modified to work with Microsoft Foundry. It is a lightweight OpenAI-compatible proxy that forwards OpenAI-format requests to Microsoft Azure Foundry/AI endpoints. This project lets applications written for the OpenAI API run against Azure Foundry deployments with minimal or no code changes.

**Key points**
- Listens for OpenAI-compatible requests and translates them to Azure Foundry (Responses API) calls.
- Default listening address: `0.0.0.0:11437` (configurable).
- Exposes a small set of OpenAI-compatible endpoints (see Supported Endpoints).
- Basic model mapping and provider configuration are defined in `main.go` and can be adapted to your environment.

**Status**
- Proxy mode: `azure` (default).
- Wire API: `responses` (default provider config).

**License**
- See the repository `LICENSE` file for license details.

**Table Of Contents**
- Overview
- Quickstart
- Configuration
- Supported Endpoints
- Example Requests
- Model Mapping
- Health and Diagnostics
- Contributing


Overview
--------
This proxy accepts OpenAI-style requests (for example, `/v1/chat/completions`) and forwards them to Azure Foundry endpoints while performing the required URL, header, and payload translations. It's useful to run OpenAI-compatible tools (LangChain, Open WebUI, custom apps) against Azure-managed models without rewriting the client code.


Quickstart
----------
Prerequisites:
- Go 1.20+ (this repository uses Go modules).

Build and run locally:

- Run directly:

  `go run .`

- Build a binary and run:

  `go build -o azure-oai-proxy .`
  `./azure-oai-proxy`

The proxy will listen on `0.0.0.0:11437` by default.


Configuration
-------------
The proxy reads a few environment variables at startup (see `main.go:1` for the code locations):

- `AZURE_OPENAI_PROXY_ADDRESS` — override the listen address (default: `0.0.0.0:11437`).
- `AZURE_OPENAI_PROXY_MODE` — proxy mode (default: `azure`).
- `FOUNDRY_PROVIDER_BASE_URL` — base URL for the Foundry provider. If not set, a fallback URL is used in the code (used to build outgoing Foundry requests).

Provider configuration and model presets are defined inside `main.go`. See `main.go:1` for the initial `ModelsRegistry` and `ProvidersRegistry` declarations and defaults.


Supported Endpoints
-------------------
The proxy exposes these OpenAI-compatible endpoints (requests are routed/translated to the configured provider):

- `GET /healthz` — health check (returns `{"status":"healthy"}`).
- `GET /v1/models` — list the models known to the proxy (from the in-memory registry).
- `OPTIONS /v1/*` — CORS preflight support.
- `POST/ANY /v1/*` — catch-all proxy for OpenAI-style API paths such as:
  - `/v1/chat/completions`
  - `/v1/completions`
  - `/v1/embeddings`
  - `/v1/images/generations`
  - `/v1/responses` (Azure Responses API)
  - audio endpoints (transcriptions, speech)
- `ANY /deployments/*` — pass-through for provider deployment operations.

Note: This repository includes code for many OpenAI model name mappings and an extensive built-in model list — the proxy maps incoming OpenAI model names to provider deployment identifiers.


Example Requests
----------------
Send OpenAI-compatible requests to the proxy. The proxy accepts the common OpenAI headers (for example `api-key` or `Authorization`), and forwards requests to the configured Foundry provider after converting the payload.

Example: chat completion (curl)

```
curl -sS -X POST http://localhost:11437/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "api-key: YOUR_API_KEY_OR_TOKEN" \
  -d '{
    "model": "foundry-gpt-5-mini",
    "messages": [{"role":"user","content":"Hello from the proxy!"}],
    "max_tokens": 200
  }'
```

The proxy will map `foundry-gpt-5-mini` to the configured provider model/deployment and call the provider's Responses API (or other wire API configured).


Model Mapping
-------------
Model presets are seeded inside `main.go` using `ModelsRegistry`. Example defaults include `foundry-gpt-5-mini`, `foundry-gpt-5.4-mini`, and `foundry-gpt-5.4`.

To add or change mappings you can edit `main.go` (search for `ModelsRegistry` initialization) or extend the proxy to load model mappings from a JSON file or environment variable.

Files to inspect for customization:

- `main.go:1` — startup, `ModelsRegistry`, `ProvidersRegistry`, and environment variable handling.
- `pkg/azure/proxy.go` and `pkg/openai/proxy.go` — translation logic between OpenAI-format requests and Azure Foundry wire formats.


Health and Diagnostics
----------------------
- `GET /healthz` — returns `200 OK` with `{"status":"healthy"}`.
- Basic logging is printed to stdout on startup and when routing requests.


Contributing
------------
Contributions, bug reports and enhancements are welcome. If you modify model mappings or provider behavior, please document the changes and consider adding configuration options rather than hard-coding values.


Where to look next
------------------
- `main.go:1` — entrypoint, model and provider registries, and routing.
- `pkg/azure/proxy.go` — Azure Foundry request building and response handling.
- `pkg/openai/proxy.go` — OpenAI-format parsing and normalization.


