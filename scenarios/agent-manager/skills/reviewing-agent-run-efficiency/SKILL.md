---
name: "reviewing-agent-run-efficiency"
description: "Practice method for finding avoidable friction in successful agent runs."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["agent-manager","efficiency","investigation"]
  icon: "search"
  status: "active"
  revision: 3
  createdAt: "2026-02-17T00:00:00Z"
  updatedAt: "2026-09-12T00:00:00Z"
  requires:
    scenarios: ["agent-manager"]
    commands: ["agent-manager run report", "agent-manager run tools", "agent-manager run messages", "agent-manager run events", "agent-manager run result"]
  origin:
    kind: "authored"
---
## Reviewing agent-run efficiency

Start with `agent-manager run report <id>`. Review successful and failed runs
for avoidable friction before diagnosing implementation correctness.

| Signal | Interpretation | Drill-down |
| --- | --- | --- |
| Repeated tool calls | The agent retried an equivalent action without new evidence. | `run tools`, `run messages` |
| Files reread | The context or skill did not preserve a prior read. | `run tools`, `run messages` |
| Long event gap | A tool, environment, or reasoning loop consumed time without visible progress. | `run events` |
| High turns/tokens with a successful result | The run delivered but likely has reusable process friction. | `run messages`, `run result` |

Only call a pattern waste when the report and drill-down evidence rule out a
necessary retry or deliberate verification. Recommend the smallest durable
change: context preservation, a clearer skill, a capability fix, or a CLI
affordance that removes repeated interpretation.

Apply the same test to supervisors, investigators and reviewers. Include their
observation, reasoning, disruption and method-improvement cost in the assessment.
Before reading more evidence, name the decision it could change. A necessary
investigation with a small repair or a retained negative result can be efficient.
Compare similar assignments; event, diff and intervention counts are not accepted
outcomes. Follow `path:docs/agent-system/EFFORT_SUPERVISION.md` for effort metrics,
unknown cost, independent samples and bounded inward improvement.
