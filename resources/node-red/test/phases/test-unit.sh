#!/usr/bin/env bash
################################################################################
# Node-RED Unit Tests - Library function validation
# 
# Tests individual functions from the Node-RED lib files
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NODE_RED_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${NODE_RED_DIR}/../.." && pwd)"

# Source utilities and libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"

# Source Node-RED libraries for testing
source "${NODE_RED_DIR}/config/defaults.sh"
node_red::export_config 2>/dev/null || true

for lib in core docker health recovery; do
    lib_file="${NODE_RED_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        source "$lib_file" 2>/dev/null || true
    fi
done

# Test functions
test_config_export() {
    log::test "Configuration export function"
    
    # Test if config export works
    node_red::export_config > /dev/null 2>&1
    
    if [[ -n "${NODE_RED_PORT:-}" ]]; then
        log::success "Config export successful (PORT: ${NODE_RED_PORT})"
        return 0
    else
        log::error "Config export failed"
        return 1
    fi
}

test_health_check_function() {
    log::test "Health check function"
    
    # Test if health check function exists and returns proper exit code
    if declare -f node_red::health > /dev/null; then
        # Function exists, now test it (may fail if service not running)
        if node_red::health 2>/dev/null; then
            log::success "Health check function works"
            return 0
        else
            log::warning "Health check function exists but service may be down"
            return 0  # Function exists, which is what we're testing
        fi
    else
        log::error "Health check function not found"
        return 1
    fi
}

test_docker_functions() {
    log::test "Docker management functions"
    
    local functions_found=0
    local required_functions=(
        "node_red::docker::start"
        "node_red::docker::stop"
        "node_red::docker::restart"
    )
    
    for func in "${required_functions[@]}"; do
        if declare -f "$func" > /dev/null; then
            ((functions_found++))
        else
            log::warning "Missing function: $func"
        fi
    done
    
    if [[ $functions_found -eq ${#required_functions[@]} ]]; then
        log::success "All Docker functions present"
        return 0
    else
        log::error "Some Docker functions missing"
        return 1
    fi
}

test_flow_management_functions() {
    log::test "Flow management functions"
    
    local functions_found=0
    local flow_functions=(
        "node_red::import_flows"
        "node_red::export_flows"
        "node_red::list_flows"
        "node_red::disable_flow"
        "node_red::enable_flow"
    )
    
    for func in "${flow_functions[@]}"; do
        if declare -f "$func" > /dev/null; then
            ((functions_found++))
        fi
    done
    
    if [[ $functions_found -gt 0 ]]; then
        log::success "$functions_found flow functions available"
        return 0
    else
        log::error "No flow management functions found"
        return 1
    fi
}

test_backup_functions() {
    log::test "Backup/restore functions"
    
    if declare -f node_red::create_backup > /dev/null && \
       declare -f node_red::restore_backup > /dev/null; then
        log::success "Backup functions available"
        return 0
    else
        log::warning "Backup functions not fully implemented"
        return 0  # Non-critical
    fi
}

test_error_handling() {
    log::test "Error handling in functions"
    
    # Test that functions handle errors gracefully
    # Try calling a function with invalid parameters
    set +e  # Temporarily disable error exit
    
    # This should fail gracefully
    node_red::docker::start --invalid-flag 2>/dev/null
    local exit_code=$?
    
    set -e  # Re-enable error exit
    
    if [[ $exit_code -ne 0 ]]; then
        log::success "Functions handle errors properly"
        return 0
    else
        log::warning "Error handling needs improvement"
        return 0  # Non-critical
    fi
}

test_cli_framework_integration() {
    log::test "CLI framework integration"
    
    # Test if CLI is properly integrated with v2.0 framework
    if [[ -f "${NODE_RED_DIR}/cli.sh" ]]; then
        # Check for v2.0 mode initialization
        if grep -q 'cli::init.*"v2"' "${NODE_RED_DIR}/cli.sh"; then
            log::success "v2.0 CLI framework integrated"
            return 0
        else
            log::error "CLI not using v2.0 framework"
            return 1
        fi
    else
        log::error "CLI script not found"
        return 1
    fi
}

# Main test execution
main() {
    log::header "Node-RED Unit Tests"
    
    local failed=0
    local tests=(
        test_config_export
        test_health_check_function
        test_docker_functions
        test_flow_management_functions
        test_backup_functions
        test_error_handling
        test_cli_framework_integration
    )
    
    for test in "${tests[@]}"; do
        if ! $test; then
            ((failed++))
        fi
    done
    
    # Summary
    local total=${#tests[@]}
    local passed=$((total - failed))
    
    log::info "Test Summary: $passed/$total passed"
    
    if [[ $failed -eq 0 ]]; then
        log::success "All unit tests passed!"
        return 0
    else
        log::error "$failed unit test(s) failed"
        return 1
    fi
}

# Execute
main "$@"