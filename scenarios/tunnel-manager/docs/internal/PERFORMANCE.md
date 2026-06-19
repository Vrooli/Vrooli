# Performance — Tunnel Manager

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-06-18 |

## Planned product targets (Phase 2)

> **Status: planned, not measured.** Implementation is Phase 2; the
> numbers below are target budgets for the seven product domains
> ([`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)), not measurements.
> They size the scheduling cadence, scrape interval, recovery timings,
> and SQLite write/retention volume so the workload is bounded before
> code lands. Tunnel Manager is **SQLite only** ([`DECISIONS.md`](DECISIONS.md))
> and must stay lightweight — it is foundational infra that runs
> continuously.

| Concern | Planned target | Rationale | Domain |
|---|---|---|---|
| Internal/external probe cadence | every 30s per route (planned; tunable) | Fast enough to catch outages within the recovery window without hammering routes or the public tunnel. | `probes` |
| Prometheus scrape interval | every 15s from `127.0.0.1:20241` | Aligns with typical Prometheus scrape granularity for HA-connection/RTT trends; cheap local-loopback read. | `tunnel` |
| `/ready` health check | every 15s | Primary trigger for recovery; must detect `/ready` failure / HA=0 quickly. | `tunnel` |
| Recovery backoff | exponential, base ≈ 5s, doubling, cap ≈ 5min | Avoids restart storms on a flapping tunnel while still recovering promptly. | `recovery` |
| Circuit breaker | open after ≈ 5 consecutive failed attempts | Bounds blast radius of live actuation (see [`SECURITY.md`](SECURITY.md)); stops infinite restart loops. | `recovery` |
| Cloudflare API hot-reload latency | ingress push applied within seconds of a manifest change | Remote-mode exposure should feel immediate; bounded by Cloudflare API v4 round-trip. | `config` |
| Expected route count | ~9 CORE (`api-core/coreset`) + leased on demand | Sizes the manifest, reconciliation loop, and probe schedule; well within SQLite/single-host limits. | `routes`/`exposure` |
| Metrics time-series write volume | ~1 row / scrape-interval / metric series, retained then purged | At 15s scrape, ~5.7k samples/day/series; retention + periodic purge keeps the `metrics` table bounded. | `tunnel` |
| Probe history write volume | ~1 row / cadence / route / kind (internal+external), retained then purged | At 30s cadence over ~9+ routes × 2 kinds, retention + purge keeps the `probes` table bounded. | `probes` |
| Recovery event volume | low (only on actual recovery attempts) | Append-only audit log; small even under failure. | `recovery` |
| Retention / purge | time-windowed retention with a periodic purge job for `metrics` and `probes` | Continuous telemetry on SQLite must not grow unbounded; recovery + audit history kept longer (lower volume). | `tunnel`/`probes` |

All numbers above are **planned defaults to validate against the live
tunnel**, not commitments; capture real measurements in the table above
once Phase 2 ships and tune cadence/retention from observed volume.

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Foundational-infra constraint: continuous probing + scraping writes to
  SQLite on every cycle, so retention/purge for `metrics` and `probes`
  is a correctness concern (unbounded growth), not just a tuning one.
- Live auto-recovery timings (backoff + circuit breaker) are bounded by
  the safety contract in [`SECURITY.md`](SECURITY.md), not purely by
  performance — never tune the breaker open-threshold up for "faster
  recovery" without re-checking blast radius.
- Performance budgets for real product workflows are the planned targets
  above; replace with measurements as domains and UX flows land.

## Regression Procedure

1. Run `make test`.
2. Capture relevant API/UI command timing.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
