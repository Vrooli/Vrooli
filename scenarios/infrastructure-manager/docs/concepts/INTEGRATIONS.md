# Integrations — Infrastructure Manager

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

**Every scenario dependency here is a read.** This scenario invokes no
mutating verb on any dependency — no restart, no shelve, no reconcile-and-fix,
no policy change. That is not a current limitation; it is the
"supervise, don't operate" boundary from
`docs/infra-health/operating/OPERATING_MODEL.md` rule 3, and an extension
that breaks it is an architectural defect rather than a feature.

### Transport: typed Connect through `api-core/discovery`

Every source is read as a **typed Connect-RPC call**, resolved through
`api-core/discovery` and bounded by a 10s per-source deadline, concurrently.
This mirrors `meta-optimization-manager`'s numerator join — including the
lesson its `numeratorclient.go` records verbatim: the previous CLI substrate
spawned each owner's binary with a 30s timeout, so one slow or hung owner
stalled the whole board. That failure matters more here than there, because
this scenario's sources are the layers that are *already unhealthy* during the
incidents the board exists to surface. A hanging read is the expected case.

**One CLI read survives, and it is fenced.** `vrooli capacity` is control-plane
`internal/capacity`, so a separate Go module cannot import it and `discovery`
cannot resolve it. It is read as a bounded subprocess with `--json`. This is a
construction constraint, not a precedent: a second CLI read is a design smell
and should be challenged.

**Two things each owner must expose**, beyond the data itself:

| Obligation | Why |
|---|---|
| `space --projection <p> --json` | Serves the coverage denominator — which cells exist in that owner's reliability space. Identical to the contract `search-hub`, `test-genie`, `prompt-manager` and `program-runtime` already implement for `meta-optimization-manager`. |
| A typed read RPC for the numerator | Serves the live join. Without it the join falls back to nothing, and the cells stay `IN-REACH` rather than `NOW`. |

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, `condition`, `focus` | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| Setpoint file | checked-in data | yes | `coverage` | `setpoint/reliability-setpoint.json` in this scenario | Unparseable setpoint is a hard, loud failure — the board has nothing to measure against and must say so rather than report an empty map as zero targets. |
| `vrooli-autoheal` | scenario (read-only) | no | `coverage`, `condition` | Typed Connect over checks, actions, incidents and healing; `space --projection supervision\|availability\|recovery --json`. **No proto surface exists today — it is built as part of this work.** | Per-source `UNAVAILABLE` with stated reason. Three projections lose their readings and two of four trust rules go `UNTRUSTED`; the rest keep ranking. |
| `vrooli capacity` | control plane (read-only) | no | `coverage`, `condition` | Bounded CLI subprocess: `capacity reconcile --json`, `capacity recommend --json`. The one legitimate CLI read. | Per-source `UNAVAILABLE`. The capacity projection loses its readings. |
| `storage-manager` | scenario (read-only) | no | `coverage`, `condition` | Typed Connect; `space --projection headroom --json` | Per-source `UNAVAILABLE`. The headroom projection loses its readings. |
| `test-genie` | scenario (read-only) | no | `coverage`, `condition` | Typed Connect over `RunsService`; `space --projection validation-cost --json` | Per-source `UNAVAILABLE`. The validation-cost projection loses its readings. |
| `system-monitor` | scenario (read-only) | no | `coverage`, `condition` | Typed Connect over `metrics` and `investigations`; `space --projection attribution --json` | Per-source `UNAVAILABLE`. The attribution projection loses its readings. |
| `data-backup-manager` | scenario (read-only) | no | `coverage`, `condition` | Typed Connect over `plans`, `coverage`, `drills`, `audits`; `space --projection durability --json` | Per-source `UNAVAILABLE`. The durability projection loses its readings. |
| `agent-manager` | scenario (read-only) | no | `coverage`, `condition` | Typed Connect over `measures`; `space --projection agent-throughput --json` | Per-source `UNAVAILABLE`. The agent-throughput projection loses its readings. |
| `scenario-dependency-analyzer` | scenario (read-only) | no | `condition` (`supervision` projection) | core-set closure query | The supervision reconcile degrades to `UNAVAILABLE` — it must never fall back to an enumerated roster. |

## Vrooli Resources

No external Vrooli resources are declared. Storage is embedded SQLite via
`api-core/storage`; every other input is a read against a scenario or the
control plane.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None. | not-applicable | SQLite is embedded; the scenario holds no queue, cache, or vector state. | A reading volume that outgrows embedded SQLite, or a genuine need for cross-host reading history. |

## Scenario Dependencies

All read-only, all optional, all degrading independently. "Optional" here
means *the board still functions without it and says so* — never that the
reading is silently dropped.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `vrooli-autoheal` | **typed / partial** | Backs three of ten projections, the check registry, and two of four trust rules. Read-only Connect services and Measures expose checks, actions, incidents, healing, reconcile, shelves, and evidence counts; owner availability history remains an explicit gap. | Read-only. Never `shelve`, `unshelve`, `retire`, or any remediation verb — reading the reconcile is not the same as running the fix. |
| `vrooli capacity` | active | Claim coverage and declared-vs-observed reserve drift. Control-plane `internal/`, so the one legitimate CLI read. | Read-only. Never a policy-lever change, degrade, preempt, or release. |
| `storage-manager` | active | Device census, growth slope, declared-ceiling coverage. Has a typed handler surface. | Read-only; never triggers cleanup. |
| `test-genie` | active | Validation cost and cache reliability aggregate. `RunsService` is already consumed typed by `meta-optimization-manager`, so the precedent exists. | Read-only. Phase semantics stay with test-genie. |
| `system-monitor` | active | Process attribution over a saturation window, grouped by source scenario; investigation records. Has `metrics` and `investigations` protos. | Read-only; never triggers an investigation. |
| `data-backup-manager` | active | Backup freshness, coverage, and verified-restore drill outcomes — a projection this team never wrote despite the sensors already shipping. Has typed handlers for `plans`, `coverage`, `drills`, `audits`. | Read-only; never runs a backup or a restore. |
| `agent-manager` | active | Run statistics and durable error patterns. The only source that already exposes **Measures**, which is the direction the others should follow. | Read-only. |
| `scenario-dependency-analyzer` | planned | The core-set closure half of the derived should-be-supervised set. | Read-only derivation query, executed per read. |
| `secrets-manager` | blocked | Secrets availability is load-bearing for scenario start and has no cell. The scenario ships a `cli/` directory but **no `cli/manifest.json`**, so it is unbindable and exposes no typed read. | Blocked until it grows a health/status verb and a manifest. Tracked as an instrumentation gap, not compensated for here. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Every input is local. The scenario makes no outbound network call and needs no credential. | Would require a PRD change; there is no foreseen need. |

## Failure Modes

The governing rule: **a source that cannot be read produces a visible
availability entry with a stated reason — never a zero, never a silently
dropped row, and never a fabricated healthy verdict.** An unreadable
sensor is a finding about the instrument, not evidence about the plant.

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| Setpoint file | missing, unreadable, or unparseable | Hard failure with a stated reason. The board refuses to report a target count it could not read. | `coverage` parser tests |
| Setpoint file | parses, but a `NOW` cell has no bar, or a bar equals its authoring-time reading | Integrity finding against the *instrument*; the rest of the grid still loads. | `coverage` integrity tests |
| Owner `space --projection` verb | absent or unreadable | That projection's denominator is `UNAVAILABLE` with a stated reason and confidence drops to `SKETCH`. Cells keep their last authored status; none is fabricated as `MISSING`. | `coverage` space-reader tests |
| Any sensor source | timeout past `readDeadline` (10s) | That cell's reading is `UNAVAILABLE` with the reason verbatim. Other cells keep their readings. | per-source client tests with fakes |
| Any sensor source | process not running or unresolvable via discovery | Same as timeout — `UNAVAILABLE`, reason stated. Never reported as in-band. | per-source client tests |
| `scenario-dependency-analyzer` | unreachable | The `supervision` projection reports `UNAVAILABLE`. **Never** falls back to a cached or enumerated member list. | supervision seam tests |
| `vrooli-autoheal` check registry | ghost or saturated checks present | Readings are marked untrusted per [`TRUST-MODEL.md`](TRUST-MODEL.md) and excluded from aggregates; a finding routes to the instrument. | trust verdict tests |
| `vrooli-autoheal` Gap 10 verbs | reconcile / shelve unshipped | `GHOST` and `SHELVED` cannot be computed; affected readings are `UNTRUSTED`, never `VALID`, and the shortfall is reported as a finding against the instrument. | trust coverage tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
