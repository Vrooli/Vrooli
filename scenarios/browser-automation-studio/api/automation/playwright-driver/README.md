# Playwright Driver (Desktop Engine)

Thin Node service that executes `automation/contracts` instructions with Playwright. The Go `PlaywrightEngine` talks to this driver over HTTP so desktop/Electron bundles can reuse the executor/recorder/events stack without Browserless.

## Quick start (dev)
```bash
# From scenarios/browser-automation-studio/api/automation/playwright-driver
PLAYWRIGHT_DRIVER_PORT=39400 node server.js
```
- Ensure `playwright` (or `playwright-core`) is installed in the environment that runs this driver.
- Defaults: launches Chromium headless, listens on `127.0.0.1:39400`.

## API (stable, minimal)
- `POST /session/start` → `{ "session_id": "..." }`
  - Body: `{ execution_id, workflow_id, viewport: {width,height}, base_url, reuse_mode, labels }`
  - Optional flags: `tracing: true`, `video: true`
  - Optional HAR: `har_path: "/tmp/capture.har"`
- `POST /session/:id/run` → StepOutcome payload with inline `screenshot_*` and `dom_*` fields consumed by the Go adapter.
- `POST /session/:id/reset` → `{ status: "ok" }`
- `POST /session/:id/close` → `{ status: "closed" }`
- `GET /health` → `{ status: "ok", sessions }`

## Behavior
- Captures console + network events per step.
- Best-effort screenshot + DOM snapshot per step.
- Supports core instructions: `navigate`, `click`, `hover`, `type`, `scroll`, `wait`, `evaluate` (returns `extracted_data.result`), `extract` (returns map keyed by selector), `assert` (populates assertion outcome), `uploadFile`, `screenshot`. Unsupported instructions return an engine error until implemented.

## Bundling notes
- Desktop/Electron builds should embed this driver and a Chromium binary, then set `PLAYWRIGHT_DRIVER_URL` for the Go API (e.g., `http://127.0.0.1:39400` or a Unix socket bridge).
- Server/Tier 1 builds keep using Browserless; selection happens via `ENGINE=playwright|browserless` and `ENGINE_OVERRIDE`.
- Replay export: when `PLAYWRIGHT_DRIVER_URL` is set and Browserless is not configured, the `ReplayRenderer` will fall back to this driver for capture, using the same export page URL.
- Multi-frame capture: Replay capture via Playwright takes repeated wait+screenshot steps based on the movie spec duration and frame interval; ensure the export page emits `bas:render` to start playback.

## Testing
- Unit tests for the Go adapter run by default (`go test ./automation/engine`).
- Optional integration smoke: start the driver (`PLAYWRIGHT_DRIVER_PORT=39400 node server.js`) and run `PLAYWRIGHT_DRIVER_URL=http://127.0.0.1:39400 go test ./automation/engine -run PlaywrightEngine_IntegrationSmoke`.
