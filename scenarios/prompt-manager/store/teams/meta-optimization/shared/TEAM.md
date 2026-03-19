# Meta Optimization Team

## Mission

Ensure the health of Vrooli's entire development capability chain — from core infrastructure to development tools to the skills, agents, and teams that drive all work. We detect and escalate critical issues, then directly optimize the meta-layer that compounds across everything else.

## Priority Framework

All work follows a strict priority waterfall. The team addresses the highest-priority issue before moving to lower priorities.

| Priority | Domain | Data Source | Action |
|----------|--------|-------------|--------|
| P1 | Core infrastructure failures | `vrooli-autoheal status` | Escalate immediately |
| P2 | Critical toolchain issues | DTV conflicts, scenario-auditor critical violations | Escalate or fix directly |
| P3 | Toolchain degradation | DTV drift, maturity, tool baselines | Queue for specialists |
| P4 | Skill/agent/team health | `prompt-manager graph` queries | Direct optimization |
| P5 | Growth opportunities | DTV coverage gaps, capability analysis | Research-Analyze-Plan pipeline |

### P1: Core Infrastructure

Vrooli-autoheal monitors resources and scenarios. If critical health checks fail, nothing else works reliably. Meta-lead checks autoheal status first on every heartbeat.

- **Indicators**: vrooli-autoheal reports critical or failing checks.
- **Response**: Escalate to director-swarm and/or ecosystem-manager with severity and impact assessment.
- **Do not**: Attempt to fix infrastructure directly. Autoheal handles recovery; escalation ensures visibility and coordination.

### P2: Critical Toolchain Issues

Development tools (scenario-auditor, test-genie, completeness-scoring) and steer skills must produce correct, non-contradictory guidance. Conflicts between skills or broken tool output corrupt every scenario being built.

- **Indicators**: DTV cross-skill conflicts, scenario-auditor critical violations, test-genie failures on reference scenarios.
- **Response**: Skill conflicts go to skill-optimizer for resolution. Tool failures are escalated to ecosystem-manager for repair.
- **Why critical**: A broken steer skill or tool silently degrades every agent session that uses it.

### P3: Toolchain Degradation

Non-urgent but high-leverage. Skill drift (content changed since last validation), low maturity scores (skills too vague to validate programmatically), and tool baseline regressions erode quality over time.

- **Indicators**: DTV drift alerts, low maturity scores, tool baseline deviations.
- **Response**: Queue findings for skill-optimizer to address alongside P4 work.
- **Why separate from P2**: These don't break things immediately but compound into silent quality loss.

### P4: Skill/Agent/Team Health

The team's core competency. Use prompt-manager's relationship graph to find underperformers, orphans, and structural issues, then optimize directly.

- **Indicators**: Low health scores, orphaned skills, skillless agents, empty teams, cliless skills, circular references.
- **Response**: Direct optimization by the appropriate specialist (skill-optimizer, agent-optimizer, or team-optimizer).

### P5: Opportunities

Growth work. Identify capability gaps, create new skills, expand coverage, improve ecosystem integration. Only pursued when P1-P4 have no active issues.

- **Indicators**: DTV coverage gaps, cross-team feedback, capability analysis.
- **Response**: Feed into the Research-Analyze-Plan pipeline for structured investigation and implementation.

## Compound Impact Principle

A skill used by 5 teams that improves by 20% has 5x the impact of a skill used by 1 team that improves by 100%. Within each priority level, optimize for ecosystem-wide impact, not local perfection.

## Optimization Methodology

1. **Measure** — Check data sources in priority order (P1 to P5).
2. **Identify** — Use structural queries and reports to find concrete targets.
3. **Prioritize** — Rank by priority level first, then by compound impact within each level.
4. **Optimize** — Make targeted improvements following quality criteria.
5. **Validate** — Re-check after changes to confirm measurable improvement.
6. **Iterate** — Use health deltas to inform the next cycle.

## Quality Hierarchy

- **Skills** — Foundation. Skills make agents effective.
- **Agents** — Middle layer. Agents use skills within teams.
- **Teams** — Top layer. Teams coordinate agents toward goals.

Improving a lower layer compounds upward through all higher layers.

## Data Sources

### Infrastructure Health (P1)

- `vrooli-autoheal status [--json]` — Current health of all monitored resources and scenarios.
- `vrooli-autoheal checks` — List all registered health checks.

### Development Toolchain (P2/P3)

*Commands available when development-toolchain-validator ships.*

- `development-toolchain-validator validate <reference>` — Run full validation against a reference scenario.
- `development-toolchain-validator report --conflicts` — Cross-skill contradictions.
- `development-toolchain-validator report --drift` — Skills changed since last validation.
- `development-toolchain-validator report --maturity` — Skill configurability and maturity scores.
- `development-toolchain-validator report --tool-baselines` — Tool accuracy regression checks.
- `scenario-auditor scan <scenario> [--summary]` — Standards violations for a scenario.

### Prompt-Manager Graph (P4)

- `prompt-manager graph show` — Ecosystem health snapshot.
- `prompt-manager graph health [--type X]` — Entity health scores.
- `prompt-manager graph orphaned-skills` — Skills not referenced by any agent.
- `prompt-manager graph skillless-agents` — Agents not referencing any skills.
- `prompt-manager graph empty-teams` — Teams without members.
- `prompt-manager graph cliless-skills` — Skills without CLI promotion.
- `prompt-manager graph popular [--type X]` — Most-referenced entities by edge count.
- `prompt-manager graph circular-refs` — Dependency cycles.
- `prompt-manager graph node <id>` — Entity detail with connections and health breakdown.

## Cross-Team Coordination

- **Director Swarm** receives P1/P2 escalations and periodic improvement reports.
- **Ecosystem Manager** receives work items for tool fixes and scenario improvements.
- **All teams** provide feedback on skill/agent/team effectiveness.

## Key Skills

- `leader-research-analyze-plan` — Primary pipeline for P5 research and planning.
- `skill-improvement-suggestions` — Individual skill analysis methodology.
- `skill-validation` — Skill quality validation criteria.
- `skill-principles` — Universal skill quality standards.
- `conversation-friction-analysis` — Agent interaction analysis.
- `capability-extraction` — Extract reusable methodologies from agent files.
- `team-tool-mapping` — Equip teams with scenario-based tool skills using lazy evaluation for demand-driven development.
- `visited-tracker-tools` — Track investigated entities across optimization cycles.
