# UI Health Phase

The `ui-health` phase is the single authority for **all** UI validation. It runs in **execution mode** (`IncludeExecution: true`) and produces one consolidated report covering: `ui/manifest.json` bindings + slot directories + overlay rules; static UI-interop rules; net-new UI standards (i18n parity, design-token/no-raw-hex, a11y harness, PWA/viewport, tsconfig-strict, eslint stability); UI-bundle freshness (the canonical content-hash engine — a stale bundle gates); and a **BAS-driven runtime render + iframe-bridge handshake** group. The runtime group absorbed the retired native `smoke` phase.

The phase declares `NeedsUI: true` and `RequiredResources: [browser-automation-studio]`, so the runnability gate **skips** (never fails) the runtime work when no UI surface or BAS is present; the static groups always run and gate. ui-health additionally degrades its runtime group internally — an unreachable BAS or UI yields a skipped (info) finding, never a phase failure on infra absence.

## How It Runs

Test Genie calls `ui-health` through `scenario-validation/v1.ScenarioValidationService.ValidateScenario` with `include_execution: true`. ui-health runs its static groups always; when execution is requested and the scenario has a UI, it resolves the running UI via discovery and drives BAS for the render + handshake check, then returns shared `status` plus `assessment.findings`.

Equivalent human operator flow:

```bash
ui-health validate scenario <scenario>            # full report (static + runtime)
ui-health validate scenario <scenario> --static-only   # static groups only, no BAS
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
