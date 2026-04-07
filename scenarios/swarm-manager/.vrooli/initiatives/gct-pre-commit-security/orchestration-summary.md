# Meta-Orchestrator Summary: GCT Pre-Commit Security

## Source
Planning conversation covering Git Control Tower enhancement roadmap. This initiative focuses on integrating Secrets Manager's security scanning into GCT's commit workflow.

## Decisions Made
- PII detection uses two layers: generic regex patterns (emails, phones, SSNs, IPs, AWS keys) + user-uploaded custom watchlist (personal values to flag)
- Test file exemptions via allowlist system using configurable path patterns (e.g., `*_test.go`, `testdata/`, `fixtures/`)
- Secrets Manager needs a new file-list-based scan endpoint (accepts arbitrary file paths, not just directories) so GCT can scan staged files regardless of repo location
- Integration follows same pattern as tidiness-manager, test-genie, etc. (ReviewSecurityClient interface, capabilities map, API core discovery)
- Scan scope must be agnostic to scenario boundaries — resources, shared locations, project-level files all get scanned
- Commit gating is three-tier per check type: Required (blocks with emergency override) / Advisory (warns but allows) / Disabled
- Commit-level agent reuses review panel agent components (AgentTab, agentContext.ts patterns)

## Dependency Notes
- Secrets Manager may need modernization before integration (check for current patterns/prototypes)
- The security review tab depends on the file-list scan endpoint existing in Secrets Manager first
- Commit gating builds on the review tab (same data, different presentation — inline in commit panel)
- Commit-level agent extends the existing agent infrastructure, not a new system

## Unresolved Questions Deferred To Workshop
- Exact UX for the custom watchlist management (upload flow, editing, categories)
- Whether Secrets Manager needs significant modernization or just the new endpoint
- How emergency override for "Required" gating should work (admin password? confirmation dialog? audit log entry?)
- Whether code quality checks (from tidiness-manager) should also be gatable at commit level, or just security/PII
