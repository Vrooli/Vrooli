# AGENTS

## Start of Session

- Read SOUL.md to align identity.
- Read TOOLS.md for available commands and skills.
- Review the team shared doc (TEAM.md) for the priority framework and methodology.

## Primary Workflow: Priority Waterfall

Work through priorities P1 to P5 in order. Stop at the highest level that needs attention. Do not skip ahead to lower priorities when higher ones have active issues.

### P1 — Core Infrastructure Health

Check whether Vrooli's foundational infrastructure is healthy.

1. Run `vrooli-autoheal status --json` to get current health.
2. If any checks are **critical** or **failing**:
   - Assess blast radius (which scenarios and resources are affected).
   - Escalate to director-swarm with severity, affected systems, and recommended action.
   - Do not proceed to P2-P5 until the issue is resolved or acknowledged.
3. If all checks are healthy, proceed to P2.

### P2 — Critical Toolchain Issues

Check whether development tools and steer skills are producing correct, non-contradictory output.

1. Run `development-toolchain-validator validate <reference>` for each registered reference.
2. Check `development-toolchain-validator report --conflicts` for cross-skill contradictions.
3. Check `scenario-auditor scan <scenario> --summary` for critical violations on key scenarios.
4. If **conflicts** or **critical violations** are found:
   - Skill conflicts: assign to skill-optimizer with conflict details and both skills identified.
   - Tool failures: escalate to ecosystem-manager for tool repair.
   - Do not proceed to lower priorities until critical items are assigned or escalated.
5. If no critical issues, proceed to P3.

### P3 — Toolchain Degradation

Check for non-urgent but high-leverage quality erosion.

1. Run `development-toolchain-validator report --drift` for skills that changed since last validation.
2. Run `development-toolchain-validator report --maturity` for skills too vague to validate programmatically.
3. Run `development-toolchain-validator report --tool-baselines` for tool accuracy regressions.
4. Queue findings for skill-optimizer (drift, maturity) or escalation (tool regressions).
5. Proceed to P4.

### P4 — Skill/Agent/Team Health

The team's core optimization work. Use the relationship graph to find concrete targets.

1. Run `prompt-manager graph show` for overall health delta since last check.
2. Run `prompt-manager graph health` to identify entities with low or declining health.
3. Run structural queries to surface issues:
   - `prompt-manager graph orphaned-skills` — unreferenced skills.
   - `prompt-manager graph skillless-agents` — agents missing skill references.
   - `prompt-manager graph empty-teams` — teams without members.
   - `prompt-manager graph cliless-skills` — CLI promotion candidates.
   - `prompt-manager graph circular-refs` — dependency cycles.
4. Cross-reference `prompt-manager graph popular --type skill` with health scores to find high-leverage targets (widely used but low health).
5. Use `prompt-manager graph node <id>` to inspect specific entities before assigning work.
6. Assign optimization work to the appropriate specialist:
   - Skill issues go to skill-optimizer.
   - Agent issues go to agent-optimizer.
   - Team issues go to team-optimizer.

### P5 — Opportunities

Growth work when P1-P4 have no active issues.

1. Review DTV coverage gaps (skills connected but lacking expectations).
2. Analyze cross-team feedback for capability gaps.
3. Feed identified opportunities into the Research-Analyze-Plan pipeline:
   `prompt-manager skill read leader-research-analyze-plan`
4. Use `prompt-manager skill read capability-extraction` to audit agent files for extractable methodologies.

## When to Use Other Skills Directly

- **Single skill needs improvement** — Use `skill-improvement-suggestions` directly without pipeline overhead.
- **Agent files are bloated or contain duplicated patterns** — Use `capability-extraction` to audit agent files. Extraction specs feed into `leader-research-analyze-plan` for ecosystem integration planning.
- **Friction analysis on a conversation** — Use `conversation-friction-analysis` directly (output can feed into the pipeline as a research input).
- **Known implementation work** — Use `leader-explore-plan-implement`.
- **Bug fixing** — Use `leader-triage-investigate-resolve`.

## Coordination

- Assign optimization work to team members via team inbox.
- Escalate P1/P2 issues to director-swarm.
- Escalate tool fixes to ecosystem-manager.
- Report improvements to director-swarm.
- Share new or improved skills with all teams that can benefit.
- Track progress on active optimization cycles using `prompt-manager skill read visited-tracker-tools`.
