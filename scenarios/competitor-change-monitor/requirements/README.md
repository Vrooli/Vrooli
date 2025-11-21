# Competitor Change Monitor Requirements Registry

This directory contains the requirements registry for competitor-change-monitor, organized to track operational targets from the PRD.

## Structure

```
requirements/
└── index.json                    # Parent requirements + validation tracking
```

## Coverage

Run coverage reports with:

```bash
# JSON output (for CI/automation)
node ../../scripts/requirements/report.js --scenario competitor-change-monitor --format json

# Markdown output (for README badges)
node ../../scripts/requirements/report.js --scenario competitor-change-monitor --format markdown

# Auto-sync from test results
node ../../scripts/requirements/report.js --scenario competitor-change-monitor --mode sync
```

## Requirement Categories

- **CCM-FUNC-001**: Multi-source monitoring (3 child requirements)
- **CCM-FUNC-002**: Change detection and storage (2 child requirements)
- **CCM-FUNC-003**: AI-powered analysis (2 child requirements)
- **CCM-FUNC-004**: Alert system (2 child requirements)
- **CCM-FUNC-005**: Scheduled scanning (2 child requirements)
- **CCM-FUNC-006**: Business intelligence dashboard (3 child requirements)

**Total**: 6 parent requirements → 14 child requirements

## Validation Strategy

All requirements link to one or more validation methods:

- **test**: Go unit tests in `api/*_test.go` and integration tests in `test/`
- **manual**: UI verification via manual testing

Test tags like `[REQ:CCM-WEB-001]` automatically update requirement status when tests run.

## PRD Linkage

Each parent requirement maps to an operational target in PRD.md:
- `CCM-FUNC-001` → `OT-P0-001` (Multi-source monitoring)
- `CCM-FUNC-002` → `OT-P0-002` (Change detection and storage)
- `CCM-FUNC-003` → `OT-P0-003` (AI-powered analysis)
- `CCM-FUNC-004` → `OT-P0-004` (Alert system)
- `CCM-FUNC-005` → `OT-P0-005` (Scheduled scanning)
- `CCM-FUNC-006` → `OT-P0-006` (Business intelligence dashboard)
