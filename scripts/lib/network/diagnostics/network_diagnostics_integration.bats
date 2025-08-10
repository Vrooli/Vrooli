#!/usr/bin/env bats

# Integration tests for main network diagnostics orchestrator
# Tests the overall behavior, not implementation details

setup() {
    export BATS_TEST_DIRNAME="$BATS_TEST_DIRNAME"
    export TEST_TEMP_DIR="$(mktemp -d)"
    
    # Source the main script
    source "${BATS_TEST_DIRNAME}/network_diagnostics.sh"
    
    # Mock log functions
    log::header() { echo "[HEADER] $*"; }
    log::subheader() { echo "[SUBHEADER] $*"; }
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*"; }
    
    # Export error codes
    export ERROR_NO_INTERNET=5
}

teardown() {
    rm -rf "$TEST_TEMP_DIR"
}

# =============================================================================
# Behavior Tests - Test what users experience, not how it works
# =============================================================================

@test "integration: successful network connectivity" {
    # Mock all commands to succeed
    ping() { return 0; }
    getent() { echo "93.184.216.34 google.com"; return 0; }
    nc() { return 0; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics::run
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"All network tests passed!"* ]]
}

@test "integration: no internet connectivity" {
    # Mock critical failure - can't ping 8.8.8.8
    ping() { 
        [[ "$*" == *"8.8.8.8"* ]] && return 1 || return 0
    }
    getent() { return 1; }
    nc() { return 1; }
    curl() { return 1; }
    timeout() { shift; "$@"; }
    command() { return 0; }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics::run
    
    [ "$status" -eq 5 ]  # ERROR_NO_INTERNET
    [[ "$output" == *"Critical network issues detected"* ]]
}

@test "integration: DNS issues detected" {
    # Mock DNS failure but connectivity works
    ping() { 
        [[ "$*" == *"8.8.8.8"* ]] && return 0
        [[ "$*" == *"google.com"* ]] && return 1
    }
    getent() { return 1; }
    nc() { return 1; }
    curl() { return 1; }
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics::run
    
    # Should continue despite DNS issues
    [ "$status" -eq 0 ]
    [[ "$output" == *"✓ IPv4 ping to 8.8.8.8"* ]]
    [[ "$output" == *"✗ DNS lookup (getent)"* ]]
    [[ "$output" == *"Network issues detected"* ]]
}

@test "integration: HTTPS issues with working TCP" {
    # Mock HTTPS failure but TCP works
    ping() { return 0; }
    getent() { echo "93.184.216.34 google.com"; return 0; }
    nc() { return 0; }
    curl() { return 1; }  # HTTPS fails
    timeout() { shift; "$@"; }
    ethtool() { return 0; }
    command() { 
        case "$2" in
            curl|ethtool) return 0 ;;
            *) return 1 ;;
        esac
    }
    
    export -f ping getent nc curl timeout ethtool command
    
    run network_diagnostics::run
    
    # Should detect partial TLS issue
    [ "$status" -eq 0 ]
    [[ "$output" == *"✓ TCP port 443 (HTTPS)"* ]]
    [[ "$output" == *"✗ HTTPS to google.com"* ]]
    [[ "$output" == *"PARTIAL TLS ISSUE"* ]]
}

@test "integration: module functions are available" {
    # Source modules the same way the main script does
    source "${BATS_TEST_DIRNAME}/network_diagnostics_tcp.sh"
    source "${BATS_TEST_DIRNAME}/network_diagnostics_analysis.sh"
    source "${BATS_TEST_DIRNAME}/network_diagnostics_fixes.sh"
    
    # Now verify functions exist
    declare -f network_diagnostics_tcp::check_tcp_settings > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_analysis::diagnose_connection_failure > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_fixes::fix_ipv6_issues > /dev/null
    [ "$?" -eq 0 ]
}

@test "integration: respects test mode environment variable" {
    export VROOLI_TEST_MODE=true
    
    # In test mode, non-critical tests should pass
    ping() { return 1; }  # Would normally fail
    getent() { return 1; }
    nc() { return 1; }
    curl() { return 1; }
    timeout() { shift; "$@"; }
    command() { return 0; }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics::run
    
    # Critical test (8.8.8.8) still runs and fails
    [ "$status" -eq 5 ]
    [[ "$output" == *"Critical network issues detected"* ]]
}

@test "integration: handles curl not available" {
    # Mock curl command not found
    ping() { return 0; }
    getent() { echo "93.184.216.34 google.com"; return 0; }
    nc() { return 0; }
    curl() { return 127; }  # Command not found
    timeout() { shift; "$@"; }
    command() { 
        [[ "$2" == "curl" ]] && return 1  # curl not available
        return 0
    }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics::run
    
    # Should pass basic tests even without curl
    [ "$status" -eq 0 ]
    [[ "$output" == *"✓ IPv4 ping to google.com"* ]]
    [[ "$output" == *"✓ DNS lookup (getent)"* ]]
}

@test "integration: local service check handles failure gracefully" {
    # Mock all remote tests pass but local service fails
    ping() { return 0; }
    getent() { echo "93.184.216.34 google.com"; return 0; }
    nc() { 
        [[ "$*" == *"localhost"* ]] && return 1 || return 0
    }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics::run
    
    # Should continue despite local service failure
    [ "$status" -eq 0 ]
    [[ "$output" == *"✗ Localhost port 9200 (SearXNG)"* ]]
    [[ "$output" == *"Network issues detected"* ]]
    [[ "$output" == *"Proceeding despite network issues"* ]]
}