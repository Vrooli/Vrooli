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

# Fallback log functions if log.sh not available
log::header() { echo -e "\n[HEADER]  $*"; }
log::info() { echo "[INFO]    $*"; }
log::test() { echo "[TEST]    $*"; }
log::success() { echo "[SUCCESS] $*"; }
log::error() { echo "[ERROR]   $*" >&2; }
log::warning() { echo "[WARNING] $*"; }

# Source utilities (will override fallback functions if available)
if [[ -f "${APP_ROOT}/scripts/lib/utils/log.sh" ]]; then
    source "${APP_ROOT}/scripts/lib/utils/log.sh"
fi

# Source libraries to test
source "${HAYSTACK_LIB_DIR}/common.sh" 2>/dev/null || true
source "${HAYSTACK_LIB_DIR}/status.sh" 2>/dev/null || true

# Test functions
test_port_retrieval() {
    log::test "Port retrieval"
    
    # Simple test - port file should exist
    local port_file="${APP_ROOT}/scripts/resources/port_registry.sh"
    if [[ -f "${port_file}" ]] && grep -q '"haystack".*"8075"' "${port_file}"; then
        log::success "Port configuration found: 8075"
        return 0
    else
        log::error "Port configuration not found in ${port_file}"
        return 1
    fi
}

test_library_files() {
    log::test "Required library files exist"
    
    local required_files=("core.sh" "test.sh")
    local missing=0
    
    for file in "${required_files[@]}"; do
        if [[ -f "${HAYSTACK_LIB_DIR}/${file}" ]]; then
            log::success "Found ${file}"
        else
            log::error "Missing ${file}"
            ((missing++))
        fi
    done
    
    return ${missing}
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
    test_library_files || ((failed++))
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

# Run main directly (simpler approach)
main