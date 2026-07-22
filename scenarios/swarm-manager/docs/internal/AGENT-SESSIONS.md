# Agent Sessions

Agent Sessions are durable Swarm Manager-owned human conversations backed by Agent Manager runs. Programmatic prompt composition and typed result handling belong to declared Agent Manager workflows, not sessions.

## Lifecycle

A session starts from the graph action launcher or from a session-aware internal flow as a pre-spawn draft. The API creates a `sess_*` record under `agent-sessions/` with no messages and no Agent Manager run. The first real operator message is sent to the explicit start route, appended to `messages.jsonl`, used as the initial prompt, and only then does Swarm Manager spawn Agent Manager and record the returned run and task IDs.

Supported statuses:

| Status | Meaning |
|---|---|
| `draft` | Session exists, has not spawned Agent Manager, and is waiting for the operator's first real message. |
| `starting` | Session exists and the initial Agent Manager run is being created. |
| `running` | The active run is processing a user message. |
| `waiting_for_user` | The run has yielded control and the operator can continue the conversation. |
| `proposal_ready` | At least one proposal is ready for explicit operator action. |
| `applying` | Swarm Manager is applying an approved proposal through API-owned mutation seams. |
| `complete` | The session run completed. This does not imply every proposed project artifact is complete. |
| `failed` | Spawn, continuation, refresh, or proposal apply failed. |
| `canceled` | The operator canceled the session. |

Refresh is a no-op for draft/no-run sessions. After start, refresh maps Agent Manager run state into the session lifecycle and appends the run summary as an assistant message when the summary is new.

## Run Events

Session details read run events through Swarm Manager, not directly from Agent Manager:

```text
GET /api/v1/agent-sessions/{session_id}/events?after_sequence=<n>&limit=<n>
```

Draft sessions and other no-run sessions return an empty event list. Active sessions proxy the session-owned Agent Manager run events and expose a bounded timeline shape for messages, tool calls, tool results, status/progress, errors, compaction, and a safe fallback for unknown event types. Large tool payload fields are truncated at the Swarm Manager API boundary.

## Message Context And Attachments

Session context is attached at message time from the session detail composer.
The picker is not a pre-session dialog: the same draft session can start with
plain text, images, typed context refs, or any combination of the three, and
follow-up messages use the same composer controls.

Entity detail pages can also stage typed context into an existing or new draft
session through `Attach to session`. This is the inverse entry point for the
same composer-scoped model: selecting a session stores the current entity as
pending composer context for that session and routes to the session detail page.
The operator still sends the message or context-only start/continue request.
No server-side session context inventory is created.

Supported context ref types are closed at the API boundary:

| Type | Purpose |
|---|---|
| `backlog_item` | Attach an existing backlog item summary and planning state. |
| `milestone` | Attach milestone metadata and rollup context. |
| `capture` | Attach a captured note or classified input. |
| `execution` | Attach an execution-control record. |
| `agent_activity` | Attach tracked Agent Manager activity. |
| `scenario` | Attach scenario status and metadata. |
| `session` | Attach another Agent Session summary for continuity. |
| `startup_brief` | Attach the kind-specific startup packet used to answer broad first prompts quickly. |
| `operations_briefing` | Attach the current operations briefing directly for operations drill-downs. |

The UI sends refs as `{type, id}` values. The API resolves those refs into
bounded `AgentSessionContextItem` snapshots before appending the operator
message and before building the Agent Manager prompt. Agents receive the
resolved context, not raw UI store records.

Draft sessions can fetch a startup brief before the first message without
starting an Agent Manager run:

```text
GET  /api/v1/agent-sessions/{session_id}/startup-brief
POST /api/v1/agent-sessions/{session_id}/startup-brief
```

The resolved brief is attached as `startup_brief/<kind>` by default on start
unless the request sets `"auto_context_policy": "none"` or supplies an
equivalent explicit startup context. This is the fast path for broad prompts
such as current status, existing context inspection, and mode classification.

Image attachments are uploaded to the session before the message is sent:

```text
POST /api/v1/agent-sessions/{session_id}/attachments
GET  /api/v1/agent-sessions/{session_id}/attachments/{attachment_id}
```

Only image uploads are accepted for session messages. Attachment files are
stored below the owning session folder, referenced from `session.json`, and
linked to individual messages by attachment ID. Deleting a session removes its
attachments with the rest of the session-owned storage.

## Supported Kinds

Initial session kinds are closed at the contract boundary:

| Kind | Skill | Purpose |
|---|---|---|
| `meta_orchestration` | `swarm-manager-meta-orchestrator` | Conversational planning that can propose multiple milestones and backlog items in one audited apply action. |
| `swarm_operations` | `swarm-manager-operations-session` | Conversational operations coordination for milestone progress, pending decisions, and run review. It routes decision draining to `workshop-decision-sync` and keeps mutations operator-gated. |
| `workflow_authoring` | `swarm-manager-workflow-authoring` | Conversational design of reviewed workflow and transition changes. It distinguishes existing transition improvements, new declared workflows, and required Swarm domain backlog work; it never silently applies a declaration. |

Adding a kind should mean adding a skill mapping, prompt builder behavior if needed, allowed proposal kinds, tests, stats expectations, and docs. Do not add an untyped generic chat mode to bypass those contracts.

## Identity And Attribution

Agent identity and session identity are separate:

- Agent Manager injects `VROOLI_AGENT_IDENTITY_TOKEN` into spawned runs. It does not inject Swarm Manager, Agent Manager, Prompt Manager, or Workspace Sandbox API-base variables.
- The Swarm Manager CLI forwards the verified token to the API as `X-Agent-Identity-Token`.
- Swarm Manager API middleware verifies the token through Agent Manager discovery and builds `identity.Provenance`.
- Session middleware resolves `provenance.run_id` back to the owning session and enriches provenance with `session_id`, `session_kind`, and `source`.

Session spawns also receive Swarm-owned observability environment variables:

| Variable | Value |
|---|---|
| `VROOLI_SWARM_MANAGER_SESSION_ID` | The `sess_*` ID. |
| `VROOLI_SWARM_MANAGER_SESSION_KIND` | The typed session kind. |
| `VROOLI_SPAWN_SOURCE` | `session/<session_id>`. |

Scenario CLI API location remains owned by the CLI lifecycle discovery chain (`--api-base`, saved config, scenario-specific env, `vrooli scenario port <scenario> API_PORT`). Session env describes the session; it is not a service-location channel.

Agents should not manually pass run IDs, session IDs, attribution fields, or created-by data in normal Swarm Manager CLI commands. Attribution is owned by the API and derived from verified identity plus the run-to-session resolver.

## Proposals

Proposals are machine-validated JSON payloads plus a human-readable summary. A proposal can be drafted by an agent, but applying it is an explicit Swarm Manager API action.

Supported proposal kinds:

| Kind | Apply behavior |
|---|---|
| `backlog_batch_import` | Uses the backlog batch applier to create or update milestones and create multiple backlog items in one audited action. |
| `mutation_list` | A proposal-session change set. It is decided through the proposal-session decision flow, rather than the generic session apply route. |

Proposal apply never lets a session agent directly mutate project-management files from the chat flow. The session can propose; Swarm Manager applies. Proposal kinds and apply behavior are server-owned; response clients must render an unfamiliar future kind generically instead of rejecting the complete session response.

## Artifacts

Artifacts are first-class session handoff records. They connect a session to
entities or files that were proposed, created, updated, deleted, or linked.
They are retained separately from Vrooli Events receipts: an artifact is domain
review material, while a receipt is an independently observed operation fact.

Artifact records include:

- `session_id`
- `artifact_type`
- `action`
- `entity_ref`
- optional `proposal_id`, `activity_id`, and `run_id`
- `mutation_source`
- verified attribution
- `created_at`

Backlog and milestone detail views use persisted session attribution and
artifact lookup endpoints instead of scraping event logs. Event logs remain for
metrics and chronology. New artifact links are durably appended to the
session's `artifacts.jsonl`; this preserves non-receipt review handoffs without
recreating a cross-scenario evidence ledger.

## UI Entry Points

The graph bottom action launcher owns session creation:

- Quick Capture opens the existing one-shot capture panel.
- Plan Work With Agent creates a `meta_orchestration` session.
- Manage Swarm creates a `swarm_operations` session.
- Author Workflow creates a `workflow_authoring` session.

All agent-session launchers create a draft and route to the session detail surface immediately. They do not send canned bootstrap prompts. The composer placeholder is kind-specific, and the first submitted message starts the run.

The session detail composer reuses the shared message composer used by Quick
Capture for text entry, image previews, keyboard submit behavior, and attachment
deduplication. Session details adds the context picker button and pending
context chip tray on top of the shared composer so first messages and follow-up
messages behave consistently.

The graph sidebar owns session history through the `Sessions` tab. Selecting a session opens the session detail panel rather than navigating to a dedicated Sessions page.

Routed entity detail pages for backlog items, milestones, captures,
executions, scenarios, and sessions expose `Attach to session`.
The action filters target sessions by the same context-type policy used by the
composer picker, excludes the source session when attaching a session, and can
quick-start a compatible draft session with the entity staged in its composer.

Entity attribution chips can reopen the related session when `created_by.session_id` is present. Non-session provenance should fall back to agent or operator display text.

## Storage

Sessions are stored in the scenario data root, never below the scenario source root:

```text
~/.vrooli/data/vrooli/swarm-manager/agent-sessions/
  sess_<id>/
    session.json
    messages.jsonl
    proposals/
      <proposal_id>.json
    attachments/
      <attachment_id>/
        <safe_filename>
```

`session.json` is the indexable snapshot and `messages.jsonl` is the
conversation transcript. Session artifact views are projected from the
canonical evidence ledger. Historical `artifacts.jsonl` files are import-only
migration input and are not part of the active storage contract.

On first startup after this storage migration, the API copies any legacy
`scenarios/swarm-manager/agent-sessions/` tree into the data root without
overwriting or deleting the source copy. It records a migration marker and
refuses an ambiguous merge into a non-empty destination; source-tree cleanup
is only safe after backup/restore and API-read verification.

## Maintenance Notes

- Keep session-owned Agent Manager work under `agentactivity.OwnerSession`.
- Keep mutation attribution service-owned and API-owned.
- Add proposal kinds only when they have a typed validation and apply policy.
- Keep `swarm_operations` advisory in v1: use existing UI/API/CLI flows for state changes, and add typed proposal kinds only after review/apply semantics are designed.
- Update `SEAMS.md`, stats tests, UI contract mappers, and this document when adding a new session kind or artifact type.
- Prefer extending shared proposal/apply seams over introducing mode-specific UI or handler branches.
