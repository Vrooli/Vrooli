#!/usr/bin/env bats
# Essential tests for unified network diagnostics module
# Focused on integration and critical functionality

setup() {
    export TEST_TEMP_DIR="$(mktemp -d)"
    
    # Source the unified module
    source "${BATS_TEST_DIRNAME}/network_diagnostics.sh"
    
    # Mock log functions
    log::header() { echo "[HEADER] $*"; }
    log::subheader() { echo "[SUBHEADER] $*"; }
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*"; }
    
    # Mock flow functions
    flow::can_run_sudo() { return 1; }  # Default to no sudo
    
    # Export error codes
    export ERROR_NO_INTERNET=5
}

teardown() {
    [[ -d "$TEST_TEMP_DIR" ]] && rm -rf "$TEST_TEMP_DIR"
}

# =============================================================================
# Core Diagnostics Tests
# =============================================================================

@test "core::run - all tests pass" {
    # Mock successful commands
    ping() { return 0; }
    getent() { echo "93.184.216.34 google.com"; return 0; }
    nc() { return 0; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics_core::run
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"All core network tests passed!"* ]]
}

@test "core::run - critical failure (no internet)" {
    # Mock critical failure
    ping() { 
        [[ "$*" == *"8.8.8.8"* ]] && return 1 || return 0
    }
    getent() { return 0; }
    nc() { return 0; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { return 0; }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics_core::run
    
    [ "$status" -eq 5 ]  # ERROR_NO_INTERNET
    [[ "$output" == *"Critical network issues detected"* ]]
}

@test "core::run - partial TLS issue" {
    # Mock TLS-only failure
    ping() { return 0; }
    getent() { echo "93.184.216.34 google.com"; return 0; }
    nc() { return 0; }
    curl() { return 1; }  # HTTPS fails
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics_core::run
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"PARTIAL TLS ISSUE"* ]]
}

# =============================================================================
# TCP Optimization Tests
# =============================================================================

@test "tcp::make_tso_permanent - systemd-networkd" {
    # Mock systemd-networkd enabled
    systemctl() {
        [[ "$1" == "is-enabled" && "$2" == "systemd-networkd" ]] && return 0
        return 0
    }
    
    # Mock sudo
    sudo() {
        [[ "$1" == "tee" ]] && cat > /dev/null && return 0
        return 0
    }
    
    # Override flow to allow sudo
    flow::can_run_sudo() { return 0; }
    
    export -f systemctl sudo flow::can_run_sudo
    
    run network_diagnostics_tcp::make_tso_permanent "eth0"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"systemd-networkd"* ]]
}

@test "tcp::check_tcp_settings" {
    # Mock sysctl
    sysctl() {
        case "$*" in
            *"tcp_congestion_control"*) echo "cubic" ;;
            *"tcp_keepalive_time"*) echo "7200" ;;
            *"tcp_window_scaling"*) echo "1" ;;
            *"tcp_timestamps"*) echo "1" ;;
            *"tcp_ecn"*) echo "2" ;;
        esac
    }
    
    export -f sysctl
    
    run network_diagnostics_tcp::check_tcp_settings
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"cubic"* ]]
    [[ "$output" == *"7200"* ]]
}

@test "tcp::test_mtu_discovery" {
    # Mock ping for MTU testing
    ping() {
        [[ "$*" == *"1472"* ]] && return 0  # 1500 MTU works
        return 1
    }
    
    command() { [[ "$2" == "ping" ]] && return 0 || return 1; }
    
    export -f ping command
    
    run network_diagnostics_tcp::test_mtu_discovery "test.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Maximum working MTU"* ]]
    [[ "$output" == *"1500"* ]]
}

# =============================================================================
# Analysis Tests
# =============================================================================

@test "analysis::diagnose_connection_failure - time issue" {
    # Mock incorrect system time
    date() {
        [[ "$*" == *"%Y"* ]] && echo "2019" || echo "2019-01-01"
    }
    
    export -f date
    
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS test" "google.com"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"System time is incorrect"* ]]
}

@test "analysis::diagnose_connection_failure - IPv4 only" {
    # Mock IPv4 works, IPv6 doesn't
    ping() {
        [[ "$*" == *"-4"* ]] && return 0
        [[ "$*" == *"-6"* ]] && return 1
        return 1
    }
    
    date() { echo "2024"; }
    
    export -f ping date
    
    run network_diagnostics_analysis::diagnose_connection_failure "ping test" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"IPv4 works, IPv6 doesn't"* ]]
}

# =============================================================================
# Fixes Tests
# =============================================================================

@test "fixes::fix_ipv6_issues" {
    # Override sudo availability
    flow::can_run_sudo() { return 0; }
    
    # Mock file operations
    grep() { return 1; }  # Not found
    
    sudo() {
        [[ "$*" == *"cp"* ]] && return 0
        [[ "$*" == *"tee"* ]] && cat > /dev/null && return 0
        return 0
    }
    
    export -f flow::can_run_sudo grep sudo
    
    run network_diagnostics_fixes::fix_ipv6_issues
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"System configured to prefer IPv4"* ]]
}

@test "fixes::fix_ipv4_only_issues" {
    # Mock file checks
    grep() { return 1; }  # Configs not found
    
    # Mock git
    git() { return 0; }
    
    # Mock command
    command() {
        [[ "$*" == *"git"* ]] && return 0
        [[ "$*" == *"apt-get"* ]] && return 0
        return 1
    }
    
    # Override sudo
    flow::can_run_sudo() { return 0; }
    sudo() { cat > /dev/null 2>&1 || true; return 0; }
    
    export -f grep git command flow::can_run_sudo sudo
    
    run network_diagnostics_fixes::fix_ipv4_only_issues
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Configured curl to use IPv4"* ]]
    [[ "$output" == *"Configured git to prefer IPv4"* ]]
}

@test "fixes::fix_dns_issues" {
    flow::can_run_sudo() { return 0; }
    
    # Mock systemd-resolved not active
    systemctl() { return 1; }
    
    # Mock grep to not find existing DNS
    grep() { return 1; }
    
    sudo() {
        [[ "$*" == *"cp"* ]] && return 0
        [[ "$*" == *"tee"* ]] && cat > /dev/null && return 0
        return 0
    }
    
    # Mock resolvectl
    command() {
        [[ "$*" == *"resolvectl"* ]] && return 0
        return 1
    }
    
    export -f flow::can_run_sudo systemctl grep sudo command
    
    run network_diagnostics_fixes::fix_dns_issues
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Added reliable DNS servers"* ]]
}

@test "fixes::fix_ufw_blocking" {
    flow::can_run_sudo() { return 0; }
    
    # Mock ufw command available
    command() {
        [[ "$*" == *"ufw"* ]] && return 0
        return 1
    }
    
    # Mock ufw active
    sudo() {
        case "$*" in
            "ufw status") echo "Status: active" ;;
            "ufw allow"*) return 0 ;;
            "ufw reload") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    export -f flow::can_run_sudo command sudo
    
    run network_diagnostics_fixes::fix_ufw_blocking
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Added outbound rule for HTTP"* ]]
    [[ "$output" == *"Added outbound rule for HTTPS"* ]]
    [[ "$output" == *"Added outbound rule for DNS"* ]]
}

# =============================================================================
# Integration Tests
# =============================================================================

@test "integration: main diagnostic function" {
    # Mock successful core run
    network_diagnostics_core::run() {
        log::success "Core tests passed"
        return 0
    }
    
    export -f network_diagnostics_core::run
    
    run network_diagnostics::run
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"All network tests passed!"* ]]
}

@test "integration: diagnose on failure" {
    # Mock failed core run
    network_diagnostics_core::run() {
        log::error "Tests failed"
        return 1
    }
    
    # Mock diagnosis
    network_diagnostics_analysis::diagnose_connection_failure() {
        log::info "Diagnosing..."
        return 0
    }
    
    export -f network_diagnostics_core::run network_diagnostics_analysis::diagnose_connection_failure
    
    run network_diagnostics::run
    
    [ "$status" -eq 0 ]  # Still returns 0 for non-critical failures
    [[ "$output" == *"many apps work offline"* ]]
}

@test "integration: all functions exported" {
    # Verify all required functions are defined and exported
    declare -f network_diagnostics::run > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_core::run > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_fixes::fix_ipv6_issues > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_fixes::fix_dns_issues > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_tcp::check_tcp_settings > /dev/null
    [ "$?" -eq 0 ]
}

# =============================================================================
# Error Handling Tests
# =============================================================================

@test "error: handles missing commands" {
    command() { return 1; }  # All commands missing
    
    export -f command
    
    run network_diagnostics_tcp::test_mtu_discovery
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"ping command not available"* ]]
}

@test "error: handles permission denied" {
    flow::can_run_sudo() { return 1; }
    
    export -f flow::can_run_sudo
    
    run network_diagnostics_tcp::fix_ecn
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Skipping ECN fix (requires sudo)"* ]]
}

@test "error: validates parameters" {
    run network_diagnostics_fixes::add_ipv4_host_override "" ""
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Domain and IPv4 address required"* ]]
}