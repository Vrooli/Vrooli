# Configuration

## Runtime Ports
Ports are assigned by the scenario lifecycle and exposed as environment variables.

- `API_PORT` (15000-19999)
- `UI_PORT` (35000-39999)

[CODE: .vrooli/service.json]

## API Runtime Configuration
Environment variables consumed by the API:

- `QDRANT_URL` / `QDRANT_API_KEY`
- `OLLAMA_URL` / `OLLAMA_EMBEDDING_MODEL`
- `RESOURCE_QDRANT_CLI` (path to resource wrapper)
- `CORS_ALLOWED_ORIGINS` (comma-separated list)
- `SKIP_DB_CONNECT` or `SKIP_DB_TESTS` (disable DB connect)

[CODE: api/server.go]

## Knowledge Schema Defaults
- Default collection: `knowledge_chunks_v1`
- Default visibility: `shared`
- Schema version: `ko.knowledge.v1`

[CODE: api/knowledge_schema.go]

## CLI Configuration
Stored at `~/.knowledge-observatory/config.json`.

Keys:
- `api_base`
- `api_token`
- `output_format`

Environment overrides:
- `KNOWLEDGE_OBSERVATORY_API_BASE`
- `KNOWLEDGE_OBSERVATORY_API_TOKEN`

[CODE: cli/knowledge-observatory]

## Optional Resources
- **Ollama** is optional but improves embedding quality.
- **Postgres** is required for metadata + job state.

[CODE: api/internal/adapters/embedder/ollama.go]
[CODE: api/internal/adapters/metadatastore/postgres.go]
