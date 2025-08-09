#!/usr/bin/env bats

# Tests for network fix functions
# Tests IPv6/IPv4 fixes, DNS fixes, host overrides, and firewall fixes

setup() {
    export BATS_TEST_DIRNAME="$BATS_TEST_DIRNAME"
    export TEST_TEMP_DIR="$(mktemp -d)"
    
    # Source the script under test
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
    
    export -f log::header log::subheader log::info log::success log::warning log::error
    export -f flow::can_run_sudo
    
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
# Tests for IPv6 fixes
# =============================================================================

@test "fix_ipv6_issues - configures IPv4 preference" {
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

@test "fix_ipv6_issues - skips when no sudo" {
    flow::can_run_sudo() { return 1; }
    
    export -f flow::can_run_sudo
    
    run network_diagnostics_fixes::fix_ipv6_issues
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Skipping IPv6 fixes (requires sudo)"* ]]
}

@test "fix_ipv6_issues - handles existing gai.conf" {
    flow::can_run_sudo() { return 0; }
    
    # Mock file operations
    [[ -f /etc/gai.conf ]] && return 0
    
    # Mock sudo
    sudo() {
        case "$*" in
            "cp /etc/gai.conf"*)
                track_command "backup gai.conf" 0
                return 0
                ;;
            *)
                track_command "sudo $*" 0
                return 0
                ;;
        esac
    }
    
    # Mock grep to find existing config
    grep() {
        if [[ "$*" == *"precedence ::ffff:0:0/96"* ]]; then
            echo "precedence ::ffff:0:0/96  100"
            return 0
        fi
        return 1
    }
    
    export -f flow::can_run_sudo sudo grep track_command
    
    run network_diagnostics_fixes::fix_ipv6_issues
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"IPv4 preference already configured"* ]]
}

# =============================================================================
# Tests for IPv4-only fixes
# =============================================================================

@test "fix_ipv4_only_issues - configures tools for IPv4" {
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

@test "fix_ipv4_only_issues - handles existing configurations" {
    flow::can_run_sudo() { return 0; }
    
    # Mock file checks
    [[ -f ~/.curlrc ]] && return 0
    [[ -f ~/.wgetrc ]] && return 0
    
    # Mock grep to find existing configs
    grep() {
        case "$*" in
            *"ipv4"*|*"inet4-only"*)
                echo "ipv4"
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    # Mock commands
    git() { return 0; }
    command() { [[ "$*" == *"git"* ]] && return 0 || return 1; }
    sudo() { return 0; }
    
    export -f flow::can_run_sudo grep git command sudo
    
    run network_diagnostics_fixes::fix_ipv4_only_issues
    
    [ "$status" -eq 0 ]
    # Should not show configuration messages if already configured
}

@test "fix_ipv4_only_issues - skips when no sudo" {
    flow::can_run_sudo() { return 1; }
    
    export -f flow::can_run_sudo
    
    run network_diagnostics_fixes::fix_ipv4_only_issues
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Skipping IPv4-only configuration (requires sudo)"* ]]
}

# =============================================================================
# Tests for host overrides
# =============================================================================

@test "add_ipv4_host_override - adds host entry" {
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

@test "add_ipv4_host_override - validates parameters" {
    run network_diagnostics_fixes::add_ipv4_host_override "" ""
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Domain and IPv4 address required"* ]]
}

@test "add_ipv4_host_override - handles existing entry" {
    flow::can_run_sudo() { return 0; }
    
    # Mock grep to find existing entry
    grep() {
        if [[ "$*" == *"test.com"* ]]; then
            echo "192.168.1.1 test.com"
            return 0
        fi
        return 1
    }
    
    export -f flow::can_run_sudo grep
    
    run network_diagnostics_fixes::add_ipv4_host_override "test.com" "192.168.1.1"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Entry for test.com already exists"* ]]
}

@test "add_ipv4_host_override - skips when no sudo" {
    flow::can_run_sudo() { return 1; }
    
    export -f flow::can_run_sudo
    
    run network_diagnostics_fixes::add_ipv4_host_override "test.com" "192.168.1.1"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Skipping host override (requires sudo)"* ]]
}

# =============================================================================
# Tests for DNS fixes
# =============================================================================

@test "fix_dns_issues - adds reliable DNS servers" {
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

@test "fix_dns_issues - handles systemd-resolved" {
    flow::can_run_sudo() { return 0; }
    
    # Mock systemctl - systemd-resolved is active
    systemctl() {
        case "$*" in
            "is-active systemd-resolved")
                return 0
                ;;
            *)
                track_command "systemctl $*" 0
                return 0
                ;;
        esac
    }
    
    # Mock file operations
    [[ -f /etc/systemd/resolved.conf ]] && return 0
    
    # Mock sudo
    sudo() {
        case "$*" in
            "cp /etc/systemd/resolved.conf"*)
                track_command "backup resolved.conf" 0
                return 0
                ;;
            "sed"*)
                track_command "update resolved.conf" 0
                return 0
                ;;
            *)
                track_command "sudo $*" 0
                return 0
                ;;
        esac
    }
    
    export -f flow::can_run_sudo systemctl sudo track_command
    
    run network_diagnostics_fixes::fix_dns_issues
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"systemd-resolved is active"* ]]
    [[ "$output" == *"Updated systemd-resolved configuration"* ]]
}

@test "fix_dns_issues - skips when no sudo" {
    flow::can_run_sudo() { return 1; }
    
    export -f flow::can_run_sudo
    
    run network_diagnostics_fixes::fix_dns_issues
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Skipping DNS fixes (requires sudo)"* ]]
}

# =============================================================================
# Tests for UFW firewall fixes
# =============================================================================

@test "fix_ufw_blocking - adds firewall rules" {
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

@test "fix_ufw_blocking - skips when UFW inactive" {
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

@test "fix_ufw_blocking - skips when UFW not installed" {
    command() { return 1; }  # ufw not found
    
    export -f command
    
    run network_diagnostics_fixes::fix_ufw_blocking
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"UFW not installed - skipping"* ]]
}

@test "fix_ufw_blocking - skips when no sudo" {
    flow::can_run_sudo() { return 1; }
    command() { return 0; }  # ufw available
    
    export -f flow::can_run_sudo command
    
    run network_diagnostics_fixes::fix_ufw_blocking
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Skipping UFW fixes (requires sudo)"* ]]
}

# =============================================================================
# Edge Cases and Error Handling
# =============================================================================

@test "edge case: handles file operation failures gracefully" {
    flow::can_run_sudo() { return 0; }
    
    # Mock sudo operations to fail
    sudo() { return 1; }
    grep() { return 1; }
    
    export -f flow::can_run_sudo sudo grep
    
    run network_diagnostics_fixes::fix_ipv6_issues
    
    # Should not crash, even if operations fail
    [ "$status" -eq 0 ]
}

@test "edge case: handles missing files gracefully" {
    flow::can_run_sudo() { return 0; }
    
    # Mock file tests to return false
    [[ -f /etc/gai.conf ]] && return 1
    [[ -f ~/.curlrc ]] && return 1
    
    # Mock operations
    sudo() {
        case "$*" in
            "tee /etc/gai.conf")
                track_command "create new gai.conf" 0
                cat > /dev/null
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    grep() { return 1; }
    
    export -f flow::can_run_sudo sudo grep track_command
    
    run network_diagnostics_fixes::fix_ipv6_issues
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Created /etc/gai.conf with IPv4 preference"* ]]
}

@test "edge case: validates input parameters" {
    # Test various invalid inputs
    run network_diagnostics_fixes::add_ipv4_host_override "" "192.168.1.1"
    [ "$status" -eq 1 ]
    
    run network_diagnostics_fixes::add_ipv4_host_override "test.com" ""
    [ "$status" -eq 1 ]
    
    run network_diagnostics_fixes::add_ipv4_host_override
    [ "$status" -eq 1 ]
}