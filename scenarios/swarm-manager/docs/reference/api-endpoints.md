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
