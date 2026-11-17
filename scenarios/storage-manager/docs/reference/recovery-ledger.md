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

The Go build cache reclaims only through its tool owner. Go promises that an
output returned by a cache `Get` or `Put` stays on disk until that go command
exits. The command does not hold the file open for that whole time: an output
written by compile is reopened later by link or vet. File age, cache size, disk
pressure, an open-handle snapshot, and operator or standing approval therefore
prove nothing about whether a running build still needs an entry.

The proof comes from the control plane's build-cache protocol
([`internal/gocache`](../../../../internal/gocache/doc.go), operated by
`vrooli host go-cache`):

- `vrooli host go-cache activate` writes `GOCACHEPROG` into this user's go env
  file, so every go command this user starts uses the `vrooli-gocacheprog`
  helper. The helper stores the native on-disk format, and pins every output
  it serves in a session journal. The journal stays locked until the go command
  exits.
- Before it deletes from each cache shard, the owner audits the host for go
  commands outside the protocol: `GOENV=off`, another env file, another `GOCACHEPROG`, Go older
  than 1.24, commands started before activation, other users, and nested PID
  namespaces. While any of these runs against this cache, the owner refuses to
  delete.
- The owner removes entries least recently used first, down to the
  `build_cache` budget in `internal/tools/go/tool.json`. It never removes an
  entry used within the last two hours. Under its exclusive shard lock it
  checks each candidate again for pins and recent use before removing it.
- A plan token binds every apply to the store identity and the planned cutoff.
  A token from another store, or one with a forged cutoff, is refused.

Storage Manager plans and invokes; it never walks or deletes the root. The
`go-build-cache` root is `safe_with_owner` and declares
`owner_reclaim_command: ["vrooli", "host", "go-cache"]`. The
`OwnerReclaimProvider` runs the owner-reclaim/v1 contract through the process
runner seam ([providers](providers.md#owner-reclaim-providers)). Pressure
recovery runs it autonomously at rung R2 in bounded batches: at most 2 GiB per
apply, with a time budget inside the 45-second provider window. The provider
reports the bytes the owner actually freed. A refusal from the owner appears
verbatim as the provider's blocked reason, never as an empty cache.

The file provider and the tool-prune provider still refuse `go-build-cache` and
the legacy `spec-go-build-cache` identity on every path: estimate, preview,
apply, fresh-reclaim recovery overrides, and replay of previously approved
plans. `go clean -cache` remains refused. It deletes shards without build-use
coordination, and it resolves its cache from the runner's environment instead
of from the declared root.

Residual risk, accepted and documented in the protocol package:

- **Uncovered short command.** A go command outside the protocol that starts
  and finishes inside one shard's deletion window escapes the audit. The audit
  runs again before every shard to keep that window short.
- **Long-held outputs.** A `go/packages` consumer that holds an output for
  longer than the two-hour floor is unprotected.
- **Helper removal.** If the helper binary is removed while the protocol is
  active, every go command fails. The fix is `vrooli host go-cache deactivate`
  or `go env -u GOCACHEPROG`.

Provider registration reads the root declaration at service construction. A
source or declaration edit alone does not change an already-running Storage
Manager; activation requires its canonical lifecycle rollout. Verify the
running registry lists `go-build-cache` as an owner-budgeted `safe_with_owner`
provider. Verify too that `vrooli host go-cache status` reports the protocol
active, before relying on autonomous reclaim.
