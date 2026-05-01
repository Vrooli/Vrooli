# Swarm Manager Native Agent Sessions Implementation Plan

## Purpose

Add first-class conversational agent sessions to Swarm Manager so operators can start, resume, inspect, and audit longer-running planning and authoring conversations directly from the UI. Sessions must support meta-orchestration, operating-mode authoring, and future conversational workflows without requiring agents to manually pass identity or attribution data around.

This is a greenfield hard cutover plan. The final implementation must be production-ready, well-tested, and free of legacy compatibility branches, dead code, or temporary UI-only attribution.

## Required Reading

- `prompt-manager skill read implementation-plan-authoring documentation-health react-coherence seam-discovery-and-enforcement test utils-unification cli-steer interoperability-steer`
- `scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md`
- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/docs/internal/OPERATING-MODE-AUTHORING.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-meta-orchestrator/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-operating-mode-authoring/SKILL.md`
- `scenarios/swarm-manager/research/agent-identity-standard/conclusion.md`
- `packages/cli-core/cliutil/identity.go`
- `scenarios/agent-manager/api/internal/orchestration/phases/env.go`
- `scenarios/agent-manager/api/internal/orchestration/phases/identity.go`
- `scenarios/agent-manager/api/internal/orchestration/service.go`
- `scenarios/agent-manager/api/internal/handlers/handlers.go`
- `scenarios/swarm-manager/cli/app.go`
- `scenarios/swarm-manager/cli/identity_transport.go`
- `scenarios/swarm-manager/api/internal/identity/middleware.go`
- `scenarios/swarm-manager/api/internal/identity/provenance.go`
- `scenarios/swarm-manager/api/internal/agentactivity/types.go`
- `scenarios/swarm-manager/api/internal/backlog/service.go`
- `scenarios/swarm-manager/api/internal/backlog/handler_create.go`
- `scenarios/swarm-manager/api/internal/backlog/batch_handler.go`
- `scenarios/swarm-manager/api/internal/initiatives/model.go`
- `scenarios/swarm-manager/api/internal/initiatives/service.go`
- `scenarios/swarm-manager/api/internal/captures/handler.go`
- `scenarios/swarm-manager/api/internal/captures/classify.go`
- `scenarios/swarm-manager/api/internal/backlog/clarification.go`
- `scenarios/swarm-manager/api/internal/backlog/clarification_service.go`
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/GraphWorkspace.tsx`
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/CapturePanel.tsx`
- `scenarios/swarm-manager/ui/src/components/capture/quick-capture-input.tsx`
- `scenarios/swarm-manager/ui/src/components/backlog/clarification-panel.tsx`
- `scenarios/swarm-manager/ui/src/hooks/useClarificationThread.ts`
- `scenarios/swarm-manager/ui/src/stores/clarification-store.ts`
- `scenarios/swarm-manager/ui/src/components/initiative/feedback-dialog.tsx`
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/sidebar/Sidebar.tsx`
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/sidebar/types.ts`
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/sidebar/useSidebarState.ts`
- `scenarios/swarm-manager/ui/src/services/agent-activity-service.ts`
- `scenarios/swarm-manager/ui/src/lib/api-endpoints.ts`
- `packages/proto/schemas/swarm-manager/v1/domain/backlog.proto`
- `packages/proto/schemas/swarm-manager/v1/domain/initiative.proto`

## Problem Statement

Swarm Manager currently has three adjacent concepts that do not form a complete native session system:

1. Quick Capture is a one-shot classification flow. It can turn operator input into backlog suggestions, but it does not support long-lived conversation, resumption, proposal refinement, or artifact-level audit navigation.
2. Clarification chats prove that a multi-turn Agent Manager conversation can work in the UI, but the backend/domain is tied to backlog workshop decision questions and cannot represent general planning or authoring sessions.
3. Meta-orchestration and operating-mode authoring already exist as Prompt Manager skills, but operators currently have to launch coding agents manually and pass skill paths or instructions outside Swarm Manager.

The missing product layer is a durable "agent session" domain: a session can be opened from Swarm Manager, can continue across UI reloads, can produce proposals and artifacts, can apply multi-entity changes through Swarm Manager APIs, and can explain exactly which Agent Manager run created or changed each initiative, backlog item, or operating-mode artifact.

## Scope

In scope:

- A first-class agent session API, storage model, proto contract, service layer, and UI service.
- A Sessions tab in the existing graph sidebar. There must be no dedicated full page for session history.
- A session detail panel that can reopen/resume sessions, continue chat, show Agent Manager run state, show proposals, show applied artifacts, and navigate to related entities.
- A launcher UX from the current graph bottom action area that offers Quick Capture, Meta-Orchestration Session, and Operating-Mode Authoring Session.
- Native meta-orchestration sessions that use the existing `swarm-manager-meta-orchestrator` skill and can preview/apply multiple initiatives and backlog items in one apply action.
- Native operating-mode authoring sessions that use the existing `swarm-manager-operating-mode-authoring` skill and create proposal drafts before implementation.
- Automatic attribution for backlog, initiative, and operating-mode artifacts created or changed by session agents.
- Metrics/events for session usage, session outcomes, proposal/application rates, and artifact creation by session kind.
- Documentation for the identity, session, proposal, artifact, and attribution contracts.
- Unit, integration, contract, UI, and scenario validation tests.

Out of scope:

- Runtime plugin loading for operating modes.
- Data-file-only dynamic operating-mode creation while Swarm Manager is running.
- A separate Sessions route/page.
- Generic arbitrary chat without a typed session kind.
- Bypassing Swarm Manager API mutation endpoints from agents.
- Requiring agents to manually pass their run identity, session identity, or created-by data in normal CLI commands.
- Keeping duplicate old and new session implementations after cutover.

## Current Technical Context

### Quick Capture

`GraphWorkspace.tsx` currently renders a single bottom action labeled "New capture" that toggles `CapturePanel`. `CapturePanel` is a `FloatingPanel` wrapper around `QuickCaptureInput`. The capture backend stores files under `captures/{id}` and spawns a classification agent through `captures/classify.go`.

This path is intentionally simple and one-shot. It is useful as one launcher option, but it should not be stretched into a general chat/session abstraction.

### Clarification Chat

`ClarificationPanel`, `useClarificationThread`, and `clarification-store` already implement a reusable UI pattern: floating desktop panel, mobile bottom sheet, message history, attachments, send/continue, polling, stale run handling, and action buttons. The backend uses `ContinueRun` against Agent Manager.

The existing clarification backend is domain-specific to backlog workshop questions. Reuse the UI and hook patterns, but build a new session backend instead of generalizing the clarification domain in place.

### Initiative Feedback and Meta-Orchestration

`feedback-dialog.tsx` and initiative feedback services provide preview/decision patterns that are relevant for proposal-first flows. The meta-orchestrator skill already describes a conversational workflow that previews backlog imports before applying them. The batch import seam is `api/internal/backlog/batch_handler.go`; it supports multiple backlog items and initiative metadata in one request, including preview mode.

The current batch path does not stamp `CreatedBy` on batch items from request provenance, and initiative create/update calls do not carry a durable attribution field. This is a key gap for session artifact attribution.

### Operating Mode Authoring

Operating modes are static Go definitions in `api/internal/operatingmode`. The authoring skill intentionally avoids runtime plugins and requires proposal-first authoring. The current initiative proto still hard-codes the mode string constraint to `item-level`, `holistic-loop`, and `phased-plan-drain`; that friction should be addressed as part of making authoring maintainable.

### Agent Identity Standard

Agent Manager already injects `VROOLI_AGENT_IDENTITY_TOKEN` into runner processes via `IdentityEnvVars`. Tokens are verified through `POST /api/v1/identity/verify`. `packages/cli-core/cliutil/identity.go` detects and verifies the token using `VROOLI_AGENT_MANAGER_API_BASE`.

Swarm Manager CLI already detects the token in `NewApp` and injects `X-Agent-Identity-Token` into every outgoing API request. Swarm Manager API registers `identity.Middleware(identity.CLIUtilVerifier{})`, verifies the token, and stores `identity.Provenance` in request context. `agentactivity.Spec.normalized` uses that provenance for `RequestedBy` when a request comes from an agent.

This is the right foundation. The session implementation should use it rather than inventing a parallel identity mechanism.

### Current Attribution Gaps

- Single backlog create sets `BacklogItem.CreatedBy` from `identity.FromContext`.
- Batch backlog create currently builds `BacklogItem` values without `CreatedBy`.
- Initiative model has no `created_by`, `updated_by`, or artifact attribution field.
- Eventlog actor attribution exists for several paths, but it is not a durable artifact-link contract that the UI can navigate.
- `spawned_from` is a weak origin string for backlog items. It is useful but insufficient for "created by session X/run Y" UX.
- UI agent activity filters do not include `ownerType: "initiative"` even though backend supports it.

## Target End State

Swarm Manager has a native Agent Sessions surface:

- The graph sidebar has a `Sessions` tab alongside Activity, Backlog, Captures, Initiatives, and Executions.
- Sessions are sortable/filterable by recency, status, kind, and artifact count.
- Selecting a session opens a session detail panel. The panel shows:
  - session title, kind, status, started/updated times, owning skill, Agent Manager run ID, task ID, profile key, and current run status;
  - the conversation transcript and attachments;
  - proposal drafts produced by the agent;
  - apply actions and their results;
  - artifacts created, updated, deleted, or proposed by the session;
  - links to related initiatives, backlog items, operating mode proposals, captures, and Agent Manager activity records.
- Backlog and initiative detail views show attribution chips for created/updated artifacts. A chip such as `Created by session: Meta-orchestration - API quality gates` opens the session detail panel.
- The existing graph bottom action becomes a compact launcher. It offers:
  - Quick Capture;
  - Plan Work With Agent;
  - Author Operating Mode.
- Session agents can use the existing Swarm Manager CLI normally. They do not manually pass identity flags or attribution fields for routine commands.
- A meta-orchestration session can apply multiple initiatives and multiple backlog items in one proposal/application action.
- An operating-mode authoring session can draft an operating-mode proposal from the UI, and only after explicit user acceptance can it create implementation work.

## Contract Decisions

### D1: `AgentSession` is a new domain, not a clarification subtype

Create a dedicated session domain under Swarm Manager. Clarification remains workshop-specific. Shared UI components can be extracted, but the storage and API contracts must be separate.

### D2: Session kind is explicit and closed at the contract boundary

Initial kinds:

- `meta_orchestration`
- `operating_mode_authoring`

The code should make adding a kind straightforward, but this plan does not introduce untyped arbitrary chat sessions. Each kind has a skill, prompt builder, allowed proposal kinds, and apply policy.

### D3: Agent identity is automatic and verified through the existing standard

The session agent process receives Agent Manager identity through `VROOLI_AGENT_IDENTITY_TOKEN`. When it runs `swarm-manager` CLI commands, the CLI forwards the token as `X-Agent-Identity-Token`. The API verifies it and gets `run_id`, `task_id`, and `profile_key`.

Do not ask the agent to include identity in prompts, JSON, or CLI arguments.

### D4: Session identity is separate from agent identity

Agent identity answers "which Agent Manager run made this API request?" Session identity answers "which Swarm Manager session owns this conversational workflow?"

Agent identity comes from the token. Session identity is attached by Swarm Manager in two ways:

- The session service records the Agent Manager `run_id` returned when it spawns the session agent.
- Swarm Manager sets session context environment variables for observability and source grouping, such as:
  - `VROOLI_SWARM_MANAGER_SESSION_ID`
  - `VROOLI_SWARM_MANAGER_SESSION_KIND`
  - `VROOLI_SPAWN_SOURCE=session/<session_id>`

The authoritative attribution join is `request provenance run_id -> agent session run_id`, not agent-supplied metadata.

### D5: Artifacts are first-class link records

Create a durable artifact attribution model rather than embedding ad hoc strings in every UI component. A session artifact record should include:

- artifact ID;
- session ID;
- artifact type: `backlog_item`, `initiative`, `operating_mode_proposal`, `operating_mode_definition`, `capture`, `file`, `agent_activity`;
- action: `proposed`, `created`, `updated`, `deleted`, `linked`;
- entity reference fields: backlog kind/name, initiative name, file path, mode ID, activity ID, run ID as applicable;
- agent provenance: run ID, task ID, profile key;
- API mutation source: endpoint or service entrypoint;
- timestamps.

### D6: Entity attribution is queryable from entity details

Backlog items and initiatives need enough persisted metadata for detail views to ask "what session created or changed this?" without scraping event logs. Use a common provenance shape in proto/domain JSON, backed by artifact records where possible.

Recommended shape:

```json
{
  "created_by": {
    "type": "agent",
    "run_id": "...",
    "task_id": "...",
    "profile_key": "swarm-manager/default",
    "session_id": "sess_...",
    "session_kind": "meta_orchestration"
  }
}
```

For backlog items, extend the existing `CreatedBy` provenance rather than adding a second unrelated field. For initiatives, add the same provenance shape.

### D7: Proposal apply is the only multi-entity write path for sessions

Session agents can generate proposals, but applying a proposal happens through Swarm Manager API. The UI/operator approves the proposal, and the API applies it atomically where feasible. For meta-orchestration, use and harden the existing batch backlog/initiative seam. For operating-mode authoring, create an explicit proposal/apply seam rather than having the session agent directly edit registry files from the chat flow.

### D8: Metrics come from events, not UI state

Emit session lifecycle, proposal, apply, artifact, and failure events into the existing eventlog/stats pipeline. The stats API should expose session adoption and effectiveness. The UI consumes stats from the stats service rather than recomputing from session lists.

## Data and API Design

### Proto Additions

Add `packages/proto/schemas/swarm-manager/v1/domain/agent_session.proto`:

- `AgentSession`
- `AgentSessionKind`
- `AgentSessionStatus`
- `AgentSessionMessage`
- `AgentSessionAttachment`
- `AgentSessionProposal`
- `AgentSessionProposalStatus`
- `AgentSessionArtifact`
- `AgentSessionAttribution`

Add `packages/proto/schemas/swarm-manager/v1/api/agent_session.proto`:

- `ListAgentSessionsRequest/Response`
- `GetAgentSessionRequest/Response`
- `CreateAgentSessionRequest/Response`
- `ContinueAgentSessionRequest/Response`
- `RefreshAgentSessionRequest/Response`
- `CancelAgentSessionRequest/Response`
- `ApplyAgentSessionProposalRequest/Response`
- `ListAgentSessionArtifactsRequest/Response`

Update backlog and initiative domain protos with a shared attribution/provenance message. If proto package organization makes a shared file awkward, define it in `domain/agent_session.proto` and import it.

### Backend Routes

Add a new `api/internal/agentsessions` package with routes:

- `GET /api/v1/agent-sessions`
- `POST /api/v1/agent-sessions`
- `GET /api/v1/agent-sessions/{session_id}`
- `POST /api/v1/agent-sessions/{session_id}/continue`
- `POST /api/v1/agent-sessions/{session_id}/refresh`
- `POST /api/v1/agent-sessions/{session_id}/cancel`
- `POST /api/v1/agent-sessions/{session_id}/proposals/{proposal_id}/apply`
- `GET /api/v1/agent-sessions/{session_id}/artifacts`
- `GET /api/v1/artifacts/by-entity?type=...&ref=...`

Do not mount routes under captures, backlog, or initiatives. Sessions are their own domain.

### Storage

Store sessions under the Swarm Manager scenario root in a dedicated folder:

```text
agent-sessions/
  sess_<id>/
    session.json
    messages.jsonl
    proposals/
      <proposal_id>.json
    artifacts.jsonl
    attachments/
      <attachment_id>/
        metadata.json
        content
```

Use append-only JSONL for messages and artifacts so long conversations and artifact histories are easy to stream and audit. Keep `session.json` as the indexable snapshot.

### Session Statuses

Use explicit statuses:

- `draft`
- `starting`
- `running`
- `waiting_for_user`
- `proposal_ready`
- `applying`
- `complete`
- `failed`
- `canceled`

Avoid ambiguous "done" states. `complete` means the session itself is complete, not necessarily that every proposed project artifact is done.

### Proposal Kinds

Initial proposal kinds:

- `backlog_batch_import`: initiatives plus backlog items; supports preview and apply.
- `operating_mode_draft`: human-readable mode proposal, contract decisions, implementation steps, impacted files, tests.
- `operating_mode_implementation_plan`: a backlog/initiative creation proposal for implementing the mode.

Each proposal must have a machine-validated payload and a human-readable summary.

## Implementation Strategy

### Phase 0 - Lock the Existing Identity and Mutation Seams

1. Add focused tests that prove the current identity flow end-to-end:
   - Agent Manager injects `VROOLI_AGENT_IDENTITY_TOKEN`.
   - Swarm Manager CLI forwards it as `X-Agent-Identity-Token`.
   - Swarm Manager API middleware verifies it and puts `identity.Provenance` in context.
   - A single backlog create stores `CreatedBy` from context.
2. Add a regression test showing batch backlog create currently needs provenance stamping. This test should define the desired behavior and initially fail until Phase 3.
3. Add a regression test showing initiative create needs the same attribution shape.
4. Confirm `VROOLI_AGENT_MANAGER_API_BASE` is available in Agent Manager spawned runs. If it is not injected globally today, add that to the Agent Manager spawn environment contract in the implementation phase.
5. Document the identity contract in `scenarios/swarm-manager/docs/internal/SEAMS.md` and cross-link Agent Manager's identity docs.

### Phase 1 - Add Session Domain Contracts

1. Add the new proto domain/API schemas for sessions, proposals, artifacts, and attribution.
2. Generate Go and TypeScript proto artifacts through the repo's proto generation workflow.
3. Add TypeScript contract mappers in `ui/src/services/proto-contracts` or the nearest current proto contract seam.
4. Add Go conversion helpers in `api/internal/agentsessions/proto.go`.
5. Add type-level tests for:
   - status/kind validation;
   - artifact reference validation;
   - proposal payload validation;
   - JSON round-trip of sessions, messages, proposals, and artifacts.

### Phase 2 - Build Backend Session Service and Storage

1. Create `api/internal/agentsessions`.
2. Implement a storage interface and file-backed store.
3. Implement the service methods:
   - create session;
   - list sessions;
   - get session;
   - append user message;
   - refresh session from Agent Manager run state;
   - cancel session;
   - attach artifact;
   - record proposal;
   - apply proposal.
4. Use `agentactivity.WithSpec` when spawning/continuing session agents:
   - owner type: likely `initiative` for initiative-scoped sessions and a new session owner once supported;
   - purpose: `meta_orchestration` or `operating_mode_authoring`;
   - metadata: `entrypoint=agent_sessions.<kind>`, `session_id`, `skill_id`, `proposal_kind`.
5. Because `agentactivity.OwnerType` currently lacks `session`, decide during implementation whether to:
   - add `OwnerSession` as the clean long-term owner type; or
   - use `OwnerInitiative` only for initiative-scoped sessions and record the session relationship separately.
   
   Preferred: add `OwnerSession` in backend/proto/UI filters. This is a hard cutover, so avoid pretending session-owned activity is initiative-owned when it is not.
6. Spawn session agents with:
   - resolved skill instructions;
   - session context;
   - current Swarm Manager state summary relevant to the kind;
   - environment variables from D4.
7. Continue session agents through Agent Manager `ContinueRun`.
8. Persist the assistant's final/partial response into messages on refresh.
9. Emit eventlog records for session lifecycle changes.

### Phase 3 - Implement Durable Attribution Across Mutation Chokepoints

1. Extend `identity.Provenance` to optionally include:
   - `SessionID`;
   - `SessionKind`;
   - `Source`;
   - `ClaimsMeta` if needed.
2. Add an API-side resolver that maps verified `run_id` to an active or historical `AgentSession`. The resolver should be service-owned, not UI-owned.
3. On every mutation request, derive `ArtifactAttribution` from:
   - verified request provenance;
   - run-to-session lookup;
   - explicit internal session apply context when the API applies a proposal itself.
4. Update single backlog create to include session-aware attribution.
5. Update batch backlog create:
   - stamp `CreatedBy` on every created item;
   - stamp artifact records for every created item;
   - stamp artifact records for every created or updated initiative in the same apply action;
   - keep rollback behavior atomic and remove artifact records if the write rolls back.
6. Update initiative create/update/delete service methods to accept a mutation context or provenance context.
7. Add `created_by` and, if useful, `updated_by` to initiatives.
8. Add entity lookup endpoint `GET /api/v1/artifacts/by-entity`.
9. Keep `spawned_from` for backlog lineage only if it remains semantically useful, but do not use it as the session attribution mechanism.
10. Add tests for:
   - single create attribution;
   - batch create attribution across multiple initiatives and items;
   - initiative update attribution;
   - agent token invalid/missing fallback;
   - artifact rollback on batch failure.

### Phase 4 - Add Session Proposal Apply Workflows

1. Implement `backlog_batch_import` proposal parsing and validation.
2. Reuse `BatchCreate` logic through a service-level applier rather than calling HTTP from the API to itself.
3. Support preview and apply:
   - preview validates the exact initiatives/items and returns warnings;
   - apply persists all accepted entities and creates artifact records.
4. Allow multiple initiatives and multiple backlog items in one apply action.
5. Implement `operating_mode_draft` proposal storage:
   - mode ID;
   - purpose;
   - workflow phases;
   - prompt/skill needs;
   - expected contracts;
   - files to change;
   - tests to add.
6. Implement `operating_mode_implementation_plan` apply:
   - creates an initiative and backlog items for implementation, or attaches to an existing initiative if the operator chooses that;
   - does not directly edit operating mode code from the session chat.
7. Require explicit operator approval before any apply action.
8. Emit events for proposal created, proposal previewed, proposal applied, proposal rejected, proposal superseded.

### Phase 5 - Build UI Session Service and Store

1. Add `agent-session-service.ts` with methods matching the API.
2. Add `agent-session-store.ts` or React Query hooks consistent with current Swarm Manager UI patterns.
3. Add polling for active sessions:
   - active statuses poll every few seconds;
   - stale sessions show a clear state;
   - completed/failed/canceled sessions stop polling.
4. Add optimistic message append only where it cannot desynchronize proposal state.
5. Add typed error handling for:
   - spawn unavailable;
   - Agent Manager run failed;
   - proposal invalid;
   - apply conflict;
   - identity/attribution unavailable.
6. Add unit tests for store polling, refresh, message append, and proposal apply state transitions.

### Phase 6 - Add Sidebar Sessions Tab

1. Extend sidebar tab types:
   - add `sessions` to `SIDEBAR_TABS`;
   - add labels, default sort, filters, and persisted-state migration.
2. Add `SessionsTab.tsx`.
3. Session list card requirements:
   - title;
   - kind icon;
   - status chip;
   - relative updated time;
   - artifact count;
   - proposal count;
   - run status indicator when active.
4. Add filters:
   - status;
   - kind;
   - active-only;
   - has proposals;
   - has applied artifacts.
5. Selecting a session opens the session detail panel, not a route.
6. Add tests for sidebar state restore, tab rendering, filtering, and empty states.

### Phase 7 - Add Session Detail Panel

1. Extract reusable conversation components from `ClarificationPanel` where sensible:
   - message list;
   - composer;
   - attachments;
   - stale/polling banner;
   - action footer.
2. Do not import clarification-specific stores or backlog decision models into sessions.
3. Build `AgentSessionPanel`.
4. Panel sections:
   - Header: title, kind, status, run state, close.
   - Chat: transcript and composer.
   - Proposals: proposal cards with preview/apply/reject/supersede.
   - Artifacts: linked entities and files.
   - Run details: Agent Manager run ID, task ID, profile key, timestamps, failure text.
5. Support reopening/resuming a session from:
   - Sessions sidebar tab;
   - artifact attribution chip;
   - launcher after creating a session;
   - proposal/apply success links.
6. Add UI tests for:
   - create session;
   - continue session;
   - proposal ready state;
   - apply proposal with multiple artifacts;
   - artifact click navigation;
   - attribution chip reopening the session.

### Phase 8 - Replace Single Quick Capture FAB With a Launcher

1. Replace the single "New capture" button behavior in `GraphWorkspace` with a compact launcher.
2. Launcher options:
   - Quick Capture: opens existing capture panel.
   - Plan Work With Agent: creates/opens a `meta_orchestration` session.
   - Author Operating Mode: creates/opens an `operating_mode_authoring` session.
3. On desktop, use a small popover/menu anchored to the bottom action.
4. On mobile, use a bottom sheet or menu that fits the existing `FloatingPanel` behavior.
5. Preserve Quick Capture as a first-class option; do not hide it behind session language.
6. Use clear labels without explanatory marketing copy.
7. Add keyboard/a11y behavior:
   - focus trap for menu/panel;
   - escape closes;
   - menu items are buttons with icons;
   - no text overflow at mobile widths.
8. Add visual regression/playwright coverage for desktop and mobile launcher states.

### Phase 9 - Add Entity Attribution Chips

1. Add a reusable `AttributionChip` component.
2. Add chips to:
   - `BacklogDetailsPage` or the existing detail header/info tab component;
   - `InitiativeDetailsPage`;
   - operating-mode proposal/detail surfaces introduced by this plan.
3. Chip behavior:
   - shows `Created by <session title>` when session attribution exists;
   - shows `Created by agent:<profile>/<run>` if there is verified agent provenance but no session;
   - shows `Created by operator` for operator-created entities;
   - clicking a session attribution opens `AgentSessionPanel`;
   - clicking non-session agent provenance can open Agent Activity or Agent Manager run details where available.
4. Add artifact list backlinks from session detail to entity details.
5. Add tests for chip rendering and navigation.

### Phase 10 - Metrics and Stats

1. Add eventlog event types:
   - `agent_session.created`;
   - `agent_session.started`;
   - `agent_session.continued`;
   - `agent_session.completed`;
   - `agent_session.failed`;
   - `agent_session.canceled`;
   - `agent_session.proposal_created`;
   - `agent_session.proposal_applied`;
   - `agent_session.artifact_linked`.
2. Extend stats aggregation with `SessionStats`:
   - sessions by kind;
   - sessions by status;
   - proposal apply rate by kind;
   - artifacts created by kind;
   - average messages per session;
   - average time from session start to first proposal;
   - failed session rate;
   - session-created backlog/initiative counts.
3. Add stats UI cards/charts to the existing stats surface. Keep it work-focused and compact.
4. Update any analytics docs to describe the new metrics.
5. Add stats unit tests using synthetic event streams.

### Phase 11 - Documentation and Seams

1. Update `scenarios/swarm-manager/docs/internal/SEAMS.md` with:
   - session service seam;
   - proposal apply seam;
   - artifact attribution seam;
   - identity-to-session attribution seam.
2. Update `scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md` with native sessions as a project-management capability.
3. Add `scenarios/swarm-manager/docs/internal/AGENT-SESSIONS.md` describing:
   - session lifecycle;
   - supported kinds;
   - identity/attribution behavior;
   - proposal/application behavior;
   - artifact model;
   - UI entry points.
4. Update the meta-orchestrator and operating-mode authoring skill docs only if their CLI/API instructions need to mention the native session flow. Do not make the skills responsible for manual attribution.
5. Add maintenance notes for adding future session kinds.

### Phase 12 - End-to-End Validation

1. Start Swarm Manager through the scenario lifecycle system.
2. Validate Quick Capture still works from the new launcher.
3. Create a meta-orchestration session from the UI.
4. Continue the session across at least two user messages.
5. Produce a proposal containing multiple initiatives and backlog items.
6. Preview the proposal.
7. Apply the proposal.
8. Confirm:
   - all entities exist;
   - sidebar Sessions tab lists the session;
   - session detail lists all artifacts;
   - artifact links navigate to entity details;
   - entity details show session attribution chips;
   - chips reopen the session;
   - stats include session counts and artifact counts.
9. Create an operating-mode authoring session.
10. Produce a proposal draft.
11. Apply only the implementation planning proposal, not direct code mutation.
12. Confirm Agent Manager run IDs and Swarm Manager session IDs are joined correctly.

## Testing Plan

Backend unit tests:

- `api/internal/agentsessions` store round-trip.
- Session status transition validation.
- Proposal payload validation.
- Artifact reference validation.
- Run-to-session attribution resolver.
- Batch backlog create stamps `CreatedBy` and artifact links.
- Initiative create/update stamps attribution.
- Event emission for session/proposal/artifact lifecycle.

Backend integration tests:

- Create session -> spawn Agent Manager run -> record run ID.
- Continue session -> Agent Manager continue called with correct run ID.
- Refresh session -> terminal run summary becomes assistant message.
- Apply meta-orchestration proposal with multiple initiatives/items.
- Failed apply rolls back entities and artifact records.
- Missing/invalid identity token falls back to operator without false session attribution.

CLI tests:

- Existing `identity_transport` tests remain.
- Add tests for any new session CLI commands if introduced.
- If CLI adds `agent-session` commands, ensure commands are thin wrappers over API and inherit identity transport automatically.

UI unit tests:

- Sidebar tab type and persisted state migration.
- Sessions tab filtering/sorting.
- Launcher opens correct panel/session kind.
- Agent session panel message rendering and send states.
- Proposal cards preview/apply/reject behavior.
- Attribution chip behavior.

UI integration/Playwright tests:

- Desktop launcher and session panel.
- Mobile launcher and session panel.
- No text overflow in launcher/menu/session cards.
- Session artifact click navigates to details.
- Attribution chip reopens session panel.

Stats tests:

- Synthetic event stream builds session adoption stats.
- Proposal apply rate denominator/numerator are correct.
- Artifact counts group by session kind and artifact type.

Suggested validation commands:

```bash
cd packages/proto && make generate
cd packages/cli-core && go test ./...
cd scenarios/agent-manager && make test
cd scenarios/swarm-manager/api && GOWORK=off go test ./...
cd scenarios/swarm-manager/cli && GOWORK=off go test ./...
cd scenarios/swarm-manager/ui && pnpm test -- --run
cd scenarios/swarm-manager && make test
```

Use `vrooli scenario test swarm-manager` or test-genie for full scenario validation when the local services are available.

## Rollout and Validation Checklist

- [ ] Identity flow documented and regression-tested.
- [ ] Session proto contracts generated for Go and TypeScript.
- [ ] Session API routes added and covered.
- [ ] Session storage is append-friendly and audited.
- [ ] Session service spawns and continues Agent Manager runs.
- [ ] Session agents receive session env vars.
- [ ] API joins verified agent run IDs to sessions automatically.
- [ ] Backlog single create, batch create, update, delete paths have attribution where relevant.
- [ ] Initiative create, update, delete paths have attribution.
- [ ] Meta-orchestration proposal can apply multiple initiatives/items.
- [ ] Operating-mode authoring proposal is proposal-first and does not directly mutate code.
- [ ] Sidebar Sessions tab implemented.
- [ ] Session detail panel implemented.
- [ ] Graph launcher replaces single-purpose capture button.
- [ ] Artifact list and attribution chips are bidirectional.
- [ ] Stats include session usage and artifact outcomes.
- [ ] Docs updated.
- [ ] Full tests and scenario validation pass.

## Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Session attribution relies on run ID but a session spawns multiple runs over time. | Artifacts may attach to the wrong session or no session. | Store a `run_ids[]` history on the session and maintain a run-to-session index. |
| Batch create partially succeeds before artifact records are written. | Audit trail drifts from entity state. | Apply entities and artifact records through one service transaction boundary with explicit rollback for file storage. |
| Reusing clarification components pulls backlog-specific assumptions into sessions. | Future session kinds become hard to add. | Extract only presentation primitives; keep session state/store/API separate. |
| Agent Manager identity verification is fail-open. | Invalid tokens can appear as operator-created work. | Never attach session attribution unless run ID is verified and joined to a session. Show operator attribution explicitly. |
| UI launcher becomes too crowded. | Common quick-capture workflow slows down. | Keep launcher to three icon+label commands; make Quick Capture the first item. |
| Operating-mode authoring encourages direct code mutation from chat. | Reviewability and safety regress. | Proposal-first contract; implementation work is created as Swarm Manager artifacts before coding. |
| Proto mode enum constraints keep causing generated contract churn. | Adding modes remains harder than intended. | Replace hard-coded string validation in proto with runtime registry validation where feasible, and document the registry as source of truth. |

## Non-Goals and Prohibited Patterns

- Do not create a dedicated Sessions page.
- Do not implement runtime operating-mode plugins.
- Do not keep a duplicate legacy session model once this lands.
- Do not require agents to manually pass `--created-by`, `--session-id`, or similar identity flags for normal Swarm Manager CLI operations.
- Do not infer artifact attribution solely from UI state.
- Do not scrape eventlog as the only way to render entity attribution.
- Do not add untyped catch-all chat sessions.
- Do not bypass Swarm Manager API mutation services from session agents.
- Do not add fallback prompt behavior that silently continues with missing skill contracts.

## Definition of Done

The work is done when an operator can open Swarm Manager, start a meta-orchestration or operating-mode authoring session from the graph launcher, continue it later from the Sessions sidebar tab, apply reviewed proposals that create or update multiple project-management artifacts, and navigate bidirectionally between sessions and those artifacts with verified attribution. The implementation must be covered by backend, CLI, UI, stats, and scenario tests, and the identity/session/artifact seams must be documented for future session kinds.
