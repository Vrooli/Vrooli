# Vrooli Scenario Testing Guide

This guide explains how to properly test Vrooli scenarios using the modern phased testing architecture.

## Overview

Vrooli uses a comprehensive **phased testing approach** that replaces the legacy `scenario-test.yaml` system with a modern, extensible framework configured through `.vrooli/service.json`.

> **Note**: The `scenario-test.yaml` format is **deprecated**. All new scenarios should use the modern `.vrooli/service.json` configuration with phased testing.

## Modern Testing Architecture

### Configuration via service.json

All testing is configured in `.vrooli/service.json`:

```json
{
  "test": {
    "description": "Comprehensive phased testing",
    "steps": [{
      "name": "run-comprehensive-tests",
      "run": "test/run-tests.sh",
      "description": "Execute comprehensive phased testing"
    }]
  }
}
```

### Testing Phases

The modern architecture uses 6 progressive phases:

1. **Structure Validation** (15s) - Validates files and configuration
2. **Dependencies Check** (30s) - Verifies all dependencies available
3. **Unit Tests** (60s) - Language-specific unit testing
4. **Integration Tests** (120s) - Component interaction testing
5. **Business Logic** (180s) - End-to-end workflow validation
6. **Performance** (60s) - Benchmarks and resource usage

## Running Tests

### Recommended Method: Makefile

```bash
# From scenario directory
cd scenarios/<scenario-name>
make test     # Run all tests
make logs     # View test logs
```

### Alternative: Vrooli CLI

```bash
# Test a specific scenario
vrooli scenario test <scenario-name>

# Or from scenario directory  
cd scenarios/<scenario-name>
make test
```

### Direct Execution

```bash
# Run test orchestrator directly
./test/run-tests.sh

# Run specific phase
./test/phases/test-unit.sh
```

## Test Directory Structure

```
scenario/
├── .vrooli/
│   └── service.json           # Modern configuration
├── test/
│   ├── run-tests.sh          # Main test orchestrator
│   ├── phases/               # Phased test scripts
│   │   ├── test-structure.sh
│   │   ├── test-dependencies.sh
│   │   ├── test-unit.sh
│   │   ├── test-integration.sh
│   │   ├── test-business.sh
│   │   └── test-performance.sh
│   └── cli/                  # CLI BATS tests
│       └── my-cli.bats
├── api/
│   └── *_test.go             # Go unit tests
└── ui/
    └── *.test.js             # JavaScript tests
```

## Writing Tests

### Complete Working Example: Main Test Orchestrator

**File: `test/run-tests.sh` (complete orchestrator implementation)**
```bash
#!/bin/bash
# Main test orchestrator for comprehensive phased testing
# This script runs all testing phases in sequence with proper error handling

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly APP_ROOT="${APP_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
readonly SCENARIO_NAME="$(basename "$(cd "$SCRIPT_DIR/.." && pwd)")"
readonly LOG_FILE="/tmp/test-${SCENARIO_NAME}-$$.log"

# Test phases configuration
readonly PHASES=(
    "structure:15:Structure Validation"
    "dependencies:30:Dependencies Check" 
    "unit:60:Unit Tests"
    "integration:120:Integration Tests"
    "business:180:Business Logic Tests"
    "performance:60:Performance Tests"
)

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Test results tracking
TOTAL_PHASES=${#PHASES[@]}
PASSED_PHASES=0
FAILED_PHASES=0
SKIPPED_PHASES=0
START_TIME=$(date +%s)

# Logging functions
log() {
    echo -e "${1}" | tee -a "$LOG_FILE"
}

log_success() {
    log "${GREEN}✅ ${1}${NC}"
}

log_error() {
    log "${RED}❌ ${1}${NC}"
}

log_warning() {
    log "${YELLOW}⚠️  ${1}${NC}"
}

log_info() {
    log "${BLUE}ℹ️  ${1}${NC}"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    
    # Generate summary
    local end_time=$(date +%s)
    local duration=$((end_time - START_TIME))
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))
    
    echo
    log "=== Test Summary for $SCENARIO_NAME ==="
    log "Total phases: $TOTAL_PHASES"
    log_success "Passed: $PASSED_PHASES"
    log_error "Failed: $FAILED_PHASES"
    log_warning "Skipped: $SKIPPED_PHASES"
    log "Duration: ${minutes}m ${seconds}s"
    log "Log file: $LOG_FILE"
    
    if [ $exit_code -eq 0 ] && [ $FAILED_PHASES -eq 0 ]; then
        log_success "🎉 All tests passed!"
    else
        log_error "💥 Some tests failed!"
    fi
    
    exit $exit_code
}

# Run single phase
run_phase() {
    local phase_name="$1"
    local timeout="$2"
    local description="$3"
    local phase_script="$SCRIPT_DIR/phases/test-${phase_name}.sh"
    
    log_info "[Phase ${PASSED_PHASES:-0}/$TOTAL_PHASES] $description..."
    
    if [ ! -f "$phase_script" ]; then
        log_warning "Phase script not found: $phase_script (skipping)"
        ((SKIPPED_PHASES++))
        return 0
    fi
    
    if [ ! -x "$phase_script" ]; then
        chmod +x "$phase_script"
    fi
    
    local phase_start=$(date +%s)
    local phase_log="/tmp/phase-${phase_name}-${SCENARIO_NAME}-$$.log"
    
    # Run phase with timeout
    if timeout "${timeout}" bash "$phase_script" > "$phase_log" 2>&1; then
        local phase_end=$(date +%s)
        local phase_duration=$((phase_end - phase_start))
        log_success "$description completed (${phase_duration}s)"
        ((PASSED_PHASES++))
        
        # Show important output
        if [ -s "$phase_log" ]; then
            tail -5 "$phase_log" | while read line; do
                log "  $line"
            done
        fi
    else
        local exit_code=$?
        log_error "$description failed (exit code: $exit_code)"
        log_error "Phase log: $phase_log"
        
        # Show error details
        if [ -s "$phase_log" ]; then
            log "=== Error Details ==="
            tail -10 "$phase_log" | while read line; do
                log "  $line"
            done
            log "===================="
        fi
        
        ((FAILED_PHASES++))
        
        # Decide whether to continue
        if [ "${FAIL_FAST:-false}" = "true" ]; then
            log_error "Fail-fast enabled, stopping execution"
            return 1
        fi
    fi
    
    # Append phase log to main log
    echo "=== Phase: $description ===" >> "$LOG_FILE"
    cat "$phase_log" >> "$LOG_FILE" 2>/dev/null || true
    rm -f "$phase_log"
}

# Main execution
main() {
    # Setup
    trap cleanup EXIT
    
    log "=== Starting Comprehensive Tests for $SCENARIO_NAME ==="
    log "Timestamp: $(date)"
    log "Working directory: $(pwd)"
    log "App root: $APP_ROOT"
    log "Log file: $LOG_FILE"
    echo
    
    # Check if scenario is configured properly
    if [ ! -f ".vrooli/service.json" ]; then
        log_error "No .vrooli/service.json found. This doesn't appear to be a properly configured scenario."
        exit 1
    fi
    
    # Parse command line options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --phase)
                SINGLE_PHASE="$2"
                shift 2
                ;;
            --skip-phase)
                SKIP_PHASES="${SKIP_PHASES:-} $2"
                shift 2
                ;;
            --fail-fast)
                FAIL_FAST=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --phase NAME       Run only specified phase"
                echo "  --skip-phase NAME  Skip specified phase"
                echo "  --fail-fast        Stop on first failure"
                echo "  --verbose          Verbose output"
                echo "  --help             Show this help"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Run phases
    for phase_config in "${PHASES[@]}"; do
        IFS=':' read -r phase_name timeout description <<< "$phase_config"
        
        # Skip if requested
        if [[ "${SKIP_PHASES:-}" =~ $phase_name ]]; then
            log_warning "Skipping $description (--skip-phase)"
            ((SKIPPED_PHASES++))
            continue
        fi
        
        # Run only specific phase if requested
        if [ -n "${SINGLE_PHASE:-}" ] && [ "$phase_name" != "$SINGLE_PHASE" ]; then
            continue
        fi
        
        # Run the phase
        if ! run_phase "$phase_name" "$timeout" "$description"; then
            exit 1
        fi
    done
    
    # Final check
    if [ $FAILED_PHASES -gt 0 ]; then
        exit 1
    fi
}

# Execute main function with all arguments
main "$@"
```

### Phase 1: Complete Structure Validation

**File: `test/phases/test-structure.sh` (complete structure testing)**
```bash
#!/bin/bash
# Comprehensive structure validation for Vrooli scenarios
# Validates files, directories, configuration, and project structure

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly SCENARIO_NAME="$(basename "$SCENARIO_DIR")"

# Test counters
tests_passed=0
tests_failed=0

# Test execution function
run_test() {
    local description="$1"
    local test_command="$2"
    
    echo "[TEST] $description"
    if eval "$test_command" 2>/dev/null; then
        echo "✅ PASS: $description"
        tests_passed=$((tests_passed + 1))
        return 0
    else
        echo "❌ FAIL: $description"
        tests_failed=$((tests_failed + 1))
        return 1
    fi
}

# Core file validation
test_core_files() {
    echo "=== Testing Core Files ==="
    
    run_test "README.md exists" "[ -f 'README.md' ]"
    run_test "README.md not empty" "[ -s 'README.md' ]"
    run_test "PRD.md exists" "[ -f 'PRD.md' ]"
    run_test "PRD.md not empty" "[ -s 'PRD.md' ]"
    
    # Configuration files
    run_test ".vrooli/service.json exists" "[ -f '.vrooli/service.json' ]"
    run_test ".vrooli/service.json valid JSON" "jq empty .vrooli/service.json"
    
    # Makefile (modern scenarios should have this)
    run_test "Makefile exists" "[ -f 'Makefile' ]"
    if [ -f "Makefile" ]; then
        run_test "Makefile has test target" "grep -q '^test:' Makefile"
        run_test "Makefile has run target" "grep -q '^run:' Makefile"
    fi
}

# Directory structure validation
test_directory_structure() {
    echo "=== Testing Directory Structure ==="
    
    run_test "Test directory exists" "[ -d 'test' ]"
    
    if [ -d "test" ]; then
        run_test "Test phases directory exists" "[ -d 'test/phases' ]"
        run_test "Main test runner exists" "[ -f 'test/run-tests.sh' ]"
        run_test "Main test runner executable" "[ -x 'test/run-tests.sh' ]"
    fi
    
    # Check for modern vs legacy structure
    if [ -f "scenario-test.yaml" ]; then
        echo "⚠️  WARNING: Found legacy scenario-test.yaml - consider migrating to modern structure"
    fi
}

# Service configuration validation
test_service_configuration() {
    echo "=== Testing Service Configuration ==="
    
    if [ -f ".vrooli/service.json" ]; then
        # Test basic structure
        run_test "Service has name field" "jq -e '.name' .vrooli/service.json >/dev/null"
        run_test "Service has version field" "jq -e '.version' .vrooli/service.json >/dev/null"
        
        # Test component configurations
        local components=$(jq -r 'keys[]' .vrooli/service.json 2>/dev/null | grep -v -E '^(name|version|description)$' || true)
        
        for component in $components; do
            if [[ "$component" =~ ^(api|ui|cli)$ ]]; then
                run_test "$component has port configured" "jq -e '.${component}.port' .vrooli/service.json >/dev/null"
                
                # Check if component directory exists
                local component_dir="$component"
                if [ "$component" = "ui" ] && [ ! -d "ui" ] && [ -d "web" ]; then
                    component_dir="web"
                fi
                
                run_test "$component directory exists" "[ -d '$component_dir' ]"
            fi
        done
        
        # Test configuration
        if jq -e '.test' .vrooli/service.json >/dev/null 2>&1; then
            run_test "Test configuration has steps" "jq -e '.test.steps[]' .vrooli/service.json >/dev/null"
        fi
    fi
}

# Language-specific validation
test_language_structure() {
    echo "=== Testing Language-Specific Structure ==="
    
    # Go API
    if [ -d "api" ]; then
        run_test "Go module file exists" "[ -f 'api/go.mod' ] || [ -f 'go.mod' ]"
        
        # Check for tests
        local go_tests=$(find api -name "*_test.go" 2>/dev/null | wc -l)
        if [ "$go_tests" -gt 0 ]; then
            echo "✅ Found $go_tests Go test files"
        else
            echo "⚠️  No Go test files found in api/"
        fi
    fi
    
    # Node.js/UI
    if [ -d "ui" ] || [ -d "web" ]; then
        local ui_dir="ui"
        [ -d "web" ] && ui_dir="web"
        
        run_test "Package.json exists" "[ -f '$ui_dir/package.json' ]"
        
        if [ -f "$ui_dir/package.json" ]; then
            run_test "Package.json valid JSON" "jq empty '$ui_dir/package.json'"
            
            # Check for test scripts
            if jq -e '.scripts.test' "$ui_dir/package.json" >/dev/null 2>&1; then
                echo "✅ npm test script configured"
            else
                echo "⚠️  No npm test script found"
            fi
        fi
        
        # Check for test files
        local js_tests=$(find "$ui_dir" -name "*.test.js" -o -name "*.test.ts" -o -name "*.spec.js" -o -name "*.spec.ts" 2>/dev/null | wc -l)
        if [ "$js_tests" -gt 0 ]; then
            echo "✅ Found $js_tests JavaScript/TypeScript test files"
        else
            echo "⚠️  No JavaScript/TypeScript test files found"
        fi
    fi
    
    # Python
    if find . -maxdepth 2 -name "*.py" | grep -q .; then
        echo "📁 Python files detected"
        
        # Check for Python tests
        local py_tests=$(find . -name "test_*.py" -o -name "*_test.py" 2>/dev/null | wc -l)
        if [ "$py_tests" -gt 0 ]; then
            echo "✅ Found $py_tests Python test files"
        else
            echo "⚠️  No Python test files found"
        fi
        
        # Check for common Python files
        if [ -f "requirements.txt" ] || [ -f "pyproject.toml" ] || [ -f "setup.py" ]; then
            echo "✅ Python package configuration found"
        else
            echo "⚠️  No Python package configuration found"
        fi
    fi
}

# CLI validation
test_cli_structure() {
    echo "=== Testing CLI Structure ==="
    
    if [ -d "cli" ]; then
        run_test "CLI directory exists" "[ -d 'cli' ]"
        
        # Look for CLI executable
        local cli_files=$(find cli -type f -executable 2>/dev/null | wc -l)
        if [ "$cli_files" -gt 0 ]; then
            echo "✅ Found $cli_files executable CLI files"
        else
            echo "⚠️  No executable CLI files found"
        fi
        
        # Check for BATS tests
        local bats_tests=$(find . -name "*.bats" 2>/dev/null | wc -l)
        if [ "$bats_tests" -gt 0 ]; then
            echo "✅ Found $bats_tests BATS test files"
        else
            echo "⚠️  No BATS test files found"
        fi
    fi
}

# Documentation validation
test_documentation() {
    echo "=== Testing Documentation ==="
    
    if [ -f "README.md" ]; then
        run_test "README has title" "head -1 README.md | grep -q '^#'"
        run_test "README mentions scenario name" "grep -qi '$SCENARIO_NAME' README.md"
    fi
    
    if [ -f "PRD.md" ]; then
        run_test "PRD has title" "head -1 PRD.md | grep -q '^#'"
        
        # Check for common PRD sections
        local sections=("Overview" "Features" "Requirements" "Architecture")
        for section in "${sections[@]}"; do
            if grep -qi "$section" PRD.md; then
                echo "✅ PRD contains $section section"
            else
                echo "⚠️  PRD missing $section section"
            fi
        done
    fi
}

# Security validation
test_security() {
    echo "=== Testing Security ==="
    
    # Check for sensitive files that shouldn't be committed
    local sensitive_patterns=(".env" "*.key" "*.pem" "secrets.yaml" "passwords.txt")
    
    for pattern in "${sensitive_patterns[@]}"; do
        if find . -name "$pattern" -type f 2>/dev/null | grep -q .; then
            echo "⚠️  WARNING: Found potentially sensitive files matching $pattern"
        fi
    done
    
    # Check gitignore
    if [ -f ".gitignore" ]; then
        run_test ".gitignore exists" "[ -f '.gitignore' ]"
        
        local ignore_patterns=(".env" "node_modules" "*.log" ".DS_Store")
        for pattern in "${ignore_patterns[@]}"; do
            if grep -q "$pattern" .gitignore; then
                echo "✅ .gitignore includes $pattern"
            else
                echo "⚠️  .gitignore missing $pattern"
            fi
        done
    else
        echo "⚠️  No .gitignore found"
    fi
}

# Main execution
main() {
    echo "=== Structure Validation for $SCENARIO_NAME ==="
    echo "Working directory: $SCENARIO_DIR"
    echo
    
    # Run all test categories
    test_core_files
    echo
    test_directory_structure  
    echo
    test_service_configuration
    echo
    test_language_structure
    echo
    test_cli_structure
    echo
    test_documentation
    echo
    test_security
    echo
    
    # Summary
    local total_tests=$((tests_passed + tests_failed))
    echo "=== Structure Test Summary ==="
    echo "Total tests: $total_tests"
    echo "Passed: $tests_passed"
    echo "Failed: $tests_failed"
    
    if [ $tests_failed -eq 0 ]; then
        echo "✅ All structure tests passed!"
        exit 0
    else
        echo "❌ Some structure tests failed!"
        exit 1
    fi
}

# Execute from scenario directory
cd "$SCENARIO_DIR"
main "$@"
```

### Phase 5: Complete Business Logic Testing

**File: `test/phases/test-business.sh` (complete business logic testing)**
```bash
#!/bin/bash
# Comprehensive business logic testing for Vrooli scenarios
# Tests core workflows, user journeys, and business requirements

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly SCENARIO_NAME="$(basename "$SCENARIO_DIR")"
readonly APP_ROOT="${APP_ROOT:-$(cd "$SCENARIO_DIR/../.." && pwd)}"

# Source testing libraries
source "$APP_ROOT/scripts/scenarios/testing/shell/core.sh"
source "$APP_ROOT/scripts/scenarios/testing/shell/connectivity.sh"

# Test configuration
tests_passed=0
tests_failed=0
readonly TEST_TIMEOUT=30
readonly MAX_RETRIES=3

# Helper functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

log_success() {
    echo "✅ $1"
}

log_error() {
    echo "❌ $1"
}

log_info() {
    echo "ℹ️  $1"
}

# Test execution with retry logic
run_test() {
    local test_name="$1"
    local test_function="$2"
    local retry_count=0
    
    log_info "Running: $test_name"
    
    while [ $retry_count -lt $MAX_RETRIES ]; do
        if timeout $TEST_TIMEOUT bash -c "$test_function"; then
            log_success "$test_name"
            tests_passed=$((tests_passed + 1))
            return 0
        else
            retry_count=$((retry_count + 1))
            if [ $retry_count -lt $MAX_RETRIES ]; then
                log_info "Retrying $test_name (attempt $((retry_count + 1))/$MAX_RETRIES)"
                sleep 2
            fi
        fi
    done
    
    log_error "$test_name (failed after $MAX_RETRIES attempts)"
    tests_failed=$((tests_failed + 1))
    return 1
}

# Get service URLs with dynamic port discovery
get_service_urls() {
    API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null || echo "")
    UI_URL=$(testing::connectivity::get_ui_url "$SCENARIO_NAME" 2>/dev/null || echo "")
    
    if [ -z "$API_URL" ]; then
        log_error "Could not determine API URL - is the scenario running?"
        return 1
    fi
    
    log_info "API URL: $API_URL"
    log_info "UI URL: $UI_URL"
}

# Test API health and basic functionality
test_api_health() {
    run_test "API health check" "
        curl -sf '$API_URL/health' >/dev/null 2>&1
    "
    
    run_test "API returns valid health response" "
        response=\$(curl -sf '$API_URL/health' 2>/dev/null)
        echo \"\$response\" | jq -e '.status' >/dev/null 2>&1
    "
    
    run_test "API responds within acceptable time" "
        start_time=\$(date +%s%N)
        curl -sf '$API_URL/health' >/dev/null 2>&1
        end_time=\$(date +%s%N)
        duration=\$(( (end_time - start_time) / 1000000 ))
        [ \$duration -lt 1000 ]  # Less than 1 second
    "
}

# Test core CRUD operations (customize based on scenario)
test_crud_operations() {
    local test_item='{\"name\":\"test-item\",\"description\":\"Test item for validation\"}'
    local item_id=""
    
    # Create operation
    run_test "Create new item via API" "
        response=\$(curl -sf -X POST '$API_URL/api/items' \\
            -H 'Content-Type: application/json' \\
            -d '$test_item' 2>/dev/null)
        echo \"\$response\" | jq -e '.id' >/dev/null 2>&1
    "
    
    # Get the created item ID for subsequent tests
    if [ $? -eq 0 ]; then
        item_id=$(curl -sf -X POST "$API_URL/api/items" \
            -H 'Content-Type: application/json' \
            -d "$test_item" 2>/dev/null | jq -r '.id' 2>/dev/null || echo "")
    fi
    
    if [ -n "$item_id" ] && [ "$item_id" != "null" ]; then
        # Read operation
        run_test "Read created item by ID" "
            response=\$(curl -sf '$API_URL/api/items/$item_id' 2>/dev/null)
            echo \"\$response\" | jq -e '.name' >/dev/null 2>&1
        "
        
        # Update operation
        run_test "Update item via API" "
            updated_item='{\"name\":\"updated-test-item\",\"description\":\"Updated test item\"}'
            response=\$(curl -sf -X PUT '$API_URL/api/items/$item_id' \\
                -H 'Content-Type: application/json' \\
                -d \"\$updated_item\" 2>/dev/null)
            echo \"\$response\" | jq -e '.name' >/dev/null 2>&1
        "
        
        # List operation
        run_test "List all items includes our item" "
            response=\$(curl -sf '$API_URL/api/items' 2>/dev/null)
            echo \"\$response\" | jq -e '.[] | select(.id == \"$item_id\")' >/dev/null 2>&1
        "
        
        # Delete operation
        run_test "Delete item via API" "
            curl -sf -X DELETE '$API_URL/api/items/$item_id' >/dev/null 2>&1
        "
        
        # Verify deletion
        run_test "Verify item was deleted" "
            response=\$(curl -sf '$API_URL/api/items/$item_id' 2>/dev/null || echo 'null')
            [ \"\$response\" = \"null\" ] || echo \"\$response\" | grep -q '404\\|not found'
        "
    else
        log_error "Could not create test item, skipping CRUD tests"
    fi
}

# Test user workflow scenarios
test_user_workflows() {
    # Test typical user journey
    run_test "User workflow: Registration/Login simulation" "
        # Simulate user registration (adapt to your scenario)
        user_data='{\"username\":\"test-user\",\"email\":\"test@example.com\"}'
        response=\$(curl -sf -X POST '$API_URL/api/users' \\
            -H 'Content-Type: application/json' \\
            -d \"\$user_data\" 2>/dev/null || echo 'null')
        [ \"\$response\" != \"null\" ]
    "
    
    run_test "User workflow: Profile access" "
        # Test accessing user profile (adapt to your scenario)
        response=\$(curl -sf '$API_URL/api/profile' 2>/dev/null || echo 'null')
        [ \"\$response\" != \"null\" ]
    "
}

# Test data validation and error handling
test_error_handling() {
    run_test "API handles invalid JSON gracefully" "
        response=\$(curl -s -X POST '$API_URL/api/items' \\
            -H 'Content-Type: application/json' \\
            -d 'invalid-json' 2>/dev/null)
        # Should return 4xx status code
        curl -s -o /dev/null -w '%{http_code}' -X POST '$API_URL/api/items' \\
            -H 'Content-Type: application/json' \\
            -d 'invalid-json' | grep -q '^4'
    "
    
    run_test "API handles missing required fields" "
        response=\$(curl -s -X POST '$API_URL/api/items' \\
            -H 'Content-Type: application/json' \\
            -d '{}' 2>/dev/null)
        # Should return 4xx status code  
        curl -s -o /dev/null -w '%{http_code}' -X POST '$API_URL/api/items' \\
            -H 'Content-Type: application/json' \\
            -d '{}' | grep -q '^4'
    "
    
    run_test "API handles non-existent resource requests" "
        # Request non-existent item
        curl -s -o /dev/null -w '%{http_code}' '$API_URL/api/items/99999' | grep -q '^404'
    "
}

# Test UI functionality (if available)
test_ui_functionality() {
    if [ -n "$UI_URL" ]; then
        run_test "UI loads successfully" "
            curl -sf '$UI_URL' >/dev/null 2>&1
        "
        
        run_test "UI serves static assets" "
            # Check for common static assets
            curl -sf '$UI_URL/static/' >/dev/null 2>&1 || 
            curl -sf '$UI_URL/css/' >/dev/null 2>&1 ||
            curl -sf '$UI_URL/js/' >/dev/null 2>&1
        "
        
        run_test "UI has proper HTML structure" "
            response=\$(curl -sf '$UI_URL' 2>/dev/null)
            echo \"\$response\" | grep -q '<html' &&
            echo \"\$response\" | grep -q '<head' &&
            echo \"\$response\" | grep -q '<body'
        "
    else
        log_info "No UI URL available, skipping UI tests"
    fi
}

# Test business-specific features (customize for your scenario)
test_business_features() {
    case "$SCENARIO_NAME" in
        "visited-tracker")
            test_file_tracking_features
            ;;
        "task-manager") 
            test_task_management_features
            ;;
        "data-processor")
            test_data_processing_features
            ;;
        *)
            log_info "No specific business tests for scenario: $SCENARIO_NAME"
            ;;
    esac
}

# Example: File tracking specific tests
test_file_tracking_features() {
    run_test "File tracking: Record visit" "
        visit_data='{\"files\":[\"test.py\",\"main.go\"]}'
        response=\$(curl -sf -X POST '$API_URL/api/visit' \\
            -H 'Content-Type: application/json' \\
            -d \"\$visit_data\" 2>/dev/null)
        echo \"\$response\" | grep -q 'recorded\\|success'
    "
    
    run_test "File tracking: Get visit statistics" "
        response=\$(curl -sf '$API_URL/api/stats' 2>/dev/null)
        echo \"\$response\" | jq -e '.total_visits' >/dev/null 2>&1
    "
}

# Test integration with external resources
test_resource_integration() {
    # Test database connectivity through API
    run_test "Database integration via API" "
        response=\$(curl -sf '$API_URL/api/health/db' 2>/dev/null || echo 'null')
        [ \"\$response\" != \"null\" ] && echo \"\$response\" | jq -e '.status' >/dev/null 2>&1
    "
    
    # Test cache integration if applicable
    run_test "Cache integration check" "
        response=\$(curl -sf '$API_URL/api/health/cache' 2>/dev/null || echo 'null')
        [ \"\$response\" != \"null\" ] || echo 'Cache not configured (OK)'
    "
}

# Main execution
main() {
    echo "=== Business Logic Testing for $SCENARIO_NAME ==="
    echo "Timestamp: $(date)"
    echo
    
    # Check if scenario is running
    if ! testing::core::is_scenario_running "$SCENARIO_NAME"; then
        log_error "Scenario $SCENARIO_NAME is not running. Start it first with 'make run' or 'vrooli scenario run $SCENARIO_NAME'"
        exit 1
    fi
    
    # Get service URLs
    if ! get_service_urls; then
        exit 1
    fi
    
    echo
    
    # Run test categories
    log_info "=== Testing API Health ==="
    test_api_health
    echo
    
    log_info "=== Testing CRUD Operations ==="
    test_crud_operations
    echo
    
    log_info "=== Testing User Workflows ==="
    test_user_workflows
    echo
    
    log_info "=== Testing Error Handling ==="
    test_error_handling
    echo
    
    log_info "=== Testing UI Functionality ==="
    test_ui_functionality
    echo
    
    log_info "=== Testing Business Features ==="
    test_business_features
    echo
    
    log_info "=== Testing Resource Integration ==="
    test_resource_integration
    echo
    
    # Summary
    local total_tests=$((tests_passed + tests_failed))
    echo "=== Business Logic Test Summary ==="
    echo "Total tests: $total_tests"
    echo "Passed: $tests_passed"
    echo "Failed: $tests_failed"
    
    if [ $tests_failed -eq 0 ]; then
        log_success "All business logic tests passed!"
        exit 0
    else
        log_error "$tests_failed business logic tests failed!"
        exit 1
    fi
}

# Execute from scenario directory
cd "$SCENARIO_DIR"
main "$@"
```

## Using Testing Libraries

The centralized testing library provides reusable functions:

```bash
# Source what you need
source "$APP_ROOT/scripts/scenarios/testing/shell/core.sh"
source "$APP_ROOT/scripts/scenarios/testing/shell/connectivity.sh"
source "$APP_ROOT/scripts/scenarios/testing/shell/resources.sh"

# Use the functions
scenario=$(testing::core::detect_scenario)
api_url=$(testing::connectivity::get_api_url "$scenario")
testing::resources::test_postgres "$scenario"
```

## Safety Guidelines

⚠️ **CRITICAL**: Always follow safety guidelines to prevent data loss:

1. **Never use wildcards with unvalidated variables**
2. **Always validate variables in BATS teardown functions**
3. **Set critical variables before skip conditions**
4. **Use the safety linter on test scripts**

```bash
# Lint your test scripts for safety
scripts/scenarios/testing/lint-tests.sh test/
```

See [Safety Guidelines](../safety/GUIDELINES.md) for complete details.

## Dynamic Port Discovery

Never hardcode ports. Always use dynamic discovery:

```bash
# Get ports dynamically
API_PORT=$(vrooli scenario port "$scenario_name" API_PORT)
UI_PORT=$(vrooli scenario port "$scenario_name" UI_PORT)

# Build URLs
API_URL="http://localhost:$API_PORT"
UI_URL="http://localhost:$UI_PORT"
```

## Coverage Standards

- **Unit Tests**: Minimum 70% (80% warning)
- **Integration**: All endpoints tested
- **Business Logic**: Core workflows validated
- **Performance**: Baselines established

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| "service.json not found" | Place in `.vrooli/service.json` |
| "Invalid JSON" | Validate with `jq empty .vrooli/service.json` |
| "Port discovery fails" | Ensure scenario is running |
| "Tests deleting files" | Use safety linter, validate variables |
| "Coverage too low" | Add more unit tests |

## Migration from Legacy Format

If you have an old `scenario-test.yaml`:

1. **Delete it** - No longer needed
2. **Create test directory structure**:
   ```bash
   mkdir -p test/phases
   ```
3. **Copy phase templates**:
   ```bash
   cp scripts/scenarios/templates/full/test/* test/
   ```
4. **Update service.json**:
   ```json
   {
     "test": {
       "description": "Phased testing",
       "steps": [{
         "name": "run-tests",
         "run": "test/run-tests.sh"
       }]
     }
   }
   ```

## Testing Checklist

- [ ] `.vrooli/service.json` properly configured
- [ ] Test directory with all phase scripts
- [ ] Unit tests with coverage >70%
- [ ] Integration tests for all endpoints
- [ ] Business logic tests for workflows
- [ ] Performance baselines established
- [ ] Safety linter run on all test scripts
- [ ] Dynamic port discovery implemented
- [ ] Makefile with test target

## See Also

- [Phased Testing Architecture](../architecture/PHASED_TESTING.md)
- [Safety Guidelines](../safety/GUIDELINES.md)
- [Test Runners Reference](../reference/test-runners.md)
- [Shell Libraries Reference](../reference/shell-libraries.md)