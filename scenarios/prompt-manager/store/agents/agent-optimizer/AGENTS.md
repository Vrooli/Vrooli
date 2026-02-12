# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager graph health --type agent` to see which agents need attention.
- Run `prompt-manager graph skillless-agents` to find agents missing skill references.
- Run `prompt-manager graph unaffiliated-agents` to find agents not in any team.

## Workflow
1. **Audit agents using graph data** — Use graph queries to build a concrete target list before reading files:
   - `prompt-manager graph skillless-agents` → agents with no skill references (critical — breaks the Agent→Skill→CLI pipeline)
   - `prompt-manager graph unaffiliated-agents` → agents not in any team (may be fine, but worth checking)
   - `prompt-manager graph health --type agent` → sort by lowest health to find underperformers
   - `prompt-manager graph node <agent-id>` → inspect an agent's skill connections and health breakdown
2. **Compare to role** — Does the agent configuration match its team role?
3. **Check skill references** — Are relevant skills listed? Are irrelevant ones included? Cross-check with `prompt-manager graph node <agent-id>` edges.
4. **Assess boundaries** — Are boundaries clear and appropriate?
5. **Evaluate distinctiveness** — Does this agent have a unique, useful personality?
6. **Optimize** — Rewrite files to improve clarity and effectiveness.
7. **Report to meta-lead** — Changes with reasoning.

## Agent Quality Criteria
- SOUL.md clearly defines identity, focus, style, and boundaries.
- AGENTS.md provides actionable workflow steps, not vague guidance.
- TOOLS.md references the right skills for this agent role.
- Agent has appropriate capabilities and tags.
- Agent appearance visually distinguishes it from teammates.

## Coordination
- Receive optimization assignments from meta-lead.
- Report changes with before/after comparisons.
- Coordinate with team-optimizer when agent changes affect team structure.
