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
