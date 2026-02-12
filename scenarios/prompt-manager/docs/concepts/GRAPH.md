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

The `CLIDetector` scans agent, team, and skill content for CLI tool references (scenario CLIs, external tools, scripts, API calls) and produces `code-usage` edges.

## Health Scoring

[CODE: api/graph/scoring.go]

Each node receives a composite health score (0.0–1.0) computed as a weighted average of individual factors:

| Factor | Weight | What It Measures |
|--------|--------|------------------|
| `outgoing-edges` | 1.0 | Number of edges where node is the source. 5+ edges = 1.0. |
| `incoming-edges` | 1.0 | Number of edges where node is the target. 5+ references = 1.0. |
| `code-usage` | 0.5 | Binary: 1.0 if node has any `code-usage` edge, 0.0 otherwise. |
| `recent-activity` | 0.5 | Currently returns 0.5 (neutral). The decay function (`RecentActivityScoreFromTimestamp`) is implemented — it returns 1.0 within 7 days, linearly decaying to 0.0 at 90 days — but isn't wired yet because `Node` doesn't carry timestamps. See [DOC: docs/internal/PROBLEMS.md#graph-recent-activity-scoring]. |

```
score = (factor₁ × weight₁ + factor₂ × weight₂ + ...) / Σ weights
```

## Analytical Queries

[CODE: api/graph/queries.go]

| Query | Returns | What It Finds |
|-------|---------|---------------|
| `OrphanedSkills` | `[]Node` | Skills with zero incoming edges (never referenced) |
| `SkilllessAgents` | `[]Node` | Agents with no outgoing skill-reference edges |
| `EmptyTeams` | `[]Node` | Teams with no `membership` edges |
| `UnaffiliatedAgents` | `[]Node` | Agents not targeted by any `membership` edge |
| `CLIlessSkills` | `[]Node` | Skills with no `code-usage` edges |
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
| < 0.3 | Red ring (critical) |
| 0.3 – 0.6 | Yellow ring (warning) |
| > 0.6 | No ring (healthy) |

**Interactivity:**
- Click a node to navigate to its editor panel
- Hover to see health score breakdown in a tooltip
- Use the toolbar to filter by node type, adjust health threshold, or toggle layout direction
- Run queries from the query panel to highlight matching nodes

## Related Documentation

- [API Reference — Graph Endpoints](../reference/api-endpoints.md#graph)
- [CLI Reference — Graph Commands](../reference/cli-commands.md#graph)
- [Swarm Model](SWARM-MODEL.md) — The three-domain architecture that the graph visualizes
- [Testing Seams](../internal/SEAMS.md#graph-seams) — Graph testing boundaries
