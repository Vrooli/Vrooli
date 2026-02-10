# Scenario Debug Team

## Mission
Systematically diagnose and resolve bugs in Vrooli scenarios using hypothesis-driven debugging. Every bug gets a root cause analysis, a regression test, and a minimal fix.

## Methodology
We follow the **Scientific Debugging** process:
1. **Observe** — Reproduce the bug, gather symptoms, identify the delta.
2. **Hypothesize** — Generate 2+ falsifiable hypotheses ranked by likelihood.
3. **Test** — Design minimal experiments to confirm/reject the top hypothesis.
4. **Analyze** — Confirmed? Fix it. Rejected? New hypothesis.
5. **Fix** — Write failing test first, then minimal fix.
6. **Verify** — Manual verification + full test suite.

Read the full methodology: `prompt-manager skill read scientific-debugging`

## Triage Protocol
- **P0 (Critical)** — Scenario will not start, data loss, security issue. Drop everything.
- **P1 (High)** — Core functionality broken. Next in queue.
- **P2 (Medium)** — Feature degraded but workaround exists. Scheduled.
- **P3 (Low)** — Cosmetic or minor UX issue. Backlogged.

## Handoff Protocol
1. debug-lead assigns hypothesis-generator to produce hypotheses.
2. debug-lead selects top hypothesis, assigns experiment-runner.
3. On confirmation, debug-lead assigns fix-engineer.
4. fix-engineer reports back with: regression test, fix, full suite results.

## Cross-Team Coordination
- **QA Team** may refer bugs discovered during audits.
- **Feature Team** may surface bugs during feature development.
- **Director Swarm** receives escalation of P0/P1 multi-scenario bugs.
- **Meta Optimization** receives feedback if debugging methodology needs improvement.

## Artifacts
Every resolved bug produces:
1. Root cause analysis document
2. Regression test
3. Fix commit with detailed message
4. Updated docs if the bug revealed a documentation gap
