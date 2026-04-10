# Heartbeat: Meta Lead

## Priority Waterfall

Work through each priority level in order. Stop at the highest level with active issues.

### P1: Infrastructure Health

- Run `vrooli-autoheal status --json` to check core resource and scenario health.
- If any checks are critical or failing: assess impact, escalate to director-swarm, stop here.

### P2: Critical Toolchain Issues

- Run `development-toolchain-validator report --conflicts` for cross-skill contradictions.
- Run `scenario-auditor scan <key-scenarios> --summary` for critical violations.
- If conflicts or critical violations found: assign to skill-optimizer or escalate to ecosystem-manager, stop here.

### P3: Toolchain Degradation

- Run `development-toolchain-validator report --drift` for changed skills.
- Run `development-toolchain-validator report --maturity` for vague skills.
- Run `development-toolchain-validator report --tool-baselines` for tool regressions.
- Queue findings for specialist attention.

### P4: Skill/Agent/Team Health

- Run `prompt-manager graph show` for health delta since last check.
- Run `prompt-manager graph health` to find low or declining health.
- Check `prompt-manager graph orphaned-skills`, `skillless-agents`, `empty-teams`, `cliless-skills`, `circular-refs`.
- Cross-reference `popular` with `health` to find high-leverage targets.
- Assign work to skill-optimizer, agent-optimizer, or team-optimizer as appropriate.

### P5: Opportunities

- Review DTV coverage gaps and cross-team feedback.
- Feed opportunities into the Research-Analyze-Plan pipeline.
- Track progress on active optimization cycles.
