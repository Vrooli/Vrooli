# Execution Architecture Blueprint

_Last reviewed: 2025-11-08_

## Context Snapshot
- `api/browserless/client.go` now executes `navigate`, `wait`, `click`, `type`, `extract`, and `screenshot` nodes against a persistent Browserless session. Each step persists to `execution_steps`/`execution_artifacts`, capturing console + network telemetry, element bounding boxes, click coordinates, and highlight/mask/zoom metadata. Success/failure/else branching and per-node retry/backoff policies ship; loop constructs remain outstanding.
- React Flow payloads store nodes and edges; the compiler normalises the DAG and preserves branching metadata for runtime evaluation, but loop constructs and richer compile-time validation are still pending.
- The WebSocket hub (`api/websocket/hub.go`) streams structured `execution.*`, `step.*`, and `step.heartbeat` events consumed by the UI replay panel and CLI watcher. Cursor overlays for the UI remain roadmap work, but the CLI now emits heartbeat health states and can trigger replay exports directly.
- Artifact persistence includes MinIO-backed screenshots, per-step telemetry bundles, cursor trails, replay-ready `timeline_frame` payloads, and JSON replay export packages. DOM snapshots and automated video rendering remain future milestones.

## Objectives
1. Execute arbitrarily complex workflows (branching, loops, conditions) against Browserless with first-class actions (navigate, click, type, evaluate, extract, assertions).
2. Capture rich telemetry (console, network, DOM snapshots, pointer coordinates, viewport) while the run is in-flight and stream it to all subscribers in real time.
3. Persist artifacts in a normalized schema that can power validation, analytics, and replay rendering (UI + automated video exports).
4. Offer a stable event contract for both Web UI and CLI consumers, using a single native WebSocket transport built on the existing gorilla hub.
5. Provide extension points for screenshot customization (element focus, highlight overlays, zooming) and future Chrome extension ingestion without reworking the executor.

## High-Level Flow
```
WorkflowService.ExecuteWorkflow ---> Workflow Graph Compiler ---> Execution Plan
           |                                                      |
           v                                                      v
   ExecutionRegistry (Postgres) <---- Session Manager ----> Browserless Function API
           |                                                      |
           v                                                      v
 WebSocket Hub <---- Telemetry Streamer ---- per-step events ----> Artifact Store (Postgres + MinIO)
```

## Component Breakdown

### 1. Workflow Graph Compiler
**Location:** `api/browserless/compiler` (new package)

Responsibilities:
- Parse `workflow.FlowDefinition` (React Flow JSON) into an ordered DAG respecting `edges`, including branch metadata (conditions, success/failure paths).
- Validate node configuration (required fields, supported actions, parameter types) before execution begins.
- Emit a normalized `ExecutionPlan` struct:
  ```go
  type ExecutionPlan struct {
      WorkflowID uuid.UUID
      Steps      []ExecutionStep
      Metadata   map[string]any // global defaults such as viewport
  }

  type ExecutionStep struct {
      Index        int
      NodeID       string
      Type         StepType // Navigate, Click, Type, Wait, Screenshot, Extract, Assert, Custom
      Params       map[string]any
      OutgoingEdges []EdgeRef // used for branching decisions
  }
  ```
- Support reusable sub-workflows by inlining referenced plans (with guard rails to prevent recursion loops).
- Provide static analysis helpers (`Validate(plan ExecutionPlan) error`) so the API can reject malformed workflows without touching Browserless.

### 2. Browserless Session Manager
**Location:** `api/browserless/runtime`

Responsibilities:
- Manage a long-lived Playwright session per execution (`browserless` function API + context reuse) to reduce cold starts and retain state between steps.
- Attach instrumentation before navigation:
  - `page.on('console', ...)` → streamed as console events.
  - `page.on('request')`, `page.on('response')`, `page.on('requestfailed')` → network events.
  - `page.on('pageerror')`, `browser.on('disconnected')` → error telemetry.
- Expose high-level primitives such as `Navigate`, `Click`, `Type`, `WaitFor`, `Screenshot`, `Evaluate`, `CollectElements` implemented via embedded JS modules shipped with the scenario (leveraging `resource-browserless/lib/workflow` helpers where possible).
- Provide cancellation support (`context.CancelFunc`) so `StopExecution` can gracefully terminate the Browserless job and emit a terminal artifact.

### 3. Step Executors
**Location:** `api/browserless/runtime/steps`

Each `StepType` maps to an executor implementing:
```go
type StepExecutor interface {
    Execute(ctx context.Context, session *Session, step ExecutionStep, emit EmitFunc) StepResult
}
```
Key actions:
- **Navigate**: `page.goto` with `waitUntil` and timeout control; emits navigation metrics and final URL.
- **Click**: `page.waitForSelector` + `click`. Capture cursor coordinates (target bounding box) and emit pointer trail for replay.
- **Type**: `click` + `fill`/`type`; emit text length, masking secrets (support `sensitive` flag in params).
- **Wait**: Support `time`, `selector`, `networkIdle`, `function`. Emit wait reason + elapsed time.
- **Screenshot**: Call helper respecting `fullPage`, `selector`, `padding`, `viewportPreset`, `highlightRegions`. Return both PNG and optional thumbnail.
- **Extract**: Run DOM evaluation scripts (CSS/XPath queries) returning structured JSON (arrays, key-value). Persist results to `extracted_data` table.
- **Assert**: Evaluate conditions (element exists, text equals, status code) and fail fast with captured artifact bundle (screenshot + console summary).

Each executor calls `emit(ExecutionEvent)` multiple times: start, progress updates, artifacts, completion.

### 4. Telemetry & Event Bus
**Transport Decision:** keep the existing gorilla WebSocket hub and refactor clients (UI + CLI) to use native WebSocket connections. Socket.IO adds another dependency in Go and duplicates work already solved by the hub.

**Event Envelope:**
```json
{
  "type": "step.screenshot",        // see table below
  "execution_id": "uuid",
  "workflow_id": "uuid",
  "step": {
    "index": 3,
    "node_id": "node-42",
    "type": "screenshot",
    "name": "Product Grid",
    "started_at": "2025-10-19T03:10:22.531Z"
  },
  "payload": { ... },                // type-specific data
  "timestamp": "2025-10-19T03:10:23.014Z"
}
```

| Event Type             | Payload Fields                                                                                   |
|------------------------|---------------------------------------------------------------------------------------------------|
| `execution.started`    | `plan_version`, `total_steps`, `viewport`, `parameters`                                           |
| `execution.progress`   | `step_index`, `percent`, `current_step`                                                          |
| `step.started`         | `step` metadata + `input` snapshot                                                                |
| `step.log`             | `level`, `message`, `meta` (e.g., selector, retries)                                             |
| `step.console`         | `severity`, `text`, `arguments` (stringified)                                                    |
| `step.network`         | `request_id`, `url`, `method`, `status`, `timings`, `transfer_size`, `failure_reason?`           |
| `step.heartbeat`       | `elapsed_ms`, `step_type`, `node_id`                                                             |
| `step.screenshot`      | `screenshot_id`, `url`, `thumbnail_url`, `viewport`, `highlights`, `cursor_path`                 |
| `step.extract`         | `extracted_data_id`, `schema`, `rows`, `sample`                                                  |
| `step.completed`       | `status` (`success`|`skipped`), `duration_ms`, `output_summary`                                   |
| `execution.failed`     | `error_code`, `message`, `stack`, `failing_step`                                                 |
| `execution.completed`  | `duration_ms`, `success_count`, `failure_count`, `artifact_manifest`                             |

The hub simply broadcasts the JSON; richer routing (per-execution subscriptions) remains via `Client.ExecutionID` filtering. Heartbeat cadence defaults to 2 s and can be tuned (or disabled with `0`) using the `BROWSERLESS_HEARTBEAT_INTERVAL` environment variable.

### 5. Artifact Persistence

| Table | Purpose | Notes |
|-------|---------|-------|
| `execution_steps` | One row per executed step (status, duration, input, output summary, error). | Enables deterministic replay + analytics. |
| `execution_artifacts` | Attachment metadata keyed by step (`type`, `media_url`, `payload JSONB`). | Cover screenshots, console dumps, HAR excerpts, cursor trails. |
| `execution_events` (optional) | Audit log of streamed events for debugging. | Keep short retention. |

File storage conventions (MinIO / local):
```
executions/<execution-id>/steps/<step-index>/
  screenshot.png
  screenshot.thumb.webp
  dom.json
  network.json
```

Screenshot customization pipeline:
- Baseline overlay injection now lives in `browserless/runtime/script.go`; extract reusable helpers into `resources/browserless/lib/workflow/highlight.js` so highlight/mask logic can be shared with future cursor/animation tooling.
- Step executor composes options from node params (`focus_selector`, `highlight_selectors`, `mask_selectors`, `zoom_factor`), renders overlays in the browser context, captures PNG, and returns overlay metadata for replays.

### 6. Error Handling & Cancellation
- Wrap each step in a retry policy (configurable per node) for transient Browserless/network hiccups.
- On failure, emit `step.failed` + `execution.failed`, persist a rescue artifact bundle (last screenshot, console transcript, network summary), and mark `execution_steps.status = failed`.
- Honour `StopExecution`: trigger context cancellation, wait for Browserless to settle, emit `execution.cancelled`, and persist partial progress.

### 7. CLI & UI Integration Notes
- **UI:** Replace `socket.io-client` usage with native `WebSocket`, subscribe to new event types, update stores to build filmstrip + log timeline from streamed payloads.
- **CLI:** `execution watch` listens to the same WebSocket and renders textual logs + heartbeat health, while `execution export` retrieves replay packages (use `--output` to persist the JSON for renderers).
- Provide a shared TypeScript schema (`ui/src/types/executionEvents.ts`) generated from Go structs via `quicktype` or manual definitions to keep payloads in sync.

## Implementation Phases

1. **Foundation (P0)**
   - Introduce `ExecutionPlan` compiler with validation; support sequential flows + deterministic ordering.
   - Refactor `browserless.Client` into session manager + step executors; migrate navigate/wait/screenshot to new architecture.
   - Emit new WebSocket events (`execution.started`, `step.started`, `step.completed`, `step.screenshot`, `execution.completed`, `execution.failed`).
   - Persist `execution_steps` + `execution_artifacts` tables with migrations; update repository interfaces.

2. **Interaction Support (P0)**
   - Implement `click`, `type`, `assert`, `extract` executors with telemetry capture (console/network/DOM snapshots).
   - Add cursor coordinate tracking and (DONE Oct-2025) highlight/mask metadata for screenshot steps.
   - Update UI/CLI stores to render streamed data in real time.

3. **Advanced Telemetry (P1)**
   - Normalize network logs (HAR fragments) and console output, surface via replay + API endpoints.
   - Add failure heuristics (auto-screenshot on error, aggregated error codes) for AI debugging loop.

4. **Replay & Rendering (P1)**
   - ✅ (2025-11-08) Defined replay JSON schema via `BuildReplayExport` and `/executions/{id}/export`, supplying transition hints, theme presets, and asset manifests for UI/CLI consumers.
   - Provide CLI renderer (Node + ffmpeg) to export marketing reels.

5. **Extension & Recording (P2)**
   - Align Chrome extension recording format with `execution_artifacts` schema to reuse renderer.
   - Support ingestion endpoints that accept external recordings, run validations, and populate the same tables.

### Chrome Extension Capture Ingestion

**Goal:** Let the forthcoming Chrome extension submit user-driven recordings that can be replayed with the same renderer as automated executions.

**Capture contract (extension side):**
- Background script records a sequence of `frame` objects:
  ```json
  {
    "runId": "uuid",
    "viewport": { "width": 1440, "height": 900 },
    "frames": [
      {
        "index": 0,
        "timestamp": 0,
        "event": "navigate",
        "url": "https://app.example.com",
        "screenshot": "frames/0000.png",
        "cursor": { "path": [[120, 340], [312, 410]] },
        "focus": { "selector": "#hero" },
        "annotations": { "highlights": [...], "masks": [...] },
        "console": [],
        "network": []
      }
    ]
  }
  ```
- Assets (PNG frames, optional HAR snippets) are bundled with the manifest in a zip archive using deterministic filenames.
- Extension posts the archive to the scenario API once capture stops; retries handled client-side.

**API ingestion flow:**
1. New endpoint `POST /api/v1/recordings/import` accepts multipart or raw zip payloads.
2. `RecordingService` validates manifest schema, normalises timestamps, and assigns an internal `execution_id`/`project_id` (either supplied or defaulted to the demo project).
3. Assets are streamed to MinIO under `recordings/<execution-id>/frames/####.png` and mirrored to `execution_artifacts` with `artifact_type = 'timeline_frame'` + `origin = 'extension'` metadata.
4. Each manifest frame becomes an `execution_step` with `step_type = 'recording_frame'` and `output.retry.history = []`, ensuring replay tooling can reuse cursor trails, highlight regions, and viewport data without special cases.
5. Optional console/network attachments are persisted as dedicated artifacts when present so the UI telemetry tab mirrors automated runs.
6. Once ingest completes, the API emits `execution.imported` over WebSocket so the UI can surface the new recording instantly.

**Data alignment considerations:**
- Reuse the existing replay export builder (`BuildReplayExport`) by pointing it at the ingested steps; the exporter only needs to understand `origin` to toggle cursor animation presets (real vs synthetic).
- Maintain a manifest hash to de-duplicate uploads and allow future delta sync from the extension.
- Flag imported runs in `executions.trigger_type = 'extension'` so analytics can filter by origin.

**Security & validation:**
- Enforce `maxFrames`, `maxTotalBytes`, and per-frame resolution limits before accepting the archive.
- Validate that all referenced asset paths exist inside the archive to avoid directory traversal.
- Optionally scan PNG payloads for metadata stripping to prevent leaking user PII.

## Next Actions
- Break down `browserless.Client` refactor into incremental PRs (compiler, runtime/session manager, executors, telemetry).
- Draft database migration for `execution_steps`/`execution_artifacts` (postgreSQL) and update repository interface (`api/database`).
- Prototype native WebSocket client in UI to validate the event contract before removing socket.io.
- Coordinate with `resource-browserless` maintainers to source reusable JS helpers for highlighting/annotations.
- Update `docs/action-plan.md` as milestones land.
