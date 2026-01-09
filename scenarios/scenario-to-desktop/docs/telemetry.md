# Telemetry & Operations

Every generated desktop app records lifecycle events to help deployment-manager spot failures in the wild.

## What gets recorded
- `app_start`, `dependency_unreachable`, `local_vrooli_start_failed`, `update_check`, and other lifecycle events.
- Captured in `deployment-telemetry.jsonl` inside the OS-specific user data dir.

## File locations
- Windows: `%APPDATA%/<App Name>/deployment-telemetry.jsonl`
- macOS: `~/Library/Application Support/<App Name>/deployment-telemetry.jsonl`
- Linux: `~/.config/<App Name>/deployment-telemetry.jsonl`

## Collecting telemetry
```bash
scenario-to-desktop telemetry collect \
  --scenario <name> \
  --file "<path-to>/deployment-telemetry.jsonl"
```
- The API stores results at `.vrooli/deployment/telemetry/<scenario>.jsonl`.
- The UI has an upload card under Scenario Inventory.

## Auto-manage Tier 1 (optional)
- Set `auto_manage_vrooli: true` to let the desktop wrapper attempt `vrooli setup/start/stop` locally.
- Keep it **off** when shipping thin clients that should point to an existing remote server.

## Forward-looking
- Bundle manifests will add secret prompts, port allocation, and richer telemetry once the bundled runtime (see `runtime/README.md`) is wired into the generator.
