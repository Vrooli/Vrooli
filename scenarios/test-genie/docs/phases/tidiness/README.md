# Tidiness Phase

The `tidiness` phase delegates maintainability checks to the **tidiness-manager** scenario and maps the returned findings into Test Genie findings. It is a **finding producer** (source=`tidiness`): its output feeds the ecosystem-manager `tidiness` dimension.

Static-quality contracts, lint policy, type policy, and config strictness are owned by the `quality` phase through **quality-health**.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

The scenario's source stays **effortlessly maintainable**: files remain within reviewable size boundaries with sparse technical-debt markers, functions stay below high-complexity thresholds and modules below broad-coupling thresholds, and no duplicated implementation blocks let fixes or behavior changes drift across copies. At maximum maturity tidiness-manager returns a clean maintainability scan in which local debt, complexity/coupling, and duplication are all clean — so a fix applied once holds everywhere and any file can be read and changed with low cognitive load.

## The rungs and their gates

tidiness-manager reports a monotone ladder per capability (each rung implies the one below).

| Capability | Ceiling | L1 → next unlock | Top-rung ("clean") aspiration |
|---|---|---|---|
| Scan Contract | L2 | Normalized scan output → structured findings + summary + maturity metadata | The maintainability scan contract is clean. |
| Local Debt Control | L3 | Debt visible → files within size/marker thresholds → no long-file/debt findings | Local maintainability debt is clean. |
| Complexity And Coupling | L3 | Risks visible → complexity/coupling below blocking thresholds → none remain | Complexity and coupling are clean. |
| Duplication Control | L3 | Duplication visible → duplicated-code below thresholds → none remain | Duplication posture is clean. |

## What each finding means

Every tidiness finding is `WARNING`/advisory — none fail the phase; they surface honest maintainability debt fed to the ecosystem-manager `tidiness` dimension. Each caps its capability at L2 (below the clean top rung).

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `LONG_FILE` | local_debt_control | L2 | WARNING | No |
| `TECH_DEBT_MARKERS` | local_debt_control | L2 | WARNING | No |
| `HIGH_COMPLEXITY` | complexity_coupling | L2 | WARNING | No |
| `HIGH_COUPLING` | complexity_coupling | L2 | WARNING | No |
| `DUPLICATED_CODE` | duplication_control | L2 | WARNING | No |

## The canonical fix

- **`LONG_FILE`** → split the file along its natural seams into smaller, single-responsibility units (skills: `tidiness`, `cognitive-load-reduction`).
- **`TECH_DEBT_MARKERS`** → resolve or clarify the `TODO`/`FIXME`/`HACK` markers; convert real work into tracked items rather than inline debt (skills: `tidiness`, `intent-clarification`).
- **`HIGH_COMPLEXITY`** → reduce branching by extracting helpers and flattening control flow; add tests around the seam first (skills: `tidiness`, `cognitive-load-reduction`, `test`).
- **`HIGH_COUPLING`** → narrow the import surface; enforce a clean boundary of responsibility for the module (skills: `tidiness`, `boundary-of-responsibility-enforcement`).
- **`DUPLICATED_CODE`** → unify the duplicated blocks behind a shared util so a fix lands once (skills: `tidiness`, `utils-unification`).

## How to verify

```bash
# Current rung, gaps, and next move for every capability:
tidiness-manager validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases tidiness
test-genie runs findings --scenario <scenario>
```

## How It Runs

Test Genie resolves the `tidiness-manager` API via `api-core/discovery`, then calls the shared validation RPC:

```
scenario-validation/v1.ScenarioValidationService.ValidateScenario
{ "scenario": "<scenario>" }
```

Test Genie reads the shared `status` and `assessment.findings` fields only. Tidiness-manager packs its native maintainability summary and category breakdown into `native_detail` for its own CLI/UI.

## Skip Behaviour (not a failure)

`tidiness` is an optional external dependency. If tidiness-manager is not reachable, the phase **skips with a clear message** and does not fail the suite — mirroring the runnability-gate pattern. Start tidiness-manager to enable it.

## Opt-Out

```bash
test-genie execute <scenario> --skip tidiness
```
