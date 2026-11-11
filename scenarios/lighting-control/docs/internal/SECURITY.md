# Security — Lighting Control

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
| Enrolled controller/zone identity and desired lighting state | low | control | Host-local configuration; no account or personal telemetry is stored. Enrollment can reveal installed hardware. |
| Preset configuration and active request expiry | low | policy | Host-local configuration. Source telemetry remains owned by system-monitor and Agent Manager. |

## Auth And Authorization

Remote lighting operations must pass through Vrooli's API auth boundary.
OpenRGB's raw SDK is unauthenticated and must remain loopback-only; it is
never exposed through a proxy, LAN bind, or browser. UI and CLI call the
lighting-control API, never the SDK directly.

Every public Control and Policy mutation requires a verified, unexpired
identity with `lighting-control:write`. Setup uses the same boundary helper;
read capabilities retain the shared resolver's wildcard rules. The passive
identity middleware supplies verified attribution and does not itself authorize
mutations. Missing, invalid, expired, unverified and read-only credentials are
refused before storage or backend calls. Existing scoped human, agent and
service identities remain supported without an added operation grant or
human-only restriction. Background evaluation and direct domain calls remain
inside their existing internal owner boundaries.

Private HTTP regressions exercise every non-read Control/Policy RPC through
the generated clients and shared passive middleware. They compare complete
application-table snapshots and backend call counts on denial, and cover
scoped and wildcard owner requests. This proves the source boundary; operated
identity propagation and native hardware qualification remain separate checks.


## Read-Only Policy Preview

`GetPolicy` returns normalized configuration and an evidence-backed preview of
the single policy engine. Missing policy rows and legacy optional fields receive
presentation defaults without a storage write. Cache misses may recompute the
preview, but do not persist hysteresis, debounce, expiry or pending-selection
state, finish a control selection, or call the lighting backend. The returned
preview does not claim that hardware applied it.

Only authorized mutation handlers and the internal background cadence persist
engine transitions and apply policy. Foreground reads do not initialize the
authoritative policy or advance the headless state machine between ticks.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| None by default | n/a | no | Local SDK access is loopback only; no SDK credential exists. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Remote unauthenticated SDK exposure | Remote callers could issue raw hardware commands. | Keep OpenRGB loopback-bound and route remote operations through Vrooli-authenticated API transport. | required boundary |
| Desired state mistaken for physical output | Operators could believe an unobserved command changed visible lighting. | Separate desired state, command-write outcome, and optional observed-output evidence in APIs and UI. | required contract |
| Accidental hardware probing | Startup/device refresh could access host controllers without operator enablement. | No implicit resource setup/start; bounded scans occur only after explicit enablement and lifecycle ownership remains with OpenRGB resource. | required boundary |
| Stale policy inputs | Old telemetry could drive misleading color/effect. | Enforce freshness and select configured static fallback. | required policy |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| E2 owner has not qualified host-install adoption or live hardware startup. | high | E2 lifecycle and explicit backend qualification complete. |
| Current resource only qualifies Linux amd64. | high | Additional platforms pass resource and lighting hardware qualification. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt

## Identity deployment contract

The native service declares the shared `scenario_authenticator` profile and
resource audience `scenario-authenticator:lighting-control`. The lifecycle
binds these non-secret settings; the shared verifier discovers the configured
installation provider and verifies its signing keys locally. Light Desk does
not start an identity provider, register accounts, or grant capabilities during
startup. An unavailable provider leaves reads available and mutations refused.
The shared catalog declares `lighting-control:read` and `lighting-control:write`
as agent-eligible capabilities; account administration and delegation remain
with their existing identity owners.

The API always installs passive shared identity middleware, including when no
human provider was configured. Agent Manager's independently verified baseline
identity can therefore supply its existing scopes and signed expiry. A missing
capability is a refusal, not permission to exchange the agent for a local owner
or issue another grant. Operation-specific grants remain separate from coarse
lighting capabilities.

Private HTTP issuer regressions use RS256 tokens and the real shared JWKS,
issuer and resource-audience verifier against temporary SQLite storage. They
prove metadata-only enrollment persistence for a scoped issuer token and no
storage/backend effects for wrong audience or read-only tokens. This is isolated
integration evidence; it does not prove a particular installation account has
been provisioned or signed in.

## Same-origin browser session (shipped)

The Account section proxies only typed Login (fixed resource lighting-control),
Refresh and Logout to the existing installation provider via passive discovery.
It never creates users or assigns capabilities. Login preserves MFA challenges
without issuing a browser session early. Responses redact bearer and refresh
values; canonical issuer cookies remain host-only, HttpOnly and SameSite=Lax.
GET status reads only the verified unexpired context principal. POST operations
require an explicit session header and reject hostile Origin/Fetch metadata;
forwarded browser Host is accepted only from loopback UI proxies. No tokens are
persisted in browser storage. TLS deployments preserve Secure issuer flags.
See the configuration runbook for expiry, outage and canonical Logout semantics.
