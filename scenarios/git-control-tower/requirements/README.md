# Git Control Tower Requirements Registry

This directory contains the modular requirements registry for git-control-tower, organized by functional area.

## Structure

```
requirements/
├── index.json                    # Parent requirements + imports
├── api/
│   └── core.json                # Core API endpoints for git operations
├── cli/
│   └── commands.json            # CLI command parity with API
└── ui/
    └── dashboard.json           # Web UI dashboard and visualization
```

## Coverage

Run coverage reports with:

```bash
# JSON output (for CI/automation)
node ../../scripts/requirements/report.js --scenario git-control-tower --format json

# Markdown output (for README badges)
node ../../scripts/requirements/report.js --scenario git-control-tower --format markdown

# Auto-sync from test results
node ../../scripts/requirements/report.js --scenario git-control-tower --mode sync
```

## Requirement Categories

- **GCT-FUNC-001**: P0 operational targets - Core git API functionality (8 child requirements)
- **GCT-FUNC-002**: P1 operational targets - Extended functionality (4 child requirements)
- **GCT-FUNC-003**: P2 operational targets - Web UI and future expansion (3 child requirements)

**Total**: 3 parent requirements → 15 child requirements

## Operational Target Linkage

Each operational target in PRD.md is linked to one or more requirements via the `prd_ref` field:

- **P0 Targets** (OT-P0-001 through OT-P0-008): Linked to GCT-API-* and GCT-CLI-PARITY requirements
- **P1 Targets** (OT-P1-001 through OT-P1-004): Linked to GCT-API-BRANCHES, GCT-API-AI-COMMIT, GCT-API-CONFLICTS, GCT-API-PREVIEW
- **P2 Targets** (OT-P2-001): Linked to GCT-UI-DASHBOARD

## Validation Strategy

All requirements link to one or more validation methods:

- **test**: Go unit tests in `api/*_test.go`
- **manual**: UI verification via manual testing
- **integration**: CLI integration tests in `cli/test.sh`

Test tags like `[REQ:GCT-API-STATUS]` automatically update requirement status when tests run.

## Adding New Requirements

1. Identify which module file the requirement belongs to (api/core.json, cli/commands.json, or ui/dashboard.json)
2. Add the requirement to the appropriate module file with a unique ID (e.g., `GCT-API-NEW-FEATURE`)
3. Set the `prd_ref` field to reference the operational target in PRD.md (e.g., `"Operational Targets > P1 > OT-P1-005"`)
4. Add validation entries for tests or manual verification
5. Update the parent requirement in index.json to include the new child ID in the `children` array
6. Run sync to validate the requirement linkage
