# Swarm Coordination Model

This document explains the three-domain architecture that enables coordinated agent swarms in prompt-manager.

## Overview

Prompt-manager evolved from a simple skill storage system into a comprehensive **Skills + Agents + Teams** platform. This architecture enables agent swarms - coordinated groups of AI agents that work autonomously on complex tasks by composing skills and collaborating through team structures.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        SWARM COORDINATION                                │
│                                                                          │
│   ┌─────────────┐    Relations    ┌─────────────┐    Relations          │
│   │   SKILLS    │◄───────────────►│   AGENTS    │◄──────────────►       │
│   │             │   agent-skill   │             │   team-member         │
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
- **Skill pins** - direct skill assignments with version pins
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
  "skillPins": [
    {"skillId": "debugging", "version": "latest"}
  ],
  "heartbeat": {
    "intervalSeconds": 30,
    "timeoutSeconds": 90,
    "maxMissedBeats": 3
  }
}
```

### Teams

Teams are organizational structures that coordinate multiple agents around a mission with role-based skill grants.

**Key Characteristics:**
- **Mission** statement defining the team's purpose
- **Roles** with descriptions (e.g., "lead", "developer", "reviewer")
- **Skill grants by role** via `defaults.skillGrantsByRole`
- **Org chart** defining manager/report relationships
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
  "defaults": {
    "skillGrantsByRole": {
      "developer": ["debugging", "testing", "code-review"],
      "lead": ["project-planning", "debugging", "testing", "code-review"],
      "reviewer": ["code-review"]
    }
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

The three domains connect through **relations** - normalized junction records that link agents to skills and teams.

### Flow: Agent Gets Assigned to Team

1. Agent `alice` is created with base capabilities
2. Agent-skill relations link `alice` to specific skills
3. Team-member relation adds `alice` to `engineering` team with `developer` role
4. When `alice` needs skills, effective-skills computation runs:
   - Collects `alice`'s skill pins
   - Collects enabled agent-skill relations
   - Adds skills granted by `developer` role in `engineering`

### Data Flow Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                     EFFECTIVE SKILLS COMPUTATION                       │
│                                                                        │
│  Agent Request                                                         │
│       │                                                                │
│       ▼                                                                │
│  ┌─────────────┐                                                       │
│  │ Agent Store │──────► skillPins (direct assignments)                │
│  └─────────────┘              │                                        │
│                               │                                        │
│  ┌─────────────────┐          │                                        │
│  │ Relation Store  │──────► agent-skill relations (enabled=true)      │
│  │ (agent-skill)   │          │                                        │
│  └─────────────────┘          │                                        │
│                               ▼                                        │
│                    ┌──────────────────────┐                           │
│                    │   Merge skill IDs    │                           │
│                    └──────────┬───────────┘                           │
│                               │                                        │
│  ┌─────────────────┐          │  (if teamId provided)                 │
│  │ Relation Store  │──────► team-member relation → roles              │
│  │ (team-member)   │          │                                        │
│  └─────────────────┘          │                                        │
│                               ▼                                        │
│  ┌─────────────┐    ┌──────────────────────┐                          │
│  │ Team Store  │───►│ skillGrantsByRole    │──► role-granted skills   │
│  └─────────────┘    └──────────────────────┘         │                │
│                                                       │                │
│                               ┌───────────────────────┘                │
│                               ▼                                        │
│                    ┌──────────────────────┐                           │
│                    │  EFFECTIVE SKILL SET │                           │
│                    └──────────────────────┘                           │
└──────────────────────────────────────────────────────────────────────┘
```

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
    - security: [security-audit, vulnerability-scan]
    - performance: [perf-analysis, optimization-tips]
    - style: [code-style, best-practices]

  Agents assigned to specialized roles, each bringing different skills
```

## Key Benefits

1. **Separation of Concerns**: Skills define what, agents define who, teams define how they coordinate
2. **Reusability**: Same skill can be used by many agents; same agent can be in multiple teams
3. **Role-Based Access**: Team roles grant skills without individual assignment
4. **Scalability**: Add new agents to team, they automatically get role-based skills
5. **Flexibility**: Override team grants with individual agent-skill relations
6. **Observability**: 3D world visualization shows swarm activity in real-time

## Related Documentation

- [RELATIONS.md](RELATIONS.md) - Agent-skill and team-member relation details
- [EFFECTIVE-SKILLS.md](EFFECTIVE-SKILLS.md) - Computation algorithm
- [PERSONA-SYSTEM.md](PERSONA-SYSTEM.md) - Agent SOUL.md configuration
- [CAPABILITY-MATCHING.md](CAPABILITY-MATCHING.md) - Skill-to-agent matching
- [3D-WORLD-ARCHITECTURE.md](3D-WORLD-ARCHITECTURE.md) - Visualization details
