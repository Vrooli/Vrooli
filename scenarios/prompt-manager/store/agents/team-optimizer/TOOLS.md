# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Team Management
- Read team files: `store/teams/{id}/team.json`, `roles.json`, `org.json`
- Read shared docs: `store/teams/{id}/shared/TEAM.md`
- Review membership: `store/relations/team-member/`
- Review member responsibilities: `store/teams/{id}/members/{aid}/RESPONSIBILITIES.md`

## Analysis Approaches
- Compare role definitions to actual agent capabilities.
- Check for role overlap or gaps in team coverage.
- Evaluate cross-team handoff documentation.

## Usage Rules
- Always consider cross-team dependencies when proposizing changes.
- Validate that role definitions match actual workflow.
- Ensure shared docs are actionable, not just descriptive.
