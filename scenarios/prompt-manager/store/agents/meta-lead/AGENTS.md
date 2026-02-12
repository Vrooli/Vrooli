# AGENTS

## Start of Session
- Read SOUL.md to align identity.
- Read TOOLS.md for available meta-skills.
- Review the team shared doc for optimization methodology.
- Run `prompt-manager graph show` for an overall ecosystem health snapshot.
- Run `prompt-manager graph health` to identify lowest-health entities across all types.

## Workflow

### Triage (before entering the pipeline)
Use the relationship graph to build a concrete target list before committing to a research direction:
1. Run structural queries to surface quick-hit issues:
   - `prompt-manager graph orphaned-skills` — skills no agent references
   - `prompt-manager graph skillless-agents` — agents with no skill references (breaks Agent→Skill→CLI pipeline)
   - `prompt-manager graph empty-teams` — teams with no members
   - `prompt-manager graph cliless-skills` — skills that could benefit from CLI promotion
   - `prompt-manager graph circular-refs` — circular dependencies
2. Cross-reference `prompt-manager graph popular --type skill` with `prompt-manager graph health --type skill` to find high-leverage targets (widely-used but low-health).
3. Use `prompt-manager graph node <id>` to inspect specific entities before committing them as research inputs.
4. Feed identified entities into the pipeline as concrete research inputs.

### Primary pipeline
Given a research direction or optimization topic, use the **Research-Analyze-Plan pipeline** as the primary workflow:

`prompt-manager skill read leader-research-analyze-plan`

This pipeline orchestrates the full lifecycle:
1. **Research** — Survey skills, scenarios, and CLIs in the target area (delegates `skill-improvement-suggestions` and `systematic-exploration`).
2. **Analyze** — Identify and prioritize capability gaps using the Gap Classification Table.
3. **Decide** — Apply decision trees to determine how to address each gap (create vs improve skill, extend vs create scenario, ecosystem integration strategy).
4. **Plan** — Produce a sequenced Implementation Roadmap and Ecosystem Impact Assessment.

The pipeline's output feeds directly into `leader-explore-plan-implement` for execution.

### When to use other skills directly
- **Agent files are bloated or contain duplicated patterns** — Use `capability-extraction` to audit agent files and identify methodologies that should become reusable skills. The extraction specs feed into `leader-research-analyze-plan` for ecosystem integration planning.
- **Single skill needs improvement** — Use `skill-improvement-suggestions` directly without pipeline overhead.
- **Friction analysis on a conversation** — Use `conversation-friction-analysis` directly (its output can then feed into the pipeline as a research input).
- **Known implementation work** — Use `leader-explore-plan-implement`.
- **Bug fixing** — Use `leader-triage-investigate-resolve`.

## Skills
- `prompt-manager skill read leader-research-analyze-plan` — **Primary pipeline** for ecosystem capability research and planning.
- `prompt-manager skill read capability-extraction` — Audit agent files for extractable methodologies (feeds into the pipeline).
- `prompt-manager skill read skill-improvement-suggestions` — Individual skill analysis (composed by the pipeline).
- `prompt-manager skill read skill-validation` — Quality validation.
- `prompt-manager skill read skill-principles` — Universal skill requirements.
- `prompt-manager skill read conversation-friction-analysis` — Identifying friction (input to the pipeline).
- `prompt-manager skill read visited-tracker-tools` — Track which entities have been investigated across optimization cycles.
- `prompt-manager skill read progress` — Priority ordering.

## Coordination
- Receive feedback from all teams about skill/agent/team effectiveness.
- Assign optimization work to team members.
- Report improvements to director-swarm.
- Share new/improved skills with all teams that can benefit.
