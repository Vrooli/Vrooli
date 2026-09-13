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

## Teams: departments, committees, and supervision

A Prompt Manager **team is a durable coordination container**, not a promise
that an agent is continuously running. It supplies member identity, roles,
shared context, an operating contract, and configured execution. The same
container can serve different purposes:

| Purpose | Useful organizational analogy | Responsibility and lifetime |
|---|---|---|
| Standing objective-oriented team | A department | Maintain a domain's obligations and measured outcomes over time. There is normally no final "done"; it responds to new evidence and prepares or performs work within its authority. |
| Finite effort team | A delivery task force or temporary committee | Converge on one accepted destination across workers and sessions. Retire its recurring execution after evidenced completion or withdrawal; retain its handoff and receipts. |
| Standing effort-supervision team | Delivery assurance across task forces | Observe discovered efforts, question progress and methodology, and make only authorized interventions. Individual watches retire; the service remains available for other and newly discovered efforts. |

These are explanations of purpose, not one combined `team.json` type enum. A team may
have one member. A committee does not require a permanent roster of every worker:
its leader can delegate bounded assignments through the execution owner. Team
membership, worker lineage, and an Agent Manager run are different identities.

The finite-team row describes intended lifecycle, not a qualified recurring
runtime. The current optional PM `finiteLeader` binding retains one admission
and AM run identity; subsequent heartbeat ticks observe that run. It does not
schedule continuation from an explicit owner wait/next-action decision or prove
accepted effort completion. Binding persistence and retirement fences do not
qualify recurrence or legacy-driver adoption. See the
[finite leader implementation checkpoint](HEARTBEATS.md#finite-effort-leader-binding).

**Director Swarm and Effort Supervision have different decisions to make.**
Director Swarm handles portfolio strategy, priorities, operator-readable work
preparation, and disposition through its existing contract. Effort Supervision
asks whether accepted work is progressing, whether waits and repairs are
justified, and whether delivery and supervision are worth their total cost. It
normally addresses the orchestrator, not the orchestrator's workers. It does not
replace Director Swarm, change the accepted destination, or obtain authority
from being called a supervisor. See the
[supervision ownership contract](../../../../docs/agent-system/EFFORT_SUPERVISION.md#responsibilities-and-ownership).

The department analogy does not mean "one team per objective." Objectives and
teams are many-to-many; a distinct regulated domain, cadence, and failure mode
justify a team. See [the target model](../../../../docs/agent-system/TARGET_MODEL.md#6-objectives-teams-and-members).

### Keep purpose, lifetime, authority, and execution separate

| Question | Source of the answer |
|---|---|
| What must this work achieve? | Accepted objectives/obligations or the effort's destination and acceptance evidence. |
| When does this team or watch retire? | Its operating contract and owner lifecycle; a completed agent turn is not completed work. |
| What may act without another human decision? | The actual grant, allowed effects, budget and exclusions—not team type or heartbeat frequency. |
| How is the work organized? | The selected work shape: phased plan, adaptive mandate, bounded task, or investigation. A committee does not imply a plan family. |
| How and when does a member run? | Prompt Manager heartbeat admission and execution policy, followed by the execution owner's run state. |

Permanent teams often send proposals for human disposition through Swarm
Manager. An already-authorized finite effort can execute autonomously within
its grant. Neither behavior is inherent in the team's lifetime. A standing
supervisor can run autonomous **observations** while remaining unable to resume
or repair the work it observes. Separately approved bounded repair can proceed
through the existing execution owner within its exact grant, scope, budget and
stop conditions; that approval neither grants business steering nor follows
from diagnostic admission. Reconcile any existing repair assignment before
dispatch and verify useful progress after recovery. Permission to coordinate
owner recovery does not authorize the supervisor to edit source or effort files;
its declared member write boundaries still apply. Follow
[work routing and fallback](../../../../docs/agent-system/SWARM_MANAGER_WORK.md)
for already-authorized work when a transitional execution route requires a
human interaction; never fabricate an approval or bypass an actual denial.

### Where to look in the product

#### Team purpose, lifetime, and linked work

The operator-authorized team UX change (2026-09-12) makes these distinctions
visible in the team list and dashboard. Store `purpose` and `lifetime` as
independent optional team properties. Purpose is `domain-stewardship`, `delivery`,
or `supervision`; lifetime is `standing` or `finite`. Missing values mean
unspecified. Neither property changes execution, grants permission, or retires
a heartbeat. Creation presets provide initial values; both properties remain
independently editable. Existing registered domain teams declare standing
stewardship; the effort supervisor declares standing supervision.

Keep existing `objectivesServed` records intact as the authored declaration input.
The canonical current-state authority for objective attachments, roles, coverage and
priority is the objective authority (`objectives/v1`), surfaced by the team page
editor and reconciled against the declarations by `prompt-manager graph objectives`.
Optional `effortRefs` identify the team's delivery or contribution relationships
with canonical efforts. These references do not enroll work or assert current
execution. A supervisor's observed-effort relationships come from its heartbeat
observation and Agent Manager's board, not a second manually maintained list.
Finite leader relationships come from the heartbeat's non-retired `finiteLeader`
binding. Show them separately from authored references and supervisor observations;
a leader binding alone does not establish supervision coverage.

Show the member's declared lane, permitted write surfaces, prohibitions, and
governing contract alongside the current effort's actual steering authority.
Do not infer implementation permission from purpose, lifetime, or coordination
pattern. Unknown or expired authority remains visible. The operator can inspect
the source contract and open the relevant owner view.

Use Agent Manager's existing read-only effort board for work observations.
The Teams view also exposes discovered efforts that have no registered team
binding, including transitional drivers. A complete team registry establishes
authored relationships only; absent `effortRefs` do not establish that no runtime
leader binding exists. Show runtime binding coverage as unknown until observed.
Label an effort unbound only when the relevant owner observation establishes it;
do not create an agent identity, register a team, or migrate a scheduler to make
it appear. Bound each board request, retain pagination and partial coverage,
and distinguish unavailable observations from an empty successful result.

Present scheduling eligibility, current owner execution, and accepted outcome
as separate states. A successful completed run is not an accepted effort. A
team with no execution evidence has unknown execution health. Show observation
time, next action or wait, the supervisor assessment, and requested/effective
model where the owner reports them. Link temporary worker assignments separately
from the registered member roster. Team and effort links preserve exact identity
when opened in the owning UI.

For example, Marketing remains a standing domain team while contributing to an
Aquila launch effort. Aquila's finite delivery team retires after evidenced
completion. Effort Supervision retires its Aquila watch and continues observing
other efforts. The three lifetimes do not alter their respective grants.

Validation must cover metadata create/read/update preservation; unspecified and
independent purpose/lifetime combinations; filtering and presets; finite and
standing examples; unbound efforts; stale/unavailable observations; expired
authority; and independent schedule, run, and outcome states. A served browser
check verifies that team-to-effort navigation preserves the selected reference.

Prompt Manager's team page describes the people/roles, instructions, knowledge,
and heartbeat configuration. Agent Manager's **Efforts** page and
`agent-manager effort board` describe the observed work, evidence, limits, and
supervisor assessments. Plan Manager owns any selected plans and families;
Swarm Manager owns its grants and disposition. These are linked views, not
competing work ledgers.

Do not infer that an effort is a Prompt Manager committee merely because its
workspace or tmux session exists. Transitional efforts can still use temporary
drivers outside team execution. Discovery makes them visible; it does not
migrate their scheduler, register a controllable leader, or delegate authority.
The standing `effort-supervision` team is distinct from a finite implementation
workspace with the same short name. For current operation and recovery limits,
read [standing supervision](HEARTBEATS.md#standing-effort-supervision).

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

Teams do not execute their plans directly. Instead, the member that found a signal files it once into the unified `swarm-manager` stream: raw observations use `swarm-manager captures create`, while shaped outcomes use `swarm-manager backlog create`. Material implementation work is then shaped through Plan Manager and receives one canonical `plan_ref`. The plan has a **shape**, `phased` or `mandate`, defined in [Scenario development](../../../../docs/agent-system/SCENARIO_DEVELOPMENT.md#grant-plan-shape-and-execution-mode). The operator grants the item and picks an **execution mode** in the Run dialog: `sliced` (one bounded worker run per slice with an independent review, through the `phased-plan-drain` workflow) or `goal` (one Agent Manager run that carries the finish line as a harness goal). Any shape runs under either mode. After execution, Swarm **finalization** restarts the scenario, checks health, gathers evidence, runs the review agent, and sets the item to done, needs_review, or follow-up. The operator disposition is read later with `swarm-manager backlog list --actor-id=<verified-profile-key>` and `swarm-manager backlog get`.

```
prompt-manager (teams analyze)          swarm-manager (staging/review)
┌──────────────────────────┐            ┌──────────────────────────────┐
│  Feature Team  → idea    │──┐         │ Backlog item + outcome       │
│  QA Team       → fix     │──┼────────▶│          ↓                   │
│  Other owner  → evidence │──┘         │ Plan Manager plan_ref        │
└──────────────────────────┘            │   shape: phased | mandate    │
                                        │          ↓                   │
                                        │ Operator grant + mode        │
                                        │   ├ sliced (workflow)        │
                                        │   └ goal (one run)           │
                                        └──────────┬───────────────────┘
                                                   ↓
                                        Agent Manager run(s) + evidence
                                                   ↓
                                        Swarm finalization → outcome
```

**Why staging matters:**
- Operators get a single place to review all agent-generated plans
- Swarm goals and plans shape intent and implementation separately; neither
  approves or launches work automatically
- Execution governance (manual/scheduled/yolo) controls when approved work runs
- Plans are git-tracked, human-readable, and editable before committing to execution

**Implementation status.** The backlog item today carries `execution_strategy`
with the values `phased-plan-drain`, `adaptive-improvement`, and `goal-session`;
the plan `shape` field does not exist yet. The interim mapping to the target
vocabulary is in
[SCENARIO_DEVELOPMENT.md](../../../../docs/agent-system/SCENARIO_DEVELOPMENT.md#implementation-status).

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
