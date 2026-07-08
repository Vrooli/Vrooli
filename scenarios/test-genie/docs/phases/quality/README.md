# Quality Phase

**ID**: `quality`  
**Timeout**: 120 seconds  
**Required**: Yes  
**Provider**: `quality-health`

The quality phase delegates static quality validation to the Quality Health scenario. It replaces the old native `lint` phase; Test Genie no longer owns component-specific lint/type command heuristics.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

The scenario's static-quality posture is **fleet-ready**: every discovered surface is covered by a rule pack or an explicit non-applicability decision, TypeScript and JavaScript surfaces hold their strict-compiler / typed-lint / safety-rule / build-typecheck baselines, Go surfaces hold local-module and golangci-lint baselines with justified-suppression visibility, and scenario-level Test Genie and Makefile gates run real strict lint/type/shell/format checks. At maximum maturity the posture is clean enough for Test Genie delegation, cross-scenario fleet comparison, and reusable policy indexing.

## The rungs and their gates

Quality Health reports a ladder per capability. The rungs are monotone — each implies the one below.

| Capability | Rungs | Top rung (aspiration) | Key next unlock |
|---|---|---|---|
| Surface Quality Coverage | L0–L2 | Static-quality coverage is clean across discovered surfaces. | Every discovered surface is covered by a rule pack or explicit non-applicability. |
| TypeScript Quality | L0–L3 | TypeScript quality is clean for applicable surfaces. | Strict compiler, typed lint, safety rules, and build typecheck, then clean checks. |
| Go Quality | L0–L3 | Go quality is clean for applicable surfaces. | `go.mod`, golangci-lint config, required linters, justified suppressions, then clean checks. |
| Scenario Quality Gates | L0–L3 | Scenario quality gates are clean. | Strict Test Genie lint policy + Makefile gates + shell visibility, then clean checks. |
| Fleet Static Quality | L0–L2 | Static quality is fleet-ready. | No Quality Health findings block fleet validation. |

## What each finding means

Each finding caps the capability it names at a rung; only ERROR/BLOCKER severities fail the phase, so WARNING/INFO findings are honest, non-failing debt.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `TS_CONFIG_STRICT` | typescript_quality | L2 | ERROR | Yes |
| `ESLINT_SAFETY_RULES` | typescript_quality | L2 | ERROR | Yes |
| `ESLINT_TYPED_CONFIG` | typescript_quality | L2 | ERROR | Yes |
| `NODE_BUILD_TYPECHECK` | typescript_quality | L2 | ERROR | Yes |
| `TS_DANGEROUS_PATTERNS` | typescript_quality | L2 | WARNING | No |
| `GO_MOD_PRESENT_FOR_API_OR_CLI` | go_quality | L2 | ERROR | Yes |
| `GO_LINT_CONFIG_PRESENT` | go_quality | L2 | ERROR | Yes |
| `TESTING_CONFIG_LINT_STRICT` | scenario_quality_gates | L2 | ERROR | Yes |
| `MAKEFILE_QUALITY_GATES` | scenario_quality_gates | L2 | WARNING | No |
| `QUALITY_COVERAGE_GAP` | surface_quality_coverage | L1 | INFO | No |

## The canonical fix

- **TypeScript config/lint findings** (`TS_CONFIG_STRICT`, `ESLINT_SAFETY_RULES`, `ESLINT_TYPED_CONFIG`, `NODE_BUILD_TYPECHECK`) → these are `fix_class: auto`; preview and apply the deterministic config repair with `quality-health fix-config run <scenario> --dry-run` then `quality-health fix-config apply <scenario>`. Never weaken the strict settings to pass.
- **Dangerous-pattern findings** (`TS_DANGEROUS_PATTERNS`, `GO_DANGEROUS_PATTERNS`) → manual: source-semantic suppressions require human intent; remove the suppression and fix the underlying code, or add a written reason where genuinely required.
- **Go findings** (`GO_MOD_PRESENT_FOR_API_OR_CLI`, `GO_LINT_CONFIG_PRESENT`, `GO_LINT_REQUIRED_LINTERS`) → auto-fixable config repair; add the local `go.mod`, golangci-lint baseline config, and required linters.
- **Scenario-gate findings** (`TESTING_CONFIG_LINT_STRICT`, `MAKEFILE_QUALITY_GATES`, `SHELL_SYNTAX_LINT`) → declare strict `.vrooli/testing.json` lint policy and Makefile quality gates; fix reported shell syntax by hand.
- **Coverage-gap findings** (`QUALITY_COVERAGE_GAP`) → a design decision: add a rule-pack contract for the uncovered surface or record an explicit non-applicability decision.

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
quality-health validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases quality
test-genie runs findings --scenario <scenario>
```

## What It Checks

Quality Health audits the target scenario's discovered surfaces and reports findings for:

- strict TypeScript configuration and required guardrail comments
- ESLint safety rules, typed linting setup, and required guardrail comments
- dangerous TypeScript suppression patterns
- Node build scripts that must run type checking before bundling
- strict `.vrooli/testing.json` lint policy contracts
- Go module and golangci-lint configuration
- Makefile quality gates

The phase maps Quality Health findings into Test Genie's `standards` finding channel so ecosystem-manager and maturity guidance can treat static-quality failures as standards work.

## Execution

Test Genie calls the provider through the shared validation RPC:

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

Quality Health packs its native `AuditQualityResponse` into `native_detail` for its own CLI/UI. Warnings are reported as warnings. Error findings fail the phase. If the Quality Health API is unavailable, the phase fails as a missing dependency because this is now the canonical static-quality producer.

## Troubleshooting

Run the provider directly for full detail:

```bash
quality-health audit run <scenario> --json
quality-health explain finding <finding-id> --scenario <scenario>
```

Use Quality Health's fix preview before applying deterministic config repairs:

```bash
quality-health fix-config run <scenario> --dry-run
quality-health fix-config apply <scenario>
```

Do not weaken lint/type settings to make this phase pass. The guarded config comments are part of the contract and are intentionally checked so agents fix source code instead of loosening validation.
