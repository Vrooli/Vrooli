# Scenario Feature Team

## Mission
Design feature proposals from requirements through execution-ready plans, authored as Swarm Manager backlog artifacts instead of direct scenario code edits.

## Feature Development Lifecycle
1. **Requirements** — Clarify the problem, user, and acceptance criteria.
2. **Design** — Architecture, API contracts, data models, UX flows.
3. **Backlog Authoring** — Create `idea` or `execute` backlog artifacts in Swarm Manager.
4. **Review** — Verify quality, tests, docs, and acceptance criteria.
5. **Done** — Artifact is execution-ready with complete evidence and risk notes.

## Definition of Done
A feature is done when:
- [ ] All acceptance criteria are met.
- [ ] Backlog item exists in Swarm Manager (`idea` or `execute`) with execution-ready detail.
- [ ] Evidence and risk are documented.
- [ ] Team provenance fields are populated.
- [ ] No direct target-scenario code edits were made.

## Swarm Manager Contract (Mandatory)
- Use the shared `swarm-manager-recommendations` skill for all handoffs.
- Required fields: `targetScenario`, `problemOrOpportunity`, `proposedAction`, `evidence`, `riskLevel`, `executionModeHint`, `createdByTeam`, `sourceRunId`.
- Feature team members do not directly modify target scenario code under this workflow.

## Scope Management
- feature-lead explicitly defines in/out scope for every feature.
- Nice-to-haves are deferred to a follow-up, not crammed in.
- Scope changes require feature-lead approval.

## Cross-Team Coordination
- **Director Swarm** provides feature priorities and requirements.
- **QA Team** validates feature quality post-implementation.
- **Debug Team** handles bugs found during feature work.
- **Marketing Crew** gets notified of completed features for content.
- **Revenue Research** may originate feature ideas based on market analysis.

## Key Skills
- `prompt-manager skill read implementation-plan-authoring`
- `prompt-manager skill read feature-scope`
- `prompt-manager skill read api-steer`
- `prompt-manager skill read storage-steer`
- `prompt-manager skill read react-stability`
- `prompt-manager skill read swarm-manager-recommendations`
