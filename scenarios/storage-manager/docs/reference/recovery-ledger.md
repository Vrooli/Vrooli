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
