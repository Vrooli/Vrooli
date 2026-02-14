# Operational Target Coverage

This reference maps PRD operational targets to current documentation and implementation touchpoints.

## P1 Targets

### OT-P1-001 - Execution control policy
Status: Implemented.
References:
- [DOC: docs/reference/configuration.md]
- [CODE: api/internal/execution/policy_store.go]
- [CODE: ui/src/services/execution-policy-service.ts]

### OT-P1-002 - Execution operations page
Status: Implemented.
References:
- [DOC: docs/concepts/ARCHITECTURE.md]
- [CODE: ui/src/pages/ExecutionPage.tsx]
- [CODE: api/internal/execution/handler.go]

### OT-P1-003 - Insights engine
Status: Draft.
References:
- [DOC: docs/reference/configuration.md]
- [DOC: docs/internal/ASSUMPTIONS.md]

### OT-P1-004 - Research agent modal
Status: Draft/partial (research actions exist in backlog details).
References:
- [DOC: docs/concepts/ARCHITECTURE.md]
- [CODE: ui/src/pages/BacklogDetailsPage.tsx]
- [CODE: api/internal/backlog/handler.go]

### OT-P1-005 - visited-tracker integration
Status: Draft.
References:
- [DOC: docs/internal/INTEROP_AUDIT.md]
- [DOC: docs/internal/ASSUMPTIONS.md]

### OT-P1-006 - knowledge-observatory integration
Status: Draft.
References:
- [DOC: docs/internal/INTEROP_AUDIT.md]
- [DOC: docs/internal/PROBLEMS.md]

### OT-P1-007 - scenario-completeness-scoring integration
Status: Draft.
References:
- [DOC: docs/internal/INTEROP_AUDIT.md]
- [DOC: docs/reference/configuration.md]

### OT-P1-008 - app-issue-tracker integration
Status: Draft.
References:
- [DOC: docs/internal/INTEROP_AUDIT.md]
- [DOC: docs/internal/ASSUMPTIONS.md]

### OT-P1-009 - test-genie integration
Status: Draft.
References:
- [DOC: docs/internal/INTEROP_AUDIT.md]
- [DOC: docs/internal/PROBLEMS.md]

### OT-P1-010 - Settings modal
Status: Implemented.
References:
- [DOC: docs/reference/configuration.md]
- [CODE: ui/src/pages/SettingsPage.tsx]
- [CODE: api/internal/settings/handler.go]
