# Browser Automation Studio

Visual browser automation workflow builder with AI-powered generation and debugging.

## 🎯 Overview

Browser Automation Studio transforms browser automation from code-based scripts to visual, self-healing workflows. It provides a drag-and-drop interface for creating browser automation workflows, real-time execution monitoring with screenshots, and AI assistance for both generation and debugging.

## ⚠️ Current Implementation Status (2025-11-14)
- The automation executor/engine stack (`api/automation/{executor,engine,recorder,events}`) now drives Browserless via the `BrowserlessEngine`, emitting normalized `StepOutcome` payloads through `WSHubSink` and persisting through `DBRecorder`. It covers `navigate`, `wait`, `click`, `type`, `extract`, `loop`, and `screenshot` nodes with console/network telemetry, bounding boxes, click coordinates, cursor trails, extracted payloads, and focus/highlight/mask/zoom metadata recorded in `execution_steps` and `execution_artifacts` (timeline frames included). Success/failure/else branching, runtime loop execution (for-each, repeat, while), and per-node retry/backoff policies record attempt history alongside screenshots and telemetry.
- Assertion nodes now evaluate selector existence/text/attributes directly in Browserless, emitting structured assertion artifacts, timeline metadata, and CLI/UI logs that short-circuit executions on failure.
- Structured WebSocket events (`execution.*`, `step.*`) now include mid-step `step.heartbeat` telemetry. The UI panel surfaces the latest heartbeat with elapsed timing while console/network payloads continue streaming via `step.telemetry` events.
- The CLI watcher attaches to the WebSocket stream when Node.js is available, echoing heartbeats alongside step events and retaining HTTP polling + timeline summary fallbacks. The `execution export` command now streams replay-export packages and can write the JSON payload to disk for automation tooling.
- Replay tooling offers a Replay tab with highlight/mask overlays, zoom anchoring, animated cursor trails, and storyboard navigation, and the API now serves structured `/executions/{id}/export` packages with transition hints, theme presets, and asset references. DOM snapshots are captured alongside screenshots, surface in the UI replay inspector, and ship as embedded HTML in export bundles. The CLI now includes `execution render-video`, which asks the API’s Browserless renderer to capture each frame from the composer iframe and streams MP4/WEBM bundles back to disk, while richer motion presets remain roadmap work.
- The composer now sends its fully decorated `ReplayMovieSpec` to the export API, so Browserless captures exactly what the iframe shows while still supporting JSON exports and CLI automation.
- Chrome extension recordings can be ingested via `POST /api/v1/recordings/import`, which normalises manifest + frame archives into executions, timeline artifacts, and replay assets served from `/api/v1/recordings/assets/{executionID}/…`. Automated extension packaging remains to be productised, but imported runs now appear alongside Browserless executions.
- Requirements tracking continues through `requirements/index.json` (v0.2.0 modular registry) and `scripts/requirements/report.js`, now reflecting telemetry/replay progress; automated CI hooks remain pending.
- Automated coverage exercises the compiler/runtime helpers and executor telemetry persistence; WebSocket contract, handler integration, and end-to-end Browserless tests remain gaps.
- Documentation across README/PRD/action-plan matches the current executor and replay capabilities while calling out remaining milestones.

> See `docs/action-plan.md` for the full backlog and execution roadmap.

### Current Limitations
- Workflow sub-calls are still on the roadmap, but core looping (for-each, repeat, while) and conditional branching now keep complex flows DRY.
- The replay pipeline exports interactive HTML packages (`execution render`) and experimental MP4/WEBM bundles (`execution render-video`), but advanced cursor animations, zoom keyframes, and GIF export still require future polish.
- Chrome extension capture requires the import API—the extension packaging/publishing flow still needs to be integrated into the lifecycle UI/CLI.
- Requirements coverage now consumes phase result JSON for live pass/fail state, but automation outputs still need to be integrated.
- AI workflow generation/debugging endpoints exist but still require manual validation before they can be considered production-ready.

## 📊 Status Dashboard
- Requirement coverage (`requirements/index.json` v0.2.0): totals now align with the modular registry. Running `test/phases/*` emits JSON to `coverage/phase-results/` so the requirements reporter reflects live phase pass/fail states instead of static bookkeeping.
- Generate a fresh snapshot with `node ../../scripts/requirements/report.js --scenario browser-automation-studio --format markdown` from the scenario root.

### Workflow Validation
- Run `browser-automation-studio workflow lint test/playbooks/**/*.json --strict` to perform schema + lint checks via the API. Use `--json` for machine-readable output.
- Alternatively, run the Go CLI locally: `(cd api && go run ./cmd/workflow-lint --strict ../test/playbooks/**/*.json)` to lint without the Bash CLI
- The API exposes `POST /api/v1/workflows/validate` so the UI builder and external tooling can validate workflows without running them
- `./test/run-tests.sh` now invokes the validator before executing any phase; export `WORKFLOW_LINT_STRICT=1` to fail on warnings or leave it unset to treat warnings as advisory. Pass `WORKFLOW_LINT_API=0` to temporarily skip the API-based lint inside the integration phase.
- All scenarios automatically lint their `test/playbooks/**/*.json` files via the shared test runner. Set `WORKFLOW_LINT_PRECHECK=0` to skip the preflight entirely when debugging, or `WORKFLOW_LINT_STRICT=1` to promote warnings to errors.
- Selector references now resolve through `ui/src/consts/selectors.ts`. Use `@selector/<key>` tokens (or `@selector/<dynamicKey>(param=value)` for parameterized selectors like execution filters) in workflows instead of hardcoded `[data-testid="..."]`; the resolver expands them and the validator confirms every selector maps to a registered test id.

## ✨ Features

Status legend: ✅ scaffolding exists • 🚧 active development • 🌀 planned polish

- ✅ **Visual Workflow Builder**: React Flow UI stores node/edge JSON and project folders in Postgres.
- ✅ **API/CLI Scaffolding**: REST handlers and CLI commands exist for workflows/executions; the API executes sequential Browserless steps with per-step telemetry while the CLI opens a live WebSocket stream (when Node.js is available) and prints execution timeline summaries after runs.
- ✅ **Execution History Viewer**: Full-featured execution history viewer in the UI (Project Detail → Executions tab) with filtering by status (all/completed/failed/running), execution details, timeline replay, and refresh functionality. CLI provides `execution list` command for programmatic access.
- ✅ **Execution Timeline API**: `/api/v1/executions/{id}/timeline` now assembles per-step `timeline_frame` artifacts with screenshot metadata, highlights, masks, and console/network references for replay consumers.
- 🚧 **Real-time Execution Stream**: Structured WebSocket events deliver per-step logs, mid-step heartbeats, telemetry, and screenshots; the execution viewer now surfaces last-heartbeat timing while rendering `step.telemetry` console/network output. Tune heartbeat cadence with `BROWSERLESS_HEARTBEAT_INTERVAL` (Go duration, `0` disables). Cursor trails render inside the Replay tab; richer CLI overlays remain on the roadmap.
- 🚧 **AI Workflow Generation**: Prompt pipeline calls OpenRouter but lacks validation against a runnable executor.
- 🚧 **AI Debugging Loop**: Endpoint stubs exist; needs real telemetry + replay artifacts to be effective.
- 🚧 **Replay & Marketing Renderer**: Timeline artifacts feed the replay UI, `/executions/{id}/export` packages expose animation/transition hints (including DOM snapshot HTML), and the CLI offers `execution render` (HTML bundle) plus `execution render-video` (MP4/WEBM capture streamed through the API’s Browserless pipeline). Advanced motion presets and GIF exporters remain on the roadmap.

## 🚀 Quick Start

> Current executables handle sequential workflows, conditional branches (including continue-on-failure assertions), and loop constructs that iterate over datasets or while-conditions. Replay exporters are still in development, so treat this as plumbing validation while the roadmap lands.

### Prerequisites

```bash
# Ensure required resources are running
vrooli resource browserless start
vrooli resource postgres start
vrooli resource minio start
```

### Installation

```bash
# Install CLI
cd cli
./install.sh
source ~/.bashrc

# Setup database
cd ../api
go run cmd/migrate/main.go up

# Install UI dependencies
cd ../ui
npm install
```

### Running the Scenario

```bash
# Start the scenario
vrooli scenario run browser-automation-studio

# Or start components individually:
# Start API
cd api && go run main.go

# Start UI
cd ui && npm run dev
```

### Using the CLI

```bash
# Check status
browser-automation-studio status

# Create workflow from AI prompt
browser-automation-studio workflow create "test-flow" \
  --ai-prompt "Navigate to google.com and search for automation"

# Execute workflow
browser-automation-studio workflow execute "test-flow" --wait

# List workflows
browser-automation-studio workflow list

# Generate replay export metadata
browser-automation-studio execution export <execution-id> --output bas-export.json

# Produce a stylised HTML replay package (screenshots downloaded automatically)
browser-automation-studio execution render <execution-id> --output ./replays/demo-run
```

### Importing Chrome Extension Recordings

Browser extension captures (zip archive containing `manifest.json` plus frame assets) can be normalised into Browser Automation Studio via the new import endpoint.

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/recordings/import" \
  -F "file=@/path/to/extension-recording.zip" \
  -F "project_name=Demo Browser Automations"
```

- If no project is supplied, the importer tries `project_id`, then `project_name`, falling back to the seeded **Demo Browser Automations** project or creating **Extension Recordings** on demand.
- Assets are stored under `scenarios/browser-automation-studio/data/recordings/<execution-id>/frames/` and exposed via `/api/v1/recordings/assets/{executionID}/frames/{filename}` so the UI Replay tab and CLI renderer can fetch frames without touching MinIO.
- Imported runs appear in the dashboard alongside Browserless executions with `trigger_type = extension`, complete with replay timeline, console/network artifacts, and export support.

### Demo Workflow

On first run (or whenever the database is empty) the API seeds a ready-to-run workflow named **Demo: Capture Example.com Hero** inside the `/demo` workflow folder. The seed run creates a project called **Demo Browser Automations** whose backing directory defaults to `scenarios/browser-automation-studio/data/projects/demo` (override with `BAS_DEMO_PROJECT_PATH`). The directory is created automatically so replay exports and rendered bundles have a safe place to land.

- **From the UI:** Start the scenario, open the dashboard, and select **Demo Browser Automations**. The preloaded "Demo" folder contains the workflow—click **Run** to watch live telemetry. When the execution finishes, open the Replay tab to see the branded screenshots, cursor trail, and assertion summary.
- **From the CLI:**

  ```bash
  # Execute the demo workflow and wait for completion
  browser-automation-studio workflow execute "Demo: Capture Example.com Hero" --wait

  # Stream the replay/telemetry for the most recent run
  browser-automation-studio execution watch <execution-id>
  ```

  The `execute` command prints the execution ID so you can feed it directly into `execution watch`, `execution export`, or `execution render` to pull telemetry and marketing-ready assets.
- **Playbook sanity check:** `testing::playbooks::bas::run_workflow --file test/playbooks/projects/demo-sanity.json` boots the scenario and verifies the demo project/workflow are exposed via the API—handy for CI keep-alive jobs or local smoke checks.

## 🏗️ Architecture

### Components

1. **UI (React + Vite + TypeScript)**
   - React Flow for visual workflow building
   - WebSocket scaffolding for real-time updates (requires new executor events)
   - Zustand for state management
   - Tailwind CSS for styling

2. **API (Go + Chi)**
   - RESTful API for workflow management
   - Gorilla WebSocket hub broadcasting automation `EventEnvelope` payloads (legacy wrapper opt-in via `WS_LEGACY_UPDATES=true`)
   - PostgreSQL for persistence
   - MinIO for screenshot storage

3. **CLI (Bash)**
   - Thin wrapper around API with WebSocket-enabled execution watching
   - `execution export` command streams replay-export packages for downstream tooling
   - `execution render` command downloads assets and produces a standalone HTML replay for marketing previews
   - JSON and human-readable output

### Workflow Node Types

- **Navigate**: Executes via Browserless for sequential flows; edges/branching are ignored and each step shares a single page context.
- **Click**: Supported for visible selectors; emits bounding boxes and synthetic click coordinates and now honours node-level retry/backoff policies.
- **Hover**: Smoothly moves the pointer to any selector (no clicks) with configurable glide duration so dropdowns, tooltips, and mega-menus stay open for follow-up steps.
- **Scroll**: Supports page/element/position scrolls plus "scroll until visible" sweeps with retry limits for lazy-loaded content.
- **Select**: Targets native single or multi-select dropdowns, choosing options by value/text/index while firing the correct input/change events.
- **Type**: Supported with optional delay/clear/submit flags; richer form heuristics and masking are pending.
- **Screenshot**: Captures full-page PNGs (MinIO/local fallback) with optional focus, highlight, mask, background, and zoom controls while preserving cursor trails for replay animation.
- **Wait**: Time waits execute; element waits leverage configurable retry/backoff instrumentation, though richer loop constructs are still pending.
- **Extract**: Supported for text/value/html/attribute payloads and persists results (no schema enforcement yet).
- **Assert**: Evaluate selector existence, text, or attribute conditions directly in Browserless, surfacing assertion artifacts, execution logs, and replay metadata; failures halt the run unless future branching allows continuations.
- **Shortcut**: Emits high-level combinations such as `Ctrl+K` or `Cmd+Enter` using Browserless helpers—ideal for triggering global shortcuts or menu accelerators on the focused element.
- **Keyboard**: Dispatches low-level `keydown`/`keyup`/`keypress` events with optional modifier toggles and hold delays, enabling granular navigation (e.g., arrow keys, Enter on dialogs) without recording a full shortcut sequence.
- **Tab Switch**: Jumps between existing tabs, OAuth popups, or new windows by newest/oldest ordering, explicit index, title regex, or URL match while optionally waiting for a brand-new tab and closing the previous context after the switch.
- **Script**: Runs custom JavaScript in the current page context with configurable timeouts, returning values for later nodes and paving the way for variable storage.
- **Conditional**: Evaluates JavaScript expressions, selector existence, or workflow variables and routes dedicated IF TRUE / IF FALSE edges without mutating browser state, enabling guarded flows and feature-detection branches.

### Variable System & Picker

- **Set Variable** nodes capture values from static inputs, extraction selectors, or inline JavaScript expressions. Results are stored in an execution-scoped context and persist across every downstream step (and in the execution artifacts) so assertions, scripts, and future variables can reuse them.
- **Use Variable** nodes read from that context, optionally transform the value with a `{{value}}` template, enforce required variables, and publish aliases. The UI now surfaces an inline picker powered by `useWorkflowVariables`, so every previously stored variable appears as both a datalist entry and a clickable chip—no more guessing names or retyping long aliases.
- **Workflow pattern example**:
  1. Extract the `<h1>` from `https://example.com` and store it as `pageTitle`.
  2. Set `greeting` to `Hello {{pageTitle}}!` (interpolation happens automatically before the node executes).
  3. Add a Use Variable node, pick `greeting` from the suggestions, transform it with `Greeting => {{value}}`, and store the result as `greetingEcho` for downstream assertions or screenshots.

The end-to-end regression `TestVariableFlowEndToEnd` in `api/automation/executor/integration_test.go` mirrors this flow to ensure variables are interpolated, aliased, and snapshotted correctly.

### Scenario Registry Overrides

Set the `SCENARIO_REGISTRY` environment variable to pin scenario names to explicit URLs/ports during testing or when orchestrating multiple scenarios locally. The value can be inline JSON or `@/path/to/registry.json`. Example inline payload:

```bash
export SCENARIO_REGISTRY='{"browser-automation-studio":{"url":"http://127.0.0.1:4173","ports":{"UI_PORT":4173,"API_PORT":4174}}}'
```

When present, Browser Automation Studio resolves `destinationType: "scenario"` navigations and scenario-port API calls using this registry instead of shelling out to `vrooli`, which keeps CI and isolated test runs deterministic.

## 🤖 AI Integration

> AI endpoints compile prompts today but require the production executor, telemetry, and replay artifacts before they can be trusted in automation loops.

### Workflow Generation
The AI can generate complete workflows from natural language:
```bash
"Navigate to amazon.com, search for laptops, 
click the first result, extract the price"
```

### Debugging Assistant
Failed workflows can be debugged with Claude Code:
- Analyzes error logs and screenshots
- Suggests fixes for selectors
- Handles website changes automatically

## 📁 Workflow Organization

Workflows are organized in a folder structure:
```
/ui-validation
  /checkout
    - checkout-test
    - payment-validation
  /login
    - login-flow
/data-collection
  - competitor-pricing
  - news-scraper
/automation
  - invoice-download
```

## 💾 Autosave & Version History

- Autosave debounces for roughly 2.5 seconds and only fires when the serialized nodes/edges fingerprint changes, so idle edits do not trigger redundant writes.
- Every successful save (manual, autosave, execution safety save, filesystem sync) appends a `workflow_versions` record with change description, author/source, and a deterministic hash used for diffing.
- The UI header surfaces save state, conflict warnings, and a **History** dialog listing the 50 most recent revisions with author, timestamp, node/edge counts, and the definition hash. Restoring any entry creates a brand-new version so lineage remains intact.
- CLI parity exists through `browser-automation-studio workflow versions list|get|restore`, enabling scripted audits and rollbacks.
- Version conflicts automatically fetch the latest server snapshot, summarise the differences, and expose **Reload** (keep server copy) or **Force Save** (overwrite) actions so collaborators do not clobber each other.

## 🔌 Integration with Other Scenarios

> Until richer artifacts (interactions, telemetry, replay schemas) land, treat these APIs as sequential navigation/screenshot scaffolding for future integration experiments.

Browser Automation Studio provides automation capabilities to all Vrooli scenarios (roadmap intent):

```javascript
// From any scenario
const API_PORT = process.env.BROWSER_AUTOMATION_API_PORT;
const result = await fetch(`http://localhost:${API_PORT}/api/v1/workflows/execute`, {
  method: 'POST',
  body: JSON.stringify({
    workflow_id: 'ui-validation-checkout',
    parameters: { productId: '123' }
  })
});
```

## 🎨 UI Style

The UI follows a technical, developer-focused aesthetic:
- Dark theme with syntax highlighting
- Split-pane layout for workflow building and execution viewing
- Console-style log output
- Filmstrip screenshot timeline (driven by the execution timeline API when populated)

## 🧪 Testing

> Automated coverage combines unit baselines with the telemetry smoke integration run (highlight overlays, console logs, network events, replay export manifest). Handler/websocket suites are still pending.

```bash
# Run scenario tests
vrooli scenario test browser-automation-studio

# Run unit tests
cd api && go test ./...
cd ../ui && pnpm test        # runs sharded Vitest suites (stores + component smoke tests)

# Optional: heavyweight WorkflowBuilder Vitest suite (requires lots of RAM)
pnpm test:workflow-builder   # still memory hungry; pair with Browserless workflow smoke checks for end-to-end coverage

> Note: The WorkflowBuilder Vitest suite currently requires >10GB of heap because of React Flow + Monaco mocks. The default `pnpm test` command runs the lighter shards and skips the builder suite to keep CI stable.

# Run replay renderer automation
testing::playbooks::bas::run_workflow --file test/playbooks/replay/render-check.json --folder /regression
```

## 📊 Metrics

> Metric dashboards are not wired yet; these counters will be populated once real executions and artifact storage exist.

Planned metrics include:
- Workflows created
- Workflows executed
- Success rate
- Average execution time
- Screenshots captured

## 🔒 Security

> Security features are design targets. Current implementation uses development settings without auth or encryption. Harden after executor MVP.

Planned controls:
- Screenshots encrypted at rest in MinIO
- Role-based access to workflows
- Audit trail for all executions
- No secrets in workflow definitions

## 🚧 Known Limitations

- Sub-workflows and richer expression builders are still on the execution roadmap.
- Telemetry exports (webhook routing, analytics packages) are not wired; monitoring remains console/CLI only.
- Replay pipeline outputs interactive HTML packages (including DOM snapshots and animated cursor trails) but MP4/GIF exporters still require advanced motion choreography.
- Chrome extension capture path is not integrated with the replay/automation pipeline.

## 🔮 Future Enhancements

- [ ] Visual regression testing with diff highlights
- [ ] Workflow marketplace for sharing
- [ ] Parallel execution across browsers
- [ ] Cloud execution options
- [ ] Advanced AI with auto-fix capabilities

## 📚 Documentation

- [API Documentation](docs/api.md)
- [CLI Reference](docs/cli.md)
- [Workflow Schema](docs/workflow-schema.md)
- [Integration Guide](docs/integration.md)

## 🤝 Contributing

This scenario is a core Vrooli capability. Improvements here benefit all UI-based scenarios.

---

**Part of the Vrooli Ecosystem** - Every workflow created becomes permanent intelligence the system uses forever.
