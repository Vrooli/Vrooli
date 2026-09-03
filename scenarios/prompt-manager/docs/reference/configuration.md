# Configuration Reference

Environment variables and configuration options for prompt-manager.

## Environment Variables

### API Server

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_ENABLED` | `false` | Enable skill testing through `resource-ollama gateway generate` |
| `OLLAMA_GATEWAY_BIN` | `resource-ollama` | Gateway command used for Ollama-backed skill testing and AI search embeddings |
| `STORE_DIR` | `../store` | Path to the store directory containing skills, agents, teams, and relations |
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

Prompt-manager uses embedded SQLite for relational runtime data. By default the database is resolved through `api-core/storage` under the prompt-manager data root, for example `~/.vrooli/data/vrooli/prompt-manager/prompt-manager.db`. There is no environment variable for the file location. A test or local debugging session that needs an explicit file passes the path as an argument to `storage.SQLiteDSNAt`; to relocate the whole storage tree, set `VROOLI_STORAGE_ROOT`, which is scenario-agnostic and so cannot redirect one scenario at another's database.

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

Located at `api/internal/<domain>/configuration/app-config.json`:

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
change with `vrooli-memory note --kind work-record` including trigger, approach,
evidence, and outcome.

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

Located at `api/internal/<domain>/configuration/campaign-templates.json`:

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

<!-- world-tuning:begin -->
## World tuning levers

Source: `ui/src/world/config/world.tuning.json`, validated by `tuning.schema.ts`.
Every behaviour number the world uses is one of these levers; the sim and scene
code carry no literals. Edit the JSON, keep the value inside its bounds, and run
`pnpm world:tuning-docs` to refresh this table (a test fails when it is stale).
In development the HUD settings panel has a Levers tab that edits these live.

### `version`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `version` | const 1 | — | `1` | Tuning file format version (integer) |

### `sim`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `sim.tickSeconds` | number | min 0.01, max 1 | `0.1` | Fixed simulation step; every rule advances by this much (seconds) |
| `sim.walkSpeed` | number | min 0.1, max 10 | `2.2` | Ground speed of a walking actor (metres per second) |
| `sim.hurrySpeed` | number | min 0.1, max 12 | `3.2` | Ground speed when a run starts and the actor heads to its desk (metres per second) |
| `sim.turnRateRadPerSec` | number | min 0.5, max 30 | `9` | How fast an actor turns to face its heading (radians per second) |
| `sim.arriveRadius` | number | min 0.05, max 2 | `0.3` | Distance from a target at which an actor counts as arrived (metres) |
| `sim.accelSeconds` | number | min 0.05, max 5 | `0.5` | Time for an actor to ramp between rest and its target speed (seconds) |
| `sim.gatherLeadSeconds` | number | min 0, max 3600 | `120` | How long before a scheduled heartbeat a team walks to its table (seconds) |
| `sim.gatherWindowSeconds` | number | min 0, max 7200 | `900` | How long a gathered team waits before giving up and going idle (seconds) |
| `sim.failedAckSeconds` | number | min 0, max 86400 | `900` | How long an actor stays in Failed before returning to Idle on its own (seconds) |
| `sim.eventsRing` | integer | min 8, max 1024 | `64` | How many recent world events the state keeps for the ticker (count) |
| `sim.maxReplansPerTick` | integer | min 1, max 64 | `6` | Upper bound on A* replans per tick so pathing never blows the tick budget (count) |
| `sim.pathCacheSize` | integer | min 16, max 4096 | `512` | Number of cached paths kept per world (count) |
| `sim.idle.rollIntervalSeconds` | number | min 0.5, max 120 | `5` | How often an idle actor rolls for a new idle activity (seconds) |
| `sim.idle.weights.rest` | number | min 0, max 100 | `70` | Weight of resting at home: the desk seat, or the commons for an unassigned agent (relative weight) |
| `sim.idle.weights.wander` | number | min 0, max 100 | `12` | Weight of an outing to a random commons spot (relative weight) |
| `sim.idle.weights.socialize` | number | min 0, max 100 | `10` | Weight of pairing up with another idle actor on the commons (relative weight) |
| `sim.idle.weights.sit` | number | min 0, max 100 | `8` | Weight of taking a free campfire seat (relative weight) |
| `sim.idle.maxMoversRatio` | number | min 0, max 1 | `0.35` | Largest share of idle actors allowed to be walking at once (0..1) |
| `sim.idle.spacing` | number | min 0.5, max 5 | `1.2` | Minimum distance between an idle actor and every other actor when a commons spot is chosen (metres) |
| `sim.idle.spacingAttempts` | integer | min 1, max 32 | `8` | Random commons spots tried before the spacing rule is given up for that roll (count) |
| `sim.idle.socializeSeconds.min` | number | min 1, max 600 | `8` | Duration of a socializing pair lower bound (seconds) |
| `sim.idle.socializeSeconds.max` | number | min 1, max 600 | `20` | Duration of a socializing pair upper bound (seconds) |
| `sim.idle.sitSeconds.min` | number | min 1, max 3600 | `12` | Duration of a sit at the campfire lower bound (seconds) |
| `sim.idle.sitSeconds.max` | number | min 1, max 3600 | `40` | Duration of a sit at the campfire upper bound (seconds) |
| `sim.idle.restSeconds.min` | number | min 1, max 600 | `10` | Duration of standing still before the next roll lower bound (seconds) |
| `sim.idle.restSeconds.max` | number | min 1, max 600 | `30` | Duration of standing still before the next roll upper bound (seconds) |
| `sim.idle.socializeGap` | number | min 0.5, max 5 | `1.4` | Distance between two socializing actors (metres) |

### `layout`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `layout.cellSize` | number | min 0.1, max 2 | `0.5` | Navigation grid cell size (metres) |
| `layout.roomWidth` | number | min 3, max 30 | `8` | Width (x) of a team room (metres) |
| `layout.roomDepth` | number | min 3, max 30 | `6` | Depth (z) of a team room (metres) |
| `layout.roomGap` | number | min 0, max 20 | `3` | Gap between neighbouring rooms (metres) |
| `layout.maxRoomsPerRow` | integer | min 1, max 12 | `4` | Rooms per row before the grid wraps (count) |
| `layout.deskPitch` | number | min 0.8, max 5 | `1.7` | Distance between neighbouring desks along the back wall (metres) |
| `layout.deskInset` | number | min 0.2, max 5 | `1.1` | Distance from the back wall to the desk row (metres) |
| `layout.deskSeatOffset` | number | min 0.2, max 3 | `0.8` | Distance in front of a desk where its owner stands (metres) |
| `layout.tableRadius` | number | min 0.3, max 4 | `0.9` | Radius of the team table prop footprint (metres) |
| `layout.tableSeatRadius` | number | min 0.5, max 6 | `1.5` | Radius of the seat ring around a team table (metres) |
| `layout.tableSeats` | integer | min 2, max 16 | `6` | Seats around a team table (count) |
| `layout.commonsRadius` | number | min 2, max 20 | `5.5` | Radius of the commons clearing (metres) |
| `layout.commonsSeatRadius` | number | min 0.5, max 10 | `2.2` | Radius of the seat ring around the campfire (metres) |
| `layout.commonsSeats` | integer | min 2, max 24 | `8` | Seats around the campfire (count) |
| `layout.commonsGap` | number | min 0, max 20 | `4` | Gap between the commons edge and the first room row (metres) |
| `layout.clearingRadius` | number | min 0, max 20 | `3.5` | No tree spawns within this distance of a room or the hero camera (metres) |
| `layout.slabMargin` | number | min 0, max 20 | `4` | Empty slab border around the generated layout (metres) |
| `layout.minSlabWidth` | number | min 5, max 200 | `26` | Smallest slab width even for an empty team graph (metres) |
| `layout.minSlabDepth` | number | min 5, max 200 | `20` | Smallest slab depth even for an empty team graph (metres) |
| `layout.wallHeight` | number | min 0, max 3 | `0.7` | Height of the low wall around a room (metres) |
| `layout.boardOffset` | number | min 0, max 20 | `4` | Distance from the commons centre to the runs board (metres) |
| `layout.outlineRimSamples` | integer | min 4, max 64 | `12` | Points sampled around the commons rim for the outline the camera frames (count) |
| `layout.treeDensity` | number | min 0, max 2 | `0.035` | Trees per square metre of free park ground (density) |
| `layout.treeMargin` | number | min 0, max 10 | `2.5` | Trees keep this distance from the slab edge and each other (metres) |
| `layout.treeAttemptsPerTree` | integer | min 1, max 64 | `8` | Rejection-sampling attempts per wanted tree before the scatter gives up (count) |

### `camera`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `camera.fov` | number | min 10, max 90 | `38` | Vertical field of view (degrees) |
| `camera.near` | number | min 0.01, max 10 | `0.5` | Near clip plane (metres) |
| `camera.far` | number | min 10, max 2000 | `400` | Far clip plane (metres) |
| `camera.polarMinDeg` | number | min 0, max 89 | `30` | Steepest allowed camera angle from straight above (degrees) |
| `camera.polarMaxDeg` | number | min 1, max 90 | `64` | Shallowest allowed camera angle from straight above (degrees) |
| `camera.azimuthRangeDeg` | number | min 0, max 180 | `35` | Orbit allowed either side of the hero azimuth (degrees) |
| `camera.minDistance` | number | min 1, max 100 | `6` | Closest dolly distance (metres) |
| `camera.maxDistance` | number | min 2, max 500 | `140` | Farthest dolly distance (metres) |
| `camera.introSeconds` | number | min 0, max 10 | `2` | Length of the establishing-to-hero dolly on load (seconds) |
| `camera.smoothTime` | number | min 0.01, max 3 | `0.35` | Camera-controls smoothing time for every move (seconds) |
| `camera.frameFill` | number | min 0, max 1 | `0.9` | Share of the viewport the layout outline fills at distanceFactor 1; poses scale from this (0..1) |
| `camera.focusPadding` | number | min 1, max 5 | `1.6` | fitToBox padding multiplier when focusing an actor (multiplier) |
| `camera.focusDistance` | number | min 1, max 50 | `7` | Dolly distance after focusing an actor (metres) |
| `camera.minClearance` | number | min 0, max 20 | `2` | The first frame must have no geometry closer than this to the camera (metres) |
| `camera.keyOrbitDegPerSec` | number | min 1, max 360 | `55` | Orbit speed for keyboard arrows (degrees) |
| `camera.keyDollyPerSec` | number | min 0.1, max 100 | `10` | Dolly speed for keyboard +/- (metres per second) |

### `lighting`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `lighting.keyLight.elevationDeg` | number | min 0, max 90 | `52` | Key light elevation above the slab (degrees) |
| `lighting.keyLight.azimuthDeg` | number | min -180, max 180 | `-38` | Key light azimuth around the slab (degrees) |
| `lighting.keyLight.shadowBias` | number | min -0.01, max 0.01 | `-0.0004` | Shadow map depth bias (depth units) |
| `lighting.keyLight.shadowNormalBias` | number | min 0, max 0.5 | `0.02` | Shadow map normal bias (metres) |
| `lighting.periodHours.dawn.from` | number | min 0, max 24 | `5` | Band start (hour of day) |
| `lighting.periodHours.dawn.to` | number | min 0, max 24 | `8` | Band end (hour of day) |
| `lighting.periodHours.day.from` | number | min 0, max 24 | `8` | Band start (hour of day) |
| `lighting.periodHours.day.to` | number | min 0, max 24 | `17` | Band end (hour of day) |
| `lighting.periodHours.dusk.from` | number | min 0, max 24 | `17` | Band start (hour of day) |
| `lighting.periodHours.dusk.to` | number | min 0, max 24 | `20` | Band end (hour of day) |
| `lighting.periodHours.night.from` | number | min 0, max 24 | `20` | Band start (hour of day) |
| `lighting.periodHours.night.to` | number | min 0, max 24 | `5` | Band end (hour of day) |
| `lighting.periods.dawn.exposure` | number | min 0, max 4 | `0.8` | Tone-mapping exposure (multiplier) |
| `lighting.periods.dawn.envIntensity` | number | min 0, max 4 | `0.35` | Environment map intensity (multiplier) |
| `lighting.periods.dawn.keyIntensity` | number | min 0, max 20 | `2.6` | Directional key light intensity (physical units) |
| `lighting.periods.dawn.keyColor` | string | — | `"#ffd2a8"` | Key light colour (hex colour) |
| `lighting.periods.dawn.ambientIntensity` | number | min 0, max 4 | `0.2` | Hemisphere ambient intensity (multiplier) |
| `lighting.periods.dawn.fogColor` | string | — | `"#e9c9c0"` | Height fog colour (hex colour) |
| `lighting.periods.dawn.fogNear` | number | min 0, max 10 | `1.1` | Fog starts at this multiple of the slab fit distance from the camera (multiplier) |
| `lighting.periods.dawn.fogFar` | number | min 0.1, max 20 | `3.2` | Fog is complete at this multiple of the slab fit distance (multiplier) |
| `lighting.periods.dawn.skyIntensity` | number | min 0, max 4 | `0.35` | Brightness of the HDRI sky drawn behind the diorama (multiplier) |
| `lighting.periods.dawn.skyBlur` | number | min 0, max 1 | `0.35` | Blur applied to the HDRI sky background; 0 is sharp, 1 is a soft gradient (0..1) |
| `lighting.periods.dawn.sunElevationDeg` | number | min -30, max 90 | `8` | Sun elevation used to tilt the key light for the period (degrees) |
| `lighting.periods.dawn.lampEmissive` | number | min 0, max 10 | `1.2` | Emissive intensity of lamps and the campfire; above 1 blooms (multiplier) |
| `lighting.periods.dawn.backgroundColor` | string | — | `"#f4d9d0"` | Canvas clear colour behind the sky (hex colour) |
| `lighting.periods.day.exposure` | number | min 0, max 4 | `0.85` | Tone-mapping exposure (multiplier) |
| `lighting.periods.day.envIntensity` | number | min 0, max 4 | `0.45` | Environment map intensity (multiplier) |
| `lighting.periods.day.keyIntensity` | number | min 0, max 20 | `3.6` | Directional key light intensity (physical units) |
| `lighting.periods.day.keyColor` | string | — | `"#fff1d6"` | Key light colour (hex colour) |
| `lighting.periods.day.ambientIntensity` | number | min 0, max 4 | `0.22` | Hemisphere ambient intensity (multiplier) |
| `lighting.periods.day.fogColor` | string | — | `"#cfe0f2"` | Height fog colour (hex colour) |
| `lighting.periods.day.fogNear` | number | min 0, max 10 | `1.3` | Fog starts at this multiple of the slab fit distance from the camera (multiplier) |
| `lighting.periods.day.fogFar` | number | min 0.1, max 20 | `4.5` | Fog is complete at this multiple of the slab fit distance (multiplier) |
| `lighting.periods.day.skyIntensity` | number | min 0, max 4 | `0.55` | Brightness of the HDRI sky drawn behind the diorama (multiplier) |
| `lighting.periods.day.skyBlur` | number | min 0, max 1 | `0.1` | Blur applied to the HDRI sky background; 0 is sharp, 1 is a soft gradient (0..1) |
| `lighting.periods.day.sunElevationDeg` | number | min -30, max 90 | `55` | Sun elevation used to tilt the key light for the period (degrees) |
| `lighting.periods.day.lampEmissive` | number | min 0, max 10 | `0` | Emissive intensity of lamps and the campfire; above 1 blooms (multiplier) |
| `lighting.periods.day.backgroundColor` | string | — | `"#cfe6fb"` | Canvas clear colour behind the sky (hex colour) |
| `lighting.periods.dusk.exposure` | number | min 0, max 4 | `0.75` | Tone-mapping exposure (multiplier) |
| `lighting.periods.dusk.envIntensity` | number | min 0, max 4 | `0.3` | Environment map intensity (multiplier) |
| `lighting.periods.dusk.keyIntensity` | number | min 0, max 20 | `2` | Directional key light intensity (physical units) |
| `lighting.periods.dusk.keyColor` | string | — | `"#ff9a5c"` | Key light colour (hex colour) |
| `lighting.periods.dusk.ambientIntensity` | number | min 0, max 4 | `0.18` | Hemisphere ambient intensity (multiplier) |
| `lighting.periods.dusk.fogColor` | string | — | `"#e0a98f"` | Height fog colour (hex colour) |
| `lighting.periods.dusk.fogNear` | number | min 0, max 10 | `1` | Fog starts at this multiple of the slab fit distance from the camera (multiplier) |
| `lighting.periods.dusk.fogFar` | number | min 0.1, max 20 | `3` | Fog is complete at this multiple of the slab fit distance (multiplier) |
| `lighting.periods.dusk.skyIntensity` | number | min 0, max 4 | `0.3` | Brightness of the HDRI sky drawn behind the diorama (multiplier) |
| `lighting.periods.dusk.skyBlur` | number | min 0, max 1 | `0.4` | Blur applied to the HDRI sky background; 0 is sharp, 1 is a soft gradient (0..1) |
| `lighting.periods.dusk.sunElevationDeg` | number | min -30, max 90 | `4` | Sun elevation used to tilt the key light for the period (degrees) |
| `lighting.periods.dusk.lampEmissive` | number | min 0, max 10 | `2.2` | Emissive intensity of lamps and the campfire; above 1 blooms (multiplier) |
| `lighting.periods.dusk.backgroundColor` | string | — | `"#f0b28e"` | Canvas clear colour behind the sky (hex colour) |
| `lighting.periods.night.exposure` | number | min 0, max 4 | `0.5` | Tone-mapping exposure (multiplier) |
| `lighting.periods.night.envIntensity` | number | min 0, max 4 | `0.08` | Environment map intensity (multiplier) |
| `lighting.periods.night.keyIntensity` | number | min 0, max 20 | `0.4` | Directional key light intensity (physical units) |
| `lighting.periods.night.keyColor` | string | — | `"#8fa9ff"` | Key light colour (hex colour) |
| `lighting.periods.night.ambientIntensity` | number | min 0, max 4 | `0.1` | Hemisphere ambient intensity (multiplier) |
| `lighting.periods.night.fogColor` | string | — | `"#101a33"` | Height fog colour (hex colour) |
| `lighting.periods.night.fogNear` | number | min 0, max 10 | `0.9` | Fog starts at this multiple of the slab fit distance from the camera (multiplier) |
| `lighting.periods.night.fogFar` | number | min 0.1, max 20 | `2.6` | Fog is complete at this multiple of the slab fit distance (multiplier) |
| `lighting.periods.night.skyIntensity` | number | min 0, max 4 | `0.05` | Brightness of the HDRI sky drawn behind the diorama (multiplier) |
| `lighting.periods.night.skyBlur` | number | min 0, max 1 | `0.6` | Blur applied to the HDRI sky background; 0 is sharp, 1 is a soft gradient (0..1) |
| `lighting.periods.night.sunElevationDeg` | number | min -30, max 90 | `-12` | Sun elevation used to tilt the key light for the period (degrees) |
| `lighting.periods.night.lampEmissive` | number | min 0, max 10 | `4` | Emissive intensity of lamps and the campfire; above 1 blooms (multiplier) |
| `lighting.periods.night.backgroundColor` | string | — | `"#0b1226"` | Canvas clear colour behind the sky (hex colour) |

### `labels`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `labels.budget` | integer | min 0, max 200 | `24` | Maximum labels drawn at once before clustering (count) |
| `labels.collapseDistance` | number | min 1, max 500 | `34` | Camera distance past which room labels collapse into one count label (metres) |
| `labels.fontSize` | number | min 0.05, max 2 | `0.34` | SDF label height in world units (metres) |
| `labels.offsetY` | number | min 0, max 5 | `1.45` | Label height above the actor origin (metres) |
| `labels.minScreenPx` | number | min 4, max 64 | `11` | Labels never render smaller than this on screen (pixels) |
| `labels.maxScreenPx` | number | min 4, max 128 | `20` | Labels never render larger than this on screen (pixels) |
| `labels.paddingPx` | number | min 0, max 32 | `4` | Collision padding around a projected label (pixels) |

### `actor`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `actor.bodyRadius` | number | min 0.1, max 2 | `0.42` | Slime body radius at rest (metres) |
| `actor.breathAmplitude` | number | min 0, max 0.5 | `0.035` | Idle breathing scale swing (scale units) |
| `actor.breathHz` | number | min 0.05, max 5 | `0.55` | Idle breathing rate (hertz) |
| `actor.hopHeight` | number | min 0, max 2 | `0.24` | Peak height of a locomotion hop (metres) |
| `actor.hopHz` | number | min 0.2, max 10 | `2.4` | Hops per second while walking (hertz) |
| `actor.squashOnLand` | number | min 0.3, max 1 | `0.84` | Vertical scale at the moment of landing (scale units) |
| `actor.squashRecoverPerSec` | number | min 0.5, max 60 | `6` | How fast the landing squash relaxes back to 1 (per second) |
| `actor.wobbleIntensity` | number | min 0, max 0.5 | `0.05` | Vertex wobble amplitude from the slime shader (metres) |
| `actor.blinkIntervalSeconds.min` | number | min 0.2, max 60 | `2.5` | Interval between blinks lower bound (seconds) |
| `actor.blinkIntervalSeconds.max` | number | min 0.2, max 60 | `6.5` | Interval between blinks upper bound (seconds) |
| `actor.blinkSeconds` | number | min 0.02, max 1 | `0.12` | Length of one blink (seconds) |
| `actor.emoteSeconds` | number | min 0.2, max 10 | `1.8` | How long an emote burst stays visible (seconds) |
| `actor.seatedScale` | number | min 0.5, max 1 | `0.9` | Body scale while seated (scale units) |
| `actor.equipmentTiers` | array<integer> | length 5 | `[0,3,8,15,25]` | Skill counts at which equipment upgrades: none, paper, folder, briefcase, backpack (counts) |
| `actor.look.bodySquashY` | number | min 0, max 1 | `0.82` | Resting vertical scale of the slime body sphere; below 1 makes a blob (0..1) |
| `actor.look.eyeRadius` | number | min 0, max 1 | `0.075` | Eye radius as a fraction of the body radius (0..1) |
| `actor.look.eyeSpacing` | number | min 0, max 1 | `0.16` | Half distance between the eyes as a fraction of the body radius (0..1) |
| `actor.look.eyeHeight` | number | min 0, max 1 | `0.55` | Eye height above the body centre as a fraction of the body radius (0..1) |
| `actor.look.eyeForward` | number | min 0, max 1 | `0.85` | Eye distance forward of the body centre as a fraction of the body radius (0..1) |
| `actor.look.blinkScaleY` | number | min 0, max 1 | `0.12` | Vertical eye scale while blinking (0..1) |
| `actor.look.mouthWidth` | number | min 0, max 1 | `0.16` | Mouth width as a fraction of the body radius (0..1) |
| `actor.look.mouthHeight` | number | min 0, max 1 | `0.05` | Mouth height as a fraction of the body radius (0..1) |
| `actor.look.mouthForward` | number | min 0, max 1 | `0.9` | Mouth distance forward of the body centre as a fraction of the body radius (0..1) |
| `actor.look.mouthDrop` | number | min 0, max 1 | `0.3` | Mouth height below the eye line as a fraction of the body radius (0..1) |
| `actor.look.earSize` | number | min 0, max 1 | `0.14` | Ear nub size as a fraction of the body radius (0..1) |
| `actor.look.earHeight` | number | min 0, max 1 | `0.8` | Ear height above the body centre as a fraction of the body radius (0..1) |
| `actor.look.earSpread` | number | min 0, max 1 | `0.55` | Ear sideways offset as a fraction of the body radius (0..1) |
| `actor.look.equipmentScale` | number | min 0, max 1 | `0.22` | Equipment prop size as a fraction of the body radius per tier (0..1) |
| `actor.look.equipmentBack` | number | min 0, max 1 | `0.75` | Equipment distance behind the body centre as a fraction of the body radius (0..1) |
| `actor.look.equipmentHeight` | number | min 0, max 1 | `0.45` | Equipment height above the body centre as a fraction of the body radius (0..1) |
| `actor.look.markerHeight` | number | min 0, max 5 | `1.35` | Status marker height above the actor origin (metres) |
| `actor.look.markerRadius` | number | min 0.02, max 1 | `0.16` | Status marker ring radius (metres) |
| `actor.look.emoteRise` | number | min 0, max 3 | `0.7` | How far an emote sprite rises over its lifetime (metres) |
| `actor.look.emoteSize` | number | min 0.05, max 2 | `0.42` | Emote sprite size (metres) |
| `actor.look.emoteHeight` | number | min 0, max 5 | `1.3` | Emote start height above the actor origin (metres) |
| `actor.look.messageTtlSeconds` | number | min 1, max 120 | `12` | How long a speech bubble stays readable (seconds) |

### `quality`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `quality.defaultProfile` | "low" \| "medium" \| "high" \| "ultra" | — | `"medium"` | Profile used before the governor or the user picks one (profile id) |
| `quality.degradedRatio` | number | min 0, max 1 | `0.72` | Fraction of frameCapFps below which the governor steps down while auto is on (0..1) |
| `quality.recoverRatio` | number | min 0, max 1 | `0.92` | Fraction of frameCapFps above which the governor steps up while auto is on (0..1) |
| `quality.monitorFlipflops` | integer | min 1, max 20 | `3` | Consecutive up/down flips after which the governor stops adjusting (count) |
| `quality.profiles.low.dpr` | number | min 0.5, max 3 | `1` | Device pixel ratio cap (multiplier) |
| `quality.profiles.low.shadows` | boolean | — | `false` | Directional shadow map on or off (flag) |
| `quality.profiles.low.shadowMapSize` | integer | min 256, max 8192 | `512` | Shadow map resolution (pixels, square) (count) |
| `quality.profiles.low.ao` | boolean | — | `false` | N8AO ambient occlusion pass on or off (flag) |
| `quality.profiles.low.bloom` | boolean | — | `false` | Selective bloom pass on or off (flag) |
| `quality.profiles.low.msaa` | integer | min 0, max 8 | `0` | Multisample anti-aliasing samples on the composer target; 0 disables (samples) (count) |
| `quality.profiles.low.labelBudget` | integer | min 0, max 200 | `8` | Label budget override for this profile (count) |
| `quality.profiles.low.frameCapFps` | integer | min 15, max 240 | `60` | Frame rate this profile is designed for; the governor derives its degraded threshold from it (count) |
| `quality.profiles.low.wobble` | boolean | — | `false` | Slime vertex wobble on or off (flag) |
| `quality.profiles.low.clouds` | boolean | — | `false` | Volumetric clouds on or off (flag) |
| `quality.profiles.medium.dpr` | number | min 0.5, max 3 | `1` | Device pixel ratio cap (multiplier) |
| `quality.profiles.medium.shadows` | boolean | — | `true` | Directional shadow map on or off (flag) |
| `quality.profiles.medium.shadowMapSize` | integer | min 256, max 8192 | `1024` | Shadow map resolution (pixels, square) (count) |
| `quality.profiles.medium.ao` | boolean | — | `false` | N8AO ambient occlusion pass on or off (flag) |
| `quality.profiles.medium.bloom` | boolean | — | `true` | Selective bloom pass on or off (flag) |
| `quality.profiles.medium.msaa` | integer | min 0, max 8 | `2` | Multisample anti-aliasing samples on the composer target; 0 disables (samples) (count) |
| `quality.profiles.medium.labelBudget` | integer | min 0, max 200 | `16` | Label budget override for this profile (count) |
| `quality.profiles.medium.frameCapFps` | integer | min 15, max 240 | `60` | Frame rate this profile is designed for; the governor derives its degraded threshold from it (count) |
| `quality.profiles.medium.wobble` | boolean | — | `true` | Slime vertex wobble on or off (flag) |
| `quality.profiles.medium.clouds` | boolean | — | `false` | Volumetric clouds on or off (flag) |
| `quality.profiles.high.dpr` | number | min 0.5, max 3 | `1` | Device pixel ratio cap (multiplier) |
| `quality.profiles.high.shadows` | boolean | — | `true` | Directional shadow map on or off (flag) |
| `quality.profiles.high.shadowMapSize` | integer | min 256, max 8192 | `2048` | Shadow map resolution (pixels, square) (count) |
| `quality.profiles.high.ao` | boolean | — | `true` | N8AO ambient occlusion pass on or off (flag) |
| `quality.profiles.high.bloom` | boolean | — | `true` | Selective bloom pass on or off (flag) |
| `quality.profiles.high.msaa` | integer | min 0, max 8 | `4` | Multisample anti-aliasing samples on the composer target; 0 disables (samples) (count) |
| `quality.profiles.high.labelBudget` | integer | min 0, max 200 | `24` | Label budget override for this profile (count) |
| `quality.profiles.high.frameCapFps` | integer | min 15, max 240 | `60` | Frame rate this profile is designed for; the governor derives its degraded threshold from it (count) |
| `quality.profiles.high.wobble` | boolean | — | `true` | Slime vertex wobble on or off (flag) |
| `quality.profiles.high.clouds` | boolean | — | `true` | Volumetric clouds on or off (flag) |
| `quality.profiles.ultra.dpr` | number | min 0.5, max 3 | `1.5` | Device pixel ratio cap (multiplier) |
| `quality.profiles.ultra.shadows` | boolean | — | `true` | Directional shadow map on or off (flag) |
| `quality.profiles.ultra.shadowMapSize` | integer | min 256, max 8192 | `4096` | Shadow map resolution (pixels, square) (count) |
| `quality.profiles.ultra.ao` | boolean | — | `true` | N8AO ambient occlusion pass on or off (flag) |
| `quality.profiles.ultra.bloom` | boolean | — | `true` | Selective bloom pass on or off (flag) |
| `quality.profiles.ultra.msaa` | integer | min 0, max 8 | `8` | Multisample anti-aliasing samples on the composer target; 0 disables (samples) (count) |
| `quality.profiles.ultra.labelBudget` | integer | min 0, max 200 | `40` | Label budget override for this profile (count) |
| `quality.profiles.ultra.frameCapFps` | integer | min 15, max 240 | `120` | Frame rate this profile is designed for; the governor derives its degraded threshold from it (count) |
| `quality.profiles.ultra.wobble` | boolean | — | `true` | Slime vertex wobble on or off (flag) |
| `quality.profiles.ultra.clouds` | boolean | — | `true` | Volumetric clouds on or off (flag) |

### `data`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `data.pollIntervalMs` | number | min 500, max 120000 | `5000` | Fallback poll cadence when the feed stream is unavailable (milliseconds) |
| `data.fallbackAfterMs` | number | min 100, max 120000 | `4000` | How long a silent stream is tolerated before polling starts (milliseconds) |
| `data.reconnectBaseMs` | number | min 100, max 60000 | `1000` | First reconnect delay after a stream error (milliseconds) |
| `data.reconnectMaxMs` | number | min 100, max 600000 | `30000` | Reconnect backoff ceiling (milliseconds) |
| `data.snapshotStaleMs` | number | min 1000, max 600000 | `15000` | A snapshot older than this is treated as stale by the HUD (milliseconds) |

### `editor`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `editor.snap` | number | min 0.05, max 5 | `0.5` | Drag snapping grid in edit mode (metres) |
| `editor.maxHistory` | integer | min 1, max 500 | `50` | Undo history depth (count) |
| `editor.saveDebounceMs` | number | min 0, max 30000 | `800` | Debounce before an edit is persisted (milliseconds) |
| `editor.aerialPolarDeg` | number | min 0, max 89 | `22` | Camera angle from straight above while editing (degrees) |

### `budgets`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `budgets.goldenThreshold` | number | min 0, max 1 | `0.015` | Fraction of pixels allowed to differ from a golden before the smoke tool fails (0..1) |
| `budgets.propTriangles` | integer | min 1, max 100000 | `4000` | Maximum triangles for one baked prop GLB (count) |
| `budgets.actorDrawCalls` | integer | min 1, max 100 | `4` | Draw calls the actor layer may add on top of the set (count) |
| `budgets.emptyStageDrawCalls` | integer | min 1, max 500 | `30` | Draw calls allowed for the empty slab and environment (count) |
| `budgets.periodPixelDelta` | number | min 0, max 1 | `0.25` | Minimum fraction of pixels that must differ between day and night goldens (0..1) |
| `budgets.framing.minFill` | number | min 0, max 1 | `0.7` | Smallest share of the viewport the layout outline may occupy on the hero pose (0..1) |
| `budgets.framing.maxFill` | number | min 0, max 1 | `0.97` | Largest share of the viewport the layout outline may occupy on the hero pose (0..1) |
| `budgets.scenes.park.low.drawCalls` | integer | min 1, max 5000 | `40` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.park.low.triangles` | integer | min 1, max 50000000 | `90000` | Maximum triangles per frame (count) |
| `budgets.scenes.park.low.p95Ms` | number | min 1, max 200 | `18` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.park.medium.drawCalls` | integer | min 1, max 5000 | `80` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.park.medium.triangles` | integer | min 1, max 50000000 | `160000` | Maximum triangles per frame (count) |
| `budgets.scenes.park.medium.p95Ms` | number | min 1, max 200 | `18` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.park.high.drawCalls` | integer | min 1, max 5000 | `110` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.park.high.triangles` | integer | min 1, max 50000000 | `200000` | Maximum triangles per frame (count) |
| `budgets.scenes.park.high.p95Ms` | number | min 1, max 200 | `18` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.park.ultra.drawCalls` | integer | min 1, max 5000 | `110` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.park.ultra.triangles` | integer | min 1, max 50000000 | `200000` | Maximum triangles per frame (count) |
| `budgets.scenes.park.ultra.p95Ms` | number | min 1, max 200 | `24` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.office.low.drawCalls` | integer | min 1, max 5000 | `40` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.office.low.triangles` | integer | min 1, max 50000000 | `90000` | Maximum triangles per frame (count) |
| `budgets.scenes.office.low.p95Ms` | number | min 1, max 200 | `18` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.office.medium.drawCalls` | integer | min 1, max 5000 | `80` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.office.medium.triangles` | integer | min 1, max 50000000 | `160000` | Maximum triangles per frame (count) |
| `budgets.scenes.office.medium.p95Ms` | number | min 1, max 200 | `18` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.office.high.drawCalls` | integer | min 1, max 5000 | `110` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.office.high.triangles` | integer | min 1, max 50000000 | `200000` | Maximum triangles per frame (count) |
| `budgets.scenes.office.high.p95Ms` | number | min 1, max 200 | `18` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.office.ultra.drawCalls` | integer | min 1, max 5000 | `110` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.office.ultra.triangles` | integer | min 1, max 50000000 | `200000` | Maximum triangles per frame (count) |
| `budgets.scenes.office.ultra.p95Ms` | number | min 1, max 200 | `24` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |

<!-- world-tuning:end -->
