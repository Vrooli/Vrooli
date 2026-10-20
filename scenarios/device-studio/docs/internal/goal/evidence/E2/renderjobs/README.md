# D4 render coordinator core evidence

This package is a private, caller-driven coordinator. It connects immutable shot revisions, project-owned source media, the durable jobs ledger, routed blob storage, the Device Studio private PNG worker, and the platform-owned operation API. It adds no public route, scheduler, retry queue, or process supervisor.

The coordinator pins the running API executable bytes in routed blob storage before job submission. A queued job binds the worker digest, shot revision and media digests, renderer/profile identity, source-clock digest, and an internal storage-route fingerprint. Dispatch resolves and rechecks each source through the project media owner, persists a fixed operation intent before the jobs execution claim, and permits only the CAS winner to mutate its attempt workspace. A claimed attempt is reconciled through that exact intent; ambiguous dispatch is never retried automatically. Cancellation first enters the jobs ledger, then stops only the exact operation and records quiescence evidence before confirmation. PNG publication requires the manager's known terminal result and empty process tree, a role-bound worker receipt, matching output hash/size, bounded PNG dimensions, and a full pixel decode. Native outcome and completed-render evidence are durable before terminal completion.

`loadIntent` reads only the durable control record and validates the derived fixed command paths lexically. It does not require the staged workspace, so cancellation and exact-unit cleanup remain possible after staged files are deleted or corrupted. Output verification separately treats missing or changed staged artifacts as failure after native quiescence is known. Terminal `Advance`/`Reconcile` retries release using the persisted intent; a missing unit is tolerated only after the job is already terminal.

The compiled shot revision's `CompositorProfile` is compared with the worker's shared profile constant. The shot compiler owns motion, presentation, geometry, and input-bound eligibility. This coordinator accepts only the compiled profile and PNG output; simulated phone/browser chrome and video are outside this implementation.

## Verification

Focused validation run from `scenarios/device-studio/api`:

```text
go test ./internal/renderjobs -count=1
ok device-studio/internal/renderjobs

go test -race ./internal/renderjobs -count=1
ok device-studio/internal/renderjobs
```

The SQLite-backed coordinator regressions cover pinned executable bytes surviving a source-file replacement, one native dispatch under concurrent `Advance` (while allowing a caller to observe completion instead of the queued state), cancellation winning after the execution claim but before native start, a real-ledger cancellation-versus-completion race retaining the unit until quiescence is persisted, ambiguous start never being redispatched, lost test-route rejection, verified PNG publication, tampered PNG failure only after native terminal evidence, retrying an already-requested cancellation without treating a missing native unit as quiescence, loading control intent when the workspace is absent, and explicit-by-ID exact-unit release retry after reconstructing the coordinator. Worker, compositor, and jobs packages retain their own focused behavior tests.

The coordinator fixture exercises `Submit` through in-process worker composition and a fake operation owner; it does not launch a host process or perform an API binary rebuild. The route regression proves a stored test-route job fails closed when the route context is absent, but does not substitute for a real Test Genie lease journey. Terminal release retry is currently by known job ID through explicit `Reconcile`; the jobs ledger has no release-pending marker or bounded terminal-claim scan, so automatic discovery after a crash remains deferred to the public/background integration. The coordinator re-resolves current shot compiler metadata, so cross-version replay after compiler/profile evolution is also not established. Desktop/non-Linux owner support and end-to-end lifecycle admission remain separate work.

The platform owner's RuntimeMaxSec host qualification and Linux limitations are recorded in [the process-owner evidence](../../process-owner/runtime-max/README.md). D4 source-baseline hashes and the original authorized directories are recorded in `prewrite-baseline.txt`; the exact prewrite `go.mod` and worker README files are preserved alongside it. The new coordinator source directory was absent before the task, so those hashes do not describe preexisting coordinator code.
