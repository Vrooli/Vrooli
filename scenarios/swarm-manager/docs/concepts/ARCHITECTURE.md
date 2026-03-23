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
| **Execution Run** | Governed execution record linked to backlog work | `pending` -> `scheduled` -> `running` -> `completed`/`failed`/`canceled` | [CODE: ui/src/types/domain.ts#ExecutionRecord] |
| **Scenario** | Runtime scenario in the Vrooli ecosystem | `running`, `stopped`, `error`, `unknown` | [CODE: ui/src/types/domain.ts#Scenario] |

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

3. **Execution lifecycle**
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
│ Backlog + scenarios + execution + settings orchestration     │
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
| API Gateway | Implemented | Health, backlog, scenarios, settings, queue, execution, prompts, agent-manager status |
| Domain Logic | Implemented | CRUD, archive, queue, research, execution scheduling and run control |
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

## API Boundaries

- `/health`, `/api/v1/health` - health and readiness
- `/api/v1/backlog/*` - backlog CRUD, queue, research, convert
- `/api/v1/scenarios/*` - scenario list/detail/lifecycle/delete/archive
- `/api/v1/settings/*` - settings persistence
- `/api/v1/queue/*` - queue state operations
- `/api/v1/execution/*` - execution runs and policy operations
- `/api/v1/prompts/*` - prompt skill map, CRUD, versions, revert, preview, simulate
- `/api/v1/agent-manager/status` - agent-manager availability

## Design Principles

1. **Backlog-first governance**: all planned scenario changes are represented as backlog artifacts.
2. **Execution visibility**: every run has trackable state in the execution domain.
3. **Delegated implementation**: Swarm Manager governs work; agent-manager performs work.
4. **File-based context**: backlog artifacts remain human-readable and git-trackable.
5. **Prompt-manager team ownership**: research and recommendations are generated by teams and written into backlog items.
