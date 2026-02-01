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
| `prompt-manager agent` | Manage agents (CRUD, appearance, skills) |
| `prompt-manager tag` | Manage tags |
| `prompt-manager test` | Test skills with Ollama |
| `prompt-manager search` | Search skills |
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

Agents represent AI entities visualized in the 3D world. They can be assigned skills and organized into teams.

### prompt-manager agent list

List all agents.

```bash
prompt-manager agent list [--json]
```

### prompt-manager agent show

Show agent details including appearance, persona, and assigned skills.

```bash
prompt-manager agent show <id> [--json]
```

### prompt-manager agent create

Create a new agent.

```bash
prompt-manager agent create <name> [--body-color=#RRGGBB] [--head-color=#RRGGBB] [--accent-color=#RRGGBB] [--skills=id1,id2] [--json]
```

**Options:**
| Flag | Default | Description |
|------|---------|-------------|
| `--body-color` | `#3B82F6` | Body color (hex) |
| `--head-color` | `#F59E0B` | Head color (hex) |
| `--accent-color` | `#10B981` | Accent color (hex) |
| `--skills` | | Comma-separated skill IDs |

**Example:**
```bash
prompt-manager agent create "Alice" --body-color="#3B82F6" --skills=debugging,testing
```

### prompt-manager agent update

Update an agent.

```bash
prompt-manager agent update <id> [--name=...] [--body-color=...] [--head-color=...] [--accent-color=...] [--skills=...] [--json]
```

### prompt-manager agent delete

Delete an agent (with confirmation).

```bash
prompt-manager agent delete <id> [--force]
```

### prompt-manager agent effective-skills

Compute the effective skill set for an agent. This includes:
1. Skills pinned directly to the agent (skillPins)
2. Agent-skill relations (enabled=true)
3. Team role-based grants (if --team specified)

```bash
prompt-manager agent effective-skills <id> [--team=<team-id>] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--team` | Team ID to include role-based skill grants |
| `--json` | Output as JSON |

**Example:**
```bash
prompt-manager agent effective-skills alice --team=engineering
# Output:
# Effective Skills for alice (team: engineering):
#   debugging (pinned)
#   testing (relation)
#   code-review (role: reviewer)
```

---

## Teams

[CODE: cli/teams/teams.go]

Teams coordinate multiple agents with role-based skill grants. Teams define missions, roles, and organizational structure for agent swarms.

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
```

### prompt-manager team create

Create a new team.

```bash
prompt-manager team create <name> [--mission=...] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--mission` | Team mission statement |
| `--json` | Output as JSON |

**Example:**
```bash
prompt-manager team create "Engineering" --mission="Build and maintain core platform"
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

Shows available roles for a team.

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
prompt-manager search <query> [--tag=...] [--folder=...] [--limit=N] [--text] [--output=results|combined|both] [--format=xml|markdown|json] [--render-limit=N] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--tag` | Filter by tag |
| `--folder` | Filter by folder |
| `--limit` | Maximum number of results |
| `--text` | Force text-only search (skip AI) |
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
