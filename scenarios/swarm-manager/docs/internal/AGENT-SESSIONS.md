# Agent Sessions

Agent Sessions are durable Swarm Manager-owned conversations with Agent Manager runs. They are used for workflows that need more context and iteration than Quick Capture, such as meta-orchestration and operating-mode authoring.

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

## Supported Kinds

Initial session kinds are closed at the contract boundary:

| Kind | Skill | Purpose |
|---|---|---|
| `meta_orchestration` | `swarm-manager-meta-orchestrator` | Conversational planning that can propose multiple initiatives and backlog items in one audited apply action. |
| `operating_mode_authoring` | `swarm-manager-operating-mode-authoring` | Conversational operating-mode proposal work, followed by proposal-backed implementation planning. |

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
| `backlog_batch_import` | Uses the backlog batch applier to create or update initiatives and create multiple backlog items in one audited action. |
| `operating_mode_draft` | Records an accepted operating-mode proposal artifact linked to the session. It does not edit operating-mode code. |
| `operating_mode_implementation_plan` | Extracts a backlog batch payload from `items` or `backlog_batch_import` and creates implementation work through the same audited batch applier. |

Proposal apply never lets a session agent directly mutate project-management files or operating-mode registry code from the chat flow. The session can propose; Swarm Manager applies.

## Artifacts

Artifacts are first-class link records in `artifacts.jsonl`. They connect a session to entities or files that were proposed, created, updated, deleted, or linked.

Artifact records include:

- `session_id`
- `artifact_type`
- `action`
- `entity_ref`
- optional `proposal_id`, `activity_id`, and `run_id`
- `mutation_source`
- verified attribution
- `created_at`

Backlog and initiative detail views should use persisted attribution and artifact lookup endpoints instead of scraping event logs. Event logs are for metrics and chronology; artifacts are the navigable audit model.

## UI Entry Points

The graph bottom action launcher owns session creation:

- Quick Capture opens the existing one-shot capture panel.
- Plan Work With Agent creates a `meta_orchestration` session.
- Author Operating Mode creates an `operating_mode_authoring` session.

Both agent-session launchers create a draft and route to the session detail surface immediately. They do not send canned bootstrap prompts. The composer placeholder is kind-specific, and the first submitted message starts the run.

The graph sidebar owns session history through the `Sessions` tab. Selecting a session opens the session detail panel rather than navigating to a dedicated Sessions page.

Entity attribution chips can reopen the related session when `created_by.session_id` is present. Non-session provenance should fall back to agent or operator display text.

## Storage

Sessions are stored below the scenario root:

```text
agent-sessions/
  sess_<id>/
    session.json
    messages.jsonl
    proposals/
      <proposal_id>.json
    artifacts.jsonl
    attachments/
```

`session.json` is the indexable snapshot. Messages and artifacts are append-only JSONL so long-running conversations keep an auditable transcript.

## Maintenance Notes

- Keep session-owned Agent Manager work under `agentactivity.OwnerSession`.
- Keep mutation attribution service-owned and API-owned.
- Add proposal kinds only when they have a typed validation and apply policy.
- Update `SEAMS.md`, stats tests, UI contract mappers, and this document when adding a new session kind or artifact type.
- Prefer extending shared proposal/apply seams over introducing mode-specific UI or handler branches.
