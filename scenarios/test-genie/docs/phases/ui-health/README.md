# UI Health Phase

The `ui-health` phase validates `ui/manifest.json` bindings, slot directories, and overlay rules via **ui-health**, ensuring the scenario's UI surface is wired correctly before runtime UI phases (smoke, playbooks) execute.

## How It Runs

Test Genie calls `ui-health` through `scenario-validation/v1.ScenarioValidationService.ValidateScenario`. ui-health checks the scenario's `ui/manifest.json`, slot directories, and overlay rules, then returns shared `status` plus `assessment.findings`.

Equivalent human operator flow:

```bash
ui-health validate scenario <scenario>
```

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip ui-health
```

Disable per scenario via `.vrooli/testing.json`:

```json
{
  "phases": {
    "ui-health": { "enabled": false }
  }
}
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json` (default: 60s):

```json
{
  "phases": {
    "ui-health": { "timeout": "120s" }
  }
}
```
