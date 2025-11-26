# Centralized Testing Library

A comprehensive, modular testing framework for Vrooli scenarios that provides both **sourceable libraries** and **copy-and-customize templates**.

## 📁 Directory Structure

```
scripts/scenarios/testing/
├── shell/                    # Sourceable shell libraries
│   ├── core.sh              # Scenario detection, configuration
│   ├── runner.sh            # Low-level phase execution engine
│   ├── suite.sh             # High-level test orchestrator
│   ├── phase-helpers.sh     # Phase lifecycle management
│   ├── config-helpers.sh    # JSON configuration parsing utilities
│   └── requirements-sync.sh # Requirements registry synchronization
├── unit/                     # Language-specific unit test runners
│   ├── run-all.sh           # Universal test runner
│   ├── go.sh                # Go test runner
│   ├── node.sh              # Node.js test runner
│   └── python.sh            # Python test runner
├── templates/                # Copy-and-customize templates
│   ├── README.md            # Template usage guide
│   └── go/                  # Go testing templates
│       ├── test_helpers.go.template
│       └── error_patterns.go.template
├── legacy/                   # Backwards compatibility
│   └── run-scenario-tests.sh
└── README.md                # This file
```

> UI integration workflows should live alongside each scenario's tests (for example `scenarios/<name>/test/playbooks/<flow>.json`). Use the helpers in `scripts/scenarios/testing/playbooks/browser-automation-studio.sh` or call `testing::phase::run_bas_automation_validations` to import and execute those workflows through the BAS CLI.

## 🚀 Quick Start

### Using Shell Libraries

```bash
#!/bin/bash
# In your test script

# Source the modules you need
source "${APP_ROOT}/scripts/scenarios/testing/shell/orchestration.sh"

# Run comprehensive tests
testing::orchestration::run_comprehensive_tests

# Or use individual modules (see phase-specific scripts in shell/)
```

### Using Unit Test Runners

```bash
# Source the universal runner
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Run all language tests
testing::unit::run_all_tests --coverage-warn 80 --coverage-error 70
```

### Using Templates

```bash
# Copy Go templates to your scenario
cp scripts/scenarios/testing/templates/go/test_helpers.go.template \
   scenarios/my-scenario/api/test_helpers.go

# Customize for your needs (change package, add helpers, etc.)
```

## 📚 Shell Library Modules

### core.sh
Core utilities for all testing scenarios.

**Functions:**
- `testing::core::detect_scenario()` - Auto-detect current scenario
- `testing::core::get_scenario_config()` - Read service.json configuration
- `testing::core::detect_languages()` - Detect languages in use
- `testing::core::is_scenario_running()` - Check if scenario is active
- `testing::core::wait_for_scenario()` - Wait for scenario readiness

### runner.sh
Low-level phase execution engine.

**Functions:**
- `testing::runner::register_phase()` - Register test phases for execution
- `testing::runner::execute()` - Execute all registered phases in order

### suite.sh
High-level test suite orchestrator with automatic phase discovery.

**Functions:**
- `testing::suite::run()` - Main entry point for running complete test suites

### phase-helpers.sh
Phase lifecycle management utilities.

**Functions:**
- `testing::phase::init()` - Initialize phase environment
- `testing::phase::add_test()` - Record test result
- `testing::phase::add_error()` - Record error
- `testing::phase::add_requirement()` - Record requirement coverage

### config-helpers.sh
JSON configuration parsing utilities.

**Functions:**
- `config::get_boolean()` - Parse boolean values with defaults
- `config::get_string()` - Parse string values with defaults
- `config::get_number()` - Parse numeric values with defaults
- `config::get_array()` - Extract arrays from JSON

### requirements-sync.sh
Requirements registry synchronization.

**Functions:**
- `testing::requirements::sync()` - Sync test results with requirements registry

### resources.sh
Resource integration testing.

**Functions:**
- `testing::resources::test_postgres()` - Test PostgreSQL integration
- `testing::resources::test_redis()` - Test Redis integration
- `testing::resources::test_ollama()` - Test Ollama integration
- `testing::resources::test_n8n()` - Test N8n integration
- `testing::resources::test_qdrant()` - Test Qdrant integration
- `testing::resources::test_all()` - Test all configured resources

### cli.sh
CLI testing utilities.

**Functions:**
- `testing::cli::test_integration()` - Test CLI binary functionality
- `testing::cli::test_command()` - Test specific CLI command
- `testing::cli::test_with_input()` - Test CLI with input/output

### orchestration.sh
Comprehensive test suite execution.

**Functions:**
- `testing::orchestration::run_unit_tests()` - Run all unit tests
- `testing::orchestration::run_integration_tests()` - Run integration tests only
- `testing::orchestration::run_comprehensive_tests()` - Run full test suite

## 🧪 Unit Test Runners

### Universal Runner
```bash
testing::unit::run_all_tests [options]
```

**Options:**
- `--go-dir PATH` - Go code directory (default: api)
- `--node-dir PATH` - Node.js code directory (default: ui)
- `--python-dir PATH` - Python code directory (default: .)
- `--skip-go` - Skip Go tests
- `--skip-node` - Skip Node.js tests
- `--skip-python` - Skip Python tests
- `--verbose` - Verbose output
- `--fail-fast` - Stop on first failure
- `--coverage-warn PERCENT` - Warning threshold (default: 80)
- `--coverage-error PERCENT` - Error threshold (default: 70)

### Language-Specific Runners

Each language has its own runner with specific options:

```bash
# Go tests
testing::unit::run_go_tests --dir api --coverage-warn 80

# Node.js tests
testing::unit::run_node_tests --dir ui --timeout 60000

# Python tests
testing::unit::run_python_tests --dir . --framework pytest
```

## 📝 Templates

Templates are **starting points** for your test helpers. They're meant to be copied and customized, not imported.

### Available Templates

- **go/test_helpers.go.template** - HTTP testing utilities for Go
- **go/error_patterns.go.template** - Sophisticated error testing patterns

### Template Usage

1. **Copy** the template to your scenario
2. **Customize** package names and imports
3. **Adapt** functions to your needs
4. **Delete** unused code
5. **Add** scenario-specific helpers

See [templates/README.md](templates/README.md) for detailed usage instructions.

## 💡 Usage Examples

### Example 1: Phased Testing

```bash
#!/bin/bash
# test/phases/test-integration.sh

# Use phase helpers for lifecycle management
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::auto_lifecycle_start \
    --phase-name "integration" \
    --default-target-time "300s"

# Your test logic here
testing::phase::add_test passed

testing::phase::auto_lifecycle_end "Integration tests completed"
```

### Example 2: Comprehensive Testing

```bash
#!/bin/bash
# test/run-tests.sh

# Use suite orchestrator for complete test runs
source "${APP_ROOT}/scripts/scenarios/testing/shell/suite.sh"

# Automatically discovers phases and runs with requirements sync
testing::suite::run
```

### Example 3: Resource Testing

```bash
#!/bin/bash
# test/test-resources.sh

source "${APP_ROOT}/scripts/scenarios/testing/shell/resources.sh"

# Test specific resources
testing::resources::test_postgres "my-scenario"
testing::resources::test_redis "my-scenario"
```

## 🔄 Migration Guide

### From Old Test Scripts

```bash
# Old way - Manual test management
cd api && go test ./...
cd ui && pnpm test

# New way - Use phased testing framework
source "$APP_ROOT/scripts/scenarios/testing/shell/suite.sh"
testing::suite::run
```

### From Hardcoded Configuration

```bash
# Old way - Hardcoded values
TIMEOUT=120
COVERAGE_THRESHOLD=80

# New way - Use .vrooli/testing.json
# Configuration is automatically loaded by suite.sh
# See docs/testing/guides/requirement-tracking-quick-start.md
```

## 🔒 Safety First

**CRITICAL:** Before using any testing templates or writing test scripts, review our [Safety Guidelines](/home/matthalloran8/Vrooli/docs/testing/safety/GUIDELINES.md) to prevent data loss.

### Quick Safety Checklist
- [ ] BATS `teardown()` functions validate variables before `rm` commands
- [ ] Test files are isolated under `/tmp` directories  
- [ ] Wildcard patterns (`*`) are never used with empty variables
- [ ] Use our safe BATS template: `templates/bats/cli-test.bats.template`

### Safety Linter
Run the safety linter on your test scripts:

```bash
# Lint all test scripts in current directory
scripts/scenarios/testing/lint-tests.sh

# Lint specific scenario
scripts/scenarios/testing/lint-tests.sh scenarios/my-scenario/
```

## ✅ Best Practices

### 1. Use Appropriate Level of Abstraction

Choose the right tool for your needs:

```bash
# For complete test suites with automatic discovery
source "${APP_ROOT}/scripts/scenarios/testing/shell/suite.sh"
testing::suite::run

# For custom phase execution order
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"
testing::runner::register_phase "my-custom-phase" "test/my-phase.sh"
testing::runner::execute

# For individual phase scripts
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
testing::phase::auto_lifecycle_start --phase-name "my-phase"
```

### 2. Leverage Auto-Detection

Let the framework detect your scenario:

```bash
# Auto-detects from current directory
testing::connectivity::test_api

# Or specify explicitly
testing::connectivity::test_api "my-scenario"
```

### 3. Consistent Coverage Standards

Use standard thresholds across scenarios:

```bash
# Standard: 80% warning, 70% error
testing::orchestration::run_unit_tests "" 80 70
```

### 4. Templates as Starting Points

Don't try to make templates work for everything. Copy and customize:

```bash
# Copy template
cp templates/go/test_helpers.go.template api/test_helpers.go

# Make it yours
# - Change package name
# - Add your helpers
# - Remove what you don't need
```

## 🤝 Contributing

### Adding New Shell Functions

1. Add to appropriate module in `shell/`
2. Export the function
3. Document in this README
4. Add tests

### Adding New Templates

1. Create well-documented template in `templates/<language>/`
2. Update `templates/README.md`
3. Provide real usage example

### Improving Unit Runners

1. Update language-specific runner in `unit/`
2. Ensure compatibility with `run-all.sh`
3. Test with multiple scenarios

## 📚 See Also

- [templates/README.md](templates/README.md) - Template usage guide
- [docs/testing/architecture/PHASED_TESTING.md](/home/matthalloran8/Vrooli/docs/testing/architecture/PHASED_TESTING.md) - Phased testing documentation
- [scenarios/visited-tracker/](../../scenarios/visited-tracker/) - Example implementation

## 🎯 Philosophy

This testing framework follows these principles:

1. **Modular** - Use only what you need
2. **Discoverable** - Auto-detect scenarios and languages
3. **Flexible** - Templates to copy, not rigid structures
4. **Consistent** - Same patterns across all scenarios
5. **Practical** - Based on real-world usage

The goal is to make testing **easier**, not harder. If something doesn't help, don't use it!
