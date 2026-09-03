# Problems — React Component Library

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-05-12 — iframe-bridge child wiring deferred (resolved for inspection)

**Symptom:** The live-preview iframe announced first paint via a hand-rolled `postMessage({ type: "preview-ready" })` instead of going through `@vrooli/iframe-bridge/child`.

**Root cause:** Slice 3 (req PR-001/002/003) scope-cut: getting real React execution inside the iframe was the gate, and bundling the bridge child into the per-component esbuild call adds a second resolver setup that wasn't needed for first-paint. Deferring to req-06 lets the bridge work land alongside the inspector UI that consumes it.

**Workaround:** No current user workaround is required. The harness now owns a narrow, data-only inspection wire contract alongside first-paint messages.

**Real fix:** Inspection is implemented, but adopting the full generic bridge remains optional future consolidation work rather than a blocker for the preview workspace.

**Owner:** unassigned (only if full bridge consolidation is later prioritized).

**Refs:** `docs/RESEARCH.md` (Preview harness bundling — Resolved contracts), `api/handlers/preview/static.go`, `requirements/06-element-selection-via-iframe-bridge/`.

### 2026-07-15 — setup JSON has no preview runtime model

**Symptom:** Indexed examples carry optional `setup`, which can look executable
in the component editor even though the harness never evaluates it.

**Root cause:** The examples-as-data index contract was deliberately broader
than the first preview renderer's data-only resolver.

**Workaround:** Use `props` and the documented `$` vocabulary for renderable
examples. Try props is intentionally limited to a shallow JSON-object override.

**Real fix:** Specify a safe, testable setup lifecycle and failure model in a
separately scoped requirement before adding any execution semantics.

**Owner:** unassigned.

**Refs:** `docs/concepts/DATA.md#preview-session-boundary`,
`docs/concepts/FLOWS.md#preview-workspace-experiment`,
`requirements/17-examples-as-data/module.json`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## Work ladder

- Rung: W3 / R3 (design-token vocabulary, kit compatibility, and Settings layout contract)
- Evidence: W0/W1 remain clean (`business-health validate scenario react-component-library` and `vrooli scenario requirements validate react-component-library` both pass). The live catalog now exposes calibrated blocking `fallback-parity`, `kit-compatibility`, and `affinity-compatible` gates; those gates plus `token-vocabulary` and `token-ramp-complete` report zero findings while each planted fixture is detected. The current census covers 162 components and 159 external references with zero undefined properties or affinity overclaims. Banner 1.0.12 passed focused preflight/component validation and removed the last active fallback mismatch. Browser Automation Studio executions `7b4a624d-3227-4178-9138-dcbbd9cec23d` and `445ae14d-530c-41f0-bb75-8aba52fb0379` exercised the real web-console Settings modal at 390px and 924px: narrow controls align to the trailing edge, the narrow group border is 0px, the wide border is 1px, and measured rows remain at least 48px tall.
- Blocker: None for this plan's intent. The broad scenario baseline retains unrelated historical release-provenance/index findings and other inherited suite failures; the plan-owned deterministic gates, component contract, generator check, and real consumer layout evidence are green.
- Measured: 2026-08-29.

- Rung: W3 / R0 (overlay keyboard coordinate-system regression)
- Evidence: `useOverlaySurface` placed visual-viewport variables on the panel while `data-avoid-keyboard` rules read them from the presentation root. The root therefore used fallback geometry while the child panel independently adopted the reduced visible height, producing a severe apparent shrink. Release 1.3.11 exposes root viewport props; the updated presentations use the visual viewport rectangle once and size panels against that root.
- Blocker: Manual confirmation on the operator's mobile keyboard remains pending; broad automated suites were intentionally not run for this iteration.
- Measured: 2026-08-29.

- Rung: W3 / R0 (nested overlay layer-order regression)
- Evidence: `useOverlaySurface` registered the current `close` callback as a layer-effect dependency. An inline controlled callback changed that identity when a parent drawer rendered to mount a child, so the lower parent was removed and pushed above the child. The child's swipe followed the pointer, then failed its top-layer commit check and reset. Release 1.3.10 keeps registration stable and forwards dismissal through a current-callback ref; Web Console rebuilt and reported healthy with the upgraded presentation releases.
- Blocker: Manual confirmation of the original two-drawer swipe sequence remains with the operator; broad automated suites were intentionally not run for this iteration.
- Measured: 2026-08-29.

- Rung: W3 / R0 (overlay contract and styling portability execution)
- Evidence: W0 remains aligned with goals `design-language-foundation` and `react-component-library-adoption-integrity`; `business-health validate scenario react-component-library` and `vrooli scenario requirements validate react-component-library` both pass. Baseline run `20260827-043742-0085352a` is terminal and records the pre-existing red implementation baseline that this plan must compare against while it builds the missing portability and overlay behavior.
- Blocker: None. The work is implementation and hardening at W3; inherited suite failures remain baseline evidence rather than permission to weaken the plan's acceptance bar.
- Measured: 2026-08-27.

- Rung: W3 (2026-08-27 cold version tier implementation)
- Evidence: The ledger now separates `materialized` and `evicted` presence, restores evicted files through a hash-verified atomic materializer, preserves evicted rows across reindex, sees evicted source imports, exposes reconcile/materialize/archive/doctor surfaces, and wires adoption lifecycle hooks. Focused Go tests, CLI tests, UI type-check, focused UI tests, package build, and `sync-exports:check` pass. Browser Automation Studio captured the corrected desktop shell at `/tmp/rcl-ui-20260827-fixed-shell-2/64d0272a-dcfe-4196-85e6-c9e845fc3d8d/screenshots/step-01-40a5b0aa-7c09-4b97-928d-a5d500d6b2fc.png` after a real grid-placement defect was fixed at the consumer boundary.
- Blocker: The live corpus still contains pre-existing released source/ledger hash mismatches and 15 authored version paths absent from the ledger, so full-corpus eviction and the strict no-overlay package census remain withheld. The scenario baseline also remains red on inherited phases; the adoptions staleness test reports its existing reviewed-divergence allowlist entries.
- Measured: 2026-08-27.

- Rung: W3 (2026-08-27 closing validation)
- Evidence: Comprehensive run `20260827-040235-59ec7d4a` passed portability, structure, component-tests, agent-conformance, and templates. The focused contracts rerun `20260827-041517-9b9e92a2` removed the archive `--out` binding error; live doctor and package export synchronization remain clean. The corrected Browser Automation Studio capture proves the workbench shell is no longer collapsed into the right-hand column.
- Blocker: The comprehensive run remains red on inherited contracts/storage/workflow/experience/security/unit findings; CLI still has the pre-existing duplicate `CatalogService.RunGate` binding. Plan validation cannot read its producer baseline (`baseline not found`). The post-suite live snapshot reports 205 evicted and 411 materialized versions because suite setup restored three versions; this is not a ledger-loss event.
- Measured: 2026-08-27.

- Rung: W3 (scenario suite remains baseline-red while the v4 plan surfaces are green)
- Evidence: W0 comparison passes for the named design-language-foundation and react-component-library-adoption-integrity goals; business-health and requirements validation both pass. The governed run `20260825-030615-63ca5e63` passes 22/24 phases, including `ui-health` and `component-tests`, while `workflow` retains expected-element failures and `experience` retains capture/binding failures.
- Blocker: No plan-specific implementation blocker identified; the remaining failures are inherited scenario-suite findings and must remain separated from the plan’s baseline-regression criterion.
- Measured: 2026-08-25.

- Rung: W3 (2026-08-26 control-contract plan remains full-suite red)
- Evidence: The fresh governed run `20260826-230145-8bb9c3e4` passed the target `component-tests` phase and the plan-specific gates/focused checks, but finished `FAIL` at 17/24 phases. Comparison against anchor `20260826-221616-8c7fb975` reports one regression in `performance`: `PERF_BUILD_FAILED` from the repository control-plane build. The new experience floors produced no `floor-content-not-clipped` or `floor-control-geometry` failures; unrelated existing tap-target, contract, and corpus findings remain.
- Blocker: Full plan closure still requires a clean/no-new-failure scenario comparison. The recorded performance regression is outside the plan’s scenario boundary and is caused by unrelated dirty control-plane work; web-console comparison `20260826-224928-742f3803` has zero regressions but remains suite-failed on pre-existing phases.
- Measured: 2026-08-26.

- Rung: W3 (2026-08-26 final control-contract re-measure)
- Evidence: The repaired control-plane build now passes (`go build ./cmd/vrooli` and affected package tests). Fresh governed RCL run `20260826-233228-c88bf50c` finished 18/24 phases passed; comparison against `20260826-221616-8c7fb975` is `preexisting` with no regression, while `performance` and `component-tests` pass. Web-console run `20260826-224928-742f3803` likewise has no regressions against its anchor. Current-version facades point to `ControlBase/1.1.0` and `VoiceInputButton/4.3.0`; focused geometry/voice checks pass 9/9 and UI catalog type-check passes.
- Blocker: None within the plan boundary. The six RCL terminal failures and nine web-console terminal failures remain pre-existing suite debt; residual catalog findings remain named by their owning gates.
- Measured: 2026-08-26.

- Rung: W3 (exploratory feature hardening — Button 2.2.0 and Preview-grid slices)
- Evidence: On 2026-08-18 Button 2.2.0 was released and indexed with a governed `Icon` path, retained custom-icon compatibility, and corrected decorative-icon semantics. Its first live preview exposed the API stamp scanner treating `forwardRef<HTMLButtonElement, ButtonProps>` as JSX and injecting attributes into the type arguments. A shared `forwardRef` regression fixture passes in both the Go preview transformer and the Vite transformer. On 2026-08-19 the Preview canvas gained case-specific drag grips, keyboard reordering, live case dimensions, and a dedicated focus action. Follow-up operator testing showed card-only drop listeners lost events over iframe content and grid whitespace; canvas-level hit testing plus temporary iframe drag shields now accept the full canvas, render explicit insertion edges, and use the whole card as the drag image. The focused comparison/canvas interaction test and `tsc --noEmit` pass.
- Follow-up evidence: On 2026-08-19 `EmptyState@1.3.0` was published through the focused authoring flow. Its panel now composes `Card@1.1.0`, its primary action composes `Button@2.2.0`, and the story no longer supplies a raw button fixture. The component-scoped source/dependency/story/preview check passed and the live harness returned HTTP 200. A publish-cleanup defect found during this slice was fixed and covered by a focused test; superseded draft directories are removed only after the replacement release indexes successfully.
- Follow-up evidence: On 2026-08-19 `EmptyState@1.6.0` completed the same rapid loop with `Heading@1.0.0` and `Text@1.0.0` replacing hand-authored title/description typography. Raw HTML is now limited to structural wrappers and the style tag; the standard action path remains a library `Button`, while the custom `action` slot is retained for compatibility. A stale draft-content cache let an earlier preflight miss a bad primitive import path; re-indexing before the next check exposed it. The final release also uses its canonical story path, passed the source/dependency/story/preview check, and returned live preview HTTP 200; the superseded 1.4.0 and 1.5.0 releases are deprecated rather than mutated.
- Follow-up evidence: On 2026-08-19 `EmptyState@1.7.0` removed the local stylesheet and made the intended rhythm explicit through nested primitives: outer `Stack gap="lg"` separates icon, copy, and actions; inner `Stack gap="3xs"` keeps the heading and description as one copy group. `Container` owns the readable measure, `Stack` owns alignment and responsive inset, `ButtonGroup` owns action spacing/collapse, and `Text`/`Heading` own margin normalization. `Stack@1.2.0`, `Inline@1.1.0`, `Container@1.1.0`, `Text@1.1.0`, `Heading@1.1.0`, and `ButtonGroup@1.1.0` were released to supply the missing composition contracts. All seven focused component checks passed and the live EmptyState preview returned HTTP 200; full suites and visual baselines were intentionally not rerun.
- Follow-up evidence: On 2026-08-19 a focused preview screenshot exposed `EmptyState@1.7.0` overflowing its card at the preview width: `Stack` applied `inline-size: 100%` and responsive padding without `box-sizing: border-box`, so the padding was added outside the measured width. `Stack@1.2.1` now includes border-box sizing and a zero inline minimum; `ButtonGroup@1.1.1` adds the same containment guarantees for responsive action children; `EmptyState@1.7.1` pins both fixes. The focused source/dependency/story/preview checks passed, the live preview returned HTTP 200, and a targeted canvas screenshot confirmed the action remains inside the card while copy wraps within its measure. Full suites and baselines remain intentionally out of scope.
- Follow-up evidence: On 2026-08-19 `EmptyState@1.7.2` added declarative `layout`/`noOverflow` expectations to all three stories, reusing the existing preview/headless assertion path rather than creating a parallel visual-test system. The focused component check now re-indexes the selected manifest before resolving source and dependencies, preventing stale catalog rows from hiding draft edits; a service regression test proves a newly added missing library dependency fails the focused check. Draft and released EmptyState checks passed, all three rendered story assertions passed, the exact draft directory was removed after publish, and the live harness returned HTTP 200. Full suites and baselines remain intentionally out of scope.
- Follow-up evidence: On 2026-08-19 the same release loop exposed a version-copy defect: `experience-contract.json` was preserved byte-for-byte, leaving `EmptyState@1.7.2` pointing at the prior version's story. The source store now rewrites version-local `storyRef` paths whenever it copies an experience contract; a service regression test covers both draft and release copies. `EmptyState@1.7.3` was published with a self-referential story path, passed focused source/dependency/story/preview and all three rendered story assertions, removed its draft directory, and returned live preview HTTP 200. Full suites and baselines remain intentionally out of scope.
- Follow-up evidence: On 2026-08-19 `Avatar@1.1.0` fixed shape leakage in image/initials fallbacks, made loading fixtures deterministic without a 404, normalized group children and bounded `maxVisible`, and rebuilt its showcase with `Card`, `Stack`, `Heading`, and `Text`. The focused source/dependency/story/preview check and all four rendered story assertions passed; the group layout contract passed while intentional presence-badge protrusion was left outside no-overflow assertions. The release removed its draft directory and returned live preview HTTP 200. Full suites and baselines remain intentionally out of scope.
- Follow-up evidence: On 2026-08-19 the Avatar named-identity screenshot exposed the remaining placeholder seam: the generic `ProgressiveImage` text placeholder wrapped inside the small avatar before the image settled. `Avatar@1.1.1` now supplies a shape-preserving shimmer whenever callers omit `placeholder`; its focused check and all four rendered story assertions passed, the draft was removed after publish, and the live harness returned HTTP 200. Full suites and baselines remain intentionally out of scope.
- Follow-up evidence: On 2026-08-19 `Avatar@1.1.2` completed the cleanup by reusing `ProgressiveImage`'s existing shimmer and `frameStyle` contracts. Avatar no longer duplicates loading animation CSS or requires a story-specific loading marker; only identity, presence, group, and compact-frame integration rules remain local. The focused source/dependency/story/preview check and all four rendered story assertions passed, the draft was removed after publish, and the live harness returned HTTP 200. Full suites and baselines remain intentionally out of scope.
- Constraint: Keep this as a narrow learning slice. Do not infer full-catalog quality or adoption integrity from these results; case order is intentionally session-local, and the full scenario suite and visual baselines were not rerun during rapid iteration.
- Follow-up evidence: On 2026-08-24 the catalog evidence endpoint was changed from a synthetic visual pass to the BAS-backed component runner. Full-catalog capture initially exceeded the synchronous client deadline after producing a partial manifest; bounded `limit`/`offset` requests, durable manifest continuation, and CLI batch orchestration now keep each request within the lifecycle boundary. A broad run advanced through offset 416 with 479 story rows and no failed rows before being deliberately stopped in accordance with the plan's System Monitor priority. The remaining corpus is resumable and is not evidence of full-catalog visual completion.
- Follow-up evidence: On 2026-08-24 the same BAS path captured selected Card, DataTable, Dialog, Tabs, List, Text, Chart, Icon, and Button versions with passed individual and review-sheet rows. The generic sheet route was inspected at desktop and mobile widths. Button's mobile long-label specimen initially exposed horizontal overflow; the constrained specimen and shared Pressable ellipsis now produce root `scrollWidth == clientWidth` with the full accessible label preserved.
- Measured: 2026-08-19.

- Rung: W0
- Evidence: Goal `react-component-library-adoption-integrity` requires “derive styling contracts from version source,” “block unsafe apply/reapply/reconverge,” and “support atomic batch and lifecycle operations”; the PRD's P0 targets cover indexing/style-affinity projections (`OT-P0-001`), adoption drift status (`OT-P0-006`), and CLI parity (`OT-P0-007`), but no P0 operational target names those three required capabilities. The goal therefore directs capabilities that the contract leaves below P0 or unnamed.
- Blocker: None operational; the contract mismatch must be repaired through `prd-authoring` before lower-rung validation is authoritative.
- Measured: 2026-08-28.

- Rung: W0 — pass
- Evidence: After contract repair, `OT-P0-001` names source-derived styling contracts, `OT-P0-006` names unsafe apply/reapply/reconverge refusal with explicit validated override, `OT-P0-007` names atomic batch and lifecycle operations, and `OT-P0-008` names token-complete templates/adopters. These P0 targets cover the capabilities named by `react-component-library-adoption-integrity`; the reconciled `design-language-foundation` goal's separate generation-gate and design-kit targets are not assigned to this scenario.
- Blocker: None.
- Measured: 2026-08-28.

- Rung: W3 / R0
- Evidence: W0, `business-health validate scenario react-component-library`, and `vrooli scenario requirements validate react-component-library` pass. The server-owned W3 run `20260828-011725-65a34ee0` finished 18/24 phases passed, 5 failed, 1 skipped. Current failures are storage (`STORAGE_PATH_UNCOVERED`, `STORAGE_BUDGET_BELOW_OBSERVED`), workflow (routed isolation unavailable plus an invalid `executionMode: "read_only"` BAS fixture), experience (587 errors including unresolved specimens and historical contract-shape errors), security (`gosec.G702` at `cli/domains/coverage/census.go:380`), and unit (validation RPC deadline exceeded). Plan-specific component-tests, package, API, CLI, and structural checks remain green.
- Blocker: None operational; repair the actionable implementation-layer findings and retain environmental/provider findings as measured evidence.
- Measured: 2026-08-28.

- Rung: W3 / R0 — latest re-measurement
- Evidence: W0, `business-health validate`, and `vrooli scenario requirements validate` remain passing. Comprehensive run `20260828-021659-16f76687` finished 18/24 phases passed, 5 failed, 1 skipped; independent owned component-tests run `20260828-022849-781f0349` passed 1/1. The remaining comprehensive findings include adjacent template-manager dependency setup, provider/unit evidence, historical experience/security/branding debt, and a current `Banner@1.0.11` catalog typecheck failure. Plan-specific package build, API/CLI suites, cache/BAS/AST/self-hosting/artifact checks remain green.
- Blocker: None operational; the plan DOD suite-green and strict historical byte-equivalence rows remain unproven and are recorded in the plan ledger.
- Measured: 2026-08-28.

- Rung: W3 / R0 — library unit-suite re-measurement
- Evidence: `pnpm --dir scenarios/react-component-library/ui run test:library` completed in 238 seconds with 1,076 passing and 143 failing tests across 3 files. A focused Banner story-contract run reproduces the historical-version `banner.icon` crash independently; the broader failures also include LayerManager portal/render errors, Text 1.1.0 restyle/ref failures, and catalog stamp/expectation mismatches. Package-owned typecheck, catalog lint, and the independent component-contract suite remain green.
- Blocker: None operational; the failures are released component/story-contract debt outside this setup-only plan's explicit scope and are retained as evidence rather than repaired by changing released component behavior.
- Measured: 2026-08-28.

- Rung: W3 / R0 — generic story-contract harness correction
- Evidence: The runner's raw-component coverage mount now runs only when a composed story supplies direct arguments; composed stories with empty args use their declared specimen as the valid construction path. The focused Banner contract suite passes 12/12 after this correction. A bounded corpus run then reaches 22 passing stories before exposing AppNavigation Mobile's component/story mismatch (`display: none` labels versus a visible-text expectation).
- Blocker: None operational; the remaining AppNavigation contract is component behavior debt and remains outside this setup-only plan.
- Measured: 2026-08-28.

- Rung: W3 / R0 — generic catalog conformance and workflow seam cleanup
- Evidence: Catalog conformance now type-checks and lints discovered current/draft sources directly, with no disposable-source allowlist, component-name branch, version-specific rewrite, or fixed component injection. The UI production build and `catalog:check` pass; catalog lint retains 13 existing warnings and no errors. The observer preview case now navigates directly to its preview route without a mutating click, and workflow static validation no longer reports `observer_content_unsafe`. Stale experience references were reconciled to materialized story IDs, with a local cross-reference check reporting zero invalid state examples.
- Outstanding: The full scenario run remains in progress or may expose additional inherited experience-contract debt; the current plan validation must use its terminal result rather than infer from this focused evidence.
- Measured: 2026-08-28.

- Rung: W3 / R0 — gate-loop declaration execution re-measurement
- Evidence: The scoped runner registry, declaration resolver, per-asset rule-set digest, generated determinism block, canonical asset-kind migration, protobuf provenance fields, and scoped gate CLI path compile and pass focused tests. The server-owned run `20260829-150711-c0e6c86f` finished 20/27 phases passed, 6 failed, 1 skipped. Remaining failures include inherited performance/unit/component-test baseline debt and an empty `react-component-library:Input@1.2.0` file mirror that prevents UI export materialization.
- Outstanding: Full suite health and the remaining historical selector/provenance corpus findings are retained as validation evidence; they are not repaired by weakening this gate-loop refactor.
- Measured: 2026-08-29.

- Rung: W3 / R0 — version-model and preview closure re-measurement
- Evidence: Preview resolution now carries the story-selected version into bundle assembly, resolves major selectors to the newest compatible release, publishes a mobile viewport contract, and the targeted preview tests pass. Workflow run `20260831-131617-f4b8e470` passed. Legacy unprotected experience contracts were collapsed to canonical references or removed when orphaned; the live corpus has one experience authority and zero vacuous-allowlist entries. Governed presence reconciliation completed without errors after the changed retired mirrors were explicitly reconciled.
- Outstanding: The current corpus snapshot reports 61 evicted versions without file mirrors and two mutated release-hash entries belonging to the pre-existing staged `VoiceInputButton@4.3.4` work; these prevent claiming I2/I11 until the owning release state is reconciled. Experience run `20260831-133255-ecdb5972` was still server-owned at measurement time, so its terminal result remains required.
- Measured: 2026-08-31.

- Rung: W3 / R0 — production lint triage
- Evidence: Removed redundant literal-union constituents from the hand-written UI API types and corrected an unnecessary optional clipboard chain. The three affected production files now lint cleanly without changing runtime behavior.
- Outstanding: Scenario-wide lint still has 198 production copy-registry findings plus test-query and catalog-key debt; these remain broad pre-existing UI hygiene work rather than version-model or retention failures.
- Measured: 2026-08-31.

- Rung: W3 / R0 — translation registry cleanup
- Evidence: Removed only entries proven orphaned by `strings/no-unused-keys` (including stale metadata-shaped top-level entries), regenerated `strings.generated.ts`, and restored the single real `components.versionLabel` callsite revealed by the catalog type-check. The generated registry now has zero orphan diagnostics; `catalog:check` and the UI build exit 0.
- Outstanding: Full UI lint still reports 328 copy-stability findings plus unrelated design/test hygiene rules. The safety rule remains enabled; no lint category was disabled.
- Measured: 2026-08-31.

- Rung: W3 / R0 — export manifest regression guard
- Evidence: Added a generic package test that checks every advertised `package.json` subpath has a corresponding built declaration. The sync-export suite is 6/6, and the authored-source projection remains at 244 assets / 280 versioned exports / zero broken imports.
- Outstanding: The DOD audit remains open only for the previously recorded broad validation and baseline-producer divergences; package build exclusions are now derived from manifest eviction metadata rather than component names.
- Measured: 2026-08-31.

- Rung: W3 / R0 — authored export projection and consumer reachability closure
- Evidence: Package export generation no longer projects cold ledger history into the published manifest. It now shares the authored source root with the package compiler, eliminating 767 advertised-but-unbuilt export warnings. The live export check reports 244 assets, 280 versioned exports, and zero broken imports; catalog conformance exits zero with only the existing 14 non-blocking lint warnings. Four template consumers were migrated from the retired `Button/1` major to `Button/2`. Package build, boundary/sync-export tests, catalog check, UI build, and `git-control-tower`, `offer-desk`, and `web-console` builds all pass. Matrix diff reports zero changed cells with a failed verdict.
- Outstanding: Scenario-wide lint/check and the comprehensive server-owned Test Genie run remain red on inherited findings. This export correction does not claim those unrelated gates.
- Measured: 2026-08-31.

- Rung: W3 / R0 — comprehensive final validation re-measurement
- Evidence: Server-owned run `20260831-141046-f498387d` reached terminal `FAIL` after 1390.6 seconds, with 234 errors and 1,729 warnings. Plan-specific catalog type-check, build, targeted API/CLI tests, cold-source recovery, doctor, and release-ledger audit pass; `catalog:check` passes after reconciling the staged VoiceInputButton authored bytes and durable mirror.
- Outstanding: inherited experience, security, unit, performance, and broad UI-quality findings remain. The fresh corpus report has I5=313 versus 290 and I6=18% versus 15%; these are reachable-corpus/fixture debt, not unreadable mirrors. The plan is not complete.
- Measured: 2026-08-31.

- Rung: W3 / R0 — retention floor and provenance re-measurement
- Evidence: `make restart` is healthy; catalog type-check, focused components/preview/gates/version-ledger Go suites, and CLI component tests pass. Governed materialization restored Icon, StatusBadge, and Tabs from exact cold archives; `versions doctor --json` is empty. Explicit governed retirement reclaimed five legacy directories and four further safe candidates. Fresh corpus report is I2=0, I13=0, and I20=1.
- Outstanding: corpus size is 313 directories against I5 target 290, superseded compiled-source share is 18% against I6 target 15%, and I11 remains two mutated staged `VoiceInputButton@4.3.4` release-hash entries. `catalog:check`/`exports:check` therefore still reject the current staged release as ledger-stale. The broad experience run retains inherited failures. The refused `ClassMerge@1.0.1` retirement is still referenced and was correctly preserved.
- Measured: 2026-08-31.

- Rung: W3 / R0 — final retention/provenance re-measurement
- Evidence: Governed cleanup reduced the retained candidate set to 278 rows: 244 latest releases, 33 source-referenced releases, and one already-retired row; no safe candidates remain. `versions doctor --json` is `{}`. The durable corpus report now records I5=290, I6=12%, I10=0, I11=0, I13=0, and I20=1. Package build, export sync, boundary tests (6/6), sync-export tests (5/5), focused API/CLI suites, experience-manager spec tests, catalog check, and all three consumer builds (`git-control-tower`, `offer-desk`, `web-console`) pass. The offer-desk failure was traced to stale lifecycle-provisioned package output and cleared by `make setup`.
- Outstanding: The server-owned comprehensive run `20260831-141046-f498387d` remains terminal FAIL with inherited experience, security, unit, performance, and broad UI-quality findings; scenario `make lint`/`make check` therefore remain non-green outside the focused plan gates. The catalog check has no errors but retains 14 lint warnings. No further retention is safe without changing declared latest/source dependencies.
- Measured: 2026-08-31.

- Rung: W3 / R0 — matrix and authoritative-command audit
- Evidence: `make ledger-audit`, `make matrix-diff`, and the scenario UI production build exit zero. The matrix diff confirms the repaired attribution invariant remains intact: no changed cell is a zero-finding `fail`; previous corpus-wide mirror/runner failures are now represented as passing or attributable/unmeasured results. The diff also identifies one current `navigation.tabs` documentation finding for exported `TabsDensity` and `TabsVariant`; it is retained as a release/documentation issue because the source is an immutable released version. Readiness remains `NOT_READY` with 2,381 omitted triage rows and inherited catalog evidence debt.
- Outstanding: D7, D38, and D39 are not complete: scenario-wide lint/check and the comprehensive Test Genie run remain red, and the matrix baseline is not behaviorally clean until the Tabs documentation delta and inherited findings are resolved or explicitly accepted by the owning workflow. Plan-specific retention, provenance, export, catalog, and consumer-stability evidence remains green.
- Measured: 2026-08-31.

- Rung: W3 / R0 — Tabs documentation regression closure
- Evidence: Opened and published `Tabs@1.2.3` through the governed draft path with documentation-only changes for `TabsDensity` and `TabsVariant`, updated the single canonical experience story reference, and retired `Tabs@1.2.2` using the exact cleanup hash. Package build, boundary/export tests, catalog check, matrix diff, and all three post-publish consumer setup/build proofs pass. The durable corpus remains I5=290 and I6=12%; `versions doctor` remains empty.
- Outstanding: D7 and D38 remain open for inherited scenario-wide lint/check and comprehensive Test Genie failures. D39's plan-owned Tabs documentation delta is resolved; the current matrix also surfaces an independent `controls.slider` documentation finding plus attribution/corpus baseline changes and inherited findings.
- Measured: 2026-08-31.

- Rung: W3 / R0 — Slider documentation regression closure
- Evidence: Opened and published `Slider@1.1.2` through the governed patch path, adding TSDoc for the public value-display type and props interface without editing `Slider@1.1.1`. Retired the predecessor with the exact cleanup hash. The immutable component test for `react-component-library:Slider@1.1.2` passed all closure, source, contract, declared-behavior, and experience-evidence stages; package build, export/boundary tests, catalog check, and the key corpus invariants pass.
- Outstanding: D7 and D38 remain open for inherited scenario-wide lint/check and comprehensive Test Genie failures. Slider's documentation finding is resolved; current matrix output shows documentation `pass` for `controls.slider`.
- Measured: 2026-08-31.

- Rung: W3 / R0 — one-asset verdict and final validation re-measurement
- Evidence: W0 goal search returns the archived `react-component-library-adoption-integrity` and `design-language-foundation` goals; the current PRD targets remain compatible with the scenario work. Targeted catalog, preview, component-test, graph-reconciliation, and CLI package tests pass. `asset check controls.button --version 2.2.6 --json` returns `PUBLISHABLE`; three cached runs complete in approximately 0.22 seconds, and fresh Button rerun `ctr_11089f797b910cba` passes.
- Blocker: The full server-owned run `20260903-014026-7e1c5531` finished 15/28 phases passed, 12 failed, 1 skipped. BAS was subsequently restored and the Button component rerun passed; the remaining API-suite failure is historical fallback-parity drift in immutable released sources. A re-anchored GCT baseline is ready, but the validation producer still reports `UNKNOWN` because its required `/home/matthalloran8/.vrooli/test-runs/react-component-library` workspace is not readable. The final machinery ceilings and fresh-agent proof remain open.
- Measured: 2026-09-03.

- Rung: W3 / R0 — semantic revision stability and test reuse
- Evidence: Catalog rebuilds previously changed only `dependencies.json.resolvedAt` yet invalidated folded revisions and forced a browser rerun. Revision hashing now canonicalizes that bookkeeping field while retaining lock dependency choices; the Button `2.2.6` check returns `PUBLISHABLE`, zero findings, and `tests ... reused=true` in 0.25 s. The API now honors `--run-tests`: a missing report without the flag returns `STALE_TESTS`, while the flag permits execution.
- Outstanding: Full scenario/UI validation, corpus ceilings, and fresh-agent proof remain open as recorded above.
- Measured: 2026-09-03.

- Rung: W3 / R0 — preview preference and focused UI recheck
- Evidence: A missing/empty persisted preview-kit preference now resolves to `vrooli-default`; the focused `ComponentEditor` suite passes 29/29 and the diagnostics contract again reports the canonical kit. The adoption dialog’s prior overwrite/override rapid-typing failures were fixed by wiring the released Dialog’s initial focus to the component-id input; the focused dialog suite now passes 5/5 sequentially.
- Outstanding: The comprehensive UI suites and server-owned scenario run remain red on the previously recorded broad findings. This focused correction does not claim plan completion.
- Measured: 2026-09-03.

- Rung: W3 / R0 — published fallback-parity repair and comprehensive re-measurement
- Evidence: Governed patch releases were published for `AsyncBoundary`, `BottomSheet`, `ErrorState`, `FormWizard`, `FreshnessArc`, `FullPageDrawer`, `PasswordInput`, `ProvenanceInk`, `RadioGroup`, `ResponsiveDialog`, `RollingNumber`, and `SampleSeries`; superseded exported versions were marked deprecated and catalog build regenerated 254 projections / 310 versioned exports with 0 broken imports. Full API and CLI Go suites pass, and the fallback-parity gate reports 467 inspected files with zero findings.
- Outstanding: The server-owned run `20260903-025030-d84f53c5` is terminal FAIL (14 passed, 13 failed, 1 skipped) with inherited UI, dependency, docs, unit, storage, workflow, business, experience, tidiness, security, measures, and component-test findings. UI suites remain red (general: 25 failures / 499 passes; library: 49 failures / 1,058 passes). Corpus targets and fresh-agent proof remain open.
- Measured: 2026-09-03.
