# Personal Digital Twin Requirements Registry

This directory contains the modular requirements registry for personal-digital-twin, organized by functional area.

## Structure

```
requirements/
└── index.json                    # All requirements with PRD references
```

## Coverage

Run coverage reports with:

```bash
# JSON output (for CI/automation)
node ../../scripts/requirements/report.js --scenario personal-digital-twin --format json

# Markdown output (for README badges)
node ../../scripts/requirements/report.js --scenario personal-digital-twin --format markdown

# Auto-sync from test results
node ../../scripts/requirements/report.js --scenario personal-digital-twin --mode sync
```

## Requirement Categories

- **PDT-FUNC-001**: Persona Management (4 child requirements)
- **PDT-FUNC-002**: Data Ingestion (3 child requirements)
- **PDT-FUNC-003**: Knowledge Retrieval (2 child requirements)
- **PDT-FUNC-004**: Chat Interface (3 child requirements)
- **PDT-FUNC-005**: Personality Modeling (2 child requirements)
- **PDT-FUNC-006**: Multi-Persona Support (2 child requirements)
- **PDT-FUNC-007**: RESTful API (4 child requirements)
- **PDT-FUNC-008**: CLI Interface (4 child requirements)

**Total**: 8 parent requirements → 24 child requirements

## Validation Strategy

All requirements link to one or more validation methods:

- **test**: Go unit tests in `api/*_test.go` or CLI tests
- **manual**: UI/UX verification via manual testing

Test tags like `[REQ:PDT-PERSONA-CREATE]` automatically update requirement status when tests run.

## PRD References

Each requirement includes a `prd_ref` field that links to the corresponding operational target in PRD.md:

- Format: `"Operational Targets > {Priority} > {Target-ID}"`
- Example: `"Operational Targets > P0 > OT-P0-001"`

This creates bidirectional traceability between high-level goals (PRD) and implementation details (requirements).
