# Observability — Token Economy

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |

## Metrics

**A note on what this scenario will not collect.** The privacy posture is half
the product's differentiation: nothing leaves the machine, and there is no
analytics vendor. Every metric below is *local, operator-visible, and derived
from the journal the operator already owns*. There is no telemetry pipeline to
build, and adding one would contradict the claim the product is sold on.

| Metric | Status | Details |
|---|---|---|
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Projection vs journal agreement | planned | The integrity signal that matters most: a scheduled comparison of cached balances against a full replay. Disagreement is a correctness incident, not a performance one. |
| Pending approval age | planned | How long a holder has been waiting. Surfaced in the minter console (`TKE-P0-013`); a queue with rotting requests means the gated posture is failing socially. |
| Earning submission rate per adapter | planned | Detects a looping or flooding adapter early. Local only. |
| Replay/no-op rate on earning | planned | A rising duplicate rate means an adapter is retrying more than it should. |
| Unverified-provenance event share | planned | How many events land without a verified actor (`TKE-P0-011`). A rising share means identity verification is degrading somewhere. |
| Redemption denial rate with reasons | planned | Product signal rather than ops: high denials suggest the catalog or the rules are miscalibrated. |
| Economy activity (earn/redeem volume) | planned | Feeds the dashboard's economy-health panel; also the retention signal `MONETIZATION.md` depends on. |
| Product usage telemetry (remote) | **not-applicable, deliberately** | Would require sending a minor's behavioral record off-machine. Not collected at any tier. |
| Performance budgets | planned | Defined in `../internal/PERFORMANCE.md`; measured locally. |

## Alerts / Health

Lifecycle health checks cover API and UI. Beyond those, the scenario's health is
mostly a question about its one hard dependency and its one integrity invariant.

| Condition | Severity | Where it surfaces | Why it matters |
|---|---|---|---|
| `scenario-authenticator` unreachable | **critical** | `/health` dependency status | Authenticated surfaces fail closed, so the product is effectively down. This is the correct behavior, but the operator must know why. |
| Projection disagrees with journal replay | **critical** | Scheduled integrity check | The append-only store has no repair verb by design. Disagreement means a bug already committed. |
| Pending approvals older than a declared threshold | warning | Minter console | A child is waiting. Not a system fault, but the thing the operator most wants surfaced. |
| SQLite unreachable | critical | `/health` | Mutations refuse rather than partially apply. |
| `notification-hub` unreachable | **info, never a warning** | `/health` dependency status | Absence is the baseline, not degraded mode. The approval queue works unchanged; only out-of-band reach is lost. |
| `agent-manager` unreachable | info | `/health` dependency status | Events still write, recorded as unverified. Never blocks. |

No external alerting integration is planned. In a single-household deployment
the operator *is* the alerting system, and the console is where conditions
belong.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Nothing is implemented. | Every signal above is planned; the scenario currently emits only template health. | Gate 6. |
| No scheduled projection-integrity check. | The one metric that would catch a correctness bug before a human notices a wrong balance. | Must exist before the first real household runs the loop. |
| Retention/adoption measurement is manual. | `MONETIZATION.md` needs a four-week retention signal, and there is deliberately no remote telemetry to supply it. | Accept manual observation for the household validation; do not solve this by adding remote telemetry. |
| Cost telemetry absent. | Cannot evaluate hosted unit economics. | Only if a managed deployment is ever contemplated — which would first require resolving the erasure gap in `SECURITY.md`. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
