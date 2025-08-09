#!/usr/bin/env bats

# Tests for modular network diagnostics wrapper
# Tests the integration of all diagnostic modules

setup() {
    export BATS_TEST_DIRNAME="$BATS_TEST_DIRNAME"
    export TEST_TEMP_DIR="$(mktemp -d)"
    
    # Source the script under test
    source "${BATS_TEST_DIRNAME}/network_diagnostics_modular.sh"
    
    # Mock log functions
    log::header() { echo "[HEADER] $*"; }
    log::subheader() { echo "[SUBHEADER] $*"; }
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*"; }
    
    # Mock flow functions
    flow::can_run_sudo() { 
        echo "[FLOW] Checking sudo for: $1"
        return 1  # Default to no sudo in tests
    }
    
    # Mock core network diagnostics functions
    network_diagnostics_core::run() {
        echo "[CORE] Running core diagnostics"
        return 0  # Default success
    }
    
    network_diagnostics_tcp::check_tcp_settings() {
        echo "[TCP] Checking TCP settings"
    }
    
    network_diagnostics_tcp::check_pmtu_status() {
        echo "[TCP] Checking PMTU status"
    }
    
    network_diagnostics_tcp::make_tso_permanent() {
        echo "[TCP] Making TSO permanent for $1"
    }
    
    network_diagnostics_analysis::check_time_sync() {
        echo "[ANALYSIS] Checking time sync"
    }
    
    network_diagnostics_analysis::check_ipv4_vs_ipv6() {
        echo "[ANALYSIS] Checking IPv4 vs IPv6 for $1"
    }
    
    network_diagnostics_analysis::check_ip_preference() {
        echo "[ANALYSIS] Checking IP preference"
    }
    
    network_diagnostics_fixes::fix_dns_issues() {
        echo "[FIXES] Fixing DNS issues"
    }
    
    network_diagnostics_fixes::fix_ipv6_issues() {
        echo "[FIXES] Fixing IPv6 issues"
    }
    
    network_diagnostics_fixes::fix_ufw_blocking() {
        echo "[FIXES] Fixing UFW blocking"
    }
    
    export ERROR_NO_INTERNET=5
    
    export -f log::header log::subheader log::info log::success log::warning log::error
    export -f flow::can_run_sudo
    export -f network_diagnostics_core::run
    export -f network_diagnostics_tcp::check_tcp_settings network_diagnostics_tcp::check_pmtu_status
    export -f network_diagnostics_tcp::make_tso_permanent
    export -f network_diagnostics_analysis::check_time_sync network_diagnostics_analysis::check_ipv4_vs_ipv6
    export -f network_diagnostics_analysis::check_ip_preference
    export -f network_diagnostics_fixes::fix_dns_issues network_diagnostics_fixes::fix_ipv6_issues
    export -f network_diagnostics_fixes::fix_ufw_blocking
    
    # Track which commands were called
    export COMMANDS_CALLED=()
}

teardown() {
    rm -rf "$TEST_TEMP_DIR"
    unset COMMANDS_CALLED
}

# Helper function to track command calls
track_command() {
    COMMANDS_CALLED+=("$1")
    return "${2:-0}"
}

# =============================================================================
# Tests for main modular diagnostics function
# =============================================================================

@test "modular run - successful all tests pass" {
    # Override core function to succeed
    network_diagnostics_core::run() {
        echo "[CORE] All core tests passed"
        return 0
    }
    
    export -f network_diagnostics_core::run
    
    run network_diagnostics::run
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Running Comprehensive Network Diagnostics (Modular)"* ]]
    [[ "$output" == *"Starting core network tests"* ]]
    [[ "$output" == *"All core tests passed"* ]]
    [[ "$output" == *"Advanced Network Analysis"* ]]
    [[ "$output" == *"Checking TCP settings"* ]]
    [[ "$output" == *"Checking PMTU status"* ]]
    [[ "$output" == *"Checking time sync"* ]]
    [[ "$output" == *"Checking IPv4 vs IPv6 for github.com"* ]]
    [[ "$output" == *"Checking IP preference"* ]]
    [[ "$output" == *"✅ All core network tests passed"* ]]
}

@test "modular run - handles critical network failure" {
    # Override core function to return critical failure
    network_diagnostics_core::run() {
        echo "[CORE] Critical network failure"
        return 5  # ERROR_NO_INTERNET
    }
    
    export -f network_diagnostics_core::run
    
    run network_diagnostics::run
    
    [ "$status" -eq 5 ]
    [[ "$output" == *"Critical network failure detected. Cannot continue."* ]]
    # Should not proceed to advanced analysis
    [[ "$output" != *"Advanced Network Analysis"* ]]
}

@test "modular run - applies automatic fixes for minor issues" {
    # Override core function to return minor failure
    network_diagnostics_core::run() {
        echo "[CORE] Some tests failed"
        return 1
    }
    
    # Mock commands for TSO detection and fixes
    ip() {
        if [[ "$*" == *"route"* ]]; then
            echo "default via 192.168.1.1 dev eth0"
        fi
        return 0
    }
    
    ethtool() {
        if [[ "$*" == *"-k eth0"* ]]; then
            echo "tcp-segmentation-offload: on"
        fi
        return 0
    }
    
    command() {
        case "$*" in
            *"ethtool"*)
                return 0
                ;;
            *"ufw"*)
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    # Mock ping for IPv6 issues detection
    ping() {
        case "$*" in
            *"-6"*)
                return 1  # IPv6 fails
                ;;
            *"-4"*)
                return 0  # IPv4 works
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Mock curl for GitHub test
    curl() {
        if [[ "$*" == *"github.com"* ]]; then
            return 0  # Success after TSO fix
        fi
        return 1
    }
    
    timeout() { shift; "$@"; }
    sudo() { return 0; }
    
    export -f network_diagnostics_core::run ip ethtool command ping curl timeout sudo
    
    run network_diagnostics::run
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Attempting Automatic Fixes"* ]]
    [[ "$output" == *"TCP Segmentation Offload is enabled"* ]]
    [[ "$output" == *"IPv6 connectivity issues detected"* ]]
    [[ "$output" == *"⚠️  Some network tests failed"* ]]
}

@test "modular run - TSO fix success scenario" {
    # Override core function to have issues
    network_diagnostics_core::run() {
        echo "[CORE] Some tests failed"
        return 1
    }
    
    # Override flow::can_run_sudo to allow sudo
    flow::can_run_sudo() { return 0; }
    
    # Mock successful TSO scenario
    ip() {
        echo "default via 192.168.1.1 dev eth0"
        return 0
    }
    
    ethtool() {
        case "$*" in
            *"-k"*)
                echo "tcp-segmentation-offload: on"
                return 0
                ;;
            *"-K"*)
                track_command "ethtool disable TSO" 0
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    command() { [[ "$*" == *"ethtool"* ]] && return 0 || return 1; }
    
    # Mock successful GitHub test after TSO fix
    curl() {
        if [[ "$*" == *"github.com"* ]]; then
            return 0
        fi
        return 1
    }
    
    timeout() { shift; "$@"; }
    sudo() { 
        if [[ "$*" == *"ethtool"* ]]; then
            track_command "sudo disable TSO" 0
        fi
        return 0
    }
    
    # Override TSO permanent function
    network_diagnostics_tcp::make_tso_permanent() {
        echo "[TCP] Made TSO fix permanent for $1"
        track_command "make TSO permanent" 0
    }
    
    ping() { return 0; }  # No IPv6 issues
    
    export -f network_diagnostics_core::run flow::can_run_sudo ip ethtool command
    export -f curl timeout sudo network_diagnostics_tcp::make_tso_permanent ping track_command
    
    run network_diagnostics::run
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Attempting to disable TSO on eth0"* ]]
    [[ "$output" == *"✓ Disabled TSO on eth0"* ]]
    [[ "$output" == *"✓ GitHub HTTPS now works!"* ]]
    [[ "$output" == *"Made TSO fix permanent for eth0"* ]]
}

@test "modular run - handles IPv6 issues detection" {
    network_diagnostics_core::run() { return 1; }
    
    # Mock ping to show IPv6 issues
    ping() {
        case "$*" in
            *"-6"* && *"google.com"*)
                return 1  # IPv6 fails
                ;;
            *"-4"* && *"google.com"*)
                return 0  # IPv4 works
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Mock no TSO issues
    ip() { echo ""; }  # No default interface
    command() { return 1; }  # No ethtool
    
    export -f network_diagnostics_core::run ping ip command
    
    run network_diagnostics::run
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"IPv6 connectivity issues detected, applying fixes"* ]]
    [[ "$output" == *"Fixing IPv6 issues"* ]]
}

@test "modular run - handles UFW firewall checks" {
    network_diagnostics_core::run() { return 1; }
    
    # Mock UFW available
    command() {
        [[ "$*" == *"ufw"* ]] && return 0 || return 1
    }
    
    # Mock no IPv6 or TSO issues
    ping() { return 0; }
    ip() { echo ""; }
    
    export -f network_diagnostics_core::run command ping ip
    
    run network_diagnostics::run
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Fixing UFW blocking"* ]]
}

# =============================================================================
# Tests for SUDO_MODE configuration
# =============================================================================

@test "modular run - sets SUDO_MODE based on permissions" {
    # Test with root user
    EUID=0
    
    # Mock successful core run
    network_diagnostics_core::run() { return 0; }
    
    export -f network_diagnostics_core::run
    
    run network_diagnostics::run
    
    [ "$status" -eq 0 ]
    # Should not crash with SUDO_MODE configuration
}

@test "modular run - handles sudo availability" {
    # Mock sudo command availability
    sudo() {
        case "$*" in
            "-n true")
                return 0  # sudo available without password
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    network_diagnostics_core::run() { return 0; }
    
    export -f sudo network_diagnostics_core::run
    
    run network_diagnostics::run
    
    [ "$status" -eq 0 ]
    # Should configure SUDO_MODE to "error"
}

# =============================================================================
# Tests for diagnostic summary
# =============================================================================

@test "modular run - provides appropriate summary for success" {
    network_diagnostics_core::run() { return 0; }
    
    export -f network_diagnostics_core::run
    
    run network_diagnostics::run
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Diagnostic Summary"* ]]
    [[ "$output" == *"✅ All core network tests passed"* ]]
    [[ "$output" == *"Your network connectivity appears to be working correctly"* ]]
}

@test "modular run - provides appropriate summary for no internet" {
    network_diagnostics_core::run() { return 5; }  # ERROR_NO_INTERNET
    
    export -f network_diagnostics_core::run
    
    run network_diagnostics::run
    
    [ "$status" -eq 5 ]
    [[ "$output" == *"❌ No internet connectivity"* ]]
    [[ "$output" == *"Please check your network connection and try again"* ]]
}

@test "modular run - provides appropriate summary for partial failures" {
    network_diagnostics_core::run() { return 1; }
    
    # Mock no fixes applied
    ping() { return 0; }  # No IPv6 issues
    ip() { echo ""; }     # No TSO issues
    command() { return 1; }  # No UFW
    
    export -f network_diagnostics_core::run ping ip command
    
    run network_diagnostics::run
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Diagnostic Summary"* ]]
    [[ "$output" == *"⚠️  Some network tests failed"* ]]
    [[ "$output" == *"Automatic fixes have been attempted"* ]]
    [[ "$output" == *"1. Restart your network connection"* ]]
    [[ "$output" == *"2. Check with your network administrator"* ]]
    [[ "$output" == *"3. Try using a different network"* ]]
}

# =============================================================================
# Edge Cases and Error Handling
# =============================================================================

@test "edge case: handles missing module functions gracefully" {
    # Undefine some module functions to test graceful handling
    unset -f network_diagnostics_tcp::check_tcp_settings
    unset -f network_diagnostics_analysis::check_time_sync
    
    network_diagnostics_core::run() { return 0; }
    
    export -f network_diagnostics_core::run
    
    # Should not crash even with missing functions
    run network_diagnostics::run
    
    # Exit code may vary but should not crash
}

@test "edge case: handles command failures gracefully" {
    network_diagnostics_core::run() { return 1; }
    
    # Mock all commands to fail
    ip() { return 1; }
    ethtool() { return 1; }
    ping() { return 1; }
    command() { return 1; }
    
    export -f network_diagnostics_core::run ip ethtool ping command
    
    run network_diagnostics::run
    
    [ "$status" -eq 1 ]
    # Should complete without crashing
}

@test "performance: modular integration completes within reasonable time" {
    # Mock all functions to return quickly
    network_diagnostics_core::run() { return 0; }
    network_diagnostics_tcp::check_tcp_settings() { return 0; }
    network_diagnostics_tcp::check_pmtu_status() { return 0; }
    network_diagnostics_analysis::check_time_sync() { return 0; }
    network_diagnostics_analysis::check_ipv4_vs_ipv6() { return 0; }
    network_diagnostics_analysis::check_ip_preference() { return 0; }
    
    export -f network_diagnostics_core::run network_diagnostics_tcp::check_tcp_settings
    export -f network_diagnostics_tcp::check_pmtu_status network_diagnostics_analysis::check_time_sync
    export -f network_diagnostics_analysis::check_ipv4_vs_ipv6 network_diagnostics_analysis::check_ip_preference
    
    # Run with time limit to ensure it doesn't hang
    run timeout 10 network_diagnostics::run
    
    [ "$status" -eq 0 ]
}