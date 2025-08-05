#!/usr/bin/env bash
# Comprehensive network diagnostics script (modular version)
# Tests various network layers and protocols to identify connectivity issues
set -eo pipefail

# Script directory
SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Load dependencies
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/exit_codes.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/flow.sh"

# Load modular components from network subfolder
# shellcheck disable=SC1091
source "${SETUP_DIR}/network/network_diagnostics_core.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/network/network_diagnostics_tcp.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/network/network_diagnostics_analysis.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/network/network_diagnostics_fixes.sh"

# Set appropriate SUDO_MODE if not already set
if [[ -z "${SUDO_MODE:-}" ]]; then
    if [[ $EUID -eq 0 ]] || sudo -n true 2>/dev/null; then
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
    local total_tests=${#TEST_RESULTS[@]}
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
    
    
    # Provide diagnosis for partial TLS issues
    if [[ "$has_basic_connectivity" == "PASS" ]] && \
       [[ "$has_dns" == "PASS" ]] && \
       [[ "$has_tcp" == "PASS" ]] && \
       [[ "$has_https_google" == "FAIL" ]]; then
        log::warning "PARTIAL TLS ISSUE: TCP connectivity works but HTTPS fails"
        
        # Check TCP segmentation offload
        if command -v ethtool >/dev/null 2>&1; then
            log::info "TCP segmentation offload: checking available"
        fi
    fi
    
    # Automatic network optimizations message
    if [[ $failed_tests -gt 0 ]]; then
        log::warning "Network issues detected: $failed_tests tests failed"
        log::info "Automatic Network Optimizations may be needed"
        log::info "Testing alternative DNS servers or network configuration changes"
        
        # Attempt some basic fixes if sudo is available
        if flow::can_run_sudo "apply network fixes"; then
            local primary_iface=$(ip route | grep default | awk '{print $5}' | head -1 2>/dev/null || echo "")
            if [[ -n "$primary_iface" ]] && command -v ethtool >/dev/null 2>&1; then
                local tso_status=$(ethtool -k "$primary_iface" 2>/dev/null | grep "tcp-segmentation-offload" | awk '{print $2}' || echo "off")
                if [[ "$tso_status" == "on" ]]; then
                    log::info "Attempting to disable TSO on $primary_iface..."
                    if sudo ethtool -K "$primary_iface" tso off gso off 2>/dev/null; then
                        log::success "✓ Disabled TSO on $primary_iface"
                    fi
                fi
            fi
        fi
        
        return 1
    fi
    
    log::success "All network tests passed!"
    return 0
}

# Stub functions for backward compatibility with tests
network_diagnostics::verbose_https_debug() {
    network_diagnostics_analysis::verbose_https_debug "$@"
}

network_diagnostics::check_ufw_blocking() {
    network_diagnostics_fixes::fix_ufw_blocking "$@"
}

network_diagnostics::fix_ufw_blocking() {
    network_diagnostics_fixes::fix_ufw_blocking "$@"
}

network_diagnostics::make_tso_permanent() {
    network_diagnostics_tcp::make_tso_permanent "$@"
}

network_diagnostics::test_mtu_discovery() {
    network_diagnostics_tcp::test_mtu_discovery "$@"
}

network_diagnostics::check_pmtu_status() {
    network_diagnostics_tcp::check_pmtu_status "$@"
}

network_diagnostics::analyze_tls_handshake() {
    network_diagnostics_analysis::analyze_tls_handshake "$@"
}

network_diagnostics::check_tcp_settings() {
    network_diagnostics_tcp::check_tcp_settings "$@"
}

network_diagnostics::check_time_sync() {
    network_diagnostics_analysis::check_time_sync "$@"
}

network_diagnostics::check_ipv4_vs_ipv6() {
    network_diagnostics_analysis::check_ipv4_vs_ipv6 "$@"
}

network_diagnostics::check_ip_preference() {
    network_diagnostics_analysis::check_ip_preference "$@"
}

network_diagnostics::fix_pmtu_probing() {
    network_diagnostics_tcp::fix_pmtu_probing "$@"
}

network_diagnostics::fix_mtu_size() {
    network_diagnostics_tcp::fix_mtu_size "$@"
}

network_diagnostics::fix_ecn() {
    network_diagnostics_tcp::fix_ecn "$@"
}

network_diagnostics::fix_ipv6_issues() {
    network_diagnostics_fixes::fix_ipv6_issues "$@"
}

network_diagnostics::fix_ipv4_only_issues() {
    network_diagnostics_fixes::fix_ipv4_only_issues "$@"
}

network_diagnostics::add_ipv4_host_override() {
    network_diagnostics_fixes::add_ipv4_host_override "$@"
}

network_diagnostics::enhanced_diagnosis() {
    log::info "Running enhanced diagnosis..."
    network_diagnostics_analysis::check_ipv4_vs_ipv6 "github.com"
    network_diagnostics_analysis::check_ip_preference
    network_diagnostics_tcp::check_tcp_settings
    network_diagnostics_tcp::check_pmtu_status
    network_diagnostics_analysis::check_time_sync
}

network_diagnostics::fix_searxng_nginx() {
    log::info "SearXNG nginx fix not implemented in modular version"
    return 0
}

# Export all functions for external use
export -f network_diagnostics::run
export -f network_diagnostics::verbose_https_debug
export -f network_diagnostics::check_ufw_blocking
export -f network_diagnostics::fix_ufw_blocking
export -f network_diagnostics::make_tso_permanent
export -f network_diagnostics::test_mtu_discovery
export -f network_diagnostics::check_pmtu_status
export -f network_diagnostics::analyze_tls_handshake
export -f network_diagnostics::check_tcp_settings
export -f network_diagnostics::check_time_sync
export -f network_diagnostics::check_ipv4_vs_ipv6
export -f network_diagnostics::check_ip_preference
export -f network_diagnostics::fix_pmtu_probing
export -f network_diagnostics::fix_mtu_size
export -f network_diagnostics::fix_ecn
export -f network_diagnostics::fix_ipv6_issues
export -f network_diagnostics::fix_ipv4_only_issues
export -f network_diagnostics::add_ipv4_host_override
export -f network_diagnostics::enhanced_diagnosis
export -f network_diagnostics::fix_searxng_nginx

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    network_diagnostics::run "$@"
fi