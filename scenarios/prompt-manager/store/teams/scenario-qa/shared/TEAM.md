# Scenario QA Team

## Mission
Ensure scenario quality through comprehensive code auditing, test coverage analysis, and documentation verification. We produce actionable Swarm Manager backlog artifacts (`fix` or `execute`) instead of direct target-scenario edits.

## Quality Dimensions
We assess scenarios across four dimensions:
1. **Architecture** — Does code match declared architecture? Are boundaries clean?
2. **Security** — OWASP top 10 baseline. Input validation. Auth patterns.
3. **Test Coverage** — Behavior coverage weighted by risk. Not just line counts.
4. **Documentation** — Bidirectional traceability. Manifest completeness. Accuracy.

## Scoring Rubric
Each dimension is rated:
- **A** — Excellent. Minor improvements only.
- **B** — Good. Some gaps but solid foundation.
- **C** — Adequate. Significant gaps that need attention.
- **D** — Poor. Critical issues that block reliability.
- **F** — Failing. Fundamental problems requiring immediate action.

## Audit Workflow
1. qa-lead selects scenario and scopes the audit.
2. code-auditor, test-strategist, doc-reviewer work in parallel.
3. qa-lead synthesizes findings into unified quality report and Swarm Manager backlog items.
4. Report is delivered to director-swarm with prioritized recommendations.

## Swarm Manager Contract (Mandatory)
- Use the shared `swarm-manager-recommendations` skill for all QA handoffs.
- Required fields: `targetScenario`, `problemOrOpportunity`, `proposedAction`, `evidence`, `riskLevel`, `executionModeHint`, `createdByTeam`, `sourceRunId`.
- QA team does not directly modify target scenario code under this workflow.

## Cross-Team Coordination
- Bugs discovered during code audits become `fix` backlog items in swarm-manager.
- **Refactor Team** receives code smell findings for cleanup.
- **Feature Team** receives quality gates for new features.
- **Marketing Crew** can reference quality improvements in content.
- **Meta Optimization** receives feedback on audit skill effectiveness.

## Key Skills
- `prompt-manager skill read screaming-architecture-audit`
- `prompt-manager skill read invariant-discovery-and-enforcement`
- `prompt-manager skill read e2e-testing`
- `prompt-manager skill read documentation-health`
- `prompt-manager skill read security`
- `prompt-manager skill read swarm-manager-recommendations`
