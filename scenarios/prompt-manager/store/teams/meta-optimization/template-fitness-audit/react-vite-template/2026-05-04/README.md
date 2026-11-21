# React-Vite Template Fitness — Measurement Harness

**Created**: 2026-05-04 — iteration 1 of the multi-iteration program planned at `/home/matthalloran8/.claude/plans/thanks-for-the-postgres-wise-hearth.md`.

**Slug**: `react-vite-template` (matches `docs/agent-system/REFERENCE_SCENARIOS.md` registry row).

## What this is

A frozen yardstick for measuring whether a given iteration of substrate work makes the `templates/scenarios/react-vite/` template more fit as a copy source. The lens — reference-pattern fitness, four sub-lenses — is documented at `docs/agent-system/REFERENCE_PATTERN_FITNESS.md`. This harness operationalizes that lens into reproducible numbers.

## Files

| File | Role | Populated when |
|------|------|---------------|
| [`README.md`](./README.md) | This file. Discovery + how-to-run. | Phase A (iteration 1) |
| [`SCENARIOS.md`](./SCENARIOS.md) | The 6 frozen workflow scenarios + their measurement recipes. **Frozen at iteration 1**; revising requires an explicit "scenario revision" entry. | Phase A |
| [`METRICS.md`](./METRICS.md) | The 4 numerical metrics + 1 meta-metric. Computation formulas. | Phase A |
| [`STOPPING_RULE.md`](./STOPPING_RULE.md) | When the multi-iteration program declares done. | Phase A |
| [`HYPOTHESIS.md`](./HYPOTHESIS.md) | Iteration-1 hypothesis. Predicts numerical movement before the substrate change ships. | Phase A |
| [`BASELINE.md`](./BASELINE.md) | Numbers from running the 6 scenarios against the **pre-iteration-1-substrate** template tree. | Phase B (iteration 1) |
| [`RESULTS.md`](./RESULTS.md) | Numbers from running the 6 scenarios against the **post-iteration-N-substrate** tree. Hypothesis grade. Grows a column per iteration. | Phase E (each iteration) |
| `ITERATION_<N+1>_PROPOSAL.md` | What to attack next. Iteration N writes proposal for N+1. | Phase E |

## Who runs this

The agent executing each iteration's plan. The `toolchain-validator` member of `meta-optimization` owns the harness as the longer-cadence template audit lives there.

## How to run an iteration

1. **Read** `HYPOTHESIS.md` for what the current iteration is testing, and `SCENARIOS.md` + `METRICS.md` for the measurement protocol.
2. **Phase B (iteration 1 only)** — measure the pre-substrate template against all 6 scenarios. Write rows into `BASELINE.md`.
3. **Phase C/D** — apply the substrate change per the iteration plan.
4. **Phase E** — measure the post-substrate template against the same 6 scenarios. Write rows into `RESULTS.md` next to the baseline cells. Grade the hypothesis. Author `ITERATION_<N+1>_PROPOSAL.md`.

Subsequent iterations skip Phase B and use the prior iteration's `RESULTS.md` numbers as their baseline (no re-baselining unless the substrate change is large enough to invalidate prior measurements; that's an explicit decision recorded in the new iteration's hypothesis).

## Determinism

The recipes in `SCENARIOS.md` must produce the same numbers across runs. If Phase B records varying numbers, the recipe is broken — fix the recipe, not the number. Lock formatter versions in the recipe (`gofumpt` from the cli-core toolchain; `prettier` from `templates/scenarios/react-vite/ui/package.json`).

## Retention

This harness is durable across iterations and lives in tracked git for citation from future docs and resilience to machine wipes. The **final** iteration of the multi-iteration program (whichever one lands the stopping rule) is responsible for archiving or sunsetting the harness. Until then, every iteration extends `RESULTS.md` rather than replacing it.

## Smoke-scenario cleanup recipe

Every iteration's end-to-end gate generates a throwaway scenario from the post-iteration template, runs the full gate, then deletes it. This recipe is the canonical cleanup — the inline version in iteration-1's plan §10.4 was incomplete and left orphaned proto codegen behind (resurfaced as 62 staged-deletion entries during the iteration-1 polish pass).

The trap: gen directories use **two naming conventions**. Schemas, Go gen, and TypeScript gen use the dash form (`template-fitness-iter1-smoke`); Python gen uses the underscore form (`template_fitness_iter1_smoke`). Iterating only over `gen/{go,typescript,python}/<dash-form>` misses both Python and the `gen/typescript/js/` mirror.

```bash
# Cleanup throwaway smoke scenario — handles every gen-dir + naming convention.
SCENARIO="<scenario-name-with-dashes>"
SCENARIO_UNDERSCORE="${SCENARIO//-/_}"

vrooli scenario stop "$SCENARIO" 2>/dev/null || true
rm -rf "scenarios/$SCENARIO"
rm -rf "packages/proto/schemas/$SCENARIO"
rm -rf "packages/proto/gen/go/$SCENARIO"
rm -rf "packages/proto/gen/typescript/$SCENARIO"
rm -rf "packages/proto/gen/typescript/js/$SCENARIO"
rm -rf "packages/proto/gen/python/$SCENARIO_UNDERSCORE"

# Idempotent regen. Catches anything missed and clears stale entries from
# packages/proto/gen/typescript/package.json's exports map.
( cd packages/proto && make generate )

# Confirm: only the deletions you intend, nothing else.
git status packages/proto/
```

If `git status packages/proto/` shows unexpected `D` entries, that's a prior iteration's residue surfacing — investigate before committing rather than blindly staging.

## Cross-links

- Strategic canon: [`docs/agent-system/REFERENCE_PATTERN_FITNESS.md`](../../../../../../../../docs/agent-system/REFERENCE_PATTERN_FITNESS.md)
- Registry: [`docs/agent-system/REFERENCE_SCENARIOS.md`](../../../../../../../../docs/agent-system/REFERENCE_SCENARIOS.md) — `Last audit` column on the `reference-react-vite` row links here.
- Iteration-1 plan: `/home/matthalloran8/.claude/plans/thanks-for-the-postgres-wise-hearth.md`
- Lens skill: `prompt-manager skill read reference-pattern-fitness`
