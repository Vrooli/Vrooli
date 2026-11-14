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
| Provider credentials | `resource-ollama` / `resource-openrouter` | no in AI Gateway | Gateway streaming reads only the resource-declared Credential Authority identity at dispatch; it never stores, logs, or returns the value. Hosted credential custody remains resource-owned. |
| Prompt text | caller request body | per execution | Prompt text may be sent to a selected resource command through stdin, but it must not be written to route evidence or command argv. |
| Subscription refresh token | native credential authority | optional | Declared as `vrooli/lpbs-account:refresh-token`; never loaded into browser storage, argv, gateway payloads, or logs. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Direct provider credential handling | Provider secrets could leak into logs, evidence, or UI responses. | Resources own provider credentials and concrete catalogs. Gateway execution uses their declared authority identity and endpoint; no caller supplies credentials or URLs. | active |
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

## E6b provider decision boundary

The private provider-route handlers (`models`, `plan`, `stream`, `execute`, and `receipt`) use the private network boundary without per-request authentication. Production composition reads resource-owned provider terms and hosted routes, verifying exact canonical digests through Notification Hub's scoped service identity. Proposed, changed, expired or unverifiable records deny execution. Unwired supplier proposals remain in `resources/provider-rule-proposals.json`; they have no executable routes, credentials, prices or approvals.

The streaming executor claims each operation in SQL before dispatch, never retries an unknown dispatch, and persists measured usage independently of the public caller. Supplier connection details and credential identities come from the owning resource manifest; credential values are read from Credential Authority for dispatch and never persisted or returned. Test-only approval receipts are labelled fixtures and are never loaded by production composition.
The standard API server defaults to binding `127.0.0.1`. `API_BIND_ADDRESS`
can override that bind address. If an operator configures a remote interface,
the private routes are reachable through it unless an authenticated edge blocks
them; AI Gateway does not configure such an edge. Loopback binding is a network
boundary, not per-request authentication.

The frozen F1 contract specifies private LPBS-to-Gateway routes over plain HTTP
and no protobuf. REQ-A1 says requests go through the authenticated landing page
and specifies no Gateway authentication. These sources do not define whether
the private LPBS-to-Gateway path is protected by loopback-only reachability or
another boundary. Keep this contract unresolved; do not describe loopback as a
secret check or add a Gateway credential without owner resolution.

Notification Hub's provider-decision publish and exact lookup routes require a
service token with subject `ai-gateway`, audience
`scenario-authenticator:notification-hub-provider-decision`, and the sole
`notification-hub:provider-decision` capability. The local-test profile is
configured through Scenario Authenticator. Gateway reads its credential from
the existing Credential Authority address and exchanges it through Scenario
Authenticator; both credential and exchanged token are held in memory and are
not logged or persisted by Gateway. Gateway validates the exchange metadata
before attaching the bearer to either fixed Hub route. Missing custody,
exchange failure, expiry, or identity mismatch fails closed. The shared
`vrooli-events` key is not a Gateway credential.

The Hub client is composed in Gateway, but E6b plan/execute does not yet invoke
its publish, lookup, or receipt-verification methods. Local unit tests prove
credential/exchange failure handling and fixed-route bearer use without live
credentials or Hub records. Live issuance still requires a verified human
issuer/admin session and is not part of this proof.
