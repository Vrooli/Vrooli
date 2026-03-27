# API Endpoints

This document captures the canonical Swarm Manager API shapes that matter for backlog planning and initiative management.

## Contract Rules

- Unknown JSON fields are rejected at the HTTP boundary.
- `scope` is not part of the backlog contract.
- Backlog execution boundaries are expressed with `acceptance_allow` and `acceptance_deny`.
- Initiative assignment is per backlog item (`initiative`), not a batch-level flag.

## Backlog Create

`POST /api/v1/backlog`

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

Updates are partial.

```json
{
  "title": "Desktop Release Governance",
  "description": "Revised wording only"
}
```

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
