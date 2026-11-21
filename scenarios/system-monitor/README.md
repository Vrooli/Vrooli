# Vega (System Monitor)

Vega is Vrooli's local operations monitor. It brings machine health, metric
history, investigations, reports, logs, storage analysis and capacity evidence
into one responsive dashboard. Its records can include private host metrics,
process details, logs and investigation notes.

## Privacy and deployment

The API has no authentication middleware. Run Vega only in a trusted local or
private deployment; do not expose its API to a public or untrusted network.
Metrics and investigation records use the scenario-owned SQLite store by
default. In-memory storage is a development mode and is not accepted for
production.

## Open the app

Start and inspect Vega through the scenario lifecycle:

```bash
make start
vrooli scenario status system-monitor --json
system-monitor --help
```

The lifecycle assigns the API and UI ports. The CLI is manifest-backed and
offers `overview`, `metrics`, `investigations`, `reports`, `settings`,
`maintenance` and `capacity` command groups. Use `system-monitor <group>
--help` for the current command contract.

## Dashboard and routes

The dashboard is designed for desktop and phone. Its main workspaces cover
alerts and metric cards; machine and collection state; process ownership;
infrastructure and capacity; investigation history and scripts; reports; and
settings. The five metric detail routes, logs, forensics, capacity and scripts
are directly addressable:

| Route | Purpose |
|---|---|
| `/` | Dashboard, alerts, machine state and monitoring workspaces |
| `/metrics/cpu` | CPU history, load and process evidence |
| `/metrics/memory` | Memory, swap and pressure evidence |
| `/metrics/network` | Network state and connection detail |
| `/metrics/disk` | System disk history and read-only mount analysis |
| `/metrics/gpu` | GPU utilization and device detail |
| `/forensics` | Boot, hardware-error and persistent-store evidence |
| `/logs` | Filterable host log records |
| `/capacity` | Resource findings and policy state |
| `/scripts` | Read-only investigation catalog and supported scripts |

The dashboard uses HTTP polling. Metric values keep measured, failed,
unsupported, stale and not-yet-sampled states distinct. Timeline charts preserve
gaps instead of turning missing observations into zero. Disk usage history
represents the system-monitored path; a selected mount's directory scan is a
current snapshot, not a per-mount historical series.

## Investigation and safety boundaries

Vega can create investigation records and use agent-manager where configured.
Investigation admission, cooldown, selected profile and record status are shown
separately; a timeout does not mean an investigation was accepted. Script
execution is gated by the script's declared capabilities. Terminal output is a
simulated example, not a host shell.

Process termination is not available through Vega's API and remains disabled
in the UI. Disk analysis is read-only. A remediation action, when supported,
belongs to its native owner rather than a private host-repair path in Vega.
Alert delivery and channel routing belong to Notification Hub.

## Architecture

- **UI:** React, Vite and TypeScript.
- **API:** Go with a proto-first Connect contract and a small set of REST
  exceptions.
- **Storage:** scenario-owned SQLite by default; development memory mode is
  non-durable.
- **CLI:** Go, generated from `cli/manifest.json` and backed by Connect clients.
- **Related scenario:** agent-manager for configured investigation work.

The selected dashboard composition and its preservation boundaries are recorded
in [`docs/internal/DESIGN-ADOPTION.md`](docs/internal/DESIGN-ADOPTION.md). The
metric state and timeline contract is in
[`docs/internal/METRIC_CONTRACT.md`](docs/internal/METRIC_CONTRACT.md).

## Development

```bash
make start
make status
make logs
make test
make stop
```

Use `make test` for the scenario-owned Test Genie suite. Do not start the API
binary or Vite server directly; lifecycle commands own process names, ports and
health checks.
