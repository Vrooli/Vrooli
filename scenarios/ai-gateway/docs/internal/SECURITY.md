# Security — AI Gateway

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

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Gateway requests | potentially sensitive | `gateway` / callers | Requests can contain private prompts. API validation and routing keep prompt text out of command argv; execution sends prompt text to resource commands over stdin. |
| Route evidence | internal metadata | `routing` | SQLite stores request intent, role/profile, selected provider, policy reasons, status, timing, and redaction flags. Raw prompts and raw provider responses are not persisted. |
| Provider inventory | operational metadata | resources | AI Gateway reads resource-owned role/policy output and normalized smoke status. Concrete model catalogs and credentials stay with the resources. |
| Conformance findings | internal diagnostics | `conformance` | Findings contain file paths, rule ids, redacted messages, and remediation text for target scenarios. Scanner output must not echo secrets or concrete provider identifiers beyond bounded diagnostic categories. |
| Consumer access tokens | secret, short-lived | `providers` / LPBS | A bearer access token may be forwarded to LPBS metered inference or resolved in memory through the shared credential authority. Refresh tokens never enter gateway requests, logs, or responses. |

## Auth And Authorization

AI Gateway remains an operator-local API by default. Local and BYOK candidates
do not require subscription auth. The LPBS candidate requires either a caller's
verified consumer bearer or a short-lived bearer resolved from the declared
`vrooli/lpbs-account:refresh-token` credential through the native credential
authority. LPBS remains the authority for identity, entitlement, and metering.
Before remote or multi-user exposure, an authenticated edge must be placed in
front of the gateway; the local credential authority is not an edge auth model.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Provider credentials | `resource-ollama` / `resource-openrouter` | no in AI Gateway | AI Gateway must not read, store, log, or forward provider API keys. Hosted credentials remain resource-owned. |
| Prompt text | caller request body | per execution | Prompt text may be sent to a selected resource command through stdin, but it must not be written to route evidence or command argv. |
| Subscription refresh token | native credential authority | optional | Declared as `vrooli/lpbs-account:refresh-token`; never loaded into browser storage, argv, gateway payloads, or logs. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Direct provider credential handling | Provider secrets could leak into logs, evidence, or UI responses. | AI Gateway only calls resource command surfaces; resources own provider credentials and concrete catalogs. | active |
| Prompt leakage in process lists or evidence | Sensitive prompts could appear in argv, route history, or diagnostics. | Routing execution passes prompt text over stdin and persists only redacted metadata. | active |
| Path traversal during conformance scans | Scanner could read outside a target scenario. | Scan files come from `WalkDir` under the target root and are checked with `filepath.Rel` before open. | active |
| Missing auth for multi-user deployment | Local operator data could be exposed if the scenario is deployed remotely without auth. | Keep deployment local/operator-scoped until API-layer auth is added. | deferred |
| Metered request without consumer identity | Paid inference could be misattributed or bypass subscription checks. | LPBS candidate requires a forwarded or authority-resolved bearer; LPBS derives identity from verified claims and rejects unauthenticated requests. | active |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No multi-user auth model | conditional | Required before exposing AI Gateway beyond local/operator-controlled deployments. |
| Security Health external findings remain | medium | Resolve in a platform security pass covering Go toolchain advisories, pnpm advisories, and shared test utility SAST findings. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
