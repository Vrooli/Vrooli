# Temporal Flows

## Last Updated
2026-11-02

## Pipeline Execution Flow

The core async pattern is the multi-stage pipeline orchestrator in [CODE: api/pipeline/orchestrator.go].

### Execution Modes

```
Client Request
    │
    ├─ ?block=true ──→ RunPipelineBlocking(ctx, config, timeoutSecs)
    │                   - Extends HTTP write deadline
    │                   - Returns when pipeline completes or timeout
    │
    └─ (default) ────→ RunPipeline(context.Background(), config)
                        - Returns pipeline ID immediately
                        - Pipeline runs independent of HTTP lifecycle
                        - Client polls GET /status/{id} for updates
```

### Stage Progression

```
Bundle → Preflight → Generate → Build → SmokeTest → Deploy
  │                                                     │
  └── Any failure ──→ Pipeline marked Failed ───────────┘
                      (no automatic retry)
```

Each stage updates `PipelineStatus` atomically via `InMemoryStore.UpdateStage()`.

## Concurrency Model

### Store Thread Safety

`InMemoryStore` in [CODE: api/pipeline/store.go] uses `sync.RWMutex`:

| Operation | Lock Type | Notes |
|-----------|-----------|-------|
| `Get()` | RLock | Concurrent reads safe |
| `Save()` | Lock | Exclusive write |
| `Update(fn)` | Lock | Callback executes under write lock |
| `UpdateStage()` | Lock | Atomic stage update |
| `GetByIdempotencyKey()` | RLock | Concurrent lookup safe |
| `Cleanup()` | Lock | Removes old entries |

### Goroutine Patterns

- **Async pipeline**: Launched as a goroutine with `context.Background()` — survives HTTP connection close
- **Blocking pipeline**: Runs in HTTP handler goroutine with extended deadline
- **Runtime health polling**: `main.ts` polls `/ready`, `/ports`, `/secrets` endpoints from Electron main process
- **Child process management**: Bundled runtime spawned via Node.js `fork()` with lifecycle monitoring

## Race Condition Guards

| Scenario | Guard | Location |
|----------|-------|----------|
| Duplicate pipeline submission | Idempotency key check before start | `orchestrator.go` |
| UI double-click | `isSubmitting` flag + idempotency key | `pipelineStore.ts` |
| Concurrent status reads during update | RWMutex on store | `store.go` |
| Pipeline timeout vs completion race | Context cancellation + status check | `orchestrator.go` |

## Checkpoint Flows

Pipeline stages are checkpointed to the store after each stage completes. On resume:
1. Load pipeline status from store
2. Determine last completed stage
3. Resume from next stage (stages are not re-run)
4. `resumePipeline()` in UI resets `isSubmitting` with fresh idempotency key

## Desktop update installation (SM9)

The domain-local [lifecycle spec](../../templates/vanilla/storage/update-lifecycle.spec.json)
declares the complete state/event matrix, invariants and recovery traces.
[Update recovery](../../templates/vanilla/storage/update-recovery.ts) owns the
production transition function and serialized atomic state writes;
[update apply](../../templates/vanilla/storage/update-apply.ts) owns the
mutually exclusive pending/applying/installing phases. The existing state file
retains a bounded trace across process restarts, while the intent and receipt
continue to own interruption classification and installed identity.

The template conformance test checks every declared pair and trace. The real
J10 AppImage tests replay persisted transitions against the same declarative
contract and assert predecessor/successor data preservation, ordinary quit,
interruption classification and the original Windows build proof. This is a
level-four declarative contract; no checked formal-model claim is made.

## Optional component generation lifecycle (SM10, target Level3)

`runtime/components/manager.go` owns the installation pointer and immutable
code generations; `runtime/components.go` owns service startup, readiness,
stop, and rollback-or-disable. This is a scoped W2 transition-evidence closure
for REQ-D2/D4/D5/D6 and J11, not a new lifecycle store or a Level4 claim.
`runtime/components/lifecycle_test.go` replays the following contract against
those file-backed Manager operations. It does not use a replacement transition
implementation in production.

### States and orthogonal facts

| Name | Representation and meaning |
|---|---|
| available | Catalog observation; no installation row is required. `Available` is not a persisted Manager assignment. |
| installing | Locked download, extraction and verification work; the committed row remains unchanged. `Installing` is not persisted. |
| pending_next_start | A verified generation is staged. It enters the service graph at the next boot. |
| installed | The generation passed boot admission. Readiness is separate: `ActivationPending` and `Previous` survive until the actual graph is ready. |
| needs_update | Signature/tree/collision admission failure, or an explicitly disabled activation. `Reason` explains the branch. |
| out_of_range | The current host/platform cannot admit the signed generation. A later compatible boot can admit it again. |
| removing | Supervisor stops component services, then Manager unpublishes the row before deleting code. `Removing` is not persisted. |
| absent | No row. Data may remain after an explicit keep-data removal. |

The stored-state domain is `{pending_next_start, installed, needs_update,
out_of_range}` × `ActivationPending` × `ActivationDisabled` × presence of
`Previous`, plus absent. The test checks all **33 cells**, including defensive
flag combinations not normally produced by Supervisor. A generation identity
and its signed entry remain inseparable; `Previous` is an independent retained
identity, not an extra current installation. `Reason` is explanatory text, not
a state. Available/installing/removing are operational projections of the same
cells, so no fictional persistent statuses are added to satisfy the matrix.

### Complete owner event matrix

Each row below applies to every one of those 33 cells. “Unchanged” includes
current generation, retained fallback and activation flags. All rejected
operations leave the committed row unchanged. Read/list/reopen are observations.

| Event | Absent | Every present row (guards evaluated in order) |
|---|---|---|
| install different verified generation | pending, no fallback | pending; retain existing Previous, otherwise retain current generation only if installed and different; clear disabled/pending flags and reason |
| retry same verified generation | pending, no fallback | pending; **preserve existing Previous**, clear disabled/pending flags and reason; never retain self as fallback |
| invalid signature, download failure, invalid archive | refused, absent | refused, unchanged; verification precedes pointer publication |
| merge verified graph | absent | disabled: unchanged and no services; otherwise installed, clear reason, preserve Previous; set ActivationPending if prior state was pending, otherwise preserve it |
| merge bad signature/tree | absent | disabled: unchanged; otherwise needs_update with reason, no services, same activation/fallback rule as merge |
| merge incompatible host | absent | disabled: unchanged; otherwise out_of_range with reason, no services, same activation/fallback rule as merge |
| merge service collision | absent; existing host services survive | disabled: unchanged; otherwise needs_update with reason, no component services, same activation/fallback rule as merge |
| matching readiness completion | refused | clear Previous and ActivationPending; preserve other facts (Supervisor only calls this after graph readiness) |
| superseded readiness completion | refused | ErrActivationSuperseded, unchanged |
| matching activation disable | refused | needs_update, supplied reason, ActivationDisabled=true; clear Previous and ActivationPending |
| superseded activation disable | refused | ErrActivationSuperseded, unchanged |
| matching rollback | refused | require Previous and reverify its identity/signature/platform/tree; select it as installed and clear both activation flags and Previous |
| stale rollback or invalid fallback | refused | refused, unchanged; Supervisor then disables the matching failed generation when recovery cannot start |
| remove keep data | absent, delete any remaining code | absent, code removed, data untouched |
| remove delete data | absent, delete remaining code/data | absent, code and data removed |
| reopen/read | absent | unchanged; future boot rechecks admission, and disabled generations remain off |

There are **20 executable event partitions × 33 cells = 660 transitions**.
Signature, transport, archive, boot admission, generation equality and retained
fallback checks are exercised through the production methods. `Catalog` is a
read-only discovery operation, not a pointer transition. Source-not-configured
boot is an observation/no-op; source-not-configured install is refused. Invalid
component IDs are rejected before removal. Lock/read/save errors abort the
operation; the atomic pointer writer owns partial-write protection. Removal can
be retried after unpublication to finish code/data cleanup. These infrastructure
failure guards are distinct from the 660 lifecycle cases.

### Traces, interruption and invariants

`TestComponentLifecycleTraces` installs signed archives from an isolated HTTP
server and performs clean install → boot → readiness; update → repeated stage
→ failed readiness → rollback; update → boot → superseding stage → stale
completion → next boot → readiness; and failed first activation → disable →
restart → explicit reinstall → readiness. The matrix checks both data choices
from every cell and checks tamper/range/collision admission independently.

`TestComponentProcessRecovery` kills a separate Go test process while the real
Manager holds its installation lock during staging. The previous pointer
survives; retry reacquires the lock. A second process then merges the staged
update and exits before readiness. Reopening preserves ActivationPending and
Previous, and rollback restores the old generation. This proves process-death
recovery at these boundaries, not power-loss durability or every filesystem
failure. A killed staging operation can leave an unreferenced `.install-*`
directory; it never becomes the committed executable generation.

The matrix exposed a retry defect: the same pending generation lost Previous
because fallback retention was inside the different-generation guard. The
bounded correction preserves an already retained fallback on any successful
stage and still prevents retaining the current generation as its own fallback.
No readiness, signing, selection or service-stop authority changes.

### J11 evidence bridge and limits

`TestComponentRetainedJ11Replay`, with `J11_RELEASE_ROOT` selecting the retained
signed Audio Tools release, runs the actual archive through install, merge,
readiness completion, tamper rejection, keep-data removal, reinstall and
delete-data removal. It supplies a readiness event at the Manager boundary;
actual process health and UI behavior remain covered by the unchanged J11
journeys below. The retained archive is reverified without resigning or editing
its signed URL; a test transport supplies the original archive bytes.

| Unchanged J11 check | SM10 connection |
|---|---|
| InstalledFromIntegrationsPageOnCleanHome | stage → restart → ready on actual Audio Tools process |
| EmbeddedAndInstalledEntriesIdentical | verified signed manifest projects the same owner services |
| TamperedEntryRefused | metadata admission refusal and boot tree exclusion |
| UpdateFailureKeepsOldVersion | failed download preserves pointer; failed real startup restores fallback; invalid fallback disables optional component and preserves host/data |
| UninstallOffersDataChoice | cancel is unchanged; stopped code is removed; keep/delete data choice survives reinstall |
| AddHereOrGetFullAppBothOffered | catalog/UI choice; no extra Manager state |
| LiveFullAppWinsOverSlice | SM11 selection; stopped slice retains its installation |
| SliceReturnsWhenFullAppCloses | SM11 selection reuses the verified installed generation |
| UsageSliceDrivesUsageBars | actual installed slice capability; readiness is separate from staging |

Retained evidence: `~/.vrooli/evidence/agent-connect/J11-latest.log` records all
nine passing names and archive SHA256
`7a5dbdffaed89f47ea74fc97957307cefb19ab4ded1f506fb268e951a881abe1`.
Current owner verification and failing-before retry evidence are under
`~/.vrooli/evidence/agent-connect/t8-sm10/`. No top-level board is run by this
package lane. The coordinator owns current combined J11 validation and
independent acceptance; these package traces alone do not certify UI/process
readiness or the neighboring provider-selection machine.

## Answering provider selection (SM11)

Selection is a fresh observation per call, not a persisted selected-provider row.
`runtime/peerlookup` owns identity, surface, and version admission;
`runtime/api/scenario_port.go` composes full-peer then local bare discovery;
`runtime/components.go` owns explicit feature health, local stand-down, and
verified embedded/installed fallback. `internal/peerrecord` remains the only
same-user publication/liveness owner. Program-binding ranges take precedence
on all three feature paths; a feature uses its advertised owner version.

The design states map to these existing facts:

| Design observation | Production meaning | Next lookup |
|---|---|---|
| Embedded | Exactly one declared compatible embedded owner; readiness succeeds | Full healthy compatible feature wins, otherwise embedded |
| Slice | Explicitly requested signed installed feature, verified generation and readiness | Full healthy compatible feature wins, otherwise verified feature activation; bare discovery excludes every slice |
| Full app live | Live owner PID, explicit feature/valid route, compatible owner version, successful health probe | Stop matching local owner before returning exactly one full result |
| Full app closed | Record removed or owner no longer live | Reevaluate embedded/installed fallback; no cached full route |
| No eligible responder | Missing, ambiguous, incompatible, unready, or changing provider | Return failure, never an incompatible or partial selection |

Publication alone is not feature readiness. Bare port discovery reports an
eligible advertised port and does not promise endpoint health. The feature
handoff retains the initially admitted record, then reads through the same
admission owner after stand-down. A changed owner PID, instance/start time,
feature version/route, missing record, or changed range refuses the handoff.
This is a call-boundary guarantee; it cannot prevent a process exiting after
selection, and callers must handle request failure.

The executable matrix is explicit in `TestSM11LookupTopologyEventMatrix`:
6 local topologies (none, embedded, installed, slice, ambiguous, own app) ×
8 full-peer observations (absent, eligible, owner out of range, feature out of
range, missing surface, missing feature, lost owner, moved route) × 8 incoming
observations = 384 pairs. Each pair reads real peer records before and after
and asserts `External`, `FeatureExternal`, and `Local`. Repeated observations
and repeated closure are included. `TestSM11LookupRangeAndRouteReplay` checks
same-port version changes, program range override, moved route, withdrawal,
and closure without retaining a selected provider.

`TestSM11SupervisorTopologyEventMatrix` covers both embedded-present and
embedded-absent topologies × 4 publication/health observations × 5 events =
40 pairs. Its transition table includes repeat lookup, ready publication,
incompatible publication, health loss, and close. The publication/fallback
trace exercises production `publishPeerRecord` and `ResolveFeature` with a
real HTTP health endpoint, including unready publication, stand-down, recovery,
loss, closure, and failed/recovered local readiness. Local process startup in
these focused tests is explicitly dry-run. Actual installed process stop and
restart remain the real J11 `LiveFullAppWinsOverSlice` and
`SliceReturnsWhenFullAppCloses` obligations, alongside all seven other unchanged
J11 checks; focused matrices do not replace that journey receipt.

Five deterministic mid-probe mutation regressions cover feature-version
change, withdrawal, replacement instance at the same port, route change, and
record loss. Two embedded-binding regressions enforce range precedence in
both directions. The tests exposed and repaired a port-only final recheck and
an embedded-only use of the ordinary peer range. This is Level3 evidence for
the declared observation/transition model, subject to source-current journey
and review acceptance; no declarative-spec or formal verification claim is made.
