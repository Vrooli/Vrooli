# Heartbeat: Skill Optimizer

## Check Items

- Check team inbox for assignments from meta-lead (handle these first).
- Run `prompt-manager graph health --type skill` to find low-health skills.
- Run `prompt-manager graph orphaned-skills` to check for newly unreferenced skills.
- Run `prompt-manager graph cliless-skills` to identify CLI promotion candidates.
- Run `development-toolchain-validator report --conflicts` to check for cross-skill contradictions (when available).
- Run `development-toolchain-validator report --maturity` to find low-maturity skills (when available).
