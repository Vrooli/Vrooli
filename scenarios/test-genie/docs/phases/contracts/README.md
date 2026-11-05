# Contracts Phase

The `contracts` phase validates `cli/manifest.json` bindings against the proto descriptors via **cli-health**, ensuring every advertised CLI command resolves to a real API contract and no Connect-RPC method has drifted out of sync with its CLI counterpart. With execution requested (the default for this phase), cli-health additionally runs a **runtime CLI probe**: it resolves and execs the scenario's binary, walks its `--help` tree, and reconciles the observed command surface against the manifest — so the single CLI authority covers both the static contract and the binary's real behavior. (This runtime check absorbed the former `integration` phase's CLI liveness checks, which have been retired.)

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md): the sections below follow the required remediation-doc skeleton, so a doc-search topic emitted in a run's scorecard resolves to the exact remediation section. It is the **reference lighthouse adopter** of the contract.

## North Star

Every CLI command is a **verified renderer-separated primitive** on a clean contract: its manifest binding resolves to a real proto method, the runtime `--help` surface matches the manifest, measures and entrypoint metadata are well-formed, and `--json` is an output contract rather than an operation selector. At maximum maturity the scenario's `command_architecture` capability is L4 (every declared cli-core primitive is proven by committed evidence) and the discovery surface is clean enough for cross-scenario search. The deep design SSOT for the top capability is [`cli-architecture-maturity.md`](../../../../cli-health/docs/reference/cli-architecture-maturity.md).

## The rungs and their gates

cli-health reports a ladder per capability (`manifest_contract`, `proto_bindings`, `runtime_surface`, `measures_metadata`, `entrypoint_structure`, `discovery_readiness`, and the deepest one, `command_architecture`). The rungs are monotone — each implies the one below.

| Rung | Gate (entry → exit) | Next unlock |
|---|---|---|
| L0 Unavailable | No inspectable manifest/proto/runtime surface to classify. | Resolve the manifest, proto binding, and runtime shell so the surface can be inspected. |
| L1 Foundation | The CLI surface is inspectable (cli-core shell, manifest present). | Clear schema, binding, runtime, and measure findings. |
| L2 Ready | Contract validation is reusable by discovery; no schema/binding/runtime errors. | Clear the remaining provider-owned cli-health findings; declare command architecture. |
| L3 Complete (most capabilities) / Declared (command_architecture) | The contract is clean; command architecture is *declared* but not yet verified. | Prove each declared cli-core primitive with committed evidence (`command_architecture` only). |
| L4 Verified (command_architecture) | Every declared primitive is verified by matching cli-core evidence; renderer separation is proven, not claimed. | Maximum maturity reached. |

## What each finding means

Each finding caps the capability it names at a rung; only ERROR/BLOCKER severities fail the phase, so most architecture findings are honest, non-failing debt.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `manifest.missing` / `manifest.schema_error` / `manifest.parse_error` | manifest_contract | L1 | ERROR | Yes |
| `binding.unknown_service` / `binding.unknown_method` / `proto.orphan_method` | proto_bindings | L2 | ERROR | Yes |
| `cli.binary_unrunnable` | runtime_surface | L1 | WARNING (degraded) | No |
| `cli.help_failed` / `cli.command_undeclared` | runtime_surface | L2 | ERROR | Yes |
| `measure.invalid` / `measure.schema_unread` | measures_metadata | L2 | WARNING | No |
| `arch.primitive_undeclared` / `arch.primitive_unverified` | command_architecture | L3 | WARNING (debt) | No |
| `arch.primitive_mismatch` / `arch.metadata_invalid` / `arch.claimed_maturity_violation` | command_architecture | L4 | ERROR | Yes |
| `arch.evidence_malformed` / `arch.evidence_stale` | command_architecture | L4 | ERROR / WARNING | malformed=Yes |

The full `command_architecture` inventory (exception taxonomy, primitive taxonomy, evidence trust states) is in [`cli-architecture-maturity.md`](../../../../cli-health/docs/reference/cli-architecture-maturity.md).

## The canonical fix

- **Manifest/binding/proto findings** → correct `cli/manifest.json`: add missing commands, fix `binding.service`/`binding.method`, and regenerate proto so no method is orphaned. Run `cli-health validate scenario <scenario>`.
- **Runtime-surface findings** → make the binary's `--help` tree match the manifest; declare genuinely-custom commands in the manifest `exceptions[]` with a reason.
- **`arch.primitive_undeclared` / `arch.primitive_unverified`** → declare each command's `architecture.primitive`, build the handler with the matching cli-core builder (`ProtoList`/`ProtoMutation`/`ProtoOperational`/durable/streaming), and regenerate the committed `.vrooli/generated/cli-primitive-evidence.json` (golden test with `UPDATE_CLI_EVIDENCE=1`). Never run the command to collect evidence.
- **`arch.primitive_mismatch` / `arch.metadata_invalid` / `arch.claimed_maturity_violation`** → the declaration contradicts the evidence or names a non-special command; correct the declaration or the exception.

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
cli-health validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases contracts
test-genie runs findings --scenario <scenario>
```

The `contracts` line in the scorecard shows the current rung, the single highest-unlock next move, and a runnable doc-search topic that resolves back to the section above.

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
