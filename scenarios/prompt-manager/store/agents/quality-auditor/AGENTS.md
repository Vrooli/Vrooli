# AGENTS

## Start of Session
- Read SOUL.md to align identity.
- Run `prompt-manager team member-context scenario-qa quality-auditor` for team context.

## Workflow
1. **Select scenario** — Run `swarm-manager scenarios review-queue --limit 1`. If zero returned, log "No scenarios for deep audit" and stop.
2. **Check recency** — Query knowledge log for recent `[deep-audit]` entries. Skip if this scenario was audited in the last 7 days.
3. **Select steer skill** — Choose the next skill from the rotation that has NOT been recently applied to this scenario.
4. **Read the skill** — Run `prompt-manager skill read <skill-id>` and internalize the methodology.
5. **Investigate** — Read the scenario's architecture docs, code structure, and test suite. Apply the skill's methodology to identify structural quality issues.
6. **Create backlog item** — If non-trivial findings exist:
   - `swarm-manager backlog create --kind execute --name qa-deep-<scenario>-<skill-short>-YYYYMMDD`
   - Include findings as evidence in the description
   - Set `suggested_skills` to `["<skill-id>"]`
   - Set tags: `["deep-audit", "<scenario-name>"]`
   - Set acceptance_allow: `["scenarios/<scenario>/**"]`
   - Set priority: 6
   - Write a draft plan.md in the backlog item folder
7. **Log** — Record `[YYYY-MM-DD] Deep audit: <scenario> via <skill-id> — <one-line-summary>` to knowledge log.

## Steer Skill Rotation

### Always applicable (Tier 1)
1. `screaming-architecture-audit` — Does the physical structure express the scenario's purpose?
2. `boundary-of-responsibility-enforcement` — Is each module doing only its own job?
3. `seam-discovery-and-enforcement` — Are side effects isolated? Can behavior vary without invasive changes?
4. `invariant-discovery-and-enforcement` — Are system constraints explicit and enforced?
5. `cognitive-load-reduction` — Can a new developer understand this code quickly?
6. `decision-boundary-extraction` — Are important decisions explicit and easy to locate?
7. `code-cleanup` — Is there dead code, unused imports, or backwards-compatibility cruft?

### Conditional (Tier 2 — check applicability first)
- `react-coherence` — Only for scenarios with a React UI
- `security` — OWASP baseline audit
- `e2e-testing` — End-to-end test strategy assessment
- `documentation-health` — Documentation quality and bidirectional traceability

## Coordination
- Works alongside the programmatic QA runner: the runner handles GCT-driven reviews, quality-auditor handles structural analysis (judgment-based).
- All findings feed into the same swarm-manager backlog pipeline.
