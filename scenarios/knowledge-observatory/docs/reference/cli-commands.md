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
| `ingest-health` | Inspect ingest queue/runner health |
| `collection-diagnostics` | Inspect embedding/chunk diagnostics for a collection |
| `collection-prune-stale` | Prune stale chunk versions (dry-run by default) |
| `collection-dedupe` | Remove duplicate content chunks (dry-run by default) |
| `document-delete` | Delete chunks for a document (dry-run by default) |
| `health` | Knowledge health metrics |
| `metrics` | Alias for `health` |
| `graph` | Generate a knowledge graph |
| `docs` | Documentation explorer commands |
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
| `--visibility` | Visibility (shared, private, restricted) |
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
| `--visibility` | Visibility (shared, private, restricted) |
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
knowledge-observatory job-status "<job_id>"
```

## ingest-health

```bash
knowledge-observatory ingest-health
knowledge-observatory ingest-health --watch --interval 10s
```

## collection-diagnostics

```bash
knowledge-observatory collection-diagnostics knowledge
knowledge-observatory collection-diagnostics --collection knowledge --mode full --limit 5000
```

**Options:**
| Flag | Description |
|------|-------------|
| `--collection` | Collection name (optional if provided as positional argument) |
| `--mode` | `sample` or `full` |
| `--limit` | Max points to inspect |

## collection-prune-stale

```bash
knowledge-observatory collection-prune-stale knowledge
knowledge-observatory collection-prune-stale --collection knowledge --apply --max-deletes 200
```

**Options:**
| Flag | Description |
|------|-------------|
| `--collection` | Collection name (optional if provided as positional argument) |
| `--dry-run` | Preview only (default true) |
| `--apply` | Execute deletion |
| `--max-deletes` | Max stale points to delete |

## collection-dedupe

```bash
knowledge-observatory collection-dedupe knowledge
knowledge-observatory collection-dedupe --collection knowledge --apply --max-deletes 200
```

**Options:**
| Flag | Description |
|------|-------------|
| `--collection` | Collection name (optional if provided as positional argument) |
| `--dry-run` | Preview only (default true) |
| `--apply` | Execute deletion |
| `--max-deletes` | Max duplicate points to delete |

## document-delete

```bash
knowledge-observatory document-delete --namespace docs --document-id doc-123
knowledge-observatory document-delete --namespace docs --external-id ext-123 --apply
```

**Options:**
| Flag | Description |
|------|-------------|
| `--namespace` | Namespace (required) |
| `--collection` | Collection override |
| `--document-id` | Document identifier |
| `--external-id` | External identifier (resolved server-side) |
| `--dry-run` | Preview only (default true) |
| `--apply` | Execute deletion |

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

## docs

```bash
knowledge-observatory docs search-files "**/README.md" --scope scenario --scenario knowledge-observatory
knowledge-observatory docs search-text "health score" --scope scenario --scenario knowledge-observatory --context-lines 1
knowledge-observatory docs search-deep "How does deep search work?" --scope scenario --scenario knowledge-observatory --max-results 5
knowledge-observatory docs scenarios
knowledge-observatory docs tree knowledge-observatory
knowledge-observatory docs health knowledge-observatory
knowledge-observatory docs view "scenarios/knowledge-observatory/docs/manifest.json" --format preview
knowledge-observatory docs reset "scenarios/knowledge-observatory/docs/internal/PROBLEMS.md" --max-age-days 30 --keep-min-entries 3 --preview
knowledge-observatory docs heal knowledge-observatory --dry-run --wait
knowledge-observatory docs heal-status "<job_id>"
```

### docs search-files

**Options:**
| Flag | Description |
|------|-------------|
| `--pattern` | Glob pattern (optional if positional pattern is provided) |
| `--scope` | Scope: global, scenario, or path |
| `--scenario` | Scenario name (required for scope=scenario) |
| `--base-path` | Base path (required for scope=path) |
| `--limit` | Max results |
| `--include-content` | Include content preview |

### docs search-text

**Options:**
| Flag | Description |
|------|-------------|
| `--query` | Text query (optional if positional query is provided) |
| `--scope` | Scope: global, scenario, or path |
| `--scenario` | Scenario name (required for scope=scenario) |
| `--base-path` | Base path (required for scope=path) |
| `--file-types` | Comma-separated file extensions (md, json, txt, etc.) |
| `--case-sensitive` | Case-sensitive search |
| `--limit` | Max results |
| `--context-lines` | Lines of context before/after matches |

### docs search-deep

**Options:**
| Flag | Description |
|------|-------------|
| `--query` | Deep search query (optional if positional query is provided) |
| `--scope` | Scope: global, scenario, or path |
| `--scenario` | Scenario name (required for scope=scenario) |
| `--base-path` | Base path (required for scope=path) |
| `--max-results` | Max results |
| `--follow-refs` | Follow referenced docs (default true) |
| `--timeout-seconds` | Agent timeout in seconds |
| `--wait` | Wait for completion before exiting |

### docs scenarios

List all scenarios with documentation stats.

```bash
knowledge-observatory docs scenarios
```

### docs tree

Fetch the documentation tree for a scenario.

**Options:**
| Flag | Description |
|------|-------------|
| `--scenario` | Scenario name (optional if provided as positional argument) |

### docs health

Fetch documentation health details for a scenario.

```bash
knowledge-observatory docs health knowledge-observatory
knowledge-observatory docs health knowledge-observatory --json
```

**Options:**
| Flag | Description |
|------|-------------|
| `--scenario` | Scenario name (optional if provided as positional argument) |
| `--scope` | `scenario` (default) or `path` |
| `--path` | Docs path to scan when using path scope |
| `--checks` | Comma-separated checks: `structure`, `content`, `links`, `refs`, `commands`, `manifest`, `numbers` |
| `--strict-external-links` | Treat external link failures as failures |
| `--require-all-docs-registered` | Report scenario docs missing from `docs/manifest.json` |
| `--skip-external-links` | Skip external link probing for offline runs |
| `--json` | Emit raw JSON output |

The `refs` check validates explicit marked references such as `cli:...`.
The `commands` check conservatively validates Vrooli-owned commands found in
fenced shell snippets by delegating to CLI Health (DOCS policy); it does not
execute the referenced commands.

Command-snippet finding codes:
| Code | Severity | Meaning |
|------|----------|---------|
| `broken_command_snippet` | warning | Snippet is invalid: unknown path, bad arguments, `enum_placeholder_mismatch` (an `"<a\|b\|c>"` alternation drifted from the manifest/proto vocabulary), or `invalid_literal_value` (an example value violates descriptor-derived constraints) |
| `placeholder_style` | warning | Snippet is correct but uses unquoted `<...>` placeholders; the finding carries a byte-exact quoted fix that `docs fix-placeholders` applies deterministically |
| `partial_command_snippet` | info | Command path exists but argument metadata was unavailable |
| `unknown_command_snippet` | warning | Validation could not complete (CLI Health unreachable, unknown owner) |

The preferred documentation convention is quoted placeholders: `"<session>"`
for a named slot, `"<minor|moderate|major|architectural>"` for an enum whose
alternation must exactly match the owning manifest's `values` (union any proto
enum). Quoting keeps snippets shell-safe when pasted verbatim and lets the
enum vocabulary be machine-checked instead of hand-maintained prose.

Human output includes the shared health maturity report. For Knowledge
Observatory docs health, that report separates documentation contract, required
docs, append-log integrity, content quality, link health, reference integrity,
and manifest coverage so operators can see the highest-priority documentation
capability instead of one overloaded local ladder. The `--json` form preserves
the complete shared `assessment` payload, including `assessment.local` for
legacy consumers and `assessment.capabilities[]` for capability-aware tooling.

### docs view

Fetch document content for a given path.

### docs heal

Spawn a documentation healing agent.

**Options:**
| Flag | Description |
|------|-------------|
| `--scenario` | Scenario name (optional if provided as positional argument) |
| `--issues` | Comma-separated issue labels to target |
| `--auto-approve` | Auto-approve if health improves |
| `--dry-run` | Preview-only healing (no apply) |
| `--wait` | Wait for completion before exiting |

### docs heal-status

Fetch status for a healing job by ID.

**Options:**
| Flag | Description |
|------|-------------|
| `--job-id` | Healing job ID (optional if provided as positional argument) |

### docs fix-placeholders

Apply the deterministic quoted-placeholder fixes for a scenario's markdown
command snippets. A thin wrapper over the shared scenario-validation
`PreviewFix`/`ApplyFix` RPC scoped to the `placeholder_style` rule: the server
re-validates every snippet through CLI Health and applies each returned
byte-exact fix verbatim (never recomputed). Idempotent — a second run reports
zero candidates.

```bash
knowledge-observatory docs fix-placeholders "<scenario>" --dry-run   # unified diff, no writes
knowledge-observatory docs fix-placeholders "<scenario>"             # apply
```

**Options:**
| Flag | Description |
|------|-------------|
| `--scenario` | Scenario name (optional if provided as positional argument) |
| `--dry-run` | Preview the unified diff without writing; selects exactly the files/lines the apply path would touch |
| `--json` | Emit the raw FixResponse |

### docs reset

Reset/clean supported documents that declare `operations.appendLog` with
reset support in the resolved documentation manifest.

**Options:**
| Flag | Description |
|------|-------------|
| `--path` | Document path (optional if provided as positional argument) |
| `--max-age-days` | Remove entries older than N days |
| `--keep-min-entries` | Always keep at least N entries |
| `--preview` | Preview changes without writing |

## configure

```bash
knowledge-observatory configure
knowledge-observatory configure api_base http://localhost:"<API_PORT>"
knowledge-observatory configure token "<api_token>"
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
