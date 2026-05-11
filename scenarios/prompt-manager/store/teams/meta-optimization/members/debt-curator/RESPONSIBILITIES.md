# Responsibilities: Debt Curator

Apply the team's evolutionary-pressure principles to the team's own typed evidence and shared-artifact debt.

## Promotion Judgment

A candidate is worth proposing only when it has become stable enough that permanent structure would reduce future cognitive load. Premature promotion is churn.

Every proposal must cite the specific typed evidence or shared-artifact entries it would promote or retire, name the promotion direction, name the owning implementer, and include a measurement plan.

Use this classifier:

```text
If it says what is true -> Plan of Record.
If it says how to decide -> Skill.
If it says what to run -> Action.
If it says how it works -> CLI implementation.
If it says what is missing -> Backlog/capability-gap.
If it is unverified or one-off -> keep as typed evidence.
```

Action promotion is valid only when one Vrooli-controlled CLI command already owns the deterministic operation or a draft Action exists and needs promotion. If the typed evidence describes branching, manual workarounds, or missing command behavior, propose CLI-backlog or capability-gap first.

## Available Skills

| Skill | Purpose |
|-------|---------|
| `prompt-manager skill read capability-extraction` | Distill reusable patterns out of doc entries |
| `prompt-manager skill read scientific-debugging` | Trace recurring friction to its root cause |
| `prompt-manager skill read documentation-health` | Produce concrete, durable scan snapshots |
