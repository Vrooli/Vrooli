# Configuration Reference

Environment variables and configuration options for prompt-manager.

## Environment Variables

### API Server

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_ENABLED` | `false` | Enable skill testing through `resource-ollama gateway generate` |
| `OLLAMA_GATEWAY_BIN` | `resource-ollama` | Gateway command used for Ollama-backed skill testing and AI search embeddings |
| `STORE_DIR` | `../store` | Path to the store directory containing skills, agents, teams, and relations |
| `SQLITE_PATH` | (storage root) | Optional explicit SQLite database file path |
| `SQLITE_DB` | (storage root) | Alias for `SQLITE_PATH` for local debugging and tests |
| `QDRANT_URL` | `http://localhost:6333` | Qdrant vector database URL for AI search |
| `QDRANT_API_KEY` | (none) | API key for Qdrant authentication |
| `AI_SEARCH_COLLECTION` | `prompt-manager-skills` | Qdrant collection name for skill embeddings |
| `AI_SEARCH_ACTION_COLLECTION` | `prompt-manager-actions` | Qdrant collection name for Action embeddings |
| `AI_SEARCH_THRESHOLD` | `0.5` | Minimum similarity score for AI search results |

### CLI

| Variable | Default | Description |
|----------|---------|-------------|
| `PM_API_BASE` | (auto-detected) | Override API base URL |
| `NO_COLOR` | | Disable colored output when set |

## Port Allocation

Ports are dynamically allocated by the Vrooli lifecycle system. To find active ports:

```bash
# Check scenario status
cd scenarios/prompt-manager && make status

# Or check logs
make logs | grep "listening on"
```

## Database Configuration

Prompt-manager uses embedded SQLite for relational runtime data. By default the database is resolved through `api-core/storage` under the prompt-manager data root, for example `~/.vrooli/data/vrooli/prompt-manager/prompt-manager.db`. Use `SQLITE_PATH` or `SQLITE_DB` only when a test or local debugging session needs an explicit file.

**Required Tables:**
- `tags` - Tag definitions
- `skill_metrics` - Usage tracking
- `test_results` - LLM test history

Schemas are embedded beside their owning API packages and are initialized at API startup.

## Store Directory Structure

The storage system uses a per-entity file structure under the `store/` directory:

```
store/
├── skills/
│   ├── _pack-order.json        # Active pack precedence
│   └── packs/
│       ├── core/               # System skills
│       │   └── debugging/
│       │       ├── skill.json  # Metadata
│       │       ├── SKILL.md    # Content
│       │       └── history.jsonl
│       ├── local/              # User-created skills
│       └── drafts/             # Work-in-progress
├── agents/
│   └── agent-1/
│       └── agent.json
├── teams/
│   └── engineering/
│       ├── team.json
│       ├── roles.json
│       └── org-chart.json
├── actions/
│   ├── _pack-order.json
│   └── packs/
│       ├── core/
│       ├── local/
│       └── drafts/
├── relations/
│   └── team-member/
│       └── team-id__agent-1.json
└── indexes/                    # Generated (never hand-edit)
    ├── skills.index.json
    ├── agents.index.json
    └── teams.index.json
```

Store shapes are validated in Go, not by JSON Schema. `team.json` is enforced by
[`api/teamcontract/contract.go`](../../api/teamcontract/contract.go); member
`topics.json` and the operating-model contract by
[`api/memberflow/`](../../api/memberflow/).

### skill.json Format

```json
{
  "id": "debugging",
  "name": "Debugging",
  "description": "Systematic debugging approach",
  "modes": ["agent"],
  "tags": ["debugging"],
  "icon": "bug",
  "draft": false,
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-20T14:30:00Z"
}
```

### agent.json Format

```json
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
```

### action.json Format

Actions are typed wrappers over one Vrooli-controlled CLI command. See [DOC: docs/concepts/ACTIONS.md] for the full contract.

```json
{
  "kind": "action",
  "schemaVersion": 1,
  "id": "scenario.ui.screenshot",
  "name": "Take Scenario Screenshot",
  "description": "Capture a screenshot of a running scenario UI.",
  "status": "active",
  "owner": {
    "type": "scenario",
    "id": "prompt-manager"
  },
  "command": {
    "argv": ["vrooli", "scenario", "screenshot", "{{scenario}}"]
  },
  "inputs": {},
  "outputs": {},
  "permissions": {},
  "examples": []
}
```

Action commands should be argv-shaped and should not require shell parsing, pipelines, command separators, or embedded conditional logic.

### team.json Format

```json
{
  "id": "engineering",
  "displayName": "Engineering Team",
  "mission": "Build great software",
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-20T14:30:00Z"
}
```

## Optional Resources

### Ollama (Skill Testing)

Enable LLM-based skill testing:

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull a model
ollama pull llama3.2

# Enable gateway-backed skill testing
export OLLAMA_ENABLED=true
export OLLAMA_GATEWAY_BIN=resource-ollama
```

Test via CLI:
```bash
prompt-manager test run debugging --model=llama3.2
```

### Qdrant (Semantic Search)

Enable vector-based semantic search:

```bash
# Start Qdrant
docker run -p 6333:6333 qdrant/qdrant

# Set environment variable
export QDRANT_URL=http://localhost:6333
```

## App Configuration

Located at `initialization/configuration/app-config.json`:

```json
{
  "features": {
    "semanticSearch": false,
    "skillTesting": true,
    "versionHistory": true
  },
  "ui": {
    "defaultView": "grid",
    "theme": "system"
  },
  "limits": {
    "maxSkillSize": 100000,
    "maxVersionHistory": 50
  }
}
```

## Discover Ranking & Budget Configuration

`prompt-manager discover` exposes a small control surface for how it composes
skill results in **curated (plan-authoring) mode**. See
[`reference/discovery-pipeline.md`](discovery-pipeline.md) for the full ranking
model (invariants I1–I8) these levers govern.

### Ranking levers — `store/config/discover-ranking.json`

Git-tracked and hot-loadable. Missing file → the defaults below.

```json
{
  "topicGate": 0.55,
  "highConfidenceBar": 0.65,
  "maxIndividualsAbovePack": 3,
  "topicSkillCap": 12
}
```

| Lever | Default | Valid range | What it trades off |
|---|---|---|---|
| `topicGate` | 0.55 | `(AI_SEARCH_THRESHOLD, 1]` | How strong a topic must score for its whole skill **pack** to be force-included. Higher → fewer, more-relevant packs. Must exceed the skill threshold (a pack is a bigger commitment than one skill). |
| `highConfidenceBar` | 0.65 | `(0, 1]` | How strong a *direct* skill match must score to rank **above** the pack block. Higher → packs dominate the top; lower → strong direct matches surface first. |
| `maxIndividualsAbovePack` | 3 | `≥ 0` | Caps how many high-confidence direct matches sit above the pack block. (Ignored when no pack is selected — pure score ranking.) |
| `topicSkillCap` | 12 | `> 0` | Caps the total skills all selected packs contribute. Packs are added whole in relevance order; an overflowing pack is skipped so a smaller, more-relevant one can fit. |

Levers are validated on load (bounds + `topicGate > AI_SEARCH_THRESHOLD`); an
invalid file falls back to defaults. Tune **only from `discovery-metrics`**
(see the rubric in the discovery-pipeline doc), not from a hunch, and record any
change in a `swarm-manager records create --kind execute` entry.

**Levers deliberately *not* exposed:** the similarity threshold (it is
`AI_SEARCH_THRESHOLD`, an env var — §Environment Variables) and the per-result
budget unit (always the skill's own `SKILL.md` size — no transitive budget knob).

### Complexity budgets — `store/config/budgets.json`

Per-complexity character budgets for the returned set (git-tracked, hot-loadable;
missing file → small code defaults). Tiers must be positive, ascending, and
≤ 200,000.

```json
{ "minor": 50000, "moderate": 75000, "major": 100000, "architectural": 150000 }
```

## Graph Health Configuration

Graph health scoring controls are persisted in:

`store/config/graph-health.json`

This file is git-tracked and can be tuned directly or through the Graph View `Health` settings tab.

Schema:

```json
{
  "team": {
    "outgoingEdges": 1,
    "incomingEdges": 1,
    "codeUsage": 0.5,
    "recentActivity": 0.5,
    "skillContentLength": 0,
    "agentContextLoad": 0,
    "teamMemberCountBalance": 0.75,
    "teamRoleCoverage": 0.75
  },
  "agent": {
    "outgoingEdges": 1,
    "incomingEdges": 1,
    "codeUsage": 0.5,
    "recentActivity": 0.5,
    "skillContentLength": 0,
    "agentContextLoad": 0.75,
    "teamMemberCountBalance": 0,
    "teamRoleCoverage": 0
  },
  "skill": {
    "outgoingEdges": 1,
    "incomingEdges": 1,
    "codeUsage": 0.5,
    "recentActivity": 0.5,
    "skillContentLength": 0.75,
    "agentContextLoad": 0,
    "teamMemberCountBalance": 0,
    "teamRoleCoverage": 0
  },
  "action": {
    "outgoingEdges": 1,
    "incomingEdges": 1,
    "codeUsage": 0,
    "recentActivity": 0.5,
    "skillContentLength": 0,
    "agentContextLoad": 0,
    "teamMemberCountBalance": 0,
    "teamRoleCoverage": 0,
    "actionContract": 1,
    "actionCommand": 1,
    "actionExamples": 0.75,
    "actionOwner": 0.75
  },
  "cli": {
    "neutralCommands": ["vrooli"],
    "externalToolScore": 0,
    "scenarioFallbackScore": 0
  }
}
```

After changing values, regenerate the graph to recompute health:

```bash
prompt-manager graph regenerate
```

## Campaign Templates

Located at `initialization/configuration/campaign-templates.json`:

Predefined campaign types with colors and icons for organizing skills.

## Docker Configuration

The scenario can run in Docker via the lifecycle system:

```bash
# Build and start
cd scenarios/prompt-manager && make docker-start

# View logs
make docker-logs

# Stop
make docker-stop
```

## Health Checks

The API exposes health endpoints:

```bash
# Basic health
curl http://localhost:PORT/health

# Detailed health (includes database status)
curl http://localhost:PORT/api/v1/health
```

Response:
```json
{
  "status": "healthy",
  "version": "2.0.0",
  "checks": {
    "database": "healthy"
  }
}
```

## Logging

Logs are written to stdout. Control verbosity with:

| Variable | Values | Description |
|----------|--------|-------------|
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | Minimum log level |
| `LOG_FORMAT` | `text`, `json` | Output format |
