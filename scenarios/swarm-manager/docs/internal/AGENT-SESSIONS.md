# Agent Sessions

Agent Sessions are durable Swarm Manager-owned human conversations backed by Agent Manager runs. Programmatic prompt composition and typed result handling belong to declared Agent Manager workflows, not sessions.

The design conversation behind the current prompting, kind scoping, and starter-prompt model is recorded in [`SESSION-ARCHITECTURE-DESIGN-RECORD.md`](./SESSION-ARCHITECTURE-DESIGN-RECORD.md). Read it before changing any of the three.

## Invariants

Two rules govern every session, and both are stated in the universal band of every session prompt:

| Invariant | Meaning |
|---|---|
| **Propose, never apply** | A session agent may recommend any change. Swarm Manager applies it only after the operator accepts a typed proposal. The session profile withholds the file-write tools so the boundary is not instruction alone. |
| **Resolve in-session** | A session must reach its outcome while the operator is present — a proposal, a started transition, a design record, or a recorded reason to do nothing. A session must not resolve by routing its conclusion to an autonomous agent's inbox, a team heartbeat, or a queue that only a scheduled loop drains. |

In-session resolution is the durable difference between Prompt Manager's teams (autonomous, heartbeat-driven, deferred) and Swarm Manager's sessions (collaborative, operator-present, immediate). A session may *read* any corpus that helps it answer; it may not *hand off* its decision.

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

Session kinds are closed at the contract boundary. They divide on **subject**, and the division is a type distinction rather than a menu: two kinds operate on the work ledger, and the third operates on the machine that operates the ledger.

| Kind | Launcher label | Subject | Skill |
|---|---|---|---|
| `meta_orchestration` | Plan Work With Agent | The product — **grows** the ledger. Shapes raw operator material into goals, milestones, and backlog items. | `swarm-manager-meta-orchestrator` |
| `swarm_operations` | Manage Swarm | The product — **moves** the ledger. True state of existing work, what matters most next, and its registered transition. | `swarm-manager-operations-session` |
| `workflow_authoring` | Improve the System | **The machine, not the product.** Skills, prompts, workflows, transitions, briefs, session surfaces, and agent profiles. | `swarm-manager-workflow-authoring` |

The boundary test between the first and third: *if the change is about how the operator and agents work together, it is meta; if it is about what the tool does for its users, it is plan-work.* Apply it to the subject, not to the file that changes — a change to the graph workspace is product work, and a change to how a session is prompted is system work even though it ships as a React component.

**Naming.** `workflow_authoring` is the persisted wire value and stays that way; the display label, skill scope, and doctrine are *Improve the System*. The old name described a proper subset — workflow authoring is one disposition inside the kind, alongside skill changes, backlog proposals, and design records. Renaming the stored kind is a migration across proto contracts, stats aggregation, stored `session.json` files, and TypeScript unions; it is deliberately deferred. Precedent: `operating_mode_authoring` was retired by making it non-creatable while remaining readable for attribution.

Adding a kind should mean adding a skill mapping, a startup brief, allowed context types, a prompt subject band, allowed proposal kinds, tests, stats expectations, and docs. Do not add an untyped generic chat mode to bypass those contracts.

## Prompt Construction

The initial prompt is assembled in `api/internal/agentsessions/service_prompts.go` from sections registered in `prompt_sections.go`. Reference material is wrapped in one `<context>` block; the operator message stays outside it, so the model can tell material to consult from the job to do.

Sections are emitted on a strict volatility gradient. Each registered section declares a scope, and the emitter sorts by it:

| Scope | Varies by | Sections |
|---|---|---|
| `universal` | nothing — byte-identical for every session | `session-doctrine` |
| `kind` | session kind | `session-kind` |
| `job` | proposal target | `proposal-target` |
| `volatile` | session and turn | `session-identity`, `startup-brief`, `attached-context`, `attached-images` |
| `task` | every message; emitted outside `<context>` | `operator-message` |

**Why the order is load-bearing.** A provider caches a prompt prefix up to its first differing byte. If a volatile section moves above a stable one, the prefix collapses to nothing. The defect this replaced emitted `Session ID: sess_…` third, above every instruction, so no two sessions shared more than about forty bytes. Two sessions of one kind now share roughly 94% of the initial prompt. `prompt_structure_test.go` guards the ordering and the shared-prefix floor.

This mirrors `scenarios/prompt-manager/api/heartbeat/prompt_templates.go`, which solved the same problem first. Do not introduce a second prompt architecture; extend the registry. A section kind the registry does not name panics rather than emitting an unnamed block.

**The skill is fetched, not inlined.** The kind band names the authoritative Prompt Manager skill and instructs the agent to read it before answering. Inlining the skill text would make the largest stable block part of the cached prefix and would remove a tool round trip from turn one, but Swarm Manager has no client that reads skill *content* — `promptcatalog` carries metadata only, and session spawns pass a raw prompt string with no `promptRef` seam. Adding that client is a cross-scenario dependency with its own failure mode (Prompt Manager down means no session can start) and is deferred as an explicit decision, not an oversight.

### Prompt Preview

```text
POST /api/v1/agent-sessions/{session_id}/prompt-preview
swarm-manager sessions prompt-preview --id ID [--message TEXT] [--json]
```

Returns the prompt a message would produce, plus `initial` (whether the initial or continuation builder ran). It is read-only: no message is appended, no run is spawned, no session state changes.

Assembly is server-owned and the preview calls the same builders `Start` and `Continue` call. A client that reimplemented the section order or the volatility gradient would produce a preview that agrees with nothing. `TestPreviewPromptMatchesWhatStartWouldSend` asserts the preview is byte-identical to the prompt actually spawned.

For a draft session the preview applies the same auto-context policy as `Start`, so the startup brief and any proposal target appear exactly as they will be sent.

The session composer exposes it as **Preview prompt** (`SessionPromptPreview.tsx`), which renders the assembled text and never rebuilds it client-side.

**When changing the proto, regenerate through package governance, not `make generate` alone.** Scenario UIs sit outside the root pnpm workspace by design (each has a boundary `pnpm-workspace.yaml`; root sets `link-workspace-packages: false`), so they consume `@vrooli/proto-types` through a `file:` spec — which pnpm materialises as a *copy* in `ui/node_modules/.pnpm`, not a symlink. Regenerating alone updates `packages/proto/gen/typescript` and never reaches the consumer, and restarting the scenario does not help because the staleness is on disk rather than in a process. The propagating command is:

```text
vrooli package refresh proto [<consumer>] [--no-restart]
```

Its manifest declares `"refresh": {"strategy": "generate_then_setup", "restart_running_consumers": true}` — generate, then run consumer setup, then restart. Pass a consumer name to scope it to one scenario instead of all 61 UI consumers.

## Agent Profile

Sessions run on `swarm-manager/session` (`.vrooli/agent-manager/session.json`), declared in `.vrooli/service.json` and listed in the API's required profile keys so a missing reconciliation fails at startup rather than at first message.

```json
"allowedTools": ["read", "glob", "grep", "shell", "web_search", "web_fetch"]
```

Two deliberate differences from the shared `swarm-manager/default` execute profile:

- **Web research is granted.** A conversation whose job is helping decide what to build must be able to look outside the repository.
- **`write` and `edit` are withheld**, so propose-never-apply is narrowed by capability rather than resting on instruction alone. This is a narrowing, not a hard boundary: `shell` is required for the `swarm-manager`, `prompt-manager`, and `search-hub` CLIs, and a shell can write files. A true propose-only boundary needs shell command restriction, which is a separate design question.

The key is read from `SWARM_MANAGER_SESSION_PROFILE_KEY` rather than the shared `AGENT_MANAGER_PROFILE_KEY`, so overriding the execute profile cannot silently re-grant write access to every conversation.

## Starter Prompts

A starter card carries two strings, defined in `ui/src/components/session/session-starter-suggestions.ts`:

| Field | Job |
|---|---|
| `label` | Menu text. Terse and scannable, read while choosing, never sent. |
| `prompt` | Composer seed. Prose in the operator's voice that states the situation, states the intent, names the shape of answer wanted, and — where the card needs the operator's own material — ends with an invitation and a blank line for it. |

These are incompatible jobs and one string cannot do both. When the label doubled as the prompt, `"Turn this idea into goals and backlog items."` was sent as a complete message with no idea attached and no signal that the operator was expected to supply one.

Rules enforced by `session-starter-suggestions.test.ts`:

- Every card defines both, and never reuses one as the other.
- Labels stay under 80 characters; prompts exceed 120, because a one-liner cannot state situation, intent, and desired output.
- A prompt may say "the attached …" only when the card has a non-optional requirement, because a card with only optional requirements seeds its prompt immediately and would be referring to nothing.
- A prompt ending in an invitation must end with `:\n\n`, so the slot is where the operator types.
- Send-ready cards must not trail off in a colon.

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
- Register a new prompt section in `prompt_sections.go` with its volatility scope. Never emit a section the registry does not name, and never place a volatile section above a stable one.
- Keep the skill as the home of methodology and the startup brief as the home of current state. A brief that carries procedure, or a prompt that restates the skill, splits the attention budget and drifts.
- When adding a starter card, write `label` and `prompt` as two separate strings. The tests reject a prompt that is a label in disguise.
