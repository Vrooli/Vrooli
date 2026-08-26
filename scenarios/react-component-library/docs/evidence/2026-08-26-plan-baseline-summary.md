# React Component Library asset-loop baseline

Captured 2026-08-26 before implementation changes. The gate and suite artifacts
are the authoritative machine-readable records; live probes are retained beside
this summary.

| # | Definition-of-done property | Baseline value | Evidence |
|---:|---|---|---|
| 1 | `catalog.released_version_immutable` errors | 207 | baseline gates |
| 2 | Released version directories differing from recorded hash | 207 | baseline gates |
| 3 | Preview harness HTTP 200 | 195 / 221 | baseline preview sweep |
| 4 | Preview bundle failures from CSS imports | 15 | baseline preview sweep |
| 5 | Unknown story id response | 500 | live preview contract |
| 6 | Untyped `resolveStory` causes reaching default | 6 causes | preview handler inspection |
| 7 | Existing library imports reported unresolved | 15 / 111 | baseline import resolution |
| 8 | Indexed rows vs library manifests | 205 / 221 | baseline index |
| 9 | Indexer/test-genie/ui-health kind enumeration | 5 disagreeing enumerators | catalog/config and runtime inspection |
| 10 | Monaco markers for unresolved imports | 0 | UI source inspection |
| 11 | Files tab version label | absent | UI source inspection |
| 12 | `resolveStrings` call sites | 247 | library source census |
| 13 | Linked adopters mounting `LibraryStringsProvider` | 0 | adoption source inspection |
| 14 | Adopter translation callback proven end to end | not proven | adoption/runtime inspection |
| 15 | Wheel handler cancels browser gesture | no | UI source inspection |
| 16 | Ctrl/pinch zooms canvas | no | UI source inspection |
| 17 | Story cards reorder by drag | no | UI source inspection |
| 18 | `component-tests` in run verdict set | present but failed/error | baseline suite run |
| 19 | Component-test timeout fails the run | no | baseline suite run: error with zero blockers |
| 20 | No-change component-test wall clock | 1,200s timeout behavior | baseline suite component-tests log |
| 21 | Per-capability determinism declarations | 0 / 34 | `.vrooli/test-genie.json` |
| 22 | Routine versions visited | all released versions | provider configuration |
| 23 | Typed four-value `x-adoptionMaturityFloor` | no | catalog schema inspection |
| 24 | `blocking` and `runner` required on every gate | no | catalog schema inspection |
| 25 | Semantically active unvalidated `x-` fields | 4 | catalog schema inspection |
| 26 | Calibration directories without matching gate | 2 | catalog/calibration inventory |
| 27 | Blocking gates without calibration fixture | not enforced | calibration inspection |
| 28 | Missing-binary graph findings | 199 | baseline gate report |
| 29 | Declared-but-unimplemented assets reported as warnings | 208 | baseline gate report |
| 30 | TSDoc findings on story `Default` exports | present | baseline gate report |
| 31 | Unreferenced files under `tools/` | 12 | tools inventory |
| 32 | Hash-named promoted tests | 60 | library inventory |
| 33 | Hardcoded bundler special cases | 2 | `api/internal/preview/bundler.go` |
| 34 | Bounded coverage retention | unbounded | coverage inventory |
| 35 | Retirement plan computed on every run | manual only | version lifecycle surface |
| 36 | `catalog readiness` uses `OperationalReport` | no command | CLI source inspection |
| 37 | Readiness names run id/time/completion | absent | CLI/API source inspection |
| 38 | Readiness states floor/achieved/gap | absent | CLI/API source inspection |
| 39 | Triage ordered by blast radius | no | CLI/API source inspection |
| 40 | Adopter-side gates remaining in catalog | 3 | catalog/config.json |

## Run identity

- API port: `17193`
- Baseline suite: `20260826-060632-d81b68a6`
- Baseline gate findings: `1,434`
- Baseline gate runner errors: `53`
- Indexed rows: `205`; manifests: `221`

