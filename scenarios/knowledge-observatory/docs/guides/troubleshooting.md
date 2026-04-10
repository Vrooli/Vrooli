# Troubleshooting

## API not reachable
- Confirm the scenario is running: `make start` or `vrooli scenario status knowledge-observatory`.
- Ensure the API port is set: `API_PORT` from scenario status.
- If CLI cannot resolve the API, set it explicitly:

```bash
knowledge-observatory configure api_base http://localhost:<API_PORT>
```

[CODE: cli/app.go]
[CODE: api/server.go]

## Qdrant errors or empty results
- Verify Qdrant is running (`vrooli resource status qdrant`).
- Ensure the collection exists or ingest new data (upsert creates collections).
- If results are sparse, lower the threshold parameter.

[CODE: api/internal/adapters/vectorstore/qdrant.go]
[CODE: api/search.go]

## Postgres connection failures
- Check Postgres resource status.
- If running tests, `SKIP_DB_CONNECT=1` can bypass DB connection.

[CODE: api/server.go]

## Embedding failures
- Ollama must be reachable if embeddings are enabled.
- Confirm `OLLAMA_URL` and `OLLAMA_EMBEDDING_MODEL` are set.

[CODE: api/internal/adapters/embedder/ollama.go]

## CORS issues for the UI
- Set `CORS_ALLOWED_ORIGINS` to the UI origin when hosting separately.

[CODE: api/server.go]

## CLI config reset
- Remove the config file to reinitialize:

```bash
rm -f ~/.config/vrooli/knowledge-observatory/config.json
rm -f ~/.vrooli/config/knowledge-observatory/config.json
rm -f ~/.knowledge-observatory/config.json
```

[CODE: cli/app.go]
