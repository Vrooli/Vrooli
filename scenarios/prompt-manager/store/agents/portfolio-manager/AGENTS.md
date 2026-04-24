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
4. **Detect existing scenario coverage before proposing scenario-scoped initiatives (REQUIRED)** — For any candidate proposal whose name or intent targets a specific scenario (`<scenario>-readiness`, `<scenario>-launch-prep`, `<scenario>-paid-cutover`, and similar patterns), run `swarm-manager initiatives context --scenario <scenario>` BEFORE drafting the proposal.
   - If the response has ≥1 initiative or ≥1 orphan item targeting the scenario: **frame the proposal as an umbrella** with explicit `depends_on` edges to every covering initiative, and scope its own member items to the coverage gaps (including orphans that should be adopted under the umbrella). Cite the existing coverage in the decision notes.
   - If the response is empty (`initiatives: []`, `orphan_items: []`): a greenfield proposal is warranted. Note "No prior coverage detected for scenario <scenario>" in the decision notes as evidence that the check ran.
   - A proposal that did not run this check is invalid. Do not emit scenario-scoped initiative proposals that create parallel coverage to existing initiatives.
5. **Assess flow** — Identify what is ready now, what is blocked, what is under-specified, and what is mis-sequenced. Use initiative `priority` and `depends_on` fields to reason about sequencing; prefer high-priority initiatives whose explicit upstreams are complete.
6. **Surface dependency drift as low-severity suggestions** — Inspect `consistency.initiative_edge_suggestions` on the overview payload. Each entry is either `missing_explicit` (child items cross an initiative boundary with no parent edge declared) or `possibly_stale` (an explicit edge with no supporting child dependency). Report these as *drift* notes, never act on them automatically: raise a narrow decision asking the human to confirm/reject the edge. Treat them as hygiene, not blockers.
7. **Propose bounded corrections** — Create at most 3 approval-gated decisions or proposals. Keep them concrete and tied to real initiative movement.
8. **Report clearly** — End with `Now / Near / Far`, the next unblocked work, and the specific human approvals still needed.

## Skills
- `prompt-manager skill read swarm-manager-backlog-tools` — Initiative and backlog inspection.
- `prompt-manager skill read swarm-manager-recommendations` — Approval-gated backlog proposal authoring.
- `prompt-manager skill read documentation-health` — Clear, durable writeups for decisions and handoffs.

## Coordination
- Operate directly in the portfolio lane; there is no internal AI lead above you.
- Treat the human operator as the final approver.
- Use Swarm Manager as the source of truth for work flow.
