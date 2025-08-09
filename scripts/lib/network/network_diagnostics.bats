#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Load test helpers
load "../../__test/helpers/bats-support/load"
load "../../__test/helpers/bats-assert/load"

# Load mocks
load "../../__test/fixtures/mocks/logs.sh"
load "../../__test/fixtures/mocks/commands.bash"

# Script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/network_diagnostics.sh"

# Source necessary dependencies that the script needs
# Note: These need to be sourced here so functions are available
source "${BATS_TEST_DIRNAME}/../utils/var.sh"
source "${var_LOG_FILE}"
source "${var_EXIT_CODES_FILE}"
source "${var_FLOW_FILE}"

setup() {
    # Initialize mocks
    mock::cleanup_logs "" || true
    
    # Set up test environment
    export SUDO_MODE="skip"  # Skip sudo operations in tests
    export VROOLI_TEST_MODE="true"  # Enable test mode
    export BATS_TEST_DIRNAME="$BATS_TEST_DIRNAME"  # Ensure test mode is recognized
    
    # Mock network commands to avoid real network calls
    setup_network_mocks
}

teardown() {
    # Clean up mocks
    mock::cleanup_logs "" || true
    
    # Unset environment variables
    unset SUDO_MODE VROOLI_TEST_MODE BATS_TEST_DIRNAME
}

# Set up network command mocks
setup_network_mocks() {
    # Mock ping command
    ping() {
        case "$*" in
            *"8.8.8.8"*)
                return 0  # Success for Google DNS
                ;;
            *"google.com"*)
                return 0  # Success for Google
                ;;
            *)
                return 1  # Fail for unknown hosts
                ;;
        esac
    }
    export -f ping
    
    # Mock getent command
    getent() {
        case "$*" in
            *"google.com"*)
                echo "142.250.191.14 google.com"
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    export -f getent
    
    # Mock nc (netcat) command
    nc() {
        case "$*" in
            *"google.com 80"*)
                return 0  # HTTP port open
                ;;
            *"google.com 443"*)
                return 0  # HTTPS port open
                ;;
            *"localhost 9200"*)
                return 1  # SearXNG not running in tests
                ;;
            *)
                return 1
                ;;
        esac
    }
    export -f nc
    
    # Mock curl command
    curl() {
        case "$*" in
            *"google.com"*)
                echo "<html>Mock Google response</html>"
                return 0
                ;;
            *"github.com"*)
                echo "<html>Mock GitHub response</html>"
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    export -f curl
    
    # Mock timeout command (just run the command without timeout)
    timeout() {
        local duration="$1"
        shift
        "$@"
    }
    export -f timeout
    
    # Mock ip command
    ip() {
        case "$*" in
            *"route"*)
                echo "default via 192.168.1.1 dev eth0"
                ;;
            *)
                echo "Mock ip output"
                ;;
        esac
    }
    export -f ip
    
    # Mock ethtool command
    ethtool() {
        case "$*" in
            *"-k"*)
                echo "tcp-segmentation-offload: off"
                ;;
            *"-K"*)
                return 0  # Success when setting options
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f ethtool
}

# Tests for run_test function
@test "run_test passes successful tests" {
    source "$SCRIPT_PATH"
    
    run run_test "Test Success" "true" "false"
    assert_success
    assert_output --partial "✓ Test Success"
}

@test "run_test fails unsuccessful tests but continues" {
    source "$SCRIPT_PATH"
    
    run run_test "Test Failure" "false" "false"
    assert_success  # Function returns 0 even on test failure
    assert_output --partial "✗ Test Failure"
}

@test "run_test marks critical failures" {
    source "$SCRIPT_PATH"
    
    # Reset global state
    CRITICAL_FAILURE=false
    
    run_test "Critical Test" "false" "true"
    
    # Check that CRITICAL_FAILURE was set
    [[ "$CRITICAL_FAILURE" == "true" ]]
}

# Tests for network_diagnostics::run function
@test "network_diagnostics::run performs basic connectivity tests" {
    source "$SCRIPT_PATH"
    
    run network_diagnostics::run
    assert_success
    assert_output --partial "Running Comprehensive Network Diagnostics"
    assert_output --partial "Basic Connectivity"
    assert_output --partial "IPv4 ping to 8.8.8.8"
    assert_output --partial "IPv4 ping to google.com"
}

@test "network_diagnostics::run performs DNS resolution tests" {
    source "$SCRIPT_PATH"
    
    run network_diagnostics::run
    assert_success
    assert_output --partial "DNS Resolution"
    assert_output --partial "DNS lookup (getent)"
}

@test "network_diagnostics::run performs TCP connectivity tests" {
    source "$SCRIPT_PATH"
    
    run network_diagnostics::run
    assert_success
    assert_output --partial "TCP Connectivity"
    assert_output --partial "TCP port 80 (HTTP)"
    assert_output --partial "TCP port 443 (HTTPS)"
}

@test "network_diagnostics::run performs HTTPS tests when curl is available" {
    source "$SCRIPT_PATH"
    
    run network_diagnostics::run
    assert_success
    assert_output --partial "HTTPS Tests"
    assert_output --partial "HTTPS to google.com"
    assert_output --partial "HTTPS to github.com"
}

@test "network_diagnostics::run tests local services" {
    source "$SCRIPT_PATH"
    
    run network_diagnostics::run
    assert_success
    assert_output --partial "Local Services"
    assert_output --partial "Localhost port 9200 (SearXNG)"
}

@test "network_diagnostics::run handles critical failures" {
    source "$SCRIPT_PATH"
    
    # Mock ping to fail for critical test
    ping() {
        case "$*" in
            *"8.8.8.8"*)
                return 1  # Fail critical test
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f ping
    
    # Run in subshell to capture exit code
    run bash -c "source '$SCRIPT_PATH' && network_diagnostics::run"
    assert_failure
    assert_output --partial "Critical network issues detected"
}

@test "network_diagnostics::run continues despite non-critical failures" {
    source "$SCRIPT_PATH"
    
    # Mock nc to always fail (non-critical tests)
    nc() { return 1; }
    export -f nc
    
    run network_diagnostics::run
    assert_success
    assert_output --partial "Network issues detected"
    assert_output --partial "Proceeding despite network issues"
}

@test "network_diagnostics::run reports success when all tests pass" {
    source "$SCRIPT_PATH"
    
    run network_diagnostics::run
    assert_success
    assert_output --partial "All network tests passed!"
}

# Tests for stub functions (backward compatibility)
@test "network_diagnostics::verbose_https_debug delegates to analysis module" {
    source "$SCRIPT_PATH"
    
    # Mock the analysis function
    network_diagnostics_analysis::verbose_https_debug() {
        echo "Mock verbose HTTPS debug"
        return 0
    }
    export -f network_diagnostics_analysis::verbose_https_debug
    
    run network_diagnostics::verbose_https_debug
    assert_success
    assert_output "Mock verbose HTTPS debug"
}

@test "network_diagnostics::check_ufw_blocking delegates to fixes module" {
    source "$SCRIPT_PATH"
    
    # Mock the fixes function
    network_diagnostics_fixes::fix_ufw_blocking() {
        echo "Mock UFW blocking check"
        return 0
    }
    export -f network_diagnostics_fixes::fix_ufw_blocking
    
    run network_diagnostics::check_ufw_blocking
    assert_success
    assert_output "Mock UFW blocking check"
}

@test "network_diagnostics::enhanced_diagnosis runs multiple checks" {
    source "$SCRIPT_PATH"
    
    # Mock all the analysis functions
    network_diagnostics_analysis::check_ipv4_vs_ipv6() { echo "IPv4 vs IPv6 check"; }
    network_diagnostics_analysis::check_ip_preference() { echo "IP preference check"; }
    network_diagnostics_tcp::check_tcp_settings() { echo "TCP settings check"; }
    network_diagnostics_tcp::check_pmtu_status() { echo "PMTU status check"; }
    network_diagnostics_analysis::check_time_sync() { echo "Time sync check"; }
    export -f network_diagnostics_analysis::check_ipv4_vs_ipv6
    export -f network_diagnostics_analysis::check_ip_preference
    export -f network_diagnostics_tcp::check_tcp_settings
    export -f network_diagnostics_tcp::check_pmtu_status
    export -f network_diagnostics_analysis::check_time_sync
    
    run network_diagnostics::enhanced_diagnosis
    assert_success
    assert_output --partial "Running enhanced diagnosis"
    assert_output --partial "IPv4 vs IPv6 check"
    assert_output --partial "IP preference check"
    assert_output --partial "TCP settings check"
    assert_output --partial "PMTU status check"
    assert_output --partial "Time sync check"
}

# Integration test
@test "script runs network_diagnostics::run when executed directly" {
    run bash "$SCRIPT_PATH"
    assert_success
    assert_output --partial "Running Comprehensive Network Diagnostics"
}

# Test SUDO_MODE configuration
@test "script sets appropriate SUDO_MODE when not set" {
    # Test in standalone context
    export VROOLI_CONTEXT="standalone"
    unset SUDO_MODE
    
    source "$SCRIPT_PATH"
    
    [[ "${SUDO_MODE}" == "skip" ]]
}

@test "script respects existing SUDO_MODE setting" {
    export SUDO_MODE="error"
    
    source "$SCRIPT_PATH"
    
    [[ "${SUDO_MODE}" == "error" ]]
}

# Test TEST_RESULTS tracking
@test "TEST_RESULTS tracks test outcomes" {
    source "$SCRIPT_PATH"
    
    # Run a test that should pass
    run_test "Mock Pass Test" "true" "false"
    [[ "${TEST_RESULTS["Mock Pass Test"]}" == "PASS" ]]
    
    # Run a test that should fail
    run_test "Mock Fail Test" "false" "false"
    [[ "${TEST_RESULTS["Mock Fail Test"]}" == "FAIL" ]]
}