# API Reference

Prompt Manager's supported programmatic contract is generated Connect-RPC.
Resolve the live `API_PORT` through the scenario lifecycle; clients must not
hard-code ports or construct legacy REST paths.

## Generated Connect surface

The schemas live under `packages/proto/schemas/prompt-manager/v1/<domain>` and
generated clients live under `packages/proto/gen/{go,ts}/prompt-manager/v1`.
Connect procedures use the standard path
`/<fully-qualified-service>/<method>` and work with binary protobuf or Connect
JSON. The API mounts these generated services:

| Domain | Service | RPCs | Responsibility |
|---|---|---:|---|
| skills | `SkillsService` | 16 | Skill CRUD, reads, sync, history, variants, ratings |
| experiments | `ExperimentsService` | 14 | Experiment lifecycle, assignments, evidence, outcomes, promotion |
| actions | `ActionsService` | 7 | Action authoring, validation, CRUD, governed execution |
| tags | `TagsService` | 2 | Persisted tag taxonomy |
| search | `SearchService` | 6 | Deterministic entity/content search |
| aisearch | `AISearchService` | 8 | Semantic search and index reconciliation |
| discovery | `DiscoveryService` | 4 | Capability discovery, gaps, telemetry, skill usage |
| agents | `AgentsService` | 15 | Agent identity, soul, files, membership reads |
| teams | `TeamsService` | 28 | Team aggregate, membership, roles, files, org, messages, exchange |
| topics | `TopicsService` | 8 | Topic taxonomy, matching, accumulated skills |
| templates | `TemplatesService` | 1 | Agent-file templates |
| testing | `TestingService` | 2 | Skill tests and durable history |
| metadata | `MetadataService` | 1 | Open Graph metadata lookup |
| graph | `GraphService` | 13 | Relationship graph, health, config, and node projections |
| heartbeat | `HeartbeatService` | 52 | Scheduling, runs, queues, prompts, handoffs, tasks, retention, bug intake |
| memberflow | `MemberflowService` | 14 | Topics, rules, objectives, operating models, drain/orientation evidence |
| world | `WorldService` | 5 | World config, per-scene layout overrides and the server-streamed swarm feed |

Stable identifiers and method inputs are typed. Heartbeat/memberflow payloads
whose upstream catalogs intentionally evolve are carried as
`google.protobuf.Value` behind typed method and identity boundaries; consumers
must still use the generated clients rather than reconstructing JSON routes.

## Measures substrate

- `GET /measures/declarations` returns the nine registered, read-only measure declarations.
- `POST /measures/execute` executes a named measure and returns a scalar or table plus mandatory provenance.

The measures endpoint is the fleet-wide `measures-go` contract and is not a
second business API. Each compute function calls the owning domain's real store
or telemetry service.

## Remaining HTTP compatibility routes

Only six hand-written registrations remain: `/health`, `/api/v1/health`, and
the GET/PUT budget and discovery-filter configuration pairs. They are explicit
compatibility/configuration seams. All former domain REST registrations were
retired after generated-client consumers migrated.

## Historical REST reference

The material below documents the retired pre-Connect wire surface for migration
archaeology only. It is not a supported client contract.

**Historical base URL:** `http://localhost:{PORT}/api/v1`

All endpoints return JSON. Error responses follow the format:
```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

---

## Health

### GET /health

Check API server health.

**Response:**
```json
{
  "status": "healthy",
  "version": "2.0.0",
  "checks": {
    "database": "healthy"
  }
}
```

---

## Skills

[CODE: api/skills/handlers.go]

### GET /api/v1/skills

List all skills with optional filtering.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `folder` | string | Filter by folder: `core`, `local`, `drafts` |
| `tag` | string | Filter by tag |
| `mode` | string | Filter by mode |

**Response:**
```json
[
  {
    "id": "debugging",
    "name": "Debugging",
    "description": "Systematic debugging approach",
    "content": "## Steer focus: Debugging...",
    "modes": ["agent"],
    "tags": ["debugging"],
    "folder": "core",
    "draft": false,
    "usageCount": 42,
    "effectivenessRating": 4,
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-20T14:30:00Z"
  }
]
```

### GET /api/v1/skills/{id}

Get a single skill by ID.

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Skill ID (filename without extension) |

**Response:** Same as list item above.

**Errors:**
- `404` - Skill not found

### POST /api/v1/skills

Create a new skill.

**Request Body:**
```json
{
  "name": "My Skill",
  "description": "What this skill does",
  "content": "## Steer focus: My Skill\n\nContent here...",
  "folder": "local",
  "tags": ["debugging", "testing"],
  "modes": ["agent"],
  "draft": false
}
```

**Required Fields:** `name`, `content`, `folder`

**Response:** Created skill object with generated `id`.

### PUT /api/v1/skills/{id}

Update an existing skill.

**Request Body:** (all fields optional)
```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "content": "Updated content...",
  "tags": ["new-tag"],
  "folder": "local",
  "draft": true
}
```

**Response:** Updated skill object.

**Notes:**
- Creates a new version in version history
- Can move skill to different folder by specifying `folder` (version history moves with the skill)

### DELETE /api/v1/skills/{id}

Delete a skill.

**Response:** `204 No Content`

**Notes:**
- Cannot delete skills in `core` folder
- Removes all version history

---

## Version History

[CODE: api/skills/handlers.go#GetVersions]

### GET /api/v1/skills/{id}/versions

Get version history for a skill.

**Response:**
```json
{
  "skillId": "my-skill",
  "current": 3,
  "versions": [
    {
      "version": 1,
      "name": "My Skill",
      "content": "Original content...",
      "updatedAt": "2024-01-15T10:00:00Z"
    },
    {
      "version": 2,
      "name": "My Skill v2",
      "content": "Updated content...",
      "updatedAt": "2024-01-18T12:00:00Z"
    },
    {
      "version": 3,
      "name": "My Skill v3",
      "content": "Current content...",
      "updatedAt": "2024-01-20T14:30:00Z"
    }
  ]
}
```

### POST /api/v1/skills/{id}/revert/{version}

Revert a skill to a specific version.

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Skill ID |
| `version` | int | Version number to revert to |

**Response:**
```json
{
  "skillId": "my-skill",
  "revertedTo": 2,
  "newVersion": 4,
  "restoredAt": "2024-01-21T09:00:00Z"
}
```

**Notes:**
- Reverts content and name from specified version
- Creates a new version (does not delete history)

---

## Variants

### GET /api/v1/skills/{id}/variants

List all variants for a skill.

**Response:** `VariantResponse[]`

### GET /api/v1/skills/{id}/variants/{vid}

Get a variant with its content.

**Response:** `VariantResponse` (includes `content` field)

### POST /api/v1/skills/{id}/variants

Create a new variant for a skill.

**Request:**
```json
{
  "id": "concise-v1",
  "name": "Concise Style",
  "description": "A more concise prompt variant",
  "content": "# Concise\nShort and sweet."
}
```

**Response:** `201 Created` with `VariantResponse`

### PUT /api/v1/skills/{id}/variants/{vid}

Update a variant's metadata and/or content.

**Request:**
```json
{
  "name": "Updated Name",
  "content": "Updated content"
}
```

### DELETE /api/v1/skills/{id}/variants/{vid}

Delete a variant. Returns `204 No Content`.

---

## Experiments

### GET /api/v1/experiments

List all experiments.

**Response:** `ExperimentResponse[]`

### GET /api/v1/skills/{id}/experiments

List experiments for a specific skill.

**Response:** `ExperimentResponse[]`

### GET /api/v1/experiments/{eid}

Get experiment details, including outcome counts.

**Response:** `ExperimentResponse` (includes `outcomeCounts` map)

### POST /api/v1/experiments

Create a new experiment.

**Request:**
```json
{
  "id": "exp-concise-test",
  "skillId": "swarm-manager-workshop",
  "name": "Concise vs Detailed Workshop",
  "hypothesis": "Concise prompts produce equal quality with less tokens",
  "protocol": {
    "population": "reference workflow",
    "randomizationUnit": "workflow-node-per-execution",
    "primaryMetric": "evaluator verdict",
    "effectThreshold": 0.05,
    "strata": ["workflowRevision"],
    "exposurePolicy": "exclude-contaminated",
    "outcomeCompletenessThreshold": 0.9,
    "budget": "100 executions",
    "stoppingRule": "fixed sample",
    "holdoutRequired": true,
    "holdoutPopulationHash": "sha256:...",
    "promotionAuthority": "operator",
    "evaluatorRubricHash": "sha256:...",
    "evaluatorAuthor": "independent-evaluator"
  },
  "arms": [
    {"variantId": "control", "weight": 0.5},
    {"variantId": "concise-v1", "weight": 0.5}
  ]
}
```

**Notes:**
- `arms.weight` values must sum to 1.0 (±0.01 tolerance)
- `control` is a reserved variant ID representing the original SKILL.md
- All non-control variant IDs must exist for the skill
- Experiment starts in `draft` status
- The complete protocol is required and is frozen with its hash at start

### PUT /api/v1/experiments/{eid}

Update a draft experiment (name, hypothesis, protocol, arms).

**Note:** Only draft experiments can be updated.

### DELETE /api/v1/experiments/{eid}

Delete an experiment and its outcomes. Returns `204 No Content`.

### POST /api/v1/experiments/{eid}/start

Transition experiment from `draft` to `running`.

### POST /api/v1/experiments/{eid}/conclude

Conclude a running experiment with a recommendation.

**Request:**
```json
{
  "winnerVariantId": "concise-v1",
  "notes": "Equal quality, 40% faster execution time"
}
```

**Notes:**
- Winner must be one of the experiment's arms
- Conclude never changes `SKILL.md`
- A signed, clear audit receipt matching the frozen protocol is required before a recommendation work item is published
- A separately authorized, holdout-confirmed promotion is required to apply content

### POST /api/v1/experiments/{eid}/audit-receipt

Persist a server-signed audit receipt for a running experiment. The caller
supplies sampled assignment identifiers, a findings hash, challenge state, and
anomaly/gaming counts; Prompt Manager binds the frozen protocol hash and signs
the canonical receipt. This endpoint requires `PROMPT_MANAGER_EXPERIMENT_AUDIT_SECRET`.
Provision this value from the deployment secret manager. Use one stable, non-empty
secret for the lifetime of an experiment because Prompt Manager verifies the
receipt signature again before it publishes a recommendation or promotes a
winner. Do not store this value in a workflow, skill, experiment protocol, or
source-controlled service declaration.

### POST /api/v1/experiments/{eid}/holdout-receipt

Record the separate holdout finding after conclusion. The server signs the
experiment ID, frozen protocol hash, published work item ID, finding hash, and
completion time. It requires `findingsHash` and an idempotency key.

### POST /api/v1/experiments/{eid}/promote

Apply a concluded winner only after a signed holdout receipt and the exact
published `skill-experiment-promotion` work item has status `accepted` in the
`meta-optimization` team. The caller supplies `{"workItemId":"..."}`; a

caller assertion, topic entry, or different approved work item is insufficient.

### POST /api/v1/experiments/{eid}/outcomes

Record an opaque outcome. Called by consuming applications (e.g. swarm-manager).

**Request:**
```json
{
  "variantId": "concise-v1",
  "source": "swarm-manager",
  "schemaVersion": 1,
  "data": {"classification": "ready", "durationSecs": 347}
}
```

**Notes:**
- Only running experiments accept outcomes
- The `data` field is opaque to prompt-manager
- `schemaVersion` is defined by the source system

### GET /api/v1/experiments/{eid}/outcomes

List raw outcomes for an experiment.

**Response:** `ExperimentOutcomeResponse[]`

### GET /api/v1/experiments/{eid}/report

Aggregated per-arm report for an experiment. Terminal status and token use are
guardrail observations only. They are not a primary outcome or a promotion
metric. Arms with zero records are listed in `zeroDataArms`.

**Response:** `ExperimentReportResponse`

### Variant-Aware Read (extension to POST /api/v1/skills/read)

Reads participate in controlled experiments only by explicit arming. Include
`experimentId` in the request. The experiment must be `running` and target the
resolved skill. An optional `variantId` selects a declared arm deterministically
for calibration or workflow dispatch. Reads without `experimentId` are
observational and never receive a treatment arm.

**Additional request fields:**
```json
{
  "experimentId": "exp-concise-test",
  "variantId": "concise-v1",
  "variantPolicy": "pinned",
  "source": "agent-manager"
}
```

- `variantPolicy: "pinned"` or `"control"` — serve the original SKILL.md and skip experiment sampling (agent-manager sends `pinned` for workflow prompt refs that are not deliberately armed, preserving workflow determinism)
- `source` — free-form caller label recorded with the serve

**Additional response fields:**
```json
{
  "selectedVariantId": "concise-v1",
  "experimentId": "exp-concise-test"
}
```

**Notes:**
- `control` means the original SKILL.md was used (no content replacement)
- Variable substitution is applied to variant content as normal
- Controlled evidence is written to prompt-manager's SQLite experiment store with an idempotency key. `GET /api/v1/experiments/{eid}/report` aggregates controlled records only.

---

## Usage Tracking

### POST /api/v1/skills/{id}/use

Record skill usage (call when user copies/uses a skill).

**Response:**
```json
{
  "skillId": "debugging",
  "usageCount": 43,
  "lastUsed": "2024-01-21T09:00:00Z"
}
```

### PUT /api/v1/skills/{id}/rating

Set effectiveness rating for a skill.

**Request Body:**
```json
{
  "rating": 4,
  "notes": "Very helpful for complex bugs"
}
```

**Rating:** Integer 1-5

**Response:** `204 No Content`

---

## Search

[CODE: api/search/handlers.go]

### SearchService.SearchSkills

Full-text search across skills.

**Request fields:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `q` | string | Search query (searches name, description, content) |
| `tag` | string | Filter by tag |
| `folder` | string | Filter by folder |

**Response:**
```json
{
  "results": [
    {
      "id": "debugging",
      "name": "Debugging",
      "description": "Systematic debugging approach",
      "folder": "core",
      "tags": ["debugging"],
      "modes": ["agent"],
      "score": 0.95,
      "highlight": "...systematic **debugging** approach for..."
    }
  ],
  "total": 1,
  "query": "debugging"
}
```

**Notes:**
- Results ranked by relevance score
- Highlight shows matching context

---

### SearchService.SearchSkillContent

Content-only search across skill bodies (line-level matches).

**Request fields:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `q` | string | Search query (required) |
| `tag` | string | Filter by tag (repeatable or comma-separated) |
| `folder` | string | Filter by folder (repeatable or comma-separated) |
| `caseSensitive` | boolean | Case-sensitive matching |
| `wholeWord` | boolean | Whole word matching |
| `regex` | boolean | Treat query as regex |
| `limit` | integer | Max number of matches (default: 200) |

**Response:**
```json
{
  "matches": [
    {
      "skillId": "debugging",
      "skillName": "Debugging",
      "file": "core/debugging.md",
      "folder": "core",
      "lineNumber": 42,
      "line": "Use systematic debugging to isolate failures.",
      "matchRanges": [
        { "start": 14, "end": 23 }
      ]
    }
  ],
  "total": 1,
  "query": "debugging"
}
```

**Notes:**
- Matches are line-level with character ranges for highlights.
- Invalid regex returns `400 Bad Request`.

---

## AI Search

### AISearchService.SearchSkills

Semantic search powered by embeddings, with optional combined output formatting.

**Request Body:**
```json
{
  "query": "react coherence",
  "limit": 5,
  "output": "results",
  "format": "xml",
  "renderLimit": 3
}
```

**Output Options:** `results`, `combined`, `both` (default: `results`)
**Format Options:** `xml`, `markdown`, `json` (applies when output includes `combined`)

**Response:**
```json
{
  "results": [
    {
      "id": "ui-health",
      "name": "React Coherence",
      "description": "Architectural patterns for React",
      "folder": "core",
      "tags": ["react"],
      "modes": ["frontend"],
      "score": 0.92,
      "scorePercent": 92
    }
  ],
  "combined": "<skills>...</skills>",
  "skillCount": 3,
  "totalTokens": 2500,
  "format": "xml",
  "total": 1,
  "query": "react coherence",
  "method": "ai",
  "output": "both"
}
```

### AISearchService.GetStatus

Returns AI search availability status.

### AISearchService.Reconcile

Reconcile the qdrant index with on-disk content. The reconciler uses a
content-hash diff (`payload_hash`) so unchanged items skip embedding
entirely; ghost points whose backing files are gone are deleted.

**Request fields:**
- `collection=skills|agents|teams|topics|actions|all` — restrict to one
  collection. Defaults to `all`.
- `dry_run=true` (or `X-Dry-Run: true` header) — return the planned
  upserts/deletes without mutating qdrant or running embeddings.

**Dry-run response (200):**
```json
{
  "dry_run": true,
  "plan": {
    "plannedAt": "2026-05-06T10:00:00Z",
    "collections": [
      {
        "kind": "skill",
        "toUpsert": [{"kind":"skill","pointId":"...","name":"...","payloadHash":"sha256:..."}],
        "toDelete": ["pt-orphan"],
        "unchangedCount": 30,
        "legacyCount": 0
      }
    ]
  }
}
```

**Live response:** the kickoff is asynchronous; use `GetReconcileStatus` for completion.

### AISearchService.GetReconcileStatus

Returns the reconciler's last-known state.

**Response:**
```json
{
  "running": false,
  "startedAt": "2026-05-06T10:00:00Z",
  "finishedAt": "2026-05-06T10:00:12Z",
  "lastResult": {
    "collections": [
      {"kind":"skill","upserted":2,"deleted":1}
    ],
    "errors": []
  }
}
```

### AISearchService.CancelReconcile

Cancel an in-progress reconcile operation.

**Response:** the same `ReconcileStatus` shape as `/status`, with
`canceled: true`.

### Environment knobs

- `AI_SEARCH_SYNC_INTERVAL` — periodic reconcile interval (default `5m`).
- `AI_SEARCH_SYNC_DISABLED=1` — disable the periodic loop entirely.
- `AI_SEARCH_RECONCILE_PARALLELISM` — concurrent embed/upsert workers
  (default 4, clamped to [1, 16]).

---

## Actions

Actions are typed wrappers over exactly one Vrooli-controlled CLI command. Storage, CRUD, validation, discovery, graph integration, governed API execution, the thin CLI run wrapper, and the UI run panel are implemented. See [DOC: docs/concepts/ACTIONS.md].

### GET /api/v1/actions

List Actions with optional filtering.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `pack` | string | Filter by pack: `core`, `local`, `drafts` |
| `status` | string | Filter by status: `active`, `draft`, `archived` |
| `owner` | string | Filter by owning scenario/resource/project |
| `tag` | string | Filter by tag |

**Response:**
```json
[
  {
    "id": "scenario.ui.screenshot",
    "name": "Take Scenario Screenshot",
    "description": "Capture a screenshot of a running scenario UI.",
    "status": "active",
    "owner": {
      "type": "scenario",
      "id": "prompt-manager"
    },
    "pack": "core",
    "updatedAt": "2026-04-30T00:00:00Z"
  }
]
```

### GET /api/v1/actions/{id}

Get a single Action contract by ID.

**Response:** Full Action metadata including input schema, output schema, command target, permissions, examples, and validation metadata.

### POST /api/v1/actions

Create a new Action contract.

**Notes:**
- The API must reject command strings that require shell interpretation.
- The command target must be a Vrooli-controlled CLI command.
- Creation should validate input/output schemas and permission declarations.
- Invalid contracts return `422` with a validation response.

### PUT /api/v1/actions/{id}

Update an existing Action contract. Updates validate the replacement contract before persistence.

### DELETE /api/v1/actions/{id}

Archive an Action by default. Use `?hard=true` for hard deletion.

### POST /api/v1/actions/{id}/validate

Validate an Action contract without running its target operation.

**Response:**
```json
{
  "valid": true,
  "actionId": "scenario.ui.screenshot",
  "checks": [
    {
      "name": "command-target",
      "status": "pass",
      "message": "Command target is Vrooli-controlled."
    }
  ]
}
```

### POST /api/v1/actions/{id}/run

Run an active, runnable Action with typed input through the governed argv-only runtime.

Governance enforced before process start:
- contract validation and controlled-command resolution
- declared input type/default validation and placeholder rendering
- command run-surface eligibility
- per-Action timeout capped by service maximum
- process-wide concurrency limit
- stdout/stderr byte caps with truncation flags
- bounded `runs.jsonl` audit history

**Request Body:**
```json
{
  "input": {
    "scenario": "prompt-manager",
    "viewport": "desktop"
  }
}
```

Use `"dryRun": true` to validate inputs and return the rendered argv without starting the process.

**Response:**
```json
{
  "actionId": "scenario.ui.screenshot",
  "status": "completed",
  "exitCode": 0,
  "durationMs": 1234,
  "stdout": "{\"imagePath\":\"/tmp/prompt-manager-screenshot.png\"}",
  "stderr": "",
  "stdoutTruncated": false,
  "stderrTruncated": false,
  "output": {
    "imagePath": "/tmp/prompt-manager-screenshot.png"
  }
}
```

Safe seed dry-run example:

```json
{
  "actionId": "scenario.status.show",
  "status": "dry-run",
  "durationMs": 0,
  "argv": ["vrooli", "scenario", "status", "prompt-manager"],
  "validation": {
    "actionId": "scenario.status.show",
    "valid": true,
    "runnable": true
  }
}
```

Execution uses the argv-shaped command contract from `action.json`. Branching and implementation logic belong in the owning CLI, not the Action runtime.

Rejected runs return `422` with status `rejected`. Concurrency saturation returns `429` with status `throttled`. Command failures and timeouts return `200` with status `failed` or `timed-out` so callers can inspect captured output and audit context.

### DiscoveryService.Discover

Discover skills, Actions, or both. Omitting `type` preserves the legacy skill-only response shape.

```json
{
  "queries": ["take screenshot of scenario UI"],
  "type": "all",
  "limit": 10
}
```

`type` accepts `skill`, `action`, or `all`. Mixed responses include a result type discriminator so agents can prefer exact Actions for deterministic operations and skills for judgment-heavy work.

---

## Sync

### GET /api/v1/skills/sync

Get all skills with hash for change detection.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `tag` | string | Filter by tag |

**Response:**
```json
{
  "skills": [...],
  "lastUpdated": "2024-01-21T09:00:00Z",
  "hash": "abc123def456"
}
```

**Use Case:** Clients can cache skills and compare hash to detect changes.

---

## Read

### POST /api/v1/skills/read

Read multiple skills by identifier, with optional combined output formatting.

**Request Body:**
```json
{
  "identifiers": ["debugging", "testing", "refactor"],
  "resolve": "auto",
  "allowMissing": true,
  "output": "auto",
  "format": "xml"
}
```

**Output Options:** `skills`, `combined`, `both`, `auto` (default: `auto` → combined for multiple identifiers, skills for single)
**Format Options:** `xml`, `markdown`, `json` (applies when output includes `combined`)

**Response:**
```json
{
  "skills": [
    {
      "id": "debugging",
      "name": "Debugging",
      "description": "Systematic debugging approach",
      "content": "...",
      "modes": ["agent"],
      "tags": ["debugging"],
      "folder": "core"
    }
  ],
  "combined": "<skills>...</skills>",
  "skillCount": 3,
  "totalTokens": 2500,
  "format": "xml",
  "resolve": "auto",
  "output": "both",
  "missing": [],
  "ambiguous": []
}
```

**Use Case:** Read skills for piping or generate combined output for LLM context.

---

## Tags

[CODE: api/tags/handlers.go]

### GET /api/v1/tags

List all tags.

**Response:**
```json
[
  {
    "id": "abc123",
    "name": "debugging",
    "color": "#FF5733",
    "description": "Skills for debugging issues"
  }
]
```

### POST /api/v1/tags

Create a new tag.

**Request Body:**
```json
{
  "name": "my-tag",
  "color": "#3B82F6",
  "description": "Tag description"
}
```

**Required Fields:** `name`

**Response:** Created tag object.

---

## Testing (Ollama)

[CODE: api/testing/handlers.go]

### POST /api/v1/skills/{id}/test

Test a skill with Ollama through `resource-ollama gateway generate`.

**Prerequisites:** `OLLAMA_ENABLED=true`, `resource-ollama` available on PATH (or `OLLAMA_GATEWAY_BIN` set), and the requested model loaded.

**Request Body:**
```json
{
  "model": "llama3.2",
  "variables": {
    "TARGET": "src/auth/login.ts"
  },
  "maxTokens": 1000,
  "temperature": 0.7
}
```

**Response:**
```json
{
  "testId": "test-123",
  "model": "llama3.2",
  "response": "Based on the debugging skill...",
  "responseTime": 2500.5,
  "tokenCount": 450,
  "testedAt": "2024-01-21T09:00:00Z"
}
```

**Notes:**
- Variables replace `{{VAR}}` placeholders in skill content
- Test results saved to database for history

### GET /api/v1/skills/{id}/test-history

Get test history for a skill.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 10 | Max results to return |

**Response:**
```json
[
  {
    "id": "test-123",
    "skillId": "debugging",
    "model": "llama3.2",
    "inputVariables": "{\"TARGET\": \"src/auth\"}",
    "response": "...",
    "responseTime": 2500.5,
    "tokenCount": 450,
    "rating": 4,
    "notes": "Worked well",
    "testedAt": "2024-01-21T09:00:00Z"
  }
]
```

---

## Agents

[CODE: api/agents/handlers.go]

Agents represent team entities visualized in the 3D world. They are organized into teams and reference skills in markdown.

### GET /api/v1/agents

List all agents.

**Response:**
```json
[
  {
    "id": "agent-1",
    "displayName": "Alice",
    "status": "active",
    "appearance": {
      "body": "#3B82F6",
      "head": "#F59E0B",
      "accent": "#10B981"
    },
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-20T14:30:00Z"
  }
]
```

### GET /api/v1/agents/{id}

Get a single agent.

**Response:** Same as list item above.

**Errors:**
- `404` - Agent not found

### POST /api/v1/agents

Create a new agent.

**Request Body:**
```json
{
  "id": "alice",
  "displayName": "Alice",
  "appearance": {
    "body": "#3B82F6",
    "head": "#F59E0B",
    "accent": "#10B981"
  }
}
```

**Required Fields:** `displayName`

**Optional Fields:** `id` (auto-generated from displayName), `appearance`

**Notes:**
- Colors must be valid hex format: `#RRGGBB`

### PUT /api/v1/agents/{id}

Update an existing agent.

**Request Body:** (all fields optional)
```json
{
  "displayName": "Updated Name",
  "status": "inactive",
  "appearance": {
    "body": "#FF5733",
    "head": "#3498DB",
    "accent": "#2ECC71"
  }
}
```

**Status Values:** `active`, `inactive`, `suspended`

**Response:** Updated agent object.

### DELETE /api/v1/agents/{id}

Delete an agent.

**Response:** `204 No Content`

### POST /api/v1/prompt-preview

Preview the fully constructed prompt for an agent. When `teamId` is provided, this matches the flat runtime prompt used by heartbeat execution, including `HEARTBEAT.md`.

**Request Body:**
```json
{
  "agentId": "agent-1",
  "teamId": "engineering"
}
```

**Response:**
```json
{
  "agentId": "agent-1",
  "teamId": "engineering",
  "prompt": "# Agent Files (Markdown)\n\n..."
}
```

**Errors:**
- `400` - Missing agentId or no content available
- `404` - Agent or team not found

### POST /api/v1/prompt-preview-structured

Preview the same heartbeat prompt as ordered structured sections. This is the preferred UI rendering surface because the backend owns section order.

**Request Body:**
```json
{
  "agentId": "agent-1",
  "teamId": "engineering"
}
```

**Response:**
```json
{
  "agentId": "agent-1",
  "teamId": "engineering",
  "sections": [
    {
      "kind": "operating-policy-team",
      "label": "Operating Policy (Team)",
      "sourcePath": "teams/engineering/team.json#operatingContract.members.agent-1",
      "content": "# Operating Policy (Team)\n\n..."
    }
  ]
}
```

### GET /api/v1/teams/{id}/prompt-matrix

Return structured prompt sections for every active member of a team. Use this for cross-member prompt audits and drift detection.

**Response:**
```json
{
  "teamId": "engineering",
  "entries": [
    {
      "agentId": "agent-1",
      "displayName": "Agent One",
      "sections": []
    }
  ]
}
```

## Teams

[CODE: api/teams/handlers.go]

Teams represent organizational units that group agents together with roles and shared policies.

### GET /api/v1/teams

List all teams.

**Response:**
```json
[
  {
    "id": "engineering",
    "displayName": "Engineering Team",
    "mission": "Build great software",
    "memberCount": 5,
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-20T14:30:00Z"
  }
]
```

### GET /api/v1/teams/{id}

Get a single team with full details including roles and members.

**Response:**
```json
{
  "id": "engineering",
  "displayName": "Engineering Team",
  "mission": "Build great software",
  "memberCount": 5,
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-20T14:30:00Z",
  "roles": [
    {
      "id": "lead",
      "name": "Team Lead",
      "description": "Leads the team"
    }
  ],
  "members": [
    {
      "agentId": "agent-1",
      "displayName": "Alice",
      "roles": ["lead"],
      "status": "active"
    }
  ]
}
```

**Errors:**
- `404` - Team not found

### POST /api/v1/teams

Create a new team.

**Request Body:**
```json
{
  "id": "engineering",
  "displayName": "Engineering Team",
  "mission": "Build great software",
  "runtime": {
    "mode": "multi-process"
  },
  "coordination": {
    "pattern": "independent",
    "reportingMode": "none",
    "messagingMode": "disabled",
    "capabilities": {
      "showOrgContext": false,
      "injectInbox": false,
      "allowPeerTriggers": false,
      "showTaskBoardGuidance": true,
      "showKnowledgeLogGuidance": true,
      "requireHandoff": true
    }
  },
  "execution": {
    "queuePolicy": "bounded-parallel",
    "maxConcurrentRuns": 2
  },
  "operatingContract": {
    "schemaVersion": 1,
    "documents": {
      "planOfRecord": [],
      "sharedState": []
    },
    "knowledgeTopics": {},
    "members": {}
  }
}
```

**Required Fields:** `displayName`, `runtime`, `coordination`, `execution`, `operatingContract`

**Optional Fields:** `id` (auto-generated from displayName), `mission`

**runtime.mode Values:** `multi-process` - members run as separate heartbeat processes. `single-process` - one Claude Code lead session coordinates the team.

**coordination.pattern Values:** `independent`, `peer`, `leader-led`

**execution.queuePolicy Values:** `serialized`, `bounded-parallel`

**operatingContract.documents.sharedState:** Internal JSON field for team working state. Use final `kind` values such as `charter`, `task-board`, `working-register`, `rolling-snapshot`, `append-only-event-log`, and `operator-input`. Agent-facing prompts render this category as team working state in the Storage Map.

**operatingContract:** Required structured source of truth for team/member operating policy. Heartbeat prompts render member-specific contract data inside the generated `Operating Policy` section alongside runtime, coordination, and execution policy. Prompt rendering fails if required policy is missing or invalid.

**Response:** Created team object with `201 Created`.

### PUT /api/v1/teams/{id}

Update an existing team.

**Request Body:** (all fields optional)
```json
{
  "displayName": "Updated Name",
  "mission": "New mission",
  "enabled": true,
  "runtime": {
    "mode": "single-process"
  },
  "coordination": {
    "pattern": "leader-led",
    "leadAgentId": "director",
    "reportingMode": "leader",
    "messagingMode": "in-session",
    "capabilities": {
      "showOrgContext": true,
      "injectInbox": false,
      "allowPeerTriggers": false,
      "showTaskBoardGuidance": true,
      "showKnowledgeLogGuidance": true,
      "requireHandoff": true
    }
  },
  "execution": {
    "queuePolicy": "serialized",
    "maxConcurrentRuns": 1
  },
}
```

**Response:** Updated team object.

### DELETE /api/v1/teams/{id}

Delete a team and all its member relationships.

**Response:** `204 No Content`

### POST /api/v1/teams/{id}/members

Add an agent to a team.

**Request Body:**
```json
{
  "agentId": "agent-1",
  "roles": ["developer"]
}
```

**Required Fields:** `agentId`

**Optional Fields:** `roles`

**Response:** Created member object with `201 Created`.
```json
{
  "agentId": "agent-1",
  "displayName": "Alice",
  "roles": ["developer"],
  "status": "active"
}
```

**Errors:**
- `404` - Team or agent not found

### PUT /api/v1/teams/{id}/members/{agentId}

Update a team member's roles or status.

**Request Body:**
```json
{
  "roles": ["developer", "reviewer"],
  "status": "inactive"
}
```

**Response:** Updated member object.

### DELETE /api/v1/teams/{id}/members/{agentId}

Remove an agent from a team.

**Response:** `204 No Content`

### GET /api/v1/teams/{id}/roles

Get available roles for a team.

**Response:**
```json
[
  {
    "id": "lead",
    "name": "Team Lead",
    "description": "Leads the team"
  },
  {
    "id": "developer",
    "name": "Developer",
    "description": "Writes code"
  }
]
```

### PUT /api/v1/teams/{id}/roles

Set roles for a team (replaces all existing roles).

**Request Body:**
```json
{
  "roles": [
    {
      "id": "lead",
      "name": "Team Lead",
      "description": "Leads the team"
    },
    {
      "id": "developer",
      "name": "Developer",
      "description": "Writes code"
    }
  ]
}
```

**Response:** Updated roles array.

### GET /api/v1/teams/{id}/org

Get the org chart (manager/report edges) for a team.

**Response:**
```json
{
  "teamId": "engineering",
  "edges": [
    { "managerAgentId": "alice", "reportAgentId": "bob" },
    { "managerAgentId": "alice", "reportAgentId": "charlie" }
  ]
}
```

### PUT /api/v1/teams/{id}/org

Replace all org chart edges for a team.

**Request Body:**
```json
{
  "edges": [
    { "managerAgentId": "alice", "reportAgentId": "bob" }
  ]
}
```

**Validation Rules:**
- `managerAgentId` and `reportAgentId` must be team members
- A report can have only one manager
- No self-reporting
- No reporting cycles

### PUT /api/v1/teams/{id}/org/edges/{reportId}

Set a single reporting relationship (single-manager model).

**Request Body:**
```json
{
  "managerAgentId": "alice"
}
```

**Response:**
```json
{
  "managerAgentId": "alice",
  "reportAgentId": "bob"
}
```

### DELETE /api/v1/teams/{id}/org/edges/{reportId}

Remove a reporting relationship.

**Response:** `204 No Content`

### GET /api/v1/teams/{id}/members/{agentId}/messages

List inbox messages for a team member.

**Response:**
```json
{
  "teamId": "engineering",
  "agentId": "bob",
  "messages": [
    {
      "id": "msg-123",
      "teamId": "engineering",
      "fromAgentId": "alice",
      "toAgentId": "bob",
      "content": "Please review the latest PR.",
      "createdAt": "2026-02-01T12:00:00Z"
    }
  ]
}
```

### POST /api/v1/teams/{id}/members/{agentId}/messages

Send a message to a team member.

**Request Body:**
```json
{
  "fromAgentId": "alice",
  "content": "Please review the latest PR."
}
```

**Response:** Created message object with `201 Created`.

### DELETE /api/v1/teams/{id}/members/{agentId}/messages

Clear all messages for a team member.

**Response:** `204 No Content`

### DELETE /api/v1/teams/{id}/members/{agentId}/messages/{messageId}

Delete a single message.

**Response:** `204 No Content`

### GET /api/v1/teams/import/claude-code/available

[CODE: api/teams/handlers_import.go]

List Claude Code teams available for import from `~/.claude/teams/`.

**Response:**
```json
[
  { "name": "my-team", "memberCount": 3 },
  { "name": "research-team", "memberCount": 1 }
]
```

Returns an empty array if the directory does not exist or contains no team subdirectories.

### POST /api/v1/teams/import/claude-code

[CODE: api/teams/handlers_import.go]

Import a Claude Code team into prompt-manager.

**Request Body:**
```json
{
  "teamName": "my-cc-team"
}
```

Reads the CC team config from `~/.claude/teams/{teamName}/config.json` and creates the corresponding PM team, agents, member relations, and org chart.

**Response:** `201 Created` with team details.

**Errors:**
- `400` - Missing teamName or invalid config JSON
- `404` - CC team not found at expected path
- `409` - Team with that ID already exists

### GET /api/v1/teams/{id}/export/claude-code

[CODE: api/teams/handlers_export.go]

Export a prompt-manager team as a Claude Code team config.

**Response:**
```json
{
  "teamName": "engineering",
  "description": "Build great software",
  "members": [
    {"name": "lead", "agentType": "general-purpose"},
    {"name": "researcher", "agentType": "Explore"}
  ]
}
```

### POST /api/v1/teams/{id}/trigger

[CODE: api/heartbeat/handlers.go#TriggerTeam]

Trigger heartbeats for an entire team. Behavior depends on the resolved runtime and coordination policy:

- **`single-process` + `leader-led`**: Triggers only the configured lead agent's heartbeat. The lead session coordinates teammates through Claude Code interop.
- **All other team policies**: Triggers heartbeats for all members that have heartbeat configs.

Leader-led single-process triggers are validated before enqueueing: the lead must be an active team member and must have a heartbeat config.

**Response:** `202 Accepted`
```json
{
  "teamId": "engineering",
  "runtimeMode": "multi-process",
  "coordinationPattern": "independent",
  "queuePolicy": "bounded-parallel",
  "triggers": [
    {
      "teamId": "engineering",
      "agentId": "agent-1",
      "runId": "run-abc",
      "status": "running",
      "logPath": "2026-02-01T10-00-00Z.log"
    }
  ]
}
```

**Errors:**
- `400` - Invalid leader-led single-process configuration, inactive/missing lead member, or missing lead heartbeat config
- `404` - Team not found
- `409` - Team is disabled or member already queued/running
- `503` - Executor not configured

> **See also:** [Heartbeat API Reference](heartbeat-api.md) for the full heartbeat lifecycle including execution status (`GET /teams/{id}/execution-status`), member context (`GET /teams/{id}/members/{agentId}/context`), and heartbeat CRUD endpoints.

---

## World

[CODE: api/handlers/world/connect_handler.go]
[CODE: api/internal/world/config.go]
[CODE: api/internal/world/feed.go]

`WorldService` (`vrooli.prompt_manager.v1.world`) serves the 3D world at
`/vrooli.prompt_manager.v1.world.WorldService/<Method>` over Connect. Files
live under `<config root>/world/`.

| Method | Persists | Notes |
|---|---|---|
| `GetWorldConfig` / `SetWorldConfig` | `world/config.json` | scene (`park`, `office`), `qualityProfile` (`low`..`ultra`), `qualityAuto`, `periodMode` (`clock`, `dawn`, `day`, `dusk`, `night`), `twoDMode`, `showDiagnostics`, `scale` (0.25..4). Out-of-range values return `invalid_argument`; a malformed file is an error, never silently replaced. |
| `GetLayout` / `SetLayout` | `world/layout-<scene>.json` | Per-scene `overrides[]` (`placeId`, optional `position`, optional `rotation`, `removed`) applied over the generated layout by id, plus operator `decor[]` additions. Duplicate ids and out-of-range decor scale are rejected. Agent positions are never persisted. |
| `StreamWorldFeed` | in-memory ring | Server stream. Opens with a `SNAPSHOT` (active runs from the run registry, upcoming heartbeats from the scheduler), replays buffered events newer than `since_seq`, then streams live `RUN_STARTED`, `RUN_FINISHED`, `RUN_FAILED` (with the error in `message`), `HEARTBEAT_UPCOMING` (`scheduled_at`), `HEARTBEAT_CANCELLED` and `AGENT_MESSAGE`. A subscriber that falls behind the channel depth is closed with `resource_exhausted` and reconnects with `since_seq`. |

Signal sources: `heartbeat.RunRegistry` reports `RUN_STARTED` on register and
`RUN_FINISHED` / `RUN_FAILED` on completion (the executor passes the outcome);
a schedule watcher polls the cron scheduler and announces next runs inside a
six-hour horizon, and cancellations when a schedule disappears. The UI keeps
the 5 s `ListRunning` poll as a fallback when the stream is silent or failing.

## Graph

[CODE: api/graph/handlers.go]

The relationship graph maps connections between teams, agents, skills, Actions, and CLI tools. See [Graph Concepts](../concepts/GRAPH.md) for background.

### GET /api/v1/graph

Return the full graph index (nodes, edges, health scores).

**Response:**
```json
{
  "generatedAt": "2026-02-12T10:30:45Z",
  "graph": {
    "nodes": [
      {
        "id": "debugging",
        "type": "skill",
        "label": "Debugging",
        "description": "Systematic debugging approach",
        "status": "active",
        "tags": ["debugging"]
      },
      {
        "id": "action:scenario.status.show",
        "type": "action",
        "label": "Show Scenario Status",
        "description": "Show lifecycle status for one scenario",
        "status": "active",
        "tags": ["scenario"]
      }
    ],
    "edges": [
      {
        "from": "alice",
        "to": "debugging",
        "kind": "cli-read",
        "sourceFile": "README.md",
        "lineNumber": 42
      },
      {
        "from": "action:scenario.status.show",
        "to": "cli:vrooli",
        "kind": "action-command",
        "category": "scenario-cli",
        "command": "vrooli",
        "subcommand": "scenario",
        "sourceFile": "action.json"
      }
    ],
    "healthScores": [
      {
        "nodeId": "debugging",
        "score": 0.78,
        "factors": {
          "outgoing-edges": 0.4,
          "incoming-edges": 0.8,
          "code-usage": 1.0,
          "recent-activity": 0.5
        }
      }
    ]
  }
}
```

**Notes:**
- Lazily generated on first request; cached at `store/indexes/graph.index.json`
- Auto-invalidated when skills, Actions, or agents are created, updated, or deleted
- Action graph node IDs are namespaced as `action:<action-id>` to avoid collisions with raw skill/team/agent IDs

### POST /api/v1/graph/regenerate

Force a full graph rebuild, ignoring cached index.

**Response:** Same shape as `GET /api/v1/graph`.

### GET /api/v1/graph/orphans

Return skills with zero incoming edges (never referenced by any agent or skill).

**Response:** `[]Node`

### GET /api/v1/graph/skillless

Return agents that have no outgoing skill-reference edges.

**Response:** `[]Node`

### GET /api/v1/graph/empty-teams

Return teams with no `membership` edges (no members).

**Response:** `[]Node`

### GET /api/v1/graph/unaffiliated

Return agents not targeted by any `membership` edge (not in any team).

**Response:** `[]Node`

### GET /api/v1/graph/popular

Return the most referenced nodes, ranked by incoming edge count.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 10 | Maximum number of results |

**Response:** `[]Node`

### GET /api/v1/graph/cycles

Detect circular dependencies between skills using DFS.

**Response:** `[][]string` — each inner array is a cycle (list of node IDs).

### GET /api/v1/graph/health

Return health scores for all scored nodes.

**Response:** `[]HealthScore`
```json
[
  {
    "nodeId": "debugging",
    "score": 0.78,
    "factors": {
      "outgoing-edges": 0.4,
      "incoming-edges": 0.8,
      "code-usage": 1.0,
      "recent-activity": 0.5
    }
  }
]
```

### GET /api/v1/graph/health-config

Return the active graph health scoring configuration used for weighted scoring and CLI policy.

**Response:** `HealthConfig`

### PUT /api/v1/graph/health-config

Update graph health scoring configuration and immediately regenerate the graph index.

**Request Body:** `HealthConfig`

**Response:** `HealthConfig` (saved configuration)

**Errors:**
- `400` - Invalid config (e.g., negative weights, all-zero entity weights, invalid CLI policy ranges)
- `503` - Graph health config store not configured

### GET /api/v1/graph/nodes/{id}

Return a single node with its adjacent edges and health score.

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Node ID |

**Response:**
```json
{
  "node": {
    "id": "debugging",
    "type": "skill",
    "label": "Debugging"
  },
  "adjacentEdges": [
    { "from": "alice", "to": "debugging", "kind": "cli-read" }
  ],
  "healthScore": {
    "nodeId": "debugging",
    "score": 0.78,
    "factors": { ... }
  }
}
```

**Errors:**
- `404` - Node not found

### GET /api/v1/graph/nodes/{id}/edges

Return all edges touching a node (both inbound and outbound).

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Node ID |

**Response:** `[]Edge`

---

## Open Graph Metadata

[CODE: api/ogmeta/handlers.go]

### GET /api/v1/og-metadata

Fetch Open Graph metadata from a URL (for link previews).

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `url` | string | URL to fetch metadata from |

**Response:**
```json
{
  "url": "https://example.com/article",
  "title": "Article Title",
  "description": "Article description...",
  "image": "https://example.com/image.jpg",
  "siteName": "Example Site",
  "type": "article",
  "favicon": "https://example.com/favicon.ico"
}
```

**Notes:**
- Results cached for 15 minutes
- Timeout after 10 seconds
