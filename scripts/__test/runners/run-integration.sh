#!/usr/bin/env bash
# Integration Test Runner - Run only integration tests
# Tests real services and multi-component interactions

set -euo pipefail

# Script directory
RUNNER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(dirname "$RUNNER_DIR")"

# Load shared utilities
source "$TEST_ROOT/shared/logging.bash"
source "$TEST_ROOT/shared/utils.bash"
source "$TEST_ROOT/shared/config-simple.bash"
source "$TEST_ROOT/shared/test-isolation.bash"
source "$TEST_ROOT/shared/port-manager.bash"
source "$TEST_ROOT/shared/resource-manager.bash"

# Configuration
VERBOSE="${VROOLI_TEST_VERBOSE:-false}"
FAIL_FAST="${VROOLI_TEST_FAIL_FAST:-false}"
TIMEOUT="${VROOLI_TEST_INTEGRATION_TIMEOUT:-120}"
HEALTH_CHECK_TIMEOUT="${VROOLI_TEST_HEALTH_TIMEOUT:-30}"

# Test tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

#######################################
# Print usage information
#######################################
print_usage() {
    cat << EOF
Usage: $0 [OPTIONS] [SERVICE|SCENARIO...]

Run integration tests against real services.

OPTIONS:
    --verbose         Enable verbose output
    --fail-fast       Stop on first failure
    --timeout N       Test timeout in seconds (default: 120)
    --health-timeout N  Health check timeout (default: 30)
    --skip-health     Skip initial health checks
    --services        Run only service tests
    --scenarios       Run only scenario tests
    --list           List available tests
    -h, --help       Show this help message

EXAMPLES:
    $0                         # Run all integration tests
    $0 ollama                  # Run Ollama integration test
    $0 --services             # Run all service tests
    $0 --scenarios            # Run all scenario tests
    $0 multi-ai-pipeline      # Run specific scenario

AVAILABLE SERVICES:
    ollama, postgres, redis, n8n, comfyui, whisper, 
    windmill, huginn, node-red, searxng, minio, vault,
    qdrant, questdb, judge0, unstructured-io, agent-s2

AVAILABLE SCENARIOS:
    multi-ai-pipeline, data-processing-workflow,
    automation-chain, search-and-analyze

EOF
}

#######################################
# Parse command line arguments
#######################################
parse_args() {
    local targets=()
    local run_services=true
    local run_scenarios=true
    local skip_health=false
    local list_only=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose|-v)
                VERBOSE="true"
                shift
                ;;
            --fail-fast)
                FAIL_FAST="true"
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --health-timeout)
                HEALTH_CHECK_TIMEOUT="$2"
                shift 2
                ;;
            --skip-health)
                skip_health=true
                shift
                ;;
            --services)
                run_scenarios=false
                shift
                ;;
            --scenarios)
                run_services=false
                shift
                ;;
            --list)
                list_only=true
                shift
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            -*)
                vrooli_log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
            *)
                targets+=("$1")
                shift
                ;;
        esac
    done
    
    # Export for use in other functions
    export INTEGRATION_TARGETS=("${targets[@]}")
    export RUN_SERVICES="$run_services"
    export RUN_SCENARIOS="$run_scenarios"
    export SKIP_HEALTH="$skip_health"
    export LIST_ONLY="$list_only"
}

#######################################
# Discover integration tests
#######################################
discover_tests() {
    local -a service_tests=()
    local -a scenario_tests=()
    
    # Discover service tests
    if [[ "$RUN_SERVICES" == "true" ]]; then
        if [[ -d "$TEST_ROOT/integration/services" ]]; then
            while IFS= read -r test_file; do
                local test_name
                test_name=$(basename "$test_file" .sh)
                
                # Check if specific targets requested
                if [[ ${#INTEGRATION_TARGETS[@]} -gt 0 ]]; then
                    local found=false
                    for target in "${INTEGRATION_TARGETS[@]}"; do
                        if [[ "$test_name" == "$target" ]]; then
                            found=true
                            break
                        fi
                    done
                    [[ "$found" == "false" ]] && continue
                fi
                
                service_tests+=("$test_file")
            done < <(find "$TEST_ROOT/integration/services" -name "*.sh" -type f 2>/dev/null | sort)
        fi
    fi
    
    # Discover scenario tests
    if [[ "$RUN_SCENARIOS" == "true" ]]; then
        if [[ -d "$TEST_ROOT/integration/scenarios" ]]; then
            while IFS= read -r test_file; do
                local test_name
                test_name=$(basename "$test_file" .sh)
                
                # Check if specific targets requested
                if [[ ${#INTEGRATION_TARGETS[@]} -gt 0 ]]; then
                    local found=false
                    for target in "${INTEGRATION_TARGETS[@]}"; do
                        if [[ "$test_name" == "$target" ]]; then
                            found=true
                            break
                        fi
                    done
                    [[ "$found" == "false" ]] && continue
                fi
                
                scenario_tests+=("$test_file")
            done < <(find "$TEST_ROOT/integration/scenarios" -name "*.sh" -type f 2>/dev/null | sort)
        fi
    fi
    
    TOTAL_TESTS=$((${#service_tests[@]} + ${#scenario_tests[@]}))
    
    if [[ "$LIST_ONLY" == "true" ]]; then
        vrooli_log_info "Available integration tests:"
        
        if [[ ${#service_tests[@]} -gt 0 ]]; then
            echo "Service Tests:"
            for test in "${service_tests[@]}"; do
                echo "  - $(basename "$test" .sh)"
            done
        fi
        
        if [[ ${#scenario_tests[@]} -gt 0 ]]; then
            echo "Scenario Tests:"
            for test in "${scenario_tests[@]}"; do
                echo "  - $(basename "$test" .sh)"
            done
        fi
        
        exit 0
    fi
    
    vrooli_log_info "Found $TOTAL_TESTS integration tests"
    vrooli_log_info "  Service tests: ${#service_tests[@]}"
    vrooli_log_info "  Scenario tests: ${#scenario_tests[@]}"
    
    # Export for execution
    export SERVICE_TESTS=("${service_tests[@]}")
    export SCENARIO_TESTS=("${scenario_tests[@]}")
}

#######################################
# Run health checks before tests
#######################################
run_health_checks() {
    if [[ "$SKIP_HEALTH" == "true" ]]; then
        vrooli_log_info "Skipping health checks"
        return 0
    fi
    
    vrooli_log_info "Running service health checks..."
    
    local health_script="$TEST_ROOT/integration/health-check.sh"
    if [[ ! -f "$health_script" ]]; then
        vrooli_log_warn "Health check script not found, skipping"
        return 0
    fi
    
    if timeout "$HEALTH_CHECK_TIMEOUT" bash "$health_script"; then
        vrooli_log_success "All services healthy"
        return 0
    else
        vrooli_log_error "Some services are not healthy"
        
        if [[ "$FAIL_FAST" == "true" ]]; then
            vrooli_log_error "Exiting due to health check failure (fail-fast mode)"
            exit 1
        fi
        
        vrooli_log_warn "Continuing despite health check failures"
        return 1
    fi
}

#######################################
# Run a single integration test
#######################################
run_single_test() {
    local test_file="$1"
    local test_type="$2"
    local test_name
    test_name=$(basename "$test_file" .sh)
    
    vrooli_log_info "Running $test_type test: $test_name"
    
    # Create isolated environment for this test
    local test_namespace
    test_namespace=$(vrooli_isolation_init "${test_type}_${test_name}")
    
    # Set up test-specific environment
    export VROOLI_TEST_MODE="integration"
    export VROOLI_TEST_TYPE="$test_type"
    export VROOLI_TEST_NAME="$test_name"
    
    # Run the test with timeout
    local output_file="/tmp/${test_namespace}_output.txt"
    local exit_code=0
    local start_time=$(date +%s)
    
    if timeout "$TIMEOUT" bash "$test_file" > "$output_file" 2>&1; then
        exit_code=0
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        vrooli_log_success "✅ $test_name completed in ${duration}s"
        
        if [[ "$VERBOSE" == "true" ]]; then
            echo "--- Output ---"
            cat "$output_file"
            echo "--- End Output ---"
        fi
    else
        exit_code=$?
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        if [[ $exit_code -eq 124 ]]; then
            vrooli_log_error "❌ $test_name timeout after ${TIMEOUT}s"
        else
            vrooli_log_error "❌ $test_name failed (exit: $exit_code) after ${duration}s"
        fi
        
        # Show output on failure
        if [[ -f "$output_file" ]]; then
            echo "--- Error Output ---"
            tail -n 50 "$output_file"
            echo "--- End Error Output ---"
        fi
    fi
    
    # Clean up
    rm -f "$output_file"
    vrooli_isolation_cleanup
    
    return $exit_code
}

#######################################
# Run service tests
#######################################
run_service_tests() {
    if [[ ${#SERVICE_TESTS[@]} -eq 0 ]]; then
        vrooli_log_info "No service tests to run"
        return 0
    fi
    
    vrooli_log_header "🔧 Running Service Integration Tests"
    
    local failures=0
    
    for test_file in "${SERVICE_TESTS[@]}"; do
        if run_single_test "$test_file" "service"; then
            ((PASSED_TESTS++))
        else
            ((FAILED_TESTS++))
            ((failures++))
            
            if [[ "$FAIL_FAST" == "true" ]]; then
                vrooli_log_error "Stopping due to test failure (fail-fast mode)"
                return 1
            fi
        fi
    done
    
    if [[ $failures -eq 0 ]]; then
        vrooli_log_success "All service tests passed"
        return 0
    else
        vrooli_log_error "$failures service test(s) failed"
        return 1
    fi
}

#######################################
# Run scenario tests
#######################################
run_scenario_tests() {
    if [[ ${#SCENARIO_TESTS[@]} -eq 0 ]]; then
        vrooli_log_info "No scenario tests to run"
        return 0
    fi
    
    vrooli_log_header "🎭 Running Scenario Integration Tests"
    
    local failures=0
    
    for test_file in "${SCENARIO_TESTS[@]}"; do
        if run_single_test "$test_file" "scenario"; then
            ((PASSED_TESTS++))
        else
            ((FAILED_TESTS++))
            ((failures++))
            
            if [[ "$FAIL_FAST" == "true" ]]; then
                vrooli_log_error "Stopping due to test failure (fail-fast mode)"
                return 1
            fi
        fi
    done
    
    if [[ $failures -eq 0 ]]; then
        vrooli_log_success "All scenario tests passed"
        return 0
    else
        vrooli_log_error "$failures scenario test(s) failed"
        return 1
    fi
}

#######################################
# Generate test summary
#######################################
generate_summary() {
    local success_rate=0
    if [[ $TOTAL_TESTS -gt 0 ]]; then
        success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    fi
    
    vrooli_log_header "Integration Test Summary"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Total:    $TOTAL_TESTS"
    echo "  Passed:   $PASSED_TESTS ✅"
    echo "  Failed:   $FAILED_TESTS ❌"
    echo "  Skipped:  $SKIPPED_TESTS ⏭️"
    echo "  Success:  ${success_rate}%"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ $FAILED_TESTS -gt 0 ]]; then
        vrooli_log_error "Integration tests failed"
        return 1
    else
        vrooli_log_success "All integration tests passed!"
        return 0
    fi
}

#######################################
# Main execution
#######################################
main() {
    # Parse arguments
    parse_args "$@"
    
    # Initialize
    vrooli_log_header "🔌 Integration Test Runner"
    vrooli_config_init
    
    # Set up global test isolation
    vrooli_isolation_init "integration-test-suite"
    vrooli_isolation_setup_traps
    
    # Initialize resource manager
    vrooli_resource_init
    
    # Discover tests
    discover_tests
    
    if [[ $TOTAL_TESTS -eq 0 ]]; then
        vrooli_log_warn "No integration tests found"
        exit 0
    fi
    
    # Run health checks
    run_health_checks
    
    # Run tests
    local failures=0
    
    if ! run_service_tests; then
        ((failures++))
    fi
    
    if ! run_scenario_tests; then
        ((failures++))
    fi
    
    # Clean up resources
    vrooli_resource_stop_all
    vrooli_port_release_all
    
    # Generate summary
    generate_summary
    exit $?
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi