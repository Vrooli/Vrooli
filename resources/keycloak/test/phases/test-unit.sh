#!/usr/bin/env bash
################################################################################
# Keycloak Unit Test - Library Function Validation
# Maximum execution time: 60 seconds
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Source Keycloak libraries for testing
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/common.sh" 2>/dev/null || true
source "${RESOURCE_DIR}/lib/status.sh" 2>/dev/null || true

# Test functions
test_configuration_defaults() {
    log::info "Testing configuration defaults..."
    
    local failed=0
    
    # Test required variables are set
    if [[ -z "${KEYCLOAK_CONTAINER_NAME:-}" ]]; then
        log::error "KEYCLOAK_CONTAINER_NAME not set"
        ((failed++))
    else
        log::success "KEYCLOAK_CONTAINER_NAME=${KEYCLOAK_CONTAINER_NAME}"
    fi
    
    if [[ -z "${KEYCLOAK_IMAGE:-}" ]]; then
        log::error "KEYCLOAK_IMAGE not set"
        ((failed++))
    else
        log::success "KEYCLOAK_IMAGE=${KEYCLOAK_IMAGE}"
    fi
    
    if [[ -z "${KEYCLOAK_NETWORK:-}" ]]; then
        log::error "KEYCLOAK_NETWORK not set"
        ((failed++))
    else
        log::success "KEYCLOAK_NETWORK=${KEYCLOAK_NETWORK}"
    fi
    
    return $failed
}

test_port_registry() {
    log::info "Testing port registry integration..."
    
    # Check if keycloak is defined in port registry file
    if grep -q '^\s*\["keycloak"\]="8070"' "${APP_ROOT}/scripts/resources/port_registry.sh"; then
        log::success "Port correctly registered as 8070 in registry file"
        return 0
    else
        log::error "Keycloak not found or incorrectly registered in port registry"
        return 1
    fi
}

test_runtime_config() {
    log::info "Testing runtime configuration..."
    
    local runtime_file="${RESOURCE_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        log::error "Runtime configuration file not found"
        return 1
    fi
    
    # Validate JSON structure
    if python3 -m json.tool "$runtime_file" > /dev/null 2>&1; then
        log::success "Runtime configuration is valid JSON"
    else
        log::error "Runtime configuration has invalid JSON"
        return 1
    fi
    
    # Check required fields
    local startup_order=$(grep -o '"startup_order"[[:space:]]*:[[:space:]]*[0-9]*' "$runtime_file" | cut -d: -f2 | tr -d ' ')
    local dependencies=$(grep -o '"dependencies"[[:space:]]*:[[:space:]]*\[[^]]*\]' "$runtime_file")
    
    if [[ -n "$startup_order" ]]; then
        log::success "Startup order defined: $startup_order"
    else
        log::error "Startup order not defined"
        return 1
    fi
    
    if [[ -n "$dependencies" ]]; then
        log::success "Dependencies defined"
    else
        log::error "Dependencies not defined"
        return 1
    fi
    
    return 0
}

test_cli_structure() {
    log::info "Testing CLI structure..."
    
    local cli_file="${RESOURCE_DIR}/cli.sh"
    
    if [[ ! -f "$cli_file" ]]; then
        log::error "CLI file not found"
        return 1
    fi
    
    if [[ ! -x "$cli_file" ]]; then
        log::error "CLI file not executable"
        return 1
    fi
    
    # Check for required command handlers
    local required_handlers=(
        "manage::install"
        "manage::start"
        "manage::stop"
        "test::smoke"
    )
    
    local failed=0
    for handler in "${required_handlers[@]}"; do
        if grep -q "CLI_COMMAND_HANDLERS\[\"${handler}\"\]" "$cli_file"; then
            log::success "Handler registered: $handler"
        else
            log::error "Handler missing: $handler"
            ((failed++))
        fi
    done
    
    return $failed
}

test_library_functions() {
    log::info "Testing library function existence..."
    
    local lib_dir="${RESOURCE_DIR}/lib"
    local required_libs=(
        "common.sh"
        "install.sh"
        "lifecycle.sh"
        "status.sh"
        "test.sh"
    )
    
    local failed=0
    for lib in "${required_libs[@]}"; do
        if [[ -f "${lib_dir}/${lib}" ]]; then
            log::success "Library exists: $lib"
        else
            log::error "Library missing: $lib"
            ((failed++))
        fi
    done
    
    return $failed
}

# Main test execution
main() {
    log::info "Starting Keycloak unit tests..."
    
    local failed=0
    
    # Run tests
    test_configuration_defaults || ((failed+=$?))
    test_port_registry || ((failed++))
    test_runtime_config || ((failed++))
    test_cli_structure || ((failed+=$?))
    test_library_functions || ((failed+=$?))
    
    # Report results
    if [[ $failed -eq 0 ]]; then
        log::success "All unit tests passed"
        return 0
    else
        log::error "${failed} unit test(s) failed"
        return 1
    fi
}

# Execute main function
main