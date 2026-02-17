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
Status: Implemented (3-phase Idea Agent workflow: clarify/suggest/enhance).
References:
- [DOC: docs/concepts/ARCHITECTURE.md]
- [DOC: docs/guides/idea-agent-workflow.md]
- [CODE: ui/src/pages/BacklogDetailsPage.tsx]
- [CODE: ui/src/components/backlog/backlog-agent-dialog.tsx]
- [CODE: ui/src/components/backlog/idea-clarify-panel.tsx]
- [CODE: ui/src/components/backlog/idea-suggestions-panel.tsx]
- [CODE: api/internal/backlog/handler.go]
- [CODE: api/internal/prompttrace/trace.go]

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

## P0.5 Targets (Implemented but not in original PRD)

### OT-P05-001 - Prompts management
Status: Implemented.
References:
- [CODE: ui/src/pages/PromptsPage.tsx]
- [CODE: api/internal/prompts/handler.go]
- [CODE: cli/cmd_prompts.go]

### OT-P05-002 - Backlog CRUD form dialog
Status: Implemented.
References:
- [CODE: ui/src/components/backlog/backlog-form-dialog.tsx]

### OT-P05-003 - Scenario lifecycle control
Status: Implemented.
References:
- [CODE: ui/src/pages/ScenarioDetailsPage.tsx]
- [CODE: api/internal/scenarios/handler.go]
- [CODE: cli/cmd_scenarios.go]

### OT-P05-004 - Scenario spec-sync-archive
Status: Implemented.
References:
- [CODE: api/internal/scenarios/handler.go]
- [CODE: cli/cmd_scenarios.go]

### OT-P05-005 - Backlog convert
Status: Implemented.
References:
- [CODE: api/internal/backlog/handler.go]
- [CODE: cli/cmd_backlog.go]

### OT-P05-006 - Backlog prompt trace
Status: Implemented.
References:
- [CODE: api/internal/prompttrace/trace.go]
- [CODE: cli/cmd_backlog.go]
- [CODE: cli/cmd_execution.go]
