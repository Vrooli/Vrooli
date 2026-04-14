# Scenario Auditor Storage Audit

## Last Updated
2026-04-13

## Canonical Storage Policy

`scenario-auditor` now uses `github.com/vrooli/api-core/storage` for mutable runtime filesystem state.

The scenario no longer writes repo-local runtime files under project `.vrooli/data/`.

## Current Layout

Using `AppID: "vrooli"` and `ScenarioID: "scenario-auditor"`:

- `ClassConfig`
  - `rule-preferences.json`
  - `protected-scenarios.json`
  - `automated-fix-config.json`
- `ClassData`
  - `automated-fix-history.json`
- `ClassCache`
  - `standards-violations.json`
  - `vulnerabilities.json`

## Rationale

- `rule-preferences.json` and `protected-scenarios.json` are durable operator-controlled configuration.
- automated-fix configuration is durable configuration, while automated-fix execution history is mutable application data.
- standards and vulnerability scan outputs are rebuildable scan artifacts, so they belong in cache rather than repo-local data directories.

## Anti-Patterns Removed

- Direct writes to `<repo>/.vrooli/data/scenario-auditor`
- Custom scenario-specific runtime path policy
- Mixed config-plus-history storage in a single `automated-fixes.json` file

## Validation Notes

- Storage path resolution is covered by focused unit tests in `api/storage_paths_test.go` and `api/storage_persistence_test.go`.
- The scenario should be validated with `vrooli scenario test scenario-auditor` after storage changes.
