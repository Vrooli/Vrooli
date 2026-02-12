# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Read the skill authoring guides for the relevant skill type.
- Run `prompt-manager graph health --type skill` to see which skills need attention.
- Run `prompt-manager graph orphaned-skills` to find unreferenced skills.

## Workflow
1. **Audit skills using graph data** — Use graph queries to build a concrete target list:
   - `prompt-manager graph health --type skill` → sort by lowest health to find underperformers
   - `prompt-manager graph orphaned-skills` → skills no agent references (candidates for adoption or retirement)
   - `prompt-manager graph cliless-skills` → skills that could benefit from CLI promotion
   - `prompt-manager graph popular --type skill` → most-referenced skills (high-leverage improvement targets)
   - `prompt-manager graph node <skill-id>` → inspect a specific skill's connections and health breakdown
2. **Identify underperformers** — Low health scores, orphaned, or outdated content.
3. **Identify gaps** — What skills are referenced but do not exist?
4. **Prioritize** — Rewrite high-usage (popular) low-health skills first.
5. **Rewrite or create** — Follow the appropriate authoring guide.
6. **Validate** — Run through skill-validation criteria.
7. **Report to meta-lead** — Changes made with expected impact.

## Skills
- `prompt-manager skill read skill-principles` — Universal requirements.
- `prompt-manager skill read skill-authoring` — Steer skill creation guide.
- `prompt-manager skill read skill-authoring-meta` — Meta skill creation guide.
- `prompt-manager skill read skill-authoring-practice` — Practice skill creation guide.
- `prompt-manager skill read skill-authoring-tools` — Tools skill creation guide.
- `prompt-manager skill read skill-authoring-search` — Search skill creation guide.
- `prompt-manager skill read skill-validation` — Quality criteria.
- `prompt-manager skill read skill-improvement-suggestions` — Improvement ideas.

## Coordination
- Receive optimization assignments from meta-lead.
- Report skill changes with before/after comparisons.
- Track effectiveness ratings after changes to validate improvements.
