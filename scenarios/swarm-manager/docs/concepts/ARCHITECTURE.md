# Swarm Manager Architecture

## Mental Model

Swarm Manager is the control plane where backlog work is governed and executed.

```
┌──────────────────────────────────────────────────────────────────────────┐
│                            SWARM MANAGER                                 │
│      "Backlog governance and execution control for scenario change"      │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Prompt Manager Teams                                                    │
│  (Debug / Feature / QA / Refactor)                                       │
│          │                                                               │
│          ▼                                                               │
│  ┌────────────────┐     ┌─────────────────┐     ┌────────────────────┐   │
│  │    BACKLOG     │ ──▶ │    EXECUTION    │ ──▶ │  SCENARIO CHANGES   │   │
│  │ idea/research/ │     │ pending/running │     │ via agent-manager    │   │
│  │ fix/execute    │     │ completed/failed│     │ + scenario lifecycle │   │
│  └────────────────┘     └─────────────────┘     └────────────────────┘   │
│          │                                                               │
│          ▼                                                               │
│  ┌────────────────────────────────────────────────────────────────────┐   │
│  │                 External Integrations                              │   │
│  │  agent-manager | ecosystem-manager | vrooli CLI                    │   │
│  └────────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

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
│ agent-manager + ecosystem-manager + Vrooli CLI adapters      │
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
| Integration | Implemented | Discovery-based clients and CLI-backed scenario operations |
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
