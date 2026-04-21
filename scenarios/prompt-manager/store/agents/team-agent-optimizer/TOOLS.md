# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **skill-authoring-tools** — reference for agent TOOLS.md proposals
- **capability-extraction** — distilling methodologies from agent files
- **team-tool-mapping** — when team structure changes touch scenario tool wiring
- **visited-tracker-tools** — rotation pattern
- **documentation-health** — durable snapshots

## Primary Surfaces
- `prompt-manager graph health --type agent` / `--type team`
- `prompt-manager graph popular --type agent` / `--type team`
- `prompt-manager graph skillless-agents`
- `prompt-manager graph empty-teams`
- `prompt-manager graph node <id>`
- `prompt-manager agent show <id>` / `prompt-manager team show <id>`
- `prompt-manager agent update <id>` / `prompt-manager team update <id>` (via decisions)
- `shared/TEAM_AUDIT.md`, `AGENT_AUDIT.md`, `DEPRECATION_QUEUE.md`
- `shared/RUN_LESSONS.md` (usage signals)
- `prompt-manager team decision-list meta-optimization --status=pending --context=agent-* | team-*`
- `prompt-manager team knowledge-list meta-optimization --topic-prefix=agent-visited/ | team-visited/`

## Usage Rules
- Every proposal names the specific target + concrete evidence + expected delta + measurement plan.
- Do not touch skills or scenario code. Cross-lane proposals are rejected by the contrarian.
- Cap decisions at 2 per heartbeat (prefer 1).
- Team-structure proposals: default to the smallest change (a role edit, a coordination flag) rather than a full rewrite.
