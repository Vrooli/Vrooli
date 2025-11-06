# Test Runners Reference

This document provides a complete reference for available test runners in Vrooli.

## Universal Test Runner

### Location
`/scripts/scenarios/testing/unit/run-all.sh`

### Usage
```bash
source "$APP_ROOT/scripts/scenarios/testing/unit/run-all.sh"
testing::unit::run_all_tests [options]
```

### Options
| Option | Description | Default |
|--------|-------------|---------|
| `--go-dir PATH` | Go code directory | `api` |
| `--node-dir PATH` | Node.js code directory | `ui` |
| `--python-dir PATH` | Python code directory | `.` |
| `--skip-go` | Skip Go tests | false |
| `--skip-node` | Skip Node.js tests | false |
| `--skip-python` | Skip Python tests | false |
| `--verbose` | Verbose output | false |
| `--fail-fast` | Stop on first failure | false |
| `--coverage-warn PERCENT` | Warning threshold | 80 |
| `--coverage-error PERCENT` | Error threshold | 70 |

### Example
```bash
testing::unit::run_all_tests \
    --go-dir api \
    --node-dir ui \
    --coverage-warn 80 \
    --coverage-error 70 \
    --verbose
```

## Language-Specific Runners

### Go Test Runner

#### Location
`/scripts/scenarios/testing/unit/go.sh`

#### Usage
```bash
source "$APP_ROOT/scripts/scenarios/testing/unit/go.sh"
testing::unit::run_go_tests [options]
```

#### Options
- `--dir PATH` - Directory containing Go code (default: `api`)
- `--timeout SECONDS` - Test timeout (default: 60s)
- `--coverage-warn PERCENT` - Warning threshold (default: 80)
- `--coverage-error PERCENT` - Error threshold (default: 70)
- `--verbose` - Verbose test output
- `--race` - Enable race detection

#### Features
- Automatic coverage calculation
- Race condition detection
- Parallel test execution
- Coverage HTML report generation

#### Example
```bash
testing::unit::run_go_tests \
    --dir api \
    --timeout 120 \
    --coverage-warn 80 \
    --race
```

### Node.js Test Runner

#### Location
`/scripts/scenarios/testing/unit/node.sh`

#### Usage
```bash
source "$APP_ROOT/scripts/scenarios/testing/unit/node.sh"
testing::unit::run_node_tests [options]
```

#### Options
- `--dir PATH` - Directory containing Node.js code (default: `ui`)
- `--timeout MS` - Test timeout in milliseconds (default: 60000)
- `--framework FRAMEWORK` - Test framework: jest|vitest|mocha (auto-detected)
- `--coverage` - Enable coverage reporting
- `--watch` - Run in watch mode

#### Features
- Auto-detects test framework (Jest, Vitest, Mocha)
- Coverage reporting with thresholds
- Watch mode for development
- Parallel test execution

#### Example
```bash
testing::unit::run_node_tests \
    --dir ui \
    --framework jest \
    --coverage \
    --timeout 90000
```

### Python Test Runner

#### Location
`/scripts/scenarios/testing/unit/python.sh`

#### Usage
```bash
source "$APP_ROOT/scripts/scenarios/testing/unit/python.sh"
testing::unit::run_python_tests [options]
```

#### Options
- `--dir PATH` - Directory containing Python code (default: `.`)
- `--framework FRAMEWORK` - Test framework: pytest|unittest (auto-detected)
- `--coverage` - Enable coverage reporting
- `--verbose` - Verbose test output
- `--markers MARKERS` - Pytest markers to run

#### Features
- Auto-detects test framework (pytest, unittest)
- Coverage reporting with pytest-cov
- Marker-based test selection
- Parallel execution with pytest-xdist

#### Example
```bash
testing::unit::run_python_tests \
    --dir src \
    --framework pytest \
    --coverage \
    --markers "not slow"
```

## Phase-Based Test Runners

### Structure Test Runner
Tests project structure and required files.

```bash
./test/phases/test-structure.sh
```

**Checks:**
- Required files (README.md, PRD.md, service.json)
- Directory structure
- Configuration validity
- Deprecated file warnings

**Time Limit:** 15 seconds

### Dependency Test Runner
Verifies all dependencies are available.

```bash
./test/phases/test-dependencies.sh
```

**Checks:**
- Language runtime versions
- Package dependencies
- Resource availability
- Security vulnerabilities

**Time Limit:** 30 seconds

### Unit Test Runner
Executes unit tests for all languages.

```bash
./test/phases/test-unit.sh
```

**Executes:**
- Go tests with coverage
- Node.js tests with Jest/Vitest
- Python tests with pytest
- Coverage threshold validation

**Time Limit:** 60 seconds

### Integration Test Runner
Tests component interactions.

```bash
./test/phases/test-integration.sh
```

**Tests:**
- API endpoint connectivity
- UI availability
- CLI functionality
- Resource integrations

**Time Limit:** 120 seconds

### Business Logic Test Runner
Validates business requirements.

```bash
./test/phases/test-business.sh
```

**Validates:**
- Core workflows
- User journeys
- Business rules
- Data integrity

**Time Limit:** 180 seconds

### Performance Test Runner
Measures performance baselines.

```bash
./test/phases/test-performance.sh
```

**Measures:**
- Response times
- Throughput
- Resource usage
- Scalability limits

**Time Limit:** 60 seconds

## Comprehensive Test Runner

### Makefile Integration
The recommended way to run all tests:

```bash
make test
```

**Makefile Target:**
```makefile
test:
	@echo "Running comprehensive tests..."
	@./test/run-tests.sh
```

### Direct Execution
```bash
./test/run-tests.sh [options]
```

**Options:**
- `--phase PHASE` - Run specific phase only
- `--skip-phase PHASE` - Skip specific phase
- `--verbose` - Verbose output
- `--fail-fast` - Stop on first failure
- `--report` - Generate HTML report

### Example Output
```
=== Running Comprehensive Tests for my-scenario ===

[Phase 1/6] Structure Validation...
✅ All required files present (3.2s)

[Phase 2/6] Dependency Checks...
✅ All dependencies available (12.5s)

[Phase 3/6] Unit Tests...
Go: 42 tests, 79.4% coverage ✅
Node: 18 tests, 82.1% coverage ✅
✅ Unit tests passed (45.3s)

[Phase 4/6] Integration Tests...
API: ✅ Healthy
UI: ✅ Accessible
CLI: ✅ Functional
✅ Integration tests passed (89.7s)

[Phase 5/6] Business Logic Tests...
✅ Core workflows validated (132.1s)

[Phase 6/6] Performance Tests...
✅ Performance within thresholds (41.8s)

=== All Tests Passed! ===
Total time: 5m 24s
Coverage: 80.2%
```

## CLI Test Runners

### BATS Test Runner
For CLI testing with BATS framework.

```bash
bats test/cli/*.bats [options]
```

**Options:**
- `--tap` - TAP output format
- `--filter PATTERN` - Run tests matching pattern
- `--jobs N` - Parallel execution
- `-v` - Verbose output

**Example:**
```bash
# Run all CLI tests
bats test/cli/*.bats

# Run with TAP output for CI
bats --tap test/cli/*.bats

# Run specific test
bats test/cli/my-cli.bats --filter "shows version"
```

## CI/CD Integration

### GitHub Actions
```yaml
- name: Run Tests
  run: |
    source scripts/scenarios/testing/unit/run-all.sh
    testing::unit::run_all_tests --fail-fast
```

### Jenkins
```groovy
stage('Test') {
    sh '''
        source scripts/scenarios/testing/unit/run-all.sh
        testing::unit::run_all_tests --coverage-warn 80
    '''
}
```

### GitLab CI
```yaml
test:
  script:
    - source scripts/scenarios/testing/unit/run-all.sh
    - testing::unit::run_all_tests --verbose
```

## Custom Test Runners

### Creating a Custom Runner
```bash
#!/bin/bash
# custom-test-runner.sh

set -euo pipefail

# Source libraries
source "$APP_ROOT/scripts/scenarios/testing/shell/core.sh"
source "$APP_ROOT/scripts/scenarios/testing/unit/run-all.sh"

# Custom logic
echo "Running custom tests..."

# Detect scenario and languages
scenario=$(testing::core::detect_scenario)
languages=$(testing::core::detect_languages)

# Run specific tests
if [[ "$scenario" == "special-case" ]]; then
    # Special handling
    ./test/special/run-special.sh
else
    # Standard unit tests with custom thresholds
    runner_args=("--coverage-warn" "80" "--coverage-error" "70")
    for lang in go node python; do
        if ! echo "$languages" | grep -q "$lang"; then
            runner_args+=("--skip-$lang")
        fi
    done
    testing::unit::run_all_tests "${runner_args[@]}"
fi

# Custom validation
if [ -f "custom-requirements.txt" ]; then
    echo "Validating custom requirements..."
    # Custom validation logic
fi

echo "Custom tests complete!"
```

## Environment Variables

Test runners respect these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_VERBOSE` | Enable verbose output | 0 |
| `TEST_TIMEOUT` | Global timeout override | varies |
| `COVERAGE_WARN` | Warning threshold | 80 |
| `COVERAGE_ERROR` | Error threshold | 70 |
| `SKIP_UNIT_TESTS` | Skip unit test phase | false |
| `SKIP_INTEGRATION_TESTS` | Skip integration phase | false |
| `CI` | CI environment flag | false |

## See Also

- [Shell Libraries Reference](shell-libraries.md)
- [Testing Architecture](../architecture/PHASED_TESTING.md)
- [Quick Start Guide](../guides/quick-start.md)