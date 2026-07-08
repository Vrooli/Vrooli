# Provider Conformance Phase

The `provider-conformance` phase validates scenarios that declare themselves as Test Genie phase providers. It is descriptor-backed by `test-genie` itself and only applies when the target has `.vrooli/test-genie.json`. It judges provider **descriptors**, not ordinary scenario testability — a scenario without a phase descriptor is simply not applicable.

Test Genie evaluates applicability before provider readiness. Ordinary scenarios omit this phase from normal runs; JSON previews still show the not-applicable reason. If a target declares a phase descriptor and it is malformed, drifting, or its live provider contract is broken, the phase applies and fails through the shared maturity assessment.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

A phase provider is **fully conformant** on both capabilities: its `.vrooli/test-genie.json` descriptor is contract-clean (`provider_descriptor` L3 — identity, embedded maturity, policy safety, no stale maturity file, and skeleton operator docs all pass) and its live `ScenarioValidationService` honors the shared contract with correct identity and execution metrics (`provider_contract` L3). At maximum maturity every declared phase resolves a well-formed contract that Test Genie can register and govern without per-phase glue — the descriptor is a trustworthy source of truth for the whole catalog.

## The rungs and their gates

provider-conformance reports two capability ladders. Each rung is monotone; the top rung of each is that capability's North Star.

| Capability | Rungs | Top rung (North Star) | Example next unlock |
|---|---|---|---|
| `provider_descriptor` | L0 Descriptor missing → L3 Descriptor clean | Provider descriptor maturity is clean | L1→L2: make descriptor, maturity, and policy satisfy the phase descriptor contract |
| `provider_contract` | L0 Contract unproven → L3 Contract clean | Provider contract maturity is clean | L1→L2: serve contract-valid responses with correct identity and metrics |

## What each finding means

Each finding caps the capability it names at the rung below its impact; only ERROR/BLOCKER severities fail the phase, so the skeleton/north-star/ladder-coverage checks are advisory (WARNING) debt during the contract rollout.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `PROVIDER_DESCRIPTOR_MISSING` / `PROVIDER_DESCRIPTOR_INVALID` | provider_descriptor | L1 | ERROR | Yes |
| `PROVIDER_IDENTITY_MISMATCH` / `PROVIDER_MATURITY_INVALID` | provider_descriptor | L1 | ERROR | Yes |
| `PROVIDER_STALE_MATURITY_FILE` / `PROVIDER_POLICY_UNSAFE` | provider_descriptor | L1 | ERROR | Yes |
| `PROVIDER_DOCS_MISSING` / `PROVIDER_DOCS_SKELETON_INCOMPLETE` | provider_descriptor | L2 | WARNING | No |
| `PROVIDER_NORTH_STAR_MISSING` / `PROVIDER_LADDER_INCOMPLETE` | provider_descriptor | L2 | WARNING | No |
| `PROVIDER_UNREACHABLE` | provider_contract | L1 | WARNING | No |
| `PROVIDER_CONTRACT_INVALID` / `PROVIDER_CONTRACT_IDENTITY_MISMATCH` / `PROVIDER_METRICS_MISSING` | provider_contract | L1 | ERROR | Yes |
| `PROVIDER_AUTOFIX_DECLARATION_INCOMPLETE` | provider_contract | L2 | WARNING | No |

## The canonical fix

- **Descriptor-structure findings** (`PROVIDER_DESCRIPTOR_MISSING/INVALID`, `PROVIDER_IDENTITY_MISMATCH`, `PROVIDER_MATURITY_INVALID`) → author or repair `.vrooli/test-genie.json` so it parses, matches the schema version, stamps the owning scenario's identity, and carries a well-formed embedded `maturity` block.
- **Hygiene findings** (`PROVIDER_STALE_MATURITY_FILE`, `PROVIDER_POLICY_UNSAFE`) → delete the retired `.vrooli/maturity.json` once the descriptor block is the source of truth, and choose a non-contradictory policy combination.
- **Contract-doc findings** (`PROVIDER_DOCS_MISSING`, `PROVIDER_DOCS_SKELETON_INCOMPLETE`, `PROVIDER_NORTH_STAR_MISSING`, `PROVIDER_LADDER_INCOMPLETE`) → declare `docs.path`, fill the five required skeleton headings, give the ladder's top rung a North Star aspiration, and give every non-top rung a `next_unlock`.
- **Live-contract findings** (`PROVIDER_CONTRACT_INVALID`, `PROVIDER_CONTRACT_IDENTITY_MISMATCH`, `PROVIDER_METRICS_MISSING`, `PROVIDER_AUTOFIX_DECLARATION_INCOMPLETE`) → fix provider-side response construction so assessments are contract-valid, identity-stamped, metrics-bearing, and every finding declares a `fix_class`. `PROVIDER_UNREACHABLE` is environmental — start the provider.

## How to verify

```bash
# See the current rung, gaps, and next move for both capabilities:
test-genie provider-contract check provider-conformance <scenario> --json

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases provider-conformance
test-genie runs findings --scenario <scenario>
```

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
