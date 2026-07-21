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
| `prompt-manager discover` | Discover relevant skills and opt-in Action matches |
| `prompt-manager discovery-gaps` | Clustered unmet-capability queries (discovery misses) |
| `prompt-manager action` | Manage typed executable Action contracts |
| `prompt-manager graph` | Relationship graph analysis |
| `prompt-manager metadata` | Fetch URL metadata |
| `prompt-manager status` | Check API health |
| `prompt-manager configure` | View/update CLI settings |

Action and skill command validation delegates Vrooli-owned command truth to CLI
Health. Prompt Manager owns action policy, placeholders, permissions, and
run-eligibility reporting; it does not maintain a separate current-command
catalog. When CLI Health can prove a command path exists but cannot prove
arguments or action governance, Prompt Manager reports the action as
unvalidated instead of treating partial coverage as fully safe.
Skill graph health also reports CLI Health findings for Vrooli-owned commands
detected in skill content: invalid current commands are critical diagnostics,
while command-exists/arguments-unknown results remain visible as partial
coverage warnings.

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
prompt-manager skill read react-coherence domain-clarity --output=combined --format=markdown --copy
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

### prompt-manager skill variants

List variants for a skill.

```bash
prompt-manager skill variants <id> [--json]
```

### prompt-manager skill add-variant

Create a new variant for a skill.

```bash
prompt-manager skill add-variant <id> --name NAME --file FILE_PATH [--description TEXT] [--json]
```

### prompt-manager skill rm-variant

Delete a variant.

```bash
prompt-manager skill rm-variant <id> <variant-id> [--force]
```

---

## Experiments

[CODE: cli/experiments/experiments.go]

### prompt-manager experiment list

List all experiments, optionally filtered by skill.

```bash
prompt-manager experiment list [--skill SKILL_ID] [--json]
```

### prompt-manager experiment show

Show experiment details including outcome counts.

```bash
prompt-manager experiment show <experiment-id> [--json]
```

### prompt-manager experiment create

Create a new experiment.

```bash
prompt-manager experiment create --skill SKILL_ID --name NAME [--hypothesis TEXT] --arm VARIANT:WEIGHT --arm VARIANT:WEIGHT [--json]
```

**Example:**
```bash
prompt-manager experiment create --skill swarm-manager-workshop --name "Concise test" --arm control:0.5 --arm concise-v1:0.5
```

### prompt-manager experiment start

Start a draft experiment (transition to running).

```bash
prompt-manager experiment start <experiment-id>
```

### prompt-manager experiment conclude

Conclude a running experiment with a recommended winner. This command never
changes `SKILL.md`; a separately authorized holdout-confirmed promotion is
required.

```bash
prompt-manager experiment conclude <experiment-id> <winner-variant-id> [--notes TEXT]
```

### prompt-manager experiment outcomes

List raw outcomes for an experiment.

```bash
prompt-manager experiment outcomes <experiment-id> [--json]
```

### prompt-manager experiment report

Show a per-arm aggregation report: serve counts, outcome counts, outcome status breakdown, success rate, and mean tokens used.

```bash
prompt-manager experiment report <experiment-id> [--json]
```

### prompt-manager experiment delete

Delete an experiment and its outcomes.

```bash
prompt-manager experiment delete <experiment-id> [--force]
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
prompt-manager team create <name> [--mission=...] [--runtime-mode=multi-process|single-process] [--coordination-pattern=independent|peer|leader-led] [--decision-mode=yolo|approval] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--mission` | Team mission statement |
| `--runtime-mode` | Runtime mode: `multi-process` (default) or `single-process` |
| `--coordination-pattern` | Coordination pattern: `independent` (default), `peer`, or `leader-led` |
| `--lead-agent-id` | Required when using `leader-led` coordination |
| `--reporting-mode` | Reporting mode override: `none`, `org-chart`, or `leader` |
| `--messaging-mode` | Messaging mode override: `disabled`, `async-inbox`, or `in-session` |
| `--queue-policy` | Execution queue policy: `bounded-parallel` (default) or `serialized` |
| `--max-concurrent-runs` | Concurrency limit for `bounded-parallel` execution |
| `--show-org-context`, `--inject-inbox`, `--allow-peer-triggers`, `--show-task-board-guidance`, `--show-decision-log-guidance`, `--show-knowledge-log-guidance`, `--require-handoff` | Override individual coordination capabilities |
| `--decision-mode` | Decision policy: `yolo` (default behavior) or `approval` |
| `--json` | Output as JSON |

**Example:**
```bash
prompt-manager team create "Engineering" --mission="Build and maintain core platform"
prompt-manager team create "Scenario QA" --coordination-pattern=independent --queue-policy=bounded-parallel --max-concurrent-runs=2
prompt-manager team create "Director Swarm" --runtime-mode=single-process --coordination-pattern=leader-led --lead-agent-id=director --decision-mode=approval
```

When you choose `--runtime-mode=single-process`, the CLI resolves the team onto the leader-led serialized preset before sending the request.

Teams are stored with a required `operatingContract`. The default create flow seeds an empty contract for teams with no members; production teams should replace it with member policies before enabling heartbeats.

### prompt-manager team update

Update an existing team.

```bash
prompt-manager team update <id> [--name=...] [--mission=...] [--enabled=true|false] [--runtime-mode=multi-process|single-process] [--coordination-pattern=independent|peer|leader-led] [--decision-mode=yolo|approval] [--json]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--name` | New display name |
| `--mission` | New mission statement |
| `--enabled` | Enable or disable the team |
| `--runtime-mode` | Change runtime mode |
| `--coordination-pattern` | Change coordination pattern |
| `--lead-agent-id` | Set or replace the explicit lead agent for leader-led teams |
| `--reporting-mode`, `--messaging-mode`, `--queue-policy`, `--max-concurrent-runs` | Update policy settings directly |
| Capability override flags | Update prompt and coordination capabilities individually |
| `--decision-mode` | Change decision policy |
| `--json` | Output as JSON |

### prompt-manager team operating-contract

Print the stored operating contract for a team.

```bash
prompt-manager team operating-contract <team-id>
```

The output is the `team.json.operatingContract` object. Heartbeat prompts render a member-specific resolved view from this contract.

### prompt-manager team validate-contract

Validate the team's operating contract through the API load path.

```bash
prompt-manager team validate-contract <team-id> [--json]
```

Invalid contracts fail with the same validation errors used by team loading and heartbeat prompt building.

Enabled leader-led teams require `coordination.leadAgentId` to reference an active team member. The API will reject updates that would enable an invalid lead configuration.

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

For enabled leader-led teams, the configured lead cannot be set inactive until the lead assignment changes or the team is disabled.

### prompt-manager team remove-member

Remove an agent from a team.

```bash
prompt-manager team remove-member <team-id> <agent-id> [--force]
```

For enabled leader-led teams, the configured lead cannot be removed until the lead assignment changes or the team is disabled.

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

Behavior depends on the team's resolved policy:
- **`single-process` + `leader-led`**: Triggers only the configured lead heartbeat.
- **All other team policies**: Triggers all member heartbeats with configs.

Leader-led single-process triggers fail fast if the configured lead is not an active team member or does not have a heartbeat config.

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

Skill health includes command-reference diagnostics from CLI Health for
Vrooli-owned commands detected in skill instructions.

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

### prompt-manager discover

[CODE: cli/discover/discover.go]

Discover relevant skills using topic search plus AI search with a complexity budget. Use `--type action` or `--type all` to include Action matches; omitting `--type` preserves the legacy skill-only response shape.

```bash
prompt-manager discover "concept1" "concept2" [--complexity minor|moderate|major|architectural] [--limit=N] [--type skill|action|all] [--json]
```

**Action discovery examples:**

```bash
prompt-manager discover "take screenshot of scenario UI" --type all
prompt-manager discover "take screenshot of scenario UI" --type action
prompt-manager discover "debugging methodology" --type skill
```

`--type all` returns both skills and Actions with a type discriminator. Agents should prefer exact Action matches for deterministic execution and use skills when the work requires judgment.

### prompt-manager discovery-gaps

[CODE: cli/discover/discover.go]

Show clustered unmet-capability queries — the searches that `prompt-manager discover` answered with nothing useful (zero results or only sub-threshold matches). These misses are captured server-side automatically; this command reads a time window and clusters near-duplicate queries so the meta-optimization team can route each to a new action, capability-gap, or cli-backlog. Counts are window-relative.

```bash
prompt-manager discovery-gaps [--since 7d] [--type skill|action|all] [--limit=N] [--json]
```

---

## Actions

Actions are typed wrappers over exactly one Vrooli-controlled CLI command. The CLI exposes CRUD, validation, and governed run commands; execution governance remains owned by the API. See [DOC: docs/concepts/ACTIONS.md].

### prompt-manager action list

List Actions with optional status, pack, tag, or owner filters.

```bash
prompt-manager action list [--pack=core|local|drafts] [--status=active|draft|archived] [--owner=...] [--tag=...] [--json]
```

### prompt-manager action show

Show an Action contract, including input schema, output schema, command target, permissions, examples, and validation status.

```bash
prompt-manager action show <id> [--json]
```

### prompt-manager action create

Create an Action from a single Vrooli CLI command. **Previews by default** (writes nothing): the command resolves the owner, infers inputs from `{{placeholders}}`, infers permissions from the owner's `cli/manifest.json`, validates the contract, and surfaces any similar existing actions. Add `--apply` to register it (default status `active`). Alternatively, pass a fully authored `--file=action.json`. Creating an action is free — no decision required (only *retiring prose* in favor of an action is gated; see [DOC: docs/concepts/ACTIONS.md]).

```bash
# Preview (no write): infer + validate + show similar actions
prompt-manager action create --name "Show Scenario Status" --command 'vrooli scenario status {{scenario}}'

# Register it
prompt-manager action create --name "Show Scenario Status" --command 'vrooli scenario status {{scenario}}' --apply

# Refine an inferred input, set an explicit id, or target a pack
prompt-manager action create --name "Capture Page" --command 'browser-automation-studio capture {{url}} --out {{out}}' \
  --input out:path --id bas.capture --pack core --apply

# Or create from a fully authored contract (also previews unless --apply)
prompt-manager action create --file=path/to/action.json [--pack=core|local|drafts] --apply
```

`--command` and `--file` are mutually exclusive; exactly one is required. Placeholder names may use lower camel case or snake case, such as `{{scenario}}` or `{{phase_or_provider}}`, and each placeholder infers a required input. `--input name:type[:optional]` refines an inferred input (repeatable). A near-duplicate (same executable + subcommand, or high semantic similarity) is surfaced in the preview so you can `action update` an existing action instead of creating a near-duplicate. If `--pack` is omitted, the action lands in the active `local` pack so it is immediately discoverable.

### prompt-manager action update

Replace an existing Action contract from an `action.json` file. The Action ID in the file must match the path ID.

```bash
prompt-manager action update <id> --file=path/to/action.json [--json]
```

### prompt-manager action delete

Archive an Action by default. Use `--hard` only when the underlying file should be removed.

```bash
prompt-manager action delete <id> [--yes] [--hard]
```

### prompt-manager action validate

Validate that an Action contract is well-formed and points to an allowed Vrooli-controlled command.

```bash
prompt-manager action validate <id> [--json]
```

Validation should reject shell pipelines, command separators, raw external tools, missing input/output schemas, and undeclared permissions.

### prompt-manager action run

Run an active, runnable Action with typed input through the governed API runtime. The CLI remains a thin API client and does not duplicate validation, permission checks, timeout handling, concurrency throttling, stdout/stderr caps, or audit history.

```bash
prompt-manager action run <id> [--input='{"key":"value"}'|--input-file=payload.json] [--dry-run] [--json]
```

Safe seed dry-run:

```bash
prompt-manager action run scenario.status.show --input='{"scenario":"prompt-manager"}' --dry-run --json
```

Use `--dry-run` to validate inputs and return the rendered argv without starting the process. Non-JSON output prints status, exit code, duration, rendered argv, stdout/stderr snippets, parsed output, and any API error message. `failed`, `timed-out`, `rejected`, and `throttled` responses return a non-zero CLI error after printing the run envelope.

Branching and implementation logic belong in the owning CLI, not in the Action wrapper.

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
