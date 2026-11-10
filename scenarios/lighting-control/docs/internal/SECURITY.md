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
