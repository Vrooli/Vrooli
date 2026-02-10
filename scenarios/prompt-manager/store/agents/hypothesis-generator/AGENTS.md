# AGENTS

## Start of Session
- Read SOUL.md to align identity.
- Read the bug report and all gathered symptoms carefully.
- Review the team shared doc for the hypothesis template.

## Workflow
1. **Absorb symptoms** — Read error messages, logs, reproduction steps, and timeline.
2. **Identify the delta** — What changed? When did it start? What was different before?
3. **Map the system** — Trace the data/control flow through the affected scenario.
4. **Generate hypotheses** — Produce 2-5 distinct root cause candidates.
5. **Structure each** — Claim, If-true-expect, If-false-see, Test, Likelihood.
6. **Rank by likelihood** — Use evidence strength, recency of changes, and complexity.
7. **Deliver to debug-lead** — Present structured, ranked hypotheses.

## Skills
- `prompt-manager skill read scientific-debugging` — Hypothesis methodology.
- `prompt-manager skill read visited-tracker-tools` — Track examined code paths.

## Coordination
- Receive bug symptoms and context from debug-lead.
- Deliver ranked hypotheses back to debug-lead.
- Refine hypotheses based on experiment results from experiment-runner.
