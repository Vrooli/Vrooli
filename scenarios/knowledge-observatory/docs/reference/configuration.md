# Configuration

## Runtime Ports
Ports are assigned by the scenario lifecycle and exposed as environment variables.

- `API_PORT` (15000-19999)
- `UI_PORT` (35000-39999)

[CODE: .vrooli/service.json]

## API Runtime Configuration
Environment variables consumed by the API:

- `QDRANT_URL` / `QDRANT_API_KEY`
- `OLLAMA_EMBEDDING_MODEL` (model name; daemon traffic goes through `resource-ollama gateway`)
- `OLLAMA_STRUCTURED_OUTPUT_MODEL` (optional, used to coerce deep search output to JSON)
- `RESOURCE_QDRANT_CLI` (path to resource wrapper)
- `CORS_ALLOWED_ORIGINS` (comma-separated list)
- `SKIP_DB_CONNECT` or `SKIP_DB_TESTS` (disable DB connect)
- `VROOLI_SCENARIOS_ROOT` (absolute path to scenarios root for doc health)
- `VROOLI_REPO_ROOT` (repo root used to infer `scenarios/` for doc health)

[CODE: api/server.go]
[CODE: api/docs_root.go]

## Knowledge Schema Defaults
- Default collection: `knowledge_chunks_v1`
- Default visibility: `shared`
- Schema version: `ko.knowledge.v1`

[CODE: api/knowledge_schema.go]

## CLI Configuration
Stored at `${XDG_CONFIG_HOME:-~/.config}/vrooli/knowledge-observatory/config.json` (fallback: `~/.vrooli/config/knowledge-observatory/config.json`). Legacy installs may use `~/.knowledge-observatory/config.json`.

Keys:
- `api_base`
- `token`

Environment overrides:
- `KNOWLEDGE_OBSERVATORY_API_BASE`
- `KNOWLEDGE_OBSERVATORY_API_URL`
- `KNOWLEDGE_OBSERVATORY_API_PORT`
- `KNOWLEDGE_OBSERVATORY_API_TOKEN`
- `API_BASE_URL`
- `VITE_API_BASE_URL`
- `API_PORT`
- `VROOLI_API_TOKEN`

Config directory overrides:
- `KNOWLEDGE_OBSERVATORY_CONFIG_DIR`
- `VROOLI_CLI_CONFIG_DIR`

[CODE: cli/app.go]

## Optional Resources
- **Ollama** is optional but improves embedding quality.
- **Postgres** is required for metadata + job state.
- **Agent-manager** and **prompt-manager** should be running for deep documentation search.

[CODE: api/internal/adapters/embedder/ollama.go]
[CODE: api/internal/adapters/metadatastore/postgres.go]
