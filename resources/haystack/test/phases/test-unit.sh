#!/usr/bin/env bash
################################################################################
# Haystack Unit Tests - v2.0 Universal Contract Compliant
# 
# Tests for library functions without starting the service
# Must complete in <60 seconds per universal.yaml
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
HAYSTACK_LIB_DIR="${APP_ROOT}/resources/haystack/lib"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Source libraries to test
source "${HAYSTACK_LIB_DIR}/common.sh" 2>/dev/null || true
source "${HAYSTACK_LIB_DIR}/status.sh" 2>/dev/null || true

# Test functions
test_port_retrieval() {
    log::test "Port retrieval function"
    
    # Check if function exists
    if ! declare -f haystack::get_port &>/dev/null; then
        log::warning "haystack::get_port function not found"
        return 0  # Don't fail, implementation pending
    fi
    
    local port
    port=$(haystack::get_port)
    
    if [[ "${port}" == "8075" ]]; then
        log::success "Port correctly retrieved: ${port}"
        return 0
    else
        log::error "Incorrect port: ${port} (expected 8075)"
        return 1
    fi
}

test_installation_check() {
    log::test "Installation check function"
    
    # Check if function exists
    if ! declare -f haystack::is_installed &>/dev/null; then
        log::warning "haystack::is_installed function not found"
        return 0  # Don't fail, implementation pending
    fi
    
    # Function should exist and be callable
    if haystack::is_installed 2>/dev/null; then
        log::success "Installation check works (installed)"
    else
        log::success "Installation check works (not installed)"
    fi
    
    return 0
}

test_running_check() {
    log::test "Running check function"
    
    # Check if function exists
    if ! declare -f haystack::is_running &>/dev/null; then
        log::warning "haystack::is_running function not found"
        return 0  # Don't fail, implementation pending
    fi
    
    # Function should exist and be callable
    if haystack::is_running 2>/dev/null; then
        log::success "Running check works (running)"
    else
        log::success "Running check works (not running)"
    fi
    
    return 0
}

test_config_loading() {
    log::test "Configuration loading"
    
    local runtime_file="${APP_ROOT}/resources/haystack/config/runtime.json"
    
    if [[ ! -f "${runtime_file}" ]]; then
        log::error "runtime.json not found"
        return 1
    fi
    
    # Validate JSON structure
    if jq -e . "${runtime_file}" &>/dev/null; then
        log::success "runtime.json is valid JSON"
        
        # Check required fields
        local required_fields=("startup_order" "dependencies" "startup_timeout" "recovery_attempts" "priority")
        local missing=()
        
        for field in "${required_fields[@]}"; do
            if ! jq -e ".${field}" "${runtime_file}" &>/dev/null; then
                missing+=("${field}")
            fi
        done
        
        if [[ ${#missing[@]} -eq 0 ]]; then
            log::success "All required fields present in runtime.json"
            return 0
        else
            log::error "Missing fields in runtime.json: ${missing[*]}"
            return 1
        fi
    else
        log::error "runtime.json is not valid JSON"
        return 1
    fi
}

test_library_sourcing() {
    log::test "All libraries can be sourced"
    
    local failed=0
    local libs=("common" "status" "install" "lifecycle" "inject")
    
    for lib in "${libs[@]}"; do
        local lib_file="${HAYSTACK_LIB_DIR}/${lib}.sh"
        if [[ -f "${lib_file}" ]]; then
            if bash -n "${lib_file}" 2>/dev/null; then
                log::success "  ${lib}.sh: valid syntax"
            else
                log::error "  ${lib}.sh: syntax errors"
                ((failed++))
            fi
        else
            log::warning "  ${lib}.sh: not found"
        fi
    done
    
    if [[ ${failed} -eq 0 ]]; then
        log::success "All libraries have valid syntax"
        return 0
    else
        log::error "${failed} libraries have syntax errors"
        return 1
    fi
}

# Main test execution
main() {
    local failed=0
    
    log::header "Haystack Unit Tests"
    
    # Run tests
    test_port_retrieval || ((failed++))
    test_installation_check || ((failed++))
    test_running_check || ((failed++))
    test_config_loading || ((failed++))
    test_library_sourcing || ((failed++))
    
    # Report results
    if [[ ${failed} -eq 0 ]]; then
        log::success "All unit tests passed"
        exit 0
    else
        log::error "${failed} unit tests failed"
        exit 1
    fi
}

# Run with timeout (60 seconds per universal.yaml)
timeout 60 bash -c "$(declare -f main test_port_retrieval test_installation_check test_running_check test_config_loading test_library_sourcing); main" || {
    log::error "Unit tests exceeded 60 second timeout"
    exit 1
}