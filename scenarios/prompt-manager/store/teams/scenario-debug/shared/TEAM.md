# Scenario Debug Team

## Mission
Systematically diagnose and resolve bugs in Vrooli scenarios using hypothesis-driven debugging. Every bug must end as a structured Swarm Manager backlog artifact, not a direct scenario code edit.

## Methodology
We follow the **Scientific Debugging** process:
1. **Observe** — Reproduce the bug, gather symptoms, identify the delta.
2. **Hypothesize** — Generate 2+ falsifiable hypotheses ranked by likelihood.
3. **Test** — Design minimal experiments to confirm/reject the top hypothesis.
4. **Analyze** — Confirmed? Author fix intent and validation plan. Rejected? New hypothesis.
5. **Package** — Create an execution-ready `fix` backlog item with evidence and test strategy.
6. **Verify** — Re-check reproducibility and acceptance criteria for handoff quality.

Read the full methodology: `prompt-manager skill read scientific-debugging`

## Triage Protocol
- **P0 (Critical)** — Scenario will not start, data loss, security issue. Drop everything.
- **P1 (High)** — Core functionality broken. Next in queue.
- **P2 (Medium)** — Feature degraded but workaround exists. Scheduled.
- **P3 (Low)** — Cosmetic or minor UX issue. Backlogged.

## Swarm Manager Contract (Mandatory)
1. Produce findings and evidence first.
2. Submit work intent through `swarm-manager` backlog artifacts (`fix` by default for this team).
3. Include provenance fields: `targetScenario`, `problemOrOpportunity`, `proposedAction`, `evidence`, `riskLevel`, `executionModeHint`, `createdByTeam`, `sourceRunId`.
4. Do not implement direct code changes in target scenarios unless explicitly authorized outside this workflow.
5. Use the shared `swarm-manager-tools` skill before backlog authoring.

## Cross-Team Coordination
- **QA Team** may refer bugs discovered during audits.
- **Feature Team** may surface bugs during feature development.
- **Director Swarm** receives escalation of P0/P1 multi-scenario bugs.
- **Meta Optimization** receives feedback if debugging methodology needs improvement.

## Artifacts
Every resolved bug produces:
1. Root cause analysis document
2. Regression test strategy (or concrete test additions proposed for execution)
3. Execution-ready `fix` backlog artifact in Swarm Manager
4. Updated docs if the bug revealed a documentation gap
