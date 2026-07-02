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

The same file may also contain `unit.policy_profile`, which is the Unit Health
unit-infrastructure policy contract. Test Genie preserves compatibility with
that shape but does not interpret it; the `unit` phase delegates policy,
surface, coverage, and architecture validation to Unit Health. Test Genie owns
only orchestration controls here: phase enablement, timeouts, presets, and
request-time execution flags.

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

## Visual Render-Health & Comparison Levers

Visual render-health and comparison thresholds are owned by ui-health. Test
Genie can still expose run visual deltas through `CompareRunVisuals`, but that
RPC delegates analysis to `ui-health.VisualHealthService.CompareArtifacts`.
ui-health's stdlib-only pixel engine downscales screenshots to a coarse
luminance grid so anti-aliasing/font jitter does not produce false signals.

Every threshold is an environment-overridable lever. Unset or out-of-range
values fall back to the safe default (a malformed lever degrades, never fails).
All are monotonic — "higher = stricter" is noted per lever.

| Lever | Default | Range | Effect |
|---|---|---|---|
| `UI_HEALTH_VISUAL_GRID_SIZE` | `32` | int > 0 | Edge of the square luminance grid each image is downscaled to. Higher = finer detail (more sensitive, slower); lower = more jitter-tolerant. |
| `UI_HEALTH_VISUAL_BLANK_FRACTION` | `0.98` | (0,1] | Share of grid cells in one luminance band for a render to be judged blank/solid (broken). Higher = stricter (only near-uniform frames are broken). |
| `UI_HEALTH_VISUAL_MIN_VARIANCE` | `0.0005` | >= 0 | Grid-luminance variance at/below which a render is judged flat/broken. Higher = stricter (more frames flagged broken). |
| `UI_HEALTH_VISUAL_PIXEL_DELTA` | `0.06` | [0,1] | Per-cell normalized luminance delta above which a cell counts as changed in a comparison. Higher = looser (small shifts ignored). |
| `UI_HEALTH_VISUAL_CHANGED_TOLERANCE` | `0.01` | [0,1] | Share of changed cells at/below which two captures are "identical". Higher = looser (more drift tolerated as identical). |

**Verdict semantics.** A clearly-broken render is a hard ui-health finding and
needs no baseline. Any other visual difference is surfaced by
git-control-tower's baseline diff as the neutral, advisory **`changed`** tier
("review before/after") — it never gates a run (the diff exit code stays 0).
Console errors are counted but are not, alone, a failure; network failures and
broken renders are.

## Run-Lifecycle Feedback Levers

`test-genie execute` (and `vrooli scenario test`) own a server-durable run: the
command returns a run id up front and either follows the run inline or
auto-backgrounds it, so a long run never blocks an agent's tool past its timeout.

| Lever | Default | Effect |
|---|---|---|
| `TEST_GENIE_AUTOBACKGROUND_SECONDS` | `60` | ETA (seconds) at/above which a run auto-backgrounds instead of following inline. `0` disables auto-backgrounding entirely (every run follows inline); values below `10` are clamped up to `10`. |
| `TEST_GENIE_AUTOBACKGROUND_ON_UNKNOWN_ETA` | `1` (on) | When a run's ETA is unknown (first/unestimatable run), auto-background it (treat as potentially long) rather than following inline. `0`/`false` makes unknown-ETA runs follow inline. Moot when auto-backgrounding is disabled. |

A backgrounded or interrupted run is re-attached with the streaming verb
`test-genie runs follow <scenario> <run-id>` (prints live progress + heartbeats).
`runs wait` blocks to the same exit code; in human mode it also streams, while
`--json` stays a single quiet snapshot for scripts.

## Architecture Gate Levers

The `architecture` phase delegates to architecture-cartographer. Its findings
stay graded, but blocker findings can fail the phase when the domain authority
is strong enough to trust.

| Lever | Default | Values | Effect |
|---|---|---|---|
| `TEST_GENIE_ARCHITECTURE_GATE` | `high-confidence` | `off`, `high-confidence`, `all` | Controls whether architecture blocker findings fail the phase. `high-confidence` gates only when cartographer reports curated/high authority, `all` gates every blocker, and `off` keeps all findings advisory. Invalid values fall back to the default and emit a warning observation. |

## See Also

- [API Reference](api-endpoints.md)
- [CLI Commands](cli-commands.md)
- [Test Presets](presets.md)
