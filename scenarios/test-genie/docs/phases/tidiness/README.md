# Tidiness Phase

The `tidiness` phase delegates maintainability checks to the **tidiness-manager** scenario and maps the returned findings into Test Genie findings. It is a **finding producer** (source=`tidiness`): its output feeds the swarm-manager `tidiness` dimension.

Static-quality contracts, lint policy, type policy, and config strictness are owned by the `quality` phase through **quality-health**.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

The scenario's source stays **effortlessly maintainable**: files remain within reviewable size boundaries with sparse technical-debt markers, functions stay below high-complexity thresholds and modules below broad-coupling thresholds, and no duplicated implementation blocks let fixes or behavior changes drift across copies. At maximum maturity tidiness-manager returns a clean maintainability scan in which local debt, complexity/coupling, and duplication are all clean — so a fix applied once holds everywhere and any file can be read and changed with low cognitive load.

## The rungs and their gates

tidiness-manager reports a monotone ladder per capability (each rung implies the one below).

| Capability | Ceiling | L1 → next unlock | Top-rung ("clean") aspiration |
|---|---|---|---|
| Scan Contract | L2 | Normalized scan output → structured findings + summary + maturity metadata | The maintainability scan contract is clean. |
| Local Debt Control | L4 | Debt visible → controlled → open work → no opportunity findings | Local maintainability debt is clean. |
| Complexity And Coupling | L4 | Risks visible → controlled → open work → no opportunity findings | Complexity and coupling are clean. |
| Duplication Control | L4 | Duplication visible → controlled → open refactor opportunities → no opportunity findings | Duplication posture is clean. |

## What each finding means

Natural tidiness findings are WARNING/INFO findings: they do not fail the
phase, but opportunity findings block L4 and render the capability as **Open
work**, never clean or Complete. Structural and incidental duplication remains
visible at zero debt. The only ERROR and phase failure is
`TIDINESS_BUDGET_EXCEEDED`, emitted for an explicit budget breach, a debt
regression against a recorded baseline, or an attempted ratchet loosening.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `LONG_FILE` | local_debt_control | L3 | WARNING | No |
| `TECH_DEBT_MARKERS` | local_debt_control | L3 | WARNING | No |
| `HIGH_COMPLEXITY` | complexity_coupling | L3 | WARNING | No |
| `HIGH_COUPLING` | complexity_coupling | L3 | WARNING | No |
| `DUPLICATED_CODE` | duplication_control | L3 | WARNING | No |
| `DUPLICATED_BOILERPLATE` | duplication_control | L4 | INFO | No — visible, zero debt |
| `TIDINESS_BUDGET_EXCEEDED` | duplication_control | L2 | ERROR | Yes |

## The canonical fix

- **`LONG_FILE`** → split the file along its natural seams into smaller, single-responsibility units (skills: `tidiness`, `cognitive-load-reduction`).
- **`TECH_DEBT_MARKERS`** → resolve or clarify the `TODO`/`FIXME`/`HACK` markers; convert real work into tracked items rather than inline debt (skills: `tidiness`, `domain-clarity`).
- **`HIGH_COMPLEXITY`** → reduce branching by extracting helpers and flattening control flow; add tests around the seam first (skills: `tidiness`, `cognitive-load-reduction`, `test`).
- **`HIGH_COUPLING`** → narrow the import surface; enforce a clean boundary of responsibility for the module (skills: `tidiness`, `boundary-of-responsibility-enforcement`).
- **`DUPLICATED_CODE`** → use the producer-ranked opportunity and its line-debt weight to extract the shared behavior once (skills: `tidiness`, `utils-unification`).
- **`DUPLICATED_BOILERPLATE`** → structural or incidental repetition; keep it visible, but it carries no refactor debt.
- **`TIDINESS_BUDGET_EXCEEDED`** → reduce the named measured metric or set a truthful, tighter budget.

## How to verify

```bash
# Current rung, gaps, and next move for every capability:
test-genie execute <scenario> --phases tidiness

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases tidiness
test-genie runs findings --scenario <scenario>
```

An opt-in budget lives in the target scenario's `.vrooli/testing.json`:

```json
{"phases":{"tidiness":{"budgets":{"duplication_line_debt":250,"baseline_duplication_line_debt":250,"ratchet":true}}}}
```

`duplication_line_debt` is not a detector count. It is the largest physical
span multiplied by extra copies, with a documented multiplier for
high-leverage cross-package opportunities. See [Tidiness debt semantics](../../../../tidiness-manager/docs/reference/debt-semantics.md).

## Duplication result classes

Tidiness Manager keeps every normalized group visible and assigns one class:

- `structural` — uniform declarations or wiring; zero debt.
- `incidental` — short local repetition without demonstrated extraction value; zero debt.
- `opportunity` — refactor-worthy duplicate; line-weighted debt.
- `high-leverage` — long, cross-package duplicate; weighted line debt and ranked first on ties.

The assessment presentation names the top producer-ranked duplication
opportunities. Raw locations remain in the native finding evidence for review.

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
