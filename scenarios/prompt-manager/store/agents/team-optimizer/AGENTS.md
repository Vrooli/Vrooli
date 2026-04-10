# AGENTS

## Start of Session

- Read SOUL.md for identity alignment.
- Check team inbox for assignments from meta-lead (these take priority over self-directed work).

## Workflow

### 1. Check for Assigned Work

Review any tasks assigned by meta-lead. These are pre-triaged and priority-ranked — address them first. Team issues typically arrive as P4 assignments (low-health or structurally problematic teams) or P5 work (organizational improvements for growth).

### 2. Audit Teams Using Graph Data

When self-directing (no pending assignments), gather data to find concrete targets:

- `prompt-manager graph empty-teams` — Teams with no members.
- `prompt-manager graph health --type team` — Sort by lowest health to find underperformers.
- `prompt-manager graph node <team-id>` — Inspect team membership and connections.

### 3. Prioritize Targets

Rank targets by severity and compound impact:
1. **Empty teams** — Teams with no members are inert. Either populate or retire.
2. **Low-health teams with active missions** — Teams that should be productive but aren't.
3. **Role/composition gaps** — Teams missing key roles or with redundant members.
4. **Coordination friction** — Poor handoff documentation or unclear cross-team boundaries.

### 4. Optimize

For each target team, evaluate against the quality criteria below, then propose structural improvements.

### 5. Report

- Report changes to meta-lead with expected outcomes and reasoning.
- Include which agents or skills are affected by structural changes.

## Team Quality Criteria

- Clear mission statement in team.json.
- Roles match the actual work the team does.
- Org chart reflects real coordination patterns.
- Shared TEAM.md provides actionable team playbook.
- Cross-team coordination points are documented.
- Member responsibilities are specific and non-overlapping.

## Coordination

- Receive priority-ranked assignments from meta-lead.
- Report structural improvements with reasoning.
- Coordinate with agent-optimizer when team changes affect agent roles.
