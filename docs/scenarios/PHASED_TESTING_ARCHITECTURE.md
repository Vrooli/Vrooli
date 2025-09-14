# Phased Testing Architecture for Vrooli Scenarios

## 🎯 Overview

This document outlines a comprehensive phased testing architecture for Vrooli scenarios, designed to provide fast feedback, comprehensive validation, and seamless integration with existing testing patterns.

**Key Principles:**
- **Progressive Validation**: Time-bounded phases from 15s to 180s
- **Test Type Integration**: Go, BATS, UI automation, and shell scripts
- **Fast Feedback**: Developers get structure validation in 15 seconds
- **Comprehensive Coverage**: Full business logic and UI validation
- **Backward Compatibility**: Works with existing test patterns

## 🏗️ Complete Testing Structure

### Recommended Scenario Directory Structure
```bash
scenario/
├── test/
│   ├── run-tests.sh           # Phase orchestrator with test type support
│   ├── phases/               # Phased testing approach
│   │   ├── test-structure.sh   # <15s - Files, config validation
│   │   ├── test-dependencies.sh # <30s - Resource health checks
│   │   ├── test-unit.sh       # <60s - Unit test orchestrator
│   │   ├── test-integration.sh # <120s - Integration test orchestrator
│   │   ├── test-business.sh   # <180s - Business logic validation
│   │   └── test-performance.sh # <60s - Performance baselines
│   ├── unit/                 # Unit test specifications per language
│   │   ├── go.sh             # Runs Go unit tests (*_test.go)
│   │   ├── node.sh           # Runs Node.js/Jest tests (if UI has tests)
│   │   └── python.sh         # Runs Python tests (if applicable)
│   ├── cli/                  # CLI-specific testing
│   │   ├── *.bats            # BATS tests for CLI functionality
│   │   └── run-cli-tests.sh  # CLI test orchestrator
│   ├── ui/                   # UI testing workflows
│   │   ├── workflows/        # Browser automation workflows
│   │   │   ├── smoke-test.json          # Quick UI smoke test
│   │   │   ├── user-journey.json        # Critical user paths
│   │   │   └── regression-test.json     # Visual regression tests
│   │   ├── fixtures/         # Test data for UI tests
│   │   └── run-ui-tests.sh   # UI test orchestrator
│   ├── fixtures/             # Shared test data and mocks
│   └── utils/                # Shared test utilities
├── api/
│   ├── main.go
│   ├── main_test.go          # Go unit tests (standard Go convention)
│   └── services/
│       └── *_test.go         # Service unit tests
├── cli/
│   ├── my-cli
│   ├── my-cli.bats           # CLI BATS tests (current pattern)
│   └── install.sh
└── custom-tests.sh           # Legacy compatibility
```

## 📋 Phase Definitions

### Phase 1: Structure (test-structure.sh) - `<15 seconds`
**Purpose**: Fast validation of required files and configuration
```bash
✅ service.json exists and validates against schema
✅ Required directories exist (initialization/, api/, cli/)
✅ README.md contains business context
✅ Makefile has standard targets (if present)
✅ All referenced files in service.json exist
✅ Configuration files are syntactically valid
```

### Phase 2: Dependencies (test-dependencies.sh) - `<30 seconds`  
**Purpose**: Resource availability without deep testing
```bash
✅ All required resources are running (health check)
✅ Resource endpoints respond with basic connectivity
✅ Optional resources are catalogued (warn if missing)
✅ Service discovery resolves all URLs correctly
✅ Environment variables are available
```

### Phase 3: Unit (test-unit.sh) - `<60 seconds`
**Purpose**: Language-specific unit test orchestration
```bash
✅ Go tests: Run all *_test.go files in api/ directory tree
✅ Node tests: Run npm test/jest (if UI has unit tests)
✅ Python tests: Run pytest (if applicable)
✅ Each language has its own test runner in test/unit/
✅ Parallel execution where possible
```

### Phase 4: Integration (test-integration.sh) - `<120 seconds`
**Purpose**: Service integration and API testing
```bash
✅ API endpoint testing with real dependencies
✅ Database operations and transactions
✅ Resource integrations (Ollama, N8n, etc.)
✅ CLI integration tests (BATS) via test/cli/run-cli-tests.sh
✅ Inter-service communication validation
```

### Phase 5: Business (test-business.sh) - `<180 seconds`
**Purpose**: Scenario-specific business logic and UI workflows
```bash
✅ Custom business workflow validation
✅ UI testing via browser-automation-studio workflows
✅ End-to-end user journey testing
✅ Domain-specific validation
✅ Revenue-generating feature validation
```

### Phase 6: Performance (test-performance.sh) - `<60 seconds`
**Purpose**: Performance and load validation
```bash
✅ Response time baselines
✅ Concurrent request handling
✅ Resource utilization monitoring
✅ Memory leak detection
```

## 🔧 Command Integration

### Enhanced `vrooli scenario` Commands

#### Test Execution Commands
```bash
# Primary test command structure
vrooli scenario test <name> [phase/type] [options]

# Phase-specific testing
vrooli scenario test my-scenario structure      # 15s - Fast structure validation
vrooli scenario test my-scenario dependencies   # 30s - Resource connectivity
vrooli scenario test my-scenario unit           # 60s - All unit tests
vrooli scenario test my-scenario integration    # 120s - Integration + CLI tests
vrooli scenario test my-scenario business       # 180s - Business + UI tests
vrooli scenario test my-scenario performance    # 60s - Performance validation

# Test type-specific testing
vrooli scenario test my-scenario go             # Go unit tests only
vrooli scenario test my-scenario bats           # CLI BATS tests only
vrooli scenario test my-scenario ui             # UI automation only

# Combined testing
vrooli scenario test my-scenario unit integration    # Multiple phases
vrooli scenario test my-scenario go bats            # Multiple test types
vrooli scenario test my-scenario phases 1-4         # Sequential phases
vrooli scenario test my-scenario all                # Full test suite

# Quick feedback modes
vrooli scenario test my-scenario quick              # Structure + dependencies + unit
vrooli scenario test my-scenario smoke              # Structure + dependencies only
```

#### Status Command with Automatic Test Validation
```bash
# Enhanced status command with automatic test validation
vrooli scenario status <name> [options]

# Default behavior includes:
✅ Health check schema validation (existing)
✅ Automatic structure phase validation (new)
✅ Automatic dependency phase validation (new)
✅ Test infrastructure completeness check (new)

# Enhanced options
vrooli scenario status my-scenario --validate-tests unit      # Include unit test validation
vrooli scenario status my-scenario --validate-tests all       # Include all test validation
vrooli scenario status my-scenario --no-test-validation       # Skip test validation
vrooli scenario status my-scenario --verbose                  # Detailed test status
```

### Test Orchestration Options
```bash
# Global options for all test commands
--verbose      # Show detailed output for all test phases
--parallel     # Run tests in parallel where possible
--timeout N    # Set timeout in seconds (default varies by phase)
--dry-run      # Show what tests would be run without executing
--continue     # Continue testing even if a phase fails
--junit        # Output results in JUnit XML format
--coverage     # Generate coverage reports where applicable
```

## 📁 Implementation Details

### Language-Specific Unit Test Runners

#### test/unit/go.sh
```bash
#!/bin/bash
# Run all Go unit tests in the scenario
set -euo pipefail

echo "Running Go unit tests..."
if [ -d "api/" ]; then
    cd api/ && go test -v ./... -timeout 30s -cover
    echo "✅ Go unit tests completed"
else
    echo "ℹ️  No Go code found, skipping Go tests"
fi
```

#### test/unit/node.sh
```bash
#!/bin/bash
# Run Node.js/Jest tests if UI has them
set -euo pipefail

echo "Running Node.js unit tests..."
if [ -f "ui/package.json" ] && grep -q '"test":' ui/package.json; then
    cd ui/ && npm test --passWithNoTests --coverage
    echo "✅ Node.js unit tests completed"
else
    echo "ℹ️  No Node.js tests configured, skipping"
fi
```

#### test/unit/python.sh
```bash
#!/bin/bash
# Run Python tests if any Python components exist
set -euo pipefail

echo "Running Python unit tests..."
if [ -f "requirements.txt" ] || [ -f "pyproject.toml" ]; then
    python -m pytest tests/ -v --timeout=30 --cov
    echo "✅ Python unit tests completed"
else
    echo "ℹ️  No Python tests found, skipping"
fi
```

### CLI Test Integration

#### test/cli/run-cli-tests.sh
```bash
#!/bin/bash
# Orchestrate all CLI BATS tests
set -euo pipefail

echo "Running CLI BATS tests..."
test_count=0
failed_count=0

for bats_file in cli/*.bats; do
    if [ -f "$bats_file" ]; then
        echo "Running $(basename "$bats_file")..."
        if bats "$bats_file" --tap; then
            ((test_count++))
        else
            ((failed_count++))
            ((test_count++))
        fi
    fi
done

if [ $failed_count -eq 0 ]; then
    echo "✅ All $test_count CLI test suites passed"
else
    echo "❌ $failed_count of $test_count CLI test suites failed"
    exit 1
fi
```

### UI Test Integration

#### test/ui/run-ui-tests.sh  
```bash
#!/bin/bash
# Execute browser automation workflows
set -euo pipefail

echo "Running UI automation tests..."
BROWSER_AUTOMATION_CLI="${VROOLI_ROOT}/scenarios/browser-automation-studio/cli/browser-automation-studio"

if [ ! -x "$BROWSER_AUTOMATION_CLI" ]; then
    echo "⚠️  Browser automation studio not available, skipping UI tests"
    exit 0
fi

workflow_count=0
failed_count=0

for workflow in test/ui/workflows/*.json; do
    if [ -f "$workflow" ]; then
        echo "Executing $(basename "$workflow")..."
        if "$BROWSER_AUTOMATION_CLI" execute "$workflow"; then
            ((workflow_count++))
        else
            ((failed_count++))
            ((workflow_count++))
        fi
    fi
done

if [ $failed_count -eq 0 ]; then
    echo "✅ All $workflow_count UI workflows passed"
else
    echo "❌ $failed_count of $workflow_count UI workflows failed"
    exit 1
fi
```

### Phase Test Examples

#### test/phases/test-structure.sh
```bash
#!/bin/bash
# Structure validation phase - <15 seconds
set -euo pipefail

echo "=== Structure Phase (Target: <15s) ==="
start_time=$(date +%s)

# Check required files
required_files=(
    ".vrooli/service.json"
    "README.md"
    "scenario-test.yaml"
)

missing_files=()
for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        missing_files+=("$file")
    fi
done

if [ ${#missing_files[@]} -gt 0 ]; then
    echo "❌ Missing required files:"
    printf "   - %s\n" "${missing_files[@]}"
    exit 1
fi

# Validate service.json schema
if command -v jq >/dev/null 2>&1; then
    if ! jq empty < .vrooli/service.json >/dev/null 2>&1; then
        echo "❌ Invalid JSON in service.json"
        exit 1
    fi
    echo "✅ service.json is valid JSON"
fi

# Check directory structure
required_dirs=("initialization" "api" "cli")
for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo "✅ Directory $dir exists"
    else
        echo "ℹ️  Directory $dir missing (may be optional)"
    fi
done

end_time=$(date +%s)
duration=$((end_time - start_time))
echo "✅ Structure validation completed in ${duration}s"

if [ $duration -gt 15 ]; then
    echo "⚠️  Structure phase exceeded 15s target"
fi
```

#### test/phases/test-dependencies.sh
```bash
#!/bin/bash
# Dependencies validation phase - <30 seconds  
set -euo pipefail

echo "=== Dependencies Phase (Target: <30s) ==="
start_time=$(date +%s)

# Source common utilities
FRAMEWORK_DIR="${VROOLI_ROOT}/scripts/scenarios/validation"
if [ -f "$FRAMEWORK_DIR/clients/common.sh" ]; then
    source "$FRAMEWORK_DIR/clients/common.sh"
else
    echo "⚠️  Validation framework not available, using basic checks"
fi

# Check required resources from service.json
if command -v jq >/dev/null 2>&1; then
    required_resources=$(jq -r '.spec.dependencies.resources[]? | select(.optional != true) | .name' .vrooli/service.json)
    
    for resource in $required_resources; do
        if command -v get_resource_url >/dev/null 2>&1; then
            resource_url=$(get_resource_url "$resource")
            if [ -n "$resource_url" ] && check_url_health "$resource_url"; then
                echo "✅ Resource $resource is available"
            else
                echo "❌ Resource $resource is not available"
                exit 1
            fi
        else
            echo "ℹ️  Resource $resource (basic check - validation framework unavailable)"
        fi
    done
fi

end_time=$(date +%s)
duration=$((end_time - start_time))
echo "✅ Dependencies validation completed in ${duration}s"

if [ $duration -gt 30 ]; then
    echo "⚠️  Dependencies phase exceeded 30s target"
fi
```

## 🚀 Integration with Health Check System

### Enhanced `vrooli scenario status` Behavior

**Default Status Check (No Options):**
```bash
vrooli scenario status my-scenario

# Output includes:
📊 Scenario Status: my-scenario
├── Health Check Schema: ✅ PASSED
├── Structure Validation: ✅ PASSED (12s)
├── Dependencies Check: ✅ PASSED (8s) 
├── Test Infrastructure: ✅ COMPLETE
│   ├── Unit Tests: ✅ Go tests available
│   ├── CLI Tests: ✅ BATS tests available  
│   ├── UI Tests: ⚠️  No UI workflows found
│   └── Business Tests: ✅ Custom tests available
└── Overall Status: 🟢 HEALTHY
```

**Detailed Test Validation:**
```bash
vrooli scenario status my-scenario --validate-tests all

# Output includes all above plus:
🧪 Extended Test Validation:
├── Unit Tests: ✅ PASSED (23s) - 47 tests, 2 skipped
├── Integration Tests: ✅ PASSED (89s) - 12 endpoints tested
├── CLI Tests: ✅ PASSED (15s) - 23 BATS assertions
├── Business Tests: ✅ PASSED (156s) - Custom logic validated
└── Performance Tests: ⚠️  SKIPPED - No performance tests configured
```

## 📋 Migration Strategy

### Phase 1: Infrastructure (Week 1-2)
1. **Create phase framework** 
   - Build standardized phase templates
   - Create test/unit/ runners for Go, Node, Python
   - Enhance test/cli/ orchestration for BATS tests
   - Design test/ui/ structure for browser automation

2. **Command Integration**
   - Update `vrooli scenario test` command structure
   - Enhance `vrooli scenario status` with test validation
   - Add phase and test type support

3. **Template Creation**
   - Create scenario test templates
   - Build phase-specific script templates
   - Design UI workflow templates

### Phase 2: Pilot Migration (Week 3-4)
1. **Select Pilot Scenarios** (5-10 scenarios)
   - `ai-chatbot-manager` - Has comprehensive testing already
   - `calendar` - Has Go unit tests
   - `browser-automation-studio` - UI testing capabilities
   - `brand-manager` - Complex integration testing
   - `simple-test` - Basic validation patterns

2. **Validate Integration**
   - Test Go unit test integration with existing *_test.go files
   - Validate BATS integration with existing .bats files  
   - Create sample UI workflows for scenarios with web interfaces
   - Ensure backward compatibility with custom-tests.sh

3. **Developer Feedback**
   - Collect feedback on command ergonomics
   - Refine phase time boundaries
   - Adjust test type categorization

### Phase 3: Production Rollout (Week 5-6)
1. **Template Deployment**
   - New scenarios get full test structure by default
   - `vrooli scenario create` includes test templates
   - Documentation and examples available

2. **CI/CD Integration**
   - Phase-aware testing in build pipelines
   - Parallel test execution where possible
   - Performance regression detection

3. **Enforcement**
   - Block deployment without adequate phase coverage
   - Require passing structure and dependency phases
   - Optional business phase validation for revenue scenarios

## 🎯 Expected Benefits

### Immediate (1-2 months)
- **15-second feedback** on basic scenario validity
- **Targeted debugging** - test just the failing component
- **Progressive validation** - developers build confidence incrementally
- **Unified command structure** - consistent testing interface

### Medium-term (3-6 months)
- **Zero deployment failures** due to missing dependencies or broken tests
- **Automated quality gates** - phases must pass for deployment
- **Performance regression prevention** - automated performance validation
- **Cross-scenario UI testing** via browser-automation-studio

### Long-term (6+ months)
- **Production-grade scenario quality** - comprehensive business logic validation
- **Revenue protection** - business phases prevent revenue-impacting bugs
- **Developer velocity** - fast feedback enables rapid iteration
- **Self-healing UI tests** - AI-assisted workflow repair through browser-automation-studio

## 🔍 Implementation Notes

### Backward Compatibility
- Existing `custom-tests.sh` files continue to work
- Current Go `*_test.go` files run in unit phase
- Existing BATS `.bats` files run in integration phase
- Current `scenario-test.yaml` declarative configs remain supported

### Performance Considerations
- Phase time boundaries are targets, not hard limits
- Tests run in parallel where possible (unit tests, independent phases)
- Results cached to avoid unnecessary re-runs
- Early exit on critical phase failures (structure, dependencies)

### Error Handling
- Clear error messages with actionable guidance
- Graceful degradation when optional dependencies unavailable
- Comprehensive logging for debugging test failures
- Integration with existing Vrooli error reporting

### Extensibility
- Easy to add new test types (e.g., security, accessibility)
- Simple to extend phases (e.g., deployment validation)
- Pluggable test runners for different languages/frameworks
- Template system for scenario-specific customization

---

This architecture provides a solid foundation for comprehensive scenario testing while maintaining the flexibility and speed needed for effective development workflows.