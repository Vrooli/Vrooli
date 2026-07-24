---
date: 2026-07-24
scenario: swarm-manager
interactions:
  - api-route-sweep
  - next-action-feed
  - plan-board
traces:
  cpu_profile: /tmp/claude-1000/-home-matthalloran8-Vrooli/c0e584cf-87ad-4233-8584-705db1f06fd3/scratchpad/cpu-feed.pprof
  sweep_script: scenarios/swarm-manager/scripts/perf-baseline.sh
status: fixed
related_skill_run: performance
---

# Perf audit: swarm-manager API read paths

Server-side latency audit run during
`swarm-manager-performance-hardening-known-hot-path-fixes`, after phases 2 to 4
removed the known next-action hot paths. This audit looks for the next tier:
every registered read route was timed, and a CPU profile was captured under a
feed and plan-board burst to find where the remaining time goes.

## Method

Measured against the running instance at production data scale: 608 backlog
specs (431 non-archived), 172 goals, 32 agent sessions.

- Route timing: 37 static GET routes, one warm-up request then the median of
  five timed requests each. Routes needing path parameters were excluded.
- CPU profile: `SWARM_MANAGER_ENABLE_PPROF=1`, 12 seconds sampled while 40
  interleaved `/next-actions/feed` and `/plan` requests ran.
- pprof is off unless `SWARM_MANAGER_ENABLE_PPROF=1`, because the endpoints
  expose process memory and command line.

## Per-component aggregation

Components here are routes and the hot paths behind them, ranked by median
latency before the fix. "Boundary" states whether the cost is owned by
swarm-manager or by a scenario this plan may not change.

| # | Route / hot path | Before | After | Root cause | Boundary | Outcome |
|---|---|---|---|---|---|---|
| 1 | `GET /api/v1/plan-import/plans` | 651ms | 651ms | Single delegating RPC to plan-manager returning a 227KB plan list; the time is plan-manager's own list cost, not swarm-manager compute | plan-manager server | **Deferred** |
| 2 | `GET /api/v1/prompts/skills` | 588ms | 93ms | `ListSkills` issued one `GetSkill` RPC per catalog entry, serially — the same sequential-remote-call defect class this plan exists to remove (`internal/prompts/handler.go:159`) | swarm-manager | **Fixed** |
| 3 | `GET /api/v1/integrations` | 376ms | 77ms | `Provider.Statuses` probed each integration serially (`internal/integrationstatus/provider.go:85`) | swarm-manager | **Fixed** |
| 4 | `/plan` board ETA band | 35% of `/plan` CPU | unchanged | `planview.Service.buildETA` runs a Monte Carlo simulation whose every trial walks the longest path over the whole 392-item closure (`internal/eta/montecarlo.go:135`, `eta.longestPath` = 23% of profile) | swarm-manager | **Deferred** |
| 5 | Repeated `backlog.LoadAll` per `/plan` request | 20% of `/plan` CPU | unchanged | `FileStore.LoadAll` walks and JSON-decodes every spec; a single board request performs it more than once across the projection, the board, and the attention sources | swarm-manager | **Deferred** |

### Deferral reasons

**Finding 1 — `plan-import/plans`.** The handler is a pass-through:
`h.svc.ListPlans(ctx)` and encode. All 651ms is plan-manager answering a list
request with a 227KB payload. The plan's Boundaries section names plan-manager
server changes as a non-goal and permits client-side memoization only. Caching
a plan list client-side is the wrong trade for this route: it backs the plan
import picker, where a stale list means an operator cannot see a plan they just
created. Left as the one route above 500ms, deliberately.

**Finding 4 — board ETA Monte Carlo.** Real and the single largest remaining
share of `/plan` CPU, but `/plan` now runs at 76ms against a 300ms budget, so
removing it buys headroom rather than fixing a problem. It is also the finding
most likely to change estimate values if done carelessly, and this plan holds a
byte-identical-semantics bar. The natural fix is the one phase 4 already built:
hold the board ETA per data generation behind the same invalidation seam.

**Finding 5 — repeated `LoadAll`.** Same reasoning as finding 4: the fix is a
request-scoped item cache, which duplicates the generation-cache concept phase 4
introduced. Worth doing when the board projection itself moves behind the shared
cache; not worth a second, differently-shaped cache now.

## Sweep results after fixes

Every route except finding 1 is at or under 175ms. No route other than
`plan-import/plans` exceeds the 500ms acceptance threshold.

| Route | Median | Bytes |
|---|---|---|
| `/api/v1/plan-import/plans` | 651ms | 227KB |
| `/api/v1/scenarios` | 173ms | 39KB |
| `/api/v1/agent-manager/status` | 148ms | 116B |
| `/api/v1/operations/brief` | 140ms | 4KB |
| `/api/v1/scenarios/review-queue` | 101ms | 615B |
| `/api/v1/search/ai/status` | 100ms | 123B |
| `/api/v1/prompts/skills` | 93ms | 98KB |
| `/api/v1/overview` | 78ms | 1.5MB |
| `/api/v1/integrations` | 77ms | 2KB |
| `/api/v1/plan` | 76ms | 186KB |
| `/api/v1/gct/status` | 72ms | 18B |
| `/api/v1/goals` | 64ms | 366KB |
| `/api/v1/stats` | 60ms | 8KB |
| `/api/v1/records` | 40ms | 2.8MB |
| `/api/v1/backlog` | 16ms | 786KB |
| `/api/v1/backlog/summary` | 16ms | 25KB |
| `/api/v1/agent-sessions` | 4ms | 101KB |
| `/api/v1/graph` | 3ms | 391KB |
| `/api/v1/next-actions/feed` | 2ms | 218KB |

Remaining sub-500ms routes above 70ms are remote status probes
(`agent-manager/status`, `gct/status`, `search/ai/status`) or filesystem scans
over the scenario tree (`scenarios`, `scenarios/review-queue`). Neither is a
per-item scan; both are single bounded reads.

## Payload observations

Latency is no longer the binding constraint on several routes, but payload size
is worth recording: `/api/v1/records` returns 2.8MB and `/api/v1/overview`
1.5MB on every call. Both are fast to produce and slow to transfer and parse in
a browser. Trimming them is a response-shape change rather than a latency fix,
and the plan's Boundaries section excludes payload trimming from this work
("saves bytes, not the store scans"). Recorded here so the next audit starts
from a measurement rather than a guess.
