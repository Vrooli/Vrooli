# Contracts Phase

The `contracts` phase validates `cli/manifest.json` bindings against the proto descriptors via **cli-health**, ensuring every advertised CLI command resolves to a real API contract and no Connect-RPC method has drifted out of sync with its CLI counterpart. With execution requested (the default for this phase), cli-health additionally runs a **runtime CLI probe**: it resolves and execs the scenario's binary, walks its `--help` tree, and reconciles the observed command surface against the manifest — so the single CLI authority covers both the static contract and the binary's real behavior. (This runtime check absorbed the former `integration` phase's CLI liveness checks, which have been retired.)

## How It Runs

Test Genie calls `cli-health` through `scenario-validation/v1.ScenarioValidationService.ValidateScenario` with `include_execution=true`. cli-health validates the scenario's `cli/manifest.json`, comparing declared command bindings to the generated proto descriptors, then probes the binary's runtime command surface, and returns shared `status` plus `assessment.findings`. The runtime probe emits `cli.binary_unrunnable` (warning — binary absent in this run context, degraded), `cli.help_failed` (error — binary present but `--help` is broken), and `cli.command_undeclared` (error — the runtime command surface diverges from the manifest). It degrades gracefully: a scenario whose CLI is not installed in the run context is never hard-failed.

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

Per-scenario timeout override via `.vrooli/testing.json` (default: 90s — the runtime probe execs the binary's `--help` tree):

```json
{
  "phases": {
    "contracts": { "timeout": "120s" }
  }
}
```
