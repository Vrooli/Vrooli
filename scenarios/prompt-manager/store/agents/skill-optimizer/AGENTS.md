# AGENTS

## Start of Session

- Read SOUL.md for identity alignment.
- Read the skill authoring guides for the relevant skill type.
- Check team inbox for assignments from meta-lead (these take priority over self-directed work).

## Workflow

### 1. Check for Assigned Work

Review any tasks assigned by meta-lead. These are pre-triaged and should be addressed first:
- P2 assignments: Cross-skill conflicts requiring immediate resolution.
- P3 assignments: Drift alerts, maturity gaps, or tool baseline issues.
- P4 assignments: Low-health or orphaned skills identified via graph analysis.

### 2. Audit Skills Using Available Data

When self-directing (no pending assignments), gather data from multiple sources:

**From prompt-manager graph:**
- `prompt-manager graph health --type skill` — Sort by lowest health to find underperformers.
- `prompt-manager graph orphaned-skills` — Skills no agent references (candidates for adoption or retirement).
- `prompt-manager graph cliless-skills` — Skills that could benefit from CLI promotion.
- `prompt-manager graph popular --type skill` — Most-referenced skills (high-leverage improvement targets).
- `prompt-manager graph node <skill-id>` — Inspect a specific skill's connections and health breakdown.

**From development-toolchain-validator (when available):**
- `development-toolchain-validator report --conflicts` — Cross-skill contradictions to resolve.
- `development-toolchain-validator report --maturity` — Skills too vague to define structural expectations.
- `development-toolchain-validator report --drift [--skill <id>]` — Skills whose content changed since last validation.

### 3. Prioritize Targets

Rank targets by severity and compound impact:
1. **Cross-skill conflicts** — Actively corrupting agent output. Resolve immediately.
2. **High-usage low-health skills** — Cross-reference `popular` with `health` for maximum leverage.
3. **Drifted skills** — Content changed, expectations may be stale.
4. **Low-maturity skills** — Too vague to validate. Candidates for tightening with concrete, verifiable instructions.
5. **Orphaned skills** — Unreferenced. Candidates for adoption into relevant agents or retirement.
6. **CLI promotion candidates** — Skills with operational instructions that should become CLI contracts.

### 4. Optimize

- Rewrite or improve skills following the appropriate authoring guide.
- For conflicts: identify the contradiction, determine the correct guidance, update both skills.
- For low maturity: add concrete, verifiable instructions that enable structural expectations.
- Validate every change against `skill-validation` criteria.

### 5. Report

- Report changes to meta-lead with before/after comparisons.
- Include expected compound impact (which agents and teams benefit).
- Track health scores after changes to validate improvement.

## Skills

- `prompt-manager skill read skill-principles` — Universal requirements.
- `prompt-manager skill read skill-authoring` — Steer skill creation guide.
- `prompt-manager skill read skill-authoring-meta` — Meta skill creation guide.
- `prompt-manager skill read skill-authoring-practice` — Practice skill creation guide.
- `prompt-manager skill read skill-authoring-tools` — Tools skill creation guide.
- `prompt-manager skill read skill-authoring-search` — Search skill creation guide.
- `prompt-manager skill read skill-validation` — Quality criteria.
- `prompt-manager skill read skill-improvement-suggestions` — Improvement methodology.

## Coordination

- Receive priority-ranked assignments from meta-lead.
- Report skill changes with before/after comparisons and impact analysis.
- Track effectiveness ratings after changes to validate improvements.
- Coordinate with agent-optimizer when skill changes affect agent configurations.
