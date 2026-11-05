# Audit Technique: Decision Boundary Extraction

**Status:** v1 (paired with existing `decision-boundary-extraction` skill, 2026-05-03). The skill predates this PoR doc; this entry closes the `skillless canon` smell by giving the technique a strategic-canon home.

## Definition

Audit whether the scenario's **important decisions** — branches, thresholds, rules, modes, strategies, fallbacks, routing — are explicit, named, intention-revealing, and easy to locate. Mystery conditionals get replaced with named decision helpers; closely related decisions get grouped into cohesive homes; criteria and outcomes are documented or encoded.

The full procedure (identify decisions and context → make them explicit and named → clarify criteria/inputs/outcomes → group by domain → improve testability and observability) lives in the paired skill. This document is the strategic-canon side.

## When it applies

✅ **Mystery conditionals.** `if (x > 5 && y === 'foo' && !z)` with no comment or helper name explaining what's actually being decided.

✅ **Scattered decisions about the same concern.** Permission checks duplicated across handlers; mode selection logic spread across many files; threshold comparisons embedded inline at many call sites.

✅ **Threshold creep.** Magic numbers buried in conditionals (`limit > 50`, `score > 0.7`); changing the threshold requires hunting through code rather than updating one place.

✅ **Strategy selection without a strategy table.** Branching on type/role/mode that selects between alternative behaviors; the alternatives aren't enumerated anywhere.

✅ **Fallback paths.** Error-handling decisions that pick between retry, default, propagate, or abort with no clear rationale at each site.

✅ **Feature flags evaluated multiple times.** Same flag checked at many sites instead of routed through a single decision helper.

## When it backfires

⚠️ **Decisions extracted that didn't need a name.** A simple `if (count > 0)` doesn't benefit from `function hasItems(count) { return count > 0 }`. Reserve extraction for decisions whose intent is non-obvious.

⚠️ **Domain-misaligned helper names.** A helper named `decideMode` that picks between three strategies isn't more revealing than the inline branch. The name should reveal *what* and *why*, not just acknowledge that a decision exists.

⚠️ **Decision-helper sprawl.** Many one-line helpers each used once; the indirection costs more than the inline branch saved. Extract when the decision is non-obvious, repeated, or tested.

⚠️ **Conflict with `cognitive-load-reduction`.** Excess extraction adds indirection that hurts local readability. The lenses pair well only when the extraction names a *real domain decision*; otherwise they conflict.

⚠️ **Behavior change.** The skill allows fixing "clearly incorrect decisions" but explicitly preserves observable behavior otherwise. When the lens is misapplied as "redesign the decision," it violates that boundary.

⚠️ **Logging sensitive data.** The skill cautions: avoid logging sensitive data when adding decision observability. A misapplied audit can introduce a security or privacy regression in the name of clarity.

## What the qa-contrarian watches for

The `qa-contrarian` member challenges audit outcomes; for `decision-boundary-extraction` specifically, watch for:

- **One-use helpers.** A decision was extracted but only one call site uses it; the indirection added cognitive load without reuse benefit. Challenge: why is this extraction worth the indirection?
- **Names that don't reveal criteria.** `decideRouting()` is no clearer than the inline branch. Challenge: does the name tell a reader *what* condition triggers each outcome, not just *that* a decision happens?
- **Behavior changes during extraction.** The "extraction" subtly changed which path runs in some edge case; tests didn't catch it because they didn't cover that case. Challenge: does the extracted decision produce the same outputs as the inline form for every documented input?
- **Sensitive data in logs.** New decision observability includes auth tokens, PII, or other sensitive context. Challenge: was the logging payload reviewed for sensitivity?
- **Magic numbers stayed magic.** Thresholds were extracted to a helper but the threshold value itself remained a literal. Challenge: is the threshold now configurable, or just centralized in one place?
- **Duplicate decisions consolidated incorrectly.** Multiple call sites had subtly different conditions; the extracted helper aligned them, but the alignment was wrong (some sites should have stayed different). Challenge: did the audit verify each call site's intent before merging?
- **Test coverage of branches.** New decision helpers don't have tests that exercise every branch and edge boundary. Challenge: would removing one branch be caught by the test suite?

## Paired skill

`scenarios/prompt-manager/store/skills/packs/core/decision-boundary-extraction/SKILL.md` — the executable spec. Required reading: `prompt-manager skills read knowledge-observatory-tools` and this PoR doc.

## Cross-references

- [`README.md`](README.md) — registry overview, lifecycle rules, doc + paired skill discipline.
- [`cognitive-load-reduction.md`](cognitive-load-reduction.md) — companion lens; named decisions help readability when the name is honest.
- [`invariant-discovery-and-enforcement.md`](invariant-discovery-and-enforcement.md) — companion lens; many decision criteria correspond to invariants on inputs.
- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview.
