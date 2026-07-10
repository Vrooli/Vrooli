# Template-Convergence Loop

**Status:** canon (v1). The end-to-end workflow for portfolio-wide architecture improvement: improve the ideal once, then converge every scenario toward it mechanically. Owned by the `meta-optimization` team; edits go through `meta-self-improvement` decisions.

This doc is the **index for the loop**. Each stage already has its own canon — this file names them as stations on one conveyor belt and states the governing principle, so an agent who lands on any single stage can see the whole. It deliberately does not restate the stage docs; it links to them.

## The principle

When you want to improve scenario architecture in a way you haven't before — a new contract location, a new substrate seam, a new structural shape — **do not improve scenarios one at a time.** Improve the *copy source* once, prove it, then push the difference down into reusable detection and application so every existing scenario converges toward the new ideal without re-deciding it each time.

> **Judgment flows toward the template; mechanism flows toward the scenarios.**
> Think hard once, at the ideal. Then make convergence increasingly automatic.

This is the [recursive-learning loop](../../VISION.md) applied to architecture itself: each pass makes the *next* architectural improvement cheaper to detect and apply. Improve the gene, not the organism.

## The four stages

```text
1. Improve the ideal     →  edit templates/scenarios/<name>/ (the copy source)
        │                    one edit multiplies across all future scenarios
        ▼
2. Validate the ideal    →  generate + build + test clean; measure it's actually
        │                    better as a copy source (frozen metrics harness)
        ▼
3. Distill the delta     →  crystallize "recognize old shape → transform to new"
        │                    into a skill runnable against any existing scenario
        ▼
4. Mechanize det/apply   →  absorb the deterministic parts into tooling scenarios:
                             detection (auditor rules, test-genie phases, fitness lens)
                             application (improvement campaigns, campaign tracker)
```

### Stage 1 — Improve the ideal (the template)

The canonical copy source is `path:templates/scenarios/<name>/` (today, primarily `path:templates/scenarios/react-vite/`), fed into `template-manager generate`. Every choice there multiplies across N future scenarios, so a new architectural idea lands *here first* — where one edit is worth N. You are changing what gets inherited, not patching one inheritor.

The multiplier-aware lens for deciding *what* to change at the template is [`REFERENCE_PATTERN_FITNESS.md`](REFERENCE_PATTERN_FITNESS.md) — it asks "is this fit to be copied?" (per-replica cost, drift surfaces, contract location, coordinated-edit count), distinct from "is this code good?" (the single-instance lenses in `path:docs/scenario-qa/methods/audit/`). Run the single-instance lenses first; the fitness lens assumes structural soundness.

### Stage 2 — Validate the ideal

A template change is only an improvement if the generated scenario still builds and tests clean *and* measures better on the dimension you targeted. This is the frozen-metrics discipline: a baseline, a testable hypothesis, named metrics, and post-change results, captured as a dated audit record under `path:scenarios/prompt-manager/store/teams/meta-optimization/template-fitness-audit/<artifact-slug>/<YYYY-MM-DD>/`. The `2026-05-04` react-vite record is the worked exemplar (baseline → hypothesis → metrics → results → next-iteration proposal). The template's paired gold-star reference (see [`REFERENCE_SCENARIOS.md`](REFERENCE_SCENARIOS.md)) regenerates to pick up the improvement automatically; its quality is bounded above by the template's.

This stage is also where the loop is honest about iteration: each substrate change re-prices what's left in the template, so the fitness lens is re-run, not banked. A multi-iteration program attacks different tiers of findings over successive passes.

### Stage 3 — Distill the delta into a skill

After Stage 2 the template embodies the better state, but every *existing* scenario is still on the old shape. The improvement is now a **delta**: how to recognize the old pattern and transform it into the new one. Crystallize that delta into a skill that can be run against any existing scenario to bring it up to the template's state — `scenario-improvement-campaign` and `scenario-readiness-review` are the current home for this measure-then-converge work. Skill-shaped guidance matures per [`SKILL_AUTHORING.md`](SKILL_AUTHORING.md) (note especially the "destination over direction" maturity-ladder contract for audit-shaped skills) and [`PROMOTION_LADDER.md`](PROMOTION_LADDER.md).

### Stage 4 — Mechanize detection and application

As the skill matures, its deterministic parts get absorbed into tooling **scenarios**, split across two faces:

- **Detection** — surface the old shape automatically: `scenario-auditor` rules, `test-genie` phases, the [reference-pattern-fitness](REFERENCE_PATTERN_FITNESS.md) audit, and `development-toolchain-validator`'s mechanized maturity signals.
- **Application** — converge scenarios at scale: improvement campaigns and the architecture-cartographer/campaign tracker that ingests findings and tracks remediation by stable ID.

Each pass through Stage 4 makes the *next* trip through Stage 1 cheaper, because the new architectural idea now has somewhere mechanical to be detected and applied. This is where the loop compounds.

## Where each stage's canon lives

| Stage | Question it answers | Canon |
|---|---|---|
| 1. Improve the ideal | What to change at the copy source, and is it worth copying? | [`REFERENCE_PATTERN_FITNESS.md`](REFERENCE_PATTERN_FITNESS.md), `path:templates/scenarios/react-vite/` |
| 2. Validate the ideal | Did it generate/build/test clean and measure better? | [`REFERENCE_SCENARIOS.md`](REFERENCE_SCENARIOS.md) (template↔reference coupling), `path:scenarios/prompt-manager/store/teams/meta-optimization/template-fitness-audit/` |
| 3. Distill the delta | How to bring existing scenarios up to the template | `scenario-improvement-campaign` / `scenario-readiness-review` skills, [`SKILL_AUTHORING.md`](SKILL_AUTHORING.md), [`PROMOTION_LADDER.md`](PROMOTION_LADDER.md) |
| 4. Mechanize det/apply | How detection + application become standing tooling | `scenario-auditor`, `test-genie`, `development-toolchain-validator`, architecture-cartographer campaign tracker |
| (friction in, fixes out) | How signal routes into this loop | [`../meta-optimization/README.md`](../meta-optimization/README.md) |

## When this loop applies — and when it doesn't

✅ A genuinely **new** architectural pattern you want across the portfolio, not just in one scenario.

✅ The improvement is **mechanizable** — the recognize-and-transform delta can eventually become a rule, phase, or campaign.

✅ You are about to hand-edit the same structural change into several scenarios. That's the signal to stop and lift it to the template first.

⚠️ **A one-off fix in a single production scenario.** No multiplier; use the relevant single-instance lens. Lifting it to the template is premature substrate work.

⚠️ **Substrate that doesn't exist yet.** The fitness lens flags candidates; it doesn't authorize extraction. If the proposed home (cli-core, api-core, shared lib) can't accept the change today, that's a separate decision — Vrooli's "don't extract until the third repetition" rule still holds.

⚠️ **Skipping Stage 2.** A template edit that hasn't been generated/built/tested-and-measured is a hypothesis, not an improvement. Don't distill a skill from an unvalidated delta.

## Cross-references

- [`REFERENCE_PATTERN_FITNESS.md`](REFERENCE_PATTERN_FITNESS.md) — Stage 1 detection lens; this loop is the workflow that lens sits inside.
- [`REFERENCE_SCENARIOS.md`](REFERENCE_SCENARIOS.md) — template↔reference coupling and rot triage; the Stage 2 validation substrate.
- [`../meta-optimization/README.md`](../meta-optimization/README.md) — the friction intake that feeds candidate improvements into this loop.
- [`SKILL_AUTHORING.md`](SKILL_AUTHORING.md), [`PROMOTION_LADDER.md`](PROMOTION_LADDER.md) — how the Stage 3 skill matures toward Stage 4 mechanization.
- [`README.md`](README.md) — agent-system canon-doc index.
