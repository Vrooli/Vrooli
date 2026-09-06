## Trigger and user impact

On 2026-09-04, the operator closed Web Console pane `a7e71c3c-e422-4c89-916a-03f92906fb89` after Codex appeared frozen. The operator later remembered a discussion about Plan Manager, Agent Manager, Git Control Tower, Test Genie, Program Runtime, and scenario skills. The operator could not find the conversation in the Web Console archive and reasonably expected that a plan might have been created.

No plan existed. The discussion occurred inside Codex thread `01a06a6b-88da-7422-b391-bb59c5f5e5e0`, which began as an onboarding conversation and later changed topics. The stored title described the original onboarding work rather than the recent architecture discussion. Searching plans and global Codex history therefore produced plausible but incorrect matches.

The conversation was recovered only after correlating Web Console lifecycle events, process environments, SQLite rows, per-pane agent homes, transcript checkpoints, and raw Codex rollouts. A normal operator cannot be expected to perform this forensic procedure.

## Confirmed data shape

The raw rollout remained intact at:

`/home/matthalloran8/.vrooli/state/vrooli/web-console/sessions/codex/a7e71c3c-e422-4c89-916a-03f92906fb89/sessions/2026/09/03/rollout-2026-09-03T23-17-07-01a06a6b-88da-7422-b391-bb59c5f5e5e0.jsonl`

The Web Console database retained 88 conversation events, the conversation cursor, the Codex tailer checkpoint, and the workspace pane. The corresponding `sessions` row was absent. Archive listing and archived-message search require that row, so intact evidence became invisible.

The exact deletion caller cannot be reconstructed. Web Console does not persist a lifecycle receipt that identifies the actor, intent, prior state, resulting state, and outcome. Some persistence operations discard errors. The event buffer retained pane and connection activity but no conclusive archive, delete, expiration, or cleanup transition.

## Systemic integrity drift

The live database snapshot taken during plan authoring contained:

| Relation | Count |
| --- | ---: |
| Session metadata rows | 332 |
| Archived session rows | 0 |
| Conversation identities | 421 |
| Transcript checkpoints | 330 |
| Workspace pane rows | 1421 |
| Conversations without session metadata | 116 |
| Transcript checkpoints without session metadata | 97 |
| Workspace panes without session metadata | 1342 |

Not every stale workspace row is valuable. However, recent message-bearing conversations appear among the orphans. The triggering conversation is the newest known example. The defect is therefore a systemic integrity problem rather than a one-off archive presentation bug.

## Fragmented lifecycle ownership

Session creation, process exit, archive, undo, explicit deletion, expiration, owner cleanup, startup recovery, workspace layout, conversation ingestion, and agent-home cleanup are implemented by different layers. Several layers can delete metadata directly. The generic process-exit watcher knows the backend but does not know the lifecycle intent that caused exit.

Archive currently has a delayed temporal shape: mark archived, return success, retain an undo timer, and terminate later. Tests prove the marker and undo path but do not prove final persistence across every backend, recovered-session state, restart point, or concurrent client condition. An immediate-finalization standard-session test does not assert that the archived row survives the asynchronous process-exit cleanup.

The current `docs/internal/INVARIANTS.md` describes removal and expiration as delete-oriented operations. The current `docs/internal/TEMPORAL-FLOWS.md` lists session auto-cleanup and expiration, but it does not model the archive/delete/recover lifecycle as one workflow. Documentation therefore reflects the fragmented implementation rather than a single product contract.

## Identity and discovery fragmentation

A conversation may have a Web Console pane UUID, an agent thread UUID, an agent-home path, a rollout path, a conversation-store identity, a workspace identity, a transcript checkpoint key, a title, and recovery lineage. No canonical record maps all aliases.

Per-pane `CODEX_HOME` isolation is correct for multi-agent safety. Global Codex history cannot see those homes, so Web Console must catalog them. When metadata disappears, isolation becomes invisibility.

Web Console search is partitioned into known-session search and archived-session search. Archived search inner-joins session metadata and excludes live sessions. The CLI intentionally omits stable archive-list, archive-search, and unified conversation-search commands. The UI does not surface metadata-orphaned conversations. Long-lived threads retain stale titles after their subject changes.

## Recurrence and incomplete prior hardening

Web Console documents metadata-loss incidents from 2026-05-01 and 2026-05-27. Earlier hardening stopped startup recovery from deleting some orphaned persistent sessions and added recovery commands. That work still assumes a session row survives. It cannot discover and repair a surviving agent home or conversation projection after the row disappears.

The product contract already promises durable session continuity under `OT-P0-003`, and the requirements registry says close must archive rather than permanently delete. The requirement remains in progress and its archive validations remain marked failing.

## Adjacent ownership

The active Plan Manager plan `agent-manager-searchable-conversation-history` already owns global conversation retrieval, Search Hub federation, privacy-aware indexing, and a governed recall program. Reimplementing global search inside Web Console would create a second authority and conflicting retention semantics. This plan must repair Web Console continuity and then integrate its durable aliases and sources into Agent Manager's capability.

