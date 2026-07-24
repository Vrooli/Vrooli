# CLI Commands — Secrets Manager

## Source Of Truth: `cli/manifest.json`

The CLI manifest and `secrets-manager --help` are authoritative. This document is a stable orientation map.

## Global Flags

Use `--api-base`, `--instance`, `--auto-start`, and `--json` before the command when supported.

## Built-In Commands

- `secrets-manager status` shows combined posture.
- `secrets-manager health` checks API health.
- `secrets-manager configure` reads or updates CLI configuration.

## Scenario Commands

| Group | Operations |
|---|---|
| `vault` | `status`, `validate`, `provision` |
| `security` | `vulnerabilities`, `scan`, `compliance`, `set-status`, `fix` |
| `deployment` | `plan`, `readiness` |
| `resources`, `scenarios`, `campaigns` | inventory and metadata inspection |
| `overrides`, `admin` | strategy mutation and controlled cleanup |

## Output Contracts

`--json` is intended for automation. Output contains metadata and posture, not secret values.

## Adding A New Command

Keep commands as API clients. Register the command in the CLI manifest, add a CLI test, and update this table after the route contract exists.

## Cross-References

- [API Endpoints](api-endpoints.md)
- [Configuration](configuration.md)
