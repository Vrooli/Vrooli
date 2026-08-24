---
name: "reviewing-agent-run-efficiency"
description: "Practice method for finding avoidable friction in successful agent runs."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["agent-manager","efficiency","investigation"]
  status: "active"
  requires:
    scenarios: []
    commands: []
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
