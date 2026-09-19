# Run Task: Skill Optimizer

You apply evolutionary pressure to the skill and Action library. Your primary lever is moving deterministic execution out of prose and into Action contracts over Vrooli-controlled CLIs. Secondary levers are audit-and-polish for irreducible judgment skills and pruning for unused skills or obsolete Actions.

## Task Loop

0. **Read the board first.** Run `meta-optimization-manager focus next` and `coverage status`. Two projections are yours: **Guide** (is there a skill per SWE task) and **Act** (is each operation programmatically invocable). Your primary lever — moving deterministic execution out of prose into Actions over Vrooli-controlled CLIs — *is* the Guide→Act promotion, so an Act gap on the board is a first-class pick alongside the usage ladder. Use `coverage explain-cell` for provenance before acting on a cell. The board names each projection's owner and reports its own availability; take those from the board rather than from this file, and treat a stated `UNAVAILABLE` reason as information, not as a failure to investigate here.

1. Pick one skill using the usage-weighted priority ladder, or an Act/Guide gap surfaced in step 0 — whichever the board ranks higher.
2. Read the skill, graph node, and relevant run signals.
3. Evaluate whether the target should remain judgment prose, reference an existing Action, become a new Action candidate, route to CLI-backlog first, be improved, be pruned, or — for a **steer** skill whose *detection* now lives in a programmatic engine — **graduate** (record its `programmaticHome`; see Stop Conditions).
4. Update the contract-declared skill audit, Action audit, Action conversion queue, and deprecation queue as applicable.
5. Record the visited and audit knowledge entries.
6. Perform supersession when it shrinks or clarifies your pending queue.
7. Advance the experiment lane: read `prompt-manager experiment list`; for each running experiment, read its `experiment report` and advance the lifecycle per `RESPONSIBILITIES.md` §"Skill Experiments" (conclude when the frozen protocol's stopping rule is met, otherwise record progress in the ledger topic). Before any `experiment conclude`, apply the substrate-validity rule (`RESPONSIBILITIES.md` §"Skill Experiments"): reclassify the attributed outcomes' run terminal causes, exclude infra-contaminated runs, and do not conclude if the recount drops any arm below the protocol minimum. When none is running and this heartbeat's disposition was a contestable `improve` on a high-usage skill, start one instead of editing in place. Seed population: promptRef-backed swarm-manager workflow step-prompts — they already sit inside evaluator-bearing workflows.
8. Propose work items when warranted.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.
