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
| Scenario validation | 18/18 phases passed, 1m42s | `vrooli scenario test tunnel-manager` | 2026-06-19 |
| Lighthouse home page | performance 99%, accessibility 100%, best-practices 100%, SEO 91% | test-genie performance phase | 2026-06-19 |

## Runtime Budgets And Tunables

Tunnel Manager is **SQLite only** ([`DECISIONS.md`](DECISIONS.md)) and
must stay lightweight because it is foundational infra that runs
continuously. The budgets below are the implemented defaults or
operator-tunable targets; live Cloudflare latency and retention windows
still need operator-environment measurements.

| Concern | Budget / default | Rationale | Domain |
|---|---|---|---|
| Internal/external probe cadence | `TUNNEL_MANAGER_PROBE_INTERVAL`, default `1m` | Frequent enough for operator visibility without hammering local scenarios or the public tunnel. | `probes` |
| Exposure reconcile cadence | `TUNNEL_MANAGER_EXPOSURE_RECONCILE_INTERVAL`, default `5m` | Keeps CORE routes and expired leases converged without turning ingress sync into a hot loop. | `exposure` |
| Recovery evaluation cadence | `TUNNEL_MANAGER_RECOVERY_EVALUATE_INTERVAL`, default `1m`, opt-in scheduler | Recovery can restart cloudflared, so background actuation is explicit and bounded. | `recovery` |
| Recovery backoff | exponential, service-owned | Avoids restart storms on a flapping tunnel while still recovering promptly. | `recovery` |
| Circuit breaker | service-owned threshold/cap | Bounds blast radius of live actuation (see [`SECURITY.md`](SECURITY.md)); stops infinite restart loops. | `recovery` |
| Cloudflare API hot-reload latency | ingress push applied within seconds of a manifest change | Remote-mode exposure should feel immediate; bounded by Cloudflare API v4 round-trip. | `config` |
| Expected route count | ~9 CORE (`api-core/coreset`) + leased on demand | Sizes the manifest, reconciliation loop, and probe schedule; well within SQLite/single-host limits. | `routes`/`exposure` |
| Metrics time-series write volume | bounded by scrape actions and 14-day retention prune | Repository writes delete rows older than `tunnel.MetricsRetentionWindow`. | `tunnel` |
| Probe history write volume | ~1 row / cadence / route / kind (internal+external), 14-day retention prune | At `1m` over ~9+ routes × 2 kinds, the rolling window stays bounded in SQLite. | `probes` |
| Recovery event volume | low (only on actual recovery attempts) | Append-only audit log; small even under failure. | `recovery` |
| Retention / purge | 14 days for metrics/probes; 90 days for recovery events | Pruning happens on repository writes, so no extra long-lived purge scheduler is required. | `tunnel`/`probes`/`recovery` |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Foundational-infra constraint: continuous probing + scraping writes to
  SQLite on every cycle. Repository-level retention pruning is part of the
  correctness contract, not just a tuning detail.
- Live auto-recovery timings (backoff + circuit breaker) are bounded by
  the safety contract in [`SECURITY.md`](SECURITY.md), not purely by
  performance — never tune the breaker open-threshold up for "faster
  recovery" without re-checking blast radius.
- Retention pruning is write-triggered. If a future domain adds a higher
  volume table, document its window here and keep pruning close to the
  owning repository.

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
