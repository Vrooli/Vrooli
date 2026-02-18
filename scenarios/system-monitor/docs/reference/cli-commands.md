# CLI Commands Reference

## Installation

The CLI is installed via `cli/install.sh`, which creates a symlink in `~/.vrooli/bin/`. Two entry points are available:

- `system-monitor` (primary)
- `vrooli-system-monitor` (alternative)

`[CODE: cli/system-monitor]`

## Commands

| Command | Description | API Endpoint |
|---------|-------------|-------------|
| `health` | Check API health | `GET /health` |
| `metrics` | Current CPU/memory/TCP metrics | `GET /api/v1/metrics/current` |
| `status` | System status (HEALTHY/WARNING/CRITICAL/OFFLINE) | `GET /api/v1/metrics/current` |
| `alerts` | List active alerts based on thresholds | `GET /api/v1/metrics/current` |
| `investigate` | Fetch latest investigation results | `GET /api/v1/investigations/latest` |
| `report <type>` | Generate daily/weekly report | `POST /api/v1/reports/generate` |
| `watch` | Live monitoring with ASCII progress bars (2s refresh) | `GET /api/v1/metrics/current` |
| `dashboard` | Open UI in default browser | None (uses `xdg-open`) |
| `simulate` | Simulate CPU anomaly (test endpoint) | `GET /api/test/anomaly/cpu` |
| `version` | Show CLI version | None |

## Global Flags

| Flag | Description |
|------|-------------|
| `--help` | Show help text |
| `--version` | Show CLI version |
| `--port <port>` | Override API port (default: auto-detected from lifecycle) |
| `--json` | Output in JSON format |
| `--quiet` | Suppress non-essential output |

## Examples

```bash
# Check if the API is running
system-monitor health

# Get metrics in JSON format
system-monitor metrics --json

# Generate a weekly report
system-monitor report weekly

# Live monitoring
system-monitor watch

# Open the dashboard
system-monitor dashboard
```

## Known Issues

- **`report` command**: Calls `/api/reports/generate` instead of `/api/v1/reports/generate` (missing `/v1/` prefix), resulting in a 404 error
- **`simulate` command**: References `GET /api/test/anomaly/cpu` which does not exist in the API
- **`--quiet` flag**: Parsed but never checked in command implementations (has no effect)
- **JSON parsing**: Uses `grep`/`cut` regex instead of `jq`; fragile with unexpected response formats
