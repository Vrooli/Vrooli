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
- Manage a long-lived Browserless session per execution (`/chrome/function` + context reuse) to reduce cold starts and retain state between steps.
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
- ✅ (2025-11-08) Defined replay JSON schema via `BuildReplayMovieSpec` and `/executions/{id}/export`, supplying transition hints, theme presets, and asset manifests for UI/CLI consumers.
   - Provide CLI renderer (Node + ffmpeg) to export marketing reels.

5. **Extension & Recording (P2)**
   - Align Chrome extension recording format with `execution_artifacts` schema to reuse renderer.
   - Support ingestion endpoints that accept external recordings, run validations, and populate the same tables.

### Chrome Extension Capture Ingestion

**Goal:** Let the forthcoming Chrome extension submit user-driven recordings that can be replayed with the same renderer as automated executions.

**Capture contract (extension side):**
- Manifest (`manifest.json`) is required and must include:
  - `runId`: stable identifier for the capture session (string)
  - `recordedAt`: ISO-8601 timestamp for the first frame (optional)
  - `viewport`: `{ "width": <number>, "height": <number> }`
  - `frames`: ordered array of frame objects (minimum length = 1)
- Each frame object supports:
  - `index`, `timestamp`/`durationMs`, `event`, `stepType`, `title`, `url`
  - `screenshot`: relative path inside the archive (PNG/JPEG/WebP)
  - Optional telemetry: `cursorTrail`, `cursor.path`, `clickPosition`, `focusedElement`, `highlightRegions`, `maskRegions`, `consoleLogs`, `network`, `assertion`, `domSnapshotHtml`
- Example frame payload:
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
- Assets (PNG frames, optional HAR snippets) must be bundled with the manifest in a zip archive using deterministic filenames (`frames/0001.png`, `frames/0002.png`, ...).
- The capture client POSTs the archive to the scenario API once recording stops; client-side retries are recommended.
- Operators can trigger imports manually via `browser-automation-studio recording import <archive.zip>` or through the UI **Import Recording** button on the project dashboard.

**API ingestion flow:**
1. New endpoint `POST /api/v1/recordings/import` accepts multipart or raw zip payloads.
2. `RecordingService` validates manifest schema, normalises timestamps, and assigns an internal `execution_id`/`project_id` (either supplied or defaulted to the demo project).
3. Assets are persisted under `data/recordings/<execution-id>/frames/####.png` (and uploaded to MinIO when configured) and referenced in `execution_artifacts` with `artifact_type = 'screenshot'` while timeline frames embed the same metadata with `origin = 'extension'`.
4. Each manifest frame becomes an `execution_step` with `step_type = 'recording_frame'` and `output.retry.history = []`, ensuring replay tooling can reuse cursor trails, highlight regions, and viewport data without special cases.
5. Optional console/network attachments are persisted as dedicated artifacts when present so the UI telemetry tab mirrors automated runs.
6. Once ingest completes, the API emits `execution.imported` over WebSocket so the UI can surface the new recording instantly.

**Data alignment considerations:**
- Reuse the existing replay export builder (`BuildReplayMovieSpec`) by pointing it at the ingested steps; the exporter only needs to understand `origin` to toggle cursor animation presets (real vs synthetic).
- Maintain a manifest hash to de-duplicate uploads and allow future delta sync from the extension.
- Flag imported runs in `executions.trigger_type = 'extension'` so analytics can filter by origin.

**Security & validation:**
- Enforce `maxFrames`, `maxTotalBytes`, and per-frame resolution limits before accepting the archive.
- Validate that all referenced asset paths exist inside the archive to avoid directory traversal.
- Optionally scan PNG payloads for metadata stripping to prevent leaking user PII.
- UI/CLI validation surfaces API errors (HTTP 4xx/5xx) so operators receive actionable feedback when uploads fail.

## Loop-Aware Execution & Branch Semantics (Design Draft)

### Drivers
- Workflow authors need to express repetition (paginate, poll, iterate over selector matches) without copy/pasting node chains.
- Conditional flows already exist via success/failure edges, but complex branching (multi-way conditions, loop guards, break/continue) remains manual and error-prone.
- Telemetry/replay consumers must understand control-flow context to provide accurate analytics and marketing visuals (group iterations, show loop progress, etc.).

### Step & Graph Extensions
- Introduce `loop` node type with labelled ports:
  - `body` (required) → first node executed each iteration.
  - `after` (optional) → node executed after loop completion.
  - `empty` (optional) → node executed if the loop never runs (collection empty / condition false).
- Loop node data payload:
  ```json
  {
    "mode": "times" | "while" | "doWhile" | "forEach",
    "maxIterations": 20,
    "condition": "await page.$('#ready') !== null",
    "initial": "let index = 0;",
    "afterEach": "index++",
    "collection": "return Array.from(await page.$$('.product')).map(e => e.textContent.trim());",
    "iterationAlias": "product"
  }
  ```
- UI enforces scoping by drawing a container around nodes reachable from `body` before a `loop-end` virtual node. Edges leaving the scope must terminate on `after`, `empty`, or `loop-end` handles. `break` and `continue` palette entries create explicit control handles inside the scope.

### Compiler Representation
- Extend `ExecutionStep` with optional `LoopSpec`:
  ```go
  type LoopSpec struct {
      Mode           string
      Condition      string
      CollectionExpr string
      MaxIterations  int
      IterationAlias string
      BodyPlan       *ExecutionPlan
      EmptyEdge      *EdgeRef
  }
  ```
- The planner recognises loop nodes, extracts the scoped subgraph, and recursively compiles it into `BodyPlan` while maintaining global DAG ordering (loops remain single steps in the top-level plan).
- Add compile-time validations: `body` edge required, `maxIterations` > 0 (unless explicit), detect illegal edges (escaping the scope), ensure nested loops don’t exceed configured depth, and statically reject recursion into the same workflow.
- Condition nodes remain normal `ExecutionStep`s; branch conditions are expressed via `edge.Condition` (`equals:true`, `equals:false`, `case:A`, ...).

### Runtime Semantics
1. Resolve iteration source:
   - `times`: iterate `0..maxIterations-1` (optionally parameterised by expression).
   - `forEach`: evaluate `CollectionExpr` inside Browserless; collection items become iteration context (`ctx.loop.item`).
   - `while` / `doWhile`: run `Condition` JS between iterations. Condition scripts execute in a sandbox with whitelisted globals (`page`, `frame`, `iteration`, `contextData`).
2. For each iteration:
   - Emit `step.loop_iteration_started` with iteration index, max, and optional item preview (first 256 chars of stringified item).
   - Execute `BodyPlan` using the existing executor (`Client.ExecutePlan` refactoring) so body steps reuse session + telemetry plumbing. Loop metadata travels via context to tag logs/artifacts (e.g., `artifact.Metadata["loopIteration"] = n`).
   - Support `break` / `continue` by watching for special `runtime.StepResult.Control` values bubbled from body steps.
   - Honour `maxIterations`: if exceeded without satisfying guard, mark loop step failed with `error = "loop_guard_exceeded"` unless `AllowOverflow` flag true.
3. After completion, emit `step.loop_completed` summarising iteration count, exit reason, last condition value. If zero iterations and `EmptyEdge` is defined, continue via that edge.

### Telemetry & Replay Enhancements
- `timeline_frame.loop` payload (new optional object):
  ```json
  {
    "nodeId": "loop-products",
    "iteration": 2,
    "total": 5,
    "mode": "forEach",
    "itemPreview": { "name": "Premium Plan" }
  }
  ```
- CLI: group output by loop (indent body steps, display iteration progress, highlight guard trips).
- UI Replay: collapsible sections per loop with chips for iteration numbers, gracefully skipping repeated screenshots when unchanged (diff heuristics optional).
- Exporter: extend `ExportFrame` with `LoopContext` (iteration, total, mode, alias preview). Renderer adds timeline pills and iteration counters.

### Incremental Delivery
1. **Compiler foundations**: extend schema, add AST validation, migrate DB (`execution_steps.loop_mode`, `loop_iteration` columns, JSONB metadata).
2. **Runtime skeleton**: implement `times` mode (deterministic), including telemetry + exporter adjustments. Ship behind feature flag (`BAS_LOOP_EXECUTION=1`).
3. **forEach & condition evaluation**: sandbox JS evaluation (same environment as screenshot overlays) and stream item previews.
4. **UI/CLI updates**: scoped editing UI, iteration-aware execution viewer, replay adjustments, CLI indentation.
5. **Advanced controls**: nested loops, `break`/`continue` nodes, `doWhile`, analytics aggregation (average duration per iteration, guard usage).

Adopting hierarchical plans preserves our acyclic compiler guarantees, keeps telemetry consistent, and unlocks richer validation/testing (AI agents can assert iteration counts). This structure also dovetails with the replay/video export roadmap—loop context feeds directly into captions and motion design.

## Next Actions
- Break down `browserless.Client` refactor into incremental PRs (compiler, runtime/session manager, executors, telemetry).
- Draft database migration for `execution_steps`/`execution_artifacts` (postgreSQL) and update repository interface (`api/database`).
- Prototype native WebSocket client in UI to validate the event contract before removing socket.io.
- Coordinate with `resource-browserless` maintainers to source reusable JS helpers for highlighting/annotations.
- Update `docs/action-plan.md` as milestones land.
