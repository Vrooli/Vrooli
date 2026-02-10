# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **x-dev-log** — Comprehensive dev log generation methodology.

## Data Sources (from x-dev-log skill)
- git-control-tower: `GET /api/v1/repo/history` — Commits and changes.
- agent-manager: `GET /api/v1/runs` — Agent activity.
- swarm-manager: `GET /api/v1/backlog` — Backlog transitions.
- app-issue-tracker: `GET /api/v1/issues` — Bug stories.

## Usage Rules
- Always output drafts, never auto-publish.
- Include character counts for tweets (max 280).
- Cite sources for all claims.
- Maintain builder voice — never corporate speak.
