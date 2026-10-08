# Recovery ledger

The orchestrator schema owns three bounded evidence tables:

- `recovery_runs` stores trigger, mount, target free bytes, terminal result,
  stop reason, and reclaimed bytes.
- `recovery_actions` stores one provider action with rung, authority, files
  removed, bytes reclaimed, and free space before and after.
- `writer_snapshots` stores sender/root growth observations and hot state.

Recovery history and writer ranking read these integer columns directly. They
do not parse audit messages or rescan the filesystem. Writer snapshots are
pruned older than 30 days on write and by the retention cycle; recovery
evidence is retained for 90 days through the storage-manager declaration. The
pruner deletes only derived ledger rows, never the source paths represented by
those rows.

## Shared Go build-cache safety

The `go-build-cache` root is conditional and requires owner evidence, not a
`no_lease` regenerable declaration. Go supports concurrent local builds, but a
cached output must remain available between compile, link, and vet. A file need
not have an open handle throughout that interval. Age, cache size, disk pressure,
and operator or standing approval do not prove build quiescence.

The generic file provider MUST refuse estimate/preview and apply for
`go-build-cache` and the legacy `spec-go-build-cache` identity until a tool-owner
build-use or quiescence protocol exists. The refusal also covers old regenerable
declarations, fresh-reclaim recovery overrides, and replay of previously approved
plans. Unavailable reclaim is reported as blocked, not as an empty cache. The
declared retention limits are not authority to delete. Go's own trimming remains
the available cache lifecycle; do not substitute `go clean -cache` or an external
mtime/size cap while builds may be active.

Provider registration reads the root declaration at service construction. A
source/declaration edit alone does not protect an already-running Storage Manager.
Activation requires its canonical lifecycle rollout. Ordinary policy disablement
is not containment because autonomous recovery can override that policy. Verify
the running registry withholds the conditional root after activation; do not
test this invariant by triggering live cache deletion.
