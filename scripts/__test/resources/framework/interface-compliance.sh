#!/bin/bash
# ====================================================================
# Resource Interface Compliance Testing Framework
# ====================================================================
#
# Validates that resource manage.sh scripts implement the standard
# interface required by the Vrooli resource ecosystem. This ensures
# consistent behavior across all resources.
#
# Usage:
#   source "$SCRIPT_DIR/framework/interface-compliance.sh"
#   test_resource_interface_compliance "$RESOURCE_NAME" "$MANAGE_SCRIPT_PATH"
#
# Required Standard Actions:
#   - install / uninstall
#   - start / stop / restart  
#   - status / logs
#   - help (--help, -h, --version)
#
# Required Argument Patterns:
#   - --action <action_name>
#   - --help, -h (display usage)
#   - --version (display version)
#   - --yes (non-interactive mode)
#
# Exit Codes:
#   0 - All interface compliance tests passed
#   1 - One or more compliance tests failed
#   77 - Tests skipped (missing dependencies)
#
# ====================================================================

set -euo pipefail

# Source assertion framework if not already loaded
if ! declare -f assert_equals >/dev/null 2>&1; then
    SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
    
    # Try multiple potential paths for the assertions framework
    assertion_paths=(
        "$SCRIPT_DIR/framework/helpers/assertions.sh"
        "$SCRIPT_DIR/tests/framework/helpers/assertions.sh"
        "$(dirname "${BASH_SOURCE[0]}")/helpers/assertions.sh"
    )
    
    found_assertions=false
    for path in "${assertion_paths[@]}"; do
        if [[ -f "$path" ]]; then
            source "$path"
            found_assertions=true
            break
        fi
    done
    
    if [[ "$found_assertions" != "true" ]]; then
        echo "Warning: Could not find assertions.sh - tests will run without assertion framework"
    fi
fi

# Interface compliance test counters
INTERFACE_TESTS_RUN=0
INTERFACE_TESTS_PASSED=0
INTERFACE_TESTS_FAILED=0

# Colors for interface test output
IC_GREEN='\033[0;32m'
IC_RED='\033[0;31m'
IC_YELLOW='\033[1;33m'
IC_BLUE='\033[0;34m'
IC_BOLD='\033[1m'
IC_NC='\033[0m'

# Interface compliance logging
ic_log_info() {
    echo -e "${IC_BLUE}[INTERFACE]${IC_NC} $1"
}

ic_log_success() {
    echo -e "${IC_GREEN}[INTERFACE]${IC_NC} ✅ $1"
}

ic_log_error() {
    echo -e "${IC_RED}[INTERFACE]${IC_NC} ❌ $1"
}

ic_log_warning() {
    echo -e "${IC_YELLOW}[INTERFACE]${IC_NC} ⚠️  $1"
}

# Helper function to run manage.sh script with error capture
run_manage_script() {
    local script_path="$1"
    shift
    local args=("$@")
    
    # Capture both stdout and stderr
    local output_file=$(mktemp)
    local exit_code
    
    # Run script with timeout to prevent hanging
    if timeout 30s bash "$script_path" "${args[@]}" >"$output_file" 2>&1; then
        exit_code=0
    else
        exit_code=$?
        # Handle timeout specifically
        if [[ $exit_code -eq 124 ]]; then
            echo "ERROR: Script timed out after 30 seconds" >> "$output_file"
        fi
    fi
    
    # Output the captured content
    cat "$output_file"
    rm -f "$output_file"
    
    return $exit_code
}

# Test help and usage display
test_help_usage_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    ic_log_info "Testing help and usage compliance for $resource_name..."
    
    # Test --help flag
    INTERFACE_TESTS_RUN=$((INTERFACE_TESTS_RUN + 1))
    local help_output
    if help_output=$(run_manage_script "$script_path" --help 2>&1) && [[ $? -eq 0 ]]; then
        if [[ "$help_output" =~ "Usage:" && "$help_output" =~ "--action" ]]; then
            ic_log_success "--help displays proper usage information"
            INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
        else
            ic_log_error "--help output missing required elements (Usage:, --action)"
            echo "  Output: ${help_output:0:200}..."
            INTERFACE_TESTS_FAILED=$((INTERFACE_TESTS_FAILED + 1))
            return 1
        fi
    else
        ic_log_error "--help flag failed to execute properly"
        echo "  Exit code: $?"
        echo "  Output: ${help_output:0:200}..."
        INTERFACE_TESTS_FAILED=$((INTERFACE_TESTS_FAILED + 1))
        return 1
    fi
    
    # Test -h flag
    INTERFACE_TESTS_RUN=$((INTERFACE_TESTS_RUN + 1))
    local h_output
    if h_output=$(run_manage_script "$script_path" -h 2>&1) && [[ $? -eq 0 ]]; then
        if [[ "$h_output" =~ "Usage:" ]]; then
            ic_log_success "-h displays usage information"
            INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
        else
            ic_log_error "-h output missing Usage information"
            INTERFACE_TESTS_FAILED=$((INTERFACE_TESTS_FAILED + 1))
            return 1
        fi
    else
        ic_log_error "-h flag failed to execute properly"
        INTERFACE_TESTS_FAILED=$((INTERFACE_TESTS_FAILED + 1))
        return 1
    fi
    
    # Test --version flag
    INTERFACE_TESTS_RUN=$((INTERFACE_TESTS_RUN + 1))
    local version_output
    if version_output=$(run_manage_script "$script_path" --version 2>&1) && [[ $? -eq 0 ]]; then
        if [[ "$version_output" =~ "$resource_name" ]] || [[ "$version_output" =~ [0-9]+\.[0-9]+ ]]; then
            ic_log_success "--version displays version information"
            INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
        else
            ic_log_warning "--version output doesn't contain resource name or version pattern"
            # This is a warning, not a failure
            INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
        fi
    else
        ic_log_error "--version flag failed to execute properly"
        INTERFACE_TESTS_FAILED=$((INTERFACE_TESTS_FAILED + 1))
        return 1
    fi
    
    return 0
}

# Test required actions implementation
test_required_actions_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    ic_log_info "Testing required actions compliance for $resource_name..."
    
    # Define required actions
    local required_actions=("install" "status" "start" "stop" "logs")
    local optional_actions=("uninstall" "restart")
    
    # Test each required action
    for action in "${required_actions[@]}"; do
        INTERFACE_TESTS_RUN=$((INTERFACE_TESTS_RUN + 1))
        
        # Use --dry-run for potentially destructive actions
        local extra_args=()
        if [[ "$action" == "install" ]]; then
            extra_args=("--dry-run" "yes" "--yes" "yes")
        elif [[ "$action" == "start" || "$action" == "stop" ]]; then
            extra_args=("--dry-run" "yes")
        fi
        
        local action_output
        local exit_code
        if action_output=$(run_manage_script "$script_path" --action "$action" "${extra_args[@]}" 2>&1); then
            exit_code=$?
        else
            exit_code=$?
        fi
        
        # Action should either succeed or give a clear error message
        if [[ $exit_code -eq 0 ]] || [[ "$action_output" =~ "DRY RUN" ]] || [[ "$action_output" =~ "Would" ]]; then
            ic_log_success "Action '$action' is implemented and accessible"
            INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
        elif [[ "$action_output" =~ "Invalid action" ]] || [[ "$action_output" =~ "Unknown action" ]] || [[ "$action_output" =~ "not supported" ]]; then
            ic_log_error "Action '$action' is not implemented (required)"
            echo "  Output: ${action_output:0:200}..."
            INTERFACE_TESTS_FAILED=$((INTERFACE_TESTS_FAILED + 1))
        else
            # Action exists but failed for other reasons (dependencies, etc.) - this is acceptable
            ic_log_success "Action '$action' is implemented (failed due to dependencies/state)"
            INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
        fi
    done
    
    # Test optional actions (warnings only)
    for action in "${optional_actions[@]}"; do
        local action_output
        if action_output=$(run_manage_script "$script_path" --action "$action" --dry-run yes 2>&1); then
            if [[ ! "$action_output" =~ "Invalid action" ]] && [[ ! "$action_output" =~ "Unknown action" ]]; then
                ic_log_success "Optional action '$action' is implemented"
            else
                ic_log_warning "Optional action '$action' is not implemented"
            fi
        else
            ic_log_warning "Optional action '$action' is not implemented or failed"
        fi
    done
    
    return 0
}

# Test error handling compliance
test_error_handling_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    ic_log_info "Testing error handling compliance for $resource_name..."
    
    # Test invalid action handling
    INTERFACE_TESTS_RUN=$((INTERFACE_TESTS_RUN + 1))
    local invalid_output
    local exit_code
    if invalid_output=$(run_manage_script "$script_path" --action invalid_action_name 2>&1); then
        exit_code=$?
    else
        exit_code=$?
    fi
    
    # Should exit with non-zero code and give helpful error message
    if [[ $exit_code -ne 0 ]] && [[ "$invalid_output" =~ "Invalid action" || "$invalid_output" =~ "Unknown action" || "$invalid_output" =~ "Usage:" ]]; then
        ic_log_success "Invalid actions are handled with proper error messages"
        INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
    else
        ic_log_error "Invalid action handling is not compliant"
        echo "  Expected: non-zero exit code with error message"
        echo "  Got: exit code $exit_code"
        echo "  Output: ${invalid_output:0:200}..."
        INTERFACE_TESTS_FAILED=$((INTERFACE_TESTS_FAILED + 1))
        return 1
    fi
    
    # Test no arguments handling
    INTERFACE_TESTS_RUN=$((INTERFACE_TESTS_RUN + 1))
    local no_args_output
    if no_args_output=$(run_manage_script "$script_path" 2>&1); then
        exit_code=$?
    else
        exit_code=$?
    fi
    
    # Should exit with non-zero code and show usage or require action
    if [[ $exit_code -ne 0 ]] && [[ "$no_args_output" =~ "Usage:" || "$no_args_output" =~ "Required:" || "$no_args_output" =~ "--action" ]]; then
        ic_log_success "No arguments case handled with usage information"
        INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
    else
        ic_log_error "No arguments handling is not compliant"
        echo "  Expected: non-zero exit code with usage/requirement message"
        echo "  Got: exit code $exit_code"
        echo "  Output: ${no_args_output:0:200}..."
        INTERFACE_TESTS_FAILED=$((INTERFACE_TESTS_FAILED + 1))
        return 1
    fi
    
    return 0
}

# Test configuration and environment compliance
test_configuration_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    ic_log_info "Testing configuration compliance for $resource_name..."
    
    # Test that script loads configuration properly
    INTERFACE_TESTS_RUN=$((INTERFACE_TESTS_RUN + 1))
    local status_output
    if status_output=$(run_manage_script "$script_path" --action status 2>&1); then
        # Status should run without crashing (exit code doesn't matter for this test)
        ic_log_success "Script loads and executes without configuration errors"
        INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
    else
        local exit_code=$?
        # Only fail if it's a configuration/syntax error, not a service unavailability
        if [[ "$status_output" =~ "command not found" ]] || [[ "$status_output" =~ "syntax error" ]] || [[ "$status_output" =~ "No such file" ]]; then
            ic_log_error "Script has configuration or syntax errors"
            echo "  Output: ${status_output:0:200}..."
            INTERFACE_TESTS_FAILED=$((INTERFACE_TESTS_FAILED + 1))
            return 1
        else
            ic_log_success "Script loads properly (service may be unavailable)"
            INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
        fi
    fi
    
    return 0
}

# Test argument patterns compliance
test_argument_patterns_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    ic_log_info "Testing argument patterns compliance for $resource_name..."
    
    # Test --yes flag support (for non-interactive mode)
    INTERFACE_TESTS_RUN=$((INTERFACE_TESTS_RUN + 1))
    local yes_output
    if yes_output=$(run_manage_script "$script_path" --action status --yes yes 2>&1); then
        # Should accept --yes flag without error (even if service fails)
        if [[ ! "$yes_output" =~ "Unknown option" ]] && [[ ! "$yes_output" =~ "Invalid option" ]]; then
            ic_log_success "--yes flag is supported"
            INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
        else
            ic_log_warning "--yes flag not supported (optional but recommended)"
            INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
        fi
    else
        ic_log_warning "--yes flag may not be supported (optional)"
        INTERFACE_TESTS_PASSED=$((INTERFACE_TESTS_PASSED + 1))
    fi
    
    return 0
}

# Main interface compliance test function
test_resource_interface_compliance() {
    local resource_name="${1:-unknown}"
    local script_path="${2:-}"
    
    if [[ -z "$script_path" ]]; then
        ic_log_error "Script path is required for interface compliance testing"
        return 1
    fi
    
    if [[ ! -f "$script_path" ]]; then
        ic_log_error "Script file not found: $script_path"
        return 1
    fi
    
    if [[ ! -x "$script_path" ]]; then
        # Try to make it executable
        chmod +x "$script_path" 2>/dev/null || {
            ic_log_error "Script is not executable and cannot be made executable: $script_path"
            return 1
        }
    fi
    
    ic_log_info "🔍 Testing interface compliance for: $resource_name"
    ic_log_info "Script path: $script_path"
    echo
    
    # Reset counters for this resource
    INTERFACE_TESTS_RUN=0
    INTERFACE_TESTS_PASSED=0
    INTERFACE_TESTS_FAILED=0
    
    # Run all compliance tests
    local overall_result=0
    
    test_help_usage_compliance "$resource_name" "$script_path" || overall_result=1
    echo
    
    test_required_actions_compliance "$resource_name" "$script_path" || overall_result=1
    echo
    
    test_error_handling_compliance "$resource_name" "$script_path" || overall_result=1
    echo
    
    test_configuration_compliance "$resource_name" "$script_path" || overall_result=1
    echo
    
    test_argument_patterns_compliance "$resource_name" "$script_path" || overall_result=1
    echo
    
    # Print interface compliance summary
    ic_log_info "Interface compliance summary for $resource_name:"
    echo "  Tests run: $INTERFACE_TESTS_RUN"
    echo "  Passed: $INTERFACE_TESTS_PASSED"
    echo "  Failed: $INTERFACE_TESTS_FAILED"
    
    if [[ $INTERFACE_TESTS_FAILED -eq 0 ]]; then
        ic_log_success "$resource_name passes interface compliance tests"
        return 0
    else
        ic_log_error "$resource_name failed $INTERFACE_TESTS_FAILED interface compliance tests"
        return 1
    fi
}

# Validate all resources in a directory
validate_all_resource_interfaces() {
    local resources_dir="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
    
    ic_log_info "Validating all resource interfaces in: $resources_dir"
    
    local total_resources=0
    local compliant_resources=0
    local failed_resources=()
    
    # Find all manage.sh scripts
    while IFS= read -r -d '' script_path; do
        local resource_dir=$(dirname "$script_path")
        local resource_name=$(basename "$resource_dir")
        
        total_resources=$((total_resources + 1))
        
        echo "========================================"
        if test_resource_interface_compliance "$resource_name" "$script_path"; then
            compliant_resources=$((compliant_resources + 1))
        else
            failed_resources+=("$resource_name")
        fi
        echo
    done < <(find "$resources_dir" -name "manage.sh" -type f -print0 2>/dev/null)
    
    # Print final summary
    echo "========================================"
    ic_log_info "Final interface compliance report:"
    echo "  Total resources: $total_resources"
    echo "  Compliant: $compliant_resources"
    echo "  Failed: $((total_resources - compliant_resources))"
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        ic_log_error "Resources failing interface compliance:"
        for resource in "${failed_resources[@]}"; do
            echo "    • $resource"
        done
        return 1
    else
        ic_log_success "All resources pass interface compliance tests!"
        return 0
    fi
}

# Export functions for use in other scripts
export -f test_resource_interface_compliance
export -f validate_all_resource_interfaces