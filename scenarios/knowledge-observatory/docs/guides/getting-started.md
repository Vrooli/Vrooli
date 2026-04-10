# Getting Started Guide

This guide walks through the UI, CLI, and API to confirm Knowledge Observatory is working end to end.

## End-to-End Flow
1. Start the scenario (`make start`).
2. Confirm API health from the UI dashboard or CLI.
3. Ingest a record or document.
4. Run a search and inspect metadata.
5. Explore graph and metrics panels.

## UI Walkthrough
- **Dashboard**: run quick search, confirm knowledge + documentation health, review activity feed.
- **Search**: select semantic, file, text, unified, or deep search modes.
- **Explorer**: browse scenario docs, tree warnings, health summary, and healing controls.
- **Viewer**: open document content with preview/mermaid rendering and reset support.
- **Graph**: explore relationships around a center concept.
- **Metrics**: review coherence/freshness/redundancy scores.

[CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
[CODE: ui/src/surfaces/search/SearchPage.tsx]
[CODE: ui/src/surfaces/explorer/ExplorerPage.tsx]
[CODE: ui/src/surfaces/viewer/ViewerPage.tsx]
[CODE: ui/src/surfaces/graph/GraphPage.tsx]
[CODE: ui/src/surfaces/metrics/MetricsPage.tsx]

## CLI Walkthrough

```bash
knowledge-observatory status
knowledge-observatory ingest --namespace docs --content "Knowledge Observatory quickstart."
knowledge-observatory search "quickstart"
knowledge-observatory docs search-files "**/README.md"
knowledge-observatory docs health knowledge-observatory
knowledge-observatory graph --center "quickstart"
knowledge-observatory health
```

[CODE: cli/app.go]

## API Walkthrough

```bash
API_PORT=$(vrooli scenario status knowledge-observatory --json | jq -r '.ports.API_PORT')

curl -X POST "http://localhost:${API_PORT}/api/v1/knowledge/records/upsert" \
  -H "Content-Type: application/json" \
  -d '{"namespace":"docs","content":"Knowledge Observatory quickstart."}'

curl -X POST "http://localhost:${API_PORT}/api/v1/knowledge/search" \
  -H "Content-Type: application/json" \
  -d '{"query":"quickstart"}'

curl -X GET "http://localhost:${API_PORT}/api/v1/knowledge/health"
```

[CODE: api/ingest.go]
[CODE: api/search.go]
[CODE: api/metrics.go]

## Related Reference
- [DOC: docs/reference/api-endpoints.md]
- [DOC: docs/reference/cli-commands.md]
