#!/bin/bash
set -euo pipefail

# System Monitor Scenario - Integration Test Script

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/system-monitor"

# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh"
# shellcheck disable=SC1091
source "$var_LOG_FILE"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test function wrapper
run_test() {
    local test_name="$1"
    local test_function="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    log::info "Running test: $test_name"
    
    if $test_function; then
        log::success "✅ $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log::error "❌ $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test required files exist
test_required_files() {
    local required_files=(
        "initialization/storage/postgres/schema.sql"
        "initialization/automation/n8n/scheduled-reports.json"
        "initialization/automation/n8n/threshold-monitor.json"
        "initialization/automation/node-red/anomaly-detector.json"
        "initialization/automation/node-red/metric-collector.json"
        "initialization/configuration/monitoring-config.json"
        "scenario-test.yaml"
        "deployment/startup.sh"
    )
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$SCRIPT_DIR/$file" ]; then
            log::error "Required file missing: $file"
            return 1
        fi
    done
    
    return 0
}

# Test JSON files are valid
test_json_validity() {
    local json_files=(
        "initialization/automation/n8n/scheduled-reports.json"
        "initialization/automation/n8n/threshold-monitor.json"
        "initialization/automation/node-red/anomaly-detector.json"
        "initialization/automation/node-red/metric-collector.json"
        "initialization/configuration/monitoring-config.json"
    )
    
    for json_file in "${json_files[@]}"; do
        local file_path="$SCRIPT_DIR/$json_file"
        if [ -f "$file_path" ]; then
            if ! jq empty < "$file_path" >/dev/null 2>&1; then
                log::error "Invalid JSON in file: $json_file"
                return 1
            fi
        fi
    done
    
    return 0
}

# Test startup script is executable
test_startup_script() {
    local startup_script="$SCRIPT_DIR/deployment/startup.sh"
    
    if [ ! -f "$startup_script" ]; then
        log::error "Startup script not found"
        return 1
    fi
    
    if [ ! -x "$startup_script" ]; then
        log::error "Startup script is not executable"
        return 1
    fi
    
    return 0
}

# Test scenario configuration
test_scenario_configuration() {
    local config_file="$SCRIPT_DIR/scenario-test.yaml"
    
    if [ ! -f "$config_file" ]; then
        log::error "scenario-test.yaml not found"
        return 1
    fi
    
    # Check for expected resources in the configuration
    local expected_resources=("postgres" "questdb" "node-red" "n8n")
    
    for resource in "${expected_resources[@]}"; do
        if ! grep -q "$resource" "$config_file"; then
            log::warning "Resource '$resource' not found in scenario-test.yaml"
        fi
    done
    
    return 0
}

# Main test function
main() {
    log::info "🧪 Starting System Monitor Scenario Integration Tests"
    echo ""
    
    # Run all tests
    run_test "Required files exist" test_required_files
    run_test "JSON files validity" test_json_validity
    run_test "Startup script" test_startup_script
    run_test "Scenario configuration" test_scenario_configuration
    
    echo ""
    log::info "📊 Test Results"
    echo "=============================================="
    echo "Tests Run:    $TESTS_RUN"
    echo "Tests Passed: $TESTS_PASSED"
    echo "Tests Failed: $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log::success "🎉 All tests passed! System Monitor scenario is ready for use."
        exit 0
    else
        log::error "❌ $TESTS_FAILED tests failed. Please address the issues above."
        exit 1
    fi
}

# Handle script arguments
case "${1:-test}" in
    "test"|"run")
        main
        ;;
    "help"|"-h"|"--help")
        echo "System Monitor Scenario - Integration Test Script"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  test, run     Run all integration tests (default)"
        echo "  help          Show this help message"
        ;;
    *)
        log::error "Unknown command: $1"
        log::info "Run '$0 help' for usage information"
        exit 1
        ;;
esac