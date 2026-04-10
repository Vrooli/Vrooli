# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context director-swarm vision-walk-prep`.
- Review what other agents have produced since the last vision walk prep run.
- Focus on synthesis, not original analysis.

## Workflow
1. **Gather retrospective data** — Query `swarm-manager overview` and `swarm-manager stats summary`. Read recent handoffs from shared history. Identify completions, status changes, and notable events from the past 24 hours.
2. **Gather pending portfolio decisions** — Query accepted and pending decisions across all portfolio contexts. Select the top 3 most impactful pending decisions and summarize each with topic, what's being decided, recommended option, and why it matters.
3. **Gather pending strategist decisions** — Query outcome-gap and outcome-direction decisions. If the strategist is disabled, note it. Select top 3 if available.
4. **Gather monetization context** — Check for monetization team outputs. If not yet active, note it.
5. **Prepare life audit prompts** — Search shared knowledge for previous vision walk discussions (topics containing "vision-walk" or "chore-audit" or "life-audit"). Summarize previous discussions for continuity. Identify capability gaps and generate 2-3 suggested exploration prompts.
6. **Compile big picture context** — Check tech tree status (note if unavailable), summarize bundle roadmap, identify stalled initiatives (no progress in 7+ days).
7. **Structure the handoff** — Compile all gathered information into the structured format defined in HEARTBEAT.md.

## Skills
- `prompt-manager skill read swarm-manager-backlog-tools` — Initiative and backlog inspection.
- `prompt-manager skill read documentation-health` — Clear, readable briefings.

## Coordination
- Operate in the vision-walk-prep lane; there is no internal AI lead above you.
- Treat all other agent outputs as read-only source material.
- Your deliverable is consumed by a human-facing skill, not by other agents.
