# Capacity Broker — design note (`internal/capacity`)

Authoritative contract lives in `doc.go` (frozen, plan §8). This note is the
human-facing rationale and the map of how the pieces fit. The current source
plan is `capacity-seam-make-per-machine-capacity-placement-and`.

## Why this exists

Model-serving resources can declare more GPU memory than one machine owns.
The broker makes those claims explicit, records measured footprints, evaluates
the complete enabled set, and produces degradation proposals before a resource
start causes an out-of-memory failure.

## Where it sits

```
   resource manifests ──> fit + declared defaults ──┐
   resource CLIs ───────> shared companion ─────────┼──> internal/capacity
   scenarios/operations ─> shared broker client ────┘       │
                                                             ▼
                                              SQLite claim + footprint ledger

   system-monitor scenario ──reads──> internal/capacity ledger + hostinventory
                              (UX only; broker NEVER depends on system-monitor)
```

Resource modules use `packages/capacity`; they do not import the control
plane's `internal/` tree. The control plane owns host sensing, policy, fit,
reconciliation, and actuation. System Monitor is a pure consumer.

## The two-axis liveness/activity model (the core insight)

A claim has two independent dimensions, never conflated:

- **Liveness** — `heartbeat_deadline_at`. Is the owner process still alive? A
  missed heartbeat past the deadline sweeps the claim to `expired`. This is the
  same mechanism as `scenarioruntime` leases.
- **Activity** — `activity_state` (`active`/`idle`). Is the owner doing work
  right now? This is **reported by the work-owner**, never inferred. A coding
  agent holds an `active` claim for the duration of a session and marks `idle`
  on completion. This solves the "ollama has been resident 20 min ≠ idle"
  problem: 20 minutes of wall-clock says nothing about whether work is in
  flight.

Preemption/degradation decisions key off **activity**, not age. An `active`
interactive claim is `protected` and never touched. Only an `idle`, unprotected,
strictly-lower-priority claim is reclaim-eligible — and degradation (step down a
profile rung) is always tried before preemption (stop), which is the last rung
and config-gated.

## Placement and durable evidence

`vrooli capacity fit` combines measured host inventory, enabled-resource
choices in operator state, manifest rungs, transient headroom, and durable
footprints. It returns `fits`, `fits_with_changes`, or `over_subscribed` and a
deterministic proposal. Fit does not mutate operator state.

`capacity_footprints` keeps a monotonic high-water mark for each resource,
rung, footprint-affecting tunables key, and GPU index. Claim garbage collection
does not delete this table. The recommender uses measured footprint rows for
over-consumption warnings and stays silent when no measured sample exists.

The operator selects a capacity posture. `minimal` also derives an acquisition
preference of `force_cpu`; other postures derive `auto` unless the operator sets
`auto`, `prefer_gpu`, or `force_cpu` explicitly. Acquisition receives this
choice as the `operator.accel_preference` fact.

## Mirror, don't refactor

`internal/scenarioruntime` is the proven pattern source: `lease.go`
(Create/Heartbeat/ExpireStale/Stop + `ErrStaleGeneration`), `sqlite.go`
(`withTx`/`withRetryableTx`/`formatTime`, repo-interface-per-concern), `schema.go`
(`SchemaVersion` + `PRAGMA user_version`). Its lease engine is welded to the
`Instance`/`runtime_instances` table, so it is **not importable** for our claim.
We mirror the shapes in this package with our own `capacity_claims` table.
Destabilizing scenarioruntime/lifecycle is the catastrophic risk; the engine is
a brand-new package and the lifecycle hook is additive + flag-OFF parity-tested.

## Seams (plan §2 seam-discovery-and-enforcement)

- **`CapacitySource`** — reads `hostinventory.Snapshot`. Injectable; unit tests
  use a fake snapshot and never shell out to `nvidia-smi`.
- **Attribution** — PID → {container, scenario/resource}, via
  `/proc/<pid>/cgroup` (`docker-<id>`) + a docker-name map. Build-tagged for
  linux; non-linux returns "unknown" cleanly (cross-platform-readiness).
- **Degrade callback** — the broker invokes the adopter's declared
  `degrade_profile.apply` verb; the adopter owns the actual resize.

All three are interfaces so the engine is fully unit-testable with no real
host calls.

## Storage (plan §2 storage-steer)

SQLite ledger at `<runtime-home>/state/capacity.db` (resolved via the
repo-contract runtime-home `state` dir; no new contract entry needed — stays
within the outside-scenario allowlist). `modernc.org/sqlite`, single-conn,
WAL + `busy_timeout`, `withRetryableTx` from day one for SQLite-BUSY discipline
(cf. backlog `fix-vrooli-lifecycle-runtime-claim-release-sqlite-busy-20260616`).
Own `SchemaVersion` + `PRAGMA user_version`. Tests pass an explicit temp
`DBPath` exactly like `scenarioruntime` tests.

## Levers

Every threshold is a row in `capacity_policy`, read by `Decide`/`Reconcile`,
editable via `vrooli capacity policy set`. Defaults are conservative and the
broker currently ships **advisory**. Do not enable enforcement until the live
activity, fit, degradation, and upshift gates all pass. No threshold is a
silent cap: policy rows and verdicts expose every decision input.

## Adoption is cooperative-first (plan §11)

Reconciliation surfaces unclaimed consumers as warnings (an adoption tracker),
not enforcement. Adopters opt in by declaring a claim profile and implementing
the `capacity` verbs. Enforcement (request-degrade, preempt) exists but is
config-gated; auto-stop stays OFF behind config + an allowlist.

## Shared adopters

Test Genie uses the shared broker client for per-phase RAM and CPU admission.
The Ollama, reranker, Whisper, Kokoro, Kyutai STT, and speaker-verification
resource CLIs use the shared lifecycle companion. The repository test
`TestDocumentedSharedCapacityAdoptersExist` verifies these source boundaries.
