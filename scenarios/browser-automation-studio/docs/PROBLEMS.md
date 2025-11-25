# Known Issues & Follow-Up Tasks

This file tracks unresolved issues, technical debt, and planned improvements for the browser-automation-studio scenario.

## API Refactoring (In Progress)

### Completed
- ✅ Extracted export configuration and movie spec building logic from monolithic `handlers/execution_export_helpers.go` into focused `handlers/export` package
- ✅ Created 5 well-documented modules: presets, types, builder, overrides, spec_builder
- ✅ Reduced `execution_export_helpers.go` from 969 lines to 41 lines (95% reduction)
- ✅ Completed type conversion extraction from `services/timeline.go` to `internal/typeconv` package
- ✅ Removed 381 duplicate lines from `services/timeline.go` (52% reduction: 723→342 lines)
- ✅ Extracted node-specific lint functions from `workflow/validator/lint.go` into `node_linters.go`
- ✅ Reduced `lint.go` from 787 to 478 lines (39% reduction)
- ✅ Extracted entity-specific repository methods from `database/repository.go` into focused files
- ✅ Split 879-line `repository.go` into 5 focused files by entity type (89% reduction: 879→92 lines)
- ✅ Created organized repository structure: projects, workflows, executions, artifacts, folders
- ✅ All existing tests pass with new structure
- ✅ Extracted 7 workflow helper functions from `handlers/workflows.go` into `workflow_helpers.go`
- ✅ Reduced `workflows.go` from 732 to 644 lines (12% reduction)
- ✅ Split monolithic `services/workflow_service_execution.go` into 4 focused files by responsibility
- ✅ Created focused modules: adhoc (146 lines), lifecycle (326 lines), export (132 lines), automation (143 lines)
- ✅ Reduced `workflow_service_execution.go` from 782 to 117 lines (85% reduction)
- ✅ Clear separation of concerns: adhoc workflow management, execution orchestration, export preview, automation engine integration
- ✅ Refactored `services/recording_service.go` from 734 to 241 lines (67% reduction)
- ✅ Split recording service into 3 new focused files:
  - `recording_types.go` (305 lines) - All manifest types and their methods
  - `recording_resolution.go` (115 lines) - Project and workflow resolution logic
  - `recording_persistence.go` (150 lines) - Frame persistence and cleanup operations
- ✅ Recording service now has clear file organization across 9 files (service, types, resolution, persistence, adapter, helpers, file_store, interface, test)

### Remaining Opportunities
- Consider extracting logic from remaining large service files:
  - `services/workflow_files.go` (728 lines) - Could extract file sync and persistence logic
  - `services/replay_renderer.go` (702 lines) - Could split rendering logic and timeline processing
- Review and potentially refactor `automation/executor/simple_executor.go` (703 lines) and related executor logic
- Consider adding unit tests for the `handlers/export`, `handlers/workflow_helpers.go`, `internal/typeconv`, `workflow/validator`, `services/workflow_service_*`, and `services/recording_*` packages to improve coverage
- Service files are now well-organized with clear separation of concerns

## Requirements & Testing

### Current Issues (from scenario status)
- 14 critical requirements (P0/P1) lack multi-layer automated validation
- 7/63 requirements reference unsupported test/ directories
- 4 test files validate ≥4 requirements each (should be unique tests per requirement)
- 9 test files appear superficial (< 20 LOC, missing assertions, or no test functions)
- Requirements validation errors for non-existent test files:
  - `BAS-EXEC-TELEMETRY-STREAM` references missing `api/browserless/runtime/session_test.go`
  - `BAS-WORKFLOW-PERSIST-CRUD` references missing `api/browserless/runtime/session_test.go`

### Follow-Up Actions
- Add multi-layer validation (API + UI + e2e) for all P0/P1 requirements
- Fix requirement validation references to point to correct test file locations
- Expand test coverage for superficial tests
- Consolidate tests that validate multiple requirements into focused, single-requirement tests

## Resource Dependencies

### Missing Resources
- ⚠️ MinIO not installed (needed for object storage features)
- ⚠️ OpenRouter not installed (needed for AI features)

### Action Items
- Document whether MinIO and OpenRouter are required for core functionality
- If required, add to setup instructions; if optional, document graceful degradation
