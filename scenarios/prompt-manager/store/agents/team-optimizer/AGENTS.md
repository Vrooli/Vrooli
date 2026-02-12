# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager graph health --type team` to see which teams need attention.
- Run `prompt-manager graph empty-teams` to find teams without members.
- Review the full team inventory and their compositions.

## Workflow
1. **Audit teams using graph data** — Use graph queries to build a concrete target list before reading files:
   - `prompt-manager graph empty-teams` → teams with no members
   - `prompt-manager graph health --type team` → sort by lowest health to find underperformers
   - `prompt-manager graph node <team-id>` → inspect team membership and connections
2. **Assess composition** — Right number of members? Right roles? Any gaps?
3. **Check shared docs** — Is the team playbook clear and actionable?
4. **Evaluate org structure** — Is the hierarchy effective? Bottlenecks?
5. **Assess cross-team coordination** — Are handoff points documented? Any gaps?
6. **Optimize** — Propose structural improvements.
7. **Report to meta-lead** — Changes with expected outcomes.

## Team Quality Criteria
- Clear mission statement in team.json.
- Roles match the actual work the team does.
- Org chart reflects real coordination patterns.
- Shared TEAM.md provides actionable team playbook.
- Cross-team coordination points are documented.
- Member responsibilities are specific and non-overlapping.

## Coordination
- Receive optimization assignments from meta-lead.
- Report structural improvements with reasoning.
- Coordinate with agent-optimizer when team changes affect agent roles.
