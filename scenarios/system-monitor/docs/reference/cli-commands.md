# CLI Commands Reference

## Installation

The CLI is installed from the Go module in `cli/` via `cli/install.sh` or `cli/install.ps1`.

- `system-monitor` (primary installed command)

`[CODE: cli/app.go]`

## Commands

| Command | Description | API Endpoint |
|---------|-------------|-------------|
| `health` | Check API health and dependencies | `GET /health` |
| `status` | Operational summary: health, thresholds, alerts, maintenance | `GET /api/v1/metrics/current`, `GET /api/v1/settings`, `GET /api/v1/maintenance/state` |
| `alerts` | Active alerts from the current metrics snapshot | `GET /api/v1/metrics/current`, `GET /api/v1/settings` |
| `watch` | Streaming live metrics + alert view | `GET /api/v1/metrics/current`, `GET /api/v1/settings`, `GET /api/v1/maintenance/state` |
| `dashboard` | Open or print the UI URL | None |
| `metrics current` | Current metrics snapshot | `GET /api/v1/metrics/current` |
| `metrics detailed` | Detailed CPU, memory, GPU, network, and dependency metrics | `GET /api/v1/metrics/detailed` |
| `metrics processes` | Process health and hot process matrix | `GET /api/v1/metrics/processes` |
| `metrics infrastructure` | Pool, queue, and storage I/O metrics | `GET /api/v1/metrics/infrastructure` |
| `metrics timeline` | Recent metrics history | `GET /api/v1/metrics/timeline` |
| `investigations list` | List recent investigations | `GET /api/v1/investigations` |
| `investigations latest` | Latest investigation | `GET /api/v1/investigations/latest` |
| `investigations get <id>` | Investigation by ID | `GET /api/v1/investigations/{id}` |
| `investigations trigger` | Queue a new investigation | `POST /api/v1/investigations/trigger` |
| `investigations cooldown` | Show cooldown status | `GET /api/v1/investigations/cooldown` |
| `investigations cooldown-reset` | Reset cooldown | `POST /api/v1/investigations/cooldown/reset` |
| `investigations cooldown-set` | Update cooldown duration | `PUT /api/v1/investigations/cooldown/period` |
| `investigations triggers` | List trigger thresholds and auto-fix state | `GET /api/v1/investigations/triggers` |
| `reports generate <type>` | Generate a daily or weekly report | `POST /api/v1/reports/generate` |
| `reports list` | List generated reports | `GET /api/v1/reports` |
| `reports get <id>` | Fetch a report by ID | `GET /api/v1/reports/{id}` |
| `settings get` | Show monitor settings | `GET /api/v1/settings`, `GET /api/v1/maintenance/state` |
| `settings update` | Update settings and thresholds | `PUT /api/v1/settings` |
| `settings reset` | Reset settings to defaults | `POST /api/v1/settings/reset` |
| `settings maintenance` | Get or set maintenance state | `GET/POST /api/v1/maintenance/state` |
| `maintenance retention preview --days <n>` | Preview metrics that retention would prune (read-only) | `GET /api/v1/maintenance/metrics/retention/preview` |
| `maintenance retention apply --days <n> --confirm` | Prune metrics older than the window (destructive) | `POST /api/v1/maintenance/metrics/retention/apply` |
| `maintenance compact preview` | Preview reclaimable database space (read-only) | `GET /api/v1/maintenance/metrics/compaction/preview` |
| `maintenance compact apply --confirm` | Compact the database to reclaim space (destructive) | `POST /api/v1/maintenance/metrics/compaction/apply` |

## Global Flags

| Flag | Description |
|------|-------------|
| `--help` | Show help text |
| `--version` | Show CLI version |
| `--api-base <url>` | Override API base URL |
| `--auto-start` | Auto-start the scenario for API-backed commands |
| `--dry-run` | Validate command preflight without executing mutations |
| `--json` | Command-specific JSON mode where supported |

## Examples

```bash
# Operational overview
system-monitor status

# Fresh metrics snapshot
system-monitor metrics current --fresh --json

# Review process anomalies
system-monitor metrics processes

# Trigger an investigation
system-monitor investigations trigger --note "CPU stayed above threshold for 10m"

# Generate a weekly report
system-monitor reports generate weekly

# Put the monitor into maintenance mode
system-monitor settings maintenance --state active
```

## Compatibility Notes

- Legacy shortcuts still work:
  - `system-monitor investigate` maps to `system-monitor investigations latest`
  - `system-monitor report weekly` maps to `system-monitor reports generate weekly`
  - `system-monitor metrics` maps to `system-monitor metrics current`
- The old shell-only `simulate` command was removed because the referenced API endpoint does not exist.
