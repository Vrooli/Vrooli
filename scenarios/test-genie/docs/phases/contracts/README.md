# Contracts Phase

The `contracts` phase validates `cli/manifest.json` bindings against the proto descriptors via **cli-health**, ensuring every advertised CLI command resolves to a real API contract and no Connect-RPC method has drifted out of sync with its CLI counterpart.

## How It Runs

Test Genie calls `cli-health` through `scenario-validation/v1.ScenarioValidationService.ValidateScenario`. cli-health validates the scenario's `cli/manifest.json`, comparing declared command bindings to the generated proto descriptors, and returns shared `status` plus `assessment.findings`.

Equivalent human operator flow:

```bash
cli-health validate scenario <scenario>
```

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip contracts
```

Disable per scenario via `.vrooli/testing.json`:

```json
{
  "phases": {
    "contracts": { "enabled": false }
  }
}
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json` (default: 60s):

```json
{
  "phases": {
    "contracts": { "timeout": "120s" }
  }
}
```
