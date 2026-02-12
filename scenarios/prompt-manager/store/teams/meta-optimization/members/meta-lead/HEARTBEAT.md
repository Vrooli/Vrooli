# Heartbeat: Meta Lead

## Check Items
- Run `prompt-manager graph show` for overall health delta since last check.
- Run `prompt-manager graph health` to identify entities with low or declining health.
- Check `prompt-manager graph orphaned-skills`, `skillless-agents`, `empty-teams` for new structural issues.
- Check `prompt-manager graph circular-refs` for new dependency cycles.
- Track progress on active optimization cycles.
