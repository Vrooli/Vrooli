# Browser Automation Studio

Visual browser automation workflow builder with AI-powered generation and debugging.

## 🎯 Overview

Browser Automation Studio transforms browser automation from code-based scripts to visual, self-healing workflows. It provides a drag-and-drop interface for creating browser automation workflows, real-time execution monitoring with screenshots, and AI assistance for both generation and debugging.

## ⚠️ Current Implementation Status (2025-10-18)
- The Go executor (`api/browserless/client.go`) now generates scripts for Browserless' `/chrome/function`, updates execution records, and stores real screenshots, but it only supports sequential `navigate`/`wait`/`screenshot` nodes. React Flow edges are ignored, interaction nodes fail with "unsupported" errors, and there is no console/network/cursor telemetry.
- WebSocket updates now use the native gorilla hub events and the UI store consumes the same payloads; richer visuals still depend on adding cursor/network artifacts.
- Replay tooling, screenshot highlighting, and artifact stitching are aspirational; only ad-hoc preview endpoints shell out to `resource-browserless` today.
- Requirements tracking: `docs/requirements.yaml` (draft) plus `scripts/requirements/report.js` now emit summary coverage; integration with automated test outputs is still outstanding.
- Tests cover structure/linting only; no integration tests exercise the automation path. Use this scenario as a work-in-progress playground until the action plan lands.

> See `docs/action-plan.md` for the full backlog and execution roadmap.

## 📊 Status Dashboard
- Requirement coverage (`docs/requirements.yaml`): total 4 • complete 0 • in progress 0 • pending 4 • critical gap (P0/P1 incomplete) 4.
- Generate a fresh snapshot with `node ../../scripts/requirements/report.js --scenario browser-automation-studio --format markdown` from the scenario root.

## ✨ Features

Status legend: ✅ scaffolding exists • 🚧 active development • 🌀 planned polish

- ✅ **Visual Workflow Builder**: React Flow UI stores node/edge JSON and project folders in Postgres.
- ✅ **API/CLI Scaffolding**: REST handlers and CLI commands exist for workflows/executions; the API now runs sequential Browserless steps but responses lack interaction data and streaming updates.
- 🚧 **Real-time Execution Stream**: Native WebSocket events reach the UI; timelines still lack cursor/network overlays until the executor streams those artifacts.
- 🚧 **AI Workflow Generation**: Prompt pipeline calls OpenRouter but lacks validation against a runnable executor.
- 🚧 **AI Debugging Loop**: Endpoint stubs exist; needs real telemetry + replay artifacts to be effective.
- 🌀 **Replay & Marketing Renderer**: Highlighted screenshots, cursor animation, and stitched exports are design goals awaiting artifact schema work.

## 🚀 Quick Start

> Current executables run limited sequential workflows (navigate/wait/screenshot only). Use these commands to validate plumbing while the full executor roadmap lands.

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
```

## 🏗️ Architecture

### Components

1. **UI (React + Vite + TypeScript)**
   - React Flow for visual workflow building
   - WebSocket scaffolding for real-time updates (requires new executor events)
   - Zustand for state management
   - Tailwind CSS for styling

2. **API (Go + Chi)**
   - RESTful API for workflow management
   - Gorilla WebSocket hub broadcasting structured `ExecutionUpdate` + event payloads consumed directly by the UI
   - PostgreSQL for persistence
   - MinIO for screenshot storage

3. **CLI (Bash)**
   - Thin wrapper around API
   - Currently polls `/executions` because streaming endpoints are unfinished
   - JSON and human-readable output

### Workflow Node Types

- **Navigate**: Executes via Browserless for sequential flows; edges/branching are ignored and each step shares a single page context.
- **Click**: Not yet supported—executor returns an unsupported step error until interaction primitives land.
- **Type**: Not yet supported—text input helpers are part of the execution-core backlog.
- **Screenshot**: Stores real full-page PNGs (MinIO/local fallback) but lacks element highlighting, viewport presets, or cursor overlays.
- **Wait**: Time waits execute; element waits only perform `waitForSelector` without retry/backoff instrumentation yet.
- **Extract**: Still placeholder—the executor has no DOM extraction helpers wired.

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
- Filmstrip screenshot timeline

## 🧪 Testing

> Automated coverage is limited to structure/linting checks today; add executor + handler tests as the integration lands.

```bash
# Run scenario tests
vrooli scenario test browser-automation-studio

# Run unit tests
cd api && go test ./...
cd ../ui && npm test
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

- Workflow executions use mocked Browserless responses; no DOM interaction occurs yet.
- UI real-time panels now hydrate from native WebSocket events; enhanced telemetry (network, console, cursor) remains on the roadmap.
- Replay/highlight/custom screenshot features are unimplemented.
- Browserless resource still needs session orchestration to avoid cold start cost once executor ships.

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
