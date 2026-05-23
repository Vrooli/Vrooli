# Data — TypeScript Code Graph

This document is the canonical data ownership and storage map for the scenario. Update it when domains add tables, files, blobs, external records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

TypeScript Code Graph is a **stateless** scenario in v1. `Extract` calls do not persist anything. `Rewrite plan` stores plans in an in-process registry with a 5-minute TTL. The Node sidecar holds `ts-morph` Project state in memory during a call; nothing crosses process restarts.

The optional **Operation Log** (P1, REQ-P1-002) is the only persisted data. It lives in embedded SQLite via `modernc.org/sqlite`, with `SQLITE_PATH` provided by the lifecycle's `.vrooli/service.json`.

External storage resources (Postgres, Qdrant, Ollama, etc.) are **not** required and **not** anticipated.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Extracted Graph | graph | In-memory (returned in the response) | The target TS project's source code | Not retained; computed fresh each call | The graph is derived from source, not persisted. Consumers cache at their layer. |
| Leading-comment metadata | graph | In-memory (embedded in graph response) | Source comments in the target project | Not retained | Per declaration; verbatim from source. Load-bearing for `react-component-library`'s migration. |
| Plan registry | rewrite | In-process map keyed by `plan_id` | The normalized operation list | 5-minute TTL, lost on restart | Deliberately ephemeral. |
| Operation Log (P1) | rewrite | SQLite | `api/internal/rewrite/schema.sql` | Indefinite (audit trail) | Append-only. |
| Sidecar process state | sidecar | OS process memory (Node child) | Live `ts-morph` Project objects | Process lifetime; reset on restart | The sidecar holds no durable state. |
| Sidecar lifecycle status | sidecar | In-memory state machine | Supervisor goroutine | Lost on restart | Surfaced via `/health` and the diagnostics page. |
| Recent-calls telemetry | explorer | In-memory bounded ring buffer (256 entries) | Live extraction activity | Lost on restart | UI-only diagnostic. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `operation_log` (P1) | rewrite | `api/internal/rewrite/schema.sql` (planned) | rewrite repository + audit query handler |
| System schema | infrastructure | `api/internal/database/system.sql` | API boot |

`operation_log` planned schema (identical shape to go-code-graph):

```sql
CREATE TABLE IF NOT EXISTS operation_log (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  plan_id     TEXT    NOT NULL,
  scenario_path TEXT  NOT NULL,
  applied_at  INTEGER NOT NULL, -- unix epoch seconds
  ops_json    TEXT    NOT NULL, -- normalized operation list with per-op status
  succeeded   INTEGER NOT NULL  -- 1 = all ops applied; 0 = partial or full failure
);

CREATE INDEX IF NOT EXISTS operation_log_path_idx ON operation_log(scenario_path, applied_at DESC);
```

## Migrations And Compatibility

The scenario uses idempotent schema bootstrap (`CREATE TABLE IF NOT EXISTS`). The Operation Log table is additive; no destructive migrations anticipated in v1.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Extracted Graph | `common.v1.code_graph.Graph` proto, also serialized as JSON via Connect-JSON transport | graph | active — every Extract call exports the graph |
| Operation Log query (P1) | JSON via Connect-RPC `ListOperations` | rewrite | planned |

No file-based import/export in v1.

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Plan registry | TTL expiry (5 min) or process exit | Ephemeral by design | None |
| Operation Log (P1) | Manual purge via CLI (planned) | Indefinite by default | Define purge policy if storage growth concerns emerge |
| Sidecar process state | Sidecar restart | Ephemeral by design | None |
| Recent-calls telemetry | Ring buffer wrap (256) or process exit | Ephemeral, bounded | None |

## Privacy Notes

`Extract` reads source code from the target TS project. Importantly, **leading comments are extracted verbatim** as part of the graph response. This is the load-bearing contract that lets `react-component-library` migrate off its current regex parser onto a typed `typescript-code-graph` client. The privacy implication: any sensitive data accidentally pasted into a JSDoc comment (e.g. an API key in `@example` snippets) will appear in the graph response.

In addition to comments:
- File paths are surfaced in the graph and the Operation Log. Sensitive paths appear verbatim.
- Declaration names are surfaced. Sensitive names appear verbatim.
- Warning messages may quote source-line snippets when surfacing parse errors.

This is acceptable for a local infrastructure scenario but should be revisited if typescript-code-graph is ever exposed beyond a single-operator local install. See [`../internal/SECURITY.md`](../internal/SECURITY.md) for the full threat model.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
