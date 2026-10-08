# Security — Unit Health

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
| Validation run history (SQLite) | low | validation / runhistory | Command lines, durations, statuses, file paths, coverage percentages for local repository targets. |
| Evidence cache and command output excerpts | low to unknown | validation / evidence | Captured test output (8 KiB tail per stream) can contain whatever a test prints; treat the cache directory as local development data. |

## Auth And Authorization

The generated template does not include an auth provider. Add auth only
when product requirements identify protected data or user-specific
behavior. UI and CLI must not enforce business authorization locally;
authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Notes |
|---|---|---|---|
| None by default | n/a | no | Add entries when resources or third-party APIs require secrets. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Executing target test commands | `--execution` runs the target repository's own test commands on this host, so a hostile target can run arbitrary code. | Executor bounds each command with a timeout, no-output watchdog, process-group teardown, child-leak detection, and admission caps; the plan's `HermeticPolicy` records the isolation expectations. No sandbox beyond that. | accepted, local-operator tool |
| Fix application writes files | `ScenarioValidationService.ApplyFix` writes config/projection fixes into the target. | Deterministic low-risk rules only; before-write drift check returns `failed_precondition` if a file changed. | mitigated |
| Missing auth for product data | User/customer data could be exposed if added without access control. | Add API-layer auth before storing protected data. | deferred |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No product-specific data classification | medium | Fill after PRD/domain map defines real data. |
| No auth model | conditional | Required before protected or multi-user data. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
