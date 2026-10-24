# Audit Technique: Code Cleanup

**Status:** v1 (paired with existing `code-cleanup` skill, 2026-05-03). The skill predates this PoR doc; this entry closes the `skillless canon` smell by giving the technique a strategic-canon home.

## Definition

Audit and remove **dead code, deprecated implementations, forwarding shims, stale TODO/FIXME/HACK comments, and backwards-compatibility cruft** — the artifacts AI agents leave behind during iterative development. Every removal is verification-gated; the goal is to reduce code surface area without breaking consumers.

The full procedure (identify accumulation patterns → search detection patterns → verify before removal → category-specific strategies → aggressiveness guidelines → memory management with tidiness-manager) lives in the paired skill. This document is the strategic-canon side.

## When it applies

✅ **Mature scenarios with iterative development history.** Repeated agent loops have left old implementations alongside new ones, deprecation markers without follow-through, stale TODOs.

✅ **Duplicate parallel implementations.** Functions with `Legacy`, `Old`, `V1` suffixes; two code paths doing the same thing where the new one is fully adopted.

✅ **Forwarding-shim accumulation.** Functions that only delegate to a renamed/relocated implementation; callers haven't been updated.

✅ **`// deprecated` markers without removal.** Annotations describing replacement code that's been live for months — the deprecation never landed.

✅ **Dead exports.** Re-exports kept "for backward compatibility" with no actual importers.

✅ **Stale TODO/FIXME/HACK comments.** "Remove after migration to v2" where the migration completed; "fix after release 2.0" where we're now on 3.5.

✅ **Codebase-clarity drag.** New contributors waste time reading deprecated code, can't tell which path is active, hesitate to delete things "in case."

## When it backfires

⚠️ **Removal without verification.** The skill is unambiguous: never remove without checking all usages, test dependencies, and cross-scenario imports. A misapplied audit deletes code that downstream scenarios import — outage.

⚠️ **Touching `path:packages/*`.** The skill explicitly forbids modifying shared packages because external consumers may exist outside the visible repo. A misapplied audit reaches into shared code; consumers break.

⚠️ **Removing active feature flags.** A flag in a "rollout" state may be off in production but on for partners or beta users. The skill calls these out as do-not-remove.

⚠️ **Deleting "keep until X" code where X hasn't happened.** Comments like "remove after partner migration completes" are constraints, not noise. Verify the condition is met.

⚠️ **TODO removal that drops important context.** A FIXME may be the only documentation of a known limitation; deleting it loses the context even if the comment itself is stale-looking. Investigate intent before removing.

⚠️ **Test fixture removal that hides regressions.** Mocks and intentional stubs may look like dead code but serve real test scenarios. Verify the test that uses the fixture is also retired.

⚠️ **As a substitute for substantive audit.** Cleanup is mechanical; it can't fix architecture, boundaries, or invariants. A cleanup-heavy audit pass that ignores those is hygiene theater.

## What the qa-contrarian watches for

The `qa-contrarian` member challenges audit outcomes; for `code-cleanup` specifically, watch for:

- **Removal without verification trace.** The audit says "removed X (200 lines)" but doesn't show which `rg`/`ast-grep` queries verified zero usage. Challenge: was every category of consumer (in-tree, cross-scenario, packages, tests, docs) actually checked?
- **Cross-scenario consumer missed.** A removal in scenario A breaks scenario B because the audit didn't grep across `scenarios/`. Challenge: was the `rg "name" scenarios/` (with `--glob '!scenarios/<self>/**'`) check actually run?
- **`path:packages/*` touched.** The skill forbids this; an audit that crosses the line is a contract violation regardless of justification. Challenge: did the removal touch any `path:packages/*` file?
- **TODO removed without investigation.** A `// FIXME: workaround for race in X` was deleted because "it looks stale," but the race condition was never fixed. Challenge: was the underlying condition verified resolved?
- **Test coverage hole.** Code was removed and tests still pass — but the tests didn't exercise the removed code, so passing isn't evidence of safe removal. Challenge: did the test suite actually exercise the deleted path?
- **Bulk removal hiding behavior change.** A 1000-line removal includes a few lines that did real work. Challenge: did the audit batch removals into reviewable chunks (one logical removal per commit)?
- **`tidiness-manager` notes underspecified.** The audit logged "cleaned up dead code" without naming what was removed. Challenge: would a future contributor reading the campaign note understand what was done?
- **Active feature flag removed.** A flag in a partial-rollout state was treated as dead code. Challenge: was the flag's deployment status actually verified?

## Paired skill

`scenarios/prompt-manager/store/skills/packs/core/code-cleanup/SKILL.md` — the executable spec. Required reading: `prompt-manager skills read knowledge-observatory-tools` and this PoR doc. Uses `tidiness-manager` for cleanup recommendations and visit tracking.

## Cross-references

- [`README.md`](README.md) — registry overview, lifecycle rules, doc + paired skill discipline.
- [`cognitive-load-reduction.md`](cognitive-load-reduction.md) — companion lens; dead code is cognitive load even off to the side.
- [`screaming-architecture-audit.md`](screaming-architecture-audit.md) — companion lens; dead modules can mask the actual structure.
- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview.
