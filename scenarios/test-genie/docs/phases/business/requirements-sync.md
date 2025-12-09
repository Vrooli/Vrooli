# Requirements Auto-Sync Guide

This guide explains how to use the requirements auto-sync feature in test-genie.

## Overview

Requirements auto-sync automatically updates your scenario's requirement files with live test execution status. After running tests, it enriches requirement validations with pass/fail status from test results, keeping your requirements documentation always up-to-date.

## How It Works

After test suite execution completes, the sync pipeline:

1. **Discovers** requirement files in `requirements/` (follows `index.json` imports)
2. **Parses** JSON requirement modules
3. **Loads evidence** from phase results, Vitest coverage, and manual validations
4. **Enriches** requirements with live test status
5. **Syncs** updated statuses back to requirement files
6. **Writes** snapshots to `coverage/requirements-sync/`

## Quick Start

### 1. Enable Sync (Default: Enabled)

Sync runs automatically after successful test execution. To explicitly configure:

```json
// .vrooli/testing.json
{
  "requirements": {
    "sync": true
  }
}
```

### 2. Run Tests

```bash
# Via Makefile (recommended)
make test

# Via CLI
vrooli test suite <scenario-name>
```

### 3. Check Results

After sync completes:
- Requirement files are updated with live statuses
- Metadata written to `coverage/sync/latest.json`
- Snapshot written to `coverage/requirements-sync/latest.json`

## Configuration

### Testing.json Options

```json
{
  "requirements": {
    "sync": true
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `sync` | boolean | `true` | Enable/disable auto-sync |

### Environment Variables

| Variable | Effect |
|----------|--------|
| `TESTING_REQUIREMENTS_SYNC=false` | Disable sync entirely |
| `TESTING_REQUIREMENTS_SYNC_FORCE=true` | Force sync even if required phases are missing |

### Per-Module Control

Disable sync for specific modules in their `_metadata`:

```json
{
  "_metadata": {
    "module": "manual-only",
    "auto_sync_enabled": false
  },
  "requirements": [...]
}
```

## Requirement File Structure

### Basic Module

```json
{
  "_metadata": {
    "module": "core-features",
    "description": "Core feature requirements"
  },
  "requirements": [
    {
      "id": "REQ-001",
      "title": "User authentication",
      "status": "in_progress",
      "validation": [
        {
          "type": "test",
          "ref": "src/auth.test.ts",
          "status": "implemented"
        }
      ]
    }
  ]
}
```

### With Index Imports

```json
// requirements/index.json
{
  "imports": [
    "01-core/module.json",
    "02-features/module.json"
  ]
}
```

## Evidence Sources

The sync pipeline loads evidence from multiple sources:

| Source | Path | Content |
|--------|------|---------|
| Phase Results | Input from orchestrator | Execution status per phase |
| Phase Files | `coverage/phase-results/*.json` | Detailed phase data |
| Vitest | `ui/coverage/vitest-requirements.json` | Frontend test results |
| Manual | `coverage/manual-validations/log.jsonl` | Manual validation records |

## Validation Types

| Type | Description | Ref Format |
|------|-------------|------------|
| `test` | Unit/integration test | `path/to/test.test.ts` |
| `automation` | E2E automation | `workflows/flow.json` |
| `manual` | Manual verification | Description text |
| `review` | Code review | PR/commit reference |

## Validation Status

After sync, validations receive a live status:

| Status | Meaning |
|--------|---------|
| `passed` | Test executed and passed |
| `failed` | Test executed and failed |
| `skipped` | Test was skipped |
| `not_run` | No evidence found |

## Sync Gating

Sync is **blocked** if any of these conditions apply:

- No phase plan available
- No phases were selected
- No phase results recorded
- `requirements.sync` is disabled
- Required (non-optional) phases are missing or skipped

Use `TESTING_REQUIREMENTS_SYNC_FORCE=true` to bypass gating.

## Output Files

### coverage/sync/latest.json

```json
{
  "synced_at": "2024-12-02T10:30:00Z",
  "test_commands": ["suite my-scenario"],
  "files_updated": 3,
  "validations_added": 0,
  "validations_removed": 0,
  "statuses_changed": 5,
  "error_count": 0
}
```

### coverage/requirements-sync/latest.json

Contains a full snapshot of all requirements with their enriched statuses.

## Troubleshooting

### Sync Not Running

1. Check `.vrooli/testing.json` has `requirements.sync: true`
2. Verify `TESTING_REQUIREMENTS_SYNC` env var is not `false`
3. Ensure required phases completed successfully
4. Check `coverage/sync/latest.json` for error details

### Statuses Not Updating

1. Verify test file paths match validation `ref` fields
2. Check evidence is being generated (look in `coverage/`)
3. Run with `TESTING_REQUIREMENTS_SYNC_FORCE=true` to bypass gating

### View Sync Decision

Check the test execution logs for messages like:
- "Requirements sync completed"
- "Skipping requirements sync: [reason]"

## API Reference

### Service Methods

```go
// Sync performs full requirements synchronization
service.Sync(ctx, SyncInput{
    ScenarioName:     "my-scenario",
    ScenarioDir:      "/path/to/scenario",
    PhaseDefinitions: phases,
    PhaseResults:     results,
    CommandHistory:   []string{"suite my-scenario"},
})

// Report generates a requirements report
service.Report(ctx, scenarioDir, reporting.Options{
    Format: "markdown",
}, os.Stdout)

// Validate checks requirement structure
result, err := service.Validate(ctx, scenarioDir)
```

## See Also

### Related Guides
- [Business Phase](README.md) - Business phase overview
- [Phases Overview](../README.md) - 10-phase testing architecture
- [Test Generation](../../guides/test-generation.md) - AI-powered test creation with `[REQ:ID]` tags
- [Scenario Unit Testing](../unit/scenario-unit-testing.md) - Writing tagged unit tests

### Reference
- [API Endpoints](../../reference/api-endpoints.md) - REST API reference
- [Presets](../../reference/presets.md) - Test preset configurations

### Concepts
- [Architecture](../../concepts/architecture.md) - Go orchestrator design
