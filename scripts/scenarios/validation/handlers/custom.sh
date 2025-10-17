#!/bin/bash
# Custom Test Handler - Handles custom test scripts and functions

set -euo pipefail

# Colors for output (use CUSTOM_ prefix to avoid conflicts)
if [[ -z "${CUSTOM_RED:-}" ]]; then
    readonly CUSTOM_RED='\033[0;31m'
    readonly CUSTOM_GREEN='\033[0;32m'
    readonly CUSTOM_YELLOW='\033[1;33m'
    readonly CUSTOM_BLUE='\033[0;34m'
    readonly CUSTOM_NC='\033[0m'
fi

# Custom test results
CUSTOM_ERRORS=0
CUSTOM_WARNINGS=0

# Print functions
print_custom_info() {
    echo -e "${CUSTOM_BLUE}[CUSTOM]${CUSTOM_NC} $1"
}

print_custom_success() {
    echo -e "${CUSTOM_GREEN}[CUSTOM ✓]${CUSTOM_NC} $1"
}

print_custom_error() {
    echo -e "${CUSTOM_RED}[CUSTOM ✗]${CUSTOM_NC} $1"
    ((CUSTOM_ERRORS++))
}

print_custom_warning() {
    echo -e "${CUSTOM_YELLOW}[CUSTOM ⚠]${CUSTOM_NC} $1"
    ((CUSTOM_WARNINGS++))
}

# Source all available framework utilities for custom tests
source_framework_utilities() {
    local framework_dir="$1"
    
    # Source validators
    if [[ -f "$framework_dir/validators/resources.sh" ]]; then
        source "$framework_dir/validators/resources.sh"
    fi
    
    if [[ -f "$framework_dir/validators/structure.sh" ]]; then
        source "$framework_dir/validators/structure.sh"
    fi
    
    # Source handlers
    if [[ -f "$framework_dir/handlers/http.sh" ]]; then
        source "$framework_dir/handlers/http.sh"
    fi
    
    if [[ -f "$framework_dir/handlers/chain.sh" ]]; then
        source "$framework_dir/handlers/chain.sh"
    fi
    
    if [[ -f "$framework_dir/handlers/database.sh" ]]; then
        source "$framework_dir/handlers/database.sh"
    fi
    
    # Source clients
    if [[ -f "$framework_dir/clients/common.sh" ]]; then
        source "$framework_dir/clients/common.sh"
    fi
}

# Execute custom script
execute_custom_script() {
    local script_path="$1"
    local function_name="$2"
    shift 2
    local args=("$@")
    
    print_custom_info "Executing custom script: $(basename "$script_path")"
    
    if [[ ! -f "$script_path" ]]; then
        print_custom_error "Custom script not found: $script_path"
        return 1
    fi
    
    # Make script executable if it isn't
    if [[ ! -x "$script_path" ]]; then
        chmod +x "$script_path"
    fi
    
    # Source the script to make functions available
    if ! source "$script_path"; then
        print_custom_error "Failed to source custom script"
        return 1
    fi
    
    # Check if function exists
    if [[ -n "$function_name" ]]; then
        if ! declare -f "$function_name" >/dev/null; then
            print_custom_error "Function not found in script: $function_name"
            return 1
        fi
        
        print_custom_info "Calling function: $function_name"
        
        # Execute the function with arguments
        if "$function_name" "${args[@]}"; then
            print_custom_success "Custom function completed successfully"
            return 0
        else
            print_custom_error "Custom function failed"
            return 1
        fi
    else
        # Execute script directly if no function specified
        print_custom_info "Executing script directly"
        
        if "$script_path" "${args[@]}"; then
            print_custom_success "Custom script completed successfully"
            return 0
        else
            print_custom_error "Custom script failed"
            return 1
        fi
    fi
}

# Validate custom script before execution
validate_custom_script() {
    local script_path="$1"
    
    if [[ ! -f "$script_path" ]]; then
        print_custom_error "Script file not found: $script_path"
        return 1
    fi
    
    # Check if script has bash shebang
    local first_line=$(head -n 1 "$script_path")
    if [[ ! "$first_line" =~ ^#!/.*bash ]]; then
        print_custom_warning "Script doesn't have bash shebang: $script_path"
    fi
    
    # Check syntax if shellcheck is available
    if command -v shellcheck >/dev/null 2>&1; then
        if ! shellcheck -S error "$script_path" >/dev/null 2>&1; then
            print_custom_warning "Script has syntax issues (shellcheck)"
        fi
    fi
    
    # Check basic syntax with bash -n
    if ! bash -n "$script_path" 2>/dev/null; then
        print_custom_error "Script has syntax errors"
        return 1
    fi
    
    return 0
}

# Create custom test template
create_custom_test_template() {
    local template_path="$1"
    local scenario_name="${2:-example-scenario}"
    
    print_custom_info "Creating custom test template: $template_path"
    
    cat > "$template_path" << EOF
#!/bin/bash
# Custom tests for $scenario_name scenario
# This file provides scenario-specific test logic that can't be expressed declaratively

set -euo pipefail

# Test helper functions are available from the framework
# Use print_custom_info, print_custom_success, print_custom_error for consistent output

# Example: Custom business logic test
test_business_workflow() {
    print_custom_info "Testing custom business workflow"
    
    # Your custom test logic here
    # Use framework functions like:
    # - get_resource_url "service-name"
    # - test_http_endpoint "url" "GET" "200"
    # - execute_ollama_step "step1" "url" "model" "prompt" "output_var"
    
    # Example test
    local ollama_url=\$(get_resource_url "ollama")
    if [[ -n "\$ollama_url" ]]; then
        if test_http_endpoint "\$ollama_url/api/tags" "GET" "200"; then
            print_custom_success "Ollama service is available"
        else
            print_custom_error "Ollama service not responding"
            return 1
        fi
    fi
    
    return 0
}

# Example: Data validation test
test_data_integrity() {
    print_custom_info "Testing data integrity"
    
    # Custom validation logic
    # This might involve checking database state, file system, etc.
    
    return 0
}

# Example: Performance test
test_response_performance() {
    print_custom_info "Testing response performance"
    
    # Custom performance testing
    # Measure response times, throughput, etc.
    
    return 0
}

# Main function called by the framework
run_custom_tests() {
    print_custom_info "Running scenario-specific custom tests"
    
    local tests_passed=0
    local tests_failed=0
    
    # Run each test function
    if test_business_workflow; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    if test_data_integrity; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    if test_response_performance; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Report results
    print_custom_info "Custom tests completed: \$tests_passed passed, \$tests_failed failed"
    
    # Return success if all tests passed
    [[ \$tests_failed -eq 0 ]]
}

# Export functions for framework use
export -f run_custom_tests
export -f test_business_workflow
export -f test_data_integrity
export -f test_response_performance
EOF
    
    chmod +x "$template_path"
    
    print_custom_success "Custom test template created: $template_path"
}

# Execute custom test from configuration
execute_custom_test_from_config() {
    local test_config="$1"
    local scenario_dir="$2"
    
    # Parse configuration (simplified)
    local script_file=$(echo "$test_config" | grep "script:" | cut -d: -f2 | xargs)
    local function_name=$(echo "$test_config" | grep "function:" | cut -d: -f2 | xargs)
    
    # Default to custom-tests.sh if not specified
    script_file="${script_file:-custom-tests.sh}"
    
    # Resolve script path
    local script_path
    if [[ "$script_file" =~ ^/ ]]; then
        # Absolute path
        script_path="$script_file"
    else
        # Relative to scenario directory
        script_path="$scenario_dir/$script_file"
    fi
    
    # Validate script
    if ! validate_custom_script "$script_path"; then
        return 1
    fi
    
    # Execute script
    execute_custom_script "$script_path" "$function_name"
}

# Run standard custom tests for a scenario
run_scenario_custom_tests() {
    local scenario_dir="$1"
    local custom_script="$scenario_dir/custom-tests.sh"
    
    if [[ ! -f "$custom_script" ]]; then
        print_custom_info "No custom tests found, skipping"
        return 0
    fi
    
    print_custom_info "Running scenario custom tests"
    
    # Source framework utilities for custom tests
    local framework_dir="$(dirname "$(dirname "${BASH_SOURCE[0]}")")"
    source_framework_utilities "$framework_dir"
    
    # Execute custom tests
    execute_custom_script "$custom_script" "run_custom_tests"
}

# Provide helper functions for custom tests
# These functions are available to custom test scripts

# Helper: Assert condition
assert_custom() {
    local condition="$1"
    local message="$2"
    
    if [[ "$condition" == "true" ]]; then
        print_custom_success "$message"
        return 0
    else
        print_custom_error "$message"
        return 1
    fi
}

# Helper: Measure execution time
measure_time() {
    local command="$1"
    shift
    local args=("$@")
    
    local start_time=$(date +%s.%N)
    
    if "$command" "${args[@]}"; then
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
        print_custom_info "Execution time: ${duration}s"
        return 0
    else
        print_custom_error "Command failed: $command"
        return 1
    fi
}

# Helper: Wait for condition
wait_for_condition() {
    local condition_command="$1"
    local timeout="${2:-30}"
    local interval="${3:-1}"
    local message="${4:-Waiting for condition}"
    
    print_custom_info "$message"
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if eval "$condition_command" >/dev/null 2>&1; then
            print_custom_success "Condition met after ${elapsed}s"
            return 0
        fi
        
        sleep "$interval"
        elapsed=$((elapsed + interval))
        echo -n "."
    done
    
    echo
    print_custom_error "Condition not met after ${timeout}s"
    return 1
}

# Execute custom test handler
execute_custom_test() {
    local test_name="$1"
    local test_data="$2"
    
    print_custom_info "Executing custom test: $test_name"
    
    # Get scenario directory from environment or test data
    local scenario_dir="${SCENARIO_DIR:-}"
    
    if [[ -z "$scenario_dir" ]]; then
        if [[ -f "$test_data" ]]; then
            # Try to find scenario directory from test data path
            local current_dir="$(dirname "$test_data")"
            while [[ "$current_dir" != "/" && "$current_dir" != "." ]]; do
                if [[ -f "$current_dir/metadata.yaml" ]]; then
                    scenario_dir="$current_dir"
                    break
                fi
                current_dir="$(dirname "$current_dir")"
            done
        fi
    fi
    
    # Default to current directory if still not found
    scenario_dir="${scenario_dir:-.}"
    
    if [[ -f "$test_data" ]]; then
        # Parse test configuration from file
        local script_file=$(grep "script:" "$test_data" | head -1 | cut -d: -f2 | xargs)
        local function_name=$(grep "function:" "$test_data" | head -1 | cut -d: -f2 | xargs)
        
        # Default to custom-tests.sh if not specified
        script_file="${script_file:-custom-tests.sh}"
        
        # Resolve script path
        local script_path
        if [[ "$script_file" =~ ^/ ]]; then
            script_path="$script_file"
        else
            script_path="$scenario_dir/$script_file"
        fi
        
        if [[ ! -f "$script_path" ]]; then
            print_custom_warning "Custom test script not found: $script_path"
            return 0
        fi
        
        # Source framework utilities for custom tests
        local framework_dir="${FRAMEWORK_DIR:-$(dirname "$(dirname "${BASH_SOURCE[0]}")")}"
        source_framework_utilities "$framework_dir"
        
        # Execute the custom test
        execute_custom_script "$script_path" "$function_name"
    else
        # Run default custom tests
        run_scenario_custom_tests "$scenario_dir"
    fi
}

# Export functions for framework use
export -f execute_custom_test
export -f run_scenario_custom_tests
export -f create_custom_test_template
export -f assert_custom
export -f measure_time
export -f wait_for_condition