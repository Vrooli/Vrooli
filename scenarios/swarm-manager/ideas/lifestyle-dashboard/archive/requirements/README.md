# Lifestyle Dashboard Requirements Registry

This directory contains the modular requirements registry for lifestyle-dashboard, organized by feature area.

## Structure

```
requirements/
├── index.json                    # Parent requirements + imports
├── events/
│   └── schema.json              # Event schema and storage
├── domains/
│   └── registration.json        # Domain registration and discovery
├── queries/
│   └── api.json                 # Cross-domain query API
├── dashboard/
│   └── ui.json                  # Dashboard UI components
├── briefs/
│   └── daily.json               # Daily brief system
└── intelligence/
    └── correlation.json         # Correlation engine and experiments
```

## Coverage

| Parent Requirement | PRD Target | Child Requirements | Status |
|-------------------|------------|-------------------|--------|
| LD-FUNC-001 | OT-P0-001 | 3 (schema, storage, index) | not_started |
| LD-FUNC-002 | OT-P0-002 | 3 (register, discover, health) | not_started |
| LD-FUNC-003 | OT-P0-003 | 3 (filter, aggregate, correlate) | not_started |
| LD-FUNC-004 | OT-P0-004 | 4 (timeline, domains, trends, responsive) | not_started |
| LD-FUNC-005 | OT-P0-005 | 3 (morning, evening, consolidate) | not_started |
| LD-FUNC-006 | OT-P1-001 | 3 (analyze, significance, threshold) | not_started |
| LD-FUNC-007 | OT-P1-002 | 3 (define, track, report) | not_started |

**Total**: 7 parent requirements → 22 child requirements

## Requirement Categories

- **LD-FUNC-001**: Shared event schema & storage (P0)
- **LD-FUNC-002**: Domain registration & discovery (P0)
- **LD-FUNC-003**: Cross-domain query API (P0)
- **LD-FUNC-004**: Unified dashboard UI (P0)
- **LD-FUNC-005**: Daily brief system (P0)
- **LD-FUNC-006**: Correlation engine (P1)
- **LD-FUNC-007**: Experiment framework (P1)

## Validation Strategy

All requirements link to validation methods:

- **test**: Go unit tests in `api/*_test.go`
- **integration**: API endpoint tests
- **e2e**: End-to-end workflows via BAS

Test tags like `[REQ:LD-EVENT-SCHEMA]` automatically update requirement status when tests run.
