# Security — Tunnel Manager

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

> **Status: planned, not built.** Implementation is Phase 2; the rows
> below classify the data the seven product domains
> ([`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)) will own.
> Threat-model depth is deliberately scoped to foundational
> infrastructure — concrete enough to drive the API-layer controls, not
> a full external-pentest model. **SQLite only** ([`DECISIONS.md`](DECISIONS.md));
> no external store.

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Cloudflare API token | **high (secret)** | `config` | Remote-mode credential. Stored as a **credential reference**, never inline in `tunnel_config`, SQLite rows, or logs. Resolved at boot from the secret source (env/vault), held in memory only. |
| Tunnel id / account id | medium | `config` | Identifies the managed tunnel/account; not a secret but enables targeted action if combined with the token. |
| Exposure manifest (`routes`) | medium | `routes` | The list of what is publicly reachable, at which subdomain/port/tier. Disclosure reveals public attack surface and internal port mapping. |
| Leases (`leases`) | low-medium | `exposure` | Who requested exposure of what, and until when. Useful for attribution; no secrets. |
| Metrics / probe / recovery history | low | `tunnel`/`probes`/`recovery` | Operational telemetry (HA connections, RTT, probe results, restart outcomes). No secrets; must not capture token or request payloads. |
| Other scenarios' `service.json` ports | low | `audit` | Read **read-only** for port-compliance auditing. Tunnel Manager never writes another scenario's files. |

## Auth And Authorization

> **Status: planned, not built.**

The most security-relevant surface is the **exposure-request API**
(OT-P0-005): other scenarios and the operator can ask Tunnel Manager to
make a scenario publicly reachable. That is a privileged action — it
opens an internet-facing route — so it must be authorized.

- Authorization belongs at the **API/service layer**, never enforced
  locally in the CLI or UI (consistent with the template rule).
- The exposure-request and recovery actuation RPCs are
  operator/scenario-privileged, not anonymous. The concrete authn/authz
  mechanism follows the platform's inter-scenario auth standard
  (`scenario-authenticator` aud-scoped tokens) once Phase 2 wires it;
  until then this is the explicit gap recorded below.
- Read surfaces (`status`, `routes`, manifest queries, the app-monitor
  `IsExposed`) are lower-risk reads but still pass through the API layer.
- Port-audit reads of other scenarios' `service.json` are **read-only**
  by contract (see the `audit` seam in [`SEAMS.md`](SEAMS.md)); Tunnel
  Manager has no write path into other scenarios' files.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Cloudflare API token | secret source (env/vault), referenced — not inlined | remote mode only | Least-privilege token scoped to the managed tunnel's ingress config. Held as a **credential reference** in `tunnel_config`; resolved at boot, kept in memory, never persisted in SQLite or written to logs. Local config-mode (`~/.cloudflared/config.yml`) needs no API token. |

## Threat Model

> Depth scoped to foundational infrastructure — concrete enough to drive
> Phase 2 controls, not an exhaustive external threat model.

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Cloudflare API token leakage | Full control of the tunnel's public ingress (expose/hijack any route). | Least-privilege token; credential **reference** only (never inline/SQLite/logs); resolved in-memory at boot. | planned |
| Live auto-recovery blast radius | A false-positive restart loop could take down remote access for the whole Vrooli instance (foundational infra). | **Circuit breaker** (cap attempts, exponential backoff) + **single-owner restart contract** (Tunnel Manager is the sole cloudflared-restart owner; vrooli-autoheal downgrades to alert-only). Recovery actuation behind the exec/systemd seam so it is testable and bounded. | planned |
| Unauthorized exposure request | An attacker/buggy caller exposes an internal scenario to the internet. | Authn/authz on the exposure-request API at the service layer; exposure is tiered + lease-bounded (auto-reaped). | planned |
| Exposure as attack surface growth | Each leased route is a new internet-facing entry point. | Tiering (CORE vs LEASED), TTL auto-reaping, and (P2) hostname-budget/LRU eviction bound the live surface. | planned |
| Secret/PII capture in telemetry | Token or request payloads leak into metrics/probe/recovery rows or logs. | Telemetry stores only operational signals (HA, RTT, probe status, restart outcome); explicit no-secrets-in-SQLite/logs rule. | planned |
| Tampering via other scenarios' files | Audit reads could be abused to mutate another scenario. | `audit` reader is **read-only** by contract; no write path. | planned |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Exposure-request authn/authz not yet wired | high | Required before the exposure-request API is reachable by non-operator callers (OT-P0-005). |
| Recovery actuation authz not yet wired | medium | Required before recovery RPCs are exposed beyond the operator. |
| Credential-reference resolution mechanism unimplemented | high | Required before remote mode (OT-P0-002) ships. |
| No product code yet | informational | Resolve as Phase 2 implements each domain; this doc is the spec to build to. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
