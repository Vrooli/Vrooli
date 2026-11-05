# Usage Intelligence effort

Make Agent Manager (Mizar) the place where an operator learns, with evidence, how
much subscription capacity their agents consume, whether they will run out before
the reset, what is consuming it, and what to change to do the same work with less.
This effort also leaves the analytics, capture and accounting code materially
simpler than it was on 2026-10-29.

This folder is the effort's durable home. It is not a delivery team and has no
heartbeat. One orchestrating agent runs the [goal](GOAL.md), delegates workstreams
to sub-agents, and tracks state here.

| File | Holds | Who edits |
|---|---|---|
| [SPEC.md](SPEC.md) | Destination, target architecture, UX, acceptance criteria, non-goals | Operator-approved; amendments recorded with an ID |
| [EVIDENCE.md](EVIDENCE.md) | Measured baseline (2026-10-29) with commands and numbers | Append-only; re-measurements go in a new dated section |
| [DEBT.md](DEBT.md) | Consolidation and deletion ledger with before/after measures | Workers update rows they close |
| [WORKSTREAMS.md](WORKSTREAMS.md) | Allocation, dependencies, owners, status, forecast, accepted results | Orchestrator |
| [FEEDBACK.md](FEEDBACK.md) | Operator directions, verbatim, with disposition | Orchestrator appends before acting |
| [GOAL.md](GOAL.md) | Orchestrator goal message and worker assignment template | Changes per dispatch |

Resume path: FEEDBACK (open items) → WORKSTREAMS (current) → SPEC (the section a
workstream cites) → code.
