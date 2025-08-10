#!/usr/bin/env bash
# Comprehensive network diagnostics script (modular version)
# Tests various network layers and protocols to identify connectivity issues
set -eo pipefail

# Source var.sh first with relative path
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/diagnostics/network_diagnostics_tcp.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/diagnostics/network_diagnostics_analysis.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/diagnostics/network_diagnostics_fixes.sh"

# Set appropriate SUDO_MODE if not already set
if [[ -z "${SUDO_MODE:-}" ]]; then
    # In standalone mode, default to skip to avoid blocking on sudo
    if [[ "${VROOLI_CONTEXT:-}" == "standalone" ]]; then
        export SUDO_MODE="skip"
    elif [[ $EUID -eq 0 ]] || sudo -n true 2>/dev/null; then
        export SUDO_MODE="error"
    else
        export SUDO_MODE="skip"
    fi
fi

# Track test results for backward compatibility
declare -A TEST_RESULTS
CRITICAL_FAILURE=false

# Simple test wrapper for backward compatibility
run_test() {
    local test_name="$1"
    local test_command="$2"
    local is_critical="${3:-false}"
    
    log::info "Testing: $test_name"
    
    set +e
    # In test environment, respect function mocks and TEST_MODE
    if [[ -n "${BATS_TEST_DIRNAME:-}" ]]; then
        # In BATS test environment, always run the command to respect mocks
        eval "$test_command" >/dev/null 2>&1
        local test_result=$?
    elif [[ "${VROOLI_TEST_MODE:-false}" == "true" ]]; then
        # In pure test mode, mock success for non-critical tests
        if [[ "$is_critical" != "true" ]]; then
            local test_result=0  # Mock success
        else
            eval "$test_command" >/dev/null 2>&1
            local test_result=$?
        fi
    else
        eval "$test_command" >/dev/null 2>&1
        local test_result=$?
    fi
    set -e
    
    if [[ $test_result -eq 0 ]]; then
        TEST_RESULTS["$test_name"]="PASS"
        log::success "  ✓ $test_name"
        return 0
    else
        TEST_RESULTS["$test_name"]="FAIL"
        log::error "  ✗ $test_name"
        
        if [[ "$is_critical" == "true" ]]; then
            CRITICAL_FAILURE=true
        fi
        return 0
    fi
}

# Main diagnostics function for backward compatibility
network_diagnostics::run() {
    log::header "Running Comprehensive Network Diagnostics"
    
    # Reset test results
    unset TEST_RESULTS
    declare -A TEST_RESULTS
    CRITICAL_FAILURE=false
    
    # 1. Basic connectivity tests
    log::subheader "Basic Connectivity"
    run_test "IPv4 ping to 8.8.8.8" "ping -4 -c 1 -W 2 8.8.8.8" true
    run_test "IPv4 ping to google.com" "ping -4 -c 1 -W 2 google.com"
    
    # 2. DNS resolution tests
    log::subheader "DNS Resolution"
    run_test "DNS lookup (getent)" "getent hosts google.com"
    
    # 3. TCP connectivity tests
    log::subheader "TCP Connectivity"
    run_test "TCP port 80 (HTTP)" "nc -zv -w 2 google.com 80"
    run_test "TCP port 443 (HTTPS)" "nc -zv -w 2 google.com 443"
    
    # 4. HTTPS tests
    log::subheader "HTTPS Tests"
    if command -v curl >/dev/null 2>&1; then
        run_test "HTTPS to google.com" "timeout 8 curl -s --http1.1 --connect-timeout 5 https://www.google.com"
        run_test "HTTPS to github.com" "timeout 8 curl -s --http1.1 --connect-timeout 5 https://github.com"
        
        # Test TLS handshake timing
        log::info "Testing TLS handshake timing..."
        if timeout 8 curl -s --http1.1 --connect-timeout 5 https://www.google.com >/dev/null 2>&1; then
            log::info "TLS handshake completed successfully"
        else
            log::warning "TLS handshake failed or timed out"
        fi
    fi
    
    # 5. Local service tests
    log::subheader "Local Services"
    run_test "Localhost port 9200 (SearXNG)" "nc -zv -w 1 localhost 9200 2>/dev/null"
    
    # Check for critical failure
    if [[ "$CRITICAL_FAILURE" == "true" ]]; then
        log::error "[ERROR] Critical network issues detected"
        return "${ERROR_NO_INTERNET}"
    fi
    
    # Count failures
    local failed_tests=0
    for result in "${TEST_RESULTS[@]}"; do
        if [[ "$result" == "FAIL" ]]; then
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    # Check for specific patterns (for test compatibility)
    local has_basic_connectivity="${TEST_RESULTS["IPv4 ping to google.com"]:-FAIL}"
    local has_dns="${TEST_RESULTS["DNS lookup (getent)"]:-FAIL}"
    local has_tcp="${TEST_RESULTS["TCP port 443 (HTTPS)"]:-FAIL}"
    local has_https_google="${TEST_RESULTS["HTTPS to google.com"]:-FAIL}"
    
    
    # Run targeted diagnostics for specific failure patterns
    if [[ "$has_basic_connectivity" == "PASS" ]] && \
       [[ "$has_dns" == "PASS" ]] && \
       [[ "$has_tcp" == "PASS" ]] && \
       [[ "$has_https_google" == "FAIL" ]]; then
        log::warning "PARTIAL TLS ISSUE: TCP connectivity works but HTTPS fails"
        network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    elif [[ "$has_dns" == "FAIL" ]]; then
        network_diagnostics_analysis::diagnose_connection_failure "DNS lookup (getent)" "google.com"
    elif [[ "$has_basic_connectivity" == "FAIL" ]]; then
        network_diagnostics_analysis::diagnose_connection_failure "IPv4 ping to google.com" "google.com"
    fi
    
    # Summary and guidance for failures
    if [[ $failed_tests -gt 0 ]]; then
        log::warning "Network issues detected: $failed_tests tests failed"
        log::info "Automatic network fixes will be attempted by setup.sh"
        
        # Don't fail just because of network issues - many apps work offline
        log::info "Proceeding despite network issues (many apps work offline)"
        return 0
    fi
    
    log::success "All network tests passed!"
    return 0
}


# Export main function for external use
export -f network_diagnostics::run

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    network_diagnostics::run "$@"
fi