# Swarm Coordination Model

This document explains the current Skills + Agents + Teams architecture that enables coordinated agent swarms in prompt-manager, plus the implemented Action layer for deterministic execution.

## Overview

Prompt-manager evolved from a simple skill storage system into a comprehensive **Skills + Agents + Teams** platform. This architecture enables agent swarms - coordinated groups of AI agents that work autonomously on complex tasks by composing skills and collaborating through team structures.

The Action layer adds a fourth concept for execution, not judgment:

```text
Truth lives in the Plan of Record.
Judgment lives in Skills.
Execution lives in Actions.
Implementation lives in CLIs.
Unbuilt work lives in the Backlog.
Raw learning starts in typed knowledge topics.
```

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        SWARM COORDINATION                                │
│                                                                          │
│   ┌─────────────┐  Text References ┌─────────────┐    Relations         │
│   │   SKILLS    │◄────────────────►│   AGENTS    │◄──────────────►      │
│   │             │   (markdown)     │             │   team-member        │
│   │  judgment   │                 │  identities │                       │
│   │  with packs │                 │  + souls   │        ┌─────────────┐│
│   └─────────────┘                 └─────────────┘        │    TEAMS    ││
│                                          │               │             ││
│                                          ▼               │             ││
│                                  ┌─────────────┐         │             ││
│                                  │  ACTIONS*   │         │             ││
│                                  │ execution   │         │             ││
│                                  │ over CLIs   │         │             ││
│                                  └─────────────┘         │             ││
│                                                          │             ││
│                                                          │ coordination││
│                                                          │ + roles     ││
│                                                          └─────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
```

`*` Actions are typed command contracts with API/CLI/UI validation, opt-in discovery, graph nodes, and governed execution through the Action runtime. See [Actions](ACTIONS.md).

## The Three Current Domains

### Skills

Skills are reusable AI guidance documents that define how an agent should reason, decide, or approach a class of work. They contain prompts, instructions, and capability declarations.

**Key Characteristics:**
- Organized into **packs**: `core` (system skills), `local` (user-created), `drafts` (work-in-progress)
- Pack precedence via `_pack-order.json`
- **Capability declarations** in `requires.capabilities` - what an agent needs to use this skill
- **Version history** via `history.jsonl` for tracking changes
- **Modes** (agent, human, etc.) to indicate intended usage
- **Entry point** (`SKILL.md`) containing the actual skill content
- Best suited for judgment, methodology, synthesis, and safety constraints

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

## Proposed Execution Domain: Actions

Actions are typed executable wrappers over exactly one Vrooli-controlled CLI command. They are proposed as a first-class entity so agents can discover deterministic operations without reading long prose skills.

**Key Characteristics:**
- Declares stable input and output schemas
- Calls one controlled command such as `vrooli ...`, `prompt-manager ...`, or a lifecycle-managed scenario CLI
- Declares permissions before execution
- Provides examples and validation
- Contains no branching, routing, shell pipelines, or implementation logic

**Intended Storage:**
```
store/actions/packs/{pack}/{action-id}/
├── action.json
└── history.jsonl
```

**Boundary:**
```text
Skill = how to decide
Action = what to run
CLI = how it works
```

See [Actions](ACTIONS.md) for the full contract.

## How They Work Together

The current domains connect through **relations** for team membership and **markdown references** for skill usage. The Action layer adds discoverable execution contracts that agents can call after deciding what operation is appropriate.

### Flow: Agent Gets Assigned to Team

1. Agent `alice` is created with base capabilities
2. Agent files (SOUL.md, RESPONSIBILITIES.md) reference relevant skills in markdown
3. Team-member relation adds `alice` to `engineering` team with `developer` role
4. When `alice` needs guidance, it reads skill references from its files and team shared docs
5. When `alice` needs deterministic execution, it discovers and runs an exact Action if one exists

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

Teams do not execute their plans directly. Instead, the member that found a signal files it once into the unified `swarm-manager` stream: raw observations use `swarm-manager captures create`, while shaped outcomes use `swarm-manager backlog create`. The operator disposition is read later with `swarm-manager backlog list --actor-id=<verified-profile-key>` and `swarm-manager backlog get`.

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

Actions do not replace this staging layer. If a missing operation needs new scenario/resource/project behavior, the correct output is still a backlog item or `capability-work`. Once the CLI behavior exists and is stable, an Action can wrap it for future execution.

**Team-to-backlog mapping**:

| Team | Backlog Kind | Purpose |
|------|-------------|---------|
| Feature Team | `idea` or `execute` | New capabilities and enhancements |
| QA Team | `fix` or `execute` | Quality issues and test improvements |

See the [swarm-manager work-authoring skill](../../store/skills/packs/core/swarm-manager-work-authoring/SKILL.md) for the filing contract.

## Coordination Skills

Teams inject coordination guidance into agent prompts via dedicated skills. The resolved `coordination.pattern` determines which skill is referenced:

| Coordination Pattern | Skill | Purpose |
|----------------------|-------|---------|
| `independent` | `team-coordination-independent` | Specialist-first execution with minimal coordination overhead |
| `peer` | `team-coordination-peer` | Lateral coordination between teammates without a standing lead |
| `leader-led` | `team-coordination-leader-led` | Explicit lead/report structure with delegated work and synthesis |

Runtime mode and queue policy are resolved separately from coordination pattern:
- `runtime.mode` decides whether the team runs as separate heartbeat processes or as a single Claude Code leader session.
- `execution.queuePolicy` decides whether execution is `serialized` or `bounded-parallel`.
- `coordination.capabilities` decide which prompt sections and durable-state surfaces are enabled.

### Independent Coordination

Independent teams optimize for specialist autonomy:
- No lead is required.
- Messaging can be fully disabled.
- Agents rely on responsibilities, heartbeat instructions, durable logs, and handoffs rather than active delegation.

### Peer Coordination

Peer teams optimize for lateral collaboration:
- No standing lead exists.
- Async inbox messaging and peer triggers can be enabled when useful.
- Agents coordinate directly to avoid duplicate work and unblock dependencies.

### Leader-Led Coordination

Leader-led teams optimize for synthesis and explicit delegation:
- A lead agent is required.
- Single-process leader-led teams run through Claude Code interop.
- Multi-process leader-led teams can still use async inbox messaging and persisted team state.

The coordination skill is a static behavioral layer. Prompt Manager also injects the resolved runtime, coordination, messaging, queue, and durable-state policy directly into the generated heartbeat prompt.

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
- [WORLD-ARCHITECTURE.md](WORLD-ARCHITECTURE.md) - World visualization
