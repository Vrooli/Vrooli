# Configuration

## Environment variables

Key variables include `API_PORT`, `UI_PORT`, `VROOLI_SCENARIOS_DIR`, and API base variables consumed by the UI.

## Service manifest (`.vrooli/service.json`)

The manifest declares SQLite storage, scenario dependencies on `proto-health` and `code-facts`, resource dependencies, setup steps, and lifecycle commands.

## Schema bootstrap

SQLite schema ownership lives in API domain packages and is applied during API startup.

## CLI config file

The CLI uses shared Vrooli CLI configuration and API-base resolution for local scenario calls.

## API-base resolution precedence

The UI and CLI prefer explicit environment/config values, then lifecycle-discovered ports, then documented local defaults where the shared helper allows them.

## Test/CI configuration

`.vrooli/testing.json` enables strict Go and Node lint handlers and the scenario test suite owns phase-level validation.

## Cross-references

- `../../.vrooli/service.json`
- `../../.vrooli/testing.json`
