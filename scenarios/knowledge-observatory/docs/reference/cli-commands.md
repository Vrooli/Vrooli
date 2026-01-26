# CLI Reference

Complete documentation for the knowledge-observatory CLI (`knowledge-observatory`).

[CODE: cli/app.go]

## Installation

```bash
cd scenarios/knowledge-observatory/cli
go build -o knowledge-observatory .
```

Or install via the shared installer (recommended):

```bash
./install.sh
```

## Global Options

```
--api-base <url>   Override API base URL (default: auto-detected from lifecycle)
--auto-start       Auto-start the scenario if not running
--no-color         Disable ANSI color output (or set NO_COLOR env var)
--color            Force-enable ANSI color output
```

## Commands Overview

| Command | Description |
|---------|-------------|
| `status` | Check API health |
| `search` | Semantic search over knowledge |
| `ingest` | Ingest a single record |
| `ingest-job` | Enqueue an async document ingest job |
| `job-status` | Fetch ingest job status |
| `health` | Knowledge health metrics |
| `metrics` | Alias for `health` |
| `graph` | Generate a knowledge graph |
| `configure` | View/update CLI settings |
| `version` | Show CLI version |
| `help` | Show help |

---

## status

```bash
knowledge-observatory status [--json]
```

## search

```bash
knowledge-observatory search "agent workflows" --limit 10 --threshold 0.35
```

**Options:**
| Flag | Description |
|------|-------------|
| `--collection` | Collection name |
| `--namespaces` | Comma-separated namespaces |
| `--visibility` | Comma-separated visibility values |
| `--tags` | Comma-separated tags |
| `--ingested-after` | RFC3339 timestamp filter |
| `--ingested-before` | RFC3339 timestamp filter |
| `--limit` | Max results |
| `--threshold` | Score threshold |

## ingest

```bash
knowledge-observatory ingest --namespace docs --content "Knowledge Observatory entry" --visibility shared
```

Content can also be passed as positional arguments or via stdin.

**Options:**
| Flag | Description |
|------|-------------|
| `--namespace` | Namespace (required) |
| `--collection` | Collection name |
| `--visibility` | Visibility (shared/private/restricted) |
| `--record-id` | Explicit record ID |
| `--external-id` | External identifier |
| `--tags` | Comma-separated tags |
| `--metadata` | Metadata JSON object |
| `--source` | Source identifier |
| `--source-type` | Source type |
| `--content` | Content string |

## ingest-job

```bash
knowledge-observatory ingest-job --namespace docs --content "$(cat README.md)" --chunk-size 1200 --chunk-overlap 150
```

**Options:**
| Flag | Description |
|------|-------------|
| `--namespace` | Namespace (required) |
| `--collection` | Collection name |
| `--visibility` | Visibility (shared/private/restricted) |
| `--document-id` | Explicit document ID |
| `--external-id` | External identifier |
| `--tags` | Comma-separated tags |
| `--metadata` | Metadata JSON object |
| `--source` | Source identifier |
| `--source-type` | Source type |
| `--chunk-size` | Chunk size override |
| `--chunk-overlap` | Chunk overlap override |
| `--content` | Content string |

## job-status

```bash
knowledge-observatory job-status <job_id>
```

## health / metrics

```bash
knowledge-observatory health
knowledge-observatory metrics
knowledge-observatory health --watch --interval 10s
```

## graph

```bash
knowledge-observatory graph --center "ecosystem-manager" --depth 2
```

**Options:**
| Flag | Description |
|------|-------------|
| `--center` | Center concept (required) |
| `--collection` | Collection name |
| `--namespaces` | Comma-separated namespaces |
| `--visibility` | Comma-separated visibility values |
| `--tags` | Comma-separated tags |
| `--depth` | Graph traversal depth |
| `--limit` | Max nodes |
| `--threshold` | Score threshold |

## configure

```bash
knowledge-observatory configure
knowledge-observatory configure api_base http://localhost:<API_PORT>
knowledge-observatory configure token <api_token>
```

## Environment overrides

The CLI respects scenario-specific environment variables:
- `KNOWLEDGE_OBSERVATORY_API_BASE`
- `KNOWLEDGE_OBSERVATORY_API_URL`
- `KNOWLEDGE_OBSERVATORY_API_PORT`
- `KNOWLEDGE_OBSERVATORY_API_TOKEN`

And these shared overrides:
- `API_BASE_URL`
- `VITE_API_BASE_URL`
- `API_PORT`
- `VROOLI_API_TOKEN`
