# Swarm Coordination Model

This document explains the three-domain architecture that enables coordinated agent swarms in prompt-manager.

## Overview

Prompt-manager evolved from a simple skill storage system into a comprehensive **Skills + Agents + Teams** platform. This architecture enables agent swarms - coordinated groups of AI agents that work autonomously on complex tasks by composing skills and collaborating through team structures.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        SWARM COORDINATION                                │
│                                                                          │
│   ┌─────────────┐  Text References ┌─────────────┐    Relations         │
│   │   SKILLS    │◄────────────────►│   AGENTS    │◄──────────────►      │
│   │             │   (markdown)     │             │   team-member        │
│   │  behaviors  │                 │  identities │                       │
│   │  with packs │                 │  + souls   │        ┌─────────────┐│
│   └─────────────┘                 └─────────────┘        │    TEAMS    ││
│                                                          │             ││
│                                                          │ coordination││
│                                                          │ + roles     ││
│                                                          └─────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
```

## The Three Domains

### Skills

Skills are reusable AI behaviors that define what an agent can do. They contain prompts, instructions, and capability declarations.

**Key Characteristics:**
- Organized into **packs**: `core` (system skills), `local` (user-created), `drafts` (work-in-progress)
- Pack precedence via `_pack-order.json`
- **Capability declarations** in `requires.capabilities` - what an agent needs to use this skill
- **Version history** via `history.jsonl` for tracking changes
- **Modes** (agent, human, etc.) to indicate intended usage
- **Entry point** (`SKILL.md`) containing the actual skill content

**Storage:**
```
store/skills/packs/{pack}/{skill-id}/
├── skill.json      # Metadata, capabilities, modes
├── SKILL.md        # Skill content
└── history.jsonl   # Version history
```

**Example skill.json:**
```json
{
  "id": "debugging",
  "name": "Debugging Expert",
  "description": "Systematic approach to debugging code",
  "modes": ["agent"],
  "tags": ["debugging", "troubleshooting"],
  "status": "active",
  "requires": {
    "capabilities": ["code-analysis", "file-read"]
  }
}
```

### Agents

Agents are autonomous AI entities with identity, appearance, SOUL.md personality, and capabilities. They are the actors in the swarm.

**Key Characteristics:**
- **Appearance** (body, head, accent colors) for 3D world visualization
- **SOUL.md** defining personality and behavioral guidance
- **Capabilities** - what the agent provides and requires (with verbs)
- **Skill references** in SOUL.md and other agent files (markdown)
- **Heartbeat configuration** for health monitoring
- **Runtime workspace** reference for execution context

**Storage:**
```
store/agents/{agent-id}/
├── agent.json
└── SOUL.md
```

**Example agent.json:**
```json
{
  "id": "alice",
  "displayName": "Alice",
  "description": "Senior debugging specialist",
  "status": "active",
  "appearance": {
    "body": "#3B82F6",
    "head": "#F59E0B",
    "accent": "#10B981"
  },
  "capabilities": {
    "provides": [
      {"capabilityId": "code-analysis", "verbs": ["read", "analyze"]},
      {"capabilityId": "debugging", "verbs": ["diagnose", "fix"]}
    ],
    "requires": [
      {"capabilityId": "file-access", "verbs": ["read"]}
    ]
  },
  "heartbeat": {
    "intervalSeconds": 30,
    "timeoutSeconds": 90,
    "maxMissedBeats": 3
  }
}
```

### Teams

Teams are organizational structures that coordinate multiple agents around a mission with shared context and roles.

**Key Characteristics:**
- **Mission** statement defining the team's purpose
- **Roles** with descriptions (e.g., "lead", "developer", "reviewer")
- **Org chart** defining manager/report relationships
  - Each report can have a single manager (manager → report edges)
- **Shared documents** path for team-wide resources

**Storage:**
```
store/teams/{team-id}/
├── team.json      # Core team metadata
├── roles.json     # Role definitions (optional)
└── org-chart.json # Organizational hierarchy (optional)
```

**Example team.json:**
```json
{
  "id": "engineering",
  "displayName": "Engineering Team",
  "mission": "Build and maintain core platform features",
  "shared": {
    "path": "teams/engineering/shared",
    "mountHint": "readWrite"
  },
  "roles": [
    {"id": "lead", "name": "Team Lead", "description": "Coordinates team efforts"},
    {"id": "developer", "name": "Developer", "description": "Implements features"},
    {"id": "reviewer", "name": "Reviewer", "description": "Reviews code changes"}
  ],
  "orgChart": {
    "edges": [
      {"managerAgentId": "alice", "reportAgentId": "bob"},
      {"managerAgentId": "alice", "reportAgentId": "charlie"}
    ]
  }
}
```

## How They Work Together

The three domains connect through **relations** for team membership and **markdown references** for skill usage.

### Flow: Agent Gets Assigned to Team

1. Agent `alice` is created with base capabilities
2. Agent files (SOUL.md, RESPONSIBILITIES.md) reference relevant skills in markdown
3. Team-member relation adds `alice` to `engineering` team with `developer` role
4. When `alice` needs guidance, it reads skill references from its files and team shared docs

## Use Cases

### 1. Bug Fixing Swarm

```
Team: Bug Fixers
  Mission: "Triage, reproduce, fix, and verify bugs"
  Roles:
    - triager: [bug-triage, issue-analysis]
    - developer: [debugging, testing, code-fix]
    - verifier: [test-writing, verification]

  Agents:
    - triage-bot (triager) → receives bug-triage, issue-analysis
    - fix-bot (developer) → receives debugging, testing, code-fix
    - verify-bot (verifier) → receives test-writing, verification
```

### 2. Content Generation Swarm

```
Team: Content Creators
  Mission: "Research, write, edit, and publish content"
  Roles:
    - researcher: [research, source-analysis]
    - writer: [content-writing, formatting]
    - editor: [grammar-check, style-guide]

  Agents:
    - research-agent (researcher)
    - writer-agent (writer, editor) → multiple roles
    - quality-agent (editor)
```

### 3. Code Review Swarm

```
Team: Review Squad
  Mission: "Ensure code quality through multi-perspective review"
  Roles:
    - security
    - performance
    - style

  Agents assigned to specialized roles, with skill references documented in team markdown
```

## Swarm Manager Integration: The Staging Layer

Teams do not execute their plans directly. Instead, they deposit findings into the `swarm-manager` scenario as backlog items using the `swarm-manager-recommendations` skill. This creates a **staging and review layer** between agent analysis and scenario execution.

```
prompt-manager (teams analyze)          swarm-manager (staging/review)
┌──────────────────────────┐            ┌──────────────────────────────┐
│  Feature Team  → idea    │──┐         │                              │
│  QA Team       → fix     │──┼─ plans ▶│  Backlog (review all plans)  │
│                          │──┘         │         ↓                    │
│                          │            │  Idea Agent (refine plans)   │
└──────────────────────────┘            │         ↓                    │
                                        │  Generator / Improver        │
                                        │  (build/iterate scenarios)   │
                                        └──────────────────────────────┘
```

**Why staging matters:**
- Operators get a single place to review all agent-generated plans
- The Idea Agent's clarify/suggest/enhance pipeline refines plans before execution
- Execution governance (manual/scheduled/yolo) controls when approved work runs
- Plans are git-tracked, human-readable, and editable before committing to execution

**Team-to-backlog mapping** (defined in the `swarm-manager-recommendations` skill):

| Team | Backlog Kind | Purpose |
|------|-------------|---------|
| Feature Team | `idea` or `execute` | New capabilities and enhancements |
| QA Team | `fix` or `execute` | Quality issues and test improvements |

See [swarm-manager-recommendations SKILL.md](../../store/skills/packs/core/swarm-manager-recommendations/SKILL.md) for the full team-to-backlog contract.

## Coordination Skills

Teams inject spawn-mode-specific coordination guidance into agent prompts via dedicated skills. The team's `spawnMode` determines which skill is referenced:

| Spawn Mode | Skill | Purpose |
|-----------|-------|---------|
| `multi-process` | `team-coordination-multi-process` | CLI messaging commands, inbox checking, queue awareness |
| `single-process` | `team-coordination-single-process` | Context bootstrapping, spawn prompt formatting, work tracking |

### Multi-Process Coordination

Each agent runs as a separate process with its own heartbeat. Agents coordinate via:
- **Team messaging**: `prompt-manager team message-send/list/delete/clear`
- **Responsibilities**: `prompt-manager team responsibilities <team-id> <agent-id>`
- **Queue awareness**: Heartbeats are serialized per team; agents should use messages for coordination rather than repeated triggers

### Single-Process Coordination

The team lead runs as a single process and spawns teammates within its session. The lead:
- Tells teammates to run `prompt-manager team member-context <team-id> <agent-id>` to bootstrap context
- Uses Claude Code's `SendMessage` for follow-up conversations
- Maps the org chart to Claude Code team structure

The coordination skill is a **static behavioral layer** that complements the dynamic team-specific prompt generated by `FormatSpawnPrompt()`.

## Key Benefits

1. **Separation of Concerns**: Skills define what, agents define who, teams define how they coordinate
2. **Reusability**: Same skill can be used by many agents; same agent can be in multiple teams
3. **Text-First Skills**: Skills are referenced in markdown, keeping behavior editable and human-readable
4. **Scalability**: Add new agents to a team to share context and coordination
5. **Flexibility**: Update skill references without schema migrations or relations
6. **Observability**: 3D world visualization shows swarm activity in real-time

## Related Documentation

- [RELATIONS.md](RELATIONS.md) - Team-member relation details
- [PERSONA-SYSTEM.md](PERSONA-SYSTEM.md) - Agent SOUL.md configuration
- [CAPABILITY-MATCHING.md](CAPABILITY-MATCHING.md) - Skill-to-agent matching
- [3D-WORLD-ARCHITECTURE.md](3D-WORLD-ARCHITECTURE.md) - Visualization details
