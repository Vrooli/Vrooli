# Performance — Scenario Authenticator

This document records performance budgets, the performance *model*, known
constraints, and regression procedures.

> **Status: implemented foundation; budgets are not blanket benchmarks.**
> The auth and verification paths are live and unit-tested. Values in the
> budget table are engineering targets until a dedicated benchmark run records
> measurements; the absence of a benchmark must not be read as absence of
> implementation.

## Purpose Of This Document

Use this document to answer:

- What performance matters for an Identity Provider?
- Where is the hot path, and why does the authenticator scale?
- What budgets or thresholds apply?
- What performance risks remain, and how are they mitigated by design?

## The performance model — the hot path is not here

The defining performance fact of this scenario is that **the hot path
lives in the consumers, not in the authenticator.** This is the whole
point of the IdP↔RP split (PRD Appendix A).

- **Token verification (the hot path) is stateless and local to the RP.**
  An adopting scenario fetches the JWKS public key once, caches it, and
  verifies every token in-process: an RS256 signature check plus `iss`/
  `aud`/`exp` validation. There is **no network callback to the
  authenticator per request.** This runs in **sub-millisecond** time on the
  RP and never touches this scenario's database. *This is the scale lever:*
  consumer request throughput is bounded by the RP's CPU, not by the
  authenticator.
- **The authenticator itself is only hit on identity-lifecycle events** —
  login, register, refresh, logout, revoke, MFA challenge, admin actions.
  These are **low-frequency** relative to the RP's request volume (a user
  logs in once and then makes thousands of verified requests against the
  RP without the authenticator seeing any of them). The authenticator is
  sized for the login/refresh rate, not the request rate of the entire
  fleet.

The consequence: the authenticator can be a single, modest SQLite-backed
instance and still serve a large fleet of high-throughput RPs, because the
expensive thing (per-request verification) was deliberately moved off it.

## Budgets

| Surface | Budget (target) | Measurement | Status |
|---|---|---|---|
| RP-side token verification (the hot path — runs in the consumer) | Sub-millisecond, in-process, zero network; one signature + claims check | RP-side benchmark of local verify | planned |
| JWKS fetch (`/.well-known/jwks.json`) | Fast + cacheable; RPs fetch ~once per process and cache (the old scenario sets `Cache-Control: public, max-age=300`) | JWKS latency + cache-header check | planned |
| Login (verify Argon2id hash + mint RS256 + persist session) | Dominated by the deliberate Argon2id cost; tens-of-ms class, not sub-ms (hashing is *meant* to be expensive) | login latency measure | planned |
| Refresh (rotate refresh token + mint RS256 + reuse check) | Single-digit-ms class; Redis lookups + one RSA sign, no Argon2id | refresh latency measure | planned |
| Token validation RPC (`Validate`, for RPs that can't verify locally) | Single-digit-ms; signature + claims, no DB | validate latency measure | planned |
| Rate-limiter check (per auth request) | Negligible Redis-authoritative counter operation; protected requests fail closed if Redis is unavailable | rate-limit overhead measure | planned |
| Session revoke / "log out everywhere" | Fast Redis op(s); bounded by session-set size for a user | revoke latency measure | planned |
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API / UI health | Responsive under lifecycle health timeout | `/health` check | active |

## Latency-budget rationale

- **Argon2id is deliberately slow.** Password hashing on login (and
  registration / password change) is the one *intentionally* expensive
  operation. Argon2id is memory-hard by design — that cost is the brute-
  force defense ([`SECURITY.md`](SECURITY.md)). The cost parameters are a
  documented security/latency trade-off, not a hot-path budget to drive to
  zero. It only runs on the low-frequency login path, never on verification.
- **RSA signing on the authenticator, RSA verification on the RP.** Signing
  is moderately more expensive than verification, but it only happens on
  login/refresh. Verification (the frequent operation) is cheap and happens
  on the RP. The asymmetry is in the fleet's favor.
- **Redis is the hot/shared-state store.** Sessions, token-family
  revocation, OAuth CSRF state, and cross-replica rate-limit counters live
  in Redis, not SQLite. Redis carries the per-event hot state so the
  single-writer SQLite store is reserved for durable identity records.

## Known Constraints

- **SQLite is single-writer — that is the authenticator's own write
  ceiling.** Concurrent login/register/realm-write throughput is bounded by
  SQLite's single-writer model. This is **mitigated by design**, not
  ignored: (1) the hot path (verification) never touches SQLite, so the
  write ceiling never gates consumer request volume; (2) hot per-event
  state lives in Redis; and (3) the `api-core/storage` seam
  ([`SEAMS.md`](SEAMS.md)) keeps a clean swap to a managed server DB for
  cloud scale (OT-P2-006) — SQLite is a default, not a lock-in.
- **Redis is a required dependency, not optional.** Session-revocation
  correctness and distributed rate-limit accuracy depend on Redis being
  reachable. Treat Redis unavailability as a degraded/Unavailable state
  (see [`ERROR-HANDLING.md`](ERROR-HANDLING.md)), not a silent fallback.
- **JWKS caching is a correctness/perf coupling.** RPs cache the public key
  for performance; key rotation (P2) must publish overlapping `kid`s so
  cached-but-stale verifiers keep working during rollover
  ([`SECURITY.md`](SECURITY.md)). The cache TTL trades verification latency
  against rotation propagation time.
- Vite production builds may process thousands of modules and take several
  minutes (inherited template constraint).

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| No dedicated benchmark captured in this document yet. | n/a | benchmark suite not run | 2026-06-18 |

## Regression Procedure

1. Run `make test` (or `vrooli scenario test scenario-authenticator`),
   which exercises the test-genie performance phase.
2. The must-have performance benchmark is the **stateless-verify
   benchmark**: measure RP-side local token verification (the hot path) and
   confirm it stays sub-millisecond and makes zero network calls. See
   [`TESTING.md`](TESTING.md) (performance phase).
3. Capture login/refresh/validate latency measures; compare against prior
   runs. Argon2id login latency is expected to be the dominant cost — watch
   for *regressions*, not for it being "slow" (slow is the point).
4. Use `git-control-tower baseline diff` to detect performance regressions
   between the working tree and the baseline run.
5. For UI interaction regressions, use `ui/perf/README.md` and the provided
   capture template.
6. Record persistent findings here (accepted constraints) or in
   [`PROBLEMS.md`](PROBLEMS.md) (unresolved debt).

## Cross-References

- [`../../PRD.md`](../../PRD.md) — Appendix A (IdP↔RP split = the scale lever)
- [`SEAMS.md`](SEAMS.md) — storage seam (managed-DB swap), Redis client seam, signing-key provider
- [`SECURITY.md`](SECURITY.md) — Argon2id cost trade-off, key rotation
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — the stateless-verify benchmark
- [`PROBLEMS.md`](PROBLEMS.md) — the shared-DB blast radius this design removes
