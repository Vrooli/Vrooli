# Test Maturity Reference — Unit Health

Unit Health reports a **provider-local** test-maturity rung for a scenario.
It never reports the scenario's overall (global) maturity — `maturity-go` owns
that. The local ladder and the finding→level mapping are declared in
[`.vrooli/maturity.json`](../../.vrooli/maturity.json) and validated by
`packages/maturity-go/assessment`.

This file is the human-readable companion to that machine-readable spec. When
they disagree, `.vrooli/maturity.json` is the source of truth.

## Local Maturity Ladder (L0–L5)

**Enforced gates vs advisory tiers.** L0–L3 are **enforced gates**: each maps to
ERROR-severity findings that block local maturity until cleared. L4–L5 are
**advisory tiers** — they are *measured and reported but never gate*. Every L4/L5
finding is WARNING/INFO and carries `global_impact: advisory`, so a scenario that
is clean through L3 is reported at the advisory tier; the L4/L5 findings guide
hardening but do not hold back the level. This split is enforced by the
`TestMaturityLadderGateAdvisorySplit` anti-drift test.

| Level | Kind | Name | A scenario reaches this level when… |
|---|---|---|---|
| L0 | gate | No reliable test surface | The target cannot be resolved or has no discoverable testable workspace. |
| L1 | gate | Test surfaces discovered | Every discovered workspace has a runnable test command or an explicit gap finding, and no test execution fails or hangs. |
| L2 | gate | Canonical test frameworks configured | Each workspace uses its canonical framework (Go `go test`, React/Vite Vitest, Python pytest, Bash bats) with a coverage-capable config; no missing/noncanonical-framework or missing-coverage-config error. |
| L3 | gate | Testable architecture and shared test utilities | Tests are co-located, share test utilities, never import helpers from production, and exercise injectable seams; no test-architecture error. |
| L4 | advisory | Coverage and edge-case depth | Advisory: per-file coverage, edge-case presence, and assertion strength are measured as non-blocking warnings. |
| L5 | advisory | Drift-gated, flake-aware, requirement-linked | Advisory: cross-run flake, runtime growth, and requirement linkage are measured as non-blocking signals informing fleet-readiness. |

The **current level** is the highest enforced gate (L0–L3) with no blocking
finding below it; once L3 is clean the scenario reaches the advisory tier. The
assessment lists exactly which findings block the next gate. L4/L5 findings are
reported for guidance but never appear as blocking codes.

## Global Semantic Impact

Each finding also carries a stable `global_impact` from the shared vocabulary in
`packages/proto/schemas/common/v1/maturity.proto`. Unit Health never names R0/R1
rungs directly; `maturity-go` maps these semantic impacts to the global ladder.

| Impact | Meaning for test findings |
|---|---|
| `foundation_blocker` | No reliable test surface / tests don't run at all / no framework. |
| `safety_blocker` | Tests hang or stall (no-output watchdog), risking the host. |
| `evolvability_gap` | Noncanonical framework, missing seams/test-utils. |
| `hardening_gap` | Missing coverage config (an enforced L2 gate). |
| `advisory` | All L4/L5 signals: per-file/missing coverage, weak assertions, skipped/render-only tests, missing edge cases, flake, runtime growth, untagged requirements, snapshot overuse, unsupported parse units. These are measured, never gating. |

## Finding Codes

Every code Unit Health emits is mapped in `.vrooli/maturity.json` (the assessor
fails closed if an unmapped code is emitted).

| Code | Level | Global impact | Dimension | Default severity |
|---|---|---|---|---|
| `TEST_SURFACE_ABSENT` | L0 | foundation_blocker | tests | error |
| `UNSUPPORTED_PARSE_UNIT` | L1 | advisory | tests | info |
| `TEST_EXECUTION_FAILURE` | L1 | foundation_blocker | tests | error |
| `TEST_DEPENDENCY_MISSING` | L1 | foundation_blocker | tests | error |
| `TEST_TIMEOUT_HANG` | L1 | safety_blocker | tests | error |
| `TEST_FRAMEWORK_MISSING` | L2 | foundation_blocker | tests | error |
| `TEST_FRAMEWORK_NONCANONICAL` | L2 | evolvability_gap | tests | error |
| `COVERAGE_CONFIG_MISSING` | L2 | hardening_gap | coverage | error |
| `PACKAGE_MANAGER_MISMATCH` | L2 | evolvability_gap | tests | warning |
| `TEST_MISCONFIGURATION` | L2 | hardening_gap | tests | warning |
| `TEST_NOT_COLOCATED` | L3 | evolvability_gap | tests | warning |
| `TEST_UTIL_MISSING` | L3 | evolvability_gap | tests | warning |
| `TEST_HELPER_FROM_PRODUCTION` | L3 | hardening_gap | tests | error |
| `MISSING_INJECTABLE_SEAM` | L3 | evolvability_gap | tests | warning |
| `LOW_COVERAGE` | L4 | advisory | coverage | warning |
| `COVERAGE_ABSENT` | L4 | advisory | coverage | info |
| `TEST_SKIPPED_OR_ONLY` | L4 | advisory | tests | warning |
| `TEST_NO_ASSERTION` | L4 | advisory | tests | warning |
| `TEST_RENDER_ONLY` | L4 | advisory | tests | warning |
| `TEST_EXCESSIVE_SNAPSHOTS` | L4 | advisory | tests | info |
| `TEST_MISSING_EDGE_CASES` | L4 | advisory | tests | warning |
| `TEST_FLAKE_SUSPECTED` | L5 | advisory | tests | warning |
| `TEST_RUNTIME_GROWTH` | L5 | advisory | tests | info |
| `TEST_UNTAGGED_REQUIREMENT` | L5 | advisory | tests | warning |

Unmapped codes fall back to the `fallback` block (`L1` / `hardening_gap` /
`tests` / `warning`).

## Dimensions

Findings carry a `dimension` so `maturity-go` can route them. Unit Health emits
only `tests` and `coverage` (the two dimensions it owns after Test Genie's
`unit` + `coverage` phases collapse into one delegated `unit` phase).

## Cross-references

- [`.vrooli/maturity.json`](../../.vrooli/maturity.json) — machine-readable spec
- [`cli-commands.md`](cli-commands.md) — how to read the assessment from the CLI
- [`api-endpoints.md`](api-endpoints.md) — `ValidateScenario` response shape
- `packages/maturity-go/assessment` — the shared validator/aggregator
