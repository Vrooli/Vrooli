#!/usr/bin/env bats

# Tests for TCP optimization functions
# Tests TSO, MTU, ECN, and other TCP-specific network fixes

setup() {
    export BATS_TEST_DIRNAME="$BATS_TEST_DIRNAME"
    export TEST_TEMP_DIR="$(mktemp -d)"
    
    # Source trash module for safe test cleanup
    SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
    
    # Source the script under test
    source "${BATS_TEST_DIRNAME}/network_diagnostics_tcp.sh"
    
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
    
    export -f log::header log::subheader log::info log::success log::warning log::error
    export -f flow::can_run_sudo
    
    # Track which commands were called
    export COMMANDS_CALLED=()
}

teardown() {
    trash::safe_remove "$TEST_TEMP_DIR" --test-cleanup
    unset COMMANDS_CALLED
}

# Helper function to track command calls
track_command() {
    COMMANDS_CALLED+=("$1")
    return "${2:-0}"
}

# =============================================================================
# Tests for TSO permanent fix
# =============================================================================

@test "make_tso_permanent - systemd-networkd method" {
    local test_interface="eth0"
    
    # Mock systemctl to report systemd-networkd is enabled
    systemctl() {
        if [[ "$1" == "is-enabled" ]] && [[ "$2" == "systemd-networkd" ]]; then
            return 0
        fi
        track_command "systemctl $*" 0
        return 0
    }
    
    # Mock sudo and tee
    sudo() {
        if [[ "$1" == "tee" ]]; then
            shift  # Remove 'tee'
            local file="$1"
            track_command "sudo tee $file" 0
            cat > /dev/null  # Consume stdin
            return 0
        fi
        track_command "sudo $*" 0
        return 0
    }
    
    export -f systemctl sudo track_command
    
    run network_diagnostics_tcp::make_tso_permanent "$test_interface"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Using systemd-networkd configuration"* ]]
    [[ "$output" == *"Created /etc/systemd/network/99-tso-fix.network"* ]]
    [[ "$output" == *"TCP offload fix is now permanent via systemd-networkd"* ]]
}

@test "make_tso_permanent - NetworkManager method" {
    local test_interface="eth0"
    
    # Mock systemctl - systemd-networkd disabled, NetworkManager active
    systemctl() {
        case "$*" in
            "is-enabled systemd-networkd")
                return 1  # Not enabled
                ;;
            "is-active NetworkManager")
                return 0  # Active
                ;;
            *)
                track_command "systemctl $*" 0
                return 0
                ;;
        esac
    }
    
    # Mock sudo
    sudo() {
        if [[ "$1" == "tee" ]]; then
            shift
            track_command "sudo tee $1" 0
            cat > /dev/null
            return 0
        elif [[ "$1" == "chmod" ]]; then
            track_command "sudo chmod $*" 0
            return 0
        fi
        track_command "sudo $*" 0
        return 0
    }
    
    export -f systemctl sudo track_command
    
    run network_diagnostics_tcp::make_tso_permanent "$test_interface"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Using NetworkManager dispatcher"* ]]
    [[ "$output" == *"Created NetworkManager dispatcher script"* ]]
    [[ "$output" == *"TCP offload fix is now permanent via NetworkManager"* ]]
}

@test "make_tso_permanent - systemd service fallback" {
    local test_interface="eth0"
    
    # Mock systemctl - both methods not available
    systemctl() {
        case "$*" in
            "is-enabled systemd-networkd")
                return 1
                ;;
            "is-active NetworkManager")
                return 1
                ;;
            *)
                track_command "systemctl $*" 0
                return 0
                ;;
        esac
    }
    
    # Mock sudo
    sudo() {
        if [[ "$1" == "tee" ]]; then
            shift
            track_command "sudo tee $1" 0
            cat > /dev/null
            return 0
        fi
        track_command "sudo $*" 0
        return 0
    }
    
    export -f systemctl sudo track_command
    
    run network_diagnostics_tcp::make_tso_permanent "$test_interface"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Creating systemd service as backup"* ]]
    [[ "$output" == *"Created and enabled systemd service"* ]]
}

# =============================================================================
# Tests for TCP settings check
# =============================================================================

@test "check_tcp_settings - displays all settings" {
    # Mock sysctl - note awk gets the third field
    sysctl() {
        case "$*" in
            *"tcp_congestion_control"*)
                echo "net.ipv4.tcp_congestion_control = cubic"
                ;;
            *"tcp_keepalive_time"*)
                echo "net.ipv4.tcp_keepalive_time = 7200"
                ;;
            *"tcp_window_scaling"*)
                echo "net.ipv4.tcp_window_scaling = 1"
                ;;
            *"tcp_timestamps"*)
                echo "net.ipv4.tcp_timestamps = 1"
                ;;
            *"tcp_ecn"*)
                echo "net.ipv4.tcp_ecn = 2"
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    export -f sysctl
    
    run network_diagnostics_tcp::check_tcp_settings
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Checking TCP settings"* ]]
    [[ "$output" == *"cubic"* ]]
    [[ "$output" == *"7200"* ]]
    [[ "$output" == *"TCP window scaling: 1"* ]]
    [[ "$output" == *"TCP timestamps: 1"* ]]
    [[ "$output" == *"TCP ECN: 2"* ]]
}

@test "check_tcp_settings - handles missing sysctl values" {
    # Mock sysctl to fail
    sysctl() { return 1; }
    
    export -f sysctl
    
    run network_diagnostics_tcp::check_tcp_settings
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"TCP congestion control: unknown"* ]]
    [[ "$output" == *"TCP keepalive time: unknown seconds"* ]]
}

# =============================================================================
# Tests for ECN fixes
# =============================================================================

@test "fix_ecn - disables ECN when sudo available" {
    # Override flow::can_run_sudo to return success
    flow::can_run_sudo() { return 0; }
    
    # Mock sudo and sysctl
    sudo() {
        case "$*" in
            "sysctl -w net.ipv4.tcp_ecn=0")
                track_command "sudo sysctl ecn" 0
                return 0
                ;;
            "tee -a /etc/sysctl.conf")
                track_command "sudo tee sysctl.conf" 0
                cat > /dev/null
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    export -f flow::can_run_sudo sudo track_command
    
    run network_diagnostics_tcp::fix_ecn
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Disabled TCP ECN"* ]]
    [[ "$output" == *"Made ECN change permanent"* ]]
}

@test "fix_ecn - skips when no sudo" {
    # Override flow::can_run_sudo to deny
    flow::can_run_sudo() { return 1; }
    
    export -f flow::can_run_sudo
    
    run network_diagnostics_tcp::fix_ecn
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Skipping ECN fix (requires sudo)"* ]]
}

# =============================================================================
# Tests for MTU fixes
# =============================================================================

@test "fix_mtu_size - changes MTU successfully" {
    local test_interface="eth0"
    local test_mtu="1400"
    
    # Override flow::can_run_sudo
    flow::can_run_sudo() { return 0; }
    
    # Mock sudo and ip
    sudo() {
        case "$*" in
            "ip link set dev $test_interface mtu $test_mtu")
                track_command "sudo ip mtu" 0
                return 0
                ;;
            *)
                track_command "sudo $*" 0
                return 0
                ;;
        esac
    }
    
    export -f flow::can_run_sudo sudo track_command
    
    run network_diagnostics_tcp::fix_mtu_size "$test_interface" "$test_mtu"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Set MTU to $test_mtu on $test_interface"* ]]
}

@test "fix_mtu_size - uses default MTU when not specified" {
    local test_interface="eth0"
    
    flow::can_run_sudo() { return 0; }
    
    sudo() {
        if [[ "$*" == *"mtu 1400"* ]]; then  # Default MTU
            track_command "sudo ip mtu default" 0
            return 0
        fi
        track_command "sudo $*" 0
        return 0
    }
    
    export -f flow::can_run_sudo sudo track_command
    
    run network_diagnostics_tcp::fix_mtu_size "$test_interface"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Set MTU to 1400 on $test_interface"* ]]
}

# =============================================================================
# Tests for PMTU discovery
# =============================================================================

@test "test_mtu_discovery - finds working MTU" {
    # Mock ping to succeed for certain MTU sizes
    ping() {
        case "$*" in
            *"1472"*|*"1372"*|*"1252"*|*"996"*)  # 1500, 1400, 1280, 1024 MTU sizes
                track_command "ping mtu-test $*" 0
                return 0
                ;;
            *)
                track_command "ping mtu-test-fail $*" 1
                return 1
                ;;
        esac
    }
    
    command() { [[ "$2" == "ping" ]] && return 0 || return 1; }
    
    export -f ping command track_command
    
    run network_diagnostics_tcp::test_mtu_discovery "test.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Maximum working MTU to test.com: 1500 bytes"* ]]
}

@test "test_mtu_discovery - ping command not available" {
    command() { return 1; }
    
    export -f command
    
    run network_diagnostics_tcp::test_mtu_discovery "test.com"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"ping command not available"* ]]
}

@test "test_mtu_discovery - no working MTU found" {
    # Mock ping to always fail
    ping() { return 1; }
    command() { [[ "$2" == "ping" ]] && return 0 || return 1; }
    
    export -f ping command
    
    run network_diagnostics_tcp::test_mtu_discovery "test.com"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Could not determine working MTU"* ]]
}

# =============================================================================
# Tests for PMTU status check
# =============================================================================

@test "check_pmtu_status - reports PMTU settings" {
    # Mock sysctl
    sysctl() {
        case "$*" in
            *"tcp_mtu_probing"*)
                echo "net.ipv4.tcp_mtu_probing = 1"
                ;;
            *"tcp_base_mss"*)
                echo "net.ipv4.tcp_base_mss = 1024"
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    export -f sysctl
    
    run network_diagnostics_tcp::check_pmtu_status
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Checking Path MTU discovery status"* ]]
    [[ "$output" == *"PMTU probing: Enabled (disabled by default)"* ]]
    [[ "$output" == *"TCP base MSS: 1024"* ]]
}

@test "check_pmtu_status - different PMTU modes" {
    # Test different PMTU probing values
    sysctl() {
        case "$*" in
            *"tcp_mtu_probing"*)
                echo "net.ipv4.tcp_mtu_probing = 0"
                ;;
            *"tcp_base_mss"*)
                echo "net.ipv4.tcp_base_mss = 512"
                ;;
        esac
    }
    
    export -f sysctl
    
    run network_diagnostics_tcp::check_pmtu_status
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"PMTU probing: Disabled"* ]]
    [[ "$output" == *"TCP base MSS: 512"* ]]
}

# =============================================================================
# Tests for PMTU probing fixes
# =============================================================================

@test "fix_pmtu_probing - enables PMTU discovery" {
    flow::can_run_sudo() { return 0; }
    
    sudo() {
        case "$*" in
            "sysctl -w net.ipv4.tcp_mtu_probing=1")
                track_command "sudo pmtu probing" 0
                return 0
                ;;
            "sysctl -w net.ipv4.tcp_base_mss=1024")
                track_command "sudo base mss" 0
                return 0
                ;;
            "tee -a /etc/sysctl.conf")
                track_command "sudo tee sysctl" 0
                cat > /dev/null
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    export -f flow::can_run_sudo sudo track_command
    
    run network_diagnostics_tcp::fix_pmtu_probing
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Enabled TCP MTU probing"* ]]
    [[ "$output" == *"Set TCP base MSS to 1024"* ]]
    [[ "$output" == *"Made PMTU changes permanent"* ]]
}

@test "fix_pmtu_probing - skips when no sudo" {
    flow::can_run_sudo() { return 1; }
    
    export -f flow::can_run_sudo
    
    run network_diagnostics_tcp::fix_pmtu_probing
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Skipping PMTU fix (requires sudo)"* ]]
}

# =============================================================================
# Edge Cases and Error Handling
# =============================================================================

@test "edge case: handles missing interface parameter" {
    run network_diagnostics_tcp::make_tso_permanent ""
    
    [ "$status" -ne 0 ]
    # Function should handle empty interface gracefully
}

@test "edge case: handles sysctl command failures" {
    sysctl() { return 1; }  # Simulate sysctl failure
    
    export -f sysctl
    
    run network_diagnostics_tcp::check_tcp_settings
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"unknown"* ]]
}

@test "edge case: handles permission denied gracefully" {
    flow::can_run_sudo() { return 1; }
    
    export -f flow::can_run_sudo
    
    run network_diagnostics_tcp::fix_ecn
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"requires sudo"* ]]
}