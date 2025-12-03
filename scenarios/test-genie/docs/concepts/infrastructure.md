# Testing Infrastructure

**Status**: Active (Migrated to Go-Native)
**Last Updated**: 2025-12-02

---

## Overview

This document describes the testing infrastructure components used by Test Genie to execute and manage tests across Vrooli scenarios.

> **Architecture Note**: Test Genie now uses a **Go-native orchestrator** for all test phases. The legacy bash infrastructure documented in some sections below is retained for historical reference only. See [Architecture](architecture.md) for the current Go-based design.

### Current Architecture (Go-Native)

Test Genie's orchestrator is implemented entirely in Go:

```
api/
├── orchestrator/           # Test orchestration engine
│   ├── phases/            # Phase runners (structure, unit, etc.)
│   ├── requirements/      # Requirements sync engine
│   └── workspace.go       # Scenario workspace management
├── execution/             # Execution history and tracking
├── queue/                 # Suite request queue
└── scenarios/             # Scenario catalog and discovery
```

**Key Benefits of Go-Native:**
- No external bash dependencies
- Typed interfaces for all phases
- Structured logging and error handling
- Direct API/CLI integration
- Portable across environments

### Legacy Infrastructure (Deprecated)

The sections below document BATS and bash shell libraries that were used before the Go migration. These are kept for historical reference but **should not be used for new development**.

---

## Legacy: BATS Framework

> **⚠️ DEPRECATED**: BATS is still used for CLI testing in some scenarios but the shell testing infrastructure has been replaced by Go. For new CLI tests, consider writing Go tests instead.

[BATS (Bash Automated Testing System)](https://github.com/bats-core/bats-core) is the standard framework for testing CLI tools and shell scripts in Vrooli.

### Installation

BATS is included in the Vrooli development environment:

```bash
# Verify installation
bats --version
# Output: Bats 1.10.0

# Install if needed (Ubuntu/Debian)
sudo apt-get install bats

# Or via npm
npm install -g bats
```

### Basic Structure

```bash
#!/usr/bin/env bats

# Optional setup/teardown
setup() {
    export TEST_PREFIX="/tmp/mytest-$$"
}

teardown() {
    rm -f "${TEST_PREFIX}"* 2>/dev/null || true
}

@test "my first test" {
    run echo "hello"
    [ "$status" -eq 0 ]
    [ "$output" = "hello" ]
}

@test "my second test" {
    run false
    [ "$status" -ne 0 ]
}
```

### Key Patterns

**The `run` command:**
```bash
@test "captures output and status" {
    run my-command --flag value

    # $status contains exit code
    [ "$status" -eq 0 ]

    # $output contains stdout
    [[ "$output" == *"expected"* ]]
}
```

**Assertions:**
```bash
@test "assertion examples" {
    # Equality
    [ "$actual" = "$expected" ]

    # Inequality
    [ "$status" -ne 1 ]

    # String contains
    [[ "$output" == *"substring"* ]]

    # Numeric comparison
    [ "$count" -gt 5 ]

    # File exists
    [ -f "$filepath" ]

    # Directory exists
    [ -d "$dirpath" ]
}
```

### BATS Helpers

Vrooli includes helper libraries:

```bash
# In your .bats file
load '../node_modules/bats-support/load'
load '../node_modules/bats-assert/load'

@test "with bats-assert" {
    run my-command

    assert_success
    assert_output --partial "expected text"
    assert_line --index 0 "first line"
}
```

---

## Legacy: Shell Testing Libraries

> **⚠️ DEPRECATED**: These shell libraries have been replaced by Go implementations. See [Architecture](architecture.md) for the Go-native approach. This section is retained for historical reference only.

Test Genie previously provided shell libraries for common testing operations.

### Directory Structure

```
scripts/scenarios/testing/
├── shell/
│   ├── core.sh           # Scenario detection and config
│   ├── connectivity.sh   # API/UI endpoint testing
│   ├── dependencies.sh   # Dependency validation
│   └── phase-helpers.sh  # Phase lifecycle helpers
├── unit/
│   ├── run-go.sh         # Go test runner
│   ├── run-node.sh       # Node.js test runner
│   └── run-python.sh     # Python test runner
└── lighthouse/
    ├── runner.sh         # Lighthouse execution
    └── config.sh         # Configuration helpers
```

### Sourcing Libraries

```bash
#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

source "${APP_ROOT}/scripts/scenarios/testing/shell/core.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"
```

### Core Functions

See [Shell Libraries Reference](../reference/shell-libraries.md) for complete function documentation.

**Quick reference:**

```bash
# Detect scenario
scenario_name=$(testing::core::detect_scenario)

# Get API URL
api_url=$(testing::connectivity::get_api_url)

# Test API health
if testing::connectivity::test_api; then
    echo "API is healthy"
fi

# Validate dependencies
testing::dependencies::validate_all --scenario "$scenario_name"
```

---

## Legacy: Binary Stub Framework

> **⚠️ DEPRECATED**: For Go tests, use standard Go mocking patterns (interfaces, testify/mock). This bash stub framework is retained for historical reference.

The stub framework was used for testing CLI tools without external dependencies.

### Purpose

- Mock external commands during tests
- Control command output for specific test cases
- Isolate tests from system state

### Creating Stubs

```bash
# In your test setup
setup() {
    # Create stub directory
    export STUB_DIR="/tmp/stubs-$$"
    mkdir -p "$STUB_DIR"

    # Create stub for 'curl'
    cat > "$STUB_DIR/curl" << 'EOF'
#!/bin/bash
echo '{"status": "ok"}'
exit 0
EOF
    chmod +x "$STUB_DIR/curl"

    # Add stubs to PATH
    export PATH="$STUB_DIR:$PATH"
}

teardown() {
    rm -rf "$STUB_DIR"
}

@test "uses stubbed curl" {
    run my-command-that-calls-curl

    [ "$status" -eq 0 ]
    [[ "$output" == *"ok"* ]]
}
```

### Dynamic Stubs

```bash
# Stub that behaves differently based on arguments
cat > "$STUB_DIR/docker" << 'EOF'
#!/bin/bash
case "$1" in
    "ps")
        echo "container1"
        echo "container2"
        ;;
    "logs")
        echo "Log output for $2"
        ;;
    *)
        echo "Unknown command: $1" >&2
        exit 1
        ;;
esac
EOF
```

### Stub Library

```bash
source "${APP_ROOT}/scripts/scenarios/testing/shell/stubs.sh"

# Create stub with expected behavior
testing::stub::create "docker" "ps" "container1\ncontainer2"
testing::stub::create "docker" "logs *" "Log output"

# Verify stub was called
testing::stub::assert_called "docker" "ps"
testing::stub::assert_called_with "docker" "logs mycontainer"
```

---

## Test Directory Structure

### Scenario Test Layout

```
scenario/
├── test/
│   ├── phases/              # Phase implementation scripts
│   │   ├── test-structure.sh
│   │   ├── test-dependencies.sh
│   │   ├── test-unit.sh
│   │   ├── test-integration.sh
│   │   ├── test-business.sh
│   │   └── test-performance.sh
│   ├── playbooks/           # BAS automation workflows
│   │   ├── capabilities/
│   │   │   ├── projects/
│   │   │   │   └── create-project.json
│   │   │   └── workflows/
│   │   │       └── execute-workflow.json
│   │   └── registry.json    # Playbook manifest
│   └── run-tests.sh         # Test entry point
├── api/
│   ├── *_test.go            # Go unit tests
│   └── test_helpers.go      # Test utilities
├── ui/src/
│   └── **/*.test.tsx        # Vitest unit tests
└── cli/
    └── *.bats               # CLI BATS tests (if applicable)
```

### Coverage Output

```
coverage/
├── phase-results/           # Phase execution results
│   ├── structure.json
│   ├── dependencies.json
│   ├── unit.json
│   ├── integration.json
│   ├── business.json
│   └── performance.json
├── sync/                    # Requirements sync metadata
│   └── *.json
├── go-coverage.out          # Go coverage data
├── vitest-requirements.json # Vitest requirement evidence
└── lighthouse/              # Lighthouse reports
    └── *.html
```

---

## Legacy: Phase Scripts

> **⚠️ DEPRECATED**: Phase scripts are now implemented in Go. See `api/orchestrator/phases/` for the current implementation. Bash phase scripts are no longer needed.

### Legacy Template (Historical Reference)

```bash
#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "60s"

# Your phase logic here
echo "Running phase checks..."

# Add requirement evidence
testing::phase::add_requirement "REQ-ID" "passed" "Test passed successfully"

# End with summary
testing::phase::end_with_summary "Phase completed"
```

### Phase Helpers API

```bash
# Initialize phase context
testing::phase::init --target-time "60s" [--require-runtime]

# Add requirement result
testing::phase::add_requirement <req_id> <status> <evidence>

# Add error (doesn't fail phase immediately)
testing::phase::add_error "Error message"

# Add warning
testing::phase::add_warning "Warning message"

# Run BAS automation
testing::phase::run_workflow_yaml <workflow_file> [--timeout 30]

# End phase and write results
testing::phase::end_with_summary "Summary message"
```

---

## Environment Variables

### Standard Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ROOT` | Vrooli project root | Auto-detected |
| `VROOLI_ROOT` | Alternative root | Same as APP_ROOT |
| `TEST_VERBOSE` | Enable verbose output | `0` |
| `TEST_TIMEOUT` | Override phase timeout | Phase default |

### Phase Variables

| Variable | Description | Set By |
|----------|-------------|--------|
| `TESTING_PHASE_NAME` | Current phase name | phase-helpers.sh |
| `TESTING_PHASE_SCENARIO_NAME` | Target scenario | phase-helpers.sh |
| `TESTING_PHASE_SCENARIO_DIR` | Scenario directory | phase-helpers.sh |
| `TESTING_PHASE_COVERAGE_DIR` | Coverage output dir | phase-helpers.sh |

### CI/CD Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CI` | Running in CI environment | `true` |
| `SKIP_RESOURCE_TESTS` | Skip resource integration | `1` |
| `COVERAGE_WARN` | Coverage warning threshold | `70` |
| `COVERAGE_ERROR` | Coverage error threshold | `60` |

---

## Error Handling

### Exit Codes

| Code | Meaning | Action |
|------|---------|--------|
| 0 | Success | Phase passed |
| 1 | Test failure | Phase failed |
| 2 | Setup error | Phase skipped |
| 124 | Timeout | Phase timed out |

### Safe Cleanup

Always use guarded cleanup to prevent accidental deletion:

```bash
teardown() {
    # CRITICAL: Validate variables before cleanup
    if [ -n "${TEST_PREFIX:-}" ] && [[ "$TEST_PREFIX" == /tmp/* ]]; then
        rm -rf "${TEST_PREFIX}"* 2>/dev/null || true
    fi
}
```

See [Safety Guidelines](../safety/GUIDELINES.md) for complete safety patterns.

---

## Debugging Tests

### Verbose Mode

```bash
# Enable verbose output
export TEST_VERBOSE=1
./test/phases/test-unit.sh

# Or via bash
bash -x ./test/phases/test-unit.sh
```

### BATS Debugging

```bash
# Run specific test
bats test/cli/my-tests.bats --filter "my specific test"

# Show full output
bats test/cli/my-tests.bats --tap

# Print during test (bypasses capture)
@test "debug example" {
    echo "Debug: variable=$my_var" >&3
    run my-command
}
```

### Phase Debugging

```bash
# Check phase results
cat coverage/phase-results/unit.json | jq .

# View requirement evidence
cat coverage/phase-results/unit.json | jq '.requirements[] | select(.id == "MY-REQ")'
```

---

## See Also

- [Shell Libraries Reference](../reference/shell-libraries.md) - Complete function docs
- [CLI Testing Guide](../guides/cli-testing.md) - BATS best practices
- [Safety Guidelines](../safety/GUIDELINES.md) - Safe testing patterns
- [Phase Catalog](../reference/phase-catalog.md) - Phase specifications
