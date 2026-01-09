# Chart Generator Requirements

This directory contains the requirements registry for the chart-generator scenario, organized by operational target.

## Structure

Requirements are organized into modules aligned with PRD operational targets. Each module groups related requirements that together fulfill broader operational goals.

### P0 (Must-have) Modules
- `01-core-features/` - Core chart types, data ingestion, and professional styling (3 requirements: CHART-P0-001, CHART-P0-002, CHART-P0-003)
- `02-interfaces/` - CLI and Web UI for chart generation and management (2 requirements: CHART-P0-004, CHART-P0-007)
- `03-export-persistence/` - Multi-format export and database persistence (2 requirements: CHART-P0-005, CHART-P0-006)

### P1 (Post-launch) Modules
- `04-advanced-charts/` - Specialized chart types for business and financial use (2 requirements: CHART-P1-001, CHART-P1-002)
- `05-advanced-features/` - Customization, interactivity, and template library (3 requirements: CHART-P1-003, CHART-P1-004, CHART-P1-008)
- `06-advanced-export/` - PDF export and multi-chart composition (2 requirements: CHART-P1-005, CHART-P1-006)
- `07-data-pipeline/` - Data transformation and processing (1 requirement: CHART-P1-007)

## Validation Coverage

Each requirement is validated through multiple test layers:
- **Unit tests** (`api/*_test.go`) - Core logic validation
- **Integration tests** (`cli/chart-generator.bats`, `api/main_test.go`) - End-to-end workflows
- **Manual validation** (for UI-only features like style preview)

## Test Execution

Run all tests via the phased testing system:
```bash
./test/run-tests.sh
```

Individual phases:
- `./test/phases/test-unit.sh` - Go unit tests
- `./test/phases/test-integration.sh` - CLI BATS tests
- `./test/phases/test-structure.sh` - Infrastructure validation

## Coverage Reports

- Unit test coverage: `coverage/chart-generator/aggregate.json`
- HTML coverage: `coverage/.work/unit/go/coverage.html`
- Current coverage: 62.4% (target: 80%+)

## Notes

- All requirements reference actual test files (no phantom test paths)
- Multi-layer validation ensures requirements are tested at appropriate layers
- Manual validation tracked for UI-only features (CHART-P0-007)
