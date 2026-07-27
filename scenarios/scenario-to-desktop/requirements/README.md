# Requirements Tracking

This directory tracks PRD coverage and validation for the scenario-to-desktop system.

## Structure

```
requirements/
├── index.json                              # Master requirements registry
├── templates/electron.json                 # Electron template validation
├── scenarios/reference-implementations.json # Reference scenario tests
└── integration/e2e-validation.json         # End-to-end validation
```

## Operational Targets

This requirements registry tracks coverage against the PRD operational targets defined in [PRD.md](../PRD.md#-operational-targets). Each requirement module maps to one or more `OT-*` targets:

| Target | Priority | Status | Module(s) |
|--------|----------|--------|-----------|
| OT-P0-001 | P0 | Complete | `templates/electron.json` |
| OT-P0-002 | P0 | Complete | `templates/electron.json` |
| OT-P0-003 | P0 | Partial | `integration/e2e-validation.json` |
| OT-P0-004 | P0 | Complete | `integration/e2e-validation.json` |
| OT-P0-005 | P0 | Complete | `templates/electron.json` |
| OT-P0-006 | P0 | Complete | `templates/electron.json` |
| OT-P0-007 | P0 | Complete | `templates/electron.json` |
| OT-P1-001 | P1 | Pending | — |
| OT-P1-002 | P1 | Pending | — |
| OT-P1-003 | P1 | Pending | — |
| OT-P1-004 | P1 | Pending | — |
| OT-P1-005 | P1 | Pending | — |
| OT-P2-001 | P2 | Pending | — |
| OT-P2-002 | P2 | Pending | — |
| OT-P2-003 | P2 | Pending | — |

## Auto-Sync

Requirement statuses can be synced against the codebase using:

```bash
vrooli scenario requirements validate scenario-to-desktop --json
```

This compares `module.json` status fields against actual test results and code presence. After syncing, PRD checkboxes may auto-update to reflect validated completion.

Requirements reconciliation is owned by the business-health workflow. Consult
`business-health wizard apply --help` for the current argument contract rather
than copying a stale invocation here.

## Requirements Status

**Overall Progress**: Both bundled offline mode (recommended) and thin-client mode are production-ready.

### Stable Today ✅
- **Bundled offline mode** (recommended default) - complete offline desktop applications with runtime supervisor
- Template generation for Electron (multiple template configs)
- Thin-client mode for shared-server scenarios
- Development tooling (CLI, API, UI) and accompanying tests
- Native features in templates (tray, menus, file dialogs)
- API integration patterns (secure IPC, bundled runtime wiring)
- Auto-updater hooks available (requires manual signing/publish setup)

### Partial / Requires Environment ⚠️
- Cross-platform packaging depends on electron-builder and platform prerequisites (e.g., Wine for Windows on Linux)

### Future (P2) 🔮
- Code signing and notarization automation
- App store submission automation

## Validation

### E2E Tests
- **test/e2e/test-desktop-generation.sh**: 26/26 tests passing
- Validates generation → structure → dependencies → build workflow

### API Tests
- **api/main_test.go**: 27/27 tests passing
- **api/performance_test.go**: All benchmarks passing

### CLI Tests
- **cli/test.sh**: 8/8 BATS tests passing

### Business Tests
- **test/phases/test-business.sh**: 5/5 tests passing

### UI Tests
- **ui/src/components/__tests__/**: 7/7 tests passing

**Total**: Tests exist across API/CLI/UI/e2e; rerun suites to refresh pass/fail and coverage before reporting numbers.

## Reference Implementations

### Completed
- **picker-wheel**: Has `platforms/electron/` directory with generated desktop wrapper

### Planned
- System-monitor with advanced features
- Research-assistant with multi-window support

## Reporting

Requirements coverage can be generated via:
```bash
vrooli scenario requirements report scenario-to-desktop --format markdown
```

## Last Updated
Needs refresh after the next full test run and requirement sync.
