# Simple Test Requirements Registry

This directory contains the requirements registry for simple-test, organized to validate Vrooli's lifecycle and testing infrastructure.

## Structure

```
requirements/
└── index.json                    # Core requirements for lifecycle validation
```

## Coverage

Run coverage reports with:

```bash
# JSON output (for CI/automation)
node ../../scripts/requirements/report.js --scenario simple-test --format json

# Markdown output (for README badges)
node ../../scripts/requirements/report.js --scenario simple-test --format markdown

# Auto-sync from test results
node ../../scripts/requirements/report.js --scenario simple-test --mode sync
```

## Requirement Categories

- **ST-FUNC-001**: Lifecycle Management (4 child requirements)
- **ST-FUNC-002**: API Health Endpoint (2 child requirements)
- **ST-FUNC-003**: Test Coverage (2 child requirements)
- **ST-FUNC-004**: Phased Testing Framework (3 child requirements)

**Total**: 4 parent requirements → 11 child requirements

## Validation Strategy

All requirements are validated through automated testing:

- **test**: Jest unit tests and Bash integration scripts
- Test tags like `[REQ:ST-LIFECYCLE-SETUP]` automatically update requirement status when tests run

## Current Status

All P0 and P1 requirements are validated with 93.75% code coverage achieved through comprehensive test suite.
