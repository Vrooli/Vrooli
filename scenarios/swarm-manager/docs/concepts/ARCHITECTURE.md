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
│  │ QA Team          │──┼── fix ─────▶│  │ items    │  plans, uses Idea  │
│  │ Refactor Team    │──┼── execute ─▶│  └────┬─────┘  Agent to refine   │
│  └──────────────────┘  │             │       │                           │
│                        │             │       ▼                           │
│  Skills define how     │             │  ┌──────────┐                    │
│  teams analyze and     │             │  │  IDEA    │  clarify → suggest │
│  produce findings      │             │  │  AGENT   │  → enhance         │
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

**Why this matters:** Agent teams in prompt-manager do analysis and produce plans, but they do not execute directly. Instead, they deposit their findings as backlog items into swarm-manager. This gives the operator a single place to review all agent-generated plans, refine them with the Idea Agent (clarify/suggest/enhance), and control execution — effectively a "pull request review" for agent work.

Recommendation generation lives in Prompt Manager teams, not in Swarm Manager.

## Domain Concepts

| Concept | Description | Lifecycle States | Implementation |
|---------|-------------|------------------|----------------|
| **Backlog Item** | Unit of work stored as git-tracked folders (`idea`, `research`, `fix`, `execute`) | `backlog` -> `researching` -> `ready` -> `queued` -> `in_progress` -> `completed`/`archived` | [CODE: ui/src/types/domain.ts#BacklogItem] |
| **Execution Run** | Governed execution record linked to backlog work | `pending` -> `scheduled` -> `running` -> `completed`/`failed`/`canceled` | [CODE: ui/src/types/domain.ts#ExecutionRecord] |
| **Scenario** | Runtime scenario in the Vrooli ecosystem | `running`, `stopped`, `error`, `unknown` | [CODE: ui/src/types/domain.ts#Scenario] |

## Key Flows

1. **Backlog creation and refinement**
   ```
   Team finding -> Backlog item (idea/fix/execute/research) -> optional clarify/suggest/enhance -> queue
   ```
   For idea items, the **Idea Agent** provides a 3-phase refinement workflow (clarify questions, suggest improvements, enhance into spec). See [DOC: docs/guides/idea-agent-workflow.md] for the full pipeline, schemas, and visual flow.

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
│ Backlog, Scenarios, Execution, Settings pages               │
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
| Presentation | Functional | 4 primary tabs wired (`backlog`, `scenarios`, `execution`, `settings`) |
| API Gateway | Implemented | Health, backlog, scenarios, settings, queue, execution, agent-manager status |
| Domain Logic | Implemented | CRUD, archive, queue, research, execution scheduling and run control |
| Integration | Implemented | Discovery-based clients (agent-manager, prompt-manager) and CLI-backed scenario operations |
| Persistence | Filesystem-first | Backlog items and execution/settings/queue JSON persisted on disk |

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
- `/api/v1/agent-manager/status` - agent-manager availability

## Design Principles

1. **Backlog-first governance**: all planned scenario changes are represented as backlog artifacts.
2. **Execution visibility**: every run has trackable state in the execution domain.
3. **Delegated implementation**: Swarm Manager governs work; agent-manager performs work.
4. **File-based context**: backlog artifacts remain human-readable and git-trackable.
5. **Prompt-manager team ownership**: research and recommendations are generated by teams and written into backlog items.
