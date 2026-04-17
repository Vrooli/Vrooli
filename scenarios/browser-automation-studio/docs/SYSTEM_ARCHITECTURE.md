# Vrooli Ascension - Complete System Architecture

## 🎯 Mental Model: The Big Picture

This document provides a holistic view of how all major components of Vrooli Ascension fit together, from HTTP request to browser automation to database persistence to WebSocket streaming.

---

## 📊 System Overview Diagram

```mermaid
flowchart TB
    subgraph Client["Client Layer"]
        UI["React UI (Vite + TypeScript)\n• React Flow workflow builder\n• Execution viewer with replay\n• WebSocket live updates\n• Zustand state management"]
        CLI["Bash CLI\n• Workflow CRUD\n• Execute & watch\n• Export & render\n• WebSocket streaming"]
    end

    subgraph API["API Server (Go + Chi)"]
        ROUTER["HTTP Router\n• Chi multiplexer\n• CORS middleware\n• Request validation"]

        subgraph Handlers["handlers/*"]
            WF_HANDLER["Workflow Handlers\n• Create/Update/Delete\n• Execute/List\n• Validation"]
            EXEC_HANDLER["Execution Handlers\n• List/Get/Cancel\n• Export/Render\n• Timeline/Artifacts"]
            AI_HANDLER["AI Handlers\n• Generate workflow\n• Debug failures\n• DOM/element analysis"]
            REC_HANDLER["Recording Handler\n• Import Chrome ext\n• Normalize archives"]
            EXPORT_HANDLER["Export Handlers\n• Replay packages\n• Video rendering\n• Asset bundling"]
        end
    end

    subgraph Services["services/* (Business Logic)"]
        WF_SVC["WorkflowService\n• Orchestrates execution\n• Engine selection\n• Validation"]
        REPLAY_SVC["ReplayRenderer\n• Video capture\n• FFmpeg processing\n• Asset assembly"]
        REC_SVC["RecordingService\n• Archive parsing\n• Normalization"]
        EXPORT_SVC["ExportService\n• Package assembly\n• Asset resolution"]
    end

    subgraph Automation["automation/* (Engine-Agnostic Execution Stack)"]
        CONTRACTS["contracts/\n• CompiledInstruction\n• StepOutcome\n• EngineCapabilities\n• Schema versioning"]

        COMPILER["compiler/\n• Workflow → ExecutionPlan\n• Node type registry\n• Parameter validation"]

        EXECUTOR["executor/\n• SimpleExecutor: orchestration\n• FlowExecutor: graph traversal\n• Retries, loops, branches\n• Heartbeats & telemetry\n• Variable interpolation"]

        ENGINE["engine/\n• AutomationEngine interface\n• Factory pattern\n• PlaywrightEngine (HTTP client)\n• Engine selection (ENV)"]

        RECORDER["recorder/\n• DBRecorder: persistence\n• ID generation\n• Artifact storage\n• Truncation/dedupe"]

        EVENTS["events/\n• EventSink interface\n• Sequencer: ordering\n• WSHubSink: WebSocket\n• MemorySink: testing"]
    end

    subgraph External["External Systems"]
        PW_DRIVER["Playwright Driver\n(Node.js/TypeScript)\n• HTTP server :39400\n• Session management\n• 28 instruction handlers\n• Telemetry collection"]

        CHROMIUM["Chromium Browser\n• Playwright automation\n• CDP protocol\n• Screenshot/DOM capture"]
    end

    subgraph Data["Data Layer"]
        SQLITE["SQLite (embedded)\n• Workflows & projects index\n• Executions & schedules\n• Credit usage & operation log\n• UX metrics traces"]

        MINIO["MinIO (S3)\n• Screenshots\n• Videos/traces\n• HAR files\n• Export bundles"]

        FILESYSTEM["Filesystem\n• Recordings/\n• Temp exports/\n• Local assets"]
    end

    subgraph Realtime["Real-time Layer"]
        WS_HUB["WebSocket Hub\n• Per-execution rooms\n• Event broadcasting\n• Client registry"]
    end

    subgraph Workflow["workflow/* (Workflow System)"]
        VALIDATOR["Validator\n• JSON schema\n• Type checking\n• Selector resolution"]
        SCHEMA["Schema Definitions\n• Node types\n• Parameter schemas\n• Validation rules"]
    end

    %% Client to API connections
    UI -->|HTTP REST| ROUTER
    CLI -->|HTTP REST| ROUTER
    UI -->|WebSocket| WS_HUB
    CLI -->|WebSocket| WS_HUB

    %% Router to Handlers
    ROUTER --> WF_HANDLER
    ROUTER --> EXEC_HANDLER
    ROUTER --> AI_HANDLER
    ROUTER --> REC_HANDLER
    ROUTER --> EXPORT_HANDLER

    %% Handlers to Services
    WF_HANDLER --> WF_SVC
    EXEC_HANDLER --> WF_SVC
    EXEC_HANDLER --> EXPORT_SVC
    AI_HANDLER --> WF_SVC
    REC_HANDLER --> REC_SVC
    EXPORT_HANDLER --> REPLAY_SVC

    %% Services to Automation Stack
    WF_SVC --> COMPILER
    WF_SVC --> EXECUTOR
    WF_SVC --> VALIDATOR

    %% Automation Stack Internal Flow
    COMPILER -->|ExecutionPlan| EXECUTOR
    EXECUTOR -->|Resolve engine| ENGINE
    ENGINE -->|HTTP| PW_DRIVER
    PW_DRIVER -->|Playwright API| CHROMIUM
    CHROMIUM -->|Screenshots/DOM| PW_DRIVER
    PW_DRIVER -->|StepOutcome| ENGINE
    ENGINE -->|StepOutcome| EXECUTOR
    EXECUTOR -->|RecordOutcome| RECORDER
    EXECUTOR -->|Events| EVENTS
    EVENTS -->|Broadcast| WS_HUB

    %% Data persistence
    RECORDER --> SQLITE
    RECORDER --> MINIO
    WF_SVC --> SQLITE
    REC_SVC --> SQLITE
    REC_SVC --> FILESYSTEM
    REPLAY_SVC --> SQLITE
    REPLAY_SVC --> MINIO

    %% Validation
    WF_SVC --> VALIDATOR
    VALIDATOR --> SCHEMA

    %% Contracts flow through system
    CONTRACTS -.->|Shared types| COMPILER
    CONTRACTS -.->|Shared types| EXECUTOR
    CONTRACTS -.->|Shared types| ENGINE
    CONTRACTS -.->|Shared types| RECORDER
    CONTRACTS -.->|Shared types| EVENTS

    classDef client fill:#e1f5ff,stroke:#01579b
    classDef api fill:#fff3e0,stroke:#e65100
    classDef service fill:#f3e5f5,stroke:#4a148c
    classDef automation fill:#e8f5e9,stroke:#1b5e20
    classDef data fill:#fce4ec,stroke:#880e4f
    classDef external fill:#fff9c4,stroke:#f57f17

    class UI,CLI client
    class ROUTER,WF_HANDLER,EXEC_HANDLER,AI_HANDLER,REC_HANDLER,EXPORT_HANDLER api
    class WF_SVC,REPLAY_SVC,REC_SVC,EXPORT_SVC service
    class CONTRACTS,COMPILER,EXECUTOR,ENGINE,RECORDER,EVENTS automation
    class SQLITE,MINIO,FILESYSTEM data
    class PW_DRIVER,CHROMIUM external
```

---

## 🏗️ Component Deep Dive

### 1. Client Layer

**React UI** (`ui/`)
- Visual workflow builder using React Flow
- Execution viewer with real-time updates
- Replay tab with timeline scrubbing
- WebSocket connection for live telemetry
- State management via Zustand

**Bash CLI** (`cli/`)
- Thin wrapper around API
- WebSocket support for `execution watch`
- Export/render commands
- JSON output for automation

### 2. HTTP/API Layer

**Router** (`main.go`)
- Chi HTTP router
- CORS middleware
- Health checks
- Request/response logging

**Handlers** (`handlers/`)
- **Workflows**: CRUD, execute, validate
- **Executions**: List, get, cancel, export
- **AI**: Generate, debug, analyze DOM
- **Recordings**: Import Chrome extension archives
- **Export**: Replay packages, video rendering

### 3. Services Layer (`services/`)

**WorkflowService**
- Orchestrates workflow execution
- Selects automation engine
- Invokes compiler → executor → recorder → events
- Handles validation

**ReplayRenderer**
- Captures frames via Playwright
- Processes video with FFmpeg
- Assembles export bundles

**RecordingService** (`services/recording/`)
- Unified action recording (manual, AI, playback all use same pipeline)
- Dual-write strategy: hot cache + SQLite persistence
- WebSocket broadcast with delivery metrics (BroadcastResult)
- Correlation ID tracing for debugging
- Navigate action deduplication (500ms window)
- Implements `ActionRecorder` interface for full observability

**ArchiveIngestionService** (`services/archive-ingestion/`)
- Parses Chrome extension archives
- Normalizes to standard format
- Stores artifacts

**ExportService**
- Assembles replay packages
- Resolves asset URLs
- Bundles screenshots/DOM/metadata

### 4. Automation Stack (`automation/`)

This is the **heart of the system** - an engine-agnostic execution framework.

#### **Contracts** (`contracts/`)
Engine-agnostic type definitions:
- `CompiledInstruction` - Generic instruction format (type + params)
- `StepOutcome` - Normalized result (success, telemetry, screenshots, errors)
- `EngineCapabilities` - Feature advertisement (HAR, video, iframes, etc.)
- `StepFailure` - Taxonomized errors for retry logic
- Schema versioning for backward compatibility

#### **Compiler** (`compiler/`)
- Transforms workflow JSON → `ExecutionPlan`
- Validates node types and parameters
- Generates `CompiledInstruction` array
- No engine-specific code

#### **Executor** (`executor/`)
The orchestration layer:
- **SimpleExecutor**: Retry logic, heartbeats, outcome normalization
- **FlowExecutor**: Graph traversal, branching, loops (repeat/forEach/while)
- Variable interpolation (`${var}` replacement)
- Capability preflight checks
- Session lifecycle management (fresh/clean/reuse modes)

Flow:
```
1. Receive ExecutionPlan
2. Validate engine capabilities
3. For each instruction:
   a. Emit step.started event
   b. Start heartbeat timer
   c. Run with retries
   d. Normalize outcome
   e. Record to DB
   f. Emit step.completed/failed
4. Handle loops/branches
5. Clean up session
```

#### **Engine** (`engine/`)
Browser automation abstraction:
- `AutomationEngine` interface (Name, Capabilities, StartSession)
- `EngineSession` interface (Run, Reset, Close)
- `PlaywrightEngine` - HTTP client to Playwright driver
- Factory pattern for engine selection
- Environment-based selection (`ENGINE`, `ENGINE_OVERRIDE`)

#### **Recorder** (`recorder/`)
Persistence layer:
- `DBRecorder` - Saves `StepOutcome` to PostgreSQL
- Generates UUIDs and correlation IDs
- Applies size limits to DOM/console/network
- Uploads screenshots to MinIO
- Handles truncation/deduplication

#### **Events** (`events/`)
Real-time streaming:
- `EventSink` interface
- `Sequencer` - Enforces per-execution ordering
- `WSHubSink` - WebSocket broadcasting
- `MemorySink` - In-memory for tests
- Backpressure handling (drop counters)
- Never drops completion/failure events

### 5. Playwright Driver (`playwright-driver/`)

**TypeScript HTTP server** (port 39400):
- Session management (browser/context/page lifecycle)
- 28 instruction handlers (navigation, interaction, assertions, etc.)
- Telemetry collection (console, network, screenshots, DOM)
- Returns `StepOutcome` JSON to Go adapter
- Modular architecture (50+ files)

**Endpoints**:
- `POST /session/start` - Create session
- `POST /session/:id/run` - Execute instruction
- `POST /session/:id/reset` - Clear state
- `POST /session/:id/close` - Cleanup
- `GET /health` - Health check

### 6. Data Layer

**PostgreSQL** (`database/`)
- Workflows, projects, folders
- Executions, steps, artifacts
- Timeline frames
- Users, auth

**MinIO** (S3-compatible)
- Screenshots (PNG)
- Videos, traces, HAR files
- Export bundles

**Filesystem**
- Chrome extension recordings
- Temporary exports
- Local development assets

### 7. WebSocket Layer

**WebSocket Hub** (`websocket/`)
- Per-execution rooms
- Client registry
- Event broadcasting
- Message ordering

**Event Types**:
- `execution.started/completed/failed`
- `step.started/heartbeat/completed/failed`
- `step.telemetry` (console, network)
- `step.screenshot`

### 8. Workflow System (`workflow/`)

**Validator**
- JSON schema validation
- Type checking
- Selector resolution (`@selector/key`)
- Parameter validation

**Schema Definitions**
- Node type registry (28 types)
- Parameter schemas
- Validation rules

### 9. AI Layer (`handlers/ai/`)

- **Workflow Generation**: Prompt → workflow JSON (OpenRouter)
- **Debug Loop**: Failure analysis and fix suggestions
- **DOM Analysis**: Extract element metadata
- **Screenshot Analysis**: Visual element detection

---

## 🔄 Data Flow: Workflow Execution

Let's trace a complete execution from UI click to WebSocket update:

```mermaid
sequenceDiagram
    participant UI as React UI
    participant API as HTTP Handler
    participant SVC as WorkflowService
    participant COMP as Compiler
    participant EXEC as Executor
    participant ENG as PlaywrightEngine
    participant DRV as Playwright Driver
    participant REC as DBRecorder
    participant EVT as WSHubSink
    participant WS as WebSocket Hub
    participant DB as PostgreSQL

    UI->>API: POST /api/v1/workflows/{id}/execute
    API->>SVC: ExecuteWorkflow(workflowID, params)

    SVC->>DB: Load workflow JSON
    DB-->>SVC: Workflow nodes/edges

    SVC->>COMP: Compile(workflow) → ExecutionPlan
    COMP-->>SVC: ExecutionPlan{instructions[]}

    SVC->>EXEC: Execute(plan, engine, recorder, eventSink)

    EXEC->>ENG: Capabilities(ctx)
    ENG-->>EXEC: EngineCapabilities{iframes, HAR, video...}

    EXEC->>EXEC: Validate capabilities vs plan requirements

    EXEC->>ENG: StartSession(spec)
    ENG->>DRV: POST /session/start
    DRV-->>ENG: {session_id}
    ENG-->>EXEC: EngineSession

    loop For each instruction
        EXEC->>EVT: Emit(step.started)
        EVT->>WS: Broadcast to clients
        WS->>UI: WebSocket message

        EXEC->>ENG: session.Run(instruction)
        ENG->>DRV: POST /session/{id}/run
        DRV->>DRV: Execute in browser
        DRV-->>ENG: StepOutcome{success, screenshot, DOM, telemetry}
        ENG-->>EXEC: StepOutcome

        EXEC->>REC: RecordStepOutcome(outcome)
        REC->>DB: Insert execution_step
        REC->>DB: Insert artifacts (console, network, assertions)
        REC->>DB: Upload screenshot → MinIO, save URL
        REC-->>EXEC: RecordResult{IDs}

        EXEC->>EVT: Emit(step.completed, screenshot, telemetry)
        EVT->>WS: Broadcast
        WS->>UI: Live update with screenshot

        opt If session mode = clean
            EXEC->>ENG: session.Reset()
            ENG->>DRV: POST /session/{id}/reset
        end
    end

    EXEC->>ENG: session.Close()
    ENG->>DRV: POST /session/{id}/close

    EXEC->>EVT: Emit(execution.completed)
    EVT->>WS: Broadcast
    WS->>UI: Execution complete

    EXEC-->>SVC: ExecutionResult
    SVC-->>API: HTTP 200 {execution_id}
    API-->>UI: JSON response
```

---

## 🎯 Key Design Principles

### 1. **Engine Agnosticism**
- Contracts define stable types (`CompiledInstruction` → `StepOutcome`)
- Engines are pluggable (Browserless removed, Playwright active)
- No vendor-specific fields in instructions
- Engine selection via environment variables

### 2. **Separation of Concerns**
- **Handlers**: HTTP validation, thin façade
- **Services**: Business logic orchestration
- **Automation**: Pure execution logic
- **Engines**: Browser control only
- **Recorder**: Persistence only
- **Events**: Streaming only

### 3. **Contract-First**
- Schema versioning (`automation-step-outcome-v1`)
- Payload versioning for backward compatibility
- Size limits enforced centrally
- Taxonomized failures for retry logic

### 4. **Observability**
- Comprehensive telemetry (console, network, screenshots, DOM)
- Real-time WebSocket streaming
- Heartbeat events during long operations
- Structured logging throughout
- **Recording pipeline observability** (correlation IDs, BroadcastResult metrics)

### 5. **Testability**
- `MemorySink` for event testing
- Mock engines for unit tests
- Contract tests for compatibility
- Integration tests with testcontainers

---

## 🔑 Critical Paths & Where to Look

### Adding a New Instruction Type
1. Define in `api/automation/compiler/step_types.go`
2. Add handler in `playwright-driver/src/handlers/`
3. Register in `playwright-driver/src/handlers/registry.ts`
4. Update `EngineCapabilities` if needed
5. Add tests in `playwright-driver/tests/unit/handlers/`

### Modifying Execution Flow
1. Graph logic: `api/automation/executor/flow_executor.go`
2. Retry/heartbeat: `api/automation/executor/simple_executor.go`
3. Variable interpolation: `api/automation/executor/flow_utils.go`
4. Capability checks: `api/automation/executor/preflight.go`

### Changing Persistence
1. DB schema: `api/database/migrations/`
2. Repository: `api/database/repository*.go`
3. Recorder: `api/automation/recorder/db_recorder.go`
4. Storage: `api/storage/` (MinIO client)

### Adding WebSocket Events
1. Event types: `api/automation/contracts/events.go`
2. Emission: `api/automation/executor/simple_executor.go`
3. Sink: `api/automation/events/ws_sink.go`
4. UI handling: `ui/src/hooks/useExecutionWebSocket.ts`

### Debugging Recording Issues
1. Correlation ID tracing: `api/handlers/record_mode.go` (generateCorrelationID)
2. Unified recording: `api/services/recording/service.go` (RecordActionUnified)
3. Broadcast metrics: `api/websocket/hub.go` (BroadcastResult)
4. Integration tests: `api/handlers/record_mode_integration_test.go`

### Modifying Replay/Export
1. Capture: `api/services/replay_renderer_playwright_client.go`
2. FFmpeg: `api/services/replay_renderer_ffmpeg.go`
3. Export API: `api/handlers/export/`
4. UI viewer: `ui/src/pages/ReplayTab.tsx`

---

## 🎓 Understanding the System - Mental Shortcuts

**Think of it as layers**:
1. **Presentation**: UI + CLI
2. **API**: HTTP handlers (thin)
3. **Business Logic**: Services (orchestration)
4. **Execution Engine**: Automation stack (the core innovation)
5. **Browser Control**: Playwright driver
6. **Persistence**: DB + Storage
7. **Real-time**: WebSocket events

**The Automation Stack is the key innovation**:
- Before: Monolithic `browserless/client.go` (tightly coupled)
- After: Modular stack with clean interfaces (engine swappable)
- Result: Migrated Browserless → Playwright without touching orchestration

**Contracts are the glue**:
- `CompiledInstruction` - What to do (engine-agnostic)
- `StepOutcome` - What happened (normalized result)
- Everything else builds on these two types

**Data flows in a pipeline**:
```
Workflow JSON → Compiler → ExecutionPlan → Executor → Engine → Browser
                                              ↓
                                           Recorder → DB
                                              ↓
                                           Events → WebSocket → UI
```

---

## 📚 Where to Start

**Want to understand execution?**
- Start: `api/automation/README.md`
- Read: `api/automation/executor/README.md`
- Trace: `api/services/workflow_service_execution.go`

**Want to add features?**
- Instructions: `playwright-driver/docs/ARCHITECTURE.md`
- Flow control: `api/automation/executor/flow_executor.go`
- Validation: `api/workflow/validator/`

**Want to debug issues?**
- Logs: Check `api/automation/executor/simple_executor.go`
- WebSocket: `api/automation/events/ws_sink.go`
- Persistence: `api/automation/recorder/db_recorder.go`

**Want the full picture?**
- This document!

---

## 📖 Related Architecture Documents

For deeper dives into specific subsystems:

| Document | Focus |
|----------|-------|
| [Driver Interface](architecture/driver-interface.md) | Pluggable driver/navigator interface, policy patterns |
| [AI Navigation](architecture/ai-navigation.md) | Vision-based autopilot, navigator registry |
| [Recording](architecture/recording.md) | Manual recording, WebSocket flows, action persistence, **ActionRecorder observability** |
| [Execution](architecture/execution.md) | Workflow execution, telemetry, artifacts |
| [Reconciliation](architecture/reconciliation.md) | State reconciliation patterns |
