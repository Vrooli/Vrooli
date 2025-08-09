#!/usr/bin/env bash
# Core network diagnostics functionality
# Simplified and testable version of network diagnostics
set -eo pipefail

# Script directory
LIB_NETWORK_DIAGNOSTICS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${LIB_NETWORK_DIAGNOSTICS_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"

# Track test results
declare -A CORE_TEST_RESULTS
CORE_CRITICAL_FAILURE=false

# Test wrapper function that respects mocks in test environment
core_run_test() {
    local test_name="$1"
    local test_command="$2"
    local is_critical="${3:-false}"
    
    log::info "Testing: $test_name"
    
    # Use explicit if-else to avoid script exit on test failure
    set +e  # Temporarily disable exit on error
    
    # In test environment (when BATS_TEST_DIRNAME is set), respect function mocks
    if [[ -n "${BATS_TEST_DIRNAME:-}" ]]; then
        # For tests: execute in current shell context to respect mocks
        eval "$test_command" >/dev/null 2>&1
    else
        # For production: use normal execution
        eval "$test_command" >/dev/null 2>&1
    fi
    
    local test_result=$?
    set -e  # Re-enable exit on error
    
    if [[ $test_result -eq 0 ]]; then
        CORE_TEST_RESULTS["$test_name"]="PASS"
        log::success "  ✓ $test_name"
        return 0
    else
        CORE_TEST_RESULTS["$test_name"]="FAIL"
        log::error "  ✗ $test_name"
        
        if [[ "$is_critical" == "true" ]]; then
            CORE_CRITICAL_FAILURE=true
        fi
        return 0  # Don't exit the script, just continue testing
    fi
}

# Simplified core diagnostics function
network_diagnostics_core::run() {
    log::header "Running Core Network Diagnostics"
    
    # Reset test results
    unset CORE_TEST_RESULTS
    declare -A CORE_TEST_RESULTS
    CORE_CRITICAL_FAILURE=false
    
    # 1. Basic connectivity tests
    log::subheader "Basic Connectivity"
    core_run_test "IPv4 ping to 8.8.8.8" "ping -4 -c 1 -W 2 8.8.8.8" true
    core_run_test "IPv4 ping to google.com" "ping -4 -c 1 -W 2 google.com"
    
    # 2. DNS resolution tests
    log::subheader "DNS Resolution"
    core_run_test "DNS lookup (getent)" "getent hosts google.com"
    
    # 3. TCP connectivity tests
    log::subheader "TCP Connectivity"
    core_run_test "TCP port 443 (HTTPS)" "nc -zv -w 2 google.com 443"
    
    # 4. Basic HTTPS test
    log::subheader "HTTPS Tests"
    if command -v curl >/dev/null 2>&1; then
        core_run_test "HTTPS to google.com" "timeout 8 curl -s --http1.1 --connect-timeout 5 https://www.google.com"
        
        # TLS handshake timing test
        if command -v date >/dev/null 2>&1; then
            log::info "Testing TLS handshake timing..."
            local start_time=$(date +%s)
            if timeout 8 curl -s --http1.1 --connect-timeout 5 https://www.google.com >/dev/null 2>&1; then
                local end_time=$(date +%s)
                local duration=$((end_time - start_time))
                log::info "TLS handshake completed in ${duration}s"
            else
                log::warning "TLS handshake failed or timed out"
            fi
        fi
    fi
    
    # 5. Local service tests
    log::subheader "Local Services"
    core_run_test "Localhost port 9200 (SearXNG)" "nc -zv -w 1 localhost 9200 2>/dev/null"
    
    # Count test results
    local total_tests=${#CORE_TEST_RESULTS[@]}
    local passed_tests=0
    local failed_tests=0
    
    for test_result in "${CORE_TEST_RESULTS[@]}"; do
        if [[ "$test_result" == "PASS" ]]; then
            passed_tests=$((passed_tests + 1))
        else
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    # Report results
    log::subheader "Core Test Results"
    log::info "Total tests: $total_tests"
    log::info "Passed: $passed_tests"
    log::info "Failed: $failed_tests"
    
    # Analyze results and provide diagnosis
    if [[ "$CORE_CRITICAL_FAILURE" == "true" ]]; then
        log::error "[ERROR] Critical network issues detected"
        log::info "Basic connectivity is broken - setup cannot continue"
        return "${ERROR_NO_INTERNET}"
    fi
    
    # Look for specific patterns that indicate partial TLS issues
    if [[ "${CORE_TEST_RESULTS["IPv4 ping to google.com"]:-}" == "PASS" ]] && \
       [[ "${CORE_TEST_RESULTS["DNS lookup (getent)"]:-}" == "PASS" ]] && \
       [[ "${CORE_TEST_RESULTS["TCP port 443 (HTTPS)"]:-}" == "PASS" ]] && \
       [[ "${CORE_TEST_RESULTS["HTTPS to google.com"]:-}" == "FAIL" ]]; then
        log::warning "PARTIAL TLS ISSUE: TCP connectivity works but HTTPS fails"
        log::info "This suggests TLS version negotiation or cipher suite issues"
        
        # Check TCP segmentation offload (simplified for testing)
        if command -v ethtool >/dev/null 2>&1; then
            log::info "TCP segmentation offload: checking available"
        fi
    fi
    
    if [[ $failed_tests -gt 0 ]]; then
        log::warning "Network issues detected: $failed_tests tests failed"
        log::info "Automatic Network Optimizations may be needed"
        log::info "Testing alternative DNS servers or network configuration changes"
        return 1
    fi
    
    log::success "All core network tests passed!"
    return 0
}

# Export the main function for external use
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Script was executed directly
    network_diagnostics_core::run "$@"
fi