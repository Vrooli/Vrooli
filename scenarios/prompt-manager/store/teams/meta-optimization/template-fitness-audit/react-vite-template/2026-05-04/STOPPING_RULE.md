# Stopping Rule — when the multi-iteration program declares done

The program ends when **any** of the following are true. The agent executing the iteration evaluates the rule in Phase E; the operator confirms in the iteration handoff.

## 1. Architectural target reached

Both of these hold simultaneously across all 6 measurement scenarios in the most recent `RESULTS.md` column:

- No scenario has more than **2** central-registry edits (`central_edits ≤ 2`).
- Adding a new domain end-to-end (Scenario 2) costs **≤ 200** total non-test lines (`lines_added ≤ 200`).

The 200-line floor is informed by the iteration-1 audit's anecdotal estimate (~500 lines today, much of which is genuine domain code). 200 leaves room for the genuine code while collapsing the boilerplate.

## 2. Diminishing returns

Two consecutive iterations achieve **< 20% improvement** on the same metric they targeted. This signals that the substrate-extractable cost has been mined out and further iterations are paint.

Concretely: if iteration N's hypothesis predicted (and achieved) a 30% drop on `lines_added` for Scenario 2, and iteration N+1's hypothesis predicts (and achieves) only an 8% additional drop on the same metric, the program stops *unless* iteration N+2's hypothesis names a *different* metric to attack with strong evidence.

## 3. Hard cap

**Six iterations**, regardless of whether rules 1 or 2 fired. After iteration 6, the operator decides whether the program continues, pivots, or sunsets. The cap exists so a poorly-converging program doesn't sprawl indefinitely.

## 4. External invalidation

Any of:

- The react-vite template is demoted from gold-star reference (per `docs/agent-system/REFERENCE_SCENARIOS.md` §"Demotion rules").
- A separate substrate change (e.g., a Vrooli-wide CLI overhaul) lands and renders this harness's measurements meaningless. The new substrate gets its own harness or replaces this one.
- The program's owner (currently `meta-optimization` / `toolchain-validator`) decides the marginal value of further iterations is below the cost of running them.

## On firing the rule

The terminal iteration is responsible for:

1. Writing a final `STOPPING_RULE_FIRED.md` next to this file, naming which rule fired and citing the `RESULTS.md` evidence.
2. Updating `docs/agent-system/REFERENCE_SCENARIOS.md` `Last audit` column on the react-vite row to point at the final results file.
3. Filing a `meta-self-improvement` decision recording the program's outcome (target reached, diminishing returns, hard cap, or external invalidation) and what carries forward.
4. **Not** archiving the harness — leave the directory in place for citation. Future audits of unrelated templates can model their harnesses on this one.
