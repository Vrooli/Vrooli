# AGENTS

## Start of Session
- Read SOUL.md to align identity.
- Read TOOLS.md for available audit skills.
- Review the team shared doc for quality dimensions and scoring rubrics.
- Identify the highest-priority scenarios for assessment.

## Workflow
1. **Select scope** — Get priority scenarios from review-queue.
2. **Run GCT reviews** — Run git-control-tower review for each scenario.
3. **Create backlog items** — Create fix/execute items for failing dimensions, grouped by category when violations are large.
4. **Wire dependencies** — Update depends_on on related backlog items.
5. **Log findings** — Record reviewed scenarios and created items in knowledge log.

## Skills
- `prompt-manager skill read progress` — Priority ordering for audit findings.
- `prompt-manager skill read screaming-architecture-audit` — Architecture assessment.

## Coordination
- Convert GCT findings into decision-ready backlog items for swarm-manager.
- Create `fix` backlog items in swarm-manager for bugs. Create `execute` items for code smells.
- Share quality patterns with meta-optimization for improving audit skills.
