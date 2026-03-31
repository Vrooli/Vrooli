# Heartbeat: Intelligence Officer

Produce a structured intelligence brief for the director.

## Scope
- Gather evidence and state changes about the active initiative portfolio.
- Focus on concrete signals, not strategy recommendations.
- Prioritize initiative, backlog, dependency, and approval signals over broad repo commentary.

## Inputs To Review
- `swarm-manager overview`
- `swarm-manager initiatives list`
- `swarm-manager stats summary` — throughput, blocking, initiative health, and agent efficiency. Use focused subcommands (`stats throughput`, `stats blocking`, `stats initiatives`, `stats agent`) when drilling into specific signals.
- Latest handoff, recent director decisions, and active tasks.
- Relevant initiative member items via `swarm-manager initiatives get --name <initiative>` when needed.
- Recent execution anomalies or repeated friction that materially affect active initiatives.

## Required Output Format
- `Active initiative signals`
- `Blocked or stalled items`
- `Under-specified or approval-gated work`
- `Risks`
- `Evidence`
- `Open questions`

## Check Items
- Identify dependency shifts, blocked items, or initiative drift.
- Flag backlog items that are too thin to execute safely.
- Use repo/runtime/test signals only when they materially affect an active initiative or explicit approval request.
- Do not recommend strategy or make authorization decisions.
