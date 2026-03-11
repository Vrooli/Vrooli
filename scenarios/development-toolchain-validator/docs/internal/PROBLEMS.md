# Known Issues & Deferred Work

## Open Issues

- **prd-control-tower requirements generation**: The tool experienced EOF errors during initial requirements generation. The requirements were eventually generated but this may recur. Workaround: Create requirements manually if the tool fails.

## Deferred Ideas

- **Conflict detection sophistication**: Simple path-based overlap detection may miss semantic conflicts (e.g., two skills expecting different patterns in the same file). More sophisticated analysis (AST-based, content comparison) is deferred to P1 conflict detection work.

- **Auto-config suggestions (P2)**: Analyzing SKILL.md content to suggest structural expectations requires NLP/pattern matching against free-form markdown. This is a significant effort deferred to P2.

- **Multi-template references**: Currently only react-vite template is planned. Adding CLI-only or landing-page templates requires defining which steer skills apply to each template type.

- **Concurrent CLI execution**: Running multiple CLI tool assertions sequentially can be slow. Parallel execution with goroutines is possible but needs careful resource management (port conflicts, CPU/memory).

## Tech Debt

- **Operational target 1:1 mapping penalty**: 76% of operational targets have 1:1 requirement mapping, causing an 8pt scoring penalty. The requirements structure mirrors the PRD targets too closely. To resolve: consolidate related requirement modules under shared operational targets (e.g., combine reference registry + skill connection under a "Core Data Layer" target, or validation domains under "Validation Engine"). This requires requirements restructuring and index.json updates.

## Test Gaps

- **Validation domain handlers**: The `domain/validation/` package now has structural checker and CLI executor implementations with 91.0% coverage, but no HTTP handlers yet for running validations via API.
- **Report domain tests**: The `domain/report/` package has no tests. This domain is `draft` status (REQ-P0-010) and not yet implemented.
- **Expectation handler response body validation**: Some tests check HTTP status but not response body content. Added tests for InvalidJSON error responses (2026-03-11).
- **CLI test coverage**: CLI tests cover reference and connection commands with 53.1% coverage. Expectation/validation CLI commands are not yet implemented (REQ-P0-011a pending).

## Resolved Tech Debt

- ~~**Missing expectation handler tests**: GetByID and Delete handler tests missing.~~ **RESOLVED (2026-03-11)**: Added `TestExpectationHandler_GetStructural`, `TestExpectationHandler_GetCLI`, `TestExpectationHandler_DeleteStructural`, `TestExpectationHandler_DeleteCLI`, plus InvalidJSON and filter tests. Added proper [REQ:ID] header tags. Test count: 396→411 API tests.
- ~~**Monolithic test file**: One test file validates 4+ requirements, causing a 2pt penalty.~~ **RESOLVED (2026-03-11)**: Split `api/domain/skill/service_test.go` into `connection_service_test.go` (REQ-P0-003) and `drift_service_test.go` (REQ-P0-004). Updated requirement module refs accordingly.
- ~~**Handlers coverage below target**: Coverage at 62.9% was below 80% target.~~ **RESOLVED (2026-03-11)**: Added `errors_test.go`, dry-run tests for all mutating endpoints, `TestNewReferenceHandlerWithConfig`, `TestReferenceHandler_Create_DryRun`, `TestReferenceHandler_Update_DryRun`, `TestReferenceHandler_Delete_DryRun`, `TestReferenceHandler_Update_PartialFields`. Coverage improved 62.9% → 90.9% (exceeds 80% target).
- ~~**Testutil coverage at 40.3%**: Missing tests for DecodeJSONResponse and factory builder methods.~~ **RESOLVED (2026-03-11)**: Added `TestDecodeJSONResponse`, `TestReferenceFactory` (9 subtests), `TestCreateInputFactory` (7 subtests). Coverage improved 40.3% → 90.3%.
- ~~**Expectation domain coverage at 76.8%**: Missing tests for GetByID, Delete, and CLI operations.~~ **RESOLVED (2026-03-11)**: Added 12 new tests covering GetStructuralByID, DeleteStructural, GetCLIByID, ListCLI, DeleteCLI, DeleteCLIByConnection, WithConfig, plus validation failure tests. Coverage improved 76.8% → 98.2%.
- ~~**CLI coverage at 32.4%**: CLI tests covered command routing but limited coverage for create/update/get/delete/drift validation flows.~~ **IMPROVED (2026-03-11)**: Added 23 new CLI tests covering alias commands, create field validation (all 4 required fields), update field validation, get/delete validation, connection connect/get/disconnect/drift validation. Coverage improved 32.4% → 53.1%.
- ~~**Validation domain tests**: The `domain/validation/` package has no tests.~~ **RESOLVED (2026-03-11)**: Implemented structural validation engine with `model.go` (validation types), `structural_checker.go` (file/folder/content validation), and comprehensive `structural_checker_test.go` (25+ tests). Coverage: 89.4%. REQ-P0-007/REQ-P0-007a status updated from `draft` to `implemented`.
- ~~**CLI tool validation**: The CLI executor for running assertions against tool output (OT-P0-007) is not yet implemented.~~ **RESOLVED (2026-03-11)**: Implemented CLI executor with `cli_executor.go` (command execution with timeout/sandboxing, JSON parsing, JSONPath extraction, all 10 assertion operators). Split tests: `cli_command_test.go` (REQ-P0-008, 8 tests) and `cli_assertion_test.go` (REQ-P0-008a, 12 tests). Coverage: 91.0%. REQ-P0-008/REQ-P0-008a status updated from `draft` to `implemented`.
