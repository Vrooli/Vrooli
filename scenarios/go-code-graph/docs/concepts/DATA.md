# Data — Go Code Graph

This document is the canonical data ownership and storage map for the scenario. Update it when domains add tables, files, blobs, external records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

Go Code Graph is a **stateless** scenario in v1. `Extract` calls do not persist anything. `Rewrite plan` stores plans in an in-process registry with a 5-minute TTL (lost on restart, by design — operators must re-plan).

The optional **Operation Log** (P1, REQ-P1-002) is the only persisted data. It lives in embedded SQLite via `modernc.org/sqlite`, with `SQLITE_PATH` provided by the lifecycle's `.vrooli/service.json`. The Operation Log records each `RewriteApply` invocation for audit purposes.

External storage resources (Postgres, Qdrant, Ollama, etc.) are **not** required and **not** anticipated for this scenario. Document any additions in [`INTEGRATIONS.md`](INTEGRATIONS.md) before editing `.vrooli/service.json`.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Extracted Graph | graph | In-memory (returned in the response) | The target Go module's source code | Not retained; computed fresh each call | The graph is derived from source, not persisted. Consumers cache at their layer. |
| Plan registry | rewrite | In-process map keyed by `plan_id` | The normalized operation list | 5-minute TTL, lost on restart | Deliberately ephemeral — apply requires a fresh plan, no resurrection across process boundaries. |
| Operation Log (P1) | rewrite | SQLite | `api/internal/rewrite/schema.sql` | Indefinite (audit trail) | Append-only. One row per `RewriteApply` invocation including success/failure status per op. |
| Recent-calls telemetry | explorer | In-memory bounded ring buffer (256 entries) | Live extraction activity | Lost on restart | UI-only diagnostic; not part of the durable contract. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `operation_log` (P1) | rewrite | `api/internal/rewrite/schema.sql` (planned) | rewrite repository + audit query handler |
| System schema | infrastructure | `api/internal/database/system.sql` | API boot |

`operation_log` planned schema:

```sql
CREATE TABLE IF NOT EXISTS operation_log (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  plan_id     TEXT    NOT NULL,
  module_path TEXT  NOT NULL,
  applied_at  INTEGER NOT NULL, -- unix epoch seconds
  ops_json    TEXT    NOT NULL, -- normalized operation list with per-op status
  succeeded   INTEGER NOT NULL  -- 1 = all ops applied; 0 = partial or full failure
);

CREATE INDEX IF NOT EXISTS operation_log_path_idx ON operation_log(module_path, applied_at DESC);
```

## Migrations And Compatibility

The scenario uses idempotent schema bootstrap (`CREATE TABLE IF NOT EXISTS`). The Operation Log table is additive; no destructive migrations anticipated in v1.

If a future capability adds tables, follow the template convention:
- One `.sql` file per domain, alongside the domain code.
- Idempotent statements only.
- Document the migration plan in [`../internal/DECISIONS.md`](../internal/DECISIONS.md) before adding column drops, renames, or backfills.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Extracted Graph | `common.v1.code_graph.Graph` proto, also serialized as JSON via Connect-JSON transport | graph | active — every Extract call exports the graph |
| Operation Log query (P1) | JSON via Connect-RPC `ListOperations` | rewrite | planned |

No file-based import/export in v1. Consumers receive the graph over the wire and persist it themselves if needed (cartographer's snapshot store is the canonical example).

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Plan registry | TTL expiry (5 min) or process exit | Ephemeral by design | None — this is intentional |
| Operation Log (P1) | Manual purge via CLI (planned) | Indefinite by default | Define purge policy if/when storage growth becomes a concern |
| Recent-calls telemetry | Ring buffer wrap (256 entries) or process exit | Ephemeral, bounded | None |

## Privacy Notes

`Extract` and `Rewrite` **read source code** from the target Go module. Source code may contain credentials, customer identifiers, hostnames, or other regulated data if the scenario maintainer has been careless. The graph response itself contains **structure** (file paths, declaration names, import edges, leading comments via P1 method-set work) but not source bodies. However:

- File paths are surfaced in the graph and the Operation Log. Sensitive paths (e.g. `/secret/keys.go`) appear verbatim.
- Declaration names are surfaced. Sensitive names (e.g. `apiKey`, `dbPassword`) appear verbatim.
- Warning messages may quote source-line snippets when surfacing parse errors.

This is acceptable for a local infrastructure scenario but should be revisited if go-code-graph is ever exposed beyond a single-operator local install. See [`../internal/SECURITY.md`](../internal/SECURITY.md) for the full threat model.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
