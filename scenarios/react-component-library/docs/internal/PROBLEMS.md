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
