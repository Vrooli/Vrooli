# Graph Studio Test Suite

Comprehensive phased testing architecture for Graph Studio scenario.

## Overview

The test suite is organized into phases that validate different aspects of the scenario:

1. **Unit Tests** - Individual component testing (Go code quality, compilation)
2. **Integration Tests** - End-to-end workflows (graph lifecycle, conversions)
3. **API Tests** - All API endpoints (CRUD, validation, rendering)
4. **CLI Tests** - Command-line interface (help, status, operations)
5. **UI Tests** - User interface (accessibility, content loading)

## Quick Start

```bash
# Run all tests
./test/run-tests.sh

# Run specific phase
./test/phases/test-unit.sh
./test/phases/test-integration.sh
./test/phases/test-api.sh
./test/phases/test-cli.sh
./test/phases/test-ui.sh
```

## Test Phases

### Phase 1: Unit Tests (`test-unit.sh`)

Tests individual components without dependencies:

- **Go Build** - Verifies code compiles without errors
- **Go Format** - Checks code follows gofmt standards
- **Go Vet** - Static analysis for common Go issues
- **Go Unit Tests** - Runs `*_test.go` files (when present)

**Requirements**: Go toolchain installed

### Phase 2: Integration Tests (`test-integration.sh`)

Tests complete workflows and component interactions:

- **Plugin Listing** - Verifies plugins load correctly
- **Conversion Matrix** - Checks conversion paths available
- **Graph Lifecycle** - Create → Get → Update → Delete flow
- **Graph Validation** - Validates graph data structures
- **Graph Conversion** - Tests format conversion engine

**Requirements**: Scenario running (`make run`)

### Phase 3: API Tests (`test-api.sh`)

Comprehensive API endpoint validation:

- **Core Endpoints** - Health, stats, plugins, graphs, conversions
- **CRUD Operations** - Create, read, update, delete graphs
- **Validation** - Request validation and error handling
- **Graph Operations** - Validate, render, convert endpoints
- **System Endpoints** - Metrics, detailed health

**Requirements**: API server running on configured port

### Phase 4: CLI Tests (`test-cli.sh`)

Command-line interface testing:

- **CLI Installation** - Verifies `graph-studio` command available
- **Core Commands** - help, status, plugins, list
- **Graph Operations** - create, get, delete workflows
- **Error Handling** - Invalid inputs handled gracefully

**Requirements**: CLI installed (`cli/install.sh`)

### Phase 5: UI Tests (`test-ui.sh`)

User interface validation:

- **Accessibility** - UI responds on configured port
- **Content Loading** - Page renders without errors
- **Asset Serving** - Static files serve correctly
- **Error Detection** - No obvious error messages in HTML

**Requirements**: UI server running on configured port

## Test Results

Each phase reports:
- Individual test pass/fail status
- Total tests passed/failed per phase
- Overall summary of all phases

Example output:
```
=========================================
Test Summary
=========================================
Phases Passed: 4
Phases Failed: 1

✓ Unit Tests: 4 passed, 0 failed
✓ Integration Tests: 5 passed, 0 failed
✗ API Tests: 13 passed, 1 failed
✓ CLI Tests: 7 passed, 0 failed
✓ UI Tests: 4 passed, 0 failed
```

## Writing New Tests

### Adding Unit Tests

Create `*_test.go` files in `api/` directory:

```go
package main

import "testing"

func TestGraphValidation(t *testing.T) {
    // Test implementation
}
```

### Adding Integration Tests

Add new test function to `test-integration.sh`:

```bash
test_new_workflow() {
    # Setup
    local response=$(curl -sf ...)

    # Validation
    echo "$response" | jq -e '.expected_field' &>/dev/null
}

# Add to main execution
run_test "New Workflow" test_new_workflow || true
```

### Adding API Tests

Add endpoint test to `test-api.sh`:

```bash
test_new_endpoint() {
    local response=$(curl -sf "$API_URL/api/v1/new-endpoint")
    echo "$response" | jq -e 'has("expected_key")' &>/dev/null
}
```

## Continuous Integration

The test suite integrates with Vrooli's lifecycle system:

```bash
# Via Makefile
make test

# Via CLI
vrooli scenario test graph-studio

# Via service.json
# Automatically runs phased test suite
```

## Debugging Failed Tests

### Check Scenario Status
```bash
make status
vrooli scenario status graph-studio
```

### View Logs
```bash
make logs
vrooli scenario logs graph-studio
```

### Run Individual Test
```bash
./test/phases/test-api.sh  # Run specific phase
```

### Verbose Output
Edit test scripts to add `set -x` for bash debugging:
```bash
set -euxo pipefail  # Add 'x' for verbose mode
```

## Test Coverage

Current coverage by phase:

| Phase | Coverage | Notes |
|-------|----------|-------|
| Unit | Go code quality | Format, vet, build |
| Integration | 5 core workflows | CRUD, validation, conversion |
| API | 14 endpoints | All documented API routes |
| CLI | 7 commands | Core CLI operations |
| UI | 4 checks | Accessibility, rendering |

## Performance Benchmarks

Target execution times:
- Unit Tests: < 10 seconds
- Integration Tests: < 30 seconds
- API Tests: < 45 seconds
- CLI Tests: < 15 seconds
- UI Tests: < 10 seconds
- **Total Suite: < 2 minutes**

## Known Issues

See `PROBLEMS.md` for documented test failures and workarounds.

## Contributing

When adding features:
1. Write tests before implementation (TDD)
2. Ensure all existing tests still pass
3. Update this README if adding new test phases
4. Document any new test requirements

---

**Test Suite Version**: 1.0.0
**Last Updated**: 2025-10-02
**Maintainer**: Graph Studio Team
