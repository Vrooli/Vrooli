# Web Console Unlosable Conversations — Authoring Evidence

## Purpose

This directory preserves the evidence and design context used to author the
Plan Manager plan `web-console-unlosable-conversations`. The structured Plan
Manager record is authoritative for execution. These files preserve incident
facts, reproducible diagnostic queries, and source references that should not
depend on the original chat remaining available.

## Triggering incident

On 2026-09-04, the operator closed a Web Console pane after Codex appeared to
freeze. The operator later remembered a recent discussion about Plan Manager,
Agent Manager, Git Control Tower, Test Genie, skills, and Program Runtime, but
could not find the conversation in the Web Console archive.

The conversation had not created a plan. Its current topic did not match its
historical title because an older onboarding thread had been resumed and later
changed subjects. Initial searches therefore found unrelated plans and global
Codex sessions.

## Recovered identities

| Identity | Value |
| --- | --- |
| Web Console pane/session | `a7e71c3c-e422-4c89-916a-03f92906fb89` |
| Codex thread | `01a06a6b-88da-7422-b391-bb59c5f5e5e0` |
| Last recovered assistant message | `2026-09-04T17:42:19.025018486Z` |
| Conversation event count | `88` |
| Archive-loss bug | `knw-1788545141995388770` |
| Debugging-skill command drift bug | `knw-1788545581902820117` |
| Investigation work record | `eef203d5-7500-4504-aec8-2c6e52149fd8` |

## Preserved local artifacts

- Raw Codex rollout:
  `/home/matthalloran8/.vrooli/state/vrooli/web-console/sessions/codex/a7e71c3c-e422-4c89-916a-03f92906fb89/sessions/2026/09/03/rollout-2026-09-03T23-17-07-01a06a6b-88da-7422-b391-bb59c5f5e5e0.jsonl`
- Pane-specific Codex home:
  `/home/matthalloran8/.vrooli/state/vrooli/web-console/sessions/codex/a7e71c3c-e422-4c89-916a-03f92906fb89/`
- Web Console SQLite database:
  `/home/matthalloran8/.vrooli/data/vrooli/web-console/web-console.db`
- Existing recovery guide:
  `/home/matthalloran8/Vrooli/scenarios/web-console/docs/guides/SESSION_RECOVERY.md`
- Existing recovery-hardening plan:
  `/home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/plans/persistent-session-recovery-hardening-plan.md`
- Related active cross-run search plan:
  `/home/matthalloran8/.vrooli/plans/agent-manager-searchable-conversation-history.md`

The raw rollout is intentionally not copied into Git. It is approximately
26 MiB, contains the complete agent conversation and tool activity, and may
contain sensitive material. The plan must preserve it during reconciliation.

## Confirmed incident facts

The database retained the following rows for the missing pane:

- `conversation_sessions`: present, `last_sequence=88`.
- `conversation_events`: 88 rows, including the final response.
- `agent_transcript_checkpoints`: present, with the exact rollout path.
- `workspace_panes`: present.
- `sessions`: absent.

The archive and archived-message search require a matching `sessions` row.
The missing metadata row therefore made intact data undiscoverable.

The event buffer retained terminal connection and pane presentation events.
It did not retain a typed, durable lifecycle receipt that identifies which
actor or path removed the session metadata.

## Live integrity snapshot

The following snapshot was measured on 2026-09-04 during plan authoring:

| Relation | Rows |
| --- | ---: |
| `sessions` | 332 |
| archived `sessions` | 0 |
| `conversation_sessions` | 421 |
| `workspace_panes` | 1421 |
| `agent_transcript_checkpoints` | 330 |
| conversations without a session row | 116 |
| workspace panes without a session row | 1342 |
| transcript checkpoints without a session row | 97 |

These counts are evidence of structural drift, not a deletion list. The
implementation must classify every orphan before mutation. It must preserve
message-bearing and agent-history-bearing records by default.

## Current architectural findings

1. Session metadata, conversation events, workspace layout, transcript
   checkpoints, and agent homes use related string identifiers without one
   canonical catalog or complete reconciliation loop.
2. Multiple layers can delete session metadata. Process exit, explicit delete,
   expiration, cleanup, and historical recovery paths do not share one typed
   lifecycle authority.
3. Archive is a multi-step temporal flow. It marks metadata, returns success,
   waits through an undo interval, and later terminates the process.
4. Generic process-exit cleanup does not carry the initiating lifecycle intent.
5. Several persistence operations discard errors.
6. Archive tests verify the pre-finalization undo window but do not fully prove
   post-finalization persistence for every backend and recovery state.
7. Archive search uses an inner join to `sessions`. Missing metadata suppresses
   evidence rather than producing a recoverable result.
8. Per-pane `CODEX_HOME` isolation is valuable but not discoverable outside the
   Web Console catalog.
9. Web Console omits stable CLI commands for archive listing, archive search,
   and unified conversation discovery.
10. Historical pane titles do not represent later topic changes in long-lived
    agent sessions.

## Ownership decision

Web Console must own session continuity, local transcript attribution, archive,
recovery, and orphan reconciliation. Agent Manager must own cross-run search
and conversation recall. Search Hub must own federation. The plan must consume
the active `agent-manager-searchable-conversation-history` plan instead of
creating a second global search authority inside Web Console.

## Prior incidents

Web Console already documents metadata-loss incidents from 2026-05-01 and
2026-05-27. In those incidents, recovery or storage-path behavior removed the
only database pointer to surviving agent histories. The current incident is a
recurrence of the same failure class: durable evidence survives while its
catalog identity disappears.

## Evidence limitations

The exact function that removed the triggering `sessions` row cannot be proven
from current telemetry. There is no durable lifecycle ledger, the in-memory
event buffer does not contain a deletion receipt, and some store errors are
discarded. The plan must fix this inability to attribute future transitions.

