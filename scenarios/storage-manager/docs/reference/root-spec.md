# Governed root specification

The repository contract and owner `storage.entries` declarations are the
source of truth for storage roots. A root declaration names its path, class,
safety tier, age limit, byte limit, protected patterns, lease check, and
supported platforms. Adding a root is a declaration plus validation; it does
not require a provider-specific filesystem walker.

The repository roots are in `.vrooli/repo-contract.json` under `storage.roots`.
Examples include the Go build cache, Go module cache, uv cache, Hugging Face
cache, Playwright cache, scenario-to-desktop staging, browser evidence,
web-console sessions, test runs, and the governed Go work directory.

A root is eligible for recovery only when its declared tier and proof authorize
the action. `regenerable` means that the bytes derive from other inputs, the
owning tool can recreate them, deletion is contained to the exact root, and no
active lease protects the data. Durable models, scenario databases, and active
resources remain outside this class.

A root may instead name the tool that proves which of its bytes are unused.
`tool_prune_command` runs a tool's own prune command as one indivisible
effect. `owner_reclaim_command` delegates the root to an owner under the
owner-reclaim/v1 contract; it requires tier `safe_with_owner` and excludes
`tool_prune_command` ([owner-reclaim providers](providers.md#owner-reclaim-providers)).
The Go build cache uses `owner_reclaim_command`, because no file walk or
whole-cache clean can prove that a running build has released an output.

Use the storage-manager provider and recovery read surfaces to inspect the
resolved declaration. Do not hardcode a physical path in a new provider.

The same declarations tell coding agents where they may delete. At start-up
and every 30 minutes, `internal/policybridge` publishes them as `path_rules`
in the agent-policy bundle, through `vrooli-policy-runner publish`:
- **Storage roots.** Safe and regenerable roots allow deletion, safe-with-owner and conditional roots ask, and forbidden roots deny.
- **Runtime-home entries.** Regenerable entries that storage-manager reaps allow, and protected or `cleanup: never` entries deny.
- **Owner storage declarations.** An entry's `agent_removal` (allow, ask, or deny) states the answer explicitly. Without it, regenerable data allows, durable data denies, and SQLite sidecars always deny.

A new root therefore changes agent permissions as well as recovery. See
[agent-policy runtime](../../../../docs/architecture/agent-policy-runtime.md)
§"Filesystem removal".
