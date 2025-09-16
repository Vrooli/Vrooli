#!/usr/bin/env bash
################################################################################
# AutoGPT Unit Tests - Library function tests (<60s)
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/lib/common.sh"

# Test helper functions
test_autogpt_container_exists() {
    log::test "autogpt_container_exists function"
    
    # Function should be defined
    if type -t autogpt_container_exists > /dev/null; then
        log::success "Function is defined"
        
        # Test return value (may be true or false depending on state)
        autogpt_container_exists
        local result=$?
        if [[ ${result} -eq 0 ]] || [[ ${result} -eq 1 ]]; then
            log::success "Function returns valid exit code"
            return 0
        else
            log::error "Function returns invalid exit code: ${result}"
            return 1
        fi
    else
        log::error "Function not defined"
        return 1
    fi
}

# Test container running check
test_autogpt_container_running() {
    log::test "autogpt_container_running function"
    
    if type -t autogpt_container_running > /dev/null; then
        log::success "Function is defined"
        
        # Test return value
        autogpt_container_running
        local result=$?
        if [[ ${result} -eq 0 ]] || [[ ${result} -eq 1 ]]; then
            log::success "Function returns valid exit code"
            return 0
        else
            log::error "Function returns invalid exit code: ${result}"
            return 1
        fi
    else
        log::error "Function not defined"
        return 1
    fi
}

# Test configuration validation
test_config_validation() {
    log::test "Configuration validation"
    
    local failed=0
    
    # Check required variables are defined
    local required_vars=(
        "AUTOGPT_CONTAINER_NAME"
        "AUTOGPT_IMAGE"
        "AUTOGPT_PORT_API"
        "AUTOGPT_DATA_DIR"
        "AUTOGPT_AGENTS_DIR"
    )
    
    for var in "${required_vars[@]}"; do
        if [[ -n "${!var:-}" ]]; then
            log::success "Variable ${var} is defined: ${!var}"
        else
            log::error "Variable ${var} is not defined"
            ((failed++))
        fi
    done
    
    return ${failed}
}

# Test path validation
test_path_validation() {
    log::test "Path validation"
    
    local failed=0
    
    # Check if required directories can be created
    local test_dirs=(
        "${AUTOGPT_DATA_DIR}"
        "${AUTOGPT_AGENTS_DIR}"
        "${AUTOGPT_TOOLS_DIR}"
        "${AUTOGPT_WORKSPACE_DIR}"
    )
    
    for dir in "${test_dirs[@]}"; do
        if mkdir -p "${dir}" 2>/dev/null; then
            log::success "Directory ${dir} is accessible"
        else
            log::error "Cannot create directory ${dir}"
            ((failed++))
        fi
    done
    
    return ${failed}
}

# Test port allocation
test_port_allocation() {
    log::test "Port allocation"
    
    # Check if port is a valid number
    if [[ "${AUTOGPT_PORT_API}" =~ ^[0-9]+$ ]]; then
        log::success "Port is a valid number: ${AUTOGPT_PORT_API}"
        
        # Check if port is in valid range
        if [[ ${AUTOGPT_PORT_API} -ge 1024 ]] && [[ ${AUTOGPT_PORT_API} -le 65535 ]]; then
            log::success "Port is in valid range"
            return 0
        else
            log::error "Port ${AUTOGPT_PORT_API} is out of range"
            return 1
        fi
    else
        log::error "Port is not a valid number: ${AUTOGPT_PORT_API}"
        return 1
    fi
}

# Test environment variable handling
test_env_handling() {
    log::test "Environment variable handling"
    
    local failed=0
    
    # Test LLM provider validation
    local valid_providers=("openrouter" "ollama" "openai" "none")
    local provider="${AUTOGPT_AI_PROVIDER:-none}"
    
    if [[ " ${valid_providers[*]} " =~ " ${provider} " ]]; then
        log::success "LLM provider is valid: ${provider}"
    else
        log::error "Invalid LLM provider: ${provider}"
        ((failed++))
    fi
    
    # Test memory backend validation
    local valid_backends=("redis" "postgres" "local")
    local backend="${AUTOGPT_MEMORY_BACKEND:-local}"
    
    if [[ " ${valid_backends[*]} " =~ " ${backend} " ]]; then
        log::success "Memory backend is valid: ${backend}"
    else
        log::error "Invalid memory backend: ${backend}"
        ((failed++))
    fi
    
    return ${failed}
}

# Test inject function parsing
test_inject_parsing() {
    log::test "Inject function parsing"
    
    # Source inject library if it exists
    if [[ -f "${RESOURCE_DIR}/lib/inject.sh" ]]; then
        source "${RESOURCE_DIR}/lib/inject.sh"
        
        if type -t autogpt_inject > /dev/null; then
            log::success "Inject function is defined"
            
            # Test with invalid file (should fail gracefully)
            if autogpt_inject "/nonexistent/file.yaml" 2>/dev/null; then
                log::error "Inject accepted nonexistent file"
                return 1
            else
                log::success "Inject properly rejects invalid file"
                return 0
            fi
        else
            log::error "Inject function not defined"
            return 1
        fi
    else
        log::warning "Inject library not found - skipping"
        return 0
    fi
}

# Main test execution
main() {
    log::header "AutoGPT Unit Tests"
    
    local failed=0
    
    # Run tests
    test_autogpt_container_exists || ((failed++))
    test_autogpt_container_running || ((failed++))
    test_config_validation || ((failed++))
    test_path_validation || ((failed++))
    test_port_allocation || ((failed++))
    test_env_handling || ((failed++))
    test_inject_parsing || ((failed++))
    
    # Summary
    if [[ ${failed} -eq 0 ]]; then
        log::success "All unit tests passed"
    else
        log::error "${failed} unit tests failed"
    fi
    
    return ${failed}
}

# Execute
main "$@"