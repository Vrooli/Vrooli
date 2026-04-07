# Meta-Orchestrator Summary: Team Organizational Maturity

## Source
Planning session workshopping how to add organizational properties to prompt-manager teams and build a project-level wiki. This initiative covers the team schema upgrade; the companion initiative (project-wiki) covers the wiki scenario itself.

## Decisions Made
- Teams are framed as mission-oriented squads in a virtual company. The "department" mental model provides the right properties (mission, objectives, KPIs, contributions) while acknowledging that the lifecycle is more dynamic than traditional org charts.
- New team.json v2 fields: objectives, kpis, consumers, contributions. These are structured JSON fields, not freeform markdown.
- TEAM.md gets simplified: remove anything that's now a structured field. Keep only narrative/freeform content (charter nuances, operating procedures, decision processes, skill references).
- Dependencies should be computed programmatically (derived from HEARTBEAT.md skill references, CLI usage, acceptance_allow paths), NOT stored as a manual field in team.json.
- Consumers are manually maintained because consumer relationships aren't derivable from code.
- Decision mode (authority) stays as-is for now — the existing decisionMode field plus TEAM.md charter text is sufficient.
- Status and cadence fields deferred — enabled boolean and heartbeat.json cron are sufficient for now.
- The heartbeat prompt builder should render v2 fields as natural language, not raw JSON. Agents get a readable briefing synthesized from structured data.
- A teams overview endpoint serves both meta-optimization (organizational health) and the wiki maintenance agent (reading contribution declarations).

## Current team.json Schema (v1)
Fields: kind, schemaVersion, id, displayName, mission, enabled, spawnMode, decisionMode, shared, revision, createdAt, updatedAt. The prompt builder in scenarios/prompt-manager/api/heartbeat/prompt_builder.go reads team.json for SpawnMode and DecisionMode only — rich context comes from TEAM.md freeform markdown.

## Proposed v2 Additions
- objectives (array of strings): Current period goals. E.g., "Achieve 80% portfolio completion"
- kpis (array of {name, target}): Measurable indicators. E.g., {name: "agent_success_rate", target: ">90%"}
- consumers (array of team IDs): Who depends on this team's outputs. Manually maintained.
- contributions (array of {type, section, description}): Wiki-worthy outputs. type = semantic label (e.g., "campaign-launched"), section = wiki path (e.g., "marketing/campaigns"), description = what it means. This is the integration point with project-wiki — the wiki maintenance agent reads these to know what events to scan for.

## Teams To Audit (all 8)
- director-swarm (enabled, single-process, approval mode) — Strategic direction, portfolio prioritization
- marketing-crew (disabled, single-process) — Content creation, brand communication
- revenue-research (disabled, multi-process) — Revenue opportunity identification
- meta-optimization (disabled, multi-process) — Health of skills/agents/teams
- scenario-feature (disabled, multi-process) — Design & implement new features
- scenario-qa (disabled, multi-process) — Quality auditing
- scenario-debug (disabled, multi-process) — Hypothesis-driven debugging
- scenario-refactor (disabled, multi-process) — Code quality improvement

## Dependency Notes
- This research is the root — everything in both initiatives depends on its output
- The project-wiki initiative's maintenance team needs the contributions field to exist
- heartbeat-prompt-team-context and teams-overview-endpoint can proceed in parallel after migration

## Unresolved Questions For This Research
- Exact values for each team's objectives, kpis, consumers, and contributions (draft these based on TEAM.md, HEARTBEAT.md, RESPONSIBILITIES.md)
- Whether dependencies computation should be a CLI command, API endpoint, or both
- Backward compatibility strategy for schema v1 → v2 migration
- Whether any teams should be merged (e.g., Debug + QA + Refactor into "Engineering Quality") — flagged as a meta-optimization question for later
