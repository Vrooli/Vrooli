# Effort supervision (retiring)

This page described supervising finite delivery teams, a construct RULES.md retires ([RULES.md](RULES.md)). It remains only because team records under `scenarios/prompt-manager/store/teams/` still cite it, and is deleted with them.

Where its facts went:

- **Commands, formats, goal homes, admission and the owner-principal rules** are in the Agent Manager reference, [effort-supervision.md](../../scenarios/agent-manager/docs/reference/effort-supervision.md).
- **Method** is in the skills `large-effort-supervision` and `large-effort-orchestration`.
- **Open reliability work** is in the design-to-implementation research topic, not in this repository.
- **The full prior text** is kept in the convergence topic's `retired-docs/`.

Authority that has no other home:

- The orchestrator selects and admits epochs; its mechanical gate plus the supervising manager's sampled audits decide acceptance. Workers never accept.
- The supervising manager may read goal homes and owner state, rerun an epoch's exit gate, repair shared infrastructure in its owning scenario, and write `FEEDBACK.md` entries. It never edits epoch files, directs workers, or changes a destination, acceptance rule, owner grant, secret, production system, paid service or unapproved dependency; those need an operator amendment.
- Delivery execution stays with Agent Manager. Work outside the approved destination is filed to its owner (`report-bug`).

The limits that stop a delivery effort are its granted limits and authority (see "Authorization and change classification" in [SCENARIO_DEVELOPMENT.md](SCENARIO_DEVELOPMENT.md)); planned pace is a friction signal (R2).
