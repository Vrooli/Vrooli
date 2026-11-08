# platform-go

This governed Go package is owned by the Vrooli repository. Its package
manifest is the machine-readable adoption and lifecycle record.

`StopScopeContext` retains `StopScope` ownership guards while bounding the Linux
user-manager request by a caller context. The guarded cgroup.kill fallback still
runs if the manager cannot finish. `StopScope` remains the compatibility entry.
Non-Linux unsupported behavior is unchanged. A bounded stop request does not
itself prove descendant exit; callers retain scope-observation evidence.

`Contained.StartContext` bounds Linux scope acquisition with the caller's
context. If acquisition fails after the root starts, the platform reaps that
known root and stops a scope only when it has a verified scope reference. An
unresolved reference returns `ErrContainmentCleanupUnverified`; root reaping
does not establish descendant cleanup. Existing `Start` remains compatible.

`ScopeProcessesContext` recognizes a collected Linux scope only when the exact
cgroup directory is absent and a bounded manager query confirms the matching
owned unit is inactive or absent. Missing cgroup.procs alone, an active unit,
an unavailable manager or inconsistent identity stays an error.

`StartOwnedOperation` is a separate Linux-first boundary for bounded local
work. The caller creates and durably stores `OperationIntent` before dispatch;
the owner derives a unique `vrooli-operations.slice` service from its immutable
spec digest and invocation ID. It requires `systemd-run` service placement,
`RuntimeMaxSec`, memory/task/CPU ceilings, `KillMode=mixed` (main-process
graceful shutdown followed by whole-cgroup kill), a
30-second/1,000-record journal rate limit, and a
manager `InvocationID`. It will not fall back to direct start followed by
cgroup movement. The transient service retains terminal `Result` and
`ExecMainStatus` until the caller has stored its own completion evidence.

`LookupOwnedOperation` recovers only a still-retained exact manager unit;
missing or ambiguous state is an unknown outcome. `InspectOwnedOperation`
checks manager invocation identity and recursive `cgroup.events populated`.
`StopOwnedOperation` sends TERM to the unit's main process, giving bounded
cleanup time for descendants such as a worker's GPU-lease releaser, then sends
KILL to the whole control group only if it remains populated. It verifies the
same recursive population signal. Its
receipt separates native quiescence from the local systemd-run launcher reap.
After the caller durably records output refs and terminal job evidence, it may
call `ReleaseOwnedOperation` to unload the retained unit. Release intentionally
destroys manager-side recovery evidence. The manager journals stdout/stderr;
this API provides no journal readback. `Env` is for non-secret executable
settings, not credentials, because systemd manager process arguments/properties
may expose those values to the same user.

Other operating systems currently return `ErrUnsupported` from owned-operation
entry points. This does not alter the existing agent, test, or service stop
guards and does not qualify Device Studio E2/E3 integration by itself.

`NewServiceIntent` / `StartOwnedService` provide a separate Linux-first boundary
for long-lived resource services. Persist the intent before dispatch. The fixed
invocation retains memory, task and CPU ceilings, uses `vrooli-services.slice`
and a distinct `vrooli-svc-*` namespace, and explicitly sets native runtime to
infinity. Bounded job admission still requires its wall limit and refuses service
intents. Service admission refuses job intents. There is no automatic native
restart that could replace the recorded invocation behind the domain owner's
back. Services retain the same exact native invocation, root incarnation and
recursive cgroup proofs as bounded work; lookup, stop and release reuse that
owner. Missing manager state remains unknown. An unsupported backend refuses
instead of falling back to an unowned process. This primitive does not implement
resource update admission, promotion or rollback by itself.
The resource owner must separately verify the running artifact, ownership token,
launch declarations, listener ownership and domain readiness. A native unit ref
alone does not establish those resource-level facts.
`ValidateIdentity` checks receipt shape without native I/O or a backend-support
claim. Service receipts retain the Linux cgroup namespace and exact native
invocation; resource checkpoints can inspect their structure on other hosts.

`CaptureServiceScope` / `ServiceScopeRef` cover historical resource services
already running in a dedicated `vrooli-service-resource-*.scope` under the
services slice. Capture requires an exclusive singleton native root and no
nested groups; it records the native invocation, root start time and cgroup
device/inode without inventing a launch intent. The domain owner separately
verifies its resource token, image and launch declarations. `InspectServiceScope`
keeps unknown observations distinct from absence. `StopServiceScope` uses a
held PID handle and held cgroup directory, so PID/unit-name recycling cannot
retarget the stop. It also supports reconciling an already-removed empty scope.
Other hosts fail closed. The opt-in `PLATFORM_GO_SERVICE_SCOPE_PROBE=1 go test
-race -run '^TestServiceScopeCurrentHostPinnedStopProbe$' -count=1 -v .` checks
stale receipt refusal, a child created after capture, forced tree cleanup,
repeat reconciliation, and actual scope-name reuse by a new invocation.

`ProcessArguments` preserves argv boundaries and empty values, with bounded
argument-block reads. Compare returned values in memory; they may contain
credentials and must not be copied into receipts. `OpenProcessExecutable` opens
the actual running image on Linux rather than a possibly replaced pathname.
Callers separately bind these observations to an OS incarnation. Darwin reuses
its exact native argv parser; running-image retention and unsupported backends
refuse rather than silently substituting a pathname or flattened command line.

`ReplaceProcessImage` executes an already-open ELF image on Linux. Replacing its
pathname between verification and execution cannot select a different file.
Scripts require a separate interpreter contract and are refused; other hosts
return `ErrUnsupported` without a pathname fallback. A subprocess regression
replaces the pathname with a different script and verifies that the held native
image executes. Darwin/Windows compilation does not qualify native execution.
`ServiceIntent.ValidateSpec` checks the complete fixed invocation without native
effects, allowing a resource wrapper to verify its persisted specification.

The opt-in `PLATFORM_GO_SERVICE_PROBE=1 go test -race -run
'^TestOwnedServiceCurrentHostCallerExitRecoveryProbe$' -count=1 -v .` drill creates
only its own fixed test service and descendant. It proves recovery from persisted
intent after normal dispatcher exit and forced dispatcher death, rejects duplicate
dispatch, checks native infinite runtime, and verifies exact tree stop and unit
release. Cleanup never selects unrelated services by name or PID.

`OperationSpec.Gate` optionally binds a typed, private `OperationGate` into a
bounded native invocation. A nil gate preserves the historical spec digest and
launch behavior. Prepare a gate under a private owner directory, persist its
identity with the operation intent, then dispatch once. The native manager runs
a fixed `ExecCondition` before `ExecStart`; no shell or arbitrary condition
command is accepted. Native argument quoting preserves spaces, quotes and
literal percent specifiers through systemd-run's D-Bus property encoder.

`PrepareOperationGate`, `Revoke`, and `Revoked` serialize under one per-attempt
platform file lock. Revoke atomically publishes and syncs a private tombstone
before unlinking readiness and syncing the directory. A replacement finishes an
interrupted bounded private marker without reopening execution authority; aliases
and non-private marker files remain refused. Preparation cannot reopen a revoked gate, including
when another process was paused after its initial check. Do not delete or
recreate an admitted gate's lock or tombstone while its domain can still replay
that operation. The gate is a revocable execution condition, not a once-only
dispatch claim; the domain must retain its own consumed request identity.

Before interpreting `OwnedOperationIntentAbsent` as safe cleanup, durably revoke
the gate and prove readiness absence. A unit whose condition already passed can
still be pending before ExecStart; its exact native identity must be stopped and
recursively empty. A denied condition is never successful executable evidence.
Unknown native state retains ownership. Other platforms do not acquire Linux
native-operation support through this gate.

The opt-in `PLATFORM_GO_GATE_PROBE=1 go test -race . -run
'TestOperationGate|TestOperationIntent|TestOperationSpec|TestOwnedServiceCommand'
-count=1 -v` checks delayed native submission after actual owner death,
condition-passed/pre-exec stop, literal gate paths, legacy ungated digests and
barrier-controlled cross-process preparation/revocation.
