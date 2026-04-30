# API Endpoints

This document captures the canonical Swarm Manager API shapes that matter for backlog planning and initiative management.

## Contract Rules

- Unknown JSON fields are rejected at the HTTP boundary.
- `scope` is not part of the backlog contract.
- Backlog execution boundaries are expressed with `acceptance_allow` and `acceptance_deny`.
- Initiative assignment is per backlog item (`initiative`), not a batch-level flag.

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
definition. The backend validates the registered phase graph before reserving a
round, acquiring the initiative lock, rendering the prompt, or spawning an
agent. Failed/canceled rounds do not advance the graph; active rounds block all
new phase starts. Prompt skills, AgentManager profiles, activity purposes, lock
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

Supported non-default phases:
- `holistic-loop`: `investigate`, `plan`, `execute`, `review`
- `phased-plan-drain`: `prepare_plan`, `execute_next`, `classify_progress`, `review`

## Initiative Operating Mode Round Control

`POST /api/v1/initiatives/{name}/operating-mode/rounds/{round}/refresh?mode={mode}`

Polls AgentManager for the round's run and persists terminal state when the run
is complete, failed, or canceled. Completed runs may include a final
`operating_mode_result` JSON envelope; when present, Swarm Manager persists
declared artifacts, handoffs, progress state, verdicts, and replan signals into
the round/workspace.

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
applier used by initiative feedback before applying the accepted mutation IDs.

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

## Initiative Feedback Proposals

The feedback flow accepts an agent-emitted proposal envelope and applies the
selected mutations. The envelope shape is owned by `proposals.Proposal`; the
mutation list shown below is the surface the apply layer enforces.

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
      "purpose": "Run one workshop round for non-research backlog items and update plan.md.",
      "output_paths": ["workshop/round-NNN.json", "plan.md"]
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
