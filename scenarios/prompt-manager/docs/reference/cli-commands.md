# CLI Reference

Complete documentation for the prompt-manager CLI (`prompt-manager`).

## Installation

```bash
cd scenarios/prompt-manager/cli
go build -o prompt-manager .
```

## Global Options

```
--api-base <url>   Override API base URL (default: auto-detected from lifecycle)
--auto-start       Auto-start the scenario if not running
--no-color         Disable ANSI color output (or set NO_COLOR env var)
--color            Force-enable ANSI color output
```

## Commands Overview

| Command | Description |
|---------|-------------|
| `prompt-manager skill` | Manage skills (CRUD, versions, ratings) |
| `prompt-manager agent` | Manage agents (CRUD, appearance, files) |
| `prompt-manager tag` | Manage tags |
| `prompt-manager test` | Test skills with Ollama |
| `prompt-manager search` | Search skills |
| `prompt-manager graph` | Relationship graph analysis |
| `prompt-manager metadata` | Fetch URL metadata |
| `prompt-manager status` | Check API health |
| `prompt-manager configure` | View/update CLI settings |

---

## Skills

[CODE: cli/skills/skills.go]

### prompt-manager skill list

List all skills with optional filtering.

```bash
prompt-manager skill list [--folder=core|local|drafts] [--tag=TAG] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--folder` | Filter by folder |
| `--tag` | Filter by tag |
| `--json` | Output as JSON |

**Examples:**
```bash
prompt-manager skill list
prompt-manager skill list --folder=core
prompt-manager skill list --tag=debugging --json
```

### prompt-manager skill show

Show details of a specific skill.

```bash
prompt-manager skill show <id> [--json]
```

**Examples:**
```bash
prompt-manager skill show debugging
prompt-manager skill show my-skill --json
```

### prompt-manager skill add

Create a new skill interactively.

```bash
prompt-manager skill add <name> [--folder=local|drafts] [--description=...] [--tags=...] [--draft]
```

**Options:**
| Flag | Default | Description |
|------|---------|-------------|
| `--folder` | `local` | Target folder |
| `--description` | | Skill description |
| `--tags` | | Comma-separated tags |
| `--draft` | `false` | Mark as draft |

Content is read from stdin. End input with Ctrl+D.

**Example:**
```bash
echo "## My Skill Content" | prompt-manager skill add "My Skill" --tags=debugging,testing
```

### prompt-manager skill update

Update an existing skill.

```bash
prompt-manager skill update <id> [--name=...] [--description=...] [--content=...] [--tags=...] [--draft|--undraft] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--name` | New name |
| `--description` | New description |
| `--content` | New content |
| `--tags` | Comma-separated tags (replaces existing) |
| `--draft` | Mark as draft |
| `--undraft` | Remove draft status |
| `--json` | Output as JSON |

**Example:**
```bash
prompt-manager skill update my-skill --name="Updated Name" --tags=new-tag
```

### prompt-manager skill delete

Delete a skill (with confirmation).

```bash
prompt-manager skill delete <id> [--force]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

### prompt-manager skill use

Record usage and copy skill content to clipboard.

```bash
prompt-manager skill use <id>
```

**Example:**
```bash
prompt-manager skill use debugging
# Output: Usage recorded! Content copied to clipboard.
```

### prompt-manager skill sync

Sync skills with hash-based change detection.

```bash
prompt-manager skill sync [--tag=TAG] [--json]
```

Returns all skills with a hash for cache invalidation.

### prompt-manager skill read

Read skills by identifier, with optional combined output formatting.

```bash
prompt-manager skill read <identifier> [<identifier>...] [--resolve=auto|id|file|name] [--output=skills|combined|both|auto] [--format=xml|markdown|json] [--sep=STRING] [--strict] [--copy] [--json]
```

**Options:**
| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `auto` | Output mode (auto = combined for multiple, skills for single) |
| `--format` | `xml` | Combined output format |
| `--resolve` | `auto` | Resolution mode |
| `--sep` | `\n\n---\n\n` | Separator between skills (skills output) |
| `--strict` | | Fail if any identifier is missing or ambiguous |
| `--copy` | | Copy combined output to clipboard |
| `--json` | | Output full response as JSON |

**Example:**
```bash
prompt-manager skill read react-coherence domain-compression --output=combined --format=markdown --copy
```

### prompt-manager skill rate

Rate skill effectiveness (1-5).

```bash
prompt-manager skill rate <id> <rating> [--notes=...]
```

**Example:**
```bash
prompt-manager skill rate debugging 4 --notes="Very helpful for complex bugs"
```

### prompt-manager skill versions

Show version history for a skill.

```bash
prompt-manager skill versions <id> [--json]
```

**Example:**
```bash
prompt-manager skill versions my-skill
# Output:
# Version History for my-skill (current: v3):
#   v1 - 2024-01-15 10:00 - My Skill
#   v2 - 2024-01-18 12:00 - My Skill v2
#   v3 - 2024-01-20 14:30 - My Skill v3 (current)
```

### prompt-manager skill revert

Revert a skill to a specific version.

```bash
prompt-manager skill revert <id> <version> [--force]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

**Example:**
```bash
prompt-manager skill revert my-skill 2
# Output: Reverted to version 2 (new version: 4)
```

---

## Tags

[CODE: cli/tags/tags.go]

### prompt-manager tag list

List all tags.

```bash
prompt-manager tag list [--json]
```

### prompt-manager tag create

Create a new tag.

```bash
prompt-manager tag create <name> [--color=#RRGGBB] [--description=...] [--json]
```

**Example:**
```bash
prompt-manager tag create performance --color="#FF5733" --description="Performance-related skills"
```

---

## Agents

[CODE: cli/agents/agents.go]

Agents represent AI entities visualized in the 3D world. They are organized into teams and reference skills in their markdown files.

### prompt-manager agent list

List all agents.

```bash
prompt-manager agent list [--json]
```

### prompt-manager agent show

Show agent details including appearance and metadata.

```bash
prompt-manager agent show <id> [--json]
```

### prompt-manager agent soul

Get or set SOUL.md content for an agent.

```bash
prompt-manager agent soul <id> [--set='content'] [--file=path] [--json]
```

### prompt-manager agent create

Create a new agent.

```bash
prompt-manager agent create <name> [--body-color=#RRGGBB] [--head-color=#RRGGBB] [--accent-color=#RRGGBB] [--json]
```

**Options:**
| Flag | Default | Description |
|------|---------|-------------|
| `--body-color` | `#3B82F6` | Body color (hex) |
| `--head-color` | `#F59E0B` | Head color (hex) |
| `--accent-color` | `#10B981` | Accent color (hex) |

**Example:**
```bash
prompt-manager agent create "Alice" --body-color="#3B82F6"
```

### prompt-manager agent update

Update an agent.

```bash
prompt-manager agent update <id> [--name=...] [--body-color=...] [--head-color=...] [--accent-color=...] [--json]
```

### prompt-manager agent delete

Delete an agent (with confirmation).

```bash
prompt-manager agent delete <id> [--force]
```

## Teams

[CODE: cli/teams/teams.go]

Teams coordinate multiple agents with shared context. Teams define missions, roles, and organizational structure for agent swarms.

### prompt-manager team list

List all teams.

```bash
prompt-manager team list [--json]
```

### prompt-manager team show

Show team details including roles, members, and org chart.

```bash
prompt-manager team show <id> [--json]
```

**Example:**
```bash
prompt-manager team show engineering
# Output:
# Team: Engineering Team
# Mission: Build and maintain core platform
# Roles: lead, developer, reviewer
# Members: 3 agents
# Org Chart: alice -> bob
```

### prompt-manager team create

Create a new team.

```bash
prompt-manager team create <name> [--mission=...] [--spawn-mode=multi-process|single-process] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--mission` | Team mission statement |
| `--spawn-mode` | How the team is spawned: `multi-process` (default) or `single-process` |
| `--json` | Output as JSON |

**Example:**
```bash
prompt-manager team create "Engineering" --mission="Build and maintain core platform"
prompt-manager team create "Agent Swarm" --spawn-mode=single-process
```

### prompt-manager team add-member

Add an agent to a team with optional roles.

```bash
prompt-manager team add-member <team-id> <agent-id> [--roles=role1,role2]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--roles` | Comma-separated role IDs to assign |

**Example:**
```bash
prompt-manager team add-member engineering alice --roles=developer,reviewer
```

### prompt-manager team update-member

Update an agent's roles within a team.

```bash
prompt-manager team update-member <team-id> <agent-id> [--roles=role1,role2] [--status=active|inactive]
```

### prompt-manager team remove-member

Remove an agent from a team.

```bash
prompt-manager team remove-member <team-id> <agent-id> [--force]
```

### prompt-manager team roles

Manage team roles.

```bash
prompt-manager team roles <team-id> [--json]
```

### prompt-manager team org-list

List org chart relationships.

```bash
prompt-manager team org-list <team-id> [--json]
```

### prompt-manager team org-set

Set a reporting relationship (single-manager model).

```bash
prompt-manager team org-set <team-id> <report-id> <manager-id> [--json]
```

### prompt-manager team org-remove

Remove a reporting relationship.

```bash
prompt-manager team org-remove <team-id> <report-id>
```

### prompt-manager team message-list

List inbox messages for a team member.

```bash
prompt-manager team message-list <team-id> <agent-id> [--json]
```

### prompt-manager team message-send

Send a message to a team member.

```bash
prompt-manager team message-send <team-id> <agent-id> --from=<agent-id> --content="..." [--file=path] [--json]
```

### prompt-manager team message-delete

Delete a single inbox message.

```bash
prompt-manager team message-delete <team-id> <agent-id> <message-id>
```

### prompt-manager team message-clear

Clear all inbox messages for a team member.

```bash
prompt-manager team message-clear <team-id> <agent-id>
```

### prompt-manager team import-cc

Import a Claude Code team into prompt-manager.

```bash
prompt-manager team import-cc <team-name> [--json]
```

Reads the CC team config from `~/.claude/teams/{team-name}/config.json` and creates the corresponding PM team, agents, member relations, and org chart.

**Example:**
```bash
prompt-manager team import-cc my-cc-team
prompt-manager team import-cc my-cc-team --json
```

### prompt-manager team export-cc

Export a prompt-manager team as a Claude Code team config.

```bash
prompt-manager team export-cc <team-id> [--json]
```

**Example:**
```bash
prompt-manager team export-cc engineering
prompt-manager team export-cc engineering --json
```

### prompt-manager team trigger

Trigger heartbeats for an entire team.

```bash
prompt-manager team trigger <team-id> [--json]
```

Behavior depends on the team's `spawnMode`:
- **`single-process`**: Triggers only the team lead's heartbeat.
- **`multi-process`** (default): Triggers all member heartbeats.

**Example:**
```bash
prompt-manager team trigger engineering
```

---

## Graph

[CODE: cli/graph/graph.go]

Inspect the relationship graph between teams, agents, skills, and CLI tools. See [Graph Concepts](../concepts/GRAPH.md) for background.

### prompt-manager graph show

Print a summary of node and edge counts by type.

```bash
prompt-manager graph show
# Output:
# Graph Summary (generated 2026-02-12T10:30:45Z)
#
# Nodes:
#   Teams:  3
#   Agents: 8
#   Skills: 25
#   CLIs:   4
#   Total:  40
#
# Edges:
#   membership: 8
#   cli-read: 15
#   ...
#   Total: 42
#
# Health: 0.65 avg across 40 scored nodes
```

### prompt-manager graph dump

Print the full graph data.

```bash
prompt-manager graph dump [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--json` | Output as JSON instead of human-readable |

### prompt-manager graph node

Show details for a single node including adjacent edges and health.

```bash
prompt-manager graph node <id> [--json]
```

**Example:**
```bash
prompt-manager graph node debugging
# Output:
# ID:     debugging
# Type:   skill
# Label:  Debugging
# Health: 0.78
#   Factors:
#     outgoing-edges: 0.40
#     incoming-edges: 0.80
#
# Inbound edges (3):
#   alice -[cli-read]-> (this)
#   ...
#
# Outbound edges (1):
#   (this) -[code-usage]-> cli:vrooli
```

### prompt-manager graph regenerate

Force a full graph rebuild.

```bash
prompt-manager graph regenerate
```

**Aliases:** `regen`

### prompt-manager graph orphaned-skills

List skills not referenced by any agent or other skill.

```bash
prompt-manager graph orphaned-skills [--limit N] [--json]
```

**Aliases:** `orphans`

### prompt-manager graph skillless-agents

List agents that don't reference any skills.

```bash
prompt-manager graph skillless-agents [--limit N] [--json]
```

**Aliases:** `skillless`

### prompt-manager graph empty-teams

List teams with no members.

```bash
prompt-manager graph empty-teams [--json]
```

### prompt-manager graph unaffiliated-agents

List agents not in any team.

```bash
prompt-manager graph unaffiliated-agents [--json]
```

**Aliases:** `unaffiliated`

### prompt-manager graph cliless-skills

List skills that don't reference any CLI tools.

```bash
prompt-manager graph cliless-skills [--limit N] [--json]
```

**Aliases:** `cliless`

**Note:** Computed client-side from the full graph (no dedicated API endpoint).

### prompt-manager graph popular

List the most referenced nodes.

```bash
prompt-manager graph popular [--limit 10] [--type team|agent|skill|cli] [--json]
```

**Options:**
| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | 10 | Number of results |
| `--type` | | Filter by node type (client-side) |

### prompt-manager graph circular-refs

Detect circular dependencies between skills.

```bash
prompt-manager graph circular-refs [--json]
```

**Aliases:** `cycles`

### prompt-manager graph health

Show health scores for all nodes or a specific node.

```bash
prompt-manager graph health [--type team|agent|skill|cli] [--json]
prompt-manager graph health <node-id> [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--type` | Filter by node type |

**Example:**
```bash
prompt-manager graph health --type=skill
prompt-manager graph health debugging
```

---

## Testing

[CODE: cli/testing/testing.go]

Test skills with Ollama LLM.

### prompt-manager test run

Execute a test against a skill.

```bash
prompt-manager test run <skill-id> [--model=llama3.2] [--vars=key=value,key2=value2] [--max-tokens=1000] [--temperature=0.7] [--json]
```

**Options:**
| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `llama3.2` | Ollama model to use |
| `--vars` | | Variables to substitute (key=value format) |
| `--max-tokens` | `1000` | Maximum response tokens |
| `--temperature` | `0.7` | Generation temperature |
| `--json` | | Output as JSON |

**Example:**
```bash
prompt-manager test run debugging --model=llama3.2 --vars="TARGET=src/auth/login.ts"
# Output:
# Testing skill debugging with llama3.2...
# Test completed in 2500ms (450 tokens)
# Response: Based on the debugging skill...
```

### prompt-manager test history

View test history for a skill.

```bash
prompt-manager test history <skill-id> [--json]
```

---

## Search

[CODE: cli/search/search.go]

### prompt-manager search

Search skills by content, name, or tags.

```bash
prompt-manager search <query> [--tag=...] [--folder=...] [--content] [--text] [--case-sensitive] [--whole-word] [--regex] [--limit=N] [--output=results|combined|both] [--format=xml|markdown|json] [--render-limit=N] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--tag` | Filter by tag |
| `--folder` | Filter by folder |
| `--limit` | Maximum number of results |
| `--content` | Search within skill contents (line-level matches) |
| `--text` | Force text-only search (skip AI) |
| `--case-sensitive` | Case-sensitive content search |
| `--whole-word` | Whole word matching for content search |
| `--regex` | Treat query as regex for content search |
| `--output` | Output mode |
| `--format` | Combined output format |
| `--render-limit` | Limit number of skills rendered in combined output |
| `--json` | Output as JSON |

**Example:**
```bash
prompt-manager search "debugging" --folder=core
# Output:
# Search Results (3 found):
#   Debugging - core (0.95) [debugging] [debugging]
#     → ...systematic debugging approach for...
```

**Combined output example:**
```bash
prompt-manager search "react coherence" --output=combined --format=markdown
```

---

## Metadata

[CODE: cli/metadata/metadata.go]

### prompt-manager metadata fetch

Fetch Open Graph metadata from a URL.

```bash
prompt-manager metadata fetch <url> [--json]
```

**Example:**
```bash
prompt-manager metadata fetch https://example.com/article
# Output:
# URL: https://example.com/article
# Title: Article Title
# Description: Article description...
# Site: Example Site
```

---

## Status & Configuration

### prompt-manager status

Check API health and connectivity.

```bash
prompt-manager status
```

### prompt-manager configure

View or update CLI settings.

```bash
prompt-manager configure [api_base=URL] [token=TOKEN]
```

---

## Output Formats

Most commands support `--json` for machine-readable output:

```bash
prompt-manager skill list --json | jq '.[] | .name'
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | API connection failed |
| 3 | Resource not found |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PM_API_BASE` | Override API base URL |
| `NO_COLOR` | Disable colored output |
