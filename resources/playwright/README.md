# Playwright Driver Resource

Lightweight, non-Docker Playwright driver for Vrooli scenarios. It runs the Node-based driver that the Go `PlaywrightEngine` talks to over HTTP, so scenarios can execute browser automation without Browserless.

## Layout
- `driver/server.js` — HTTP API (`/session/start|run|reset|close`, `/health`), optional Chromium path override via `PLAYWRIGHT_CHROMIUM_PATH`.
- `cli.sh` — start/stop/status helpers (PID-file based, no `pkill`).
- `manage.sh` — resource manager entrypoint (`install|start|stop|status|health`).
- `package.json` — pins `playwright-core`; add `@playwright/browser-chromium` if you are not reusing an existing Chromium.
- `scripts/install-browsers.sh` — optional helper to download the Playwright Chromium build.
- `config/default.env` — default port/host settings.
- `config/schema.json` — resource configuration schema.
- `config/runtime.json` — lifecycle/runtime hints.
- `config/messages.sh` — status messages for lifecycle tooling.

## Env vars
- `PLAYWRIGHT_DRIVER_PORT` (default `39400`; set `0` to pick a free port)
- `PLAYWRIGHT_DRIVER_HOST` (default `127.0.0.1`)
- `PLAYWRIGHT_CHROMIUM_PATH` (optional path to a Chromium/Chrome binary; use this to reuse Electron’s Chromium)
- `PLAYWRIGHT_BROWSERS_PATH` (defaults to the vendored `@playwright/browser-chromium` if present)
- `HEADLESS` (`true` by default)
- `PLAYWRIGHT_STATE_DIR` (optional; where PID/log/port files live; defaults to `TMPDIR`/`/tmp`)

## Quick start (Tier‑1/dev)
```bash
cd resources/playwright
pnpm install   # or npm/yarn; installs playwright-core
./scripts/install-browsers.sh  # optional: downloads Chromium if you don't reuse a vendored/Electron binary
./manage.sh start
curl http://127.0.0.1:39400/health
./manage.sh stop
```

To export engine env vars for the Go API/UI after the driver is running:
```bash
resource-playwright env  # prints ENGINE=playwright and PLAYWRIGHT_DRIVER_URL
```

## Desktop/Electron notes
- Bundle `driver/server.js`, its `node_modules`, and a Chromium binary with your Electron app. The driver auto-uses a vendored `@playwright/browser-chromium` if present, or `PLAYWRIGHT_CHROMIUM_PATH` if you want to reuse Electron’s Chromium.
- Start the driver from Electron’s main process (set `PLAYWRIGHT_DRIVER_PORT=0`), read the chosen port from stdout/port file, export `PLAYWRIGHT_DRIVER_URL`, set `ENGINE=playwright`, and only then launch the bundled API/UI.
- Keep everything offline: no `npm install` or browser downloads at first launch. Ship the browser bits in the app or repoint `PLAYWRIGHT_CHROMIUM_PATH` to Electron’s binary.
- Prefer pinning Electron + Playwright to compatible versions so reusing Electron’s Chromium doesn’t break.

## Commands
- `resource-playwright install|start|stop|restart|status|health|logs|info` (symlinked to `resources/playwright/manage.sh`)
- `status --json` returns a compact JSON object with `status`, `running`, `healthy`, `host`, `port`, and `pid`.
- `info` prints `config/runtime.json` (useful for deployment tooling).
- `resources/playwright/manage.sh` and `resources/playwright/cli.sh` both support the same verbs.
