# API Architecture

This document provides a visual overview of the scenario-to-desktop API architecture. For detailed seam definitions and testability patterns, see [SEAMS.md](../internal/SEAMS.md).

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              HTTP SERVER (main.go)                          │
│                                  Port 15200                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                              Domain Handlers                                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
│  │  Build   │ │ Records  │ │ Scenario │ │  System  │ │    Pipeline      │  │
│  │ Handler  │ │ Handler  │ │ Handler  │ │ Handler  │ │    Handler       │  │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────────┬─────────┘  │
│       │            │            │            │                 │            │
│  ┌────┴────────────┴────────────┴────────────┴─────────────────┴─────────┐  │
│  │                        Tool Execution Layer                           │  │
│  │   POST /api/v1/tools/execute    GET /api/v1/tools                    │  │
│  └───────────────────────────────────┬───────────────────────────────────┘  │
└──────────────────────────────────────┼──────────────────────────────────────┘
                                       │
                                       ▼
```

The server follows a **screaming architecture** pattern where each domain owns its handler. The central server struct orchestrates domain handlers for build, telemetry, records, scenario, system, pipeline, state, deploy-target management, and tools.

**Key characteristics:**
- JSON structured logging with `slog`
- Port configuration via `API_PORT` or `PORT` environment variables (default: 15200)
- CORS and logging middleware on all routes

---

## Pipeline System (Core Engine)

The pipeline orchestrator is the core engine that coordinates multi-stage desktop deployment workflows.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                             PIPELINE ORCHESTRATOR                           │
│         Manages execution flow, cancellation, resume, idempotency           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────┐    ┌───────────┐    ┌──────────┐    ┌─────────┐    ┌───────┐ │
│   │ BUNDLE  │───▶│ PREFLIGHT │───▶│ GENERATE │───▶│  BUILD  │───▶│ SMOKE │ │
│   │         │    │           │    │          │    │         │    │ TEST  │ │
│   │ Package │    │ Validate  │    │ Electron │    │ Compile │    │       │ │
│   │ assets  │    │ prereqs   │    │ wrapper  │    │ native  │    │ Verify│ │
│   └─────────┘    └───────────┘    └──────────┘    └─────────┘    └───┬───┘ │
│                                                                       │     │
│                                                   ┌───────────────────┘     │
│                                                   ▼                         │
│                                           ┌──────────────┐                  │
│                                           │    DEPLOY    │                  │
│                                           │              │                  │
│                                           │ Upload via   │                  │
│                                           │ LPBS proxy   │                  │
│                                           └──────────────┘                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Stage Descriptions

| Stage | Purpose | Skippable | Dependencies |
|-------|---------|-----------|--------------|
| **Bundle** | Package scenario resources and assets | Yes (if `deployment_mode=proxy`) | None |
| **Preflight** | Validate system prerequisites (Node, npm, Wine, Xcode) | Yes (via config) | None |
| **Generate** | Create Electron wrapper for the scenario | No (required) | Preflight |
| **Build** | Compile native binaries per platform (Windows, macOS, Linux) | No | Generate |
| **SmokeTest** | Verify built application runs correctly ([details](./smoke-test-pipeline.md)) | Yes (via config) | Build |
| **Deploy** | Upload artifacts via LPBS remote profile flow | Yes | SmokeTest |

### Pipeline States

```
                    ┌──────────────┐
                    │    idle      │◀─── CreateIdlePipeline()
                    └──────┬───────┘
                           │ StartPipeline()
                           ▼
                    ┌──────────────┐
                    │   pending    │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
     CancelPipeline │   running    │
    ───────────────▶└──────┬───────┘
           │               │
           ▼               │
    ┌──────────────┐       │
    │  cancelled   │       ├───────────────┬────────────────┐
    └──────────────┘       │               │                │
                           ▼               ▼                ▼
                    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
                    │  completed   │ │    failed    │ │   skipped    │
                    └──────────────┘ └──────────────┘ └──────────────┘
```

**State transitions:**
- `idle` → `pending` → `running` → `completed`/`failed`/`skipped`
- `running` can be cancelled at any time
- Failed/stopped pipelines can be resumed from their last stage

---

## Tool Execution System

The Tool Execution Protocol provides a unified interface for AI agents to invoke any capability.

```
┌──────────────────────────────────────────────────────────────────────┐
│                         ServerExecutor                                │
│                                                                       │
│   POST /api/v1/tools/execute                                         │
│   { "tool_name": "...", "arguments": {...} }                         │
│                                                                       │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐  │
│   │    Pipeline     │    │     Signing     │    │   Inspection    │  │
│   │    Executor     │    │    Executor     │    │    Executor     │  │
│   ├─────────────────┤    ├─────────────────┤    ├─────────────────┤  │
│   │ run_pipeline    │    │ configure_sign  │    │ check_build     │  │
│   │ check_status    │    │ sign_app        │    │ list_wrappers   │  │
│   │ cancel_pipeline │    │ verify_sig      │    │ validate_config │  │
│   │ resume_pipeline │    │ discover_certs  │    │ get_prereqs     │  │
│   │ list_pipelines  │    │                 │    │                 │  │
│   └─────────────────┘    └─────────────────┘    └─────────────────┘  │
│                                                                       │
│   ┌─────────────────┐    ┌─────────────────┐                         │
│   │   Deploy Poll   │    │     Legacy      │                         │
│   │    Endpoint     │    │    Executor     │                         │
│   ├─────────────────┤    ├─────────────────┤                         │
│   │ check_deploy_   │    │ generate_wrapper│ (deprecated)            │
│   │ status          │    │ build_platform  │                         │
│   │                 │    │ cancel_build    │                         │
│   │                 │    │ list_builds     │                         │
│   └─────────────────┘    └─────────────────┘                         │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

### Tool Categories

| Category | Tools | Purpose |
|----------|-------|---------|
| **Pipeline** | `run_pipeline`, `check_pipeline_status`, `cancel_pipeline`, `resume_pipeline`, `list_pipelines` | Multi-stage deployment orchestration |
| **Signing** | `configure_signing`, `sign_application`, `verify_signature`, `get_signing_status`, `discover_certificates` | Code signing for production distribution |
| **Inspection** | `check_build_status`, `list_generated_wrappers`, `validate_configuration`, `get_system_prerequisites`, `check_deploy_status` | Build/system inspection and deploy status polling |
| **Legacy** | `generate_desktop_wrapper`, `build_for_platform`, `cancel_build`, `list_builds` | Deprecated (use pipeline tools) |

---

## Data Flow Through Stages

Each stage receives a `StageInput` and produces a `StageResult`. Results accumulate as the pipeline progresses.

```
┌───────────────────────────────────────────────────────────────────────┐
│                           StageInput                                  │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  Config ─────────┬──────────────────────────────────────────────────▶ │
│  PipelineID      │                                                    │
│  ScenarioPath    │                                                    │
│                  ▼                                                    │
│           ┌──────────┐     BundleResult                               │
│           │  Bundle  │────────────────┐                               │
│           └──────────┘                │                               │
│                  ▼                    ▼                               │
│           ┌──────────┐     PreflightResult                            │
│           │Preflight │────────────────┐                               │
│           └──────────┘                │                               │
│                  ▼                    ▼                               │
│           ┌──────────┐     GenerationResult (DesktopPath)             │
│           │ Generate │────────────────┐                               │
│           └──────────┘                │                               │
│                  ▼                    ▼                               │
│           ┌──────────┐     BuildResult (per platform)                 │
│           │  Build   │────────────────┐                               │
│           └──────────┘                │                               │
│                  ▼                    ▼                               │
│           ┌──────────┐     SmokeTestResult                            │
│           │SmokeTest │────────────────┐                               │
│           └──────────┘                │                               │
│                  ▼                    ▼                               │
│           ┌──────────┐     DeployResult                               │
│           │  Deploy  │─────────────────▶ Final artifacts uploaded     │
│           └──────────┘                                                │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

### StageInput Structure

```go
type StageInput struct {
    Config              *Config             // User's pipeline configuration
    PipelineID          string              // Current pipeline ID
    ScenarioPath        string              // Path to scenario directory
    DesktopPath         string              // Generated wrapper location
    BundleResult        *BundleResult       // Output from bundle stage
    PreflightResult     *PreflightResult    // Validation results
    GenerationResult    *GenerationResult   // Generated wrapper code
    BuildResult         *BuildResult        // Compiled binaries
    SmokeTestResult     *SmokeTestResult    // Test results
    DeployResult        *DeployResult       // Deploy/upload results
    ScenarioMetadata    *ScenarioMetadata   // Analyzed scenario info
}
```

---

## Adapter Pattern

Adapters bridge domain boundaries to avoid data duplication and maintain clean interfaces.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Domain Boundaries                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐      ┌──────────────────┐      ┌─────────────────┐    │
│  │   build     │◀────▶│  buildStore      │◀────▶│    system       │    │
│  │  .Store     │      │   Adapter        │      │   .BuildStore   │    │
│  └─────────────┘      └──────────────────┘      └─────────────────┘    │
│                                                                         │
│  ┌─────────────┐      ┌──────────────────┐      ┌─────────────────┐    │
│  │  pipeline   │◀────▶│ pipelineStore    │◀────▶│    tasks        │    │
│  │Orchestrator │      │   Adapter        │      │ .PipelineStore  │    │
│  └─────────────┘      └──────────────────┘      └─────────────────┘    │
│                                                                         │
│  ┌─────────────┐      ┌──────────────────┐      ┌─────────────────┐    │
│  │  records    │◀────▶│ recordStore      │◀────▶│   generation    │    │
│  │ .FileStore  │      │   Adapter        │      │  .RecordStore   │    │
│  └─────────────┘      └──────────────────┘      └─────────────────┘    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Adapter Types

| Adapter | Source | Target | Purpose |
|---------|--------|--------|---------|
| `systemBuildStoreAdapter` | `build.Store` | `system.BuildStore` | Build status for system handlers |
| `pipelineStoreAdapter` | `pipeline.Orchestrator` | `tasks.PipelineStore` | Pipeline state for task orchestration |
| `generationBuildStoreAdapter` | `build.InMemoryStore` | `generation.BuildStore` | Build state for generation service |
| `toolBuildStoreAdapter` | `build.InMemoryStore` | `toolexecution.BuildStore` | Build state for tool execution |
| `generationRecordStoreAdapter` | `records.FileStore` | `generation.RecordStore` | Record access for generation |
| `scenarioRecordStoreAdapter` | `records.FileStore` | `scenario.RecordStore` | Record access for scenario handlers |

---

## HTTP Endpoints

### Pipeline Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/pipeline/run` | POST | Start new pipeline (async or `?block=true&timeout=600`) |
| `/api/v1/pipeline/{id}` | GET | Get pipeline status (`?verbose=true` for details) |
| `/api/v1/pipeline/{id}/cancel` | POST | Cancel running pipeline |
| `/api/v1/pipeline/{id}/resume` | POST | Resume stopped pipeline |
| `/api/v1/pipelines` | GET | List all pipelines |

### Scenario-Based Pipeline Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/scenarios/{name}/pipeline/active` | GET | Get/create active pipeline for scenario |
| `/api/v1/scenarios/{name}/pipeline` | POST | Create new pipeline |
| `/api/v1/scenarios/{name}/pipeline/start` | POST | Start active pipeline |
| `/api/v1/scenarios/{name}/pipeline/reset` | POST | Clear active pipeline |
| `/api/v1/scenarios/{name}/pipeline/history` | GET | Get historical pipelines |

### Tool Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/tools` | GET | Discover all available tools |
| `/api/v1/tools/{name}` | GET | Get specific tool metadata |
| `/api/v1/tools/execute` | POST | Execute a tool by name |

---

## Request Flow Examples

### Starting a Pipeline via HTTP

```
POST /api/v1/pipeline/run
  │
  ▼
Handler.handleRun() [pipeline/handler.go]
  │
  ▼
Orchestrator.RunPipeline(ctx, config)
  │
  ▼ [Async] Start background goroutine:
runPipelineAsync(ctx, pipelineID, config)
  │
  ├─▶ For each stage in order:
  │   ├─ Fetch StageInput from store/parent pipeline
  │   ├─ Call Stage.Execute(ctx, input)
  │   ├─ Save StageResult to Status
  │   └─ Update progress, check stop conditions
  │
  ▼
Status persisted to Store
  │
  ▼ (HTTP response returns immediately)
Return: { "pipeline_id": "...", "status_url": "/api/v1/pipeline/..." }
```

### Executing a Tool

```
POST /api/v1/tools/execute
  Body: { "tool_name": "run_pipeline", "arguments": { "scenario_name": "..." } }
  │
  ▼
Handler.Execute() [toolexecution/handlers.go]
  │
  ▼
ServerExecutor.Execute(ctx, toolName, args)
  │
  ├─▶ Switch on toolName:
  │   ├─ "run_pipeline" → PipelineExecutor.RunPipeline(args)
  │   ├─ "sign_application" → SigningExecutor.SignApplication(args)
  │   ├─ "upload_artifact" → DistributionExecutor.UploadArtifact(args)
  │   └─ etc.
  │
  ▼
Domain executor calls appropriate service
  │
  ▼
Response: { "success": true, "result": {...}, "is_async": true/false }
```

### Resuming a Pipeline

```
POST /api/v1/pipeline/{id}/resume
  │
  ▼
Orchestrator.ResumePipeline(pipelineID, config)
  │
  ▼ [Load parent pipeline state]
  ├─ Get parent Status from store
  ├─ Extract StageInput from ResumedInput (carries forward results)
  └─ Determine next stage from StopAfterStage
  │
  ▼ [Create new child pipeline]
  ├─ Generate new pipelineID
  ├─ Set ParentPipelineID and ResumeFromStage
  └─ Save to store
  │
  ▼ [Run remaining stages asynchronously]
  Starting from ResumeFromStage through final stage
```

---

## State Persistence

### Store Hierarchy

| Store Type | Implementation | Persistence |
|------------|---------------|-------------|
| **In-Memory Store** | `NewInMemoryStore()` | Lost on restart |
| **File Store** | `NewFileStore(dataDir)` | Persists to `data/pipelines/` |
| **Index Store** | `ScenarioIndexStore` | Maps scenario names to active pipeline IDs |
| **Investigation Store** | (optional) | Persists task investigations |

### Pipeline Status Structure

```go
type Status struct {
    PipelineID      string
    State           string                    // idle, pending, running, completed, failed, cancelled
    Stages          map[string]*StageResult   // Results indexed by stage name
    CurrentStage    string                    // Currently executing stage
    ProgressPercent int                       // 0-100, computed from complete stages
    ProgressMessage string                    // Human-readable status
    IdempotencyKey  string                    // For request deduplication
    ParentPipelineID string                   // If resumed from another pipeline
    ResumeFromStage  string                   // Stage where resume started
}
```

---

## Key Design Patterns

1. **Screaming Architecture**: File names express domain purpose (`build_compiler.go`, `platform.go`)
2. **Options Pattern**: Builders use functional options for flexible configuration
3. **Adapter Pattern**: Clean translation between domain boundaries
4. **Store Pattern**: Pluggable storage (in-memory, file-backed)
5. **Stage Pipeline**: Composable, independently executable stages
6. **Tool Provider Pattern**: Extensible tool registration system
7. **Context Cancellation**: Goroutines respect `context.Context` for graceful shutdown
8. **Idempotency Keys**: Safe retries via client-provided deduplication keys

---

## Related Documentation

- [SEAMS.md](../internal/SEAMS.md) - Detailed seam definitions and testability patterns
- [Smoke Test Pipeline](./smoke-test-pipeline.md) - Deep dive into smoke test stage execution
- [Pipeline Interfaces](../../api/pipeline/interfaces.go) - Go interface definitions
- [Tool Execution](../../api/toolexecution/executor.go) - Tool executor implementations

---

## Code References

Key implementation files for this architecture:

- [CODE: api/main.go] - Server initialization and route registration
- [CODE: api/pipeline/orchestrator.go] - Pipeline orchestrator implementation
- [CODE: api/pipeline/interfaces.go] - Stage, Store, and Orchestrator interfaces
- [CODE: api/pipeline/handler.go] - HTTP handlers for pipeline endpoints
- [CODE: api/toolexecution/executor.go] - Tool execution dispatcher
- [CODE: api/adapters.go] - Domain boundary adapters
