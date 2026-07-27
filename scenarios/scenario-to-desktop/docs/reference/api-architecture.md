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
└──────────────────────────────────────┼──────────────────────────────────────┘
                                       │
                                       ▼
```

The server follows a **screaming architecture** pattern where each domain owns its handler. The central server struct orchestrates domain handlers for build, telemetry, records, scenario, system, pipeline, state, and deploy-target management.

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

### Adapter Types

| Adapter | Source | Target | Purpose |
|---------|--------|--------|---------|
| `systemBuildStoreAdapter` | `build.Store` | `system.BuildStore` | Build status for system handlers |
| `pipelineStoreAdapter` | `pipeline.Orchestrator` | `tasks.PipelineStore` | Pipeline state for task orchestration |
| `generationBuildStoreAdapter` | `build.InMemoryStore` | `generation.BuildStore` | Build state for generation service |
| `generationRecordStoreAdapter` | `records.FileStore` | `generation.RecordStore` | Record access for generation |
| `scenarioRecordStoreAdapter` | `records.FileStore` | `scenario.RecordStore` | Record access for scenario handlers |

---

## Pipeline API Surface

Pipeline orchestration is served only through the generated
`PipelineService` Connect contract. The generated UI and CLI clients expose
`Run`, `Get`, `Resume`, `Cancel`, `List`, `GetActive`, `CreateActive`,
`ResetActive`, `GetHistory`, `StartActive`, and `CleanBundle`. A caller polls a
long-running run by calling `Get(pipeline_id)`; no response contains a
hand-authored REST status URL.

## Request Flow Examples

### Starting a Pipeline via Connect

```
PipelineService.Run(PipelineRunRequest)
  │
  ▼
ConnectService.Run() [pipeline/connect_handler.go]
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
  ▼ (Connect response returns immediately)
Return: { "pipeline_id": "..." }
```

### Resuming a Pipeline

```
PipelineService.Resume(PipelineResumeRequest)
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

## Live Desktop Control Endpoint

The live desktop system exposes a unified control endpoint for executing desktop actions against active VNC sessions.

### Control Endpoint

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/livedesktop/sessions/{id}/control` | Execute a control action |
| `GET` | `/api/v1/livedesktop/sessions/{id}/files/{filename}` | Serve captured files (screenshots, recordings) |

### Control Request Format

```json
{
  "action": "screenshot",
  "params": {}
}
```

### Available Actions

| Action | Category | Description |
|--------|----------|-------------|
| `launch_app` | App | Launch the scenario's app on the display |
| `quit_app` | App | Kill the running app |
| `screenshot` | App | Capture PNG screenshot |
| `start_recording` | App | Begin screen recording |
| `stop_recording` | App | Stop recording, return video URL |
| `offline_mode` | Environment | Toggle network isolation via `unshare --net` |
| `slow_connection` | Environment | Bandwidth throttling via `tc` |
| `inject_env` | Environment | Set environment variables for next app launch |
| `resize_display` | Environment | Change display resolution via `xrandr` |
| `clipboard_read` | Advanced | Read clipboard (requires xclip) |
| `clipboard_write` | Advanced | Write to clipboard (requires xclip) |
| `dark_mode` | Advanced | Toggle GTK dark theme and Electron `--force-dark-mode` |
| `locale` | Advanced | Set locale (LANG, LC_ALL) |

### Architecture

Actions implement the `ActionExecutor` interface and are registered in an action registry. Shell commands go through an injectable `ShellFunc` seam for testability. See [SEAMS.md](../internal/SEAMS.md) for details.

---

## Key Design Patterns

1. **Screaming Architecture**: File names express domain purpose (`build_compiler.go`, `platform.go`)
2. **Options Pattern**: Builders use functional options for flexible configuration
3. **Adapter Pattern**: Clean translation between domain boundaries
4. **Store Pattern**: Pluggable storage (in-memory, file-backed)
5. **Stage Pipeline**: Composable, independently executable stages
6. **Context Cancellation**: Goroutines respect `context.Context` for graceful shutdown
7. **Idempotency Keys**: Safe retries via client-provided deduplication keys

---

## Related Documentation

- [SEAMS.md](../internal/SEAMS.md) - Detailed seam definitions and testability patterns
- [Smoke Test Pipeline](./smoke-test-pipeline.md) - Deep dive into smoke test stage execution
- [Pipeline Interfaces](../../api/pipeline/interfaces.go) - Go interface definitions

---

## Code References

Key implementation files for this architecture:

- [CODE: api/main.go] - Server initialization and route registration
- [CODE: api/pipeline/orchestrator.go] - Pipeline orchestrator implementation
- [CODE: api/pipeline/interfaces.go] - Stage, Store, and Orchestrator interfaces
- [CODE: api/pipeline/handler.go] - HTTP handlers for pipeline endpoints
- [CODE: api/adapters.go] - Domain boundary adapters
