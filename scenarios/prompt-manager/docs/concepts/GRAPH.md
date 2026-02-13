# Relationship Graph

The relationship graph maps how teams, agents, skills, and CLI tools connect to each other. It scans store files for references, builds a directed graph, and exposes analytical queries that surface structural problems like orphaned skills, empty teams, or circular dependencies.

## Why It Exists

As the number of skills, agents, and teams grows, it becomes difficult to answer questions like:

- Which skills are never referenced by any agent?
- Which agents have no skills assigned?
- Are there circular dependencies between skills?
- Which nodes are the most connected (most influential)?

The graph automates these answers by scanning markdown content for references and computing health scores.

## Node Types

[CODE: api/graph/models.go]

| Type | Description | Source |
|------|-------------|--------|
| `team` | Organizational unit grouping agents | Team store (`store/teams/`) |
| `agent` | AI entity that performs work | Agent store (`store/agents/`) |
| `skill` | Reusable prompt/capability | Skill store (`store/skills/packs/`) |
| `cli` | CLI tool referenced in skill content | Extracted from `code-usage` edges |

CLI nodes are synthetic — they are created when a `code-usage` edge points to a target that doesn't exist as a skill, agent, or team.

## Edge Types

| Kind | Direction | Meaning | Detected By |
|------|-----------|---------|-------------|
| `membership` | team → agent | Team contains agent | Team-member relation store |
| `cli-read` | agent → skill | Agent reads skill via `prompt-manager skills read <id>` | Regex scan of agent markdown |
| `bold-listed` | agent/skill → skill | Skill referenced as `**kebab-case-id**` in markdown | Regex scan |
| `path-ref` | agent/skill → skill | Skill referenced via filesystem path (`store/skills/packs/...`) | Regex scan |
| `default-scope` | skill → skill | Skill declares another as its `DefaultScope` | skill.json field |
| `code-usage` | skill/agent → cli | Node references a CLI tool in its content | CLIDetector |

## How the Graph Is Built

[CODE: api/graph/builder.go]

The build pipeline runs in four stages:

```
1. Collect nodes     ← Read teams, agents, skills from store directories
        │
        ▼
2. Scan for edges    ← Regex-scan markdown files for references between entities
        │
        ▼
3. Extract CLI nodes ← Any code-usage edge targeting a non-existent node creates a CLI node
        │
        ▼
4. Score nodes       ← Compute weighted health score per node
```

### Reference Detection

[CODE: api/graph/scanner.go]

The scanner uses four regex patterns to extract edges from markdown content:

1. **CLI read commands** — `prompt-manager skills read <ids>` produces `cli-read` edges
2. **Bold-listed IDs** — `**skill-id**` produces `bold-listed` edges
3. **Filesystem paths** — `store/skills/packs/{pack}/{id}/SKILL.md` produces `path-ref` edges
4. **Default scope** — `DefaultScope` field in `skill.json` produces `default-scope` edges

Edges are deduplicated per `(skillID, edgeKind)` pair to prevent duplicates within the same file.

### Code Detection

[CODE: api/graph/code_detector.go]

The `CLIDetector` scans agent, team, and skill content for CLI tool references. It handles piped and chained commands by splitting backtick content on `|`, `||`, `&&`, and `;`, classifying each segment independently:

| Category | When | Example |
|----------|------|---------|
| `CodeScenarioCLI` | First word is in the `scenarioCLIs` map (`vrooli`, `prompt-manager`, etc.) | `` `vrooli scenario start foo` `` |
| `CodeExternalTool` | First word is NOT a known scenario CLI | `` `grep -r pattern .` `` |
| `CodeScript` | File path with script extension (.sh, .py, .js, .ts, .rb, .pl) | `scripts/deploy.sh` |
| `CodeAPICall` | Bare HTTP pattern (GET/POST/PUT/DELETE + URL) | `GET https://api.example.com` |

**Dynamic scenario discovery:** The detector receives all scenario folder names automatically via `discoverScenarioNames()` in `main.go`, which walks the `scenarios/` directory at startup. This ensures every scenario CLI (e.g., `visited-tracker`, `app-monitor`) is correctly classified as `CodeScenarioCLI` without hardcoding.

**Multi-line backtick handling:** Backtick commands are matched on the full content (not per-line), so multi-line spans with `\` continuation are captured. Line numbers are calculated from the byte offset of the opening backtick in the stripped content.

**Code fence stripping:** Triple-backtick fenced blocks (` ``` ... ``` `) are replaced with equivalent newlines before backtick matching, preventing false positives from code examples. The replacement preserves line numbering.

**Edge creation policy:**
- `CodeScenarioCLI`, `CodeExternalTool`, and `CodeScript` produce `code-usage` edges in the graph.
- `CodeAPICall` is intentionally excluded — it documents API endpoints, not tool invocations.
- `prompt-manager skill read` commands are excluded — those are Skill→Skill relations handled as `cli-read` edges.

**Pipe splitting example:** `` `vrooli scenario start foo | grep error` `` produces two references — `CodeScenarioCLI` for `vrooli` and `CodeExternalTool` for `grep`. Both become edges in the graph.

## Health Scoring

[CODE: api/graph/scoring.go]

Team/Agent/Skill nodes receive a composite health score (0.0–1.0) computed as a weighted average of individual factors:

| Factor | Weight | What It Measures |
|--------|--------|------------------|
| `outgoing-edges` | 1.0 | Number of edges where node is the source. 5+ edges = 1.0. |
| `incoming-edges` | 1.0 | Number of edges where node is the target. 5+ references = 1.0. |
| `code-usage` | 0.5 | 3-level: 1.0 if only Vrooli CLIs, 0.5 if no tool usage (neutral), 0.1 if any external tool or script usage (penalty). |
| `recent-activity` | 0.5 | Currently returns 0.5 (neutral). The decay function (`RecentActivityScoreFromTimestamp`) is implemented — it returns 1.0 within 7 days, linearly decaying to 0.0 at 90 days — but isn't wired yet because `Node` doesn't carry timestamps. See [DOC: docs/internal/PROBLEMS.md#graph-recent-activity-scoring]. |

```
score = (factor₁ × weight₁ + factor₂ × weight₂ + ...) / Σ weights
```

### Configurable Weights (Per Entity Type)

Weights are configurable per `team`, `agent`, and `skill` in:

`store/config/graph-health.json`

The Graph View `Settings -> Health` tab provides:
- **Live unsaved preview**: draft slider changes immediately update rendered node health without writing files.
- **Save + Recompute**: persists to `store/config/graph-health.json` and refreshes graph health from backend.

### CLI Health Policy

CLI nodes intentionally use a different policy from Team/Agent/Skill health factors:

- `cli:vrooli` → **neutral / unscored** (no health row)
- Scenario CLIs (for example `cli:prompt-manager`) → score from `scenario-completeness-scoring score <scenario> --json`
- Non-Vrooli tools/scripts (for example `cli:grep`, `cli:deploy.sh`) → score `0.0` (portability penalty)

CLI policy levers are also stored in `store/config/graph-health.json`:
- `neutralCommands`
- `externalToolScore`
- `scenarioFallbackScore`

## Analytical Queries

[CODE: api/graph/queries.go]

| Query | Returns | What It Finds |
|-------|---------|---------------|
| `OrphanedSkills` | `[]Node` | Skills with zero incoming edges (never referenced) |
| `SkilllessAgents` | `[]Node` | Agents with no outgoing skill-reference edges |
| `EmptyTeams` | `[]Node` | Teams with no `membership` edges |
| `UnaffiliatedAgents` | `[]Node` | Agents not targeted by any `membership` edge |
| `CLIlessSkills` | `[]Node` | Skills with no Vrooli CLI `code-usage` edges (external-only skills still appear) |
| `ExternalToolSkills` | `[]Node` | Skills with external tool or script `code-usage` edges (need wrapping in Vrooli CLIs) |
| `Popular(limit)` | `[]Node` | Top N nodes by incoming edge count |
| `DetectCircularRefs` | `[][]string` | DFS-based cycle detection across skill-to-skill edges |

## Index Persistence

[CODE: api/graph/index.go]

The graph is cached at `store/indexes/graph.index.json`. The `GraphIndexStore`:

- **Lazy generates** — loads from file on first read, regenerates if missing or corrupt
- **Thread-safe** — mutex-protected reads and writes
- **Auto-invalidates** — skill and agent CRUD handlers call `Invalidate()` to delete the cached file; the next read triggers a fresh rebuild

## UI Visualization

The frontend renders the graph using React Flow with Dagre hierarchical layout.

**Node shapes by type:**

| Type | Shape | Color |
|------|-------|-------|
| Team | Rounded rectangle | Blue |
| Agent | Circle | Emerald |
| Skill | Diamond (rotated square) | Violet |
| CLI | Hexagon | Orange |

**Health indicators:**

| Score Range | Visual |
|-------------|--------|
| < 0.3 | Red fill + border (critical) |
| 0.3 – 0.6 | Yellow fill + border (warning) |
| > 0.6 | Green fill + border (healthy) |

**Interactivity:**
- Click a node to open a detail popover with health breakdown, connection counts, and a "Go to editor" button
- The popover tracks the node during pan/zoom and closes on background click or re-clicking the same node
- Use the toolbar to filter by node type, adjust health threshold, toggle low-signal edges, collapse CLI nodes, and switch layout mode (`hierarchical`, `compact`, `grouped`)
- Run queries from the query panel to highlight matching nodes

## Related Documentation

- [API Reference — Graph Endpoints](../reference/api-endpoints.md#graph)
- [CLI Reference — Graph Commands](../reference/cli-commands.md#graph)
- [Swarm Model](SWARM-MODEL.md) — The three-domain architecture that the graph visualizes
- [Testing Seams](../internal/SEAMS.md#graph-seams) — Graph testing boundaries
