# Execution Configuration

This document covers the operator-facing controls that affect execution planning and timing.

## Principles

- Timeouts are configuration.
- Runtime estimates are telemetry.
- Test Genie does not let operators hard-code runtime estimates.

That separation is intentional. A timeout is a safety boundary for execution. An estimate is guidance derived from recent history.

## Per-Scenario Configuration

Configure execution behavior in the target scenario's `.vrooli/testing.json`.

```json
{
  "phases": {
    "unit": {
      "timeout": "120s",
      "enabled": true
    },
    "performance": {
      "enabled": false
    }
  },
  "presets": {
    "focused": ["structure", "unit", "integration"]
  }
}
```

## Supported Planning Controls

| Control | Location | Effect |
|---------|----------|--------|
| Phase timeout | `.vrooli/testing.json` → `phases.<name>.timeout` | Sets the phase runtime budget used during execution and shown as `timeoutSeconds` in previews |
| Phase enablement | `.vrooli/testing.json` → `phases.<name>.enabled` | Removes a phase from default preset expansion when disabled |
| Custom presets | `.vrooli/testing.json` → `presets` | Defines named phase bundles that the planner can expand |
| Global phase toggle | Test Genie API settings surface | Disables a phase by default across all scenarios until explicitly re-enabled |
| Fail-fast | Request payload / CLI flag | Stops execution on the first failed phase, but does not change the precomputed plan |

## What The Planner Stores

Each execution record stores enough metadata to make future estimates honest:

- `requestedPreset`
- `requestedPhases`
- `requestedSkipPhases`
- `plannedPhases`
- `failFast`
- Per-phase runtime results

This prevents partial or fail-fast runs from looking like complete-suite runtimes.

## How Estimates Are Computed

The planning endpoint and execution preview use the same hierarchy for each selected phase:

1. Recent scenario-specific phase history
2. Blended scenario-specific and global phase history
3. Recent global phase history
4. Timeout fallback when no useful history exists

The current implementation uses recent medians rather than raw averages so outliers do not dominate the estimate.

## What Is Not Configurable

The following are intentionally internal implementation details, not user-facing knobs:

- Historical sample thresholds
- Blending weights
- Recency windows
- Confidence labeling rules

If those become operator controls later, they should be added only with a clear use case and dedicated validation.

## Reading The Preview

`POST /api/v1/executions/plan`, the CLI, and the UI all expose the same planning fields:

- `estimatedDurationSeconds`: runtime guidance from history
- `timeoutSeconds`: execution ceiling after config overrides
- `estimateSource`: where the estimate came from
- `estimateConfidence`: how much evidence supports it
- `estimateSampleSize`: how many runs informed it

## See Also

- [API Reference](api-endpoints.md)
- [CLI Commands](cli-commands.md)
- [Test Presets](presets.md)
