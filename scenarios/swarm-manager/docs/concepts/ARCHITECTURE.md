# Swarm Manager Architecture

## Mental Model

Swarm Manager is the **staging and review layer** for agent-generated plans. Its primary role is to receive work proposals from prompt-manager agent teams, let operators review and refine them, and then control when and how they execute.

```
prompt-manager                         swarm-manager
┌────────────────────────┐             ┌──────────────────────────────────┐
│  Agent Teams           │             │  STAGING / REVIEW / EXECUTION    │
│  ┌──────────────────┐  │             │                                  │
│  │ Debug Team       │──┼── fix ─────▶│  ┌──────────┐                    │
│  │ Feature Team     │──┼── idea ────▶│  │ BACKLOG  │  Operator reviews  │
│  │ QA Team          │──┼── fix ─────▶│  │ items    │  plans, uses       │
│  │ Refactor Team    │──┼── execute ─▶│  └────┬─────┘  Workshop loop     │
│  └──────────────────┘  │             │       │        to refine          │
│                        │             │       ▼                           │
│  Skills define how     │             │  ┌──────────┐                    │
│  teams analyze and     │             │  │ WORKSHOP │  iterative rounds  │
│  produce findings      │             │  │  LOOP    │  → plan.md          │
│                        │             │  └────┬─────┘                    │
└────────────────────────┘             │       │                           │
                                       │       ▼                           │
                                       │  ┌──────────────┐                │
                                       │  │  EXECUTION   │  manual /      │
                                       │  │  CONTROL     │  scheduled /   │
                                       │  └──────┬───────┘  yolo          │
                                       │         │                         │
                                       │         ▼                         │
                                       │  Generator / Improver agents      │
                                       │  build or iterate the scenario    │
                                       └──────────────────────────────────┘
```

**Why this matters:** Agent teams in prompt-manager do analysis and produce plans, but they do not execute directly. Instead, they deposit their findings as backlog items into swarm-manager. This gives the operator a single place to review all agent-generated plans, refine them through the workshop loop (iterative rounds of questions, proposals, and readiness scoring), and control execution -- effectively a "pull request review" for agent work.

Recommendation generation lives in Prompt Manager teams, not in Swarm Manager.

## Domain Concepts

| Concept | Description | Lifecycle States | Implementation |
|---------|-------------|------------------|----------------|
| **Backlog Item** | Unit of work stored as git-tracked folders (`idea`, `research`, `fix`, `execute`, `chore`) | `backlog` -> `researching` -> `ready` -> `queued` -> `in_progress` -> `completed`/`failed`/`archived` | [CODE: ui/src/types/domain.ts#BacklogItem] |
| **Initiative** | Lightweight grouping of related backlog items by a shared label plus explicit initiative metadata | Derived from member items with explicit operator-managed metadata (`name`, `title`, `description`, `status`) | [CODE: api/internal/initiatives/] |
| **Dependency** | Directed edge between backlog items (`depends_on` field in spec.json) | N/A (structural, validated on write) | [CODE: api/internal/depgraph/] |
| **Execution Run** | Governed execution record linked to backlog work | `pending` -> `scheduled` -> `running` -> `completed`/`failed`/`canceled` | [CODE: ui/src/types/domain.ts#ExecutionRecord] |
| **Scenario** | Runtime scenario in the Vrooli ecosystem | `running`, `stopped`, `error`, `unknown` | [CODE: ui/src/types/domain.ts#Scenario] |

### Initiatives

Initiatives are a lightweight grouping mechanism. Each backlog item may carry an `initiative` string field in its `spec.json`. Items sharing the same initiative value are considered members of that initiative. Initiative metadata is managed through the initiatives API or supplied inline during backlog batch-create preview/create. Initiative updates are partial: callers only send the fields they intend to change.

### Dependencies

Backlog items can declare dependencies on other items via the `depends_on` field in `spec.json`. Each entry is a `"kind/name"` reference (e.g., `"fix/auth-bug"`). The `depgraph` package builds a directed acyclic graph from these references and provides:

- **Cycle detection** -- rejects writes that would introduce circular dependencies
- **Topological sort** -- determines safe execution order for batch operations
- **Validation** -- ensures all referenced items actually exist

## Key Flows

1. **Backlog creation and refinement**
   ```
   Team finding -> Backlog item (idea/fix/execute/research/chore) -> workshop loop -> plan.md -> queue
   ```
   All backlog kinds use the **workshop loop** for iterative refinement. Each round generates questions, proposals, and readiness scores across 5 dimensions. The loop continues until all dimensions reach score 3, producing `plan.md` as the primary execution artifact. See [DOC: docs/guides/workshop-workflow.md] for the full pipeline, schemas, and readiness model.

2. **Archive scenario into backlog context**
   ```
   Scenario delete with archive=true -> scenario removed -> archived backlog idea created with preserved files
   ```

3. **Batch operations**
   ```
   POST /api/v1/backlog/batch (preview=true) -> validate items + initiative plan + dependency refs -> no writes
   POST /api/v1/backlog/batch -> apply initiative changes -> create items atomically -> assign initiative membership -> auto-trigger workshop init
   POST /api/v1/backlog/batch/queue -> topological sort via depgraph -> queue items in dependency order
   ```

4. **Execution lifecycle**
   ```
   Queue backlog item (manual/scheduled/yolo) -> execution run record -> agent-manager run -> status tracked in Execution page
   ```

4. **Scenario lifecycle control**
   ```
   List scenarios -> inspect details -> start/stop/restart/delete/archive
   ```

## Logical Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ PRESENTATION LAYER (UI)                                     │
│ Backlog, Scenarios, Execution, Prompts, Settings pages       │
├─────────────────────────────────────────────────────────────┤
│ API GATEWAY LAYER (Go API)                                  │
│ HTTP/proto endpoints, validation, response contracts         │
├─────────────────────────────────────────────────────────────┤
│ DOMAIN LOGIC LAYER                                           │
│ Backlog + initiatives + depgraph + overview + scenarios +    │
│ execution + skills + settings orchestration                  │
├─────────────────────────────────────────────────────────────┤
│ INTEGRATION LAYER                                            │
│ agent-manager + prompt-manager + ecosystem-manager + CLI     │
├─────────────────────────────────────────────────────────────┤
│ PERSISTENCE LAYER                                            │
│ Filesystem: backlog folders + .vrooli/*.json state           │
└─────────────────────────────────────────────────────────────┘
```

### Current Implementation State

| Layer | Status | Notes |
|-------|--------|-------|
| Presentation | Functional | 5 primary tabs wired (`backlog`, `scenarios`, `execution`, `prompts`, `settings`) |
| API Gateway | Implemented | Health, backlog (incl. batch), scenarios, settings, queue, execution, prompts, initiatives, overview, captures, agent-manager status |
| Domain Logic | Implemented | CRUD, archive, queue, research, batch ops, dependency graph, initiatives, overview aggregation, execution scheduling and run control |
| Integration | Implemented | Discovery-based clients (agent-manager, prompt-manager) and CLI-backed scenario operations |
| Persistence | Filesystem-first | Backlog items and execution/settings/queue JSON persisted on disk |

## Workshop Readiness Model

The workshop system uses a 5-dimension readiness model to measure how prepared a backlog item is for execution. Each dimension is scored 0-3 per round by the workshop agent:

| Dimension | Measures |
|-----------|----------|
| `problem_clarity` | Is the problem well-understood? |
| `scope_defined` | Are boundaries and deliverables defined? |
| `approach_solid` | Is the technical approach viable? |
| `testable` | Can success be verified? |
| `risk_awareness` | Are risks identified and mitigated? |

A **round-based boost** rewards iterative engagement: `effective = raw >= 2 ? min(3, raw + floor(rounds/N)) : raw`, where N varies by kind (1 for fix/chore, 2 for idea/research/execute). An item is **ready** when all 5 effective scores reach 3.

The primary output of the workshop loop is `plan.md`, which serves as the execution specification handed to Generator/Improver agents. Workshop rounds are supporting evidence and audit trail.

See [DOC: docs/guides/workshop-workflow.md] for the full workshop pipeline and schemas. Readiness computation lives in [CODE: api/internal/workshop/workshop.go].

## Physical Structure

Key implementation files:
- Domain types: [CODE: ui/src/types/domain.ts]
- API routes/composition: [CODE: api/main.go]
- Backlog service: [CODE: ui/src/services/backlog-service.ts]
- Scenarios service: [CODE: ui/src/services/scenarios-service.ts]
- Execution service: [CODE: ui/src/services/execution-service.ts]
- CLI commands: [CODE: cli/app.go]

### API Package Structure

The backlog handler has been decomposed from a single large file into focused modules:

```
api/internal/
├── backlog/           # Backlog domain (refactored)
│   ├── types.go       # Domain types and interfaces
│   ├── store.go       # Filesystem CRUD abstraction
│   ├── handler.go     # HTTP route registration and core handlers
│   ├── files.go       # File upload/download handlers
│   ├── research.go    # Research spawn handlers
│   ├── queue_ops.go   # Queue/dequeue handlers
│   ├── archive_handlers.go  # Archive operations
│   ├── convert.go     # Kind conversion handlers
│   ├── batch_handler.go     # Batch create (all-or-nothing)
│   └── batch_queue_handler.go # Batch queue (topological order)
├── depgraph/          # Dependency graph (pure computation)
│   └── graph.go       # Cycle detection, topological sort
├── initiatives/       # Initiative CRUD + rollup status
├── overview/          # Aggregation endpoint (backlog + initiatives + graph + stats)
├── captures/          # Capture CRUD and classification
├── skills/            # Centralized skill registry (skill name -> prompt-manager path)
├── workshop/          # Readiness scoring, round I/O
├── execution/         # Execution run lifecycle
├── scenarios/         # Scenario CRUD and lifecycle
├── queue/             # Queue state operations
├── settings/          # Settings persistence
├── prompts/           # Prompt skill CRUD
└── integrations/      # agent-manager, prompt-manager, ecosystem-manager clients
```

## API Boundaries

- `/health`, `/api/v1/health` - health and readiness
- `/api/v1/backlog/*` - backlog CRUD, queue, research, convert
- `/api/v1/backlog/batch` - batch create (all-or-nothing with dependency validation)
- `/api/v1/backlog/batch/queue` - batch queue (topologically sorted, dependency-aware)
- `/api/v1/initiatives/*` - initiative CRUD with rollup status from member items
- `/api/v1/overview` - aggregated view (backlog, initiatives, dependency graph, summary stats)
- `/api/v1/captures/*` - capture CRUD and AI classification
- `/api/v1/scenarios/*` - scenario list/detail/lifecycle/delete/archive
- `/api/v1/settings/*` - settings persistence
- `/api/v1/queue/*` - queue state operations
- `/api/v1/execution/*` - execution runs and policy operations
- `/api/v1/prompts/*` - prompt skill map, CRUD, versions, revert, preview, simulate
- `/api/v1/agent-manager/status` - agent-manager availability

## Meta-Orchestrator Skill

The `swarm-manager-meta-orchestrator` skill is the primary entry point for turning large goals into structured backlog imports. Its actual flow is:

1. Parse high-level input into clusters and candidate items
2. Discuss and refine the plan with the user, potentially across many turns before creation
3. Inspect existing scenarios/codepaths when the target systems already exist
4. Shape items with canonical backlog fields (`initiative`, `depends_on`, `acceptance_allow`, `acceptance_deny`, `research_target`)
5. Preview the multi-initiative import through `backlog batch-create --preview`
6. Create the items only after user approval

The skill intentionally supports long pre-creation planning so workshop auto-spawn happens only after the backlog descriptions are front-loaded with useful context.

## Design Principles

1. **Backlog-first governance**: all planned scenario changes are represented as backlog artifacts.
2. **Execution visibility**: every run has trackable state in the execution domain.
3. **Delegated implementation**: Swarm Manager governs work; agent-manager performs work.
4. **File-based context**: backlog artifacts remain human-readable and git-trackable.
5. **Prompt-manager team ownership**: research and recommendations are generated by teams and written into backlog items.
6. **Dependency-aware ordering**: batch operations respect the dependency graph to ensure items are processed in safe topological order.
7. **Canonical backlog contract**: change boundaries are expressed with `acceptance_allow` / `acceptance_deny`; `scope` is not part of the backlog model.
