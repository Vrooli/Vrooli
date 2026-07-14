# API Endpoints

This document captures the canonical Swarm Manager API shapes that matter for backlog planning and initiative management.

## Contract Rules

- Unknown JSON fields are rejected at the HTTP boundary.
- `scope` is not part of the backlog contract.
- Backlog execution boundaries are expressed with `acceptance_allow` and `acceptance_deny`.
- Initiative assignment is per backlog item (`initiative`), not a batch-level flag.

## Agent Sessions

`POST /api/v1/agent-sessions`

Creates a typed draft session. It does not spawn Agent Manager and does not
append a message.

Valid `kind` values are `meta_orchestration`, `swarm_operations`, and
`operating_mode_authoring`.

```json
{
  "kind": "swarm_operations",
  "title": "Manage Swarm operations"
}
```

`POST /api/v1/agent-sessions/{session_id}/start`

Starts a draft session with the first real operator message. This is the only
path that turns a draft into an Agent Manager run. A start request must include
at least one of `message`, `attachment_ids`, or `context_refs`.
By default, starting any session kind attaches the kind-specific
`startup_brief/<kind>` context unless the request sets
`"auto_context_policy": "none"` or already includes an equivalent startup
context. `swarm_operations` also accepts the legacy direct
`operations_briefing/latest` context.

```json
{
  "message": "Here is the context to plan...",
  "attachment_ids": ["att_abc123"],
  "auto_context_policy": "default",
  "context_refs": [
    { "type": "startup_brief", "ref": "startup_brief/swarm_operations" },
    { "type": "initiative", "ref": "desktop-release-governance" },
    { "type": "backlog_item", "ref": "fix/auth-bug" }
  ]
}
```

`POST /api/v1/agent-sessions/{session_id}/continue`

Appends a follow-up message after the session has a run. The request shape
matches start and supports text, session-owned image attachment IDs, typed
context refs, or a combination.

`POST /api/v1/agent-sessions/{session_id}/attachments`

Uploads image files owned by the session. Use multipart form data with one or
more `files` parts. The response returns attachment records that can be
referenced by `start` or `continue`.

```json
{
  "attachments": [
    {
      "id": "att_abc123",
      "filename": "whiteboard.png",
      "content_type": "image/png",
      "size_bytes": "42121",
      "created_at": "2026-05-15T14:30:00Z"
    }
  ]
}
```

`GET /api/v1/agent-sessions/{session_id}/attachments/{attachment_id}`

Streams a session-owned uploaded image. Attachments are deleted with the owning
session.

`GET /api/v1/agent-sessions/{session_id}/events?after_sequence=0&limit=100`

Returns bounded session-owned Agent Manager run events. Draft/no-run sessions
return `events: []`. Tool inputs/outputs and raw fallbacks are truncated by the
API boundary.

## Plan Board

`GET /api/v1/plan`

Returns the Plan-lens board projection (`PlanBoardResponse`): a Now header
summary (active count, queue depth, lane utilization), Next/Later/Done card
columns, and meta (generated_at, effective `window_seconds`, max dependency
wave, cycle diagnostics). Next mixes human gate cards (decide / review /
classify from the gates read-model) with runnable and needs-workshop item
cards at dependency wave 0; Later groups blocked items by nearest blocker
with ordinal wave badges from `depgraph.Waves` frontier peeling; Done carries
window-capped recent outcomes. Now-column *cards* come from
`GET /api/v1/operations` — this endpoint only carries the header counts.

Query parameters:
- `window_seconds`: Done-column window in seconds, clamped to [60, 86400],
  default 86400

Board refreshes ride `/ws/graph`: mutating services include the
dispatch-only `plan` lens in invalidation payloads.

`GET /api/v1/graph?lens=topology`

Returns the topology projection (nodes/edges/meta). Topology is the only
HTTP graph projection lens. The UI Graph surface renders this projection
directly by default and filters it client-side when `mode=focus` is active.
Any other `lens` value returns an invalid-lens error.

## Operations

`GET /api/v1/operations`

Returns the live operations aggregate: lane utilization, queue depth, active
activities, recently finished activities, `generated_at`, and the effective
`window_seconds`.

Query parameters:
- `window`: ISO-8601 PT duration, default `PT3H`, max `PT24H`
- `status`, `lane`, `mode`, `owner_type`: repeatable filters
- `q`: substring search over owner title/name and run ID

`GET /api/v1/operations/brief`

Returns the bounded operations briefing contract used by CLI, UI, and
`swarm_operations` agent-session prompt context. The response is generated on
demand from current operations, overview, stats, and director-handoff sources.
Optional source failures appear in `warnings`; core operations aggregation
failure returns an API error.

Response fields:
- `generated_at`, `freshness_seconds`, `window_seconds`
- `summary`: active/recent counts, queue depth, saturated lanes, backlog,
  initiative, blocked-item, and active-session counts
- `active_work`: top bounded active activity rows
- `needs_attention`: failed, review-needed, saturated, or queue-pressure items
- `recent_completions`: top bounded recent completions
- `director_handoffs`: bounded excerpts from director-swarm handoffs
- `recommended_next_actions` and `drill_down_commands`: deterministic command
  and UI hints
- `warnings`: stale/missing optional source notes

## Backlog Create

`POST /api/v1/backlog`

JSON create:

```json
{
  "kind": "idea",
  "name": "my-feature",
  "title": "My Feature",
  "description": "Short description",
  "priority": 3,
  "effort": "M",
  "initiative": "release-control",
  "depends_on": ["fix/auth-bug"],
  "acceptance_allow": ["scenarios/swarm-manager/**"],
  "acceptance_deny": ["scenarios/swarm-manager/secrets/**"]
}
```

Multipart create with files:

- `Content-Type: multipart/form-data`
- `item`: JSON `CreateBacklogItemRequest`
- `files_manifest`: JSON object with `files[]` entries
- file parts: one uploaded file part for each manifest entry

```json
{
  "files": [
    {
      "field": "file_0",
      "path": "evidence/report.json",
      "content_type": "application/json"
    },
    {
      "field": "file_1",
      "path": "evidence/screenshot.png",
      "content_type": "image/png"
    }
  ]
}
```

Attached file paths must be explicit safe relative paths. Absolute paths,
traversal, duplicate paths, and `spec.json` are rejected. The item and files
are created as one logical operation; validation or write failures roll back the
new item directory.

## Backlog Update

`PUT /api/v1/backlog/{kind}/{name}`

Only send the fields you want to change.

```json
{
  "title": "Updated Title",
  "priority": 2,
  "acceptance_allow": ["scenarios/swarm-manager/api/**"]
}
```

## Backlog Batch Create / Preview

`POST /api/v1/backlog/batch`

The same endpoint supports preview and real creation.

```json
{
  "preview": true,
  "items": [
    {
      "kind": "research",
      "name": "desktop-release-control-plane-audit",
      "title": "Audit desktop release control plane",
      "description": "Trace the release path across deployment-manager, scenario-to-desktop, LPBS, and prompt-manager skills.",
      "priority": 1,
      "effort": "M",
      "initiative": "desktop-release-governance",
      "acceptance_allow": [
        "scenarios/deployment-manager/**",
        "scenarios/scenario-to-desktop/**",
        "scenarios/landing-page-business-suite/**",
        "scenarios/prompt-manager/**"
      ]
    }
  ],
  "initiatives": [
    {
      "name": "desktop-release-governance",
      "title": "Desktop Release Governance",
      "description": "Shared release-control and desktop delivery work.",
      "status": "active"
    }
  ]
}
```

Behavior:
- `preview=true` performs validation only
- omitting `preview` or setting `false` performs the real create
- initiative metadata is created or updated before items are written
- failures roll back the whole batch

## Workshop Cancel Pending Advance

`DELETE /api/v1/backlog/{kind}/{name}/workshop/pending-advance`

Cancels a pending auto-advance countdown for the given backlog item. When `auto_advance_delay_seconds > 0` in settings, the `WorkshopSave` endpoint creates a deferred advance instead of spawning immediately. This endpoint cancels that deferred advance before it fires.

Response:
```json
{
  "cancelled": true
}
```

- `cancelled: true` — a pending advance was found and cancelled
- `cancelled: false` — no pending advance existed (idempotent)

## Backlog Archive / Unarchive

Archive sets `archived_at` on a backlog item. Items retain their terminal status when archived.

`PATCH /api/v1/backlog/{kind}/{name}/archive-item`

Response: the updated backlog item with `archived_at` set.

`DELETE /api/v1/backlog/{kind}/{name}/archive-item`

Unarchives the item (clears `archived_at`). Response: the updated backlog item.

### Archive Query Filter

All list endpoints support `?archived=` query parameter:
- `false` (default) — exclude archived items
- `true` — only archived items
- `all` — include everything

## Initiatives Create

`POST /api/v1/initiatives`

```json
{
  "name": "desktop-release-governance",
  "title": "Desktop Release Governance",
  "description": "Shared release-control and desktop delivery work.",
  "status": "active"
}
```

## Initiatives Update

`PUT /api/v1/initiatives/{name}`

Updates are partial. This endpoint owns descriptive initiative metadata and
acceptance criteria. Operating-mode changes use the dedicated switch endpoint
below so active item-level executions can be handled explicitly.

```json
{
  "title": "Desktop Release Governance",
  "description": "Revised wording only",
  "acceptance_criteria": [
    "The full initiative can be reviewed at system scope."
  ]
}
```

New initiatives always start in `item-level`. The generic create/update
endpoints reject `mode`; use the operating-mode switch endpoint below for every
mode change.

## Operating Mode Catalog

`GET /api/v1/operating-modes`

Returns the backend registry's switchable operating modes, merged with any
persisted overlay (label/description) and annotated with current per-mode
`usage_count`. UI and CLI selection surfaces consume this endpoint instead of
maintaining hard-coded mode lists.

```json
{
  "modes": [
    {
      "mode": "item-level",
      "label": "Item Level",
      "description": "Default mode. Each backlog item flows through the existing item execution pipeline...",
      "usage_count": 12,
      "target_kind": "initiative",
      "run_strategy": "existing_item_flow",
      "workspace_tab_id": "info",
      "default": true,
      "switchable": true,
      "supports_phases": false,
      "phases": []
    }
  ]
}
```

`GET /api/v1/operating-modes/{mode}`

Returns one mode's catalog entry plus the list of initiatives currently bound
to it. Backs the operating-mode details page. Returns `404` for unknown modes.

```json
{
  "entry": { "mode": "holistic-loop", "label": "Holistic Loop", "...": "..." },
  "linked_initiatives": [
    { "name": "init-a", "title": "Initiative A", "status": "active", "updated": "2026-04-30" }
  ]
}
```

`PATCH /api/v1/operating-modes/{mode}`

Edits user-visible fields (label, description) via the overlay store. Pointer
semantics: a missing field leaves the existing value unchanged; an empty-string
`description` clears the override (registry default returns); empty-string
`label` is rejected with `400`. Mode IDs and structural fields (phases,
capabilities) are immutable. Returns the same shape as the GET endpoint.

```json
{ "label": "Holistic Loop (renamed)", "description": "Updated for our team." }
```

`POST /api/v1/operating-modes/{mode}/simulate?preset={preset}`

Backs the operating-mode detail page's **Flow** tab. Deterministically walks a
phase mode's registered graph against an ephemeral fixture target (an initiative
or a plan, per the mode's declared `target`) and returns the same
structured-result projection live rounds use — without spawning agents,
rendering prompts, acquiring locks, or persisting state. Rejects `item-level`
(no phase graph) with a `400`.

The optional `preset` query param (or `{"preset": "..."}` body field) selects a
named branch-covering scenario; an empty/unknown id falls back to the first
(happy-path) preset. The response lists all `presets` for the mode and echoes the
resolved `active_preset`. Presets seed different phase outputs to exercise real
transition guards (replan, continue, blocked, non-accepting review, reconcile).

```json
{
  "mode": "phased-plan-drain",
  "label": "Phased Plan Drain",
  "active_preset": "blocked",
  "presets": [
    { "id": "happy-path", "label": "Drains in one slice", "branch": "execute →(progress=continue)→ execute →(progress=complete)→ stop" },
    { "id": "blocked", "label": "Work is blocked", "branch": "execute →(progress=blocked)→ guarded stop" }
  ],
  "target": { "kind": "plan-manager-plan", "…": "…" },
  "trace": [ { "phase": "execute", "inputs": {}, "output": {}, "transition": {} } ]
}
```

## Initiative Operating Mode Switch

`POST /api/v1/initiatives/{name}/operating-mode/switch`

Switches the initiative's operating mode through the lifecycle-aware mode
boundary. When switching from `item-level` into a non-default mode, active
member item executions cause a `409 active_item_executions` response unless the
request explicitly confirms cancellation. Switching out of an initiative-scoped
mode is rejected with `409 active_operating_mode_round` while any mode round is
reserved or agent-running.

```json
{
  "mode": "holistic-loop",
  "cancel_active_item_executions": true,
  "requested_by": "operator"
}
```

Response:

```json
{
  "initiative_name": "desktop-release-governance",
  "from_mode": "item-level",
  "to_mode": "holistic-loop",
  "canceled_item_executions": [
    {
      "item_ref": "execute/item-1",
      "execution_id": "exec-123",
      "run_id": "run-456",
      "status": "canceled"
    }
  ]
}
```

## Initiative Operating Mode Workspace

`GET /api/v1/initiatives/{name}/operating-mode/workspace`

Returns the current mode definition, backend-computed phase action state, live
initiative lock holder if present, declared mode artifacts, and durable phase
rounds. For active rounds, the API best-effort refreshes AgentManager state
before responding.

## Initiative Operating Mode Phase Start

`POST /api/v1/initiatives/{name}/operating-mode/phases/{phase}/start`

Starts an initiative-scoped operating-mode phase through the registered mode
definition. The backend validates the registered phase graph before attempting
the lifecycle, then owns round reservation, initiative lock acquisition, prompt
rendering, AgentManager spawn, and run-ID lock ownership as one temporal flow
[CODE: api/internal/operatingmode/phase_runner.go]. Failed/canceled rounds do
not advance the graph; active rounds block all new phase starts. Lock conflicts,
prompt failures, spawn failures, and run-ID lock swap failures are terminally
recorded as failed audit rounds and must not leave an active reserved/running
round behind. Prompt skills, AgentManager profiles, activity purposes, lock
purposes, run strategies, and artifact policies are resolved from the registry.
Prompt rendering is fail-closed: catalog misses, skill mismatches,
prompt-manager errors, and empty rendered content fail the start before
AgentManager spawn.

```json
{
  "note": "Focus this pass on the API runner foundation.",
  "override": false,
  "requested_by": "operator"
}
```

Response: `202 Accepted` with the created round envelope.

Supported non-default phases (initiative-target modes only — this surface is
initiative-keyed and rejects plan-target modes such as `phased-plan-drain`,
which start via `OperatingModeService.StartTargetPhase`):
- `holistic-loop`: `investigate`, `plan`, `execute` (`executed_by: phased-plan-drain`), `review`, `reconcile`

## Initiative Operating Mode Round Control

`POST /api/v1/initiatives/{name}/operating-mode/rounds/{round}/refresh?mode={mode}`

Round-control endpoints are non-default-mode-only. Callers should pass
`mode={mode}` explicitly. If the query value is omitted, the API may infer the
mode from the initiative only when the initiative is already in a non-default
operating mode; blank mode and `item-level` are rejected rather than treated as
fallbacks [CODE: api/internal/operatingmode/connect_service.go].

Polls AgentManager for the round's run and persists terminal state when the run
is complete, failed, or canceled. Completed runs may include a final
`operating_mode_result` JSON envelope. Required phases fail closed when the
envelope is missing, malformed, empty, or violates the phase output contract.
Swarm Manager stages artifact and payload changes, validates required
artifacts, handoffs, progress decisions, review verdicts, and allowed replan
signals, and only then marks the round completed [CODE: api/internal/operatingmode/artifact_applier.go].
Contract failures mark the round failed, release the initiative lock, and emit
a phase-failed event rather than a misleading completion.

`POST /api/v1/initiatives/{name}/operating-mode/rounds/{round}/cancel?mode={mode}`

Stops the AgentManager run when it is still active, marks the round canceled,
and releases the initiative lock when the run is the current holder.

`POST /api/v1/initiatives/{name}/operating-mode/rounds/{round}/complete-items?mode={mode}`

Run-id-validated backlog reconciliation endpoint for non-default operating
modes. It marks only member backlog items complete and emits an
`operating_mode.backlog_synced` audit event. The underlying
`backlog.status_changed` event also carries a structured `source` payload with
entrypoint, initiative, mode, phase, round, run ID, requested-by, and item refs.
Agents must use this boundary instead of editing backlog `spec.json` files.

```json
{
  "run_id": "run-456",
  "item_refs": ["execute/item-1"],
  "requested_by": "operator-or-agent"
}
```

Response:

```json
{
  "initiative_name": "desktop-release-governance",
  "mode": "holistic-loop",
  "phase": "execute",
  "round": 3,
  "run_id": "run-456",
  "completed_items": [
    {
      "item_ref": "execute/item-1",
      "from_status": "ready",
      "to_status": "completed"
    }
  ]
}
```

`POST /api/v1/initiatives/{name}/operating-mode/rounds/{round}/apply-backlog-sync?mode={mode}`

Run-id-validated backlog reconciliation endpoint for create/update/follow-up
work proposed by a completed operating-mode round. The round must contain a
`backlog_sync.proposal` object in its final `operating_mode_result`; Swarm
Manager normalizes and validates that proposal through the same proposal
applier used by proposal sessions before applying the accepted mutation IDs.

```json
{
  "run_id": "run-456",
  "accepted_mutation_ids": ["m1", "m3"],
  "requested_by": "operator"
}
```

Response includes `proposal_result` with applied/failed/skipped counts,
per-mutation outcomes, and created/updated summary counts. The round payload is
updated with the applied sync result and an `operating_mode.backlog_synced`
event is emitted. Applied proposal mutation events carry the same operating-mode
source metadata (`mode`, `phase`, `round`, `run_id`, and `entrypoint`) through
the proposal event payload.

## Initiative Archive / Unarchive

Archive sets `archived_at` on an initiative. Initiatives retain their status when archived.

`PATCH /api/v1/initiatives/{name}/archive-item`

`DELETE /api/v1/initiatives/{name}/archive-item`

## Session Mutation Proposals

Proposal sessions accept an agent-emitted envelope and apply only the selected
mutations. Create a target-bound session with `POST /api/v1/proposal-sessions`,
then review its `mutation_list` proposal on the target's Proposals tab. The
envelope shape is owned by `proposals.Proposal`; the mutation list shown below
is the surface the apply layer enforces.

### Proposal envelope

```json
{
  "form": "mutation_list",
  "rationale": "One sentence on the overall intent.",
  "mutations": [
    { "id": "m1", "op": "...", "rationale": "..." }
  ]
}
```

`form` is `mutation_list` for direct ops or `full_graph` for target-state
synthesis (server diffs and emits the equivalent mutation_list).

### Supported mutation ops

| `op` | Required fields | Effect |
|------|-----------------|--------|
| `add_item` | `item: ItemSpec` | Creates a new backlog item attached to the initiative. |
| `update_item` | `target`, `patch: ItemPatch` | Patches metadata (title, description, priority, tags, depends_on, effort, acceptance globs, note). |
| `change_status` | `target`, `status` | Non-terminal, non-lifecycle status transitions only. |
| `change_priority` | `target`, `priority` | Sets priority to 1-10. |
| `add_edge` | `from`, `to` | Adds a `from depends_on to` edge. |
| `remove_edge` | `from`, `to` | Removes an existing edge. |
| `move_initiative` | `target`, `initiative` | Transfers an item to another initiative; empty `initiative` detaches. |
| `archive_item` | `target` | Sets `archived_at`. |
| `interrupt_in_progress` | `target` | Cancels the active execution; must be a separate mutation, not implicit. |
| `split_item` | `target`, `into: [ItemSpec]` (≥2) | Atomic: creates children, archives source. **Dependents are not auto-retargeted** — emit explicit `add_edge`/`remove_edge` mutations alongside the split if you need to repoint dependents. |
| `merge_items` | `sources: [ref]` (≥2), `item: ItemSpec` | Atomic: creates the merged item, retargets external edges (to/from sources) onto the merged item, drops intra-source edges, archives sources. Validation rejects if any source is `in_progress` — emit `interrupt_in_progress` as a prior mutation if interruption is intended. The merged item enters as `backlog`. |

### `merge_items` wire shape

```json
{
  "id": "m1",
  "op": "merge_items",
  "sources": ["execute/sandbox-aware-cli", "execute/sandbox-lifecycle-coord"],
  "item": {
    "kind": "execute",
    "name": "sandbox-runtime-coord",
    "title": "Coordinate sandbox runtime path",
    "description": "Combines aware-cli + lifecycle-coord; substrate is shared.",
    "priority": 3,
    "effort": "M"
  },
  "rationale": "Both items refactor the same workspace-sandbox runtime entrypoint."
}
```

**Edge handling:** for each edge `(a, b)` in the current graph,

- `a ∈ sources, b ∈ sources` → dropped
- `a ∈ sources, b ∉ sources` → merged item gains dep on `b`
- `a ∉ sources, b ∈ sources` → `a`'s `depends_on` is rewritten to point at the merged ref (deduped)

**Event audit:** the resulting `backlog.proposal_applied` event attaches to the merged item's ref and carries `payload.sources = [...]` so per-source history queries can render "this item was merged into X" without re-deriving from archive timestamps.

**Constraints:**

- `sources` must contain ≥2 distinct refs, all current members of the initiative.
- The merged item's ref must differ from every source.
- The merged item's ref must not collide with an existing non-source item or with another item staged earlier in the same proposal.
- No source may be in `in_progress` at validation time.

## Settings

`GET /api/v1/settings`

The CLI uses `settings.default_mode` when `execution create` is called without `--mode`.

## Execution

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/execution` | List executions with optional filters |
| POST | `/api/v1/execution` | Create a new execution |
| GET | `/api/v1/execution/{id}` | Get execution by ID |
| GET | `/api/v1/execution/{id}/prompt-trace` | Get prompt trace for execution |
| POST | `/api/v1/execution/{id}/start` | Start a pending/scheduled execution |
| POST | `/api/v1/execution/{id}/cancel` | Cancel an active execution |
| POST | `/api/v1/execution/{id}/retry` | Retry a failed execution |
| POST | `/api/v1/execution/{id}/follow-up` | Create follow-up from terminal execution |
| POST | `/api/v1/execution/{id}/trigger-review` | Trigger or re-trigger a GCT review for a terminal execution |
| GET | `/api/v1/gct/status` | Check git-control-tower availability (`{"available": true/false}`) |

## Canonical Evidence

Canonical evidence is immutable producer output linked to exactly one Session
or operating-mode execution after exhaustive owner resolution. The endpoints
return owner-linked records; they never expose identity tokens or raw Agent
Manager tool payloads.

| RPC | Description |
|-----|-------------|
| `EvidenceService.ListRun` | List canonical records for a verified run. |
| `EvidenceService.ListEntity` | List records affecting a normalized entity. |
| `EvidenceService.Reconcile` | Retry supported producers for a run. |
| `EvidenceService.RecordOperatorVerification` | Append an operator repair with owner, stable event id, actor, reason, subject, and action. |

An operator repair is a new `operator_verified` observation, never a mutation
or confidence promotion of an existing reported record. Requests attributed to
a verified agent are rejected from this endpoint.

## Agent Activities

`AgentActivity` is the durable telemetry/audit record for tracked AgentManager usage. Unlike execution records, activities are created for workshop/research/classify/follow-up/spec-sync flows in addition to governed backlog processing.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/agent-activities` | List tracked agent activities with optional filters (`owner_type`, `owner_kind`, `owner_name`, `execution_id`, `purpose`, `status`, `run_id`, `active`) |
| GET | `/api/v1/agent-activities/{activity_id}` | Get one tracked agent activity by ID |

### Trigger Review

Manually triggers a git-control-tower review for executions in terminal status (`completed`, `needs_fixup`, `failed`). Returns the updated execution record with `status: "validating"` and `review_job_id` set.

Returns 400 if the execution is not in a terminal status. Returns 500 if ReviewClient is not configured or GCT is unreachable.

### GCT Status

Lightweight health check against git-control-tower. Always returns 200 with `{"available": true}` or `{"available": false}`. Uses a 3-second timeout.

## Prompts

Swarm Manager owns the prompt inventory contract. Prompt-manager still owns prompt skill content.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/prompts/catalog` | List the canonical runtime prompt catalog, including generated execution prompts and support/reference skills |
| GET | `/api/v1/prompts/skills` | List prompt-manager skills referenced by the catalog with usage summaries |
| GET | `/api/v1/prompts/skills/{id}` | Get one catalog-backed prompt skill |
| PUT | `/api/v1/prompts/skills/{id}` | Update one catalog-backed prompt skill |
| GET | `/api/v1/prompts/skills/{id}/versions` | List prompt skill version history |
| POST | `/api/v1/prompts/skills/{id}/revert/{version}` | Revert a prompt skill to a previous version |
| POST | `/api/v1/prompts/preview` | Render a catalog-backed prompt-manager skill with variables |
| POST | `/api/v1/prompts/simulate` | Simulate backlog runtime prompts for `workshop`, `initialize`, or `finalize` |

### Prompt Catalog Entry

`GET /api/v1/prompts/catalog`

```json
{
  "items": [
    {
      "id": "backlog-workshop",
      "title": "Backlog Workshop",
      "group": "backlog",
      "usage_type": "direct_runtime",
      "source_type": "skill",
      "trigger": "Backlog workshop round",
      "skill_id": "swarm-manager-workshop",
      "backlog_kinds": ["idea", "fix", "execute", "chore"],
      "modes": ["workshop"],
      "purpose": "Run one workshop round for non-research backlog items and update the canonical plan-manager plan.",
      "output_paths": ["workshop/round-NNN.json", "spec.json.plan_ref"]
    },
    {
      "id": "execution-process",
      "title": "Execution Process Prompt",
      "group": "execution",
      "usage_type": "generated_runtime",
      "source_type": "generated",
      "trigger": "Execution start / retry",
      "builder": "execution.buildExecutionPrompt",
      "operations": ["generator", "improver"],
      "purpose": "Build the runtime execution prompt from the backlog deliverable."
    }
  ]
}
```

### Prompt Simulation

`POST /api/v1/prompts/simulate`

```json
{
  "kind": "idea",
  "mode": "workshop",
  "item_title": "Prompt Catalog",
  "item_folder": "scenarios/swarm-manager/ideas/prompt-catalog"
}
```
