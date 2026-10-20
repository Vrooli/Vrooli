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

## Correctness boundaries for admission and observation

Idle dwell begins at the transition from active to idle. `LastActiveAt` records
the last instant known active, including that transition. Repeated idle reports
and liveness heartbeats do not restart the dwell. Planning and CLI projection
use the same `IdleSince` calculation; `UpdatedAt` is not an activity clock.

A successful inventory call can contain an unsuccessful or incomplete GPU
process probe. Only a complete, successful process probe proves absence. During
unknown sensing, positive unambiguous presence can renew a resident, but missing
owners are neither expired nor sampled as zero. Operation lease expiry remains
a separate liveness concern. Native CLI admission performs the same safe sweep
before deciding, using its already captured snapshot.

Escalation plans carry the target generation. Before external actuation the
owner rechecks that exact generation and that the row remains live, idle and
unprotected. Heartbeats preserve generation and therefore do not invalidate a
plan. This closes stale-plan actuation before the callback; it does not make
activity arriving during a callback atomic with a restart or unload. Enforcement
remains unchanged. Adopter request/drain fencing requires a separate owner proof.
The current preempt action records ledger intent only. Without an owner stop or
unload proof it reports no applied physical action and zero reclaimed bytes.

One observed process can belong to at most one active claim on its GPU. Matching
uses exact normalized identities and refuses ambiguous matches; resource-family
prefixes do not identify individual operation jobs. Reconciliation compares the
aggregate assigned workers with their one reservation and retains worker context.
Presence renewal and sampling share that assignment rather than independently
attributing the same process to multiple rows. Unbound jobs remain unknown.

Durable footprint peaks remain monotonic, but their source describes the evidence
for the winning peak. A smaller measured sample cannot turn a larger declared
estimate into a measured peak. Manifest seeds do not count as measurements.
Previously contaminated measured rows cannot be reconstructed from this table;
an explicit owner reset/rebuild needs original evidence and is not automatic.

Outstanding design gaps are identity-aware overlap between physical usage and
pending reservations, parent/resource versus operation reservation semantics,
stable process binding for generic operations, cross-process admission
serialization, and the activity-during-callback drain boundary. None is solved
by loose matching, a policy flip, or evicting existing workloads.

## Storage (plan §2 storage-steer)

SQLite ledger at `<runtime-home>/state/capacity.db` (resolved via the
repo-contract runtime-home `state` dir; no new contract entry needed — stays
within the outside-scenario allowlist). `modernc.org/sqlite`, single-conn,
WAL + `busy_timeout`, `withRetryableTx` from day one for SQLite-BUSY discipline
(cf. backlog `fix-vrooli-lifecycle-runtime-claim-release-sqlite-busy-20260616`).
Own `SchemaVersion` + `PRAGMA user_version`. Tests pass an explicit temp
`DBPath` exactly like `scenarioruntime` tests.

The shared `ResolveConfiguredDBPath` owner boundary rejects an implicit live
ledger during Go tests unless `DBPath` or `HomeDir` explicitly isolates it.
Both the SQLite store and public broker call this boundary before resolving the
default path or taking a constructor lock. The store still owns canonical file
identity; the broker still serializes schema initialization before opening it.

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

### Absence observation freshness

A successful GPU process probe proves absence only when its `gpu_processes`
provenance timestamp is nonzero, no later than the evaluation clock, at most
the default heartbeat TTL old (30 seconds), and not before the claim's creation, latest update, heartbeat,
or active boundary. The shared default heartbeat tuning bounds the default host fact-cache interval;
changing sweep cadence does not relax this safety bound. Missing, future, old,
or pre-liveness observations defer resident expiry and zero sampling. Positive
presence remains conservative. A cached successful empty probe is not proof of
absence for a newer lease. The fact cache does not serve expired data on errors.

### Reclaim cohort preservation

Enforcement plans only the exact claim IDs named by Decide.ReclaimTargets. It
must preserve the admission resource/GPU cohort rather than replan against all
active claims. Current generation/activity/protection is still revalidated
before each callback; an absent/stale admitted cohort does not authorize a
substitute owner or GPU.

### Atomic absence proof

Resident expiry defaults to unknown. Its explicit positive proof carries the
captured generation and process-probe timestamp. The SQLite transaction checks
the CURRENT row's generation, liveness/activity boundary and deadline before
expiring it. A newly appearing row has no proof and cannot expire. Zero usage
sampling requires the equivalent proof checked inside its write transaction.
Heartbeat keeps generation, so timestamp/deadline checks remain necessary.
Unknown diagnostics are separate from mutation authority; an exclusion denylist
is insufficient. Conflicting canonical owner identities are unresolved before
ledger lookup and cannot become authoritative because only one candidate holds
a claim.

Declared adoption applies the same canonical conflict refusal before ANY spec
lookup or creation. Refusing matchClaim alone is insufficient because adoption
would otherwise fall through to candidate-by-candidate declared-owner lookup.

### Canonical native GPU wire identity

Core nil VRAM GPUIndex already denotes GPU0, including lifecycle-declared and
historical rows. Native ClaimView projects that existing identity explicitly
as gpu_index=0 so strict companion consumers need no malformed-payload default.
Explicit devices remain unchanged; RAM/CPU absent GPU stays absent. This is
projection normalization, not a storage rewrite or inferred process binding.

### Opened ledger identity and operation heartbeat write authority

The SQLite owner fixes the ledger's absolute/canonical path BEFORE opening the
database, preserving :memory: as a special non-file identity. DBPath describes
that opened identity even if a caller changes working directory before creating
a public broker. Existing symlink paths resolve to the same opened file.

Operation heartbeat expiry is evaluated after transaction connection AND SQLite
write authority are obtained. Deferred BeginTx alone is insufficient: UPDATE may
wait behind another SQLite writer. Heartbeat first performs a no-row UPDATE in
the existing transaction to acquire write authority without changing rows, then
reads the owner clock and performs the existing conditional renewal. Busy retries
repeat authority acquisition and clock reading. No global transaction-mode,
schema or host/lifecycle policy changes are required; resident heartbeats still
renew independently of expired operation semantics.

Absence-proof consumers use the same narrow write-authority gate as heartbeat.
Resident expiry and zero-observation sampling must acquire SQLite write
authority before reading current rows/clock. Expiry rechecks proof age with a
fresh clock immediately before EACH candidate write, so a long scan/batch does
not reuse a once-fresh timestamp. Returned expiry UpdatedAt uses that row's
actual mutation clock. Zero sampling checks after its current row read and
immediately before writing. Operation TTL remains independent of resident
proofs; supplied expiry cutoff remains the caller's explicit sweep boundary.
