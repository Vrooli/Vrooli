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
Composition rows also come from `scenes/*.json` and `biomes.json`; vegetation-entry
rows define each per-prop record. An em dash means no override or no shared default.
Structural constants are separately reviewed by the literal gate. Edit the JSON, keep values inside their bounds, and run
`pnpm world:tuning-docs` to refresh this table (a test fails when it is stale).
In development the HUD settings panel has a Levers tab that edits these live.

### `version`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `version` | const 1 | — | `1` | Tuning file format version (integer) |

### `visual`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `visual.terrain.aoRadius` | number | min 0.1, max 20 | `3` | Terrain colour occlusion sampling radius (metres) |
| `visual.terrain.aoSamples` | integer | min 1, max 32 | `8` | Terrain colour occlusion samples (count) |
| `visual.terrain.tintBase` | number | min 0, max 1 | `0.7` | Base strength of secondary terrain tint (0..1) |
| `visual.terrain.tintAmplitude` | number | min 0, max 1 | `0.15` | Strength of each terrain tint variation wave (0..1) |
| `visual.terrain.tintFrequencyX1` | number | min 0, max 2 | `0.17` | First terrain tint wave x frequency (radians per metre) |
| `visual.terrain.tintFrequencyZ1` | number | min 0, max 2 | `0.11` | First terrain tint wave z frequency (radians per metre) |
| `visual.terrain.tintFrequencyX2` | number | min 0, max 2 | `0.07` | Second terrain tint wave x frequency (radians per metre) |
| `visual.terrain.tintFrequencyZ2` | number | min 0, max 2 | `0.19` | Second terrain tint wave z frequency (radians per metre) |
| `visual.terrain.wetColor` | string | — | `"#d4dce0"` | Wet terrain material colour multiplier (hex colour) |
| `visual.terrain.dryColor` | string | — | `"#ffffff"` | Dry terrain material colour multiplier (hex colour) |
| `visual.terrain.minimumRoughness` | number | min 0, max 1 | `0.3` | Minimum wet terrain material roughness (0..1) |
| `visual.terrain.wetRoughnessScale` | number | min 0, max 1 | `0.65` | Roughness reduction per unit terrain wetness (0..1) |
| `visual.water.color` | string | — | `"#4f9db8"` | Water colour (hex colour) |
| `visual.water.waveFrequencyX` | number | min 0, max 2 | `0.18` | Water x wave frequency (radians per metre) |
| `visual.water.waveFrequencyZ` | number | min 0, max 2 | `0.16` | Water z wave frequency (radians per metre) |
| `visual.water.waveSpeed` | number | min 0, max 10 | `1` | Water primary wave speed (radians per second) |
| `visual.water.crossWaveSpeed` | number | min 0, max 10 | `0.7` | Water cross wave speed (radians per second) |
| `visual.water.waveAmplitude` | number | min 0, max 0.5 | `0.025` | Water wave vertical amplitude (metres) |
| `visual.water.shoreFadeWidth` | number | min 0.01, max 10 | `1.25` | Water transparency transition width at the shore (metres) |
| `visual.water.shoreBrightness` | number | min 0, max 1 | `0.82` | Water colour multiplier at the shore (0..1) |
| `visual.water.shoreOpacity` | number | min 0, max 1 | `0.18` | Water opacity at the shore (0..1) |
| `visual.water.deepOpacity` | number | min 0, max 1 | `0.72` | Water opacity beyond the shore fade (0..1) |
| `visual.post.aoRadius` | number | min 0.01, max 10 | `1.6` | Screen-space ambient occlusion radius (metres) |
| `visual.post.aoIntensity` | number | min 0, max 10 | `2.2` | Screen-space ambient occlusion strength (multiplier) |
| `visual.post.aoFalloff` | number | min 0.01, max 10 | `1` | Screen-space ambient occlusion distance falloff (multiplier) |
| `visual.post.bloomThreshold` | number | min 0, max 10 | `1` | Bloom luminance threshold (linear luminance) |
| `visual.post.bloomIntensity` | number | min 0, max 5 | `0.55` | Bloom intensity (multiplier) |
| `visual.post.bloomRadius` | number | min 0, max 1 | `0.65` | Bloom blur radius (0..1) |

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
| `layout.lampInsetRatio` | number | min 0, max 0.5 | `0.08` | Room lamp inset relative to its shortest side (ratio) |
| `layout.corridorLampSpacing` | number | min 1, max 50 | `10` | Distance used to distribute corridor lamps (metres) |
| `layout.corridorLampScale` | number | min 0.1, max 3 | `0.8` | Corridor lamp size relative to room lamps (multiplier) |
| `layout.cellSize` | number | min 0.1, max 2 | `0.5` | Navigation grid cell size (metres) |
| `layout.roomWidth` | number | min 3, max 30 | `8` | Width (x) of a team room (metres) |
| `layout.roomDepth` | number | min 3, max 30 | `6` | Depth (z) of a team room (metres) |
| `layout.deskPitch` | number | min 0.8, max 5 | `1.7` | Distance between neighbouring desks along the back wall (metres) |
| `layout.deskInset` | number | min 0.2, max 5 | `1.1` | Distance from the back wall to the desk row (metres) |
| `layout.deskSeatOffset` | number | min 0.2, max 3 | `0.8` | Distance in front of a desk where its owner stands (metres) |
| `layout.tableRadius` | number | min 0.3, max 4 | `0.9` | Radius of the team table prop footprint (metres) |
| `layout.tableSeatRadius` | number | min 0.5, max 6 | `1.5` | Radius of the seat ring around a team table (metres) |
| `layout.tableSeats` | integer | min 2, max 16 | `6` | Seats around a team table (count) |
| `layout.commonsRadius` | number | min 2, max 20 | `5.5` | Radius of the commons clearing (metres) |
| `layout.commonsSeatRadius` | number | min 0.5, max 10 | `2.2` | Radius of the seat ring around the campfire (metres) |
| `layout.commonsSeats` | integer | min 2, max 24 | `8` | Seats around the campfire (count) |
| `layout.clearingRadius` | number | min 0, max 20 | `3.5` | No tree spawns within this distance of a room or the hero camera (metres) |
| `layout.wallHeight` | number | min 0, max 3 | `0.7` | Height of the low wall around a room (metres) |
| `layout.surfaces.wallThickness` | number | min 0.01, max 1 | `0.18` | Room wall thickness (metres) |
| `layout.surfaces.doorFrameScale` | number | min 0.1, max 4 | `1.5` | Door frame thickness relative to wall thickness (multiplier) |
| `layout.surfaces.floorLift` | number | min 0, max 0.2 | `0.012` | Room floor centre above terrain (metres) |
| `layout.surfaces.corridorLift` | number | min 0, max 0.2 | `0.011` | Corridor floor centre above terrain (metres) |
| `layout.surfaces.floorThickness` | number | min 0.001, max 0.4 | `0.02` | Room and corridor floor slab thickness (metres) |
| `layout.surfaces.commonsLift` | number | min 0, max 0.2 | `0.01` | Commons disc height above terrain (metres) |
| `layout.surfaces.commonsSegments` | integer | min 3, max 128 | `48` | Commons disc circumference segments (count) |
| `layout.surfaces.wallRoughness` | number | min 0, max 1 | `0.85` | Wall material roughness (0..1) |
| `layout.surfaces.floorRoughness` | number | min 0, max 1 | `0.95` | Room floor material roughness (0..1) |
| `layout.surfaces.corridorRoughness` | number | min 0, max 1 | `0.9` | Corridor floor material roughness (0..1) |
| `layout.surfaces.commonsRoughness` | number | min 0, max 1 | `1` | Commons disc material roughness (0..1) |
| `layout.boardOffset` | number | min 0, max 20 | `4` | Distance from the commons centre to the runs board (metres) |
| `layout.outlineRimSamples` | integer | min 4, max 64 | `12` | Points sampled around the commons rim for the outline the camera frames (count) |
| `layout.siteCandidates` | integer | min 16, max 4096 | `2048` | Seeded candidates scored for each settlement site (count) |
| `layout.siteRadiusMax` | number | min 10, max 300 | `55` | Maximum distance of a team site from the commons (metres) |
| `layout.siteSpacing` | number | min 2, max 100 | `10` | Minimum separation between settlement site centres (metres) |
| `layout.siteWeightFlat` | number | min 0, max 10 | `3` | Buildability weight favouring flat ground (relative weight) |
| `layout.siteWeightDry` | number | min 0, max 10 | `4` | Buildability weight favouring ground outside water and shore (relative weight) |
| `layout.siteWeightNear` | number | min 0, max 10 | `1` | Buildability weight favouring sites near the commons (relative weight) |
| `layout.siteWeightApart` | number | min 0, max 10 | `2` | Buildability weight favouring separation from selected sites (relative weight) |
| `layout.siteRotationSnapRad` | number | min 0.01, max 1.5707963267948966 | `0.2617993878` | Angular increment used to snap generated site rotations (radians) |
| `layout.scatterJitter` | number | min 0, max 1 | `0.45` | Share of one terrain cell available for decor position jitter (0..1) |
| `layout.shoreClearance` | number | min 0, max 20 | `1.5` | Dry margin required around water for vegetation and decor (metres) |
| `layout.stands.frequency` | number | min 0.001, max 1 | `0.03` | Vegetation stand noise frequency (cycles per metre) |
| `layout.stands.octaves` | integer | min 1, max 8 | `3` | Noise octaves shaping vegetation stands (count) |
| `layout.stands.lacunarity` | number | min 1, max 4 | `2` | Stand frequency increase per octave (multiplier) |
| `layout.stands.gain` | number | min 0, max 1 | `0.5` | Stand amplitude retained per octave (0..1) |
| `layout.stands.threshold` | number | min 0, max 1 | `0.45` | Noise level at which a vegetation stand begins (0..1) |
| `layout.stands.softness` | number | min 0.001, max 1 | `0.15` | Noise transition width from gaps to stands (0..1) |
| `layout.stands.contrast` | number | min 0.1, max 8 | `2` | Exponent concentrating vegetation inside stands (power) |
| `layout.stands.floor` | number | min 0, max 1 | `0.05` | Minimum density multiplier outside vegetation stands (0..1) |
| `layout.decorSpacingFactor` | number | min 0, max 1 | `0.55` | Decor spacing as a fraction of tree spacing (0..1) |
| `layout.decorScale.min` | number | min 0.1, max 4 | `0.6` | Seeded decor scale lower bound (multiplier) |
| `layout.decorScale.max` | number | min 0.1, max 4 | `1.55` | Seeded decor scale upper bound (multiplier) |
| `layout.decorColorJitter` | number | min 0, max 1 | `0.08` | Maximum seeded per-channel vegetation colour variation (0..1) |
| `layout.floorplan.corridorWidth` | number | min 1, max 10 | `3` | Primary and secondary corridor width (metres) |
| `layout.floorplan.secondaryCorridors.min` | number | min 0, max 8 | `1` | Secondary corridor count lower bound (count) |
| `layout.floorplan.secondaryCorridors.max` | number | min 0, max 8 | `2` | Secondary corridor count upper bound (count) |
| `layout.floorplan.splitRatio.min` | number | min 0.25, max 0.75 | `0.4` | Seeded room split ratio lower bound (ratio) |
| `layout.floorplan.splitRatio.max` | number | min 0.25, max 0.75 | `0.6` | Seeded room split ratio upper bound (ratio) |
| `layout.floorplan.maxAspect` | number | min 1, max 6 | `2.5` | Maximum room aspect ratio (ratio) |
| `layout.floorplan.roomAreaPerMember` | number | min 2, max 50 | `7` | Target room area per team member (square metres) |
| `layout.floorplan.roomMinArea` | number | min 8, max 200 | `30` | Minimum room area (square metres) |
| `layout.floorplan.plateMargin` | number | min 1, max 30 | `4` | Floorplate margin around rooms (metres) |
| `layout.floorplan.doorWidth` | number | min 0.8, max 4 | `1.4` | Room doorway width (metres) |
| `layout.floorplan.lobbyRadius` | number | min 1, max 15 | `3` | Lobby gathering radius (metres) |
| `layout.floorplan.plateAspect.min` | number | min 1, max 3 | `1.15` | Seeded office floorplate aspect ratio lower bound (ratio) |
| `layout.floorplan.plateAspect.max` | number | min 1, max 3 | `1.65` | Seeded office floorplate aspect ratio upper bound (ratio) |
| `layout.floorplan.primaryOffset` | number | min 0, max 1 | `0.25` | Maximum seeded primary-corridor offset as a fraction of corridor width (0..1) |
| `layout.floorplan.secondaryJitter` | number | min 0, max 1 | `0.12` | Maximum seeded secondary-corridor jitter within its even spacing (0..1) |
| `layout.interior.tableMinMembers` | integer | min 1, max 100 | `2` | Minimum team size for a meeting table (count) |
| `layout.interior.fillerMax` | integer | min 0, max 3 | `3` | Maximum seeded filler props per room (count) |

### `terrain`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `terrain.radius` | number | min 10, max 500 | `90` | Radius of the generated terrain field (metres) |
| `terrain.cellSize` | number | min 0.25, max 8 | `1` | Spacing between terrain field samples (metres) |
| `terrain.amplitude` | number | min 0, max 20 | `1.5` | Maximum absolute terrain elevation (metres) |
| `terrain.frequency` | number | min 0.001, max 1 | `0.018` | Base terrain noise frequency (cycles per metre) |
| `terrain.detailAmplitude` | number | min 0, max 5 | `0.1` | Higher-frequency surface-detail amplitude (metres) |
| `terrain.detailFrequency` | number | min 0.001, max 2 | `0.07` | Surface-detail noise frequency (cycles per metre) |
| `terrain.octaves` | integer | min 1, max 8 | `4` | Fractal noise octaves used for height and moisture (count) |
| `terrain.lacunarity` | number | min 1, max 4 | `2` | Frequency multiplier between terrain noise octaves (multiplier) |
| `terrain.gain` | number | min 0, max 1 | `0.5` | Amplitude multiplier between terrain noise octaves (0..1) |
| `terrain.moistureFrequency` | number | min 0.001, max 1 | `0.025` | Base moisture noise frequency (cycles per metre) |
| `terrain.moistureWarp` | number | min 0, max 50 | `12` | Domain-warp distance applied to moisture sampling (metres) |
| `terrain.falloffStart` | number | min 0, max 1 | `0.55` | Fraction of terrain radius where elevation begins fading to zero (0..1) |
| `terrain.waterLevel` | number | min -20, max 20 | `-0.45` | Water surface elevation (metres) |
| `terrain.shoreMargin` | number | min 0, max 20 | `1.2` | Dry navigation margin around water (metres) |
| `terrain.waterSurfaceLift` | number | min 0, max 0.2 | `0.015` | Water surface lift above its terrain threshold (metres) |
| `terrain.wetShoreWidth` | number | min 0.1, max 10 | `2` | Width of terrain darkening immediately inside water (metres) |
| `terrain.wetShoreDarkening` | number | min 0, max 1 | `0.22` | Maximum terrain darkening immediately inside water (0..1) |
| `terrain.maxSiteSlope` | number | min 0, max 1.5707963267948966 | `0.14` | Steepest ground eligible for a team site (radians) |
| `terrain.maxWalkSlope` | number | min 0, max 1.5707963267948966 | `0.45` | Steepest ground eligible for navigation (radians) |
| `terrain.kerbWidth` | number | min 0.25, max 10 | `2` | Width over which a level site pad blends into terrain (metres) |
| `terrain.pathWidth` | number | min 0.25, max 10 | `1.4` | Width of paths painted into terrain colour (metres) |
| `terrain.innerCellSize` | number | min 0.25, max 4 | `1` | Terrain mesh spacing near the settlement (metres) |
| `terrain.innerRadius` | number | min 5, max 300 | `60` | Radius of the dense inner terrain mesh (metres) |
| `terrain.ringFalloff` | number | min 1, max 8 | `2` | Terrain mesh spacing multiplier beyond the inner ring (multiplier) |
| `terrain.tileSize` | number | min 4, max 100 | `30` | Side length of one vegetation culling tile (metres) |
| `terrain.moistureBasinDepth` | number | min 0, max 5 | `0.35` | Maximum moisture bias subtracted when classifying water (metres) |
| `terrain.shoreMinGrade` | number | min 0.001, max 1 | `0.03` | Minimum grade used to estimate horizontal shore distance (rise over run) |
| `terrain.padClearance` | number | min 0, max 5 | `0.75` | Minimum terrace elevation above the configured water surface (metres) |
| `terrain.siteLevelTolerance` | number | min 0.001, max 1 | `0.05` | Maximum elevation variation allowed across a site pad (metres) |

### `camera`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `camera.minimumProjectionAspect` | number | min 0.000001, max 1 | `0.01` | Minimum aspect ratio used by the framing solver (width/height ratio) |
| `camera.minimumFrameFill` | number | min 0.001, max 1 | `0.05` | Minimum requested viewport share used by the framing solver (ratio) |
| `camera.initialPosition` | array<unknown> | — | `[0,20,40]` | Camera position before the framing rig is ready (metres, x/y/z) |
| `camera.boundaryHeight` | number | min 1, max 200 | `30` | Maximum camera target boundary height (metres) |
| `camera.frameHeight` | number | min 0.1, max 20 | `2` | Default framed box height for walls, actors and labels (metres) |
| `camera.input.mouse.left` | "none" \| "rotate" \| "truck" \| "offset" \| "dolly" \| "zoom" | — | `"rotate"` | Left-button drag action |
| `camera.input.mouse.middle` | "none" \| "rotate" \| "truck" \| "offset" \| "dolly" \| "zoom" | — | `"truck"` | Middle-button drag action |
| `camera.input.mouse.right` | "none" \| "rotate" \| "truck" \| "offset" \| "dolly" \| "zoom" | — | `"truck"` | Right-button drag action |
| `camera.input.mouse.wheel` | "none" \| "rotate" \| "truck" \| "offset" \| "dolly" \| "zoom" | — | `"dolly"` | Mouse-wheel action |
| `camera.input.touch.one` | "none" \| "rotate" \| "truck" \| "screen-pan" \| "offset" \| "dolly" \| "zoom" | — | `"rotate"` | One-finger drag action |
| `camera.input.touch.two` | "none" \| "rotate" \| "truck" \| "screen-pan" \| "offset" \| "dolly" \| "zoom" \| "dolly-truck" \| "dolly-screen-pan" \| "dolly-offset" \| "dolly-rotate" \| "zoom-truck" \| "zoom-screen-pan" \| "zoom-offset" \| "zoom-rotate" | — | `"dolly-truck"` | Two-finger gesture action |
| `camera.input.touch.three` | "none" \| "rotate" \| "truck" \| "screen-pan" \| "offset" \| "dolly" \| "zoom" \| "dolly-truck" \| "dolly-screen-pan" \| "dolly-offset" \| "dolly-rotate" \| "zoom-truck" \| "zoom-screen-pan" \| "zoom-offset" \| "zoom-rotate" | — | `"truck"` | Three-finger gesture action |
| `camera.dollyToCursor` | boolean | — | `true` | Zoom toward the pointer instead of the orbit target |
| `camera.truckSpeed` | number | min 0.1, max 10 | `1.5` | Pan speed per pointer unit (multiplier) |
| `camera.dollySpeed` | number | min 0.1, max 10 | `1` | Dolly speed per wheel unit (multiplier) |
| `camera.cullEpsilonMetres` | number | min 0, max 1 | `0.05` | Camera movement required to refresh vegetation visibility (metres) |
| `camera.cullEpsilonRadians` | number | min 0, max 0.1 | `0.002` | Camera rotation required to refresh vegetation visibility (radians) |
| `camera.fov` | number | min 10, max 90 | `38` | Vertical field of view (degrees) |
| `camera.near` | number | min 0.01, max 10 | `0.5` | Near clip plane (metres) |
| `camera.far` | number | min 10, max 2000 | `400` | Far clip plane (metres) |
| `camera.polarMinDeg` | number | min 0, max 89 | `30` | Steepest allowed camera angle from straight above (degrees) |
| `camera.polarMaxDeg` | number | min 1, max 90 | `64` | Shallowest allowed camera angle from straight above (degrees) |
| `camera.azimuthRangeDeg` | number | min 0, max 180 | `35` | Orbit allowed either side of the hero azimuth (degrees) |
| `camera.minDistance` | number | min 1, max 100 | `1.5` | Closest dolly distance (metres) |
| `camera.maxDistance` | number | min 2, max 500 | `140` | Farthest dolly distance (metres) |
| `camera.introSeconds` | number | min 0, max 10 | `2` | Length of the establishing-to-hero dolly on load (seconds) |
| `camera.smoothTime` | number | min 0.01, max 3 | `0.35` | Camera-controls smoothing time for every move (seconds) |
| `camera.followEpsilon` | number | min 0.001, max 2 | `0.15` | Target movement that starts a follow update (metres) |
| `camera.followSmoothTime` | number | min 0.01, max 3 | `0.35` | Camera gesture smoothing while following an actor (seconds) |
| `camera.frameFill` | number | min 0, max 1 | `0.9` | Share of the viewport the layout outline fills at distanceFactor 1; poses scale from this (0..1) |
| `camera.focusPadding` | number | min 1, max 5 | `1.15` | Divides viewport fill when focusing an actor or room (multiplier) |
| `camera.minClearance` | number | min 0, max 20 | `2` | The first frame must have no geometry closer than this to the camera (metres) |
| `camera.keyOrbitDegPerSec` | number | min 1, max 360 | `55` | Orbit speed for keyboard arrows (degrees) |
| `camera.keyDollyPerSec` | number | min 0.1, max 100 | `10` | Dolly speed for keyboard +/- (metres per second) |

### `lighting`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `lighting.rig.environmentResolution` | integer | min 16, max 1024 | `256` | Environment cube map edge length in pixels (count) |
| `lighting.rig.sunDistance` | number | min 10, max 1000 | `100` | Directional key distance from the shadow centre (metres) |
| `lighting.rig.shadowExtentScale` | number | min 0.1, max 2 | `0.62` | Shadow half extent relative to the longest footprint edge (ratio) |
| `lighting.rig.shadowExtentPadding` | number | min 0, max 50 | `6` | Additional shadow half extent (metres) |
| `lighting.rig.hemisphereHeight` | number | min 1, max 200 | `50` | Hemisphere light height (metres) |
| `lighting.rig.keyPanel.intensity` | number | min 0, max 10 | `1.2` | Environment panel radiance (multiplier) |
| `lighting.rig.keyPanel.position` | array<unknown> | — | `[8,6,-8]` | Environment panel position (metres, x/y/z) |
| `lighting.rig.keyPanel.scale` | array<unknown> | — | `[8,4,1]` | Environment panel dimensions (metres, x/y/z) |
| `lighting.rig.fillPanel.intensity` | number | min 0, max 10 | `0.6` | Environment panel radiance (multiplier) |
| `lighting.rig.fillPanel.position` | array<unknown> | — | `[-9,4,4]` | Environment panel position (metres, x/y/z) |
| `lighting.rig.fillPanel.scale` | array<unknown> | — | `[6,3,1]` | Environment panel dimensions (metres, x/y/z) |
| `lighting.rig.topPanel.intensity` | number | min 0, max 10 | `0.4` | Environment panel radiance (multiplier) |
| `lighting.rig.topPanel.position` | array<unknown> | — | `[0,10,0]` | Environment panel position (metres, x/y/z) |
| `lighting.rig.topPanel.scale` | array<unknown> | — | `[6,6,6]` | Environment panel dimensions (metres, x/y/z) |
| `lighting.rig.topPanelColor` | string | — | `"#ffffff"` | Top environment ring colour (hex colour) |
| `lighting.lampLightIntensity` | number | min 0, max 200 | `1` | Lamp point-light intensity before period scaling (candela) |
| `lighting.lampLightDistance` | number | min 0.1, max 100 | `14` | Maximum range of lamp point lights (metres) |
| `lighting.lampLightHeight` | number | min 0, max 10 | `1.8` | Lamp light centre above the placement ground (metres) |
| `lighting.clockPollSeconds` | number | min 1, max 3600 | `60` | How often clock mode re-reads the local hour (seconds) |
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

### `weather`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `weather.lightingLimits.fogNearMax` | number | min 0, max 100 | `10` | Maximum weather-adjusted fog near distance (framing-distance multiplier) |
| `weather.lightingLimits.fogFarMin` | number | min 0.001, max 100 | `0.1` | Minimum weather-adjusted fog far distance (framing-distance multiplier) |
| `weather.lightingLimits.fogFarMax` | number | min 0.001, max 100 | `20` | Maximum weather-adjusted fog far distance (framing-distance multiplier) |
| `weather.lightingLimits.exposureMax` | number | min 0, max 20 | `4` | Maximum weather-adjusted exposure (multiplier) |
| `weather.lightingLimits.keyIntensityMax` | number | min 0, max 100 | `20` | Maximum weather-adjusted key intensity (light intensity units) |
| `weather.lightingLimits.ambientIntensityMax` | number | min 0, max 20 | `4` | Maximum weather-adjusted ambient intensity (light intensity units) |
| `weather.states.clear.fogNearScale` | number | min 0, max 3 | `1` | Fog near-distance scale (multiplier) |
| `weather.states.clear.fogFarScale` | number | min 0, max 3 | `1` | Fog far-distance scale (multiplier) |
| `weather.states.clear.exposureScale` | number | min 0, max 3 | `1` | Exposure scale (multiplier) |
| `weather.states.clear.keyIntensityScale` | number | min 0, max 3 | `1` | Directional light intensity scale (multiplier) |
| `weather.states.clear.ambientScale` | number | min 0, max 3 | `1` | Ambient light intensity scale (multiplier) |
| `weather.states.clear.skyBlurAdd` | number | min 0, max 1 | `0` | Additional sky blur (0..1) |
| `weather.states.clear.cloudCoverage` | number | min 0, max 1 | `0.05` | Cloud layer coverage (0..1) |
| `weather.states.clear.particleRate` | number | min 0, max 1 | `0` | Weather particle rate (0..1) |
| `weather.states.clear.particleFallSpeed` | number | min 0, max 100 | `12` | Particle downward speed (metres per second) |
| `weather.states.clear.particleSize` | number | min 0.001, max 5 | `0.08` | Particle point size before perspective scaling (metres) |
| `weather.states.clear.particleColor` | string | — | `"#82b7d2"` | Particle colour (hex colour) |
| `weather.states.clear.wetness` | number | min 0, max 1 | `0` | Terrain wetness (0..1) |
| `weather.states.clear.terrainTint` | string | — | `"#ffffff"` | Terrain weather tint (hex colour) |
| `weather.states.clear.terrainTintMix` | number | min 0, max 1 | `0` | Share of the terrain colour replaced by the weather tint (0..1) |
| `weather.states.clear.terrainShadowTint` | string | — | `"#ffffff"` | Secondary terrain tint used for weather variation (hex colour) |
| `weather.states.clear.terrainTintVariation` | number | min 0, max 1 | `0` | Maximum deterministic blend from terrainTint toward terrainShadowTint (0..1) |
| `weather.states.clear.skyTint` | string | — | `"#ffffff"` | Sky and fog weather tint (hex colour) |
| `weather.states.clear.skyTintMix` | number | min 0, max 1 | `0` | Share of the period sky and fog colours replaced by the weather tint (0..1) |
| `weather.states.clear.minSeconds` | number | min 1, max 3600 | `90` | Minimum state duration (seconds) |
| `weather.states.clear.maxSeconds` | number | min 1, max 3600 | `240` | Maximum state duration (seconds) |
| `weather.states.cloudy.fogNearScale` | number | min 0, max 3 | `0.85` | Fog near-distance scale (multiplier) |
| `weather.states.cloudy.fogFarScale` | number | min 0, max 3 | `0.8` | Fog far-distance scale (multiplier) |
| `weather.states.cloudy.exposureScale` | number | min 0, max 3 | `0.92` | Exposure scale (multiplier) |
| `weather.states.cloudy.keyIntensityScale` | number | min 0, max 3 | `0.65` | Directional light intensity scale (multiplier) |
| `weather.states.cloudy.ambientScale` | number | min 0, max 3 | `1.08` | Ambient light intensity scale (multiplier) |
| `weather.states.cloudy.skyBlurAdd` | number | min 0, max 1 | `0.2` | Additional sky blur (0..1) |
| `weather.states.cloudy.cloudCoverage` | number | min 0, max 1 | `0.65` | Cloud layer coverage (0..1) |
| `weather.states.cloudy.particleRate` | number | min 0, max 1 | `0` | Weather particle rate (0..1) |
| `weather.states.cloudy.particleFallSpeed` | number | min 0, max 100 | `12` | Particle downward speed (metres per second) |
| `weather.states.cloudy.particleSize` | number | min 0.001, max 5 | `0.08` | Particle point size before perspective scaling (metres) |
| `weather.states.cloudy.particleColor` | string | — | `"#82b7d2"` | Particle colour (hex colour) |
| `weather.states.cloudy.wetness` | number | min 0, max 1 | `0.15` | Terrain wetness (0..1) |
| `weather.states.cloudy.terrainTint` | string | — | `"#aab6c2"` | Terrain weather tint (hex colour) |
| `weather.states.cloudy.terrainTintMix` | number | min 0, max 1 | `0.35` | Share of the terrain colour replaced by the weather tint (0..1) |
| `weather.states.cloudy.terrainShadowTint` | string | — | `"#aab6c2"` | Secondary terrain tint used for weather variation (hex colour) |
| `weather.states.cloudy.terrainTintVariation` | number | min 0, max 1 | `0` | Maximum deterministic blend from terrainTint toward terrainShadowTint (0..1) |
| `weather.states.cloudy.skyTint` | string | — | `"#8894a0"` | Sky and fog weather tint (hex colour) |
| `weather.states.cloudy.skyTintMix` | number | min 0, max 1 | `0.35` | Share of the period sky and fog colours replaced by the weather tint (0..1) |
| `weather.states.cloudy.minSeconds` | number | min 1, max 3600 | `60` | Minimum state duration (seconds) |
| `weather.states.cloudy.maxSeconds` | number | min 1, max 3600 | `180` | Maximum state duration (seconds) |
| `weather.states.rain.fogNearScale` | number | min 0, max 3 | `0.7` | Fog near-distance scale (multiplier) |
| `weather.states.rain.fogFarScale` | number | min 0, max 3 | `0.6` | Fog far-distance scale (multiplier) |
| `weather.states.rain.exposureScale` | number | min 0, max 3 | `0.85` | Exposure scale (multiplier) |
| `weather.states.rain.keyIntensityScale` | number | min 0, max 3 | `0.45` | Directional light intensity scale (multiplier) |
| `weather.states.rain.ambientScale` | number | min 0, max 3 | `1.15` | Ambient light intensity scale (multiplier) |
| `weather.states.rain.skyBlurAdd` | number | min 0, max 1 | `0.25` | Additional sky blur (0..1) |
| `weather.states.rain.cloudCoverage` | number | min 0, max 1 | `0.85` | Cloud layer coverage (0..1) |
| `weather.states.rain.particleRate` | number | min 0, max 1 | `1` | Weather particle rate (0..1) |
| `weather.states.rain.particleFallSpeed` | number | min 0, max 100 | `12` | Particle downward speed (metres per second) |
| `weather.states.rain.particleSize` | number | min 0.001, max 5 | `0.08` | Particle point size before perspective scaling (metres) |
| `weather.states.rain.particleColor` | string | — | `"#82b7d2"` | Particle colour (hex colour) |
| `weather.states.rain.wetness` | number | min 0, max 1 | `0.8` | Terrain wetness (0..1) |
| `weather.states.rain.terrainTint` | string | — | `"#335b78"` | Terrain weather tint (hex colour) |
| `weather.states.rain.terrainTintMix` | number | min 0, max 1 | `0.65` | Share of the terrain colour replaced by the weather tint (0..1) |
| `weather.states.rain.terrainShadowTint` | string | — | `"#335b78"` | Secondary terrain tint used for weather variation (hex colour) |
| `weather.states.rain.terrainTintVariation` | number | min 0, max 1 | `0` | Maximum deterministic blend from terrainTint toward terrainShadowTint (0..1) |
| `weather.states.rain.skyTint` | string | — | `"#243b55"` | Sky and fog weather tint (hex colour) |
| `weather.states.rain.skyTintMix` | number | min 0, max 1 | `0.65` | Share of the period sky and fog colours replaced by the weather tint (0..1) |
| `weather.states.rain.minSeconds` | number | min 1, max 3600 | `60` | Minimum state duration (seconds) |
| `weather.states.rain.maxSeconds` | number | min 1, max 3600 | `240` | Maximum state duration (seconds) |
| `weather.states.snow.fogNearScale` | number | min 0, max 3 | `0.55` | Fog near-distance scale (multiplier) |
| `weather.states.snow.fogFarScale` | number | min 0, max 3 | `0.45` | Fog far-distance scale (multiplier) |
| `weather.states.snow.exposureScale` | number | min 0, max 3 | `0.95` | Exposure scale (multiplier) |
| `weather.states.snow.keyIntensityScale` | number | min 0, max 3 | `0.55` | Directional light intensity scale (multiplier) |
| `weather.states.snow.ambientScale` | number | min 0, max 3 | `1.1` | Ambient light intensity scale (multiplier) |
| `weather.states.snow.skyBlurAdd` | number | min 0, max 1 | `0.3` | Additional sky blur (0..1) |
| `weather.states.snow.cloudCoverage` | number | min 0, max 1 | `0.8` | Cloud layer coverage (0..1) |
| `weather.states.snow.particleRate` | number | min 0, max 1 | `0.75` | Weather particle rate (0..1) |
| `weather.states.snow.particleFallSpeed` | number | min 0, max 100 | `2.5` | Particle downward speed (metres per second) |
| `weather.states.snow.particleSize` | number | min 0.001, max 5 | `0.18` | Particle point size before perspective scaling (metres) |
| `weather.states.snow.particleColor` | string | — | `"#f5fbff"` | Particle colour (hex colour) |
| `weather.states.snow.wetness` | number | min 0, max 1 | `0.25` | Terrain wetness (0..1) |
| `weather.states.snow.terrainTint` | string | — | `"#f5fbff"` | Terrain weather tint (hex colour) |
| `weather.states.snow.terrainTintMix` | number | min 0, max 1 | `1` | Share of the terrain colour replaced by the weather tint (0..1) |
| `weather.states.snow.terrainShadowTint` | string | — | `"#c7e3f2"` | Secondary terrain tint used for weather variation (hex colour) |
| `weather.states.snow.terrainTintVariation` | number | min 0, max 1 | `0.25` | Maximum deterministic blend from terrainTint toward terrainShadowTint (0..1) |
| `weather.states.snow.skyTint` | string | — | `"#dceeff"` | Sky and fog weather tint (hex colour) |
| `weather.states.snow.skyTintMix` | number | min 0, max 1 | `0.95` | Share of the period sky and fog colours replaced by the weather tint (0..1) |
| `weather.states.snow.minSeconds` | number | min 1, max 3600 | `60` | Minimum state duration (seconds) |
| `weather.states.snow.maxSeconds` | number | min 1, max 3600 | `180` | Maximum state duration (seconds) |
| `weather.pressure.recentFailureWeight` | number | min 0, max 1 | `0.5` | Weight of recent run failures (0..1) |
| `weather.pressure.failedActorWeight` | number | min 0, max 1 | `0.4` | Weight of actors currently failed (0..1) |
| `weather.pressure.expiredGatheringWeight` | number | min 0, max 1 | `0.1` | Weight of expired gatherings (0..1) |
| `weather.pressure.eventWindowSeconds` | number | min 1, max 3600 | `1200` | Age window for run outcome events (seconds) |
| `weather.pressureSmoothingSeconds` | number | min 1, max 3600 | `45` | Time constant used to smooth weather pressure (seconds) |
| `weather.particleBaseCount` | integer | min 0, max 20000 | `700` | Particle count at rate and profile scale 1 (count) |
| `weather.particles.spiralAngleStep` | number | min 0.001, max 6.283185307179586 | `2.399963` | Angle between consecutive particles (radians) |
| `weather.particles.columnRadius` | number | min 0, max 200 | `26` | Particle column radius around the camera target (metres) |
| `weather.particles.columnHeight` | number | min 0.1, max 200 | `24` | Particle column vertical wrap height (metres) |
| `weather.particles.verticalStride` | number | min 0, max 200 | `7.13` | Initial height increment between particles (metres) |
| `weather.particles.pointSizeScale` | number | min 1, max 2000 | `300` | Shader point-size perspective scale (pixels) |
| `weather.particles.opacity` | number | min 0, max 1 | `0.78` | Particle opacity (0..1) |
| `weather.cloudPlaneSpan` | number | min 0.1, max 10 | `3` | Cloud plane span relative to world bounds (multiplier) |
| `weather.cloudOpacityScale` | number | min 0, max 1 | `0.28` | Cloud opacity per unit coverage (0..1) |
| `weather.cloudAltitude` | number | min 10, max 1000 | `170` | Cloud layer height above the world (metres) |

### `labels`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `labels.color` | string | — | `"#ffffff"` | Label text colour (hex colour) |
| `labels.strokeColor` | string | — | `"#101423"` | Label stroke colour; stroke avoids the unsupported outline material path (hex colour) |
| `labels.strokePercent` | number | min 0, max 100 | `10` | Label stroke width (percent of font size) |
| `labels.charWidthFactor` | number | min 0.1, max 2 | `0.58` | Estimated character width relative to label height (ratio) |
| `labels.refreshEveryFrames` | integer | min 1, max 60 | `3` | Frames between label visibility updates (count) |
| `labels.basePxPerUnit` | number | min 1, max 200 | `40` | Label sizing reference density (pixels/metre) |
| `labels.pinnedBonus` | number | min 0, max 100 | `10` | Focused or hovered label priority bonus (priority points) |
| `labels.priorities.failed` | integer | min 0, max 100 | `5` | Failed label priority points (count) |
| `labels.priorities.working` | integer | min 0, max 100 | `4` | Working label priority points (count) |
| `labels.priorities.walkingToDesk` | integer | min 0, max 100 | `4` | Walking-to-desk label priority points (count) |
| `labels.priorities.gathered` | integer | min 0, max 100 | `3` | Gathered label priority points (count) |
| `labels.priorities.walkingToTable` | integer | min 0, max 100 | `3` | Walking-to-table label priority points (count) |
| `labels.priorities.socializing` | integer | min 0, max 100 | `2` | Socializing label priority points (count) |
| `labels.priorities.idle` | integer | min 0, max 100 | `1` | Idle label priority points (count) |
| `labels.syncSizeEpsilon` | number | min 0, max 0.01 | `0.0001` | Minimum font size change triggering text geometry synchronization (metres) |
| `labels.renderOrder` | integer | min 0, max 100 | `10` | Label render ordering index (count) |
| `labels.budget` | integer | min 0, max 200 | `24` | Maximum labels drawn at once before clustering (count) |
| `labels.collapseDistance` | number | min 1, max 500 | `34` | Camera distance past which room labels collapse into one count label (metres) |
| `labels.fontSize` | number | min 0.05, max 2 | `0.34` | SDF label height in world units (metres) |
| `labels.offsetY` | number | min 0, max 5 | `1.45` | Label height above the actor origin (metres) |
| `labels.roomOffsetY` | number | min 0, max 8 | `2.8` | Static height for clustered room labels above props (metres) |
| `labels.minScreenPx` | number | min 4, max 64 | `11` | Labels never render smaller than this on screen (pixels) |
| `labels.maxScreenPx` | number | min 4, max 128 | `20` | Labels never render larger than this on screen (pixels) |
| `labels.paddingPx` | number | min 0, max 32 | `4` | Collision padding around a projected label (pixels) |

### `actor`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `actor.extras.tierSizes` | array<number> | length 5 | `[0,0.45,0.65,0.85,1.1]` | Equipment sizes for none, paper, folder, briefcase and backpack (body-radius multipliers) |
| `actor.extras.tierColors` | array<string> | length 5 | `["#000000","#f5f0e6","#e0b35a","#6b4a2b","#3b6ea5"]` | Equipment colours for none, paper, folder, briefcase and backpack (hex colours) |
| `actor.extras.failed.color` | string | — | `"#ff3b3b"` | Accessory glow colour (hex colour) |
| `actor.extras.failed.intensity` | number | min 0, max 10 | `3` | Accessory colour radiance (multiplier) |
| `actor.extras.gathered.color` | string | — | `"#ffb020"` | Accessory glow colour (hex colour) |
| `actor.extras.gathered.intensity` | number | min 0, max 10 | `2.2` | Accessory colour radiance (multiplier) |
| `actor.extras.working.color` | string | — | `"#59d0ff"` | Accessory glow colour (hex colour) |
| `actor.extras.working.intensity` | number | min 0, max 10 | `2.6` | Accessory colour radiance (multiplier) |
| `actor.extras.offColor` | string | — | `"#000000"` | Hidden accessory colour (hex colour) |
| `actor.extras.emotes.start.color` | string | — | `"#59d0ff"` | Accessory glow colour (hex colour) |
| `actor.extras.emotes.start.intensity` | number | min 0, max 10 | `2` | Accessory colour radiance (multiplier) |
| `actor.extras.emotes.done.color` | string | — | `"#5ce27a"` | Accessory glow colour (hex colour) |
| `actor.extras.emotes.done.intensity` | number | min 0, max 10 | `2` | Accessory colour radiance (multiplier) |
| `actor.extras.emotes.fail.color` | string | — | `"#ff3b3b"` | Accessory glow colour (hex colour) |
| `actor.extras.emotes.fail.intensity` | number | min 0, max 10 | `2.5` | Accessory colour radiance (multiplier) |
| `actor.extras.emotes.message.color` | string | — | `"#ffffff"` | Accessory glow colour (hex colour) |
| `actor.extras.emotes.message.intensity` | number | min 0, max 10 | `1.8` | Accessory colour radiance (multiplier) |
| `actor.extras.emotes.gather.color` | string | — | `"#ffb020"` | Accessory glow colour (hex colour) |
| `actor.extras.emotes.gather.intensity` | number | min 0, max 10 | `2` | Accessory colour radiance (multiplier) |
| `actor.extras.spinRate` | number | min 0, max 20 | `2.4` | Working marker angular speed (radians/second) |
| `actor.extras.gearHeight` | number | min 0.1, max 4 | `1.2` | Equipment height relative to width (ratio) |
| `actor.extras.gearDepth` | number | min 0.1, max 4 | `0.5` | Equipment depth relative to width (ratio) |
| `actor.extras.gearRoughness` | number | min 0, max 1 | `0.8` | Equipment material roughness (0..1) |
| `actor.extras.ringThickness` | number | min 0.01, max 1 | `0.18` | Working ring tube radius relative to ring radius (ratio) |
| `actor.extras.ringRadialSegments` | integer | min 3, max 64 | `8` | Working ring tube circumference segments (count) |
| `actor.extras.ringTubularSegments` | integer | min 3, max 128 | `24` | Working ring circumference segments (count) |
| `actor.extras.markWidthSegments` | integer | min 3, max 64 | `10` | Status marker sphere longitude segments (count) |
| `actor.extras.markHeightSegments` | integer | min 2, max 64 | `10` | Status marker sphere latitude segments (count) |
| `actor.extras.markScale` | number | min 0, max 3 | `0.6` | Status marker radius relative to working ring radius (ratio) |
| `actor.extras.emoteOpacity` | number | min 0, max 1 | `0.9` | Emote material opacity (0..1) |
| `actor.extras.emoteShrink` | number | min 0, max 1 | `0.5` | Emote size reduction over its lifetime (0..1) |
| `actor.shadow.textureSize` | integer | min 8, max 512 | `64` | Actor contact shadow texture edge length in pixels (count) |
| `actor.shadow.lift` | number | min 0, max 0.5 | `0.035` | Actor contact shadow lift above terrain (metres) |
| `actor.shadow.opacity` | number | min 0, max 1 | `0.38` | Actor contact shadow opacity (0..1) |
| `actor.shadow.spread` | number | min 0.1, max 4 | `1.15` | Contact shadow radius relative to body radius (multiplier) |
| `actor.shadow.hopShrink` | number | min 0, max 1 | `0.35` | Contact shadow radius reduction at the top of a hop (0..1) |
| `actor.shadow.color` | string | — | `"#000000"` | Actor contact shadow colour (hex colour) |
| `actor.shadow.gradient` | array<object> | — | `[{"position":0,"color":"rgba(255,255,255,1)"},{"position":0.55,"color":"rgba(255,255,255,0.55)"},{"position":1,"color":"rgba(255,255,255,0)"}]` | Radial contact shadow falloff stops |
| `actor.material.color` | string | — | `"#ffffff"` | Base slime material colour multiplier (hex colour) |
| `actor.material.sheenColor` | string | — | `"#ffffff"` | Slime sheen colour (hex colour) |
| `actor.material.roughness` | number | min 0, max 1 | `0.55` | Slime surface roughness (0..1) |
| `actor.material.clearcoat` | number | min 0, max 1 | `0.6` | Slime clearcoat strength (0..1) |
| `actor.material.clearcoatRoughness` | number | min 0, max 1 | `0.35` | Slime clearcoat roughness (0..1) |
| `actor.material.sheen` | number | min 0, max 1 | `0.4` | Slime fabric-like sheen strength (0..1) |
| `actor.material.wobbleScale` | number | min 0, max 20 | `3` | Slime noise spatial frequency (cycles per metre) |
| `actor.material.wobbleSpeed` | number | min 0, max 20 | `1.5` | Slime noise travel speed (noise units per second) |
| `actor.mesh.widthSegments` | integer | min 3, max 64 | `16` | Slime sphere longitude segments (count) |
| `actor.mesh.heightSegments` | integer | min 2, max 64 | `10` | Slime sphere latitude segments (count) |
| `actor.mesh.timeShiftSeconds` | number | min 0, max 60 | `16` | Range of seeded per-actor animation time offsets (seconds) |
| `actor.facing.restSpeed` | number | min 0, max 2 | `0.05` | Maximum ground speed for a focused actor to face the viewer (metres per second) |
| `actor.facing.blendSeconds` | number | min 0.05, max 5 | `0.6` | Time to blend resting actor facing toward or away from the viewer (seconds) |
| `actor.facing.maxYawDeg` | number | min 0, max 180 | `180` | Maximum presentation yaw away from the simulation heading (degrees) |
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
| `actor.look.eyeColor` | string | — | `"#1b1b2a"` | Actor eye colour (hex colour) |
| `actor.look.mouthColor` | string | — | `"#2a1b2a"` | Actor mouth colour (hex colour) |
| `actor.look.eyeRoughness` | number | min 0, max 1 | `0.4` | Actor eye material roughness (0..1) |
| `actor.look.mouthRoughness` | number | min 0, max 1 | `0.6` | Actor mouth material roughness (0..1) |
| `actor.look.earRoughness` | number | min 0, max 1 | `0.7` | Actor ear material roughness (0..1) |
| `actor.look.eyeWidthSegments` | integer | min 3, max 32 | `8` | Eye sphere longitude segments (count) |
| `actor.look.eyeHeightSegments` | integer | min 2, max 32 | `5` | Eye sphere latitude segments (count) |
| `actor.look.earSegments` | integer | min 3, max 32 | `8` | Ear cone radial segments (count) |
| `actor.look.largeEarScale` | number | min 0.1, max 4 | `1.5` | Large ear variant size (multiplier) |
| `actor.look.earTiltRad` | number | min 0, max 3.141592653589793 | `0.5` | Outward ear tilt (radians) |
| `actor.look.mouthVariantScales` | array<number> | length 3 | `[1,1.2,1.6]` | Mouth width scales for the three variants |
| `actor.look.emoteMouthScale` | number | min 0.1, max 4 | `2` | Emoting mouth height (multiplier) |
| `actor.look.minDetailPx` | number | min 0, max 128 | `8` | Projected body height below which face and equipment detail is culled (pixels) |
| `actor.look.minimumProjectionDepth` | number | min 0.000001, max 1 | `0.001` | Minimum perspective denominator for actor detail culling (metres) |
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
| `quality.diagnostics.minimumReadyFps` | number | min 0, max 240 | `12` | Minimum measured rendered FPS required for diagnostic readiness (frames/second) |
| `quality.diagnostics.passSampleWindow` | integer | min 2, max 3600 | `120` | Recent GPU pass frames retained for timing percentiles (count) |
| `quality.diagnostics.passMaxPending` | integer | min 1, max 512 | `64` | Maximum unresolved GPU timing spans before new spans pause (count) |
| `quality.diagnostics.gpuSampleWindow` | integer | min 2, max 3600 | `120` | Recent GPU frame samples retained for timing percentiles (count) |
| `quality.diagnostics.gpuMaxInFlight` | integer | min 1, max 64 | `4` | Maximum unresolved GPU frame timer queries (count) |
| `quality.diagnostics.frameWindow` | integer | min 2, max 3600 | `120` | Recent CPU frame samples retained for timing percentiles (count) |
| `quality.diagnostics.overlayRefreshMs` | number | min 16, max 5000 | `250` | Diagnostics overlay refresh cadence (milliseconds) |
| `quality.diagnostics.publishEveryFrames` | integer | min 1, max 120 | `6` | Frames between diagnostic snapshot publications (count) |
| `quality.diagnostics.fpsWindowMs` | number | min 100, max 10000 | `1000` | Wall-clock window used to estimate rendered frames per second (milliseconds) |
| `quality.frameDriver.introMs` | number | min 0, max 30000 | `5000` | Continuous rendering window for the intro (milliseconds) |
| `quality.frameDriver.minimumSettleMs` | number | min 0, max 5000 | `250` | Minimum rendering window after an input or state update (milliseconds) |
| `quality.frameDriver.diagnosticsHeartbeatMs` | number | min 16, max 5000 | `250` | Diagnostic invalidation cadence (milliseconds) |
| `quality.frameDriver.movingSpeed` | number | min 0, max 1 | `0.001` | Actor speed above which continuous rendering stays active (metres/second) |
| `quality.ultraMinRefreshRate` | number | min 1, max 500 | `90` | Minimum display refresh rate permitting automatic ultra quality (hertz) |
| `quality.defaultProfile` | "low" \| "medium" \| "high" \| "ultra" | — | `"medium"` | Profile used before the governor or the user picks one (profile id) |
| `quality.degradedRatio` | number | min 0, max 1 | `0.72` | Fraction of frameCapFps below which the governor steps down while auto is on (0..1) |
| `quality.recoverRatio` | number | min 0, max 1 | `0.92` | Fraction of frameCapFps above which the governor steps up while auto is on (0..1) |
| `quality.monitorFlipflops` | integer | min 1, max 20 | `3` | Consecutive up/down flips after which the governor stops adjusting (count) |
| `quality.profiles.low.lampLights` | integer | min 0, max 32 | `0` | Maximum lamp point lights rendered at once (count) |
| `quality.profiles.low.dpr` | number | min 0.5, max 3 | `1` | Device pixel ratio cap (multiplier) |
| `quality.profiles.low.shadows` | boolean | — | `false` | Directional shadow map on or off (flag) |
| `quality.profiles.low.shadowMapSize` | integer | min 256, max 8192 | `512` | Shadow map resolution (pixels, square) (count) |
| `quality.profiles.low.shadowRefreshHz` | integer | min 0, max 60 | `0` | Maximum shadow-map refreshes per second while actors move (count) |
| `quality.profiles.low.ao` | boolean | — | `false` | N8AO ambient occlusion pass on or off (flag) |
| `quality.profiles.low.aoQuality` | "off" \| "low" \| "medium" | — | `"off"` | Ambient-occlusion quality; off is the mount switch |
| `quality.profiles.low.bloom` | boolean | — | `false` | Selective bloom pass on or off (flag) |
| `quality.profiles.low.msaa` | integer | min 0, max 8 | `0` | Multisample anti-aliasing samples on the composer target; 0 disables (samples) (count) |
| `quality.profiles.low.labelBudget` | integer | min 0, max 200 | `3` | Label budget override for this profile (count) |
| `quality.profiles.low.frameCapFps` | integer | min 15, max 240 | `60` | Frame rate this profile is designed for; the governor derives its degraded threshold from it (count) |
| `quality.profiles.low.wobble` | boolean | — | `false` | Slime vertex wobble on or off (flag) |
| `quality.profiles.low.clouds` | boolean | — | `false` | Volumetric clouds on or off (flag) |
| `quality.profiles.low.terrainInnerRadius` | number | min 5, max 500 | `35` | Radius rendered at the terrain profile base resolution (metres) |
| `quality.profiles.low.terrainCellScale` | number | min 0.5, max 8 | `2` | Terrain sample spacing multiplier (multiplier) |
| `quality.profiles.low.vegetationDensityScale` | number | min 0, max 1 | `0.45` | Share of deterministic vegetation instances rendered (0..1) |
| `quality.profiles.low.weatherParticleScale` | number | min 0, max 1 | `0` | Share of weather particles rendered (0..1) |
| `quality.profiles.low.waterEnabled` | boolean | — | `false` | Whether the water surface is rendered (flag) |
| `quality.profiles.low.vegetationInstanceBudget` | integer | min 0, max 5000 | `300` | Maximum visible vegetation instances rendered at once (count) |
| `quality.profiles.medium.lampLights` | integer | min 0, max 32 | `2` | Maximum lamp point lights rendered at once (count) |
| `quality.profiles.medium.dpr` | number | min 0.5, max 3 | `1` | Device pixel ratio cap (multiplier) |
| `quality.profiles.medium.shadows` | boolean | — | `true` | Directional shadow map on or off (flag) |
| `quality.profiles.medium.shadowMapSize` | integer | min 256, max 8192 | `1024` | Shadow map resolution (pixels, square) (count) |
| `quality.profiles.medium.shadowRefreshHz` | integer | min 0, max 60 | `4` | Maximum shadow-map refreshes per second while actors move (count) |
| `quality.profiles.medium.ao` | boolean | — | `false` | N8AO ambient occlusion pass on or off (flag) |
| `quality.profiles.medium.aoQuality` | "off" \| "low" \| "medium" | — | `"off"` | Ambient-occlusion quality; off is the mount switch |
| `quality.profiles.medium.bloom` | boolean | — | `true` | Selective bloom pass on or off (flag) |
| `quality.profiles.medium.msaa` | integer | min 0, max 8 | `2` | Multisample anti-aliasing samples on the composer target; 0 disables (samples) (count) |
| `quality.profiles.medium.labelBudget` | integer | min 0, max 200 | `16` | Label budget override for this profile (count) |
| `quality.profiles.medium.frameCapFps` | integer | min 15, max 240 | `60` | Frame rate this profile is designed for; the governor derives its degraded threshold from it (count) |
| `quality.profiles.medium.wobble` | boolean | — | `true` | Slime vertex wobble on or off (flag) |
| `quality.profiles.medium.clouds` | boolean | — | `false` | Volumetric clouds on or off (flag) |
| `quality.profiles.medium.terrainInnerRadius` | number | min 5, max 500 | `50` | Radius rendered at the terrain profile base resolution (metres) |
| `quality.profiles.medium.terrainCellScale` | number | min 0.5, max 8 | `1.5` | Terrain sample spacing multiplier (multiplier) |
| `quality.profiles.medium.vegetationDensityScale` | number | min 0, max 1 | `0.7` | Share of deterministic vegetation instances rendered (0..1) |
| `quality.profiles.medium.weatherParticleScale` | number | min 0, max 1 | `0.5` | Share of weather particles rendered (0..1) |
| `quality.profiles.medium.waterEnabled` | boolean | — | `true` | Whether the water surface is rendered (flag) |
| `quality.profiles.medium.vegetationInstanceBudget` | integer | min 0, max 5000 | `600` | Maximum visible vegetation instances rendered at once (count) |
| `quality.profiles.high.lampLights` | integer | min 0, max 32 | `1` | Maximum lamp point lights rendered at once (count) |
| `quality.profiles.high.dpr` | number | min 0.5, max 3 | `1` | Device pixel ratio cap (multiplier) |
| `quality.profiles.high.shadows` | boolean | — | `true` | Directional shadow map on or off (flag) |
| `quality.profiles.high.shadowMapSize` | integer | min 256, max 8192 | `2048` | Shadow map resolution (pixels, square) (count) |
| `quality.profiles.high.shadowRefreshHz` | integer | min 0, max 60 | `4` | Maximum shadow-map refreshes per second while actors move (count) |
| `quality.profiles.high.ao` | boolean | — | `true` | N8AO ambient occlusion pass on or off (flag) |
| `quality.profiles.high.aoQuality` | "off" \| "low" \| "medium" | — | `"low"` | Ambient-occlusion quality; off is the mount switch |
| `quality.profiles.high.bloom` | boolean | — | `true` | Selective bloom pass on or off (flag) |
| `quality.profiles.high.msaa` | integer | min 0, max 8 | `4` | Multisample anti-aliasing samples on the composer target; 0 disables (samples) (count) |
| `quality.profiles.high.labelBudget` | integer | min 0, max 200 | `24` | Label budget override for this profile (count) |
| `quality.profiles.high.frameCapFps` | integer | min 15, max 240 | `60` | Frame rate this profile is designed for; the governor derives its degraded threshold from it (count) |
| `quality.profiles.high.wobble` | boolean | — | `true` | Slime vertex wobble on or off (flag) |
| `quality.profiles.high.clouds` | boolean | — | `true` | Volumetric clouds on or off (flag) |
| `quality.profiles.high.terrainInnerRadius` | number | min 5, max 500 | `65` | Radius rendered at the terrain profile base resolution (metres) |
| `quality.profiles.high.terrainCellScale` | number | min 0.5, max 8 | `1` | Terrain sample spacing multiplier (multiplier) |
| `quality.profiles.high.vegetationDensityScale` | number | min 0, max 1 | `0.85` | Share of deterministic vegetation instances rendered (0..1) |
| `quality.profiles.high.weatherParticleScale` | number | min 0, max 1 | `0.8` | Share of weather particles rendered (0..1) |
| `quality.profiles.high.waterEnabled` | boolean | — | `true` | Whether the water surface is rendered (flag) |
| `quality.profiles.high.vegetationInstanceBudget` | integer | min 0, max 5000 | `900` | Maximum visible vegetation instances rendered at once (count) |
| `quality.profiles.ultra.lampLights` | integer | min 0, max 32 | `1` | Maximum lamp point lights rendered at once (count) |
| `quality.profiles.ultra.dpr` | number | min 0.5, max 3 | `1.5` | Device pixel ratio cap (multiplier) |
| `quality.profiles.ultra.shadows` | boolean | — | `true` | Directional shadow map on or off (flag) |
| `quality.profiles.ultra.shadowMapSize` | integer | min 256, max 8192 | `2048` | Shadow map resolution (pixels, square) (count) |
| `quality.profiles.ultra.shadowRefreshHz` | integer | min 0, max 60 | `4` | Maximum shadow-map refreshes per second while actors move (count) |
| `quality.profiles.ultra.ao` | boolean | — | `false` | N8AO ambient occlusion pass on or off (flag) |
| `quality.profiles.ultra.aoQuality` | "off" \| "low" \| "medium" | — | `"off"` | Ambient-occlusion quality; off is the mount switch |
| `quality.profiles.ultra.bloom` | boolean | — | `true` | Selective bloom pass on or off (flag) |
| `quality.profiles.ultra.msaa` | integer | min 0, max 8 | `2` | Multisample anti-aliasing samples on the composer target; 0 disables (samples) (count) |
| `quality.profiles.ultra.labelBudget` | integer | min 0, max 200 | `40` | Label budget override for this profile (count) |
| `quality.profiles.ultra.frameCapFps` | integer | min 15, max 240 | `120` | Frame rate this profile is designed for; the governor derives its degraded threshold from it (count) |
| `quality.profiles.ultra.wobble` | boolean | — | `true` | Slime vertex wobble on or off (flag) |
| `quality.profiles.ultra.clouds` | boolean | — | `true` | Volumetric clouds on or off (flag) |
| `quality.profiles.ultra.terrainInnerRadius` | number | min 5, max 500 | `90` | Radius rendered at the terrain profile base resolution (metres) |
| `quality.profiles.ultra.terrainCellScale` | number | min 0.5, max 8 | `1` | Terrain sample spacing multiplier (multiplier) |
| `quality.profiles.ultra.vegetationDensityScale` | number | min 0, max 1 | `1` | Share of deterministic vegetation instances rendered (0..1) |
| `quality.profiles.ultra.weatherParticleScale` | number | min 0, max 1 | `1` | Share of weather particles rendered (0..1) |
| `quality.profiles.ultra.waterEnabled` | boolean | — | `true` | Whether the water surface is rendered (flag) |
| `quality.profiles.ultra.vegetationInstanceBudget` | integer | min 0, max 5000 | `1200` | Maximum visible vegetation instances rendered at once (count) |

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
| `editor.handleLift` | number | min 0, max 1 | `0.05` | Room edit handle height above its interaction plane (metres) |
| `editor.handleOpacity` | number | min 0, max 1 | `0.28` | Unselected room handle opacity (0..1) |
| `editor.selectedOpacity` | number | min 0, max 1 | `0.45` | Selected room handle opacity (0..1) |
| `editor.handleColor` | string | — | `"#ffffff"` | Unselected room handle colour (hex colour) |
| `editor.selectedColor` | string | — | `"#7c9cff"` | Selected room handle colour (hex colour) |
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
| `budgets.weatherPixelTolerance` | number | min 0, max 1 | `0.02` | Per-pixel colour tolerance when comparing distinct weather states; lower values detect subtler palette changes (0..1) |
| `budgets.framing.minFill` | number | min 0, max 1 | `0.7` | Smallest share of the viewport the layout outline may occupy on the hero pose (0..1) |
| `budgets.framing.maxFill` | number | min 0, max 1 | `0.97` | Largest share of the viewport the layout outline may occupy on the hero pose (0..1) |
| `budgets.scenes.park.low.drawCalls` | integer | min 1, max 5000 | `50` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.park.low.triangles` | integer | min 1, max 50000000 | `61000` | Maximum triangles per frame (count) |
| `budgets.scenes.park.low.p95Ms` | number | min 1, max 200 | `6.8` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.park.low.provenance.actors` | integer | min 1, max 100000 | `25` | Pinned synthetic actor count used for calibration (count) |
| `budgets.scenes.park.low.provenance.gpu` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | GPU renderer string used for calibration (renderer name) |
| `budgets.scenes.park.low.provenance.renderer` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | Exact hardware renderer string reported by WebGL |
| `budgets.scenes.park.low.provenance.gpuTier` | "igpu" \| "dgpu" | — | `"igpu"` | Hardware tier used for the gating calibration |
| `budgets.scenes.park.low.provenance.deviceScaleFactor` | number | min 0.5, max 3 | `1` | Device pixel ratio used for calibration |
| `budgets.scenes.park.low.provenance.measuredP95Ms` | number | — | `5.89299` | Observed GPU p95 before headroom was applied |
| `budgets.scenes.park.low.provenance.target` | boolean | — | `false` | True when p95Ms is a delivery target rather than observed-plus-headroom |
| `budgets.scenes.park.low.provenance.calibratedAt` | string | — | `"2026-09-05"` | Calibration date (ISO 8601 date) |
| `budgets.scenes.park.low.provenance.method` | string | — | `"gpu-timer; igpu; dsf 1; worst of 16 captures; observed-plus-15pct-headroom"` | Frame-time measurement method (method name) |
| `budgets.scenes.park.medium.drawCalls` | integer | min 1, max 5000 | `74` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.park.medium.triangles` | integer | min 1, max 50000000 | `82000` | Maximum triangles per frame (count) |
| `budgets.scenes.park.medium.p95Ms` | number | min 1, max 200 | `10.4` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.park.medium.provenance.actors` | integer | min 1, max 100000 | `25` | Pinned synthetic actor count used for calibration (count) |
| `budgets.scenes.park.medium.provenance.gpu` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | GPU renderer string used for calibration (renderer name) |
| `budgets.scenes.park.medium.provenance.renderer` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | Exact hardware renderer string reported by WebGL |
| `budgets.scenes.park.medium.provenance.gpuTier` | "igpu" \| "dgpu" | — | `"igpu"` | Hardware tier used for the gating calibration |
| `budgets.scenes.park.medium.provenance.deviceScaleFactor` | number | min 0.5, max 3 | `1` | Device pixel ratio used for calibration |
| `budgets.scenes.park.medium.provenance.measuredP95Ms` | number | — | `9.02497` | Observed GPU p95 before headroom was applied |
| `budgets.scenes.park.medium.provenance.target` | boolean | — | `false` | True when p95Ms is a delivery target rather than observed-plus-headroom |
| `budgets.scenes.park.medium.provenance.calibratedAt` | string | — | `"2026-09-05"` | Calibration date (ISO 8601 date) |
| `budgets.scenes.park.medium.provenance.method` | string | — | `"gpu-timer; igpu; dsf 1; worst of 16 captures; observed-plus-15pct-headroom"` | Frame-time measurement method (method name) |
| `budgets.scenes.park.high.drawCalls` | integer | min 1, max 5000 | `98` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.park.high.triangles` | integer | min 1, max 50000000 | `135000` | Maximum triangles per frame (count) |
| `budgets.scenes.park.high.p95Ms` | number | min 1, max 200 | `21` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.park.high.provenance.actors` | integer | min 1, max 100000 | `25` | Pinned synthetic actor count used for calibration (count) |
| `budgets.scenes.park.high.provenance.gpu` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | GPU renderer string used for calibration (renderer name) |
| `budgets.scenes.park.high.provenance.renderer` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | Exact hardware renderer string reported by WebGL |
| `budgets.scenes.park.high.provenance.gpuTier` | "igpu" \| "dgpu" | — | `"igpu"` | Hardware tier used for the gating calibration |
| `budgets.scenes.park.high.provenance.deviceScaleFactor` | number | min 0.5, max 3 | `1` | Device pixel ratio used for calibration |
| `budgets.scenes.park.high.provenance.measuredP95Ms` | number | — | `18.26041` | Observed GPU p95 before headroom was applied |
| `budgets.scenes.park.high.provenance.target` | boolean | — | `false` | True when p95Ms is a delivery target rather than observed-plus-headroom |
| `budgets.scenes.park.high.provenance.calibratedAt` | string | — | `"2026-09-05"` | Calibration date (ISO 8601 date) |
| `budgets.scenes.park.high.provenance.method` | string | — | `"gpu-timer; igpu; dsf 1; worst of 16 captures; observed-plus-15pct-headroom"` | Frame-time measurement method (method name) |
| `budgets.scenes.park.ultra.drawCalls` | integer | min 1, max 5000 | `75` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.park.ultra.triangles` | integer | min 1, max 50000000 | `128000` | Maximum triangles per frame (count) |
| `budgets.scenes.park.ultra.p95Ms` | number | min 1, max 200 | `20` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.park.ultra.provenance.actors` | integer | min 1, max 100000 | `25` | Pinned synthetic actor count used for calibration (count) |
| `budgets.scenes.park.ultra.provenance.gpu` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | GPU renderer string used for calibration (renderer name) |
| `budgets.scenes.park.ultra.provenance.renderer` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | Exact hardware renderer string reported by WebGL |
| `budgets.scenes.park.ultra.provenance.gpuTier` | "igpu" \| "dgpu" | — | `"igpu"` | Hardware tier used for the gating calibration |
| `budgets.scenes.park.ultra.provenance.deviceScaleFactor` | number | min 0.5, max 3 | `1.5` | Device pixel ratio used for calibration |
| `budgets.scenes.park.ultra.provenance.measuredP95Ms` | number | — | `17.40517` | Observed GPU p95 before headroom was applied |
| `budgets.scenes.park.ultra.provenance.target` | boolean | — | `false` | True when p95Ms is a delivery target rather than observed-plus-headroom |
| `budgets.scenes.park.ultra.provenance.calibratedAt` | string | — | `"2026-09-05"` | Calibration date (ISO 8601 date) |
| `budgets.scenes.park.ultra.provenance.method` | string | — | `"gpu-timer; igpu; dsf 1.5; worst of 16 captures; observed-plus-15pct-headroom"` | Frame-time measurement method (method name) |
| `budgets.scenes.office.low.drawCalls` | integer | min 1, max 5000 | `50` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.office.low.triangles` | integer | min 1, max 50000000 | `72000` | Maximum triangles per frame (count) |
| `budgets.scenes.office.low.p95Ms` | number | min 1, max 200 | `5.8` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.office.low.provenance.actors` | integer | min 1, max 100000 | `25` | Pinned synthetic actor count used for calibration (count) |
| `budgets.scenes.office.low.provenance.gpu` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | GPU renderer string used for calibration (renderer name) |
| `budgets.scenes.office.low.provenance.renderer` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | Exact hardware renderer string reported by WebGL |
| `budgets.scenes.office.low.provenance.gpuTier` | "igpu" \| "dgpu" | — | `"igpu"` | Hardware tier used for the gating calibration |
| `budgets.scenes.office.low.provenance.deviceScaleFactor` | number | min 0.5, max 3 | `1` | Device pixel ratio used for calibration |
| `budgets.scenes.office.low.provenance.measuredP95Ms` | number | — | `5.07193` | Observed GPU p95 before headroom was applied |
| `budgets.scenes.office.low.provenance.target` | boolean | — | `false` | True when p95Ms is a delivery target rather than observed-plus-headroom |
| `budgets.scenes.office.low.provenance.calibratedAt` | string | — | `"2026-09-04"` | Calibration date (ISO 8601 date) |
| `budgets.scenes.office.low.provenance.method` | string | — | `"gpu-timer; igpu; dsf 1; worst of 16 captures; observed-plus-15pct-headroom"` | Frame-time measurement method (method name) |
| `budgets.scenes.office.medium.drawCalls` | integer | min 1, max 5000 | `74` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.office.medium.triangles` | integer | min 1, max 50000000 | `90000` | Maximum triangles per frame (count) |
| `budgets.scenes.office.medium.p95Ms` | number | min 1, max 200 | `10.2` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.office.medium.provenance.actors` | integer | min 1, max 100000 | `25` | Pinned synthetic actor count used for calibration (count) |
| `budgets.scenes.office.medium.provenance.gpu` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | GPU renderer string used for calibration (renderer name) |
| `budgets.scenes.office.medium.provenance.renderer` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | Exact hardware renderer string reported by WebGL |
| `budgets.scenes.office.medium.provenance.gpuTier` | "igpu" \| "dgpu" | — | `"igpu"` | Hardware tier used for the gating calibration |
| `budgets.scenes.office.medium.provenance.deviceScaleFactor` | number | min 0.5, max 3 | `1` | Device pixel ratio used for calibration |
| `budgets.scenes.office.medium.provenance.measuredP95Ms` | number | — | `8.84679` | Observed GPU p95 before headroom was applied |
| `budgets.scenes.office.medium.provenance.target` | boolean | — | `false` | True when p95Ms is a delivery target rather than observed-plus-headroom |
| `budgets.scenes.office.medium.provenance.calibratedAt` | string | — | `"2026-09-04"` | Calibration date (ISO 8601 date) |
| `budgets.scenes.office.medium.provenance.method` | string | — | `"gpu-timer; igpu; dsf 1; worst of 16 captures; observed-plus-15pct-headroom"` | Frame-time measurement method (method name) |
| `budgets.scenes.office.high.drawCalls` | integer | min 1, max 5000 | `96` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.office.high.triangles` | integer | min 1, max 50000000 | `141000` | Maximum triangles per frame (count) |
| `budgets.scenes.office.high.p95Ms` | number | min 1, max 200 | `20.2` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.office.high.provenance.actors` | integer | min 1, max 100000 | `25` | Pinned synthetic actor count used for calibration (count) |
| `budgets.scenes.office.high.provenance.gpu` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | GPU renderer string used for calibration (renderer name) |
| `budgets.scenes.office.high.provenance.renderer` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | Exact hardware renderer string reported by WebGL |
| `budgets.scenes.office.high.provenance.gpuTier` | "igpu" \| "dgpu" | — | `"igpu"` | Hardware tier used for the gating calibration |
| `budgets.scenes.office.high.provenance.deviceScaleFactor` | number | min 0.5, max 3 | `1` | Device pixel ratio used for calibration |
| `budgets.scenes.office.high.provenance.measuredP95Ms` | number | — | `17.58624` | Observed GPU p95 before headroom was applied |
| `budgets.scenes.office.high.provenance.target` | boolean | — | `false` | True when p95Ms is a delivery target rather than observed-plus-headroom |
| `budgets.scenes.office.high.provenance.calibratedAt` | string | — | `"2026-09-04"` | Calibration date (ISO 8601 date) |
| `budgets.scenes.office.high.provenance.method` | string | — | `"gpu-timer; igpu; dsf 1; worst of 16 captures; observed-plus-15pct-headroom"` | Frame-time measurement method (method name) |
| `budgets.scenes.office.ultra.drawCalls` | integer | min 1, max 5000 | `74` | Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run (count) |
| `budgets.scenes.office.ultra.triangles` | integer | min 1, max 50000000 | `138000` | Maximum triangles per frame (count) |
| `budgets.scenes.office.ultra.p95Ms` | number | min 1, max 200 | `19.2` | Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds) |
| `budgets.scenes.office.ultra.provenance.actors` | integer | min 1, max 100000 | `25` | Pinned synthetic actor count used for calibration (count) |
| `budgets.scenes.office.ultra.provenance.gpu` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | GPU renderer string used for calibration (renderer name) |
| `budgets.scenes.office.ultra.provenance.renderer` | string | — | `"ANGLE (AMD, AMD Ryzen 9 7950X 16-Core Processor (radeonsi raphael_mendocino LLVM 20.1.2), OpenGL ES 3.2)"` | Exact hardware renderer string reported by WebGL |
| `budgets.scenes.office.ultra.provenance.gpuTier` | "igpu" \| "dgpu" | — | `"igpu"` | Hardware tier used for the gating calibration |
| `budgets.scenes.office.ultra.provenance.deviceScaleFactor` | number | min 0.5, max 3 | `1.5` | Device pixel ratio used for calibration |
| `budgets.scenes.office.ultra.provenance.measuredP95Ms` | number | — | `16.67796` | Observed GPU p95 before headroom was applied |
| `budgets.scenes.office.ultra.provenance.target` | boolean | — | `false` | True when p95Ms is a delivery target rather than observed-plus-headroom |
| `budgets.scenes.office.ultra.provenance.calibratedAt` | string | — | `"2026-09-04"` | Calibration date (ISO 8601 date) |
| `budgets.scenes.office.ultra.provenance.method` | string | — | `"gpu-timer; igpu; dsf 1.5; worst of 16 captures; observed-plus-15pct-headroom"` | Frame-time measurement method (method name) |

### `scenes`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `scenes.park.centre.source` | "floorplate" | — | `—` | Geometry that owns the centre region |
| `scenes.park.centre.margin` | number | min 0, max 40 | `—` | Centre extension past the floorplate (metres) |
| `scenes.park.centre.blend` | number | min 0, max 40 | `—` | Smooth transition back to landscape (metres) |
| `scenes.park.centre.terrain.radius` | number | min 10, max 500 | `—` | Radius of the generated terrain field (metres) |
| `scenes.park.centre.terrain.cellSize` | number | min 0.25, max 8 | `—` | Spacing between terrain field samples (metres) |
| `scenes.park.centre.terrain.amplitude` | number | min 0, max 20 | `—` | Maximum absolute terrain elevation (metres) |
| `scenes.park.centre.terrain.frequency` | number | min 0.001, max 1 | `—` | Base terrain noise frequency (cycles per metre) |
| `scenes.park.centre.terrain.detailAmplitude` | number | min 0, max 5 | `—` | Higher-frequency surface-detail amplitude (metres) |
| `scenes.park.centre.terrain.detailFrequency` | number | min 0.001, max 2 | `—` | Surface-detail noise frequency (cycles per metre) |
| `scenes.park.centre.terrain.octaves` | integer | min 1, max 8 | `—` | Fractal noise octaves used for height and moisture (count) |
| `scenes.park.centre.terrain.lacunarity` | number | min 1, max 4 | `—` | Frequency multiplier between terrain noise octaves (multiplier) |
| `scenes.park.centre.terrain.gain` | number | min 0, max 1 | `—` | Amplitude multiplier between terrain noise octaves (0..1) |
| `scenes.park.centre.terrain.moistureFrequency` | number | min 0.001, max 1 | `—` | Base moisture noise frequency (cycles per metre) |
| `scenes.park.centre.terrain.moistureWarp` | number | min 0, max 50 | `—` | Domain-warp distance applied to moisture sampling (metres) |
| `scenes.park.centre.terrain.falloffStart` | number | min 0, max 1 | `—` | Fraction of terrain radius where elevation begins fading to zero (0..1) |
| `scenes.park.centre.terrain.waterLevel` | number | min -20, max 20 | `—` | Water surface elevation (metres) |
| `scenes.park.centre.terrain.shoreMargin` | number | min 0, max 20 | `—` | Dry navigation margin around water (metres) |
| `scenes.park.centre.terrain.waterSurfaceLift` | number | min 0, max 0.2 | `—` | Water surface lift above its terrain threshold (metres) |
| `scenes.park.centre.terrain.wetShoreWidth` | number | min 0.1, max 10 | `—` | Width of terrain darkening immediately inside water (metres) |
| `scenes.park.centre.terrain.wetShoreDarkening` | number | min 0, max 1 | `—` | Maximum terrain darkening immediately inside water (0..1) |
| `scenes.park.centre.terrain.maxSiteSlope` | number | min 0, max 1.5707963267948966 | `—` | Steepest ground eligible for a team site (radians) |
| `scenes.park.centre.terrain.maxWalkSlope` | number | min 0, max 1.5707963267948966 | `—` | Steepest ground eligible for navigation (radians) |
| `scenes.park.centre.terrain.kerbWidth` | number | min 0.25, max 10 | `—` | Width over which a level site pad blends into terrain (metres) |
| `scenes.park.centre.terrain.pathWidth` | number | min 0.25, max 10 | `—` | Width of paths painted into terrain colour (metres) |
| `scenes.park.centre.terrain.innerCellSize` | number | min 0.25, max 4 | `—` | Terrain mesh spacing near the settlement (metres) |
| `scenes.park.centre.terrain.innerRadius` | number | min 5, max 300 | `—` | Radius of the dense inner terrain mesh (metres) |
| `scenes.park.centre.terrain.ringFalloff` | number | min 1, max 8 | `—` | Terrain mesh spacing multiplier beyond the inner ring (multiplier) |
| `scenes.park.centre.terrain.tileSize` | number | min 4, max 100 | `—` | Side length of one vegetation culling tile (metres) |
| `scenes.park.centre.terrain.moistureBasinDepth` | number | min 0, max 5 | `—` | Maximum moisture bias subtracted when classifying water (metres) |
| `scenes.park.centre.terrain.shoreMinGrade` | number | min 0.001, max 1 | `—` | Minimum grade used to estimate horizontal shore distance (rise over run) |
| `scenes.park.centre.terrain.padClearance` | number | min 0, max 5 | `—` | Minimum terrace elevation above the configured water surface (metres) |
| `scenes.park.centre.terrain.siteLevelTolerance` | number | min 0.001, max 1 | `—` | Maximum elevation variation allowed across a site pad (metres) |
| `scenes.park.centre.biomeSet` | "park" \| "office" | — | `—` | Biome set inside the centre; the base set remains outside (identifier) |
| `scenes.park.centre.levelTo` | "plateMean" \| "none" | — | `—` | Level to the natural plate mean, raised only for dry clearance, or retain relief |
| `scenes.park.centre.maxBoundaryGrade` | number | min 0.01, max 4 | `—` | Maximum permitted height gradient across the centre transition (metres per metre) |
| `scenes.park.emissive` | object | — | `{"hearth":"#ffab52","lamp":"#ffd9a0"}` | Emission by rendered prop slot; omitted slots do not emit (hex colours) |
| `scenes.park.biomeSet` | "park" \| "office" | — | `"park"` | Biome set that supplies terrain colours and ground-bound props |
| `scenes.park.assetSet` | string | — | `"park"` | Directory under public/assets/world holding this scene props |
| `scenes.office.centre.source` | "floorplate" | — | `"floorplate"` | Geometry that owns the centre region |
| `scenes.office.centre.margin` | number | min 0, max 40 | `6` | Centre extension past the floorplate (metres) |
| `scenes.office.centre.blend` | number | min 0, max 40 | `4` | Smooth transition back to landscape (metres) |
| `scenes.office.centre.terrain.radius` | number | min 10, max 500 | `—` | Radius of the generated terrain field (metres) |
| `scenes.office.centre.terrain.cellSize` | number | min 0.25, max 8 | `—` | Spacing between terrain field samples (metres) |
| `scenes.office.centre.terrain.amplitude` | number | min 0, max 20 | `0` | Maximum absolute terrain elevation (metres) |
| `scenes.office.centre.terrain.frequency` | number | min 0.001, max 1 | `—` | Base terrain noise frequency (cycles per metre) |
| `scenes.office.centre.terrain.detailAmplitude` | number | min 0, max 5 | `0` | Higher-frequency surface-detail amplitude (metres) |
| `scenes.office.centre.terrain.detailFrequency` | number | min 0.001, max 2 | `—` | Surface-detail noise frequency (cycles per metre) |
| `scenes.office.centre.terrain.octaves` | integer | min 1, max 8 | `—` | Fractal noise octaves used for height and moisture (count) |
| `scenes.office.centre.terrain.lacunarity` | number | min 1, max 4 | `—` | Frequency multiplier between terrain noise octaves (multiplier) |
| `scenes.office.centre.terrain.gain` | number | min 0, max 1 | `—` | Amplitude multiplier between terrain noise octaves (0..1) |
| `scenes.office.centre.terrain.moistureFrequency` | number | min 0.001, max 1 | `—` | Base moisture noise frequency (cycles per metre) |
| `scenes.office.centre.terrain.moistureWarp` | number | min 0, max 50 | `—` | Domain-warp distance applied to moisture sampling (metres) |
| `scenes.office.centre.terrain.falloffStart` | number | min 0, max 1 | `—` | Fraction of terrain radius where elevation begins fading to zero (0..1) |
| `scenes.office.centre.terrain.waterLevel` | number | min -20, max 20 | `—` | Water surface elevation (metres) |
| `scenes.office.centre.terrain.shoreMargin` | number | min 0, max 20 | `—` | Dry navigation margin around water (metres) |
| `scenes.office.centre.terrain.waterSurfaceLift` | number | min 0, max 0.2 | `—` | Water surface lift above its terrain threshold (metres) |
| `scenes.office.centre.terrain.wetShoreWidth` | number | min 0.1, max 10 | `—` | Width of terrain darkening immediately inside water (metres) |
| `scenes.office.centre.terrain.wetShoreDarkening` | number | min 0, max 1 | `—` | Maximum terrain darkening immediately inside water (0..1) |
| `scenes.office.centre.terrain.maxSiteSlope` | number | min 0, max 1.5707963267948966 | `—` | Steepest ground eligible for a team site (radians) |
| `scenes.office.centre.terrain.maxWalkSlope` | number | min 0, max 1.5707963267948966 | `—` | Steepest ground eligible for navigation (radians) |
| `scenes.office.centre.terrain.kerbWidth` | number | min 0.25, max 10 | `—` | Width over which a level site pad blends into terrain (metres) |
| `scenes.office.centre.terrain.pathWidth` | number | min 0.25, max 10 | `—` | Width of paths painted into terrain colour (metres) |
| `scenes.office.centre.terrain.innerCellSize` | number | min 0.25, max 4 | `—` | Terrain mesh spacing near the settlement (metres) |
| `scenes.office.centre.terrain.innerRadius` | number | min 5, max 300 | `—` | Radius of the dense inner terrain mesh (metres) |
| `scenes.office.centre.terrain.ringFalloff` | number | min 1, max 8 | `—` | Terrain mesh spacing multiplier beyond the inner ring (multiplier) |
| `scenes.office.centre.terrain.tileSize` | number | min 4, max 100 | `—` | Side length of one vegetation culling tile (metres) |
| `scenes.office.centre.terrain.moistureBasinDepth` | number | min 0, max 5 | `—` | Maximum moisture bias subtracted when classifying water (metres) |
| `scenes.office.centre.terrain.shoreMinGrade` | number | min 0.001, max 1 | `—` | Minimum grade used to estimate horizontal shore distance (rise over run) |
| `scenes.office.centre.terrain.padClearance` | number | min 0, max 5 | `—` | Minimum terrace elevation above the configured water surface (metres) |
| `scenes.office.centre.terrain.siteLevelTolerance` | number | min 0.001, max 1 | `—` | Maximum elevation variation allowed across a site pad (metres) |
| `scenes.office.centre.biomeSet` | "park" \| "office" | — | `"office"` | Biome set inside the centre; the base set remains outside (identifier) |
| `scenes.office.centre.levelTo` | "plateMean" \| "none" | — | `"plateMean"` | Level to the natural plate mean, raised only for dry clearance, or retain relief |
| `scenes.office.centre.maxBoundaryGrade` | number | min 0.01, max 4 | `1` | Maximum permitted height gradient across the centre transition (metres per metre) |
| `scenes.office.emissive` | object | — | `{"lamp":"#ffe4bc"}` | Emission by rendered prop slot; omitted slots do not emit (hex colours) |
| `scenes.office.biomeSet` | "park" \| "office" | — | `"park"` | Biome set that supplies terrain colours and ground-bound props |
| `scenes.office.assetSet` | string | — | `"office"` | Directory under public/assets/world holding this scene props |

### `biomes`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `biomes.park.assetSet` | string | — | `"park"` | Baked asset set for landscape vegetation |
| `biomes.park.propScale` | number | min 0.1, max 10 | `1.8` | Landscape prop scale (world metres per asset unit) |
| `biomes.park.treeScale` | number | min 0.1, max 10 | `1.5` | Additional scale for landscape trees (multiplier) |
| `biomes.office.assetSet` | string | — | `"office"` | Baked asset set for landscape vegetation |
| `biomes.office.propScale` | number | min 0.1, max 10 | `1.7` | Landscape prop scale (world metres per asset unit) |
| `biomes.office.treeScale` | number | min 0.1, max 10 | `1` | Additional scale for landscape trees (multiplier) |

### `vegetationEntry`

| Lever | Type | Bounds | Default | Effect |
|---|---|---|---|---|
| `vegetationEntry.density` | number | min 0, max 1 | `—` | Prop density (instances per square metre) |
| `vegetationEntry.class` | "tree" \| "shrub" \| "ground" | — | `—` | Vegetation class controlling spacing and navigation |
| `vegetationEntry.scaleRef` | "tree" \| "prop" | — | `—` | Scene scale multiplier applied to this prop |

<!-- world-tuning:end -->
