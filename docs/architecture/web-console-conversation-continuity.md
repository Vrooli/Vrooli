# Web Console conversation continuity

Web Console owns the lifecycle and local continuity of its sessions. Agent
Manager owns cross-run conversation retrieval, and Search Hub owns federation.
Neither downstream service may infer archive, recovery, retention, or deletion
authority from a Web Console search result.

## Source contract

Web Console exposes the typed `ContinuityService` under the Web Console API.
`ListCatalog` returns bounded, metadata-only source records containing the
stable Web Console session identity, native agent/session aliases, working
directory, title, lifecycle state, timestamps, and source fingerprint. It does
not return raw transcript paths, prompts, tool payloads, or secrets.

`Search` is the local recovery surface. It performs deterministic exact-text
search over retained Web Console events and can filter by lifecycle state,
agent, and lower time bound. It remains usable when Agent Manager, Search Hub,
embeddings, and LLM services are unavailable. Orphaned events are searchable
even when the `sessions` projection is absent.

`Integrity` is read-only and reports counts, orphan classes, and a deterministic
generation. `Reconcile` is dry-run by default. Apply requires an explicit
operation identifier and an optional generation guard; it writes only the
Web Console catalog projection and records a durable receipt. Repeating the
same operation is a replay, not a second mutation.

Each applied manifest stores a catalog-only preimage. An explicit, receipt-backed
rollback can restore that preimage transactionally; it never deletes or rewrites
conversation events, transcript checkpoints, workspace evidence, or native
agent histories.

## Import and federation boundary

Agent Manager may import catalog records and local-search hits through the
typed API. It owns any cross-run projection, ranking, semantic index, privacy
policy, and Search Hub provider descriptor after import. Search Hub must call
Agent Manager's provider contract rather than read Web Console SQLite files or
agent-home directories. A provider outage therefore degrades global discovery
without disabling local Web Console recovery.

Lifecycle changes remain authoritative in Web Console. Archive, recovery,
reopen, dismissal, retention, and permanent deletion produce Web Console
receipts and tombstone/source updates; downstream indexes consume those updates
and may not issue lifecycle mutations back into Web Console implicitly.

## Evidence and preservation

The catalog is a rebuildable projection. Conversation events, native agent
history, transcript checkpoints, workspace identity, and recovery lineage are
durable evidence. Reconciliation unions all known Web Console evidence
identities, including records whose session metadata is missing. It creates a
deterministic fallback title and marks message-bearing or native-history-bearing
orphans recoverable. Ambiguous aliases are quarantined rather than guessed.

The triggering regression is represented by a sanitized test fixture using the
original pane and native thread identities and 88 events. The fixture contains
no transcript body from the incident.

## Operational rule

Use `continuity integrity` before repair, review the dry-run generation and
classification, then apply with an explicit operation ID only after backup and
rollback evidence exists. Local search and inspection are safe read paths;
permanent deletion remains an explicit, receipt-backed operator action.
