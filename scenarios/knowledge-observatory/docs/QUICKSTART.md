# Quickstart

Get Knowledge Observatory running and perform a first search in under 5 minutes.

## Prerequisites
- Qdrant and Postgres resources running (required). Optional: Ollama for embeddings.
- Scenario installed via Vrooli lifecycle.

## Start the scenario
From the scenario directory:

```bash
make start
```

Find the assigned ports:

```bash
vrooli scenario status knowledge-observatory
```

## Open the UI
Navigate to the UI port reported above:

```
http://localhost:${UI_PORT}
```

The dashboard surfaces health status and quick links. [REQ: OT-P0-006]

## Run a search (CLI)

```bash
knowledge-observatory search "How do scenarios work?"
```

This uses the API search endpoint. [REQ: OT-P0-001]
[CODE: cli/app.go]
[CODE: api/search.go]

## Ingest a record (CLI)

```bash
knowledge-observatory ingest --namespace ecosystem-manager --content "Scenarios are reusable capabilities composed from resources."
```

This writes through the canonical record upsert path. [REQ: OT-P0-004]
[CODE: api/ingest.go]

## Next steps
- Explore the graph view in the UI or via CLI. [REQ: OT-P0-003]
- Review health/quality metrics. [REQ: OT-P0-002]
- See: [DOC: docs/guides/getting-started.md#end-to-end-flow]
