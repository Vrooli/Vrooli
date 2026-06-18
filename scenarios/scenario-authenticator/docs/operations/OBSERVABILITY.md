# Observability — Scenario Authenticator

This document records logs, metrics, telemetry, health checks, and
security signals for the fleet's Identity Provider (IdP).

> **Status: documentation-first orientation.** The signals and metrics
> below are the **target** observability contract derived from
> [`../../PRD.md`](../../PRD.md). The measures are **not wired yet** —
> only the scaffold `/health` and test-genie results exist today. An IdP
> is a security boundary, so its defining observability signal is the
> **audit log as a security event stream**, not just liveness.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the IdP is healthy?
- What signals tell us the IdP is under attack or misconfigured?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## Signals

The defining signals for an IdP are **security-shaped** (auth success vs.
failure, lockouts, revocations) and **trust-shaped** (JWKS reachability,
token-verify latency on the RP side).

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API + SQLite + Redis reachability | Healthy locally; unhealthy if Redis or SQLite unreachable. |
| JWKS reachability | trust | REST `/.well-known/jwks.json` | RPs can fetch the verifying key | Always serving the active `kid`; empty/404 breaks all RP verification. |
| Login success / failure rate | security | audit + auth metrics | Baseline auth health; spike in failures = credential stuffing | Failure-rate spike triggers review (not a fixed number pre-launch). |
| Lockouts triggered | security | rate-limit / lockout (OT-P0-006) | Brute-force defense firing | Rising lockouts = active attack or misconfigured client. |
| Tokens issued | activity | tokens domain | Issuance volume / realm | Trends with sign-in load. |
| Refresh reuse-detections | security | tokens domain (OT-P0-003) | A reused refresh token revoked a family | Any non-zero rate warrants investigation (theft or buggy client). |
| Token-verify latency (RP side) | performance | Relying Party local verify | Stateless JWKS verify stays cheap | Sub-millisecond; the hot path never calls back to the IdP. |
| Active sessions | activity | sessions domain (Redis) | Live session count / realm | Trends with usage; unexpected growth may indicate session leakage. |
| Audit event stream | security | audit domain (OT-P0-007) | The primary security event log | Continuous; gaps mean lost security visibility. |
| test-genie result | validation | `make test` | Correctness evidence | All required phases pass. |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Auth flow + dependency status. **Must never log passwords, tokens, refresh tokens, or private-key material.** |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Audit log | audit domain (queryable, append-only) | `scenario-authenticator audit list ...` (target) | The security event stream — not a debug log. Sign-in/out, token-family revoke, MFA changes, admin actions. See [`RUNBOOK.md`](RUNBOOK.md). |

**Log hygiene (hard rule):** no secrets in logs, ever — never log
passwords, access/refresh tokens, Argon2id inputs, OAuth client secrets,
TOTP seeds, recovery codes, or private-key bytes. Only hashes and signed
material are at rest (OT-P0-004), and the same discipline applies to log
output. An accidental token in a log is a credential leak.

## Metrics

Metrics are the **`cli/manifest.json` measure blocks** — the metrics
contract enforced by the test-genie measures phase. **None are wired
yet** (pre-implementation); the table is the target contract.

| Metric | Status | Notes |
|---|---|---|
| Login success/failure rate (per realm) | target | Auth-health and credential-stuffing signal. |
| Lockouts triggered | target | Brute-force-defense activity (OT-P0-006). |
| Tokens issued | target | Issuance volume per realm. |
| Refresh reuse-detections | target | Family-revoke count from reuse detection (OT-P0-003). |
| Active sessions | target | Live session count (Redis-backed). |
| Token-verify latency (RP side) | target | Measured at the Relying Party; the IdP hot path is stateless. |
| Requirement coverage | active | Tracked through requirements + test-genie coverage artifacts. |
| Product activation | deferred | Defined when real PRD users/workflows exist; monetization is realized in *adopting* products, not here (see `MONETIZATION.md`). |

The measure blocks are the contract: when a domain lands, its measures
must appear in `cli/manifest.json` and pass the measures phase.

## Alerts / Health

The lifecycle runs health checks for the API and UI surfaces. The API
exposes `/health` (reachability + SQLite/Redis status) and serves the
JWKS endpoint that RPs depend on. **Fail safe, not open:** a Redis outage
must surface as unhealthy and degrade revocation honestly, never silently
accept stale sessions (see [`RUNBOOK.md`](RUNBOOK.md)). Add
deployment-specific alerts (failure-rate threshold, lockout spike,
refresh-reuse rate, JWKS-unavailable) only when a deployment target and
operator expectations are defined.

## Telemetry Gaps

Telemetry is at the pre-implementation stage — measures are not wired.
Known gaps:

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Auth metrics not wired (success/failure, lockouts, issuance, reuse, sessions) | No live security signal yet. | When the P0 auth core lands and its measure blocks ship. |
| RP-side token-verify latency | Verify cost is borne by consumers; not yet instrumented. | When the first RP (device-sync-hub) migration is green (OT-P0-012). |
| Anomaly/attack detection (impossible-travel, brute-force trends) | Audit stream exists but no derived alerting. | Post-P0, when an attack-signal baseline is meaningful. |
| Per-realm metric isolation | Single default realm at P0; multi-realm cardinality not yet modeled. | When multi-realm lands (OT-P1-001). |
| Product/adoption telemetry | Cannot evaluate adoption or hosted unit economics. | Before any hosted/SaaS tier or monetization review. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures, reading the audit log
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — security posture and crypto invariants
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — why the IdP does not meter itself
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
