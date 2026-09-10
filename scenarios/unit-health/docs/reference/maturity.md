# Test Maturity Reference — Unit Health

Unit Health reports a **provider-local** test-maturity assessment for a scenario.
It never reports the scenario's overall (global) maturity — `maturity-go` owns
that. The local ladders and the finding→level mapping are declared in the
`maturity` block of [`.vrooli/test-genie.json`](../../.vrooli/test-genie.json)
(spec version 2.0.0) and validated by `packages/maturity-go/assessment`. The
`maturity_spec_test.go` anti-drift test holds the code's severity map and the
spec's finding map equal in both directions.

This file is the human-readable companion to that machine-readable spec. When
they disagree, the `maturity` block of `.vrooli/test-genie.json` is the source
of truth. There is no separate `.vrooli/maturity.json`; earlier drafts of this
document and the PRD referred to one, and that reference is retired.

## Local Maturity Ladder (L0–L5)

The heading keeps its historical name; the assessment is no longer a single
L0–L5 ladder. It is a set of capability ladders, each monotone, each capped by
the findings that map to it. The reported local level is the lowest capability
level that still has a **required** finding open; the assessment names that
capability as `highest_priority_capability` together with its `next_unlock`.

| Capability | Rungs | Top rung means |
|---|---|---|
| `surface_discovery` | L0 No reliable test surface → L1 Test surfaces discovered → L2 Discovery clean | Code Facts found every workspace and its parse units without an unsupported unit |
| `execution_readiness` | L0 Execution unavailable → L1 Commands runnable → L2 Execution clean | Planned commands ran to a classified result inside timeout and watchdog |
| `framework_config` | L0 Framework missing → L1 Framework configured → L2 Canonical and coverage-capable → L3 Framework config clean | Canonical runner, coverage config, and the policy profile are honored without weakening |
| `test_architecture` | L0 Architecture contracts unavailable → L1 Architecture comparable → L2 Layout aligned → L3 Architecture clean | Tests are co-located, share a test-utility root, never import helpers from production, and ambient dependencies sit behind seams |
| `coverage_quality` | L0 Coverage unavailable → L1 Coverage measured → L2 Quality depth clean | Coverage artifacts parse and the advisory depth findings are clean |
| `stability_traceability` | L0 History unavailable → L1 History measured → L2 Stable and linked | No flake suspicion, no runtime growth, requirements linked to tests |

**Enforced versus advisory.** Each finding carries `clean_requirement`:
`required` findings block their capability's next rung and, at ERROR severity,
fail the Test Genie `unit` phase; `advisory` findings are measured and reported
but never block. The split lives in the spec, not in prose; read the table
below for the current mapping.

## Global Semantic Impact

Each finding also carries a stable `global_impact` from the shared vocabulary in
`packages/proto/schemas/common/v1/maturity.proto`. Unit Health never names R0/R1
rungs directly; `maturity-go` maps these semantic impacts to the global ladder.

| Impact | Meaning for test findings |
|---|---|
| `foundation_blocker` | No reliable test surface, tests do not run at all, or no framework. |
| `safety_blocker` | Tests hang or stall (no-output watchdog), risking the host. |
| `evolvability_gap` | Noncanonical framework, ungoverned surface, missing seams or test utilities, reimplemented shared seams. |
| `hardening_gap` | Missing coverage config, weakened policy, invalid waiver, or native projection drift. |
| `advisory` | Coverage depth, skip/only, assertion, edge-case, flake, runtime growth, and requirement-link signals. Measured, never gating. |

## Finding Codes

Every code in the spec maps to a capability level, a global impact, a dimension,
a default severity, and a clean requirement. The assessor fails closed when an
unmapped code is emitted.

| Code | Capability level | Global impact | Dimension | Severity | Clean requirement |
|---|---|---|---|---|---|
| `TEST_SURFACE_ABSENT` | L0 | foundation_blocker | tests | error | required |
| `UNSUPPORTED_PARSE_UNIT` | L1 | advisory | tests | info | advisory |
| `UNSUPPORTED_TARGET_KIND` | L0 | foundation_blocker | tests | error | required |
| `UNIT_REQUIRED_ROLE_MISSING` | L0 | foundation_blocker | tests | error | required |
| `UNIT_SURFACE_UNGOVERNED` | L1 | evolvability_gap | tests | warning | required |
| `UNIT_TEST_KIND_OUT_OF_SCOPE` | L1 | advisory | tests | warning | required |
| `TEST_EXECUTION_FAILURE` | L2 | foundation_blocker | tests | error | required |
| `TEST_DEPENDENCY_MISSING` | L0 | foundation_blocker | tests | error | required |
| `TEST_TIMEOUT_HANG` | L2 | safety_blocker | tests | error | required |
| `UNIT_POLICY_PROFILE_INVALID` | L0 | foundation_blocker | tests | error | required |
| `UNIT_POLICY_WEAKENED` | L2 | hardening_gap | coverage | error | required |
| `UNIT_POLICY_WAIVER_INVALID` | L1 | hardening_gap | tests | error | required |
| `UNIT_POLICY_PROJECTION_DRIFT` | L2 | hardening_gap | tests | error | required |
| `TEST_FRAMEWORK_MISSING` | L0 | foundation_blocker | tests | error | required |
| `TEST_FRAMEWORK_NONCANONICAL` | L2 | evolvability_gap | tests | error | required |
| `COVERAGE_CONFIG_MISSING` | L2 | hardening_gap | coverage | error | required |
| `PACKAGE_MANAGER_MISMATCH` | L3 | evolvability_gap | tests | warning | required |
| `TEST_MISCONFIGURATION` | L1 | hardening_gap | tests | warning | required |
| `TEST_NOT_COLOCATED` | L2 | evolvability_gap | tests | warning | required |
| `TEST_UTIL_MISSING` | L2 | evolvability_gap | tests | warning | required |
| `TEST_HELPER_FROM_PRODUCTION` | L3 | hardening_gap | tests | error | required |
| `MISSING_INJECTABLE_SEAM` | L3 | evolvability_gap | tests | warning | required |
| `SEAM_DUPLICATED_IN_PACKAGE` | L3 | evolvability_gap | tests | error | required |
| `SEAM_REIMPLEMENTED` | L3 | evolvability_gap | tests | error | required |
| `COMPANION_REIMPLEMENTED` | L3 | evolvability_gap | tests | error | required |
| `COMPANION_AVAILABLE` | L3 | evolvability_gap | tests | info | advisory |
| `LOW_COVERAGE` | L2 | advisory | coverage | warning | advisory |
| `COVERAGE_ABSENT` | L0 | advisory | coverage | info | advisory |
| `TEST_EXCESSIVE_SNAPSHOTS` | L2 | advisory | tests | info | advisory |
| `TEST_FLAKE_SUSPECTED` | L2 | advisory | tests | warning | advisory |
| `TEST_RUNTIME_GROWTH` | L2 | advisory | tests | info | advisory |
| `TEST_SKIPPED_OR_ONLY` | L2 | advisory | tests | warning | advisory |
| `TEST_UNTAGGED_REQUIREMENT` | L2 | advisory | tests | warning | advisory |

**Typed quality rollups.** `TEST_SKIPPED_OR_ONLY` is emitted once per workspace
when typed `focused-test` or `skip-declaration` results contain a violation.
`TEST_UNTAGGED_REQUIREMENT` is emitted once per workspace for a typed
`requirement-link` violation. Three non-observable legacy quality codes were
removed on 2026-09-09 because static syntax cannot establish those claims. The typed
catalog remains the source of truth for per-test `violation`, `checked_clean`,
`unknown`, and `not_applicable` results.

Unmapped codes fall back to the spec's `fallback` block.

## Dimensions

Findings carry a `dimension` so `maturity-go` can route them. Unit Health emits
only `tests` and `coverage` (the dimensions it owns after Test Genie's
`unit` and `coverage` phases collapsed into one delegated `unit` phase).

## Cross-references

- [`.vrooli/test-genie.json`](../../.vrooli/test-genie.json) — the `maturity` block is the machine-readable spec
- [`test-quality-rules.md`](test-quality-rules.md) — the per-test rule catalog and its promotion prerequisites
- [`cli-commands.md`](cli-commands.md) — how to read the assessment from the CLI
- [`api-endpoints.md`](api-endpoints.md) — `ValidateScenario` response shape
- `packages/maturity-go/assessment` — the shared validator/aggregator
