# API Reference

Complete documentation for the prompt-manager REST API.

**Base URL:** `http://localhost:{PORT}/api/v1`

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

### PUT /api/v1/experiments/{eid}

Update a draft experiment (name, hypothesis, arms).

**Note:** Only draft experiments can be updated.

### DELETE /api/v1/experiments/{eid}

Delete an experiment and its outcomes. Returns `204 No Content`.

### POST /api/v1/experiments/{eid}/start

Transition experiment from `draft` to `running`.

### POST /api/v1/experiments/{eid}/conclude

Conclude a running experiment.

**Request:**
```json
{
  "winnerVariantId": "concise-v1",
  "notes": "Equal quality, 40% faster execution time"
}
```

**Notes:**
- Winner must be one of the experiment's arms
- If winner is not `control`, the winner's content is promoted to SKILL.md
- Previous SKILL.md content is preserved in version history

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

### Variant-Aware Read (extension to POST /api/v1/skills/read)

When `experimentId` is included in a read request, the first resolved skill's content may be replaced by a variant selected via weighted random sampling.

**Additional request field:**
```json
{
  "experimentId": "exp-concise-test"
}
```

**Additional response field:**
```json
{
  "selectedVariantId": "concise-v1"
}
```

**Notes:**
- Experiment must be `running` and target the resolved skill
- `control` means the original SKILL.md was used (no content replacement)
- Variable substitution is applied to variant content as normal

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

### GET /api/v1/search/skills

Full-text search across skills.

**Query Parameters:**
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

### GET /api/v1/search/skills/content

Content-only search across skill bodies (line-level matches).

**Query Parameters:**
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

### POST /api/v1/search/ai

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
      "id": "react-coherence",
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

### GET /api/v1/search/ai/status

Returns AI search availability status.

### POST /api/v1/search/ai/reconcile

Reconcile the qdrant index with on-disk content. The reconciler uses a
content-hash diff (`payload_hash`) so unchanged items skip embedding
entirely; ghost points whose backing files are gone are deleted.

**Query parameters:**
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

**Live response (202 Accepted):** the kickoff is async; poll the status
endpoint for completion.

### GET /api/v1/search/ai/reconcile/status

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

### POST /api/v1/search/ai/reconcile/cancel

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

### POST /api/v1/discover

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

Test a skill with Ollama LLM.

**Prerequisites:** Ollama running with model loaded.

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
      "kind": "active-task-brief",
      "label": "Active Task Brief",
      "sourcePath": "teams/engineering/team.json#operatingContract.members.agent-1",
      "content": "# Active Task Brief\n\n..."
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
      "showDecisionLogGuidance": true,
      "showKnowledgeLogGuidance": true,
      "requireHandoff": true
    }
  },
  "execution": {
    "queuePolicy": "bounded-parallel",
    "maxConcurrentRuns": 2
  },
  "decisionMode": "approval",
  "operatingContract": {
    "schemaVersion": 1,
    "governance": {
      "decisionMode": "approval",
      "teamPendingCeiling": {
        "value": 12,
        "readOnlyWhenAtOrAbove": true
      },
      "supersession": {
        "requiredBeforeNewDecision": true,
        "allowedInReadOnlyMode": true,
        "replacementMustSetSupersedes": true
      }
    },
    "documents": {
      "planOfRecord": [],
      "notebooks": [],
      "sharedState": []
    },
    "decisionContexts": {},
    "knowledgeTopics": {},
    "members": {}
  }
}
```

**Required Fields:** `displayName`, `runtime`, `coordination`, `execution`, `operatingContract`

**Optional Fields:** `id` (auto-generated from displayName), `mission`, `decisionMode`

**runtime.mode Values:** `multi-process` - members run as separate heartbeat processes. `single-process` - one Claude Code lead session coordinates the team.

**coordination.pattern Values:** `independent`, `peer`, `leader-led`

**execution.queuePolicy Values:** `serialized`, `bounded-parallel`

**operatingContract.documents.sharedState:** Internal JSON field for team working state. Use final `kind` values only: `charter`, `task-board`, `decision-log`, `knowledge-log`, `handoff-log`, `working-register`, `rolling-snapshot`, `append-only-event-log`, `operator-input`. Agent-facing prompts render this category as team working state in the Storage Map.

**decisionMode Values:** `yolo` (default behavior) - agents can proceed without human approval. `approval` - agents must wait for human acceptance before acting on gated decisions.

**operatingContract:** Required structured source of truth for team/member operating policy. `operatingContract.governance.decisionMode` must match `decisionMode`. Heartbeat prompts render member-specific contract data inside the generated `Operating Policy` section, alongside top-level runtime, coordination, execution, and decision-mode policy. Prompt rendering fails if required policy is missing or invalid.

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
      "showDecisionLogGuidance": true,
      "showKnowledgeLogGuidance": true,
      "requireHandoff": true
    }
  },
  "execution": {
    "queuePolicy": "serialized",
    "maxConcurrentRuns": 1
  },
  "decisionMode": "approval"
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

## World Scale

[CODE: api/worldscale/handlers.go]

### GET /api/v1/world-scale

Get the current object scale configuration.

**Response:**
```json
{
  "agent": 1.0,
  "furniture": 1.0,
  "decoration": 1.0,
  "overlay": 1.0
}
```

**Notes:**
- Returns default values (all 1.0) if the config file doesn't exist yet
- Config persists in `store/world-scale.json`

### PUT /api/v1/world-scale

Update the object scale configuration.

**Request Body:**
```json
{
  "agent": 1.2,
  "furniture": 0.8,
  "decoration": 1.5,
  "overlay": 1.0
}
```

**Validation:**
- All values must be between 0.1 and 3.0

**Response:** Updated config object.

**Errors:**
- `400` - Value out of range or invalid JSON

---

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
