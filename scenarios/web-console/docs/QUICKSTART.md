# Quick Start

Get a browser terminal running in under 2 minutes.

## Prerequisites

- Vrooli CLI installed (`vrooli` available in PATH)
- Go toolchain (1.21+)
- Node.js 18+

## Start the Scenario

```bash
cd scenarios/web-console
make start
```

This builds the Go API and Vite UI, then starts both services. Ports are dynamically allocated — check output for URLs.

## Open the Terminal

1. Open the UI URL printed by `make start` (typically `http://localhost:<UI_PORT>`)
2. Click **New Terminal** to open the launcher
3. Choose **Empty Shell** or a shortcut (Claude Code, Codex)
4. A PTY-backed terminal session opens in your browser

## Key Features

- **Multiple panes**: Click "New" to open additional terminals side-by-side
- **Mobile toolbar**: On mobile, a floating toolbar provides Esc, Tab, Ctrl+C, arrow keys
- **Session drawer**: Click the menu icon to see active sessions and manage them
- **Shortcut launcher**: Pre-configured commands for common workflows

## Configuration

API behavior is controlled via environment variables (e.g., `WC_MAX_SESSIONS`, `WC_TERMINAL_SCROLLBACK_LINES`). See [Configuration Reference](reference/configuration.md) for details.

UI appearance is controlled via compile-time constants in [CODE: ui/src/consts/config.ts].

## Lifecycle Commands

```bash
make start   # Build and start API + UI
make test    # Run all test suites
make logs    # View service logs
make stop    # Stop all services
```

## Troubleshooting

- **"Unable to reach the API"**: Ensure the scenario is running via `make start` or `vrooli scenario start web-console`
- **Terminal not responding**: The PTY process may have exited — close the pane and open a new terminal
- **Port conflicts**: Ports are dynamically allocated; check `vrooli scenario port web-console API_PORT`
