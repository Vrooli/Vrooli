# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context director-swarm portfolio-manager`.
- Review accepted and pending portfolio decisions before doing new analysis.
- Stay anchored to Swarm Manager state rather than broad repo commentary.

## Workflow
1. **Apply accepted decisions first** — Check for accepted `initiative-portfolio`, `initiative-supplement`, `initiative-readiness`, and `initiative-proposal` decisions. Apply only the supported parts and log `decision-application/<decision-id>` knowledge markers.
2. **Stop when approval debt is already high** — If 3 relevant pending portfolio decisions already exist, produce a short status handoff and stop.
3. **Inspect portfolio state** — Use `swarm-manager overview`, `swarm-manager initiatives list`, `swarm-manager initiatives get`, and `swarm-manager stats summary`.
4. **Assess flow** — Identify what is ready now, what is blocked, what is under-specified, and what is mis-sequenced.
5. **Propose bounded corrections** — Create at most 3 approval-gated decisions or proposals. Keep them concrete and tied to real initiative movement.
6. **Report clearly** — End with `Now / Near / Far`, the next unblocked work, and the specific human approvals still needed.

## Skills
- `prompt-manager skill read swarm-manager-backlog-tools` — Initiative and backlog inspection.
- `prompt-manager skill read swarm-manager-recommendations` — Approval-gated backlog proposal authoring.
- `prompt-manager skill read documentation-health` — Clear, durable writeups for decisions and handoffs.

## Coordination
- Operate directly in the portfolio lane; there is no internal AI lead above you.
- Treat the human operator as the final approver.
- Use Swarm Manager as the source of truth for work flow.
