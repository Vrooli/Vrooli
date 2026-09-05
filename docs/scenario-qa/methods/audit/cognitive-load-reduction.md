# Audit Technique: Cognitive Load Reduction

**Status:** v1 (paired with existing `cognitive-load-reduction` skill, 2026-05-03). The skill predates this PoR doc; this entry closes the `skillless canon` smell by giving the technique a strategic-canon home.

## Definition

Audit whether the scenario's code is **easy to read, navigate, and reason about**. The lens targets local readability, control-flow simplicity, over-abstraction, naming clarity, scattered logic, and state/data-flow opacity. Tightens the loop "open a file → understand what it does → know whether to change it."

The full procedure (optimize local readability → simplify control flow → prune over-abstraction → improve naming → reduce scattering → clarify state/data flow → safe refactoring) lives in the paired skill. This document is the strategic-canon side.

## When it applies

✅ **New contributor friction.** Reading a function takes more than a minute to understand because of nested conditionals, indirection, or unfamiliar names.

✅ **Debug sessions that require many file jumps.** A behavior is implemented across so many locations that following it requires opening five-plus files in sequence. Each file jump is a context switch and a chance for the reader to lose the thread.

✅ **Ambiguous names.** `manager`, `service`, `helper`, `utils`, `data`, `process` — names that reveal nothing about intent. Rename to domain-relevant verbs/nouns.

✅ **Buried happy path.** The success case is hard to see because it's embedded inside extensive error handling, edge-case branching, or feature-flag chains.

✅ **Speculative abstraction.** Wrappers, factories, or strategy-pattern scaffolds that exist "in case" they're needed; reading the code requires understanding the framework before understanding the logic.

✅ **High implicit-state surface.** Functions that depend on global state, ambient context, or hidden mutations — readers can't predict outputs from inputs.

## When it backfires

⚠️ **Naming churn for style.** Renames that swap one acceptable name for another based on personal preference, with no clarity gain. The skill explicitly forbids superficial style edits.

⚠️ **Over-inlining.** Collapsing a useful helper back into a call site because "it's only one line of indirection." Indirection that names a domain operation has value beyond mechanics.

⚠️ **Premature flattening.** Untangling a deeply nested conditional into a sequence that's now easier to read but harder to extend safely (e.g., when the original nesting reflected real domain hierarchy).

⚠️ **Conflict with `decision-boundary-extraction`.** Decisions extracted into named helpers add indirection but reduce ambiguity at the call site; if the cognitive-load lens sees the indirection as bad, it may undo the prior audit's gain.

⚠️ **Behavior change disguised as readability.** A "simplification" that drops an edge case the original code handled. The skill explicitly forbids this; pair with the test suite to catch it.

⚠️ **Scope creep into architecture.** Cognitive-load improvements at the local level can't fix structural problems that require `screaming-architecture-audit` or `boundary-of-responsibility-enforcement`. Stay at the local readability layer.

## What the qa-contrarian watches for

The `qa-contrarian` member challenges audit outcomes; for `cognitive-load-reduction` specifically, watch for:

- **Renames without semantic gain.** New names use synonym vocabulary rather than domain vocabulary; the rename is style preference dressed up as clarity. Challenge: does the new name reveal the function's *intent* better than the old one, or just sound nicer?
- **Helpers collapsed at the cost of named operations.** A helper function whose name expressed a domain operation was inlined; readers now have to derive intent from mechanics. Challenge: was the helper merely indirection or was its name doing work?
- **Edge-case loss.** A simplification dropped handling for a case the original branch covered; tests didn't catch it because they didn't exercise that case. Challenge: was every branch's purpose understood before it was simplified away?
- **Increased line count without clarity gain.** "Readability improvements" that expanded the code without making it easier to follow. Challenge: can a reader summarize this function in one sentence faster now than before?
- **Conflict with prior audit findings.** The audit reverted a `decision-boundary-extraction` extraction or a `seam-discovery-and-enforcement` seam, citing simplicity. Challenge: was the prior audit's reason engaged with, or just overridden?
- **State opacity persists.** The audit polished naming and control flow but didn't address ambient state, hidden mutations, or implicit dependencies — the hardest source of cognitive load. Challenge: can a reader still not predict outputs from inputs?

## Paired skill

`scenarios/prompt-manager/store/skills/packs/core/cognitive-load-reduction/SKILL.md` — the executable spec. Required reading: `prompt-manager skills read knowledge-observatory-tools` and this PoR doc.

## Cross-references

- [`README.md`](README.md) — registry overview, lifecycle rules, doc + paired skill discipline.
- [`screaming-architecture-audit.md`](screaming-architecture-audit.md) — companion lens; this lens governs *local* readability, that one governs *macro* structure.
- [`decision-boundary-extraction.md`](decision-boundary-extraction.md) — companion lens; named decisions help local readability.
- [`code-cleanup.md`](code-cleanup.md) — companion lens; dead code is cognitive load even when it's "off to the side."
- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview.
