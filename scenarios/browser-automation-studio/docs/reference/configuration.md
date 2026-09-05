# Configuration

## Environment variables

`ENGINE=playwright` selects the supported driver. `PLAYWRIGHT_DRIVER_URL` points to an existing driver; otherwise lifecycle supervision starts the local sidecar. `PLAYWRIGHT_CHROMIUM_PATH` can select an existing Chromium binary.

## Service manifest (`.vrooli/service.json`)

The scenario service manifest is the lifecycle and resource authority. Change it with scenario lifecycle conventions, not ad-hoc process commands.

## Schema bootstrap

The API initializes registered schema domains through routed connections. Use migrations/registry ownership for schema changes.

## CLI config file

Use the scenario CLI through the Vrooli control plane; keep credentials out of tracked configuration.

## API-base resolution precedence

The driver URL explicit environment variable takes precedence over the supervised local sidecar.

## Test/CI configuration

Run `vrooli scenario test browser-automation-studio`; Test Genie owns the run and its baseline comparison.

## Cross-references

- [Service manifest](../../.vrooli/service.json)
- [Operations](../operations/RUNBOOK.md)
