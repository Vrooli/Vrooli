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
  revision: 4
  createdAt: "2026-02-17T00:00:00Z"
  updatedAt: "2026-09-14T00:00:00Z"
  requires:
    scenarios: ["agent-manager"]
    commands: ["agent-manager run efficiency", "agent-manager run recent", "agent-manager run report", "agent-manager run tools", "agent-manager run messages", "agent-manager run events", "agent-manager run result"]
  origin:
    kind: "authored"
---
## Reviewing agent-run efficiency

Start with `agent-manager run efficiency --preset 24h --group-by profile --json`
to choose a bounded cohort. Use `--compare previous` when testing a change.
Then select independent sample runs from the largest or most suspicious cohort
and read `agent-manager run recent --json` for the run's durability lane and
evidence before starting each drill-down with `agent-manager run report <id>`.

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

Treat `dispositions` as terminal-status evidence only. They do not establish
that a successful run produced useful work. Treat `freshness.available=false`,
partial status, unknown rows, or a small denominator as insufficient evidence.
Treat `durability.verdict=durable` or `signal_present` as outcome evidence only
for the bounded work/evidence it names; `before_analysis_epoch`, `unknown`,
unlinked, or empty coverage must remain unknown rather than productive.
Use baseline deltas to select an experiment, not as causal proof: compare
similar assignments and preserve workload, model, and policy differences.

Do not automatically disable teams from this report. A policy change requires
repeated independent samples, a documented expected-outcome measure, and an
owner-approved bounded experiment.

Apply the same test to supervisors, investigators and reviewers. Include their
observation, reasoning, disruption and method-improvement cost in the assessment.
Before reading more evidence, name the decision it could change. A necessary
investigation with a small repair or a retained negative result can be efficient.
Compare similar assignments; event, diff and intervention counts are not accepted
outcomes. Follow `path:docs/agent-system/EFFORT_SUPERVISION.md` for effort metrics,
unknown cost, independent samples and bounded inward improvement.
