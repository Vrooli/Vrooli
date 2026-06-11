# Configuration

## Runtime Ports
Ports are assigned by the scenario lifecycle and exposed as environment variables.

- `API_PORT` (15000-19999)
- `UI_PORT` (35000-39999)

[CODE: .vrooli/service.json]

## API Runtime Configuration
Environment variables consumed by the API:

- `QDRANT_URL` / `QDRANT_API_KEY`
- `OLLAMA_EMBEDDING_ROLE` (embedding role; daemon traffic goes through `resource-ollama gateway`)
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

## Docs Search Tuning (`.vrooli/search.json`)

The docs-search tuning factors live in the scenario-owned SSOT
`.vrooli/search.json`, not in env vars. KO is the **docs** consumer of the shared
search-tuning contract; the schema + per-knob dashboard live in the engine package
[`packages/ai-go/search/docs/reference/search-json.md`](../../../../packages/ai-go/search/docs/reference/search-json.md).

| What | Value | Notes |
|---|---|---|
| Provider id | `knowledge-observatory.docs` | the project documentation corpus. |
| Engine | `hybrid` | dense + BM25 sparse with RRF fusion — recall on long-form prose. |
| `embed_task_prefix` | `false` | symmetric embeddings — **load-bearing** (see below). |
| `rerank_enabled` / `rerank_blend` | `false` / `false` | hybrid RRF + authority boosting ties the cross-encoder on recall here; reranking buys ordering parity, not recall, so it is OFF by default. |

This is the measured-best `aisearch.DocCorpusTuning()` preset. At boot the
`loadDocsTuning()` helper reads the `tuning` block from `search.json` and wires the
hybrid engine from it; `server.go` self-registers the file with `search-hub`
(idempotent upsert, best-effort, search-hub is an optional dependency).

> **Recall guard — do not change `embed_task_prefix` or the rerank defaults
> without re-measuring.** KO's guarded **recall@5 = 0.818** baseline rests on the
> symmetric embedder (`embed_task_prefix:false`, which keeps the recipe-aware
> drift hash byte-identical for the empty recipe) and rerank-off. Any sweep that
> proposes an index-time change must re-run the accuracy gate
> (`KO_AISEARCH_LIVE=1 go test ./internal/aisearch/ -run TestAccuracyCorpus`) and
> confirm recall@5 stays ≥ 0.818 before the tuning is written back. The
> `testdata/search_queries.json` corpus (186 cases) is the per-build smoke gate;
> the golden `tests` corpus in `search.json` is what the search-hub sweep
> optimizes against.

[CODE: api/server.go]
[CODE: .vrooli/search.json]
