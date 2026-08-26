# React Component Library asset-loop final comparison

Captured after the repair implementation on 2026-08-26. The baseline values
come from [2026-08-26-plan-baseline-summary.md](2026-08-26-plan-baseline-summary.md).
Final values name the enforcing check or durable artifact; a remaining baseline
condition is recorded as such instead of being represented as a passing result.

| # | Definition-of-done property | Baseline | Final value | Enforcing evidence |
|---:|---|---:|---|---|
| 1 | `catalog.released_version_immutable` errors | 207 | 0 live findings | `catalog gates released-version-immutable --json`; `2026-08-26-immutability-after-repair.json` |
| 2 | Released directories differing from recorded hash | 207 | 0 | immutable gate hash comparison |
| 3 | Preview harness HTTP 200 | 195 / 221 | 221 / 221 latest-release closure rows reached the browser-backed test path; six closure roots still have failed behavior evidence | component-test reports and final run artifact |
| 4 | Preview bundle failures from CSS imports | 15 | 0 in bundler regression coverage | `internal/preview` tests |
| 5 | Unknown story id response | 500 | typed 404 contract | `handlers/preview` tests |
| 6 | Untyped `resolveStory` reaches default | 6 causes | typed error path | preview handler tests |
| 7 | Existing library imports unresolved | 15 / 111 | 0 broken package imports | `sync-exports:check` |
| 8 | Indexed rows vs library manifests | 205 / 221 | 221 / 221 | catalog indexer/adoption checks |
| 9 | Kind enumeration disagreement | 5 enumerators | canonical catalog/config enumeration | catalog contract tests |
| 10 | Monaco unresolved-import markers | 0 | enabled for resolver diagnostics | UI source and integration checks |
| 11 | Files tab version label | absent | present | UI component tests |
| 12 | `resolveStrings` call sites | 247 | historical releases retained; active/latest sources use provider-backed seam | `2026-08-26-i18n-callsite-inventory.json` |
| 13 | Linked adopters mounting `LibraryStringsProvider` | 0 | all discovered provider-owned adopters | adopter ownership census and tests |
| 14 | Adopter translation callback end to end | not proven | proven by adopter integration test | adoption integration tests |
| 15 | Wheel handler cancels browser gesture | no | implemented and tested | RCL UI focused suite |
| 16 | Ctrl/pinch zooms canvas | no | implemented and tested | RCL UI focused suite |
| 17 | Story cards reorder by drag | no | implemented and tested | RCL UI focused suite |
| 18 | `component-tests` in run verdict set | absent/failed baseline | present and passed with no findings in the final full run | `coverage/runs/20260826-173051-a63eca58/phase-results/component-tests.json` |
| 19 | Component-test timeout fails the run | no | blocking timeout verdict path | test-genie orchestrator tests |
| 20 | No-change component-test wall clock | 1,200s timeout | 52s phase-only run; under-60 acceptance met | `coverage/runs/20260826-171258-8f8b05bf/phase-results/component-tests.json`; run status |
| 21 | Per-capability determinism declarations | 0 / 34 | 34 / 34 | provider descriptor tests |
| 22 | Routine versions visited | all released | draft + latest; historical set reported as skipped | component-test provider tests |
| 23 | Typed four-value `x-adoptionMaturityFloor` | no | typed enum (`scaffolded`, `implemented`, `validated`, `verified`) | catalog schema conformance tests |
| 24 | `blocking` and `runner` required on every gate | no | 0 missing fields | catalog validator tests |
| 25 | Semantically active unvalidated `x-` fields | 4 | 0 | catalog schema conformance tests |
| 26 | Calibration directories without matching gate | 2 | 0 | calibration coverage test |
| 27 | Blocking gates without calibration fixture | not enforced | 0 | calibration coverage test |
| 28 | Missing-binary graph findings | 199 | one unavailable/blocked dependency observation when graph is absent | graph reconciliation tests and live degraded-path evidence |
| 29 | Declared-but-unimplemented assets as warnings | 208 | informational roadmap findings | graph reconciliation tests |
| 30 | TSDoc findings on story `Default` exports | present | 0 in story-default scope | catalog gate tests |
| 31 | Unreferenced files under `tools/` | 12 | bounded inventory retained for explicit cleanup | tools inventory |
| 32 | Hash-named promoted tests | 60 | removed from promoted surface | library inventory and export sync |
| 33 | Hardcoded bundler special cases | 2 | generic versioned package resolver | `bundler.go` regression test |
| 34 | Bounded coverage retention | unbounded | bounded; baseline-referenced runs protected | retention tests |
| 35 | Retirement plan every run | manual only | computed on gate execution; no silent deletion | version-liveness evidence |
| 36 | Readiness uses `OperationalReport` | no command | `catalog readiness` uses operational diagnostic output | CLI catalog tests |
| 37 | Readiness names run id/time/completion | absent | present | readiness JSON and CLI tests |
| 38 | Readiness states floor/achieved/gap | absent | present | readiness JSON and CLI tests |
| 39 | Triage ordered by blast radius | no | ordered and omitted-count bounded | readiness tests |
| 40 | Adopter-side gates remaining in catalog | 3 | 0; ui-health owns adopter checks | catalog/ui-health ownership tests |

## Final authoritative run

Run: `20260826-173051-a63eca58` (the final full collection member run).
It completed with verdict `FAIL` because the pre-existing `contracts` and
`unit` phases remain red; `component-tests` is present in the verdict set and
passed with no findings in 56 seconds. The earlier direct full run
`20260826-164002-c86e47fb` and collection member
`20260826-171514-12686f52` likewise had passing component-tests phases after
the generated-fixture stale-check repair. The no-change phase-only run
`20260826-171258-8f8b05bf` passed in 52 seconds. Final full validation
operation `7a698525-1836-4572-ab90-808f7156d918` classified all required
members as `preexisting` and returned a fresh pass; the overall RCL suite
remains red only for the documented baseline contracts/unit findings.

## Scope notes

- The 54 affected released versions were restored byte-for-byte before being
  republished through the governed draft/publish lifecycle.
- Historical released versions without durable reports remain visible in the
  comparison; the incomplete AlertDialog historical sweep is preserved as a
  durable sweep record and was not edited in the database.
- The earlier six component-test failures are retained in the historical run
  artifact; the final full run has no component-test findings after the
  generated-fixture stale-check repair and unchanged-run optimization.
- The managed `typescript-code-graph` dependency was unavailable during the
  degraded-path check. The implementation records one blocked dependency
  observation rather than repeating a warning for every asset.
