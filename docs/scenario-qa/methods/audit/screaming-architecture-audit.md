# Audit Technique: Screaming Architecture Audit

**Status:** v1 (paired with existing `screaming-architecture-audit` skill, 2026-05-03). The skill predates this PoR doc; this entry closes the `skillless canon` smell by giving the technique a strategic-canon home.

## Definition

Audit whether the scenario's **internal architecture and structure clearly express its purpose and mental model**. The codebase should "scream" what the scenario does — top-level structure makes the domain obvious, names reflect domain concepts, modules have clear focused responsibilities, the physical file/folder layout aligns with the logical architecture.

The full procedure (build mental model from PRD/docs → map current logical architecture → inspect physical structure → align via incremental refactor → simplify dependencies → safe-refactoring guidelines) lives in the paired skill. This document is the strategic-canon side: when the lens applies, when it backfires, and what the qa-contrarian watches for.

## When it applies

✅ **God modules / files doing too many things.** A handful of files contain most of the scenario's logic; new contributors can't tell where a feature lives.

✅ **Feature logic scattered across unrelated locations.** A single user-visible behavior touches files spread across the tree with no obvious organizing principle.

✅ **Names hide intent.** Module/function/type names use technical categories (`utils`, `helpers`, `manager`, `service`) instead of domain terms — opening a file doesn't reveal what the scenario does.

✅ **Physical layout diverges from mental model.** Reading the PRD vs. browsing the file tree gives two different impressions of what the scenario is.

✅ **Documentation drift.** `docs/concepts/ARCHITECTURE.md` (or equivalent) and the actual code structure no longer match.

✅ **New scenarios moving past the prototype phase.** As scope grows, naming and grouping decisions made early start to mislead — this is the right lens to revisit.

## When it backfires

⚠️ **Pre-product scenarios still being scoped.** When the domain itself is unstable, "make it scream its purpose" produces ceremony around a name that may change next week. Wait until the domain mental model has settled.

⚠️ **Mature scenarios with high external coupling.** When module names appear in stable APIs, public docs, downstream imports, or partner integrations, renaming for clarity can break consumers. Cost may exceed benefit.

⚠️ **Big-bang rewrites.** The skill explicitly forbids these; when the lens is misapplied as "redesign everything," it becomes the rewrite the skill warned against. Stay incremental.

⚠️ **As a substitute for cross-cutting concerns.** Architecture-alignment improvements don't surface security holes, performance regressions, or invariant violations. Pair with the appropriate lens rather than expecting this one to cover everything.

⚠️ **Trivial scenarios.** A scenario with two files and one entry point doesn't benefit from this lens; the structure is already obvious. Apply where there's enough surface area for confusion to exist.

## What the qa-contrarian watches for

The `qa-contrarian` member challenges audit outcomes; for `screaming-architecture-audit` specifically, watch for:

- **Cosmetic findings.** Renames and file shuffles that don't materially improve "what does the scenario do." The skill's Output Expectations explicitly rule these out; contrarian challenge: did this change actually make the scenario clearer, or just different?
- **Domain-vocabulary appropriation.** The auditor pulled a domain term from the PRD and used it on a module that doesn't actually correspond to that concept — the rename misleads more than it clarifies. Challenge: does the new name's promise match the module's contents?
- **Boundary thrashing.** The audit moved code across boundaries in a way that conflicts with `boundary-of-responsibility-enforcement`'s rules (e.g., domain logic dragged into entrypoint files for "co-location"). Challenge: did the restructure improve or degrade responsibility ownership?
- **Documentation gap left open.** The audit found drift between docs and code but only updated code, not docs (or vice versa). Challenge: is the architecture finding visible to the next reader who relies on docs?
- **Structural change with no test coverage delta.** Modules moved or split without tests being adjusted to match the new boundaries; tests now exercise the wrong layer. Challenge: do tests still describe the scenario's intent at the new structure?

## Paired skill

`scenarios/prompt-manager/store/skills/packs/core/screaming-architecture-audit/SKILL.md` — the executable spec. Required reading: `prompt-manager skills read knowledge-observatory-tools` and this PoR doc.

## Cross-references

- [`architecture-validation-responsibilities`](../../../reference/architecture-validation-responsibilities.md) — the doctrine behind this lens's L5 ("programmatic drift checks") rung: the four validation responsibilities, the test-genie↔cartographer seam, and the audit→campaign loop (`test-genie execute <scenario> --preset architecture-audit` → `architecture-cartographer campaign …`).
- [`README.md`](README.md) — registry overview, lifecycle rules, doc + paired skill discipline.
- [`boundary-of-responsibility-enforcement.md`](boundary-of-responsibility-enforcement.md) — companion lens; this lens governs *names and grouping*, that one governs *who owns what*.
- [`cognitive-load-reduction.md`](cognitive-load-reduction.md) — companion lens; clarity at the *local readability* layer pairs with clarity at the *macro structure* layer.
- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview.
