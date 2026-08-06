# Decisions — Secrets Manager

## Purpose Of This Document

This log records durable design choices that affect safe maintenance.

## Decision Log

| Decision | Rationale | Evidence |
|---|---|---|
| Native credential authority owns ordinary resource credentials | One metadata-safe provisioning and runtime-resolution contract prevents fallback drift | `internal/secrets`, `api/credential_*.go` (legacy filenames pending source move) |
| Desktop metadata uses private SQLite | Bundle data remains private to its app root | `api/desktop_storage.go` |
| Vault is capability-specific or an explicit mirror | Ordinary desktop credentials must resolve locally when Vault is unavailable | desktop resource plan tests |
| Artifact admission fails closed | A bundle must not consume unsigned release checksums | resource artifact pipeline |

## Superseded Decisions

Legacy direct filesystem, YAML, shell-export, and direct-Vault readers are not resource contracts. They are migration-only inputs and are removed after verified migration.

## Cross-References

- [Architecture](../concepts/ARCHITECTURE.md)
- [Deployment](../operations/DEPLOYMENT.md)
