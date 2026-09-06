## In scope

- Repair the Web Console product contract and requirements for durable archive, lifecycle attribution, reconciliation, and recovery.
- Create one domain-owned session and conversation continuity lifecycle.
- Preserve existing PTY backends and agent-native transcript formats.
- Replace row absence with explicit lifecycle and tombstone states.
- Add a canonical catalog for Web Console session, agent, workspace, and transcript aliases.
- Make archive, undo, reopen, recover, expiration, cleanup, and permanent delete idempotent and restart-safe.
- Detect and reconcile existing metadata-orphaned conversations, checkpoints, workspace panes, agent homes, and processes.
- Make local conversation discovery independent of live session metadata.
- Add stable Web Console API, CLI, and UI surfaces for archive, integrity, search, repair, recovery, and confirmed deletion.
- Persist lifecycle receipts and expose privacy-safe measures and health findings.
- Integrate Web Console sources with the active Agent Manager conversation-search architecture.
- Update Web Console documentation, invariants, temporal flows, configuration, usage skill, improve skill, and governed integrity-audit program.
- Remove displaced lifecycle, direct-deletion, and archive-search paths.
- Validate desktop, mobile, restart, concurrency, failure, replay, retention, and live-data reconciliation behavior.

## Change reach

The primary implementation belongs in `scenarios/web-console/**`. Wire-contract changes reach Web Console proto schemas and generated bindings. The Agent Manager and Search Hub paths are limited to the source integration and cross-scenario proof required to avoid a second search authority. Shared API/storage packages are in scope only when the clean domain boundary requires them.

## Execution coordination

Read `agent-manager-searchable-conversation-history` before changing Agent Manager or Search Hub. Reuse its active contracts and fixtures. Do not overwrite work from its active execution. Record any required ordering or boundary conflict before editing overlapping files.

