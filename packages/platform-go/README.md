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
