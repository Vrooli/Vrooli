# Mizar: understand and improve how your agents work

Agent Manager (Mizar) is Vrooli's orchestration and evidence layer for supported
agent workloads. Its efficiency positioning complements Aquila: Aquila helps a
person manage the daily conversation/review loop; Mizar helps an operator
investigate how work proceeds, where friction repeats and whether a change helps.
This direction reflects operator feedback and current owner documentation on
2026-10-21. It is not a measured cost-reduction or conversion claim.

## Audience and useful outcome

Target builders and teams who run enough agent work that repeated failures,
unclear outcomes, weak handoffs or excessive intervention have consequences.
The job is to turn "the agents keep getting stuck" into inspectable evidence and
a bounded improvement: identify a recurring pattern, inspect representative runs,
change one relevant workflow/profile/tool, then compare suitable evidence.

**Make agent work easier to understand and improve.** Lead with recurring
friction and accountable execution, then connect that to more useful work from
existing time, compute and provider spend. Governance, scopes and budgets are
part of reliable operation, not a substitute for useful results.

## Evidence-backed capability map

| Capability | Customer-facing value | Proof boundary |
|---|---|---|
| Run/workflow history, immutable policy snapshots and lifecycle events | Know what was attempted and under which conditions | Native supported owner records; terminal status alone does not establish useful output |
| Friction messages, tool-friction profiles and recurring fingerprints | Investigate repeated obstacles rather than guessing from one failure | Classifier version, evidence, time window, omissions and representative cases matter |
| Duration, usage/cost basis and runner/model/profile breakdowns | Locate expensive or slow patterns for investigation | Reported/estimated cost is distinct from provider invoices; unavailable values remain unknown |
| Cohort comparisons and improvement records | Compare a change against suitable prior workloads | Comparable populations and outcome checks are needed; a previous-window delta is not a causal savings estimate |
| Roles, permissions, bounded execution and approval handling | Run supported work with explicit operating controls | Actual runner/target support and authority still apply; no universal external-terminal coverage |

Sources: [Measures and operational stats](../reference/operational-stats.md),
[Workflow adoption](../guides/workflow-adoption.md),
`../../.vrooli/program-runtime/efficiency-report.json`, and the recurring-friction
owner programs. Inspect their current contracts before demonstrating a report.

## Separate the workloads

The deterministic `agent-manager.efficiency-report` covers its durable invocation
projection and excludes imported and interactive workloads from that operational
cohort. Imported conversations, interactive sessions and cohort reports
have distinct evidence paths. Do not imply that the invocation report measures
all Aquila conversations, every vendor subscription or every external terminal.

Aquila can supply a useful human-facing coordination surface. Agent Manager's
run/session attribution must actually exist for the claimed workload; best-effort
attribution and imported history are not equivalent to complete live metering.

## Launch and offer strategy

Demonstrate one evidence-to-improvement loop using source-owned reports and a
clearly bounded population. Explain the limitation of any synthetic or imported
example. Keep Mizar's deeper investigation distinct from Aquila's daily interface;
one product's availability does not establish that the other is bundled.

Validate willingness to pay through recurring operational use and a measured
customer problem. Useful signals include less repeated friction, fewer manual
recoveries and stronger verified outcomes for comparable work. Subscription or
compute savings require separate billing and outcome evidence. The current
Offer Desk release path marks agent-manager as `IDEA`; it is not a released
commercial package merely because the local service works.

A unified provider allowance/reset view is an adjacent discovery opportunity.
Provider-native account APIs, freshness, privacy and terms must be evaluated
before it becomes a product promise or input to any automated selection policy.
