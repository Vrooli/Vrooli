# Playwright Driver Resource

Lightweight, non-Docker Playwright driver for Vrooli scenarios. It runs the Node-based driver that the Go `PlaywrightEngine` talks to over HTTP, so scenarios can execute browser automation without Browserless.

## Layout
- `driver/server.js` — HTTP API (`/session/start|run|reset|close`, `/health`), optional Chromium path override via `PLAYWRIGHT_CHROMIUM_PATH`.
- `cli.sh` — start/stop/status helpers (PID-file based, no `pkill`).
- `package.json` — pins `playwright-core`; add `@playwright/browser-chromium` if you are not reusing an existing Chromium.
- `scripts/install-browsers.sh` — optional helper to download the Playwright Chromium build.
- `config/default.env` — default port/host settings.

## Env vars
- `PLAYWRIGHT_DRIVER_PORT` (default `39400`; set `0` to pick a free port)
- `PLAYWRIGHT_DRIVER_HOST` (default `127.0.0.1`)
- `PLAYWRIGHT_CHROMIUM_PATH` (optional path to a Chromium/Chrome binary; use this to reuse Electron’s Chromium)
- `HEADLESS` (`true` by default)

## Quick start (Tier‑1/dev)
```bash
cd resources/playwright
pnpm install   # or npm/yarn; installs playwright-core
./scripts/install-browsers.sh  # optional: downloads Chromium
./cli.sh start
curl http://127.0.0.1:39400/health
./cli.sh stop
```

## Desktop/Electron notes
- Bundle `driver/server.js` and Playwright binaries with your Electron app.
- Start the driver from Electron’s main process, capture the chosen port (when `PORT=0`), set `PLAYWRIGHT_DRIVER_URL`, and launch the bundled API with `ENGINE=playwright`.
- To avoid bundling an extra Chromium, align Playwright with the Electron Chromium version and set `PLAYWRIGHT_CHROMIUM_PATH` to Electron’s binary.
