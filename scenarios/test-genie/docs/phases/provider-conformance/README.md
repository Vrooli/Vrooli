# Provider Conformance Phase

The `provider-conformance` phase validates scenarios that declare themselves as Test Genie phase providers. It is descriptor-backed by `test-genie` itself and only applies when the target has `.vrooli/test-genie.json`. It judges provider **descriptors**, not ordinary scenario testability — a scenario without a phase descriptor is simply not applicable.

Test Genie evaluates applicability before provider readiness. Ordinary scenarios omit this phase from normal runs; JSON previews still show the not-applicable reason. If a target declares a phase descriptor and it is malformed, drifting, or its live provider contract is broken, the phase applies and fails through the shared maturity assessment.

## Provider Contract

- **Provider:** `test-genie`
- **Source:** `validation-provider`
- **RPC:** `scenario-validation/v1.ScenarioValidationService.ValidateScenario`
- **Descriptor:** `scenarios/test-genie/.vrooli/test-genie.json`
- **Maturity:** embedded in that descriptor's `maturity` block
- **Policy:** default when applicable, required provider readiness, **check-only lifecycle** (never starts or restarts Test Genie), live contract freshness, gating results
- **Timeout:** 90s

## What It Validates

Test Genie owns the provider-conformance judgment. The rules are shared with the descriptor registry loader and the `provider-contract scan` core so they cannot drift:

- descriptor presence, parseability, schema version, and identity (`PROVIDER_DESCRIPTOR_MISSING`, `PROVIDER_DESCRIPTOR_INVALID`, `PROVIDER_IDENTITY_MISMATCH`)
- embedded maturity contract validity (`PROVIDER_MATURITY_INVALID`)
- retired `.vrooli/maturity.json` absence after the descriptor cutover (`PROVIDER_STALE_MATURITY_FILE`)
- policy safety (valid, non-contradictory policy combinations) (`PROVIDER_POLICY_UNSAFE`)
- required operator docs path declaration through `docs.path`; path resolution remains advisory (`PROVIDER_DOCS_MISSING`)
- autofix declaration coverage: every maturity finding carries an explicit `fix_class` (`PROVIDER_AUTOFIX_DECLARATION_INCOMPLETE`, advisory)
- live provider contract: reachability (advisory), contract-valid responses, assessment identity, and execution metrics (`PROVIDER_UNREACHABLE`, `PROVIDER_CONTRACT_INVALID`, `PROVIDER_CONTRACT_IDENTITY_MISMATCH`, `PROVIDER_METRICS_MISSING`)

## Self-Recursion Guard

When the target is `test-genie` itself, the live contract probe is skipped: probing would call this process's own `ValidateScenario`, which runs this validator, which would probe again. Self-validation therefore uses internal descriptor checks only, and the phase's `check_only` lifecycle policy guarantees Test Genie never starts or restarts itself during its own run. The `native_detail.probeSkipReason` field reports when and why the probe was skipped.

## Inspection

```bash
test-genie phases applicability <scenario> --phase provider-conformance --json
test-genie phases plan <scenario> --preset comprehensive --json
test-genie provider-contract check provider-conformance <scenario> --json
```
