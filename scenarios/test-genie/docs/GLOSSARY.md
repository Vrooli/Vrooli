# Testing Glossary

Quick reference for Vrooli testing terminology. Use this to understand concepts used throughout the testing documentation.

## Core Concepts

### Phase
One of 6 sequential test stages that progressively validate scenario quality:
1. **Structure** (15s) - Files and configuration
2. **Dependencies** (30s) - Package and resource availability
3. **Unit** (60s) - Code-level tests
4. **Integration** (120s) - Component interactions
5. **Business** (180s) - End-to-end workflows
6. **Performance** (60s) - Benchmarks and load testing

Phases run sequentially and fail-fast - each phase must pass before the next executes.

**See**: [Phased Testing Guide](guides/phased-testing.md)

### Requirement
A product or technical need defined in the PRD or requirement registry. Requirements have:
- **Unique ID**: Pattern `[A-Z][A-Z0-9]+-[A-Z0-9-]+` (e.g., `BAS-WORKFLOW-PERSIST-CRUD`)
- **Status**: Current implementation state (pending, planned, in_progress, complete, not_implemented)
- **Criticality**: Priority level (P0, P1, P2)
- **Validations**: Methods used to verify the requirement (tests, automations, manual checks)

Requirements bridge the gap between PRD and technical implementation.

**See**: [Requirements Sync Guide](guides/requirements-sync.md)

### Validation
A method of verifying a requirement is met. Three types:
- **test**: Unit, integration, or business tests (Go, Vitest, Python)
- **automation**: BAS workflow automation for UI testing
- **manual**: Manual verification steps or exploratory testing

Each validation has a **ref** (file path or workflow ID) and **status** (not_implemented, planned, implemented, failing).

### [REQ:ID]
Tag format for linking tests to requirements. Place in test names:
```typescript
// Vitest
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => { ... });

// Go
t.Run("creates workflow [REQ:BAS-WORKFLOW-PERSIST-CRUD]", func(t *testing.T) { ... })
```

The testing system automatically extracts these tags and updates requirement tracking - no manual tracking needed.

**See**: [Requirements Sync Guide](guides/requirements-sync.md)

### Auto-Sync
Automatic update of requirement files based on live test results. After tests run, the sync system:
- Adds missing validation entries for tagged tests
- Updates validation status (passed -> implemented, failed -> failing)
- Updates requirement status based on validation roll-up
- Removes orphaned validations (with `--prune-stale`)

Zero manual maintenance required once tests are tagged.

**See**: [Requirements Sync Guide](guides/requirements-sync.md)

### Phase Results
JSON output from each phase execution, stored in `coverage/phase-results/<phase>.json`. Contains:
- Phase status (passed/failed)
- Test counts and errors
- **Per-requirement evidence** (which requirements were validated, how they performed)
- Timing information

Phase results feed into auto-sync and reporting systems.

## Status Values

### Requirement Status
Current implementation state:
- **pending**: Requirement identified but not yet planned
- **planned**: Validation approach defined, tests not written yet
- **in_progress**: At least one test exists (auto-detected by sync)
- **complete**: All validations passing (auto-updated by sync)
- **not_implemented**: Requirement deprioritized or deferred

Status transitions happen automatically during sync based on test results.

### Validation Status
Current test implementation state:
- **not_implemented**: Validation identified but test not written
- **planned**: Validation approach defined, test not yet written
- **implemented**: Test exists and passing (auto-updated by sync)
- **failing**: Test exists but currently failing (auto-updated by sync)

## Criticality Levels

### P0 - Critical (Must Have)
- Core functionality required for MVP
- Blocking bugs that prevent major workflows
- Security vulnerabilities
- Data loss scenarios

**CI/CD**: P0 requirements with `status != complete` block releases

### P1 - Important (Should Have)
- Secondary features that significantly improve UX
- Non-blocking bugs with workarounds
- Performance issues affecting >50% of users

**CI/CD**: P1 gaps tracked but may not block releases

### P2 - Nice-to-Have (Could Have)
- Polish and convenience features
- Minor bugs with minimal impact
- Optimization for edge cases

**Tracking**: P2 requirements tracked but typically not release-critical

### Criticality Gap
Count of P0/P1 requirements with `status != complete`. Use to gate releases:
```bash
# Fail CI if criticality gap > 0
test-genie execute my-scenario --preset comprehensive --fail-on-critical-gap
```

## Tools & Systems

### BAS (Browser Automation Studio)
Vrooli's canonical UI automation system. Write UI tests as **declarative JSON workflows**:
- Author workflows visually in BAS UI
- Export to JSON for version control
- Execute via API (no manual clicking)
- Integrates with requirement tracking via `automation` validation type

**Self-testing**: BAS tests itself using its own automation capabilities.

**See**: [UI Testability Guide](guides/ui-testability.md)

### vitest-requirement-reporter
Custom Vitest reporter (`@vrooli/vitest-requirement-reporter`) that automatically extracts `[REQ:ID]` tags from test names and generates `coverage/vitest-requirements.json` for phase integration.

**Configuration**:
```typescript
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  test: {
    reporters: [
      'default',
      new RequirementReporter({
        outputFile: 'coverage/vitest-requirements.json',
        emitStdout: true,  // Required for phase integration
        verbose: true,
      }),
    ],
  },
});
```

### Phase Helpers

Phase lifecycle is managed by the Go-native test-genie orchestrator at `api/internal/orchestrator/phases/` with typed `PhaseRunner` interfaces.

See [Architecture](concepts/architecture.md) for implementation details.

### Requirements Registry
Structured JSON file(s) mapping PRD requirements to technical validations (stored in the `requirements/` directory with imports).

Registry includes metadata, requirement definitions, validation entries, and parent-child relationships.

**Bootstrap**: `vrooli scenario requirements init <scenario-name>`

## Test Presets

### Quick Preset
- Phases: Structure, Dependencies only
- Timeout: ~45 seconds
- Use: Fast iteration during development

### Smoke Preset
- Phases: Structure, Dependencies, Unit
- Timeout: ~2 minutes
- Use: Pre-commit validation

### Comprehensive Preset
- Phases: All 7 phases
- Timeout: ~10 minutes
- Use: CI/CD, release validation
- Triggers: Auto-sync of requirements

**See**: [Presets Reference](reference/presets.md)

## File Locations

### Configuration
- `.vrooli/service.json` - Scenario configuration including test lifecycle
- `.vrooli/testing.json` - Test-specific configuration
- `vite.config.ts` - Vitest configuration with requirement reporter (UI scenarios)

### Test Structure
- `test/playbooks/` - BAS workflow JSON files for E2E automation
- `api/*_test.go` - Go unit and integration tests
- `ui/src/**/*.test.ts` - Vitest unit tests for UI

### Requirements
- `requirements/index.json` - Parent requirements with imports (modular)
- `requirements/**/*.json` - Child requirement modules by feature (modular)
- `PRD.md` - Product Requirements Document

### Test Output
- `coverage/phase-results/<phase>.json` - Per-phase test results
- `ui/coverage/vitest-requirements.json` - Vitest requirement tracking output
- `api/coverage.out` - Go test coverage data

### Orchestration
All test orchestration is now handled by the Go-native test-genie API. Use `test-genie execute` or the REST API to run tests.

## Common Commands

### Run Tests
```bash
# Run all phases via CLI
test-genie execute my-scenario --preset comprehensive

# Run specific phase
test-genie execute my-scenario --phases unit

# Run with Makefile
cd scenarios/my-scenario && make test

# Quick iteration during development
test-genie execute my-scenario --preset quick
```

### Requirement Management
```bash
# Bootstrap registry
vrooli scenario requirements init my-scenario

# Validate schema
node scripts/requirements/validate.js --scenario my-scenario

# Generate report
vrooli scenario requirements report my-scenario --format markdown

# Auto-sync from test results (happens automatically with comprehensive preset)
test-genie execute my-scenario --preset comprehensive
```

### Coverage
```bash
# Go coverage
cd api && go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Vitest coverage
cd ui && npm test -- --coverage

# test-genie coverage analysis
test-genie coverage my-scenario --depth deep
```

## Related Documentation

### Getting Started
- [QUICKSTART](QUICKSTART.md) - Get started in 5 minutes
- [Safety Guidelines](safety/GUIDELINES.md) - **CRITICAL** - Prevent data loss

### Deep Dives
- [Phased Testing Guide](guides/phased-testing.md) - Complete phase system
- [Requirements Sync Guide](guides/requirements-sync.md) - Requirement tracking
- [Architecture](concepts/architecture.md) - Go orchestrator design

### Implementation
- [Scenario Unit Testing](guides/scenario-unit-testing.md) - Go, Node, Python tests
- [CLI Testing](guides/cli-testing.md) - BATS testing patterns
- [UI Testability](guides/ui-testability.md) - BAS workflow testing

---

**Tip**: Use your browser's search (Ctrl+F / Cmd+F) to quickly find terms in this glossary.
