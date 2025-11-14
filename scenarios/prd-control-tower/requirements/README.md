# PRD Control Tower Requirements Registry

This directory contains the modular requirements registry for prd-control-tower, organized by feature area.

## Structure

```
requirements/
├── index.json                    # Parent requirements + imports
├── catalog/
│   └── core.json                # Catalog browsing and PRD viewing
├── drafts/
│   └── lifecycle.json           # Draft CRUD and publishing
├── backlog/
│   └── intake.json              # Backlog intake and conversion
├── requirements/
│   └── integration.json         # Requirements parsing and linkage
├── ai/
│   └── generation.json          # AI-powered authoring
└── validation/
    └── auditor.json             # scenario-auditor integration
```

## Coverage

Run coverage reports with:

```bash
# JSON output (for CI/automation)
node ../../scripts/requirements/report.js --scenario prd-control-tower --format json

# Markdown output (for README badges)
node ../../scripts/requirements/report.js --scenario prd-control-tower --format markdown

# Auto-sync from test results
node ../../scripts/requirements/report.js --scenario prd-control-tower --mode sync
```

## Requirement Categories

- **PCT-FUNC-001**: Catalog browsing (4 child requirements)
- **PCT-FUNC-002**: Draft lifecycle (5 child requirements)
- **PCT-FUNC-003**: Backlog intake (3 child requirements)
- **PCT-FUNC-004**: Requirements integration (3 child requirements)
- **PCT-FUNC-005**: AI generation (3 child requirements)
- **PCT-FUNC-006**: Validation (3 child requirements)

**Total**: 6 parent requirements → 21 child requirements

## Validation Strategy

All requirements link to one or more validation methods:

- **test**: Go unit tests in `api/*_test.go`
- **manual**: UI verification via manual testing
- **automation**: End-to-end workflows (future)

Test tags like `[REQ:PCT-DRAFT-SAVE]` automatically update requirement status when tests run.
