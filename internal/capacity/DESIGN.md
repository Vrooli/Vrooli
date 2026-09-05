# Capacity Broker — design note (`internal/capacity`)

Authoritative contract lives in `doc.go` (frozen, plan §8). This note is the
human-facing rationale and the map of how the pieces fit. Source plan:
`capacity-broker-internal-capacity-arbitration-system-monitor-ux`.

## Why this exists

The host GPU (16 GB) is oversubscribed by always-on model-server resources
(whisper large-v3 ~7.5 GB, kyutai-stt ~2.9 GB, reranker ~1.4 GB,
speaker-verification ~0.7 GB), leaving ~2.7 GB free. When image-tools tries GPU
image generation it OOMs. There is **no arbitration**: access is
first-come-first-served, claims are implicit, nothing reports who holds what,
nothing can ask a holder to step down, and a new heavy workload has no way to
claim a share or be told "wait / degrade / run on CPU." Each new GPU-hungry
scenario makes it worse. The broker makes claims explicit, observable, and
negotiable.

## Where it sits

```
   resources ──┐
   scenarios ──┼──> internal/capacity (claim ledger + Decide + Reconcile)
   lifecycle ──┘              │  reads
                              ▼
                    internal/hostinventory (live VRAM/RAM/CPU, per-PID GPU usage)

   system-monitor scenario ──reads──> internal/capacity ledger + hostinventory
                              (UX only; broker NEVER depends on system-monitor)
```

No `resource -> scenario` dependency: the broker is project-level `internal/`,
reachable by all three layers. system-monitor is a pure consumer.

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

## Mirror, don't refactor (plan §7 Phase 1 + §11)

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

## Levers (plan §2 control-surface-tunable-levers-design)

Every threshold is a row in `capacity_policy`, read by `Decide`/`Reconcile`,
editable via `vrooli capacity policy set`. Defaults are conservative and the
broker ships **advisory/OFF** — it claims + warns + records but never blocks a
start in V1. No silent caps: every threshold and every truncation is logged.

## Adoption is cooperative-first (plan §11)

Reconciliation surfaces unclaimed consumers as warnings (an adoption tracker),
not enforcement. Adopters opt in by declaring a claim profile and implementing
the `capacity` verbs. Enforcement (request-degrade, preempt) exists but is
config-gated; auto-stop stays OFF behind config + an allowlist.

## Phasing

0. Contract lock (this note + `doc.go`).
1. Engine: `CapacityClaim` + SQLite store + `Decide` (pure).
2. Reconciliation + attribution (observe/warn only).
3. Lifecycle admission hook (advisory, flag-gated, parity-tested).
4. Degradation contract + escalation ladder.
5. system-monitor health/modernization (prerequisite for 6).
6. system-monitor Capacity UX domain.
7. Adopters: whisper, ollama (via agent-manager), image-tools SD, kyutai-stt,
   audio-tools.
8. Operational interim wins + full validation.

Phases 1–4 and 5 are independent and may run in parallel; 6 depends on 4+5;
7 depends on 4.
