#!/bin/bash
# ====================================================================
# Test Execution Engine
# ====================================================================
#
# Handles the execution of both single-resource and multi-resource
# integration tests with proper isolation, cleanup, and reporting.
#
# Functions:
#   - run_single_resource_tests()  - Execute single resource tests
#   - run_multi_resource_tests()   - Execute multi-resource tests
#   - execute_test_file()          - Run individual test file
#   - extract_required_resources() - Parse test dependencies
#   - all_resources_available()    - Check resource availability
#   - get_missing_resources()      - Identify missing dependencies
#
# ====================================================================

# Source helper functions
source "$SCRIPT_DIR/framework/helpers/assertions.sh" 2>/dev/null || true
source "$SCRIPT_DIR/framework/helpers/cleanup.sh" 2>/dev/null || true

# Run single-resource tests
run_single_resource_tests() {
    local tests_run=0
    local tests_passed=0
    local tests_failed=0
    
    log_info "Executing single-resource tests..."
    
    for resource in "${HEALTHY_RESOURCES[@]}"; do
        if [[ -n "$SPECIFIC_RESOURCE" && "$resource" != "$SPECIFIC_RESOURCE" ]]; then
            continue
        fi
        
        # Find test file for this resource
        local test_file
        test_file=$(find_resource_test_file "$resource" || echo "")
        
        log_debug "Looking for test file for $resource..."
        log_debug "Found test file: $test_file"
        
        if [[ -z "$test_file" ]]; then
            log_warning "⚠️  No test file found for $resource (skipping)"
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            SKIPPED_TEST_NAMES+=("$resource: no test file available")
            MISSING_TEST_RESOURCES+=("$resource")
            continue
        fi
        
        log_info "Testing $resource..."
        log_debug "About to execute test file: $test_file"
        
        # Execute the test file
        local test_result
        test_result=$(execute_test_file "$test_file" "$resource")
        local exit_code=$?
        
        tests_run=$((tests_run + 1))
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        
        if [[ $exit_code -eq 0 ]]; then
            tests_passed=$((tests_passed + 1))
            PASSED_TESTS=$((PASSED_TESTS + 1))
            log_success "✅ $resource tests passed"
        else
            tests_failed=$((tests_failed + 1))
            FAILED_TESTS=$((FAILED_TESTS + 1))
            FAILED_TEST_NAMES+=("$resource: $test_result")
            log_error "❌ $resource tests failed: $test_result"
            
            if [[ "$FAIL_FAST" == "true" ]]; then
                log_error "Stopping due to --fail-fast option"
                return 1
            fi
        fi
    done
    
    log_info "Single-resource tests complete: $tests_passed/$tests_run passed"
    
    if [[ $tests_failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Run multi-resource integration tests
run_multi_resource_tests() {
    local tests_run=0
    local tests_passed=0
    local tests_failed=0
    
    log_info "Executing multi-resource integration tests..."
    
    # Find all multi-resource test files
    local test_files=()
    while IFS= read -r -d '' file; do
        test_files+=("$file")
    done < <(find "$SCRIPT_DIR/multi" -name "*.test.sh" -type f -print0 2>/dev/null)
    
    if [[ ${#test_files[@]} -eq 0 ]]; then
        log_warning "⚠️  No multi-resource test files found"
        return 0
    fi
    
    for test_file in "${test_files[@]}"; do
        local test_name
        test_name=$(basename "$test_file" .test.sh)
        
        # Extract required resources for this test
        local required_resources
        required_resources=$(extract_required_resources "$test_file")
        
        if [[ -z "$required_resources" ]]; then
            log_warning "⚠️  Skipping $test_name - no resource requirements found"
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            SKIPPED_TEST_NAMES+=("$test_name: no resource requirements")
            continue
        fi
        
        # Check if all required resources are available
        local missing_resources
        missing_resources=$(get_missing_resources "$required_resources" "${HEALTHY_RESOURCES[@]}")
        
        if [[ -n "$missing_resources" ]]; then
            log_warning "⚠️  Skipping $test_name - missing resources: $missing_resources"
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            SKIPPED_TEST_NAMES+=("$test_name: missing $missing_resources")
            continue
        fi
        
        log_info "Testing $test_name (requires: $required_resources)..."
        
        # Execute the test file
        local test_result
        test_result=$(execute_test_file "$test_file" "multi:$test_name")
        local exit_code=$?
        
        tests_run=$((tests_run + 1))
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        
        if [[ $exit_code -eq 0 ]]; then
            tests_passed=$((tests_passed + 1))
            PASSED_TESTS=$((PASSED_TESTS + 1))
            log_success "✅ $test_name integration test passed"
        else
            tests_failed=$((tests_failed + 1))
            FAILED_TESTS=$((FAILED_TESTS + 1))
            FAILED_TEST_NAMES+=("$test_name: $test_result")
            log_error "❌ $test_name integration test failed: $test_result"
            
            if [[ "$FAIL_FAST" == "true" ]]; then
                log_error "Stopping due to --fail-fast option"
                return 1
            fi
        fi
    done
    
    log_info "Multi-resource tests complete: $tests_passed/$tests_run passed"
    
    if [[ $tests_failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Execute a specific test file
execute_test_file() {
    local test_file="$1"
    local test_identifier="$2"
    local start_time=$(date +%s)
    
    # Source discovery functions for get_resource_info
    source "$SCRIPT_DIR/framework/discovery.sh" 2>/dev/null || true
    
    # Get port for the resource if it's a single resource test
    local resource_port=""
    if [[ "$test_identifier" != "multi:"* ]]; then
        # Get resource info including port
        local resource_info
        resource_info=$(get_resource_info "$test_identifier" 2>/dev/null)
        if [[ -n "$resource_info" ]]; then
            resource_port=$(echo "$resource_info" | jq -r '.port // empty' 2>/dev/null)
        fi
    fi
    
    # Create test-specific environment
    local test_env_file="/tmp/vrooli_test_env_$$"
    cat > "$test_env_file" << EOF
export TEST_ID="${test_identifier}_$(date +%s)"
export TEST_TIMEOUT="$TEST_TIMEOUT"
export TEST_VERBOSE="$VERBOSE"
export TEST_CLEANUP="$CLEANUP"
export SCRIPT_DIR="$SCRIPT_DIR"
export RESOURCES_DIR="$RESOURCES_DIR"
# Export healthy resources as a space-separated string
export HEALTHY_RESOURCES_STR="${HEALTHY_RESOURCES[*]}"
EOF
    
    # Add resource-specific port if available
    if [[ -n "$resource_port" ]]; then
        local port_var_name="${test_identifier^^}_PORT"
        echo "export ${port_var_name}=\"$resource_port\"" >> "$test_env_file"
    fi
    
    # Note: Individual tests will use HEALTHY_RESOURCES_STR directly via require_resource()
    
    # Set up test logging
    local test_log_file="/tmp/vrooli_test_${test_identifier//[^a-zA-Z0-9]/_}_$$.log"
    
    log_debug "Starting test: $test_file"
    log_debug "Test log: $test_log_file"
    log_debug "Test environment: $test_env_file"
    log_debug "Test timeout: $TEST_TIMEOUT seconds"
    
    # Debug: show environment file contents
    if [[ "$VERBOSE" == "true" ]]; then
        log_debug "Environment file contents:"
        cat "$test_env_file" | sed 's/^/    /'
    fi
    
    # Execute test with timeout
    local test_output
    local exit_code
    
    log_debug "About to run test in subshell with timeout $TEST_TIMEOUT"
    
    # Run test in subshell with timeout
    (
        source "$test_env_file"
        cd "$(dirname "$test_file")"
        timeout "$TEST_TIMEOUT" bash "$test_file"
    ) > "$test_log_file" 2>&1
    exit_code=$?
    
    log_debug "Test execution completed with exit code: $exit_code"
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Read test output
    test_output=$(cat "$test_log_file" 2>/dev/null || echo "No test output")
    
    # Handle different exit codes
    case $exit_code in
        0)
            log_debug "Test passed in ${duration}s"
            if [[ "$VERBOSE" == "true" ]]; then
                echo "$test_output" | sed 's/^/    /'
            fi
            ;;
        124)
            echo "Test timed out after ${TEST_TIMEOUT}s"
            exit_code=1
            ;;
        *)
            echo "Test failed with exit code $exit_code"
            if [[ "$VERBOSE" == "true" ]]; then
                echo "$test_output" | sed 's/^/    /'
            else
                # Show last few lines of output for failed tests
                echo "$test_output" | tail -n 5 | sed 's/^/    /'
            fi
            ;;
    esac
    
    # Cleanup test environment
    rm -f "$test_env_file" "$test_log_file" 2>/dev/null || true
    
    return $exit_code
}

# Find test file for a specific resource
find_resource_test_file() {
    local resource="$1"
    
    # Map resource to category and find test file
    local categories=("ai" "automation" "agents" "search" "storage" "browser" "visual")
    
    for category in "${categories[@]}"; do
        local test_file="$SCRIPT_DIR/single/$category/$resource.test.sh"
        if [[ -f "$test_file" ]]; then
            echo "$test_file"
            return 0
        fi
    done
    
    # Return empty if not found
    echo ""
    return 1
}

# Extract required resources from test file
extract_required_resources() {
    local test_file="$1"
    
    # Look for REQUIRED_RESOURCES declaration in the test file
    local required_resources
    required_resources=$(grep -E "^[[:space:]]*REQUIRED_RESOURCES=" "$test_file" 2>/dev/null | head -n1 | cut -d'=' -f2- | tr -d '"()' | tr ' ' '\n' | grep -v '^$' | sort -u | tr '\n' ' ')
    
    # If not found, try alternative patterns
    if [[ -z "$required_resources" ]]; then
        # Look for comment-based requirements
        required_resources=$(grep -E "^[[:space:]]*#[[:space:]]*@requires[[:space:]]*:" "$test_file" 2>/dev/null | head -n1 | cut -d':' -f2- | tr ',' ' ' | sed 's/[[:space:]]\+/ /g' | xargs)
    fi
    
    # Clean up and validate
    required_resources=$(echo "$required_resources" | xargs)
    
    echo "$required_resources"
}

# Check if all required resources are available
all_resources_available() {
    local required_str="$1"
    shift
    local available_resources=("$@")
    
    # Convert required string to array
    local required_resources
    IFS=' ' read -ra required_resources <<< "$required_str"
    
    for required in "${required_resources[@]}"; do
        local found=false
        for available in "${available_resources[@]}"; do
            if [[ "$required" == "$available" ]]; then
                found=true
                break
            fi
        done
        
        if [[ "$found" == "false" ]]; then
            return 1
        fi
    done
    
    return 0
}

# Get list of missing resources
get_missing_resources() {
    local required_str="$1"
    shift
    local available_resources=("$@")
    
    # Convert required string to array
    local required_resources
    IFS=' ' read -ra required_resources <<< "$required_str"
    
    local missing_resources=()
    
    for required in "${required_resources[@]}"; do
        local found=false
        for available in "${available_resources[@]}"; do
            if [[ "$required" == "$available" ]]; then
                found=true
                break
            fi
        done
        
        if [[ "$found" == "false" ]]; then
            missing_resources+=("$required")
        fi
    done
    
    echo "${missing_resources[*]}"
}

# Generate test ID for isolation
generate_test_id() {
    echo "test_$(date +%s)_$$_$RANDOM"
}

# Wait for resource to be ready
wait_for_resource() {
    local resource="$1"
    local timeout="${2:-30}"
    local interval="${3:-2}"
    
    log_debug "Waiting for $resource to be ready (timeout: ${timeout}s)..."
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        local health_status
        health_status=$(check_resource_health "$resource")
        
        if [[ "$health_status" == "healthy" ]]; then
            log_debug "$resource is ready after ${elapsed}s"
            return 0
        fi
        
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done
    
    log_error "$resource not ready after ${timeout}s timeout"
    return 1
}

# Create isolated test environment
setup_test_environment() {
    local test_id="$1"
    
    # Create test-specific directories
    local test_dir="/tmp/vrooli_integration_test_$test_id"
    mkdir -p "$test_dir"/{logs,artifacts,temp}
    
    export TEST_DIR="$test_dir"
    export TEST_LOGS_DIR="$test_dir/logs"
    export TEST_ARTIFACTS_DIR="$test_dir/artifacts"
    export TEST_TEMP_DIR="$test_dir/temp"
    
    log_debug "Test environment created: $test_dir"
}

# Cleanup test environment
cleanup_test_environment() {
    local test_id="$1"
    
    if [[ "$CLEANUP" == "true" ]]; then
        local test_dir="/tmp/vrooli_integration_test_$test_id"
        if [[ -d "$test_dir" ]]; then
            rm -rf "$test_dir"
            log_debug "Cleaned up test environment: $test_dir"
        fi
    fi
}

# ====================================================================
# Scenario Execution Functions
# ====================================================================

# Run business scenario tests
run_scenario_tests() {
    local tests_run=0
    local tests_passed=0
    local tests_failed=0
    
    log_info "Executing business scenario tests..."
    
    # Determine which scenarios to run
    local scenarios_to_run=()
    if [[ -n "$SCENARIO_FILTER" && ${#FILTERED_SCENARIOS[@]} -gt 0 ]]; then
        scenarios_to_run=("${FILTERED_SCENARIOS[@]}")
        log_info "Running filtered scenarios: ${scenarios_to_run[*]}"
    elif [[ ${#RUNNABLE_SCENARIOS[@]} -gt 0 ]]; then
        scenarios_to_run=("${RUNNABLE_SCENARIOS[@]}")
        log_info "Running all runnable scenarios: ${scenarios_to_run[*]}"
    else
        log_warning "No scenarios available to run"
        return 0
    fi
    
    for scenario in "${scenarios_to_run[@]}"; do
        local scenario_file="${SCENARIO_METADATA["$scenario:file"]}"
        
        if [[ ! -f "$scenario_file" ]]; then
            log_warning "⚠️  Scenario file not found: $scenario_file"
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            SKIPPED_TEST_NAMES+=("$scenario: file not found")
            continue
        fi
        
        # Get scenario metadata for display
        local category="${SCENARIO_METADATA["$scenario:category"]:-unknown}"
        local complexity="${SCENARIO_METADATA["$scenario:complexity"]:-unknown}"
        local business_value="${SCENARIO_METADATA["$scenario:business_value"]:-unknown}"
        local revenue_potential="${SCENARIO_METADATA["$scenario:revenue_potential"]:-unknown}"
        
        log_info "🎯 Testing scenario: $scenario"
        log_info "   Category: $category | Complexity: $complexity"
        log_info "   Business Value: $business_value | Revenue: $revenue_potential"
        
        # Execute the scenario test
        local test_result
        test_result=$(execute_test_file "$scenario_file" "scenario:$scenario")
        local exit_code=$?
        
        tests_run=$((tests_run + 1))
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        
        if [[ $exit_code -eq 0 ]]; then
            tests_passed=$((tests_passed + 1))
            PASSED_TESTS=$((PASSED_TESTS + 1))
            PASSED_TEST_NAMES+=("$scenario")
            log_success "✅ Scenario $scenario passed"
        else
            tests_failed=$((tests_failed + 1))
            FAILED_TESTS=$((FAILED_TESTS + 1))
            FAILED_TEST_NAMES+=("$scenario: $test_result")
            log_error "❌ Scenario $scenario failed: $test_result"
            
            if [[ "$FAIL_FAST" == "true" ]]; then
                log_error "Stopping due to --fail-fast option"
                return 1
            fi
        fi
    done
    
    # Report blocked scenarios
    if [[ ${#BLOCKED_SCENARIOS[@]} -gt 0 ]]; then
        log_info "⚠️  Blocked scenarios (missing resources):"
        for blocked_scenario in "${BLOCKED_SCENARIOS[@]}"; do
            local required_services="${SCENARIO_METADATA["$blocked_scenario:services"]}"
            log_info "   $blocked_scenario (needs: $required_services)"
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            SKIPPED_TEST_NAMES+=("$blocked_scenario: missing services")
        done
    fi
    
    log_info "Business scenario tests complete: $tests_passed/$tests_run passed"
    
    if [[ $tests_failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Execute scenario test with business context
execute_scenario_test() {
    local scenario_file="$1"
    local scenario_name="$2"
    local start_time=$(date +%s)
    
    # Create scenario-specific environment
    local scenario_env_file="/tmp/vrooli_scenario_env_$$"
    cat > "$scenario_env_file" << EOF
export TEST_ID="scenario_${scenario_name}_$(date +%s)"
export TEST_TIMEOUT="$TEST_TIMEOUT"
export TEST_VERBOSE="$VERBOSE"
export TEST_CLEANUP="$CLEANUP"
export SCRIPT_DIR="$SCRIPT_DIR"
export RESOURCES_DIR="$RESOURCES_DIR"
export SCENARIO_NAME="$scenario_name"
# Export healthy resources as a space-separated string
export HEALTHY_RESOURCES_STR="${HEALTHY_RESOURCES[*]}"
# Recreate the array from the string
HEALTHY_RESOURCES=(\$HEALTHY_RESOURCES_STR)
EOF
    
    # Set up scenario logging
    local scenario_log_file="/tmp/vrooli_scenario_${scenario_name//[^a-zA-Z0-9]/_}_$$.log"
    
    log_debug "Starting scenario: $scenario_file"
    log_debug "Scenario log: $scenario_log_file"
    log_debug "Scenario environment: $scenario_env_file"
    
    # Execute scenario with extended timeout
    local scenario_timeout=$((TEST_TIMEOUT * 2))  # Scenarios get double timeout
    local test_output
    local exit_code
    
    # Run scenario in subshell with timeout
    (
        source "$scenario_env_file"
        cd "$(dirname "$scenario_file")"
        timeout "$scenario_timeout" bash "$scenario_file"
    ) > "$scenario_log_file" 2>&1
    exit_code=$?
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Read scenario output
    test_output=$(cat "$scenario_log_file" 2>/dev/null || echo "No scenario output")
    
    # Handle different exit codes
    case $exit_code in
        0)
            log_debug "Scenario passed in ${duration}s"
            if [[ "$VERBOSE" == "true" ]]; then
                echo "$test_output" | sed 's/^/    /'
            fi
            ;;
        124)
            echo "Scenario timed out after ${scenario_timeout}s"
            exit_code=1
            ;;
        *)
            echo "Scenario failed with exit code $exit_code"
            if [[ "$VERBOSE" == "true" ]]; then
                echo "$test_output" | sed 's/^/    /'
            else
                # Show last few lines of output for failed scenarios
                echo "$test_output" | tail -n 10 | sed 's/^/    /'
            fi
            ;;
    esac
    
    # Cleanup scenario environment
    rm -f "$scenario_env_file" "$scenario_log_file" 2>/dev/null || true
    
    return $exit_code
}