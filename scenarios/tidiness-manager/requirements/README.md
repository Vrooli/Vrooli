# Requirements Registry

This directory contains the requirements registry for **tidiness-manager**, mapping PRD operational targets to technical requirements.

## Structure

```
requirements/
├── index.json                          # Parent registry importing all modules
├── 01-light-scanning/module.json       # P0: Makefile integration, file metrics
├── 02-smart-scanning/module.json       # P0: AI analysis, visited-tracker integration
├── 03-agent-api/module.json            # P0: HTTP/CLI API for agents
├── 04-ui-dashboard/module.json         # P0: React UI for human management
├── 05-auto-campaigns/module.json       # P1: Background tidiness campaigns
├── 06-issue-management/module.json     # P1: Issue triage, resolution tracking
├── 07-data-auditability/module.json    # P1: Scan history, config management
├── 08-future-features/module.json      # P2: Trend analysis, integrations
└── README.md                           # This file
```

## Requirement ID Pattern

- **TM-LS-NNN**: Light Scanning (Makefile, file metrics)
- **TM-SS-NNN**: Smart Scanning (AI, visited-tracker)
- **TM-API-NNN**: Agent API (HTTP/CLI interface)
- **TM-UI-NNN**: UI Dashboard (React components)
- **TM-AC-NNN**: Auto Campaigns (background enforcement)
- **TM-IM-NNN**: Issue Management (triage, resolution)
- **TM-DA-NNN**: Data & Auditability (history, config)
- **TM-FF-NNN**: Future Features (P2 capabilities)

## Lifecycle

1. Operational targets in `PRD.md` map to numbered folders (e.g., `01-light-scanning/`)
2. `index.json` imports each `module.json`; tests auto-sync their status when they run
3. Tag tests with `[REQ:TM-XX-NNN]` so auto-sync updates requirement validation arrays
4. PRD checkboxes flip automatically when sync runs—never edit them manually

## Quick Commands

```bash
# Sync requirements after running tests
vrooli scenario requirements sync tidiness-manager

# Review last synced status
vrooli scenario requirements snapshot tidiness-manager

# Lint PRD operational target coverage
vrooli scenario requirements lint-prd tidiness-manager
```

## Validation Strategy

- **Light Scanning (TM-LS)**: Unit test parsers, integration test `make lint/type` execution
- **Smart Scanning (TM-SS)**: Mock AI calls, verify batch limits and visited-tracker integration
- **Agent API (TM-API)**: Test HTTP endpoints, CLI commands, issue ranking
- **UI Dashboard (TM-UI)**: Component tests, Lighthouse scores, keyboard navigation
- **Auto Campaigns (TM-AC)**: State machine tests, concurrency limits, error handling
- **Issue Management (TM-IM)**: CRUD operations, status transitions, scan triggers
- **Data & Auditability (TM-DA)**: History tracking, config validation, force-scan queueing

See `../test/run-tests.sh` for the complete test harness.

## Notes for Implementers

1. Start with **01-light-scanning** (foundation for all other modules)
2. **02-smart-scanning** depends on light scanning infrastructure
3. **03-agent-api** and **04-ui-dashboard** can be developed in parallel
4. **05-auto-campaigns** requires both scanning engines functional
5. Build **07-data-auditability** in from the start, not retrofitted

Full tracking guide: `docs/testing/guides/requirement-tracking-quick-start.md`
