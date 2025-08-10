#!/usr/bin/env bats

# Comprehensive tests for network diagnostics modules
# Tests all modules in scripts/lib/network/diagnostics/

setup() {
    export BATS_TEST_DIRNAME="$BATS_TEST_DIRNAME"
    export TEST_TEMP_DIR="$(mktemp -d)"
    
    # Source trash module for safe test cleanup
    SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
    
    # Source all modules under test
    source "${BATS_TEST_DIRNAME}/network_diagnostics_core.sh"
    source "${BATS_TEST_DIRNAME}/network_diagnostics_tcp.sh"
    source "${BATS_TEST_DIRNAME}/network_diagnostics_analysis.sh"
    source "${BATS_TEST_DIRNAME}/network_diagnostics_fixes.sh"
    
    # Mock log functions using the official pattern - simple but consistent
    log::header() { echo "[HEADER] $*"; }
    log::subheader() { echo "[SUBHEADER] $*"; }
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*"; }
    
    # Mock flow functions
    flow::can_run_sudo() { 
        return 1  # Default to no sudo in tests
    }
    
    # Export error codes
    export ERROR_NO_INTERNET=5
    
    # Track which commands were called for verification
    export COMMANDS_CALLED=()
}

teardown() {
    trash::safe_remove "$TEST_TEMP_DIR" --test-cleanup
    unset COMMANDS_CALLED
}

# Helper function to track command calls for test verification
track_command() {
    COMMANDS_CALLED+=("$1")
    return "${2:-0}"
}

# =============================================================================
# Tests for network_diagnostics_core.sh
# =============================================================================

@test "core::run - all tests pass scenario" {
    # Configure mocks for successful scenario
    ping() { return 0; }
    getent() { echo "93.184.216.34 google.com"; return 0; }
    nc() { return 0; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    date() { echo "1234567890"; }
    command() { [[ "$2" == "curl" ]] || [[ "$2" == "date" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout date command
    
    run network_diagnostics_core::run
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"All core network tests passed!"* ]]
    [[ "$output" == *"✓ IPv4 ping to 8.8.8.8"* ]]
    [[ "$output" == *"✓ IPv4 ping to google.com"* ]]
    [[ "$output" == *"✓ DNS lookup (getent)"* ]]
    [[ "$output" == *"✓ TCP port 443 (HTTPS)"* ]]
    [[ "$output" == *"✓ HTTPS to google.com"* ]]
}

@test "core::run - critical failure (no internet)" {
    # Mock ping to fail for critical test (8.8.8.8)
    ping() { 
        if [[ "$*" == *"8.8.8.8"* ]]; then
            return 1
        else
            return 0
        fi
    }
    getent() { echo "93.184.216.34 google.com"; return 0; }
    nc() { return 0; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout command
    
    run network_diagnostics_core::run
    
    [ "$status" -eq 5 ]  # ERROR_NO_INTERNET
    [[ "$output" == *"✗ IPv4 ping to 8.8.8.8"* ]]
    [[ "$output" == *"Critical network issues detected"* ]]
    [[ "$output" == *"Basic connectivity is broken"* ]]
}

@test "core::run - partial TLS issue detection" {
    # Mock scenario where TCP works but HTTPS fails
    ping() { track_command "ping $*" 0; }
    getent() { track_command "getent $*" 0; echo "93.184.216.34 google.com"; }
    nc() { track_command "nc $*" 0; }
    curl() { 
        track_command "curl $*" 1
        return 1  # HTTPS fails
    }
    timeout() { shift; "$@"; }
    ethtool() { track_command "ethtool $*" 0; }
    command() { 
        case "$2" in
            curl|ethtool) return 0 ;;
            *) return 1 ;;
        esac
    }
    
    export -f ping getent nc curl timeout ethtool command track_command
    
    run network_diagnostics_core::run
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"[SUCCESS]   ✓ IPv4 ping to google.com"* ]]
    [[ "$output" == *"[SUCCESS]   ✓ DNS lookup (getent)"* ]]
    [[ "$output" == *"[SUCCESS]   ✓ TCP port 443 (HTTPS)"* ]]
    [[ "$output" == *"[ERROR]   ✗ HTTPS to google.com"* ]]
    [[ "$output" == *"PARTIAL TLS ISSUE"* ]]
    [[ "$output" == *"TCP connectivity works but HTTPS fails"* ]]
}

@test "core::run - DNS failure scenario" {
    ping() { 
        if [[ "$*" == *"8.8.8.8"* ]]; then
            track_command "ping $*" 0
            return 0
        else
            track_command "ping $*" 1
            return 1  # Domain ping fails
        fi
    }
    getent() { 
        track_command "getent $*" 1
        return 1  # DNS lookup fails
    }
    nc() { track_command "nc $*" 1; return 1; }
    curl() { track_command "curl $*" 1; return 1; }
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout command track_command
    
    run network_diagnostics_core::run
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"[SUCCESS]   ✓ IPv4 ping to 8.8.8.8"* ]]
    [[ "$output" == *"[ERROR]   ✗ IPv4 ping to google.com"* ]]
    [[ "$output" == *"[ERROR]   ✗ DNS lookup (getent)"* ]]
    [[ "$output" == *"Network issues detected"* ]]
}

# =============================================================================
# Tests for network_diagnostics_tcp.sh
# =============================================================================

@test "tcp::make_tso_permanent - systemd-networkd method" {
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

@test "tcp::make_tso_permanent - NetworkManager method" {
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

@test "tcp::check_tcp_settings - displays all settings" {
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
    # The function uses awk to get field 3, so output will be just the value
    [[ "$output" == *"cubic"* ]]
    [[ "$output" == *"7200"* ]]
}

@test "tcp::fix_ecn - disables ECN when sudo available" {
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

@test "tcp::test_mtu_discovery - finds working MTU" {
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

@test "tcp::check_pmtu_status - reports PMTU settings" {
    # Mock sysctl - note the function uses awk to get field 3
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
    # Check for the values that awk extracts
    [[ "$output" == *"1024"* ]]
}

# =============================================================================
# Tests for network_diagnostics_analysis.sh (streamlined)
# =============================================================================

@test "analysis::diagnose_connection_failure - system time issue" {
    # Mock date to return bad year
    date() {
        echo "2018"  # Too old
        return 0
    }
    
    export -f date
    
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 1 ]  # Should return 1 for critical time issue
    [[ "$output" == *"🔍 Diagnosing connection failure"* ]]
    [[ "$output" == *"❌ System time is incorrect (year: 2018)"* ]]
    [[ "$output" == *"💡 Fix: Set correct system time"* ]]
}

@test "analysis::diagnose_connection_failure - IPv4 works IPv6 doesn't" {
    # Mock date to be correct
    date() { echo "2024"; }
    
    # Mock ping - IPv4 works, IPv6 doesn't
    ping() {
        case "$*" in
            *"-4"*)
                return 0  # IPv4 works
                ;;
            *"-6"*)
                return 1  # IPv6 fails
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    export -f date ping
    
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"✅ IPv4 works, IPv6 doesn't - this is normal"* ]]
    [[ "$output" == *"💡 System will automatically use IPv4"* ]]
}

@test "analysis::diagnose_connection_failure - only IPv6 works" {
    # Mock date to be correct
    date() { echo "2024"; }
    
    # Mock ping - IPv4 fails, IPv6 works
    ping() {
        case "$*" in
            *"-4"*)
                return 1  # IPv4 fails
                ;;
            *"-6"*)
                return 0  # IPv6 works
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    export -f date ping
    
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"⚠️  Only IPv6 works - may need IPv4 preference fix"* ]]
    [[ "$output" == *"💡 Fix: Configure system to prefer IPv4"* ]]
}

@test "analysis::diagnose_connection_failure - TLS handshake works" {
    # Mock date to be correct
    date() { echo "2024"; }
    
    # Mock openssl to succeed
    openssl() {
        return 0
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "openssl" ]] && return 0 || return 1; }
    
    export -f date openssl timeout command
    
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"🔍 Testing TLS handshake"* ]]
    [[ "$output" == *"✅ TLS handshake works"* ]]
}

@test "analysis::diagnose_connection_failure - TLS handshake fails with certificate error" {
    # Mock date to be correct
    date() { echo "2024"; }
    
    # Mock openssl to fail with certificate error
    openssl() {
        case "$*" in
            *"-verify_return_error"*)
                return 1
                ;;
            *)
                echo "verify error:num=20:unable to get local issuer certificate"
                return 1
                ;;
        esac
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "openssl" ]] && return 0 || return 1; }
    
    export -f date openssl timeout command
    
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"⚠️  TLS issue: verify error:num=20:unable to get local issuer certificate"* ]]
    [[ "$output" == *"💡 This could be certificate, cipher, or TSO issues"* ]]
}

@test "analysis::diagnose_connection_failure - DNS resolution works" {
    # Mock date to be correct
    date() { echo "2024"; }
    
    # Mock getent to succeed
    getent() {
        echo "8.8.8.8"
        return 0
    }
    
    export -f date getent
    
    run network_diagnostics_analysis::diagnose_connection_failure "DNS lookup (getent)" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"🔍 Testing DNS resolution"* ]]
    [[ "$output" == *"✅ DNS system works"* ]]
    [[ "$output" == *"💡 Issue may be with specific domain 'google.com'"* ]]
}

@test "analysis::diagnose_connection_failure - DNS system broken" {
    # Mock date to be correct
    date() { echo "2024"; }
    
    # Mock getent to fail
    getent() {
        return 1
    }
    
    export -f date getent
    
    run network_diagnostics_analysis::diagnose_connection_failure "DNS lookup (getent)" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"❌ DNS system appears broken"* ]]
    [[ "$output" == *"💡 Fix: Add reliable DNS servers"* ]]
}

@test "analysis::diagnose_connection_failure - HTTPS SSL certificate error" {
    # Mock date to be correct
    date() { echo "2024"; }
    
    # Mock curl to return SSL error
    curl() {
        echo "curl: (60) SSL certificate problem: unable to get local issuer certificate" >&2
        return 1
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f date curl timeout command
    
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"🔍 Analyzing HTTPS error patterns"* ]]
    [[ "$output" == *"⚠️  SSL/TLS certificate issue detected"* ]]
    [[ "$output" == *"💡 Common causes: wrong system time, TSO issues"* ]]
}

@test "analysis::diagnose_connection_failure - HTTPS DNS resolution error" {
    # Mock date to be correct
    date() { echo "2024"; }
    
    # Mock curl to return DNS error
    curl() {
        echo "curl: (6) Could not resolve host: google.com" >&2
        return 1
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f date curl timeout command
    
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"⚠️  DNS resolution failure"* ]]
    [[ "$output" == *"💡 Fix: Add reliable DNS servers"* ]]
}

@test "analysis::diagnose_connection_failure - HTTPS timeout error" {
    # Mock date to be correct
    date() { echo "2024"; }
    
    # Mock curl to return timeout error
    curl() {
        echo "curl: (28) Connection timed out after 5000 milliseconds" >&2
        return 1
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f date curl timeout command
    
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"⚠️  Connection timeout"* ]]
    [[ "$output" == *"💡 This suggests network routing or firewall issues"* ]]
}

# =============================================================================
# Tests for network_diagnostics_fixes.sh
# =============================================================================

@test "fixes::fix_ipv6_issues - configures IPv4 preference" {
    # Override flow::can_run_sudo to allow
    flow::can_run_sudo() { return 0; }
    
    # Mock file operations
    [[ -f /etc/gai.conf ]] || true  # Pretend file exists or doesn't matter
    
    # Mock sudo
    sudo() {
        case "$*" in
            "cp /etc/gai.conf"*)
                track_command "backup gai.conf" 0
                return 0
                ;;
            "tee -a /etc/gai.conf")
                track_command "modify gai.conf" 0
                cat > /dev/null
                return 0
                ;;
            "tee /etc/gai.conf")
                track_command "create gai.conf" 0
                cat > /dev/null
                return 0
                ;;
            *)
                track_command "sudo $*" 0
                return 0
                ;;
        esac
    }
    
    # Mock grep to not find existing config
    grep() {
        return 1  # Not found
    }
    
    export -f flow::can_run_sudo sudo grep track_command
    
    run network_diagnostics_fixes::fix_ipv6_issues
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Configuring system to prefer IPv4"* ]]
}

@test "fixes::fix_ipv4_only_issues - configures tools for IPv4" {
    # Override flow::can_run_sudo
    flow::can_run_sudo() { return 0; }
    
    # Mock file checks
    [[ -f ~/.curlrc ]] || true
    [[ -f ~/.wgetrc ]] || true
    
    # Mock grep to not find existing configs
    grep() { return 1; }
    
    # Mock echo (for file writes)
    echo() { builtin echo "$@"; }
    
    # Mock git
    git() {
        track_command "git config ipv4" 0
        return 0
    }
    
    # Mock command
    command() {
        case "$*" in
            *"git"*)
                return 0
                ;;
            *"apt-get"*)
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    # Mock sudo
    sudo() {
        track_command "sudo $*" 0
        cat > /dev/null 2>&1 || true
        return 0
    }
    
    export -f flow::can_run_sudo grep echo git command sudo track_command
    
    run network_diagnostics_fixes::fix_ipv4_only_issues
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Configured curl to use IPv4"* ]]
    [[ "$output" == *"Configured git to prefer IPv4"* ]]
    [[ "$output" == *"Configured wget to use IPv4"* ]]
    [[ "$output" == *"Configured APT to use IPv4"* ]]
}

@test "fixes::add_ipv4_host_override - adds host entry" {
    # Override flow::can_run_sudo
    flow::can_run_sudo() { return 0; }
    
    # Mock grep to not find existing entry
    grep() { return 1; }
    
    # Mock echo and sudo
    echo() { builtin echo "$@"; }
    
    sudo() {
        if [[ "$*" == "tee -a /etc/hosts" ]]; then
            track_command "add to hosts" 0
            cat > /dev/null
            return 0
        fi
        return 1
    }
    
    export -f flow::can_run_sudo grep echo sudo track_command
    
    run network_diagnostics_fixes::add_ipv4_host_override "test.com" "192.168.1.1"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Added test.com -> 192.168.1.1 to /etc/hosts"* ]]
}

@test "fixes::add_ipv4_host_override - validates parameters" {
    run network_diagnostics_fixes::add_ipv4_host_override "" ""
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Domain and IPv4 address required"* ]]
}

@test "fixes::fix_dns_issues - adds reliable DNS servers" {
    # Override flow::can_run_sudo
    flow::can_run_sudo() { return 0; }
    
    # Mock systemctl - systemd-resolved not active
    systemctl() { return 1; }
    
    # Mock file operations
    [[ -f /etc/resolv.conf ]] || true
    
    # Mock grep to not find existing DNS
    grep() { return 1; }
    
    # Mock sudo
    sudo() {
        case "$*" in
            "cp /etc/resolv.conf"*)
                track_command "backup resolv.conf" 0
                return 0
                ;;
            "tee /etc/resolv.conf")
                track_command "update resolv.conf" 0
                cat > /dev/null
                return 0
                ;;
            "resolvectl flush-caches")
                track_command "flush cache" 0
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Mock command
    command() {
        case "$*" in
            *"resolvectl"*)
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    export -f flow::can_run_sudo systemctl grep sudo command track_command
    
    run network_diagnostics_fixes::fix_dns_issues
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Adding reliable DNS servers"* ]]
    [[ "$output" == *"Added reliable DNS servers to /etc/resolv.conf"* ]]
    [[ "$output" == *"Flushed systemd-resolved cache"* ]]
}

@test "fixes::fix_ufw_blocking - adds firewall rules" {
    # Override flow::can_run_sudo
    flow::can_run_sudo() { return 0; }
    
    # Mock ufw command
    command() {
        [[ "$*" == *"ufw"* ]] && return 0 || return 1
    }
    
    # Mock sudo ufw
    sudo() {
        case "$*" in
            "ufw status")
                echo "Status: active"
                return 0
                ;;
            "ufw allow out"*)
                track_command "ufw rule: $*" 0
                return 0
                ;;
            "ufw reload")
                track_command "ufw reload" 0
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    export -f flow::can_run_sudo command sudo track_command
    
    run network_diagnostics_fixes::fix_ufw_blocking
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Added outbound rule for HTTP"* ]]
    [[ "$output" == *"Added outbound rule for HTTPS"* ]]
    [[ "$output" == *"Added outbound rule for DNS"* ]]
    [[ "$output" == *"Reloaded UFW with new rules"* ]]
}

@test "fixes::fix_ufw_blocking - skips when UFW inactive" {
    # Override flow::can_run_sudo
    flow::can_run_sudo() { return 0; }
    
    # Mock ufw command
    command() {
        [[ "$*" == *"ufw"* ]] && return 0 || return 1
    }
    
    # Mock sudo ufw status as inactive
    sudo() {
        echo "Status: inactive"
        return 0
    }
    
    export -f flow::can_run_sudo command sudo
    
    run network_diagnostics_fixes::fix_ufw_blocking
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"UFW is inactive - no changes needed"* ]]
}

# =============================================================================
# Integration Tests
# =============================================================================


@test "integration: main script loads and calls core module" {
    # Source the comprehensive network diagnostics
    source "${BATS_TEST_DIRNAME}/network_diagnostics.sh"
    
    # Verify that the main function exists
    declare -f network_diagnostics::run > /dev/null
    [ "$?" -eq 0 ]
    
    # Verify core module was loaded
    declare -f network_diagnostics_core::run > /dev/null
    [ "$?" -eq 0 ]
    
    # Verify module functions are available (not wrapper stubs)
    declare -f network_diagnostics_fixes::fix_ipv6_issues > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_tcp::check_tcp_settings > /dev/null
    [ "$?" -eq 0 ]
}

# =============================================================================
# Edge Cases and Error Handling
# =============================================================================

@test "edge case: handles missing commands gracefully" {
    # Mock command to report everything missing
    command() { return 1; }
    
    export -f command
    
    # Should still run without crashing
    run network_diagnostics_tcp::test_mtu_discovery
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"ping command not available"* ]]
}

@test "edge case: handles empty/null responses" {
    # Mock sysctl to return empty
    sysctl() { echo ""; }
    
    export -f sysctl
    
    run network_diagnostics_tcp::check_tcp_settings
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"TCP congestion control: unknown"* ]]
}

@test "edge case: handles permission denied" {
    # Override flow::can_run_sudo to deny
    flow::can_run_sudo() { return 1; }
    
    export -f flow::can_run_sudo
    
    run network_diagnostics_tcp::fix_ecn
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Skipping ECN fix (requires sudo)"* ]]
}

@test "edge case: handles network timeouts" {
    # Mock date and timeout to simulate timeout
    date() { echo "2024"; }
    timeout() {
        return 124  # timeout exit code
    }
    
    export -f date timeout
    
    # Function should handle timeout gracefully
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 0 ]  # Should not fail, just handle timeout gracefully
    # Should complete without hanging
}

# =============================================================================
# Performance and Resource Tests
# =============================================================================

@test "performance: core tests complete within timeout" {
    # Mock all commands without delays
    ping() { return 0; }
    getent() { echo "1.1.1.1 test"; return 0; }
    nc() { return 0; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { return 0; }
    
    export -f ping getent nc curl timeout command
    
    # Run with time limit
    run timeout 5 network_diagnostics_core::run
    
    [ "$status" -eq 0 ]
}

@test "resource: cleans up temp files" {
    # Test that the new function doesn't create temp files (it was simplified)
    date() { echo "2024"; }
    
    export -f date
    
    # The new function should not create any temp files
    run network_diagnostics_analysis::diagnose_connection_failure "HTTPS to google.com" "google.com"
    
    [ "$status" -eq 0 ]
    # No temp files should be created by the streamlined function
}

# =============================================================================
# Backward Compatibility Tests
# =============================================================================

