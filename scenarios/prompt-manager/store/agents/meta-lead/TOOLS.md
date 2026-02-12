# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **leader-research-analyze-plan** — Primary pipeline for ecosystem capability research, gap analysis, and implementation planning.
- **capability-extraction** — Audit agent files for embedded methodologies that should become reusable skills.
- **skill-improvement-suggestions** — Individual skill analysis methodology (composed by the pipeline).
- **skill-validation** — Quality validation criteria.
- **skill-principles** — Universal skill requirements.
- **conversation-friction-analysis** — Identifying agent interaction friction (feeds into the pipeline as research input).

## Graph Analysis Commands
- `prompt-manager graph show` — Overall ecosystem health snapshot (node/edge counts, average health).
- `prompt-manager graph health [--type X | <id>]` — Health scores for all entities or a specific one.
- `prompt-manager graph orphaned-skills [--limit N]` — Skills not referenced by any agent.
- `prompt-manager graph skillless-agents [--limit N]` — Agents not referencing any skills.
- `prompt-manager graph empty-teams` — Teams with no members.
- `prompt-manager graph cliless-skills [--limit N]` — Skills with no CLI code references.
- `prompt-manager graph popular [--type X] [--limit N]` — Most-referenced nodes by incoming edge count.
- `prompt-manager graph circular-refs` — Circular reference detection.
- `prompt-manager graph node <id> [--json]` — Single node with all connections and health breakdown.
- `prompt-manager graph dump --json` — Full graph data for programmatic analysis.

## Skill Management Commands
- `prompt-manager skill list` — List all skills with ratings.
- `prompt-manager skill show <id>` — View skill details and metrics.
- `prompt-manager skill rate <id>` — Rate skill effectiveness.

## Usage Rules
- Prioritize by compound impact across the ecosystem.
- Measure effectiveness before and after changes.
- Every optimization should be independently evaluable.
