#!/usr/bin/env bash
################################################################################
# Windmill Unit Tests - Library Function Validation
# 
# Must complete in <60 seconds per v2.0 contract
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WINDMILL_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${WINDMILL_DIR}/../.." && pwd)"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${WINDMILL_DIR}/config/defaults.sh"

# Test configuration
readonly TEST_NAME="Windmill Unit Tests"

# Color codes
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

################################################################################
# Unit Test Functions
################################################################################

test_config_defaults() {
    log::info "Testing: Configuration defaults..."
    
    # Test that critical configuration variables are set
    local all_passed=true
    
    if [[ -n "${WINDMILL_SERVER_PORT:-}" ]]; then
        echo -e "${GREEN}✓${NC} WINDMILL_SERVER_PORT is set: ${WINDMILL_SERVER_PORT}"
    else
        echo -e "${RED}✗${NC} WINDMILL_SERVER_PORT is not set"
        all_passed=false
    fi
    
    if [[ -n "${WINDMILL_PROJECT_NAME:-}" ]]; then
        echo -e "${GREEN}✓${NC} WINDMILL_PROJECT_NAME is set: ${WINDMILL_PROJECT_NAME}"
    else
        echo -e "${RED}✗${NC} WINDMILL_PROJECT_NAME is not set"
        all_passed=false
    fi
    
    if [[ -n "${WINDMILL_BASE_URL:-}" ]]; then
        echo -e "${GREEN}✓${NC} WINDMILL_BASE_URL is set: ${WINDMILL_BASE_URL}"
    else
        echo -e "${RED}✗${NC} WINDMILL_BASE_URL is not set"
        all_passed=false
    fi
    
    if $all_passed; then
        return 0
    else
        return 1
    fi
}

test_library_functions() {
    log::info "Testing: Library function availability..."
    
    local libs_to_test=("common" "api" "workers" "content")
    local all_passed=true
    
    for lib in "${libs_to_test[@]}"; do
        local lib_file="${WINDMILL_DIR}/lib/${lib}.sh"
        if [[ -f "$lib_file" ]]; then
            # Try to source the library
            if source "$lib_file" 2>/dev/null; then
                echo -e "${GREEN}✓${NC} Library ${lib}.sh loaded successfully"
            else
                echo -e "${RED}✗${NC} Library ${lib}.sh failed to load"
                all_passed=false
            fi
        else
            echo -e "${YELLOW}⚠${NC} Library ${lib}.sh not found (optional)"
        fi
    done
    
    if $all_passed; then
        return 0
    else
        return 1
    fi
}

test_api_url_construction() {
    log::info "Testing: API URL construction..."
    
    # Test that API URLs are properly constructed
    local test_endpoints=(
        "/api/version"
        "/api/w/starter/scripts"
        "/api/users/whoami"
    )
    
    local all_passed=true
    
    for endpoint in "${test_endpoints[@]}"; do
        local full_url="${WINDMILL_BASE_URL}${endpoint}"
        if [[ "$full_url" =~ ^http://localhost:[0-9]+/api/ ]]; then
            echo -e "${GREEN}✓${NC} Valid URL: ${full_url}"
        else
            echo -e "${RED}✗${NC} Invalid URL: ${full_url}"
            all_passed=false
        fi
    done
    
    if $all_passed; then
        return 0
    else
        return 1
    fi
}

test_docker_compose_validation() {
    log::info "Testing: Docker Compose file validation..."
    
    if [[ -f "${WINDMILL_COMPOSE_FILE}" ]]; then
        # Validate Docker Compose syntax
        if docker compose -f "${WINDMILL_COMPOSE_FILE}" config &>/dev/null; then
            echo -e "${GREEN}✓${NC} Docker Compose file is valid"
            return 0
        else
            echo -e "${RED}✗${NC} Docker Compose file has syntax errors"
            return 1
        fi
    else
        echo -e "${YELLOW}⚠${NC} Docker Compose file not found"
        return 0
    fi
}

test_environment_variables() {
    log::info "Testing: Environment variable handling..."
    
    # Test that environment variables can be properly set and read
    local test_var="WINDMILL_TEST_VAR_$$"
    export "$test_var=test_value"
    
    if [[ "${!test_var}" == "test_value" ]]; then
        echo -e "${GREEN}✓${NC} Environment variables work correctly"
        unset "$test_var"
        return 0
    else
        echo -e "${RED}✗${NC} Environment variable handling broken"
        unset "$test_var"
        return 1
    fi
}

test_port_validation() {
    log::info "Testing: Port number validation..."
    
    # Test that port is a valid number in the correct range
    if [[ "${WINDMILL_SERVER_PORT}" =~ ^[0-9]+$ ]]; then
        if [[ "${WINDMILL_SERVER_PORT}" -ge 1024 ]] && [[ "${WINDMILL_SERVER_PORT}" -le 65535 ]]; then
            echo -e "${GREEN}✓${NC} Port ${WINDMILL_SERVER_PORT} is valid"
            return 0
        else
            echo -e "${RED}✗${NC} Port ${WINDMILL_SERVER_PORT} is out of valid range"
            return 1
        fi
    else
        echo -e "${RED}✗${NC} Port ${WINDMILL_SERVER_PORT} is not a number"
        return 1
    fi
}

test_cli_command_structure() {
    log::info "Testing: CLI command structure..."
    
    # Test that CLI script exists and is executable
    if [[ -x "${WINDMILL_DIR}/cli.sh" ]]; then
        echo -e "${GREEN}✓${NC} CLI script is executable"
        
        # Test that help command works
        if "${WINDMILL_DIR}/cli.sh" help &>/dev/null; then
            echo -e "${GREEN}✓${NC} CLI help command works"
            return 0
        else
            echo -e "${RED}✗${NC} CLI help command failed"
            return 1
        fi
    else
        echo -e "${RED}✗${NC} CLI script not found or not executable"
        return 1
    fi
}

################################################################################
# Main Test Execution
################################################################################

main() {
    echo "================================"
    echo "$TEST_NAME"
    echo "================================"
    echo ""
    
    local failed_tests=()
    local exit_code=0
    
    # Run each test
    local tests=(
        "test_config_defaults"
        "test_library_functions"
        "test_api_url_construction"
        "test_docker_compose_validation"
        "test_environment_variables"
        "test_port_validation"
        "test_cli_command_structure"
    )
    
    for test in "${tests[@]}"; do
        if $test; then
            :  # Test passed
        else
            failed_tests+=("$test")
            exit_code=1
        fi
        echo ""
    done
    
    # Summary
    echo "================================"
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}✓ All unit tests passed!${NC}"
    else
        echo -e "${RED}✗ Some tests failed:${NC}"
        for test in "${failed_tests[@]}"; do
            echo -e "  ${RED}- ${test}${NC}"
        done
    fi
    echo "================================"
    
    return $exit_code
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi