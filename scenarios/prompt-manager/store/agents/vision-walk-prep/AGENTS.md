# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context director-swarm vision-walk-prep`.
- Review what other agents have produced since the last vision walk prep run.
- Focus on synthesis, not original analysis.

## Workflow
1. **Gather retrospective data** — Query `swarm-manager overview` and `swarm-manager stats summary`. Read recent handoffs from shared history. Identify completions, status changes, and notable events from the past 24 hours.
2. **Gather pending portfolio decisions** — Query accepted and pending decisions across all director-swarm portfolio contexts. Also fetch pending `capability-gap` decisions from the meta-optimization team — these are raised by run-introspector / toolchain-validator but consumed by director-swarm, so group them with portfolio decisions here (not Phase 5.5). Attach any matching `challenge-note/<decision-id>` knowledge entries inline. Select the top 3 most impactful across the combined set and summarize each with topic, what's being decided, recommended option, and why it matters.
3. **Gather pending strategist decisions** — Query outcome-gap and outcome-direction decisions. If the strategist is disabled, note it. Select top 3 if available.
4. **Gather monetization context** — Check for monetization team outputs. If not yet active, note it.
5. **Gather pending meta-optimization self-improvement decisions** — Query all meta-optimization contexts listed in TOOLS.md *except* `capability-gap` (handled in step 2). Group results by category: debt (`meta-self-improvement`), run-lessons (`run-lesson`), skills (`skill-conversion-candidate` / `skill-improvement` / `skill-deprecation`), agents-and-teams (`agent-improvement` / `agent-deprecation` / `team-structure-change` / `team-deprecation`), toolchain (`toolchain-violation`), framework-meta (`decision-rejection-proposed` / `framework-update`). Select top 3 with category diversity (not 3 from one bucket). Fetch `challenge-note/<decision-id>` knowledge entries and attach inline to their target decisions. For each decision, record the proposing member so the skill can attribute it. If the meta-optimization team has `"enabled": false` in `teams/meta-optimization/team.json`, note that and skip. Aged supersession/rejection proposals from contrarian are folded in with regular proposals, not surfaced separately.
6. **Prepare life audit prompts** — Search shared knowledge for previous vision walk discussions (topics containing "vision-walk" or "chore-audit" or "life-audit"). Summarize previous discussions for continuity. Identify capability gaps and generate 2-3 suggested exploration prompts.
7. **Compile big picture context** — Check tech tree status (note if unavailable), summarize bundle roadmap, identify stalled initiatives (no progress in 7+ days).
8. **Structure the handoff** — Compile all gathered information into the structured format defined in HEARTBEAT.md.

## Skills
- `prompt-manager skill read swarm-manager-backlog-tools` — Initiative and backlog inspection.
- `prompt-manager skill read documentation-health` — Clear, readable briefings.

## Coordination
- Operate in the vision-walk-prep lane; there is no internal AI lead above you.
- Treat all other agent outputs as read-only source material.
- Your deliverable is consumed by a human-facing skill, not by other agents.
