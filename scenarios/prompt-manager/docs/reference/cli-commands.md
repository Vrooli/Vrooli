# CLI Reference

Complete documentation for the prompt-manager CLI (`pm`).

## Installation

```bash
cd scenarios/prompt-manager/cli
go build -o pm .
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
| `pm skill` | Manage skills (CRUD, versions, ratings) |
| `pm tag` | Manage tags |
| `pm member` | Manage members |
| `pm test` | Test skills with Ollama |
| `pm search` | Search skills |
| `pm metadata` | Fetch URL metadata |
| `pm status` | Check API health |
| `pm configure` | View/update CLI settings |

---

## Skills

[CODE: cli/skills/skills.go]

### pm skill list

List all skills with optional filtering.

```bash
pm skill list [--folder=core|local|drafts] [--tag=TAG] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--folder` | Filter by folder |
| `--tag` | Filter by tag |
| `--json` | Output as JSON |

**Examples:**
```bash
pm skill list
pm skill list --folder=core
pm skill list --tag=debugging --json
```

### pm skill show

Show details of a specific skill.

```bash
pm skill show <id> [--json]
```

**Examples:**
```bash
pm skill show debugging
pm skill show my-skill --json
```

### pm skill add

Create a new skill interactively.

```bash
pm skill add <name> [--folder=local|drafts] [--description=...] [--tags=...] [--draft]
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
echo "## My Skill Content" | pm skill add "My Skill" --tags=debugging,testing
```

### pm skill update

Update an existing skill.

```bash
pm skill update <id> [--name=...] [--description=...] [--content=...] [--tags=...] [--draft|--undraft] [--json]
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
pm skill update my-skill --name="Updated Name" --tags=new-tag
```

### pm skill delete

Delete a skill (with confirmation).

```bash
pm skill delete <id> [--force]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

### pm skill use

Record usage and copy skill content to clipboard.

```bash
pm skill use <id>
```

**Example:**
```bash
pm skill use debugging
# Output: Usage recorded! Content copied to clipboard.
```

### pm skill sync

Sync skills with hash-based change detection.

```bash
pm skill sync [--tag=TAG] [--json]
```

Returns all skills with a hash for cache invalidation.

### pm skill combine

Combine multiple skills into a single output.

```bash
pm skill combine <id> [<id>...] [--format=xml|markdown|json] [--json]
```

**Options:**
| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `xml` | Output format |
| `--json` | | Output full response as JSON |

**Example:**
```bash
pm skill combine debugging testing refactor --format=markdown
# Output: Combined 3 skills (~2500 tokens) copied to clipboard
```

### pm skill rate

Rate skill effectiveness (1-5).

```bash
pm skill rate <id> <rating> [--notes=...]
```

**Example:**
```bash
pm skill rate debugging 4 --notes="Very helpful for complex bugs"
```

### pm skill versions

Show version history for a skill.

```bash
pm skill versions <id> [--json]
```

**Example:**
```bash
pm skill versions my-skill
# Output:
# Version History for my-skill (current: v3):
#   v1 - 2024-01-15 10:00 - My Skill
#   v2 - 2024-01-18 12:00 - My Skill v2
#   v3 - 2024-01-20 14:30 - My Skill v3 (current)
```

### pm skill revert

Revert a skill to a specific version.

```bash
pm skill revert <id> <version> [--force]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

**Example:**
```bash
pm skill revert my-skill 2
# Output: Reverted to version 2 (new version: 4)
```

---

## Tags

[CODE: cli/tags/tags.go]

### pm tag list

List all tags.

```bash
pm tag list [--json]
```

### pm tag create

Create a new tag.

```bash
pm tag create <name> [--color=#RRGGBB] [--description=...] [--json]
```

**Example:**
```bash
pm tag create performance --color="#FF5733" --description="Performance-related skills"
```

---

## Members

[CODE: cli/members/members.go]

Members represent team entities for 3D world visualization.

### pm member list

List all members.

```bash
pm member list [--json]
```

### pm member show

Show member details.

```bash
pm member show <id> [--json]
```

### pm member create

Create a new member.

```bash
pm member create <name> [--body-color=#RRGGBB] [--head-color=#RRGGBB] [--accent-color=#RRGGBB] [--skills=id1,id2] [--json]
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
pm member create "Alice" --body-color="#3B82F6" --skills=debugging,testing
```

### pm member update

Update a member.

```bash
pm member update <id> [--name=...] [--body-color=...] [--head-color=...] [--accent-color=...] [--skills=...] [--json]
```

### pm member delete

Delete a member (with confirmation).

```bash
pm member delete <id> [--force]
```

---

## Testing

[CODE: cli/testing/testing.go]

Test skills with Ollama LLM.

### pm test run

Execute a test against a skill.

```bash
pm test run <skill-id> [--model=llama3.2] [--vars=key=value,key2=value2] [--max-tokens=1000] [--temperature=0.7] [--json]
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
pm test run debugging --model=llama3.2 --vars="TARGET=src/auth/login.ts"
# Output:
# Testing skill debugging with llama3.2...
# Test completed in 2500ms (450 tokens)
# Response: Based on the debugging skill...
```

### pm test history

View test history for a skill.

```bash
pm test history <skill-id> [--json]
```

---

## Search

[CODE: cli/search/search.go]

### pm search

Search skills by content, name, or tags.

```bash
pm search <query> [--tag=...] [--folder=...] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--tag` | Filter by tag |
| `--folder` | Filter by folder |
| `--json` | Output as JSON |

**Example:**
```bash
pm search "debugging" --folder=core
# Output:
# Search Results (3 found):
#   Debugging - core (0.95) [debugging] [debugging]
#     → ...systematic debugging approach for...
```

---

## Metadata

[CODE: cli/metadata/metadata.go]

### pm metadata fetch

Fetch Open Graph metadata from a URL.

```bash
pm metadata fetch <url> [--json]
```

**Example:**
```bash
pm metadata fetch https://example.com/article
# Output:
# URL: https://example.com/article
# Title: Article Title
# Description: Article description...
# Site: Example Site
```

---

## Status & Configuration

### pm status

Check API health and connectivity.

```bash
pm status
```

### pm configure

View or update CLI settings.

```bash
pm configure [api_base=URL] [token=TOKEN]
```

---

## Output Formats

Most commands support `--json` for machine-readable output:

```bash
pm skill list --json | jq '.[] | .name'
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
