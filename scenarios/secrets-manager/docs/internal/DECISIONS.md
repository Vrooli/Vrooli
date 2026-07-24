# Decisions — Secrets Manager

## Purpose Of This Document

This log records durable design choices that affect safe maintenance.

## Decision Log

| Decision | Rationale | Evidence |
|---|---|---|
| Vault values stay behind `resource-vault` | Prevent plaintext fallbacks and direct endpoint bypass | `resources/vault`, `api/vault_*.go` |
| Desktop metadata uses private SQLite | Bundle data remains private to its app root | `api/desktop_storage.go` |
| Desktop Vault is private by default | Shared reuse requires explicit authority and consent | desktop resource plan tests |
| Artifact admission fails closed | A bundle must not consume unsigned release checksums | resource artifact pipeline |

## Superseded Decisions

The legacy direct filesystem fallback is not the resource contract and is being retired in favor of broker/resource-backed behavior.

## Cross-References

- [Architecture](../concepts/ARCHITECTURE.md)
- [Deployment](../operations/DEPLOYMENT.md)
