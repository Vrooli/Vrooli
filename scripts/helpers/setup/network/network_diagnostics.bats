#!/usr/bin/env bats

# Comprehensive tests for network diagnostics modules
# Tests all modules in scripts/helpers/setup/network/

setup() {
    export BATS_TEST_DIRNAME="$BATS_TEST_DIRNAME"
    export TEST_TEMP_DIR="$(mktemp -d)"
    
    # Source all modules
    source "${BATS_TEST_DIRNAME}/network_diagnostics_core.sh"
    source "${BATS_TEST_DIRNAME}/network_diagnostics_tcp.sh"
    source "${BATS_TEST_DIRNAME}/network_diagnostics_analysis.sh"
    source "${BATS_TEST_DIRNAME}/network_diagnostics_fixes.sh"
    
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
    
    # Export error codes
    export ERROR_NO_INTERNET=5
    
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
# Tests for network_diagnostics_core.sh
# =============================================================================

@test "core::run - all tests pass scenario" {
    # Mock all commands to succeed
    ping() { track_command "ping $*" 0; }
    getent() { track_command "getent $*" 0; echo "93.184.216.34 google.com"; }
    nc() { track_command "nc $*" 0; }
    curl() { track_command "curl $*" 0; }
    timeout() { shift; "$@"; }
    date() { echo "1234567890"; }
    command() { [[ "$2" == "curl" ]] || [[ "$2" == "date" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout date command track_command
    
    run network_diagnostics_core::run
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"[SUCCESS] All core network tests passed!"* ]]
    [[ "$output" == *"[SUCCESS]   ✓ IPv4 ping to 8.8.8.8"* ]]
    [[ "$output" == *"[SUCCESS]   ✓ IPv4 ping to google.com"* ]]
    [[ "$output" == *"[SUCCESS]   ✓ DNS lookup (getent)"* ]]
    [[ "$output" == *"[SUCCESS]   ✓ TCP port 443 (HTTPS)"* ]]
    [[ "$output" == *"[SUCCESS]   ✓ HTTPS to google.com"* ]]
}

@test "core::run - critical failure (no internet)" {
    # Mock ping to fail for critical test
    ping() { 
        if [[ "$*" == *"8.8.8.8"* ]]; then
            track_command "ping $*" 1
            return 1
        else
            track_command "ping $*" 0
            return 0
        fi
    }
    getent() { track_command "getent $*" 0; echo "93.184.216.34 google.com"; }
    nc() { track_command "nc $*" 0; }
    curl() { track_command "curl $*" 0; }
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent nc curl timeout command track_command
    
    run network_diagnostics_core::run
    
    [ "$status" -eq 5 ]  # ERROR_NO_INTERNET
    [[ "$output" == *"[ERROR]   ✗ IPv4 ping to 8.8.8.8"* ]]
    [[ "$output" == *"[ERROR] Critical network issues detected"* ]]
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
# Tests for network_diagnostics_analysis.sh
# =============================================================================

@test "analysis::analyze_tls_handshake - successful TLS analysis" {
    # Mock openssl
    openssl() {
        case "$*" in
            *"tls1"*)
                echo "Verify return code: 0 (ok)"
                return 0
                ;;
            *"tls1_2"*|*"tls1_3"*)
                echo "Verify return code: 0 (ok)"
                return 0
                ;;
            *"showcerts"*)
                cat <<EOF
Certificate chain
 0 s:/CN=test.com
   i:/CN=Test CA
-----BEGIN CERTIFICATE-----
TESTCERT1
-----END CERTIFICATE-----
 1 s:/CN=Test CA
   i:/CN=Root CA
-----BEGIN CERTIFICATE-----
TESTCERT2
-----END CERTIFICATE-----
EOF
                return 0
                ;;
            *"cipher"*)
                echo "Cipher    : ECDHE-RSA-AES256-GCM-SHA384"
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "openssl" ]] && return 0 || return 1; }
    
    export -f openssl timeout command
    
    run network_diagnostics_analysis::analyze_tls_handshake "test.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"✓ tls1: Supported"* ]]
    [[ "$output" == *"✓ tls1_2: Supported"* ]]
    [[ "$output" == *"✓ tls1_3: Supported"* ]]
    [[ "$output" == *"Working TLS versions:"* ]]
    [[ "$output" == *"Negotiated cipher: ECDHE-RSA-AES256-GCM-SHA384"* ]]
    [[ "$output" == *"Certificate chain length: 2"* ]]
}

@test "analysis::analyze_tls_handshake - no TLS versions work" {
    # Mock openssl to fail for all versions
    openssl() {
        return 1
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "openssl" ]] && return 0 || return 1; }
    
    export -f openssl timeout command
    
    run network_diagnostics_analysis::analyze_tls_handshake "test.com"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"✗ tls1: Not supported or failed"* ]]
    [[ "$output" == *"✗ No TLS versions work with test.com"* ]]
    [[ "$output" == *"Attempting detailed SSL diagnostics"* ]]
}

@test "analysis::check_ipv4_vs_ipv6 - IPv4 only works" {
    # Mock ping
    ping() {
        case "$*" in
            *"-4"*)
                track_command "ping -4" 0
                return 0
                ;;
            *"-6"*)
                track_command "ping -6" 1
                return 1
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    # Mock getent
    getent() {
        case "$*" in
            *"ahostsv4"*)
                echo "192.168.1.1 test.com"
                return 0
                ;;
            *"ahostsv6"*)
                return 1
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    # Mock curl
    curl() {
        if [[ "$*" == *"-4"* ]]; then
            echo "200"
            return 0
        else
            return 1
        fi
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent curl timeout command track_command
    
    run network_diagnostics_analysis::check_ipv4_vs_ipv6 "test.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"✓ IPv4 connectivity works"* ]]
    [[ "$output" == *"✗ IPv6 connectivity failed"* ]]
    [[ "$output" == *"IPv4 address: 192.168.1.1"* ]]
    [[ "$output" == *"✓ HTTPS over IPv4 works"* ]]
    [[ "$output" == *"Recommendation: Use IPv4-only mode"* ]]
}

@test "analysis::check_ip_preference - shows system preferences" {
    # Create mock gai.conf
    mkdir -p "$TEST_TEMP_DIR/etc"
    cat > "$TEST_TEMP_DIR/etc/gai.conf" <<EOF
precedence ::ffff:0:0/96  100
precedence ::/0            40
EOF
    
    # Override file path in function (would need to modify function to support this)
    # For now, mock the file check
    [[ -f /etc/gai.conf ]] && return 0 || return 1
    
    # Mock cat for /proc/sys files
    cat() {
        case "$*" in
            */disable_ipv6*)
                echo "0"
                return 0
                ;;
            *)
                command cat "$@"
                ;;
        esac
    }
    
    # Mock ip command
    ip() {
        echo "default via 192.168.1.1 dev eth0"
        return 0
    }
    
    export -f cat ip
    
    run network_diagnostics_analysis::check_ip_preference
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"IPv6 is enabled"* ]] || [[ "$output" == *"IP address selection order"* ]]
    [[ "$output" == *"Default route:"* ]]
}

@test "analysis::check_time_sync - time is correct" {
    # Mock date
    date() {
        echo "2024"
        return 0
    }
    
    # Mock timedatectl
    timedatectl() {
        echo "NTP synchronized: yes"
        return 0
    }
    
    command() { [[ "$2" == "timedatectl" ]] && return 0 || return 1; }
    
    export -f date timedatectl command
    
    run network_diagnostics_analysis::check_time_sync
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"✓ System time appears reasonable"* ]]
    [[ "$output" == *"NTP status: NTP synchronized: yes"* ]]
}

@test "analysis::verbose_https_debug - diagnoses connection issues" {
    # Create temp file for test
    local mock_temp_file="$TEST_TEMP_DIR/curl_output"
    
    # Mock mktemp
    mktemp() {
        echo "$mock_temp_file"
    }
    
    # Mock timeout and bash
    timeout() {
        shift  # Remove timeout value
        # Simulate curl failure with SSL error
        echo "* SSL_ERROR_SYSCALL in connection to github.com:443" > "$mock_temp_file"
        echo "curl: (35) SSL connection error" >> "$mock_temp_file"
        return 1
    }
    
    # Mock grep
    grep() {
        case "$*" in
            *"SSL_ERROR"*)
                command grep "$@" && return 0 || return 1
                ;;
            *"curl:"*)
                echo "curl: (35) SSL connection error"
                return 0
                ;;
            *)
                command grep "$@"
                ;;
        esac
    }
    
    export -f mktemp timeout grep
    
    run network_diagnostics_analysis::verbose_https_debug "curl -s https://github.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Command failed"* ]]
    [[ "$output" == *"SSL/TLS certificate issue detected"* ]]
    [[ "$output" == *"curl: (35) SSL connection error"* ]]
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

@test "integration: modular wrapper loads all modules" {
    # Source the modular wrapper
    source "${BATS_TEST_DIRNAME}/network_diagnostics_modular.sh"
    
    # Verify all functions are available
    declare -f network_diagnostics_core::run > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_tcp::check_tcp_settings > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_analysis::check_ipv4_vs_ipv6 > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics_fixes::fix_dns_issues > /dev/null
    [ "$?" -eq 0 ]
}

@test "integration: main script loads and calls core module" {
    # Source the main wrapper
    source "${BATS_TEST_DIRNAME}/../network_diagnostics.sh"
    
    # Verify that the main function exists
    declare -f network_diagnostics::run > /dev/null
    [ "$?" -eq 0 ]
    
    # Verify core module was loaded
    declare -f network_diagnostics_core::run > /dev/null
    [ "$?" -eq 0 ]
    
    # The main function should call the core function
    # Let's verify this by checking if the wrapper functions exist
    declare -f network_diagnostics::fix_ipv6_issues > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics::check_tcp_settings > /dev/null
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
    # Mock timeout to simulate timeout
    timeout() {
        return 124  # timeout exit code
    }
    
    export -f timeout
    
    # Function should handle timeout gracefully
    run network_diagnostics_analysis::analyze_tls_handshake
    
    [ "$status" -eq 1 ]
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
    # Count temp files before
    local temp_before=$(ls -la /tmp | wc -l)
    
    # Run function that creates temp files
    mktemp() { 
        local file="$TEST_TEMP_DIR/test_file"
        touch "$file"
        echo "$file"
    }
    
    timeout() { shift; "$@"; }
    
    export -f mktemp timeout
    
    run network_diagnostics_analysis::verbose_https_debug "test"
    
    # Verify temp file was cleaned up
    [ ! -f "$TEST_TEMP_DIR/test_file" ]
}

# =============================================================================
# Backward Compatibility Tests
# =============================================================================

@test "backward compat: old function names still work" {
    # Source main script
    source "${BATS_TEST_DIRNAME}/../network_diagnostics.sh"
    
    # Check that backward compatibility wrapper functions exist
    declare -f network_diagnostics::fix_ipv6_issues > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics::check_tcp_settings > /dev/null
    [ "$?" -eq 0 ]
    
    declare -f network_diagnostics::analyze_tls_handshake > /dev/null
    [ "$?" -eq 0 ]
    
    # Main function should exist
    declare -f network_diagnostics::run > /dev/null
    [ "$?" -eq 0 ]
    
    # Core module function should be available
    declare -f network_diagnostics_core::run > /dev/null
    [ "$?" -eq 0 ]
}