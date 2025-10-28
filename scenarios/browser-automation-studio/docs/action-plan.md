# Browser Automation Studio Action Plan

## 1. Original Request Context
- **User Prompt (abridged):** Understand how `resource-browserless` works, validate assumptions about screenshots/logs/workflows, and determine what is missing to fully realize the browser-automation-studio vision (automation builder, customizable screenshots, replay engine, Chrome extension). Need comprehensive current-state assessment, documentation updates, and better progress tracking beyond PRD checklists. Desire to programmatically link requirements to tests, using browser automations for UI validation.

## 2. Objectives
1. Provide an accurate picture of current `resource-browserless` capabilities and boundaries.
2. Map browser-automation-studio's implemented vs. promised features across API, UI, CLI, storage, and testing.
3. Define the path to deliver:
   - End-to-end browser automations (click/type/conditions/screenshots/console/network data).
   - Customizable screenshot + annotation support for both validation and marketing-style replays.
   - Replay/recording tooling that stitches executions into stylized experiences.
   - Chrome extension integration that reuses replay logic.
4. Establish durable documentation and requirement-tracking structures aligned with Vrooli's current testing infrastructure (phase-based tests consumed by `test-genie`).

## 3. Current Findings Snapshot
- **Browserless Resource:** CLI + workflow interpreter cover navigation, element actions, screenshots (page/element/full) and console capture with page/network/dialog telemetry. Generic workflows still expose `browser::get_console_logs` as a stub, so rich logging only exists when we call the console helper directly. No native utilities for overlays, highlights, cursor animation, or stitched replays.
- **Scenario Implementation:** `api/browserless/client.go` now ships generated scripts to Browserless' `/chrome/function`, persists progress updates, and stores real screenshots, but it only supports sequential `navigate`/`wait`/`screenshot` nodes. React Flow edges are ignored, click/type/extract nodes fail fast, and no console or network telemetry is captured. The WebSocket hub emits structured events consumed by the UI; CLI telemetry remains absent.
- **Testing:** Phase scripts live under `test/phases/`, but coverage stalls around 5–8%; handlers, storage, websocket, and end-to-end execution paths have no tests. The legacy `scenario-test.yaml` was retired (2025-10-20) so the phased suite is now the single source of truth.
- **Documentation & Tracking:** README/PRD continue to market finished capabilities (live execution, replay, AI debugging) without highlighting gaps. Requirements are not traceable to tests or automations, so completion scoring is subjective.

## 4. Key Gaps & Risks
- **Execution Engine Gap:** Visual workflows do not translate into Browserless executions. Without a real runner we cannot capture screenshots, DOM state, or console data per step.
- **Replay Artifact Gap:** We lack a normalized artifact schema (timeline, screenshots, DOM snapshots, cursor positions) to power validation replays or marketing-quality animations.
- **Screenshot Customization Gap:** Browserless has no helper to highlight/zoom/focus target elements before capture; replay UX demands these overlays.
- **Real-time Telemetry Gap:** API broadcasts nothing over WebSockets; UI/CLI assume socket.io events and break without live updates.
- **Requirements Traceability Gap:** PRD checkboxes are the only completion signal. No structured mapping to design-level requirements, tests, or automations.
- **Documentation Debt:** Public docs overstate maturity, increasing downstream confusion and risk of contract drift.

## 5. Action Items (Backlog)
| Priority | Area | Task | Deliverable |
|----------|------|------|------------|
| P0 | Documentation | Update `README.md` and `PRD.md` with "Current Limitations", execution status, and link to this plan. | Accurate public docs preventing feature assumptions. |
| P0 | Execution Core | Replace `api/browserless/client.go` stubs with a Browserless orchestration layer that: (1) translates React Flow graphs into ordered steps, (2) invokes Browserless (CLI or function API) using a persistent session, (3) captures per-step artifacts (screenshots, DOM extracts, console logs, network events), (4) stores artifacts in MinIO/Postgres, and (5) emits structured websocket events. | Production-ready executor powering all automations. |
| P0 | Telemetry Infrastructure | Align WebSocket implementation and UI expectations. Choose protocol (native websocket vs socket.io), implement server-side broadcasting using the new executor events, and update UI/CLI wiring. | Live progress/log streaming across UI & CLI. |
| P0 | Testing Hygiene | Legacy `scenario-test.yaml` retired (2025-10-20); next ensure phase scripts stay authoritative and add foundational unit tests for handlers/storage/websocket once the executor lands. | `test-genie` compatible tests with improved coverage baseline. |
| P1 | Replay Artifact Schema | Define `ExecutionArtifact` schema (timeline, steps, metadata, screenshot references, cursor path, viewport). Persist alongside executions, expose via API, and document contract. | Stable artifact format for replay & validation. |
| P1 | Replay Renderer | Build UI module + (optional) Node rendering utility that consumes the artifact schema to produce interactive replays with fades/zoom/cursor animation. | Executable replays for validation + marketing. |
| P1 | Screenshot Customization | Deliver Browserless helpers (JS modules or CLI flags) for element highlighting, focal zooms, background dimming, and viewport presets. Integrate with executor so workflow nodes can request specific treatments. | Configurable captures supporting validation + storytelling. |
| P1 | Requirements Tracking | Introduce `docs/requirements.yaml` mapping PRD → design-level requirements → validation assets. Implement tool/script to compute coverage by joining requirement entries with test/automation results. | Objective completion scoring + visibility. |
| P2 | Chrome Extension Integration | Adapt scenario-to-extension pipeline to record user sessions into the same artifact schema and reuse replay renderer inside the studio UI. | Unified automation + real-recording playback path. |
| P2 | Advanced Telemetry | Extend Browserless scripts to collect enriched diagnostics (HAR export, redacted response bodies, Lighthouse-style metrics) and surface them via artifacts and replay overlays. | Deep diagnostics for debugging & audits. |

> **Note:** P0 items unblock core execution and telemetry. P1 layers replay polish + automation validation. P2 focuses on expansion once foundations stabilize.

### Current Implementation Inventory (2025-10-18)
- **API – execution path**
  - `api/browserless/client.go` builds sequential instructions from workflow nodes and posts a generated script to Browserless' `/chrome/function`, persisting screenshots via MinIO/local fallback. Only `navigate`, `wait`, and `screenshot` nodes are supported today; edges/branching metadata are discarded and interaction nodes return unsupported errors.
  - `api/services/workflow_service.go` launches background executions and updates progress/result records but still emits only coarse `ExecutionUpdate` envelopes; step-level artifacts and telemetry never hit the WebSocket hub.
  - `api/handlers/handlers.go` shells out to `resource-browserless` for previews and element analysis per request, bypassing any reusable session or artifact pipeline.
- **WebSocket & clients**
  - `api/websocket/hub.go` exposes a native gorilla WebSocket hub broadcasting `ExecutionUpdate` payloads alongside structured events.
  - `ui/src/stores/executionStore.ts` now connects via native `WebSocket` and reacts to execution/step events; screenshot/log timelines still need richer payloads.
  - CLI `execution watch` (`cli/browser-automation-studio`) falls back to polling `/executions/{id}` every two seconds because streaming data is unavailable.
- **Artifact persistence**
  - Tables such as `screenshots`, `execution_logs`, and `extracted_data` now receive real PNG uploads and log entries for supported steps, but coverage stops at sequential screenshots.
  - There is still no schema or storage path for replay timelines, cursor metadata, highlight overlays, or network logs referenced in the PRD.
- **Workflow authoring**
  - React Flow JSON persists in Postgres and `api/browserless/client.go` derives sequential instructions from the saved `nodes`, ignoring `edges` entirely. Branching, ordering guarantees, and per-node validation remain unimplemented.
  - Execution viewer components (`ui/src/components/ExecutionViewer.tsx`) render screenshots/logs from the store, yet nothing ever populates these arrays beyond mocked CLI imports.
- **Testing footprint**
  - `test/phases/` scripts focus on linting/structure; there are no executor or handler unit tests, and integration coverage is effectively 0%.
  - `api/browserless/client_test.go` only asserts constructor behavior and health-check invocation—no real automation scenarios are exercised.


## 6. Execution Core Blueprint

1. **Graph Translation Layer**
   - Convert React Flow nodes into a deterministic execution plan (respecting edges, branching, loops).
   - Support node types: navigate, click, type, wait, screenshot, extract, sub-workflow, conditional.
   - Validate nodes before execution (required fields, selectors, prerequisites).

2. **Browserless Session Manager**
   - Maintain persistent sessions using Browserless function API (or stateful CLI helper) to reduce cold starts.
   - Provide helpers for navigation, clicks, typing, waiting, DOM evaluation, screenshot capture, console extraction.
   - Attach listeners for console/page errors and network failures before navigation begins.

3. **Artifact Capture**
   - For each step record: timestamp, inputs, results, screenshot references, console logs, network events, DOM snippets, error state.
   - Store binary artifacts in MinIO with deterministic keys (`executions/<id>/steps/<order>/`).
   - Insert metadata rows in Postgres (`screenshots`, `extracted_data`, upcoming `execution_artifacts`).
   - Capture pointer coordinates and viewport details for interaction nodes so replays can animate cursors accurately.
   - Normalize per-step request/response telemetry (URL, method, status, timing) alongside artifacts for downstream analytics.

4. **Telemetry Pipeline**
   - Emit websocket events (`progress`, `log`, `screenshot`, `artifact`, `status`) as steps execute.
   - Ensure WebSocket hub matches chosen protocol (if socket.io is kept, swap gorilla for compatible server).
   - Update CLI/UI to consume the unified event payloads.
   - Stream aggregated network diagnostics (e.g., request summaries, failure events) so clients can surface them in real time.

5. **Error Handling & Recovery**
   - Implement retry/backoff for transient Browserless failures.
   - Capture failure artifacts (final screenshot + console/network summary) before aborting.
   - Surface actionable error codes to UI/CLI.

## 7. Replay & Artifact Vision

1. **Artifact Schema**
   - `execution_artifacts` table (execution_id, step_index, type, payload JSON, screenshot_ids, cursor_path, viewport, highlight_regions, duration_ms).
   - JSON schema stored under `docs/artifacts/execution.json` to enforce structure, including optional highlight/zoom presets.

2. **Renderer Requirements**
   - Client-side: React component that animates artifacts, supports step scrubbing, cursor trails, highlight overlays, zoom transitions, background styling.
   - Server-side: optional CLI utility (`scenario replay render`) to export video/gif by stitching PNGs via ffmpeg with metadata (respecting highlight regions and timing).

3. **Automation Validation**
   - Add Browserless-driven integration tests that run replay-critical workflows and assert artifact completeness and schema conformity.
   - Document instructions for generating marketing reels automatically.

## 8. Requirements Tracking Strategy

1. ✅ Seed `docs/requirements.yaml` (v0.1, 2025-10-19) capturing top-level PRD items with placeholder validation entries and reporting metadata.
2. Tag tests/automations with `REQ:<id>` annotations (unit tests via comments, workflow YAML via metadata tags under `workflow.metadata.requirements`).
3. Ensure Browserless validation workflows emit machine-readable result summaries so requirement reports can ingest automation success/failure alongside phase outputs.
4. ✅ Introduced `scripts/requirements/report.js` skeleton that parses `docs/requirements.yaml` and emits JSON/Markdown summaries; next iteration should join tagged phase outputs + automation logs for full coverage.
5. Update README with a "Requirement Coverage" badge/table produced by the reporting script.

## 9. Testing Strategy Notes
- Keep `test/phases/` as the single entry point; legacy `scenario-test.yaml` removal is complete—documented retirement (2025-10-20) and follow-up unit tests remain.
- Add unit tests alongside new executor modules (browserless client, artifact store, websocket broadcaster).
- Introduce integration tests that run simplified workflows end-to-end against Browserless (gated behind `INTEGRATION=1`).
- Extend replay renderer with snapshot tests (Jest/Vitest) once implemented.
- Long term, schedule browserless-driven UI validation automations in the "integration" or "business" phases to exercise replay rendering and extension output.

## 10. Documentation & Tracking Plan
1. Maintain this action plan and update after every milestone.
2. Add "Status Dashboard" to README fed by the requirements coverage script.
3. Document execution architecture (`docs/architecture/execution.md`) outlining the layers above.
4. Provide developer how-to guides for new helpers (highlighted screenshots, replay tooling).
5. Link prominently to relevant resource docs (e.g., `resources/browserless/docs/`) so contributors can align scenario helpers with upstream capabilities.
6. Maintain onboarding notes for browserless-powered workflows, including references to scenario-specific helper scripts.

## 11. Immediate Next Steps
1. ✅ Update README/PRD + publish limitations note (2025-10-18).
2. ✅ Finalize execution core design doc and break into implementation tasks (see `docs/architecture/execution.md`, 2025-10-19).
3. ✅ Decide on WebSocket transport (retain native gorilla hub; publish event contract in `docs/architecture/execution.md`).
4. ✅ Draft `docs/requirements.yaml` skeleton + reporting script interface (reports via `scripts/requirements/report.js`, 2025-10-19).
5. ✅ Remove `scenario-test.yaml` once phase scripts + reporting are authoritative (2025-10-20).

*Last updated: 2025-10-19*
