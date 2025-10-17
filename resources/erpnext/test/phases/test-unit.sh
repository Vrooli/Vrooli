#!/usr/bin/env bash
################################################################################
# ERPNext Resource - Unit Test Phase
# 
# Library function validation (<60s) per v2.0 contract
# Tests individual functions and components
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
ERPNEXT_RESOURCE_DIR="${APP_ROOT}/resources/erpnext"
ERPNEXT_LIB_DIR="${ERPNEXT_RESOURCE_DIR}/lib"

# Source utilities and core library
source "${APP_ROOT}/scripts/lib/utils/var.sh" || exit 1
source "${var_LIB_UTILS_DIR}/format.sh" || exit 1
source "${var_LIB_UTILS_DIR}/log.sh" || exit 1
source "${APP_ROOT}/scripts/resources/port_registry.sh" || exit 1
source "${ERPNEXT_RESOURCE_DIR}/lib/core.sh" || exit 1

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

################################################################################
# Test Functions
################################################################################

test_core_library_loads() {
    log::info "Testing: Core library loads without errors..."
    
    if source "${ERPNEXT_LIB_DIR}/core.sh" 2>/dev/null; then
        log::success "  ✓ Core library loads successfully"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "  ✗ Core library failed to load"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_config_files_exist() {
    log::info "Testing: Required configuration files exist..."
    
    local all_exist=true
    
    # Check defaults.sh
    if [[ -f "${ERPNEXT_RESOURCE_DIR}/config/defaults.sh" ]]; then
        log::success "  ✓ defaults.sh exists"
    else
        log::error "  ✗ defaults.sh missing"
        all_exist=false
    fi
    
    # Check runtime.json
    if [[ -f "${ERPNEXT_RESOURCE_DIR}/config/runtime.json" ]]; then
        log::success "  ✓ runtime.json exists"
    else
        log::error "  ✗ runtime.json missing"
        all_exist=false
    fi
    
    if $all_exist; then
        ((TESTS_PASSED++))
        return 0
    else
        ((TESTS_FAILED++))
        return 1
    fi
}

test_runtime_json_valid() {
    log::info "Testing: runtime.json is valid JSON..."
    
    if jq . "${ERPNEXT_RESOURCE_DIR}/config/runtime.json" &>/dev/null; then
        log::success "  ✓ runtime.json is valid JSON"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "  ✗ runtime.json is not valid JSON"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_runtime_json_fields() {
    log::info "Testing: runtime.json contains required fields..."
    
    local runtime_file="${ERPNEXT_RESOURCE_DIR}/config/runtime.json"
    local all_fields_ok=true
    
    # Required fields per v2.0 contract
    local required_fields=("startup_order" "dependencies" "startup_timeout" "startup_time_estimate" "recovery_attempts" "priority")
    
    for field in "${required_fields[@]}"; do
        if jq -e ".$field" "$runtime_file" &>/dev/null; then
            log::success "  ✓ Field '$field' exists"
        else
            log::error "  ✗ Field '$field' missing"
            all_fields_ok=false
        fi
    done
    
    if $all_fields_ok; then
        ((TESTS_PASSED++))
        return 0
    else
        ((TESTS_FAILED++))
        return 1
    fi
}

test_port_registry_entry() {
    log::info "Testing: Port registry contains ERPNext entry..."
    
    if [[ -n "${RESOURCE_PORTS["erpnext"]:-}" ]]; then
        log::success "  ✓ ERPNext port registered: ${RESOURCE_PORTS["erpnext"]}"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "  ✗ ERPNext not in port registry"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_function_exports() {
    log::info "Testing: Core functions are exported..."
    
    # Source the core library
    source "${ERPNEXT_LIB_DIR}/core.sh" 2>/dev/null || true
    
    local all_exported=true
    local core_functions=(
        "erpnext::health_check"
        "erpnext::is_running"
        "erpnext::is_healthy"
        "erpnext::is_installed"
        "erpnext::start"
        "erpnext::stop"
        "erpnext::restart"
        "erpnext::info"
        "erpnext::status"
    )
    
    for func in "${core_functions[@]}"; do
        if declare -f "$func" &>/dev/null; then
            log::success "  ✓ Function '$func' is defined"
        else
            log::error "  ✗ Function '$func' is not defined"
            all_exported=false
        fi
    done
    
    if $all_exported; then
        ((TESTS_PASSED++))
        return 0
    else
        ((TESTS_FAILED++))
        return 1
    fi
}

test_cli_script_executable() {
    log::info "Testing: CLI script is executable..."
    
    if [[ -x "${ERPNEXT_RESOURCE_DIR}/cli.sh" ]]; then
        log::success "  ✓ cli.sh is executable"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "  ✗ cli.sh is not executable"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_test_scripts_executable() {
    log::info "Testing: Test scripts are executable..."
    
    local all_executable=true
    
    # Check run-tests.sh
    if [[ -x "${ERPNEXT_RESOURCE_DIR}/test/run-tests.sh" ]]; then
        log::success "  ✓ run-tests.sh is executable"
    else
        log::error "  ✗ run-tests.sh is not executable"
        chmod +x "${ERPNEXT_RESOURCE_DIR}/test/run-tests.sh" 2>/dev/null || true
        all_executable=false
    fi
    
    # Check phase scripts
    for phase_script in "${ERPNEXT_RESOURCE_DIR}/test/phases"/test-*.sh; do
        if [[ -f "$phase_script" ]]; then
            if [[ -x "$phase_script" ]]; then
                log::success "  ✓ $(basename "$phase_script") is executable"
            else
                log::error "  ✗ $(basename "$phase_script") is not executable"
                chmod +x "$phase_script" 2>/dev/null || true
                all_executable=false
            fi
        fi
    done
    
    if $all_executable; then
        ((TESTS_PASSED++))
        return 0
    else
        ((TESTS_FAILED++))
        return 1
    fi
}

################################################################################
# Main Test Execution
################################################################################

main() {
    log::info "=== ERPNext Unit Tests ==="
    echo
    
    # Track start time for 60s limit
    local start_time=$(date +%s)
    
    # Run tests (continue even if individual tests fail)
    test_core_library_loads || true
    test_config_files_exist || true
    test_runtime_json_valid || true
    test_runtime_json_fields || true
    test_port_registry_entry || true
    test_function_exports || true
    test_cli_script_executable || true
    test_test_scripts_executable || true
    
    # Check total time
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo
    log::info "=== Unit Test Results ==="
    log::info "Duration: ${duration}s (limit: 60s)"
    log::info "Passed: ${TESTS_PASSED}"
    log::info "Failed: ${TESTS_FAILED}"
    
    if [[ $duration -gt 60 ]]; then
        log::warn "Tests exceeded 60s time limit"
    fi
    
    if [[ ${TESTS_FAILED} -gt 0 ]]; then
        log::error "Unit tests failed"
        exit 1
    else
        log::success "All unit tests passed!"
        exit 0
    fi
}

# Run main function
main "$@"