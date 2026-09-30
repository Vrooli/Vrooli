# Data — Quality Health

## Purpose Of This Document

This document records what Quality Health persists, what remains live-only, and what future run-history storage would own.

## Storage Overview

Quality Health v1 should be able to run as a stateless live audit scenario. The generated scaffold includes SQLite lifecycle wiring, but static quality audits do not require persistence until run history, trend analysis, or latest-finding lookup is implemented.

If persistence is added, use local SQLite and document retention before storing audit output.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Surface inventory | surfaces | live response | Code Facts response normalized by Quality Health | Not persisted in v1 | Persist only as part of run history. |
| Contract definitions | contracts | code/config | Quality Health registry | Versioned with code | Contracts are language/framework policy, not template policy. |
| Audit run metadata | audit | live response, optional SQLite later | AuditQuality execution | Not persisted in v1 | Store only if history is implemented. |
| Findings | audit | live response, optional SQLite later | Contract evaluation evidence | Not persisted in v1 | Stable IDs allow repeat lookup across runs. |
| Command results | commands | live response | Bounded executor | Not persisted in v1 | Store excerpts only if history is implemented. |
| Autofix previews | autofix | live response | Deterministic planner | Not persisted in v1 | Apply mode mutates target config files only. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and health |
| audit runs | audit | deferred | Future run history |
| audit findings | audit | deferred | Future run history and explain lookup |

## Migrations And Compatibility

No product tables are required for stateless v1. If run history is added, migrations must be idempotent and covered by repository tests.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Audit JSON | API/CLI JSON | audit | Planned as live response. |
| Run history export | JSON or SQLite backup | audit | Deferred. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Live audit data | Process completion | No persistence | None. |
| Future run history | Retention window or manual cleanup | Deferred decision | Define before implementing storage. |

## Privacy Notes

Quality Health reads local source/config files and may include file paths, rule evidence, command excerpts, and snippets of observed config in findings. Do not include full source files or secrets in findings. Suppress or redact command output if it can contain sensitive values.

## Cross-References

- [DOMAINS.md](DOMAINS.md)
- [INTEGRATIONS.md](INTEGRATIONS.md)
- [configuration.md](../reference/configuration.md)
- [SECURITY.md](../internal/SECURITY.md)
