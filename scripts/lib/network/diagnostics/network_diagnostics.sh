#!/usr/bin/env bash
# Unified Network Diagnostics Module
# Comprehensive network testing, analysis, and automated fixes in a single optimized module
set -eo pipefail

# Source dependencies
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"

# Define error codes if not already defined
: "${ERROR_NO_INTERNET:=5}"

# Set appropriate SUDO_MODE if not already set
if [[ -z "${SUDO_MODE:-}" ]]; then
    if [[ "${VROOLI_CONTEXT:-}" == "standalone" ]]; then
        export SUDO_MODE="skip"
    elif [[ $EUID -eq 0 ]] || sudo -n true 2>/dev/null; then
        export SUDO_MODE="error"
    else
        export SUDO_MODE="skip"
    fi
fi

# =============================================================================
# CORE DIAGNOSTICS - Essential network tests
# =============================================================================

network_diagnostics_core::run() {
    log::header "Running Core Network Tests"
    
    local critical_failure=false
    local has_failures=false
    
    # 1. Basic connectivity - IPv4 ping to public DNS
    log::info "Testing IPv4 connectivity..."
    if ping -4 -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
        log::success "  ✓ IPv4 ping to 8.8.8.8"
    else
        log::error "  ✗ IPv4 ping to 8.8.8.8"
        critical_failure=true
    fi
    
    # 2. Domain resolution
    if ping -4 -c 1 -W 2 google.com >/dev/null 2>&1; then
        log::success "  ✓ IPv4 ping to google.com"
    else
        log::error "  ✗ IPv4 ping to google.com"
        has_failures=true
    fi
    
    # 3. DNS lookup
    if getent hosts google.com >/dev/null 2>&1; then
        log::success "  ✓ DNS lookup (getent)"
    else
        log::error "  ✗ DNS lookup (getent)"
        has_failures=true
    fi
    
    # 4. TCP connectivity
    if nc -zv -w 2 google.com 443 >/dev/null 2>&1; then
        log::success "  ✓ TCP port 443 (HTTPS)"
    else
        log::error "  ✗ TCP port 443 (HTTPS)"
        has_failures=true
    fi
    
    # 5. HTTPS test
    if command -v curl >/dev/null 2>&1; then
        if timeout 8 curl -s --http1.1 --connect-timeout 5 https://www.google.com >/dev/null 2>&1; then
            log::success "  ✓ HTTPS to google.com"
        else
            log::error "  ✗ HTTPS to google.com"
            has_failures=true
        fi
    fi
    
    # Critical failure handling
    if [[ "$critical_failure" == "true" ]]; then
        log::error "Critical network issues detected - basic connectivity is broken"
        return "${ERROR_NO_INTERNET}"
    fi
    
    # Diagnose partial TLS issues
    if ping -4 -c 1 -W 2 google.com >/dev/null 2>&1 && \
       getent hosts google.com >/dev/null 2>&1 && \
       nc -zv -w 2 google.com 443 >/dev/null 2>&1 && \
       ! timeout 8 curl -s --http1.1 --connect-timeout 5 https://www.google.com >/dev/null 2>&1; then
        log::warning "PARTIAL TLS ISSUE: TCP connectivity works but HTTPS fails"
        log::info "This may indicate TCP segmentation offload or certificate issues"
    fi
    
    if [[ "$has_failures" == "true" ]]; then
        log::warning "Network issues detected - some tests failed"
        return 1
    fi
    
    log::success "All core network tests passed!"
    return 0
}

# Main diagnostic entry point (preserves external API)
network_diagnostics::run() {
    log::header "Running Comprehensive Network Diagnostics"
    
    # Run core tests
    if ! network_diagnostics_core::run; then
        # For critical failures, return immediately
        if [[ $? -eq "${ERROR_NO_INTERNET}" ]]; then
            return "${ERROR_NO_INTERNET}"
        fi
        
        # For non-critical failures, try to diagnose
        network_diagnostics_analysis::diagnose_connection_failure "Network tests" "google.com"
        
        log::info "Network issues detected but proceeding (many apps work offline)"
        return 0
    fi
    
    log::success "All network tests passed!"
    return 0
}

# =============================================================================
# TCP OPTIMIZATIONS - Performance tuning and fixes
# =============================================================================

network_diagnostics_tcp::make_tso_permanent() {
    local interface="${1:-}"
    
    if [[ -z "$interface" ]]; then
        log::error "Interface name required for TSO fix"
        return 1
    fi
    
    log::info "Making TSO offload fix permanent for $interface"
    
    # Method 1: systemd-networkd (preferred)
    if systemctl is-enabled systemd-networkd >/dev/null 2>&1; then
        log::info "Using systemd-networkd configuration"
        
        local config_file="/etc/systemd/network/99-tso-fix.network"
        if flow::can_run_sudo; then
            sudo tee "$config_file" >/dev/null <<EOF
[Match]
Name=$interface

[Link]
GenericSegmentationOffload=false
TCPSegmentationOffload=false
EOF
            log::success "Created $config_file"
            sudo systemctl restart systemd-networkd || true
            log::success "TCP offload fix is now permanent via systemd-networkd"
        else
            log::warning "Skipping permanent TSO fix (requires sudo)"
        fi
        return 0
    fi
    
    # Method 2: NetworkManager dispatcher
    if systemctl is-active NetworkManager >/dev/null 2>&1; then
        log::info "Using NetworkManager dispatcher"
        
        local script_file="/etc/NetworkManager/dispatcher.d/99-tso-fix"
        if flow::can_run_sudo; then
            sudo tee "$script_file" >/dev/null <<'EOF'
#!/bin/bash
if [[ "$1" == "$interface" ]] && [[ "$2" == "up" ]]; then
    ethtool -K "$1" tso off gso off 2>/dev/null || true
fi
EOF
            sudo chmod +x "$script_file"
            log::success "Created NetworkManager dispatcher script"
            log::success "TCP offload fix is now permanent via NetworkManager"
        else
            log::warning "Skipping permanent TSO fix (requires sudo)"
        fi
        return 0
    fi
    
    log::warning "No supported network manager found for permanent TSO fix"
    return 1
}

network_diagnostics_tcp::check_tcp_settings() {
    log::info "Checking TCP settings..."
    
    # Check congestion control
    local congestion
    congestion=$(sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null || echo "unknown")
    log::info "  TCP congestion control: $congestion"
    
    # Check keepalive
    local keepalive
    keepalive=$(sysctl -n net.ipv4.tcp_keepalive_time 2>/dev/null || echo "unknown")
    log::info "  TCP keepalive time: $keepalive seconds"
    
    # Check window scaling
    local window_scaling
    window_scaling=$(sysctl -n net.ipv4.tcp_window_scaling 2>/dev/null || echo "unknown")
    log::info "  TCP window scaling: $window_scaling"
    
    # Check timestamps
    local timestamps
    timestamps=$(sysctl -n net.ipv4.tcp_timestamps 2>/dev/null || echo "unknown")
    log::info "  TCP timestamps: $timestamps"
    
    # Check ECN
    local ecn
    ecn=$(sysctl -n net.ipv4.tcp_ecn 2>/dev/null || echo "unknown")
    log::info "  TCP ECN: $ecn"
    
    return 0
}

network_diagnostics_tcp::fix_ecn() {
    log::info "Checking TCP ECN settings..."
    
    if ! flow::can_run_sudo; then
        log::warning "Skipping ECN fix (requires sudo)"
        return 1
    fi
    
    # Disable ECN as it can cause issues with some networks
    sudo sysctl -w net.ipv4.tcp_ecn=0 >/dev/null 2>&1
    log::success "Disabled TCP ECN"
    
    # Make permanent
    echo "net.ipv4.tcp_ecn = 0" | sudo tee -a /etc/sysctl.conf >/dev/null
    log::success "Made ECN change permanent"
    
    return 0
}

network_diagnostics_tcp::fix_mtu_size() {
    local interface="${1:-eth0}"
    local mtu="${2:-1400}"
    
    log::info "Setting MTU to $mtu on $interface"
    
    if ! flow::can_run_sudo; then
        log::warning "Skipping MTU fix (requires sudo)"
        return 1
    fi
    
    # Set MTU
    sudo ip link set dev "$interface" mtu "$mtu"
    log::success "Set MTU to $mtu on $interface"
    
    # Make permanent via systemd-networkd if available
    if systemctl is-enabled systemd-networkd >/dev/null 2>&1; then
        local config_file="/etc/systemd/network/99-mtu.network"
        sudo tee "$config_file" >/dev/null <<EOF
[Match]
Name=$interface

[Link]
MTUBytes=$mtu
EOF
        log::success "Made MTU change permanent"
    fi
    
    return 0
}

network_diagnostics_tcp::fix_pmtu_probing() {
    log::info "Enabling Path MTU discovery probing..."
    
    if ! flow::can_run_sudo; then
        log::warning "Skipping PMTU fix (requires sudo)"
        return 1
    fi
    
    # Enable PMTU probing
    sudo sysctl -w net.ipv4.tcp_mtu_probing=1 >/dev/null 2>&1
    log::success "Enabled PMTU probing"
    
    # Set base MSS
    sudo sysctl -w net.ipv4.tcp_base_mss=1024 >/dev/null 2>&1
    log::success "Set base MSS to 1024"
    
    # Make permanent
    {
        echo "net.ipv4.tcp_mtu_probing = 1"
        echo "net.ipv4.tcp_base_mss = 1024"
    } | sudo tee -a /etc/sysctl.conf >/dev/null
    log::success "Made PMTU changes permanent"
    
    return 0
}

network_diagnostics_tcp::test_mtu_discovery() {
    local target="${1:-google.com}"
    
    if ! command -v ping >/dev/null 2>&1; then
        log::error "ping command not available"
        return 1
    fi
    
    log::info "Testing MTU discovery to $target..."
    
    # Test different MTU sizes (subtract 28 for headers)
    local mtus=(1500 1400 1280 1024)
    local working_mtu=0
    
    for mtu in "${mtus[@]}"; do
        local packet_size=$((mtu - 28))
        if ping -M do -s "$packet_size" -c 1 -W 2 "$target" >/dev/null 2>&1; then
            working_mtu=$mtu
            break
        fi
    done
    
    if [[ $working_mtu -gt 0 ]]; then
        log::success "Maximum working MTU to $target: $working_mtu bytes"
    else
        log::warning "Could not determine working MTU to $target"
    fi
    
    return 0
}

network_diagnostics_tcp::check_pmtu_status() {
    log::info "Checking Path MTU discovery status..."
    
    local mtu_probing
    mtu_probing=$(sysctl -n net.ipv4.tcp_mtu_probing 2>/dev/null || echo "unknown")
    log::info "  TCP MTU probing: $mtu_probing"
    
    local base_mss
    base_mss=$(sysctl -n net.ipv4.tcp_base_mss 2>/dev/null || echo "unknown")
    log::info "  TCP base MSS: $base_mss"
    
    return 0
}

# =============================================================================
# ANALYSIS - Diagnose connection failures
# =============================================================================

network_diagnostics_analysis::diagnose_connection_failure() {
    local failed_test="${1:-Network tests}"
    local target_domain="${2:-google.com}"
    
    log::info "🔍 Diagnosing connection failure: $failed_test"
    
    # 1. Check system time (critical for TLS)
    local current_year
    current_year=$(date +%Y)
    if [[ $current_year -lt 2020 ]] || [[ $current_year -gt 2030 ]]; then
        log::error "  ❌ System time is incorrect (year: $current_year)"
        log::info "  💡 Fix: Set correct system time - this causes TLS certificate failures"
        return 1
    fi
    
    # 2. Check IPv4 vs IPv6
    if [[ "$failed_test" == *"HTTPS"* ]] || [[ "$failed_test" == *"ping"* ]]; then
        local ipv4_works=false
        local ipv6_works=false
        
        ping -4 -c 1 -W 2 "$target_domain" >/dev/null 2>&1 && ipv4_works=true
        ping -6 -c 1 -W 2 "$target_domain" >/dev/null 2>&1 && ipv6_works=true
        
        if [[ "$ipv4_works" == "true" ]] && [[ "$ipv6_works" == "false" ]]; then
            log::info "  ✅ IPv4 works, IPv6 doesn't - this is normal"
        elif [[ "$ipv4_works" == "false" ]] && [[ "$ipv6_works" == "true" ]]; then
            log::warning "  ⚠️  Only IPv6 works - may need IPv4 preference fix"
        elif [[ "$ipv4_works" == "false" ]] && [[ "$ipv6_works" == "false" ]]; then
            log::error "  ❌ Neither IPv4 nor IPv6 work to $target_domain"
        fi
    fi
    
    # 3. TLS handshake check
    if [[ "$failed_test" == *"HTTPS"* ]] && command -v openssl >/dev/null 2>&1; then
        log::info "  🔍 Testing TLS handshake..."
        if timeout 5 openssl s_client -connect "$target_domain:443" -verify_return_error </dev/null >/dev/null 2>&1; then
            log::success "  ✅ TLS handshake works"
        else
            log::error "  ❌ TLS handshake failed"
        fi
    fi
    
    # 4. DNS resolution check
    if [[ "$failed_test" == *"DNS"* ]] || [[ "$failed_test" == *"getent"* ]]; then
        log::info "  🔍 Testing DNS resolution..."
        if getent hosts 8.8.8.8 >/dev/null 2>&1; then
            log::info "  ✅ DNS system works"
        else
            log::error "  ❌ DNS system appears broken"
        fi
    fi
    
    return 0
}

# =============================================================================
# FIXES - Automated network issue remediation
# =============================================================================

network_diagnostics_fixes::fix_ipv6_issues() {
    log::info "Configuring system to prefer IPv4 over IPv6..."
    
    if ! flow::can_run_sudo; then
        log::warning "Skipping IPv6 fix (requires sudo)"
        return 1
    fi
    
    # Configure gai.conf to prefer IPv4
    local gai_conf="/etc/gai.conf"
    
    # Backup existing config
    if [[ -f "$gai_conf" ]]; then
        sudo cp "$gai_conf" "${gai_conf}.backup.$(date +%Y%m%d_%H%M%S)"
    fi
    
    # Add IPv4 preference if not already present
    if ! grep -q "precedence ::ffff:0:0/96" "$gai_conf" 2>/dev/null; then
        echo "precedence ::ffff:0:0/96  100" | sudo tee -a "$gai_conf" >/dev/null
        log::success "Added IPv4 preference to $gai_conf"
    else
        log::info "IPv4 preference already configured"
    fi
    
    log::success "System configured to prefer IPv4"
    return 0
}

network_diagnostics_fixes::fix_ipv4_only_issues() {
    log::info "Configuring tools for IPv4-only mode..."
    
    # Configure curl
    if ! grep -q "ipv4" ~/.curlrc 2>/dev/null; then
        echo "ipv4" >> ~/.curlrc
        log::success "Configured curl to use IPv4"
    fi
    
    # Configure git
    if command -v git >/dev/null 2>&1; then
        git config --global url."https://".insteadOf git://
        git config --global core.gitProxy ""
        log::success "Configured git to prefer IPv4"
    fi
    
    # Configure wget
    if ! grep -q "inet4-only" ~/.wgetrc 2>/dev/null; then
        echo "inet4-only = on" >> ~/.wgetrc
        log::success "Configured wget to use IPv4"
    fi
    
    # Configure APT if on Debian/Ubuntu
    if command -v apt-get >/dev/null 2>&1 && flow::can_run_sudo; then
        echo 'Acquire::ForceIPv4 "true";' | sudo tee /etc/apt/apt.conf.d/99force-ipv4 >/dev/null
        log::success "Configured APT to use IPv4"
    fi
    
    return 0
}

network_diagnostics_fixes::add_ipv4_host_override() {
    local domain="$1"
    local ipv4_address="$2"
    
    if [[ -z "$domain" ]] || [[ -z "$ipv4_address" ]]; then
        log::error "Domain and IPv4 address required"
        return 1
    fi
    
    if ! flow::can_run_sudo; then
        log::warning "Skipping host override (requires sudo)"
        return 1
    fi
    
    # Check if entry already exists
    if ! grep -q "$domain" /etc/hosts 2>/dev/null; then
        echo "$ipv4_address $domain" | sudo tee -a /etc/hosts >/dev/null
        log::success "Added $domain -> $ipv4_address to /etc/hosts"
    else
        log::info "Host entry for $domain already exists"
    fi
    
    return 0
}

network_diagnostics_fixes::fix_dns_issues() {
    log::info "Adding reliable DNS servers..."
    
    if ! flow::can_run_sudo; then
        log::warning "Skipping DNS fix (requires sudo)"
        return 1
    fi
    
    # Check if using systemd-resolved
    if systemctl is-active systemd-resolved >/dev/null 2>&1; then
        # Configure systemd-resolved
        sudo mkdir -p /etc/systemd/resolved.conf.d/
        sudo tee /etc/systemd/resolved.conf.d/dns.conf >/dev/null <<EOF
[Resolve]
DNS=8.8.8.8 8.8.4.4 1.1.1.1
FallbackDNS=9.9.9.9
EOF
        sudo systemctl restart systemd-resolved
        log::success "Configured systemd-resolved with reliable DNS servers"
    else
        # Traditional resolv.conf approach
        local resolv_conf="/etc/resolv.conf"
        
        # Backup existing config
        sudo cp "$resolv_conf" "${resolv_conf}.backup.$(date +%Y%m%d_%H%M%S)"
        
        # Add DNS servers if not present
        for dns in "8.8.8.8" "8.8.4.4" "1.1.1.1"; do
            if ! grep -q "nameserver $dns" "$resolv_conf" 2>/dev/null; then
                echo "nameserver $dns" | sudo tee -a "$resolv_conf" >/dev/null
            fi
        done
        
        log::success "Added reliable DNS servers to /etc/resolv.conf"
    fi
    
    # Flush DNS cache if possible
    if command -v resolvectl >/dev/null 2>&1; then
        sudo resolvectl flush-caches 2>/dev/null || true
        log::success "Flushed systemd-resolved cache"
    fi
    
    return 0
}

network_diagnostics_fixes::fix_ufw_blocking() {
    log::info "Checking firewall configuration..."
    
    if ! command -v ufw >/dev/null 2>&1; then
        log::info "UFW not installed - skipping firewall check"
        return 0
    fi
    
    if ! flow::can_run_sudo; then
        log::warning "Skipping firewall fix (requires sudo)"
        return 1
    fi
    
    # Check UFW status
    local ufw_status
    ufw_status=$(sudo ufw status | grep -i "status:" || echo "")
    
    if [[ "$ufw_status" != *"active"* ]]; then
        log::info "UFW is inactive - no changes needed"
        return 0
    fi
    
    log::info "Adding firewall rules for outbound connections..."
    
    # Allow outbound HTTP, HTTPS, and DNS
    sudo ufw allow out 80/tcp comment 'HTTP'
    log::success "Added outbound rule for HTTP"
    
    sudo ufw allow out 443/tcp comment 'HTTPS'
    log::success "Added outbound rule for HTTPS"
    
    sudo ufw allow out 53 comment 'DNS'
    log::success "Added outbound rule for DNS"
    
    # Reload UFW
    sudo ufw reload
    log::success "Reloaded UFW with new rules"
    
    return 0
}

# =============================================================================
# EXPORTS - Make functions available to external scripts
# =============================================================================

# Export all functions for external use
export -f network_diagnostics::run
export -f network_diagnostics_core::run
export -f network_diagnostics_tcp::make_tso_permanent
export -f network_diagnostics_tcp::check_tcp_settings
export -f network_diagnostics_tcp::fix_ecn
export -f network_diagnostics_tcp::fix_mtu_size
export -f network_diagnostics_tcp::fix_pmtu_probing
export -f network_diagnostics_tcp::test_mtu_discovery
export -f network_diagnostics_tcp::check_pmtu_status
export -f network_diagnostics_analysis::diagnose_connection_failure
export -f network_diagnostics_fixes::fix_ipv6_issues
export -f network_diagnostics_fixes::fix_ipv4_only_issues
export -f network_diagnostics_fixes::add_ipv4_host_override
export -f network_diagnostics_fixes::fix_dns_issues
export -f network_diagnostics_fixes::fix_ufw_blocking

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    network_diagnostics::run "$@"
fi