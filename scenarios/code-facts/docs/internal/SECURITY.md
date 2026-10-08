# Security — Code Facts

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| Source paths, symbols, and route metadata | internal | Code Facts maintainers | Derived from the local repository; do not expose outside the trusted project boundary. |
| Cache and index payloads | internal | Code Facts maintainers | May contain source-derived facts and paths; governed by local cache and generation retention. |

## Auth And Authorization

Read access remains inside the trusted local project boundary. Every index
mutation is authorized at the API boundary with a constant-time comparison of
the `X-Code-Facts-Control-Token` header against the process-local configured
token. Missing configuration, a missing header, and a mismatch all fail closed.
The CLI only transports the token; it is not an authorization authority.

## Secrets

| Secret | Source | Required? | Notes |
|---|---|---|---|
| Index control token | `CODE_FACTS_INDEX_CONTROL_TOKEN` environment variable | required for index mutations | The CLI reads the same variable and sends it as a request header. It is intentionally not accepted as a command-line flag because process arguments are observable. Never log the token. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Untrusted target traversal | A caller could request files outside the repository. | Resolver normalizes and validates target roots; analyzers receive bounded roots only. | active |
| Source-derived metadata disclosure | Internal paths or symbols could leave the trusted project. | Search Hub is a local capability; registration and status are local-only. | active |
| Control-token disclosure | A holder could reconcile, rebuild, promote, roll back, or clean index generations. | API-layer fail-closed token gate, constant-time comparison, explicit confirmation for destructive controls, and no token logging. | active |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No product-specific data classification | medium | Fill after PRD/domain map defines real data. |
| No end-user identity model | conditional | Required before protected or multi-user read access. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
