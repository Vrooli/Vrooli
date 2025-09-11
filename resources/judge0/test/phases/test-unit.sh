#!/usr/bin/env bash
################################################################################
# Judge0 Unit Tests - Library Function Validation (<60s)
# 
# Tests individual library functions and CLI commands
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"

# Load Judge0 libraries
for lib in common api status languages security; do
    lib_file="${RESOURCE_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file"
done

# Test configuration
TEST_FAILED=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_test() {
    local status="$1"
    local test_name="$2"
    local details="${3:-}"
    
    case "$status" in
        PASS)
            echo -e "  ${GREEN}✓${NC} $test_name"
            [[ -n "$details" ]] && echo "    $details"
            ;;
        FAIL)
            echo -e "  ${RED}✗${NC} $test_name"
            [[ -n "$details" ]] && echo -e "    ${RED}$details${NC}"
            TEST_FAILED=1
            ;;
        INFO)
            echo -e "  ${YELLOW}ℹ${NC} $test_name"
            [[ -n "$details" ]] && echo "    $details"
            ;;
    esac
}

test_config_functions() {
    log_test "INFO" "Testing configuration functions..."
    
    # Test judge0::export_config
    if declare -f judge0::export_config &>/dev/null; then
        judge0::export_config
        
        # Check required variables are set
        if [[ -n "${JUDGE0_PORT:-}" ]]; then
            log_test "PASS" "judge0::export_config" "Port configured: $JUDGE0_PORT"
        else
            log_test "FAIL" "judge0::export_config" "JUDGE0_PORT not set"
        fi
        
        if [[ -n "${JUDGE0_DATA_DIR:-}" ]]; then
            log_test "PASS" "Data directory config" "$JUDGE0_DATA_DIR"
        else
            log_test "FAIL" "Data directory config" "JUDGE0_DATA_DIR not set"
        fi
    else
        log_test "FAIL" "judge0::export_config" "Function not found"
    fi
}

test_api_functions() {
    log_test "INFO" "Testing API library functions..."
    
    # Test judge0::api::get_base_url
    if declare -f judge0::api::get_base_url &>/dev/null; then
        local base_url=$(judge0::api::get_base_url 2>/dev/null || echo "")
        if [[ "$base_url" == "http://localhost:${JUDGE0_PORT:-2358}" ]]; then
            log_test "PASS" "judge0::api::get_base_url" "$base_url"
        else
            log_test "FAIL" "judge0::api::get_base_url" "Invalid URL: $base_url"
        fi
    else
        log_test "FAIL" "judge0::api::get_base_url" "Function not found"
    fi
    
    # Test judge0::api::check_health
    if declare -f judge0::api::check_health &>/dev/null; then
        if judge0::api::check_health &>/dev/null; then
            log_test "PASS" "judge0::api::check_health" "Health check passed"
        else
            log_test "FAIL" "judge0::api::check_health" "Health check failed"
        fi
    else
        log_test "FAIL" "judge0::api::check_health" "Function not found"
    fi
}

test_status_functions() {
    log_test "INFO" "Testing status library functions..."
    
    # Test judge0::status::get_container_status
    if declare -f judge0::status::get_container_status &>/dev/null; then
        local status=$(judge0::status::get_container_status 2>/dev/null || echo "")
        if [[ -n "$status" ]]; then
            log_test "PASS" "judge0::status::get_container_status" "Retrieved container status"
        else
            log_test "INFO" "judge0::status::get_container_status" "No status available"
        fi
    else
        log_test "FAIL" "judge0::status::get_container_status" "Function not found"
    fi
    
    # Test judge0::status::is_healthy
    if declare -f judge0::status::is_healthy &>/dev/null; then
        if judge0::status::is_healthy &>/dev/null; then
            log_test "PASS" "judge0::status::is_healthy" "Service is healthy"
        else
            log_test "INFO" "judge0::status::is_healthy" "Service not healthy"
        fi
    else
        log_test "FAIL" "judge0::status::is_healthy" "Function not found"
    fi
}

test_language_functions() {
    log_test "INFO" "Testing language library functions..."
    
    # Test judge0::languages::list
    if declare -f judge0::languages::list &>/dev/null; then
        local languages=$(judge0::languages::list 2>/dev/null || echo "")
        if [[ -n "$languages" ]]; then
            local count=$(echo "$languages" | wc -l)
            if [[ $count -gt 20 ]]; then
                log_test "PASS" "judge0::languages::list" "Found $count languages"
            else
                log_test "FAIL" "judge0::languages::list" "Only $count languages (expected >20)"
            fi
        else
            log_test "FAIL" "judge0::languages::list" "No languages returned"
        fi
    else
        log_test "FAIL" "judge0::languages::list" "Function not found"
    fi
    
    # Test judge0::languages::get_id
    if declare -f judge0::languages::get_id &>/dev/null; then
        local python_id=$(judge0::languages::get_id "python" 2>/dev/null || echo "")
        if [[ "$python_id" == "92" ]]; then
            log_test "PASS" "judge0::languages::get_id" "Python ID: $python_id"
        else
            log_test "FAIL" "judge0::languages::get_id" "Wrong Python ID: $python_id"
        fi
    else
        log_test "INFO" "judge0::languages::get_id" "Function not found"
    fi
}

test_security_functions() {
    log_test "INFO" "Testing security library functions..."
    
    # Test judge0::security::validate_code
    if declare -f judge0::security::validate_code &>/dev/null; then
        # Test safe code
        if judge0::security::validate_code 'print("hello")' &>/dev/null; then
            log_test "PASS" "judge0::security::validate_code" "Safe code accepted"
        else
            log_test "INFO" "judge0::security::validate_code" "Safe code flagged"
        fi
        
        # Test potentially dangerous code
        if ! judge0::security::validate_code 'import os; os.system("rm -rf /")' &>/dev/null; then
            log_test "PASS" "Dangerous code detection" "Dangerous code flagged"
        else
            log_test "INFO" "Dangerous code detection" "Dangerous code not flagged"
        fi
    else
        log_test "INFO" "judge0::security::validate_code" "Function not found"
    fi
    
    # Test judge0::security::check_limits
    if declare -f judge0::security::check_limits &>/dev/null; then
        if judge0::security::check_limits &>/dev/null; then
            log_test "PASS" "judge0::security::check_limits" "Resource limits configured"
        else
            log_test "FAIL" "judge0::security::check_limits" "Resource limits not set"
        fi
    else
        log_test "INFO" "judge0::security::check_limits" "Function not found"
    fi
}

test_cli_commands() {
    log_test "INFO" "Testing CLI command structure..."
    
    # Test help command
    local help_output=$(resource-judge0 help 2>&1 || echo "")
    if echo "$help_output" | grep -q "USAGE:"; then
        log_test "PASS" "CLI help command" "Help text displayed"
    else
        log_test "FAIL" "CLI help command" "No help text"
    fi
    
    # Test info command
    local info_output=$(resource-judge0 info 2>&1 || echo "")
    if echo "$info_output" | grep -q "Judge0\|judge0"; then
        log_test "PASS" "CLI info command" "Info displayed"
    else
        log_test "FAIL" "CLI info command" "No info output"
    fi
    
    # Test status command
    local status_output=$(resource-judge0 status 2>&1 || echo "")
    if echo "$status_output" | grep -q "Status\|Running\|Stopped"; then
        log_test "PASS" "CLI status command" "Status displayed"
    else
        log_test "INFO" "CLI status command" "Limited status output"
    fi
    
    # Test manage subcommands
    local manage_help=$(resource-judge0 manage --help 2>&1 || echo "")
    if echo "$manage_help" | grep -q "install\|start\|stop"; then
        log_test "PASS" "CLI manage group" "Subcommands available"
    else
        log_test "FAIL" "CLI manage group" "Missing subcommands"
    fi
    
    # Test test subcommands
    local test_help=$(resource-judge0 test --help 2>&1 || echo "")
    if echo "$test_help" | grep -q "smoke\|integration\|unit"; then
        log_test "PASS" "CLI test group" "Test phases available"
    else
        log_test "FAIL" "CLI test group" "Missing test phases"
    fi
}

test_error_handling() {
    log_test "INFO" "Testing error handling..."
    
    # Test invalid command
    local invalid_cmd=$(resource-judge0 invalid_command 2>&1 || true)
    if echo "$invalid_cmd" | grep -qi "error\|unknown\|invalid"; then
        log_test "PASS" "Invalid command handling" "Proper error message"
    else
        log_test "FAIL" "Invalid command handling" "No error for invalid command"
    fi
    
    # Test missing parameters
    if declare -f judge0::api::submit_code &>/dev/null; then
        local missing_params=$(judge0::api::submit_code 2>&1 || true)
        if echo "$missing_params" | grep -qi "error\|required\|missing"; then
            log_test "PASS" "Missing parameter handling" "Proper error message"
        else
            log_test "INFO" "Missing parameter handling" "No parameter validation"
        fi
    fi
}

main() {
    echo -e "\n${GREEN}Running Judge0 Unit Tests${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Run all unit tests
    test_config_functions
    test_api_functions
    test_status_functions
    test_language_functions
    test_security_functions
    test_cli_commands
    test_error_handling
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ $TEST_FAILED -eq 0 ]]; then
        echo -e "${GREEN}✅ All unit tests passed${NC}\n"
        exit 0
    else
        echo -e "${RED}❌ Some unit tests failed${NC}\n"
        exit 1
    fi
}

main "$@"