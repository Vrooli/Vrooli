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
- Tauri/Neutralino frameworks
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
