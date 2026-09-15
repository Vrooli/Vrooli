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

The rows below classify the data the product domains
([`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)) own or consume.
Threat-model depth is deliberately scoped to foundational
infrastructure — concrete enough to drive the API-layer controls, not
a full external-pentest model. **SQLite only** ([`DECISIONS.md`](DECISIONS.md));
no external store.

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Cloudflare API token | **high (secret)** | `config` | Remote-mode credential. Resolved through the Vrooli credential authority and held in memory only for Cloudflare API calls. Represented to clients only as non-secret source/reference metadata; never inline in `tunnel_config`, SQLite rows, UI payloads, CLI output, or logs. |
| Tunnel id / account id | medium | `config` | Identifies the managed tunnel/account; not a secret but enables targeted action if combined with the token. |
| Exposure manifest (`routes`) | medium | `routes` | The list of what is publicly reachable, at which subdomain/port/tier. Disclosure reveals public attack surface and internal port mapping. |
| Leases (`leases`) | low-medium | `exposure` | Who requested exposure of what, and until when. Useful for attribution; no secrets. |
| Metrics / probe / recovery history | low | `tunnel`/`probes`/`recovery` | Operational telemetry (HA connections, RTT, probe results, restart outcomes). No secrets; must not capture token or request payloads. |
| Other scenarios' `service.json` ports | low | `audit` | Read **read-only** for port-compliance auditing. Tunnel Manager never writes another scenario's files. |

## Auth And Authorization

> **Status: local/operator-open by default, with an implemented
> service-layer static-token gate for privileged mutation RPCs. Enable it
> before exposing the API beyond lifecycle-managed local operator use.**

The most security-relevant surface is the **exposure-request API**
(OT-P0-005): other scenarios and the operator can ask Tunnel Manager to
make a scenario publicly reachable. That is a privileged action — it
opens an internet-facing route — so it must be authorized.

- Authorization belongs at the **API/service layer**, never enforced
  locally in the CLI or UI (consistent with the template rule).
- The exposure-request and recovery actuation RPCs are
  operator/scenario-privileged, not anonymous. Today they can be
  fail-closed with `TUNNEL_MANAGER_AUTHZ_ENFORCED=1`; privileged
  mutations then require `Authorization: Bearer <token>` or
  `X-Vrooli-Operator-Token: <token>` matching
  `TUNNEL_MANAGER_OPERATOR_TOKEN`, falling back to `API_TOKEN`.
- The static-token gate covers config sync/mode changes, route
  create/update/delete, exposure expose/extend/revoke/reconcile, and
  manual recovery. The platform's future inter-scenario auth standard
  (`scenario-authenticator` aud-scoped tokens) remains deferred for
  non-operator cross-scenario callers.
- Read surfaces (`status`, `routes`, manifest queries, the app-monitor
  `IsExposed`) are lower-risk reads but still pass through the API layer.
- Port-audit reads of other scenarios' `service.json` are **read-only**
  by contract (see the `audit` seam in [`SEAMS.md`](SEAMS.md)); Tunnel
  Manager has no write path into other scenarios' files.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Cloudflare API token | Vrooli credential authority | remote mode only | Least-privilege token scoped to the managed tunnel's ingress config. Resolved inside the API process, kept in memory only for Cloudflare calls, and exposed only as non-secret source/reference metadata. Local config-mode (`~/.cloudflared/config.yml`) needs no API token. |

## Threat Model

> Depth scoped to foundational infrastructure — concrete enough to drive
> Phase 2 controls, not an exhaustive external threat model.

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Cloudflare API token leakage | Full control of the tunnel's public ingress (expose/hijack any route). | Least-privilege token; credential **reference** only (never inline/SQLite/logs/UI/CLI); resolved in-memory per remote Cloudflare operation through the Vrooli credential authority; write-only API/CLI/UI setup. | implemented for the Vrooli credential authority; Vault/provider references deferred |
| Live auto-recovery blast radius | A false-positive restart loop could take down remote access for the whole Vrooli instance (foundational infra). | **Circuit breaker** (cap attempts, exponential backoff) + **single-owner restart contract** (Tunnel Manager is the sole cloudflared-restart owner; vrooli-autoheal downgrades to alert-only). Recovery actuation is behind the control-plane resource lifecycle seam and the background scheduler is opt-in via `TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED`; manual recovery remains available. | implemented, opt-in background actuation |
| Unauthorized exposure request | An attacker/buggy caller exposes an internal scenario to the internet. | Service-layer static-token authz for privileged mutations; exposure is tiered + lease-bounded (auto-reaped). | implemented for operator-token boundary; aud-scoped inter-scenario tokens deferred |
| Exposure as attack surface growth | Each leased route is a new internet-facing entry point. | Tiering (CORE vs LEASED), boot/periodic TTL auto-reaping, and (P2) hostname-budget/LRU eviction bound the live surface. | implemented except P2 budget/LRU |
| Secret/PII capture in telemetry | Token or request payloads leak into metrics/probe/recovery rows or logs. | Telemetry stores only operational signals (HA, RTT, probe status, restart outcome); explicit no-secrets-in-SQLite/logs/UI/CLI rule. | partially implemented; continue validating as probe/recovery surfaces evolve |
| Tampering via other scenarios' files | Audit reads could be abused to mutate another scenario. | `audit` reader is **read-only** by contract; no write path. | implemented |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Scenario-authenticator aud-scoped tokens not integrated | medium | Required before non-operator or cross-scenario callers get direct privileged mutation access without static operator-token coordination. |
| Vault/provider credential-reference resolution not implemented | medium | Required if remote credentials need to come from a Vrooli credential provider instead of process env. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
