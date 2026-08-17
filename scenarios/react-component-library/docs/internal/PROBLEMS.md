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

## Design ramp debt — `design-system/no-raw-dimensions` baseline

**Status:** rule landed at `warn` on 2026-08-15 against the baseline below. The
count is a ratchet: it may only decrease. When it reaches zero, flip the rule to
`error` in `ui/eslint.config.js` and delete this section.

**Do not** silence findings by adding files to an ignore list. A grandfathered
file is one nothing will ever come back to, and the debt here is exactly the
kind that compounds invisibly.

| Family | Count | Fix |
|---|---:|---|
| `sizing` (`w-*`, `h-*`) | 280 | Route icons through the `Icon` primitive's `size` scale; prefer layout constraints over fixed box dimensions. No autofix — each needs judgement. |
| `spacingFixable` | 14 | `eslint --fix` rewrites these to the matching ramp step. |
| `arbitrary` (`[13px]`) | 5 | Nearest ramp step, or publish the missing rung. |
| `spacingUnmapped` | 3 | Value lands between ramp steps; pick the nearest or publish the rung. |
| **Total** | **302** across 33 files | |

**Why this existed.** The Go `tokens` catalog gate has enforced this rule over
`library/**` all along and reports 0 findings across 331 active sources. Its
source glob never covered `ui/src/**`, so the workspace application — the
surface a maintainer actually looks at — accumulated 302 raw dimensions while
the library it showcases stayed clean. The lint rule now applies to both
surfaces from one implementation so the two cannot diverge again.

**The library was not as clean as the gate reported.** On its first run the new
rule found 3 real violations in `library/**` that the Go gate had been blind to:
`space-y-3` in `ColorPicker/versions/1.0.0/story.tsx` (×2) and `space-y-4` in
`Presence/versions/1.0.0/story.tsx`. The Go `literalDimension` regex enumerates
`p|m|gap|w|h` prefixes and omits `space-x`/`space-y`, so those utilities were
never inspected. All 3 are fixed; the library config runs the rule at `error`
and is green. This is the argument for the rule living in ESLint rather than
being a second Go glob: the AST rule classifies by property prefix set rather
than by a hand-maintained regex alternation, so a missed prefix is a one-line
addition to `SPACING_PREFIXES` instead of a silent blind spot.

**Largest concentrations** (`ui/src/`):

```
49  features/components/ComponentEditorController.tsx
39  features/components/ComponentEditorSource.tsx
30  features/components/ComponentTestPanel.tsx
22  features/components/EmulatorChrome.tsx
21  components/color-picker-harvest/ColorPicker.tsx
20  features/versions/VersionDiffViewer.tsx
19  features/catalog/CatalogBrowser.tsx
```

The `sizing` family dominating at 280 is not incidental: the app renders raw
`lucide-react` icons with `h-4 w-4` rather than passing them through the `Icon`
primitive, so it never inherits the size scale. That is the mechanical cause of
the icon-size inconsistency visible across the workspace chrome.

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

- Rung: W0 (goal/problem contract comparison — passed after registering the direct objective)
- Evidence: On 2026-08-14 the deterministic named-goal search returned `react-component-library-adoption-integrity` in addition to the adjacent `design-language-foundation` initiative. The new goal states the adoption-integrity capabilities directly: derived styling contracts, token-complete templates and adopters, blocking apply/reapply/reconverge, released-version immutability and source drift, atomic batch/lifecycle operations, and real browser validation on generated scenarios and the fleet. Those capabilities map to the PRD's P0 registry, live React preview, adoption workflow, and test/full-catalog coverage targets (OT-P0-001, OT-P0-003, OT-P0-006, OT-P0-008); the adjacent initiative is not used as the contract for this work.
- Constraint: The goal/PRD comparison now authorizes descending into W1–W3. Preserve the direct plan's acceptance evidence and do not treat the broader design-language initiative as a substitute for it.
- Measured: 2026-08-14.
