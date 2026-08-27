# Control contract final validation

Captured 2026-08-26 after the implementation and final focused checks.

## Plan-specific controls

| Control | Evidence | Result |
| --- | --- | --- |
| Manifest identity and source reachability | `catalog.manifest-identity`: 227 inspected, 0 findings; active source gates report 0 unresolved assets | pass |
| Published-version sweep | `implementationSources`: 383 exported, non-deprecated version directories | pass |
| Shared CSS ownership | `catalog.shared-style-ownership`: 178 assets inspected, 0 findings; active census: reduced-motion 1, forced-colors 1, focus-visible 1 | pass |
| Style injection | `catalog.style-injection`: 178 assets inspected, 0 findings; `useComponentStyles.test.tsx`: 1 sheet for 1, 3, and 10 instances | pass |
| Provenance | `catalog.provenance-stamp`: 8,841 files inspected, 0 findings | pass |
| Token vocabulary | `catalog.token-ramp-complete`: 0 findings, 0 runner errors after declaring `--control-icon-size` | pass |
| Control and Versions UI | focused RCL UI checks: 44 tests passed when run per-file; VersionsCard 9/9, ComponentDetailPage 11/11, routes 3/3, geometry 4/4 | pass |
| Consumer propagation | web-console toolbar checks: 64 tests passed; no microphone workaround remains | pass |

## Full-suite evidence

- React Component Library run `20260826-235731-aff3617e` terminated `FAIL` with 18/24 phases passed. The target `component-tests` phase passed; the six failed phases (`contracts`, `unit`, `storage`, `workflow`, `experience`, and `security`) were also failed by the immediately prior run `20260826-233228-c88bf50c`, and the comparison verdict is `preexisting` with no regression. `performance` passed in the prior and current plan-relevant validation. The earlier performance build defect was repaired in the adjacent dirty-tree control-plane code by restoring the existing JSON writer helpers, and `go build ./cmd/vrooli` plus the affected package tests pass.
- Web Console run `20260826-224928-742f3803` terminated `FAIL` with 14/23 phases passed. Comparison against `20260826-211954-c806c28e` classified all nine failed phases as pre-existing and found no regressions.
- The aggregate `catalog gates --all --json` command now completes. The remaining aggregate findings are existing corpus debt: historical immutability/evidence, generic deprecated imports, foreign palette classes, and other catalog-health findings. The new plan gates remain clean.

## Definition-of-done enforcement matrix

Each row below identifies the executable check that would fail if the stated
behavior were removed. Scenario-suite rows use the anchored comparison because
the suites contain unrelated pre-existing failures; the comparison is the
authoritative no-new-failure check.

| DoD row | Enforcing test or gate | Result |
| --- | --- | --- |
| ControlBase resolves all six geometry dimensions from `size` | `control-base-geometry.test.ts`; `catalog.token-ramp-complete` | pass |
| Documented 32px–48px rung measurements match | `control-base-geometry.test.ts` | pass |
| No blanket tap-target floor; warning-only below 44px guidance | `control-base-geometry.test.ts`; ControlBase source contract | pass |
| `density` changes internal spacing only | ControlBase source contract and geometry test | pass |
| Consumers choose documented rungs without inline size overrides | deprecated-import gate; web-console toolbar tests | pass |
| Voice overlays clip without clipping the glyph | `voiceInputButton.test.tsx`; component-tests phase | pass |
| Voice glyph scale is monotonic and shared | BaseStyles icon-token assertions; toolbar tests | pass |
| Voice control has no `app-*` classes and uses `cn` | foreign-token gate; source scan | pass |
| Web-console microphone has no workaround at three densities | `MobileToolbar.test.tsx` and toolbar suite | pass |
| Foundation owns reset, focus, motion, forced-colors, and hidden utility | `catalog.shared-style-ownership`; active exported-source census | pass |
| Each shared CSS concern has one implementation | `catalog.shared-style-ownership` | pass |
| Style node count is bounded independent of instance count | `useComponentStyles.test.tsx` (1, 3, and 10) | pass |
| Consumers can override library styles independent of render order | `useComponentStyles.test.tsx` head-order test | pass |
| All implementation-source assets are visible to gates | `catalog.manifest-identity`; active-source checks | pass |
| All exported non-deprecated versions are gated | `implementationSources` sweep (383 versions) | pass |
| Hoisted style objects are detected | inline-style gate calibration and gate tests | pass |
| Geometry/content floors reach every applicable asset | experience-manager floor execution; floor evidence | pass |
| New/changed vacuous contracts are blocking | vacuous ratchet gate | pass |
| Legacy vacuous allowlist cannot grow | `catalogvalidate` allowlist tests | pass |
| Kind-based claim presets scaffold new contracts | claim-preset tests; scaffold path | pass |
| Versions tab answers health, adoption, change, and retirement questions | `VersionsCard.test.tsx` | pass |
| Progression, Tests, and Adoptions are joined into Versions rows | `VersionsCard.test.tsx`; `api/versionHistory.ts` | pass |
| Required ledger fields are rendered | `VersionsCard.test.tsx` ledger contract test | pass |
| Zero test runs render neutral `unknown`, not pass | `VersionsCard.test.tsx` ledger contract test | pass |
| Versions tab is composed from library assets | `catalog.provenance-stamp`; composition tests | pass |
| Component-source stamp is truthful | `catalog.provenance-stamp` (8,841 files) | pass |
| No in-library dependent imports deprecated ControlBase | deprecated-import gate and dependency sweep | pass |
| Web-console uses repaired VoiceInputButton and passes toolbar checks | 64 focused toolbar tests; component-tests phase | pass |

## Deferred scope

Historical foreign-palette findings and generic deprecated-import findings remain named by their gates. Version retirement, bulk vacuous-contract backfill, the unrelated locale defect, and broad scenario-suite debt remain with their existing owners.
