# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **swarm-manager-backlog-tools** — Initiative and backlog inspection commands.
- **documentation-health** — Keep briefings concrete and readable.

## Primary Surfaces

### Phase 2 — Retrospective
- `swarm-manager overview`
- `swarm-manager stats summary`
- `swarm-manager initiatives list`

### Phase 3 — Portfolio Decisions
- `prompt-manager team decision-list director-swarm --status=pending --context=initiative-portfolio --json`
- `prompt-manager team decision-list director-swarm --status=pending --context=initiative-supplement --json`
- `prompt-manager team decision-list director-swarm --status=pending --context=initiative-proposal --json`
- `prompt-manager team decision-list director-swarm --status=pending --context=initiative-readiness --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=capability-gap --json` — raised by run-introspector / toolchain-validator but consumed by director-swarm, so group with portfolio items

### Phase 4 — Strategist Decisions
- `prompt-manager team decision-list director-swarm --status=pending --context=outcome-gap --json`
- `prompt-manager team decision-list director-swarm --status=pending --context=outcome-direction --json`

### Phase 4.6 — Marketing-Crew Decisions
- `prompt-manager team decision-list marketing-crew --status=pending --context=content-publish-proposal --json`
- `prompt-manager team decision-list marketing-crew --status=pending --context=campaign-launch-proposal --json`
- `prompt-manager team decision-list marketing-crew --status=pending --context=brand-guideline-update --json`
- `prompt-manager team decision-list marketing-crew --status=pending --context=audience-update --json`
- `prompt-manager team decision-list marketing-crew --status=pending --context=channel-update --json`
- `prompt-manager team decision-list marketing-crew --status=pending --context=coverage-gap --json`
- `prompt-manager team decision-list marketing-crew --status=pending --context=notebook-promotion --json`
- `prompt-manager team decision-list marketing-crew --status=pending --context=notebook-retirement --json`
- `prompt-manager team decision-list marketing-crew --status=pending --context=decision-rejection-proposed --json`
- `prompt-manager team decision-list marketing-crew --status=pending --context=framework-update --json`
- `prompt-manager team knowledge-list marketing-crew --topic=challenge-note` — marketing-contrarian skepticism attached to pending marketing-crew decisions; match by decision id and surface inline
- Note: `capability-gap` decisions raised by marketing-crew members are fetched under Phase 3 alongside meta-optimization's, since director-swarm is the shared consumer.

### Phase 5.5 — Meta-Optimization Self-Improvement Decisions
- `prompt-manager team decision-list meta-optimization --status=pending --context=meta-self-improvement --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=run-lesson --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=skill-conversion-candidate --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=skill-improvement --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=skill-deprecation --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=agent-improvement --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=agent-deprecation --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=team-structure-change --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=team-deprecation --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=toolchain-violation --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=decision-rejection-proposed --json`
- `prompt-manager team decision-list meta-optimization --status=pending --context=framework-update --json`
- `prompt-manager team knowledge-list meta-optimization --topic=challenge-note` — contrarian skepticism attached to pending meta-optimization decisions; match by decision id and surface inline

### Continuity
- `prompt-manager team knowledge-list director-swarm --topic=vision-walk`

## Usage Rules
- Read-only. Do not create decisions, modify backlog items, or trigger any side effects.
- Do not attempt to answer the questions you surface.
- Cap summaries at 3 decisions *per phase* (Phases 3, 4, 4.6, 5, 5.5 each capped independently).
- For Phase 5.5, group decisions by category before selecting top 3 — aim for category diversity across debt / run-lessons / skills / agents-and-teams / toolchain / framework-meta, not 3 from one bucket.
- Attach matching `challenge-note/<decision-id>` knowledge entries inline to their target decisions. Meta-optimization's contrarian scopes meta-optimization decisions; marketing-crew's `marketing-contrarian` scopes marketing-crew decisions. Director-swarm decisions do not receive challenge notes.
- Always note when a data source is unavailable (strategist disabled, monetization team not active, marketing-crew team disabled, meta-optimization team disabled, tech tree not available).
