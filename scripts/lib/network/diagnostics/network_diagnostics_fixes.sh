#!/usr/bin/env bash
# Network fix functions for diagnostics
# Handles IPv6/IPv4 fixes, DNS fixes, and host overrides
set -eo pipefail

# Script directory
LIB_NETWORK_DIAGNOSTICS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${LIB_NETWORK_DIAGNOSTICS_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"

# Fix IPv6 issues
network_diagnostics_fixes::fix_ipv6_issues() {
    log::info "Attempting to fix IPv6-related issues..."
    
    if ! flow::can_run_sudo "fix IPv6 issues"; then
        log::info "  → Skipping IPv6 fixes (requires sudo)"
        return 1
    fi
    
    # Option 1: Prefer IPv4 over IPv6 (modify gai.conf)
    log::info "  → Configuring system to prefer IPv4..."
    
    local gai_conf="/etc/gai.conf"
    if [[ -f "$gai_conf" ]]; then
        # Backup original
        sudo cp "$gai_conf" "${gai_conf}.backup" 2>/dev/null || true
        
        # Add IPv4 preference if not already present
        if ! grep -q "precedence ::ffff:0:0/96" "$gai_conf" 2>/dev/null; then
            if sudo tee -a "$gai_conf" >/dev/null 2>&1 <<EOF

# Prefer IPv4 over IPv6 (added by network diagnostics)
precedence ::ffff:0:0/96  100
EOF
            then
                log::success "  ✓ Modified $gai_conf to prefer IPv4"
            fi
        else
            log::info "  → IPv4 preference already configured"
        fi
    else
        # Create gai.conf with IPv4 preference
        if sudo tee "$gai_conf" >/dev/null 2>&1 <<EOF
# Prefer IPv4 over IPv6
precedence ::ffff:0:0/96  100
precedence ::/0            40
precedence 2002::/16       30
precedence ::/96           20
precedence ::1/128         10
EOF
        then
            log::success "  ✓ Created $gai_conf with IPv4 preference"
        fi
    fi
    
    # Option 2: Disable IPv6 completely (more drastic)
    local disable_ipv6=false
    
    if [[ "$disable_ipv6" == "true" ]]; then
        log::info "  → Disabling IPv6 completely..."
        
        # Disable via sysctl
        if sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1 >/dev/null 2>&1; then
            log::success "  ✓ Disabled IPv6 via sysctl"
        fi
        
        if sudo sysctl -w net.ipv6.conf.default.disable_ipv6=1 >/dev/null 2>&1; then
            log::success "  ✓ Disabled IPv6 for new interfaces"
        fi
        
        if sudo sysctl -w net.ipv6.conf.lo.disable_ipv6=1 >/dev/null 2>&1; then
            log::success "  ✓ Disabled IPv6 on loopback"
        fi
        
        # Make permanent
        if sudo tee -a /etc/sysctl.conf >/dev/null 2>&1 <<EOF
# Disable IPv6 (added by network diagnostics)
net.ipv6.conf.all.disable_ipv6=1
net.ipv6.conf.default.disable_ipv6=1
net.ipv6.conf.lo.disable_ipv6=1
EOF
        then
            log::success "  ✓ Made IPv6 disable permanent"
        fi
    fi
    
    return 0
}

# Fix IPv4-only issues
network_diagnostics_fixes::fix_ipv4_only_issues() {
    log::info "Configuring system for IPv4-only operation..."
    
    if ! flow::can_run_sudo "configure IPv4-only mode"; then
        log::info "  → Skipping IPv4-only configuration (requires sudo)"
        return 1
    fi
    
    # Force curl to use IPv4
    if [[ ! -f ~/.curlrc ]] || ! grep -q "ipv4" ~/.curlrc 2>/dev/null; then
        echo "ipv4" >> ~/.curlrc
        log::success "  ✓ Configured curl to use IPv4 by default"
    fi
    
    # Configure git to use IPv4
    if command -v git >/dev/null 2>&1; then
        git config --global core.ipv4 true 2>/dev/null || true
        log::success "  ✓ Configured git to prefer IPv4"
    fi
    
    # Configure wget to use IPv4
    if [[ ! -f ~/.wgetrc ]] || ! grep -q "inet4-only" ~/.wgetrc 2>/dev/null; then
        echo "inet4-only = on" >> ~/.wgetrc
        log::success "  ✓ Configured wget to use IPv4 only"
    fi
    
    # Configure APT to use IPv4 (if on Debian/Ubuntu)
    if [[ -f /etc/apt/apt.conf.d/99force-ipv4 ]] || command -v apt-get >/dev/null 2>&1; then
        if sudo tee /etc/apt/apt.conf.d/99force-ipv4 >/dev/null 2>&1 <<EOF
Acquire::ForceIPv4 "true";
EOF
        then
            log::success "  ✓ Configured APT to use IPv4 only"
        fi
    fi
    
    return 0
}

# Add IPv4 host override
network_diagnostics_fixes::add_ipv4_host_override() {
    local domain="$1"
    local ipv4_addr="$2"
    
    if [[ -z "$domain" ]] || [[ -z "$ipv4_addr" ]]; then
        log::error "  ✗ Domain and IPv4 address required"
        return 1
    fi
    
    log::info "Adding host override for $domain -> $ipv4_addr..."
    
    if ! flow::can_run_sudo "modify /etc/hosts"; then
        log::info "  → Skipping host override (requires sudo)"
        return 1
    fi
    
    # Check if entry already exists
    if grep -q "$domain" /etc/hosts 2>/dev/null; then
        log::info "  → Entry for $domain already exists in /etc/hosts"
        return 0
    fi
    
    # Add entry
    if echo "$ipv4_addr $domain" | sudo tee -a /etc/hosts >/dev/null 2>&1; then
        log::success "  ✓ Added $domain -> $ipv4_addr to /etc/hosts"
        return 0
    else
        log::error "  ✗ Failed to modify /etc/hosts"
        return 1
    fi
}

# Fix DNS issues
network_diagnostics_fixes::fix_dns_issues() {
    log::info "Attempting to fix DNS resolution issues..."
    
    if ! flow::can_run_sudo "fix DNS"; then
        log::info "  → Skipping DNS fixes (requires sudo)"
        return 1
    fi
    
    # Option 1: Add reliable DNS servers
    log::info "  → Adding reliable DNS servers..."
    
    local resolv_conf="/etc/resolv.conf"
    local resolv_backup="${resolv_conf}.backup"
    
    # Backup current resolv.conf
    if [[ -f "$resolv_conf" ]]; then
        sudo cp "$resolv_conf" "$resolv_backup" 2>/dev/null || true
    fi
    
    # Check if systemd-resolved is managing DNS
    if systemctl is-active systemd-resolved >/dev/null 2>&1; then
        log::info "  → systemd-resolved is active, updating its configuration..."
        
        # Update systemd-resolved configuration
        local resolved_conf="/etc/systemd/resolved.conf"
        if [[ -f "$resolved_conf" ]]; then
            sudo cp "$resolved_conf" "${resolved_conf}.backup" 2>/dev/null || true
            
            # Add DNS servers to resolved.conf
            if sudo sed -i 's/^#DNS=.*/DNS=8.8.8.8 8.8.4.4 1.1.1.1/' "$resolved_conf" 2>/dev/null; then
                log::success "  ✓ Updated systemd-resolved configuration"
                sudo systemctl restart systemd-resolved 2>/dev/null || true
            fi
        fi
    else
        # Directly modify resolv.conf
        if ! grep -q "8.8.8.8\|1.1.1.1" "$resolv_conf" 2>/dev/null; then
            if sudo tee "$resolv_conf" >/dev/null 2>&1 <<EOF
# Reliable DNS servers (added by network diagnostics)
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
nameserver 1.0.0.1
EOF
            then
                log::success "  ✓ Added reliable DNS servers to $resolv_conf"
            fi
        else
            log::info "  → Reliable DNS servers already configured"
        fi
    fi
    
    # Flush DNS cache
    log::info "  → Flushing DNS cache..."
    
    # systemd-resolved
    if command -v resolvectl >/dev/null 2>&1; then
        sudo resolvectl flush-caches 2>/dev/null || true
        log::success "  ✓ Flushed systemd-resolved cache"
    fi
    
    # nscd
    if command -v nscd >/dev/null 2>&1; then
        sudo nscd -i hosts 2>/dev/null || true
        log::success "  ✓ Flushed nscd cache"
    fi
    
    # dnsmasq
    if systemctl is-active dnsmasq >/dev/null 2>&1; then
        sudo systemctl restart dnsmasq 2>/dev/null || true
        log::success "  ✓ Restarted dnsmasq"
    fi
    
    return 0
}

# Fix UFW blocking issues
network_diagnostics_fixes::fix_ufw_blocking() {
    log::info "Fixing UFW firewall rules for HTTP/HTTPS..."
    
    if ! command -v ufw >/dev/null 2>&1; then
        log::info "  UFW not installed - skipping"
        return 0
    fi
    
    if ! flow::can_run_sudo "modify UFW rules"; then
        log::info "  → Skipping UFW fixes (requires sudo)"
        return 1
    fi
    
    # Check if UFW is active
    local ufw_status
    ufw_status=$(sudo ufw status 2>/dev/null || echo "inactive")
    
    if echo "$ufw_status" | grep -q "inactive"; then
        log::info "  UFW is inactive - no changes needed"
        return 0
    fi
    
    log::info "  Adding explicit outbound rules for HTTP/HTTPS..."
    
    # Add outbound rules for HTTP and HTTPS
    if sudo ufw allow out 80/tcp comment 'HTTP outbound' 2>/dev/null; then
        log::success "  ✓ Added outbound rule for HTTP (port 80)"
    fi
    
    if sudo ufw allow out 443/tcp comment 'HTTPS outbound' 2>/dev/null; then
        log::success "  ✓ Added outbound rule for HTTPS (port 443)"
    fi
    
    # Also add rules for DNS
    if sudo ufw allow out 53 comment 'DNS outbound' 2>/dev/null; then
        log::success "  ✓ Added outbound rule for DNS (port 53)"
    fi
    
    # Reload UFW
    if sudo ufw reload 2>/dev/null; then
        log::success "  ✓ Reloaded UFW with new rules"
    fi
    
    log::info "  UFW rules have been updated to allow outbound HTTP/HTTPS/DNS"
    return 0
}

# Export functions for external use
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    # Script was sourced, export functions
    # Commenting out functions with heredocs due to bash export/import issues
    # export -f network_diagnostics_fixes::fix_ipv6_issues
    # export -f network_diagnostics_fixes::fix_ipv4_only_issues
    export -f network_diagnostics_fixes::add_ipv4_host_override
    # export -f network_diagnostics_fixes::fix_dns_issues
    export -f network_diagnostics_fixes::fix_ufw_blocking
fi