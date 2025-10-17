#!/bin/bash
# Container initialization script - runs as root for privileged operations
# then starts supervisor as agents2 user

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INIT]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[INIT]${NC} $1"
}

log_error() {
    echo -e "${RED}[INIT]${NC} $1"
}

# Enhanced privilege detection and setup
detect_privileges() {
    log_info "Detecting available privileges for transparent proxy setup..."
    
    # Check if running as root
    if [[ $EUID -eq 0 ]]; then
        log_info "✓ Running as root - full system access available"
        SUDO_CMD=""
        PRIVILEGE_LEVEL="root"
        return 0
    fi
    
    log_info "Running as user $(whoami) - checking available privileges..."
    
    # Test different sudo approaches
    if sudo -n true 2>/dev/null; then
        log_info "✓ Passwordless sudo access confirmed"
        SUDO_CMD="sudo"
        PRIVILEGE_LEVEL="sudo_full"
        return 0
    fi
    
    # Check specific iptables sudo access
    if sudo -n -l iptables >/dev/null 2>&1; then
        log_info "✓ Sudo access for iptables confirmed"
        SUDO_CMD="sudo"
        PRIVILEGE_LEVEL="sudo_iptables"
        return 0
    fi
    
    # Check if we have the required capabilities directly
    if capsh --print | grep -q "net_admin" 2>/dev/null; then
        log_info "✓ NET_ADMIN capability detected - attempting direct iptables access"
        SUDO_CMD=""
        PRIVILEGE_LEVEL="capabilities"
        return 0
    fi
    
    # No suitable privileges found
    log_warn "✗ No suitable privileges for iptables manipulation found"
    log_info "Available options tried:"
    log_info "  - Root access: No"
    log_info "  - Passwordless sudo: No"
    log_info "  - Sudo for iptables: No"
    log_info "  - NET_ADMIN capability: No"
    log_warn "Transparent proxy will be disabled - falling back to AI-level security only"
    PRIVILEGE_LEVEL="none"
    return 1
}

# Main privilege detection
if detect_privileges; then
    log_info "Privilege level: $PRIVILEGE_LEVEL - proceeding with transparent proxy setup"
else
    log_warn "Insufficient privileges - skipping network setup and starting services directly"
    # Set up basic directories and start supervisor
    log_info "Setting up container directories..."
    mkdir -p /tmp/runtime-agents2 2>/dev/null || true
    chmod 700 /tmp/runtime-agents2 2>/dev/null || true
    log_info "Starting supervisor without transparent proxy..."
    exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisor.conf
fi

# Setup security proxy if enabled
if [ "${AGENT_S2_ENABLE_PROXY:-false}" = "true" ]; then
    log_info "Setting up security proxy (iptables rules) - WARNING: This will intercept ALL system HTTP/HTTPS traffic!"
    
    # Check if we have the required tools
    if ! command -v iptables >/dev/null 2>&1; then
        log_error "iptables command not found - transparent proxy will not work"
        log_warn "Falling back to AI-level security only"
    else
        log_info "iptables found - proceeding with network-level proxy setup"
    fi
    
    # Enable IP forwarding
    log_info "Enabling IP forwarding..."
    if $SUDO_CMD sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1; then
        log_info "✓ IPv4 forwarding enabled"
    else
        log_warn "✗ Failed to enable IPv4 forwarding - may affect transparent proxy"
    fi
    
    if $SUDO_CMD sysctl -w net.ipv6.conf.all.forwarding=1 >/dev/null 2>&1; then
        log_info "✓ IPv6 forwarding enabled"
    else
        log_warn "✗ Failed to enable IPv6 forwarding - may affect transparent proxy"
    fi
    
    # Setup iptables rules for transparent proxy
    PROXY_PORT="${PROXY_PORT:-8085}"
    # NOTE: We need to be careful about user exclusions to avoid blocking browser traffic
    # The rules should only exclude traffic from mitmproxy itself to prevent loops
    
    # Clean up any existing iptables rules first
    log_info "Cleaning up existing iptables rules..."
    # Remove all existing Agent-S2 related rules to prevent duplicates
    while true; do
        # Find first rule that matches our patterns
        rule_num=$($SUDO_CMD iptables -t nat -L OUTPUT --line-numbers -n 2>/dev/null | \
            grep -E "REDIRECT.*tcp.*dpt:(80|443|8080|8443).*redir ports (8080|8085)" | \
            head -1 | awk '{print $1}')
        
        if [[ -n "$rule_num" ]]; then
            $SUDO_CMD iptables -t nat -D OUTPUT "$rule_num" 2>/dev/null || break
        else
            break
        fi
    done
    # Also remove loopback exclusion rules
    while $SUDO_CMD iptables -t nat -D OUTPUT -o lo -j RETURN 2>/dev/null; do :; done
    
    log_info "Setting up iptables rules for transparent proxy on port $PROXY_PORT..."
    
    # Track success of iptables rules
    IPTABLES_SUCCESS=0
    
    # Create nat table rules for HTTP/HTTPS redirection
    # Exclude traffic from source port 8085 (mitmproxy outbound) to prevent loops
    
    # HTTP (port 80) - redirect all traffic except from mitmproxy
    if $SUDO_CMD iptables -t nat -A OUTPUT -p tcp --dport 80 ! --sport $PROXY_PORT -j REDIRECT --to-port $PROXY_PORT 2>/dev/null; then
        log_info "✓ HTTP (port 80) redirect rule added"
        IPTABLES_SUCCESS=$((IPTABLES_SUCCESS + 1))
    else
        log_warn "✗ Failed to add HTTP iptables rule"
    fi
    
    # HTTPS (port 443) - redirect all traffic except from mitmproxy  
    if $SUDO_CMD iptables -t nat -A OUTPUT -p tcp --dport 443 ! --sport $PROXY_PORT -j REDIRECT --to-port $PROXY_PORT 2>/dev/null; then
        log_info "✓ HTTPS (port 443) redirect rule added"
        IPTABLES_SUCCESS=$((IPTABLES_SUCCESS + 1))
    else
        log_warn "✗ Failed to add HTTPS iptables rule"
    fi
    
    # Also handle common alternative ports
    if $SUDO_CMD iptables -t nat -A OUTPUT -p tcp --dport 8080 ! --sport $PROXY_PORT -j REDIRECT --to-port $PROXY_PORT 2>/dev/null; then
        log_info "✓ Port 8080 redirect rule added"
        IPTABLES_SUCCESS=$((IPTABLES_SUCCESS + 1))
    else
        log_warn "✗ Failed to add port 8080 iptables rule"
    fi
    
    if $SUDO_CMD iptables -t nat -A OUTPUT -p tcp --dport 8443 ! --sport $PROXY_PORT -j REDIRECT --to-port $PROXY_PORT 2>/dev/null; then
        log_info "✓ Port 8443 redirect rule added"
        IPTABLES_SUCCESS=$((IPTABLES_SUCCESS + 1))
    else
        log_warn "✗ Failed to add port 8443 iptables rule"
    fi
    
    # Exclude local traffic (critical for avoiding loops)
    if $SUDO_CMD iptables -t nat -I OUTPUT -o lo -j RETURN 2>/dev/null; then
        log_info "✓ Loopback exclusion rule added"
        IPTABLES_SUCCESS=$((IPTABLES_SUCCESS + 1))
    else
        log_warn "✗ Failed to add loopback exclusion rule - risk of traffic loops"
    fi
    
    # Report overall iptables setup status
    if [ $IPTABLES_SUCCESS -eq 5 ]; then
        log_info "✓ All iptables rules configured successfully - transparent proxy ready"
    elif [ $IPTABLES_SUCCESS -gt 0 ]; then
        log_warn "⚠ Partial iptables setup ($IPTABLES_SUCCESS/5 rules) - transparent proxy may have limited functionality"
    else
        log_error "✗ No iptables rules configured - transparent proxy will not work"
        log_info "Falling back to AI-level security filtering only"
    fi
    
    # Verify iptables rules
    if $SUDO_CMD iptables -t nat -L OUTPUT -n | grep -q "REDIRECT.*tcp.*dpt:80.*redir ports $PROXY_PORT"; then
        log_info "✓ HTTP redirect rule verified"
    else
        log_warn "✗ HTTP redirect rule not found"
    fi
    
    if $SUDO_CMD iptables -t nat -L OUTPUT -n | grep -q "REDIRECT.*tcp.*dpt:443.*redir ports $PROXY_PORT"; then
        log_info "✓ HTTPS redirect rule verified"
    else
        log_warn "✗ HTTPS redirect rule not found"
    fi
    
else
    log_info "Transparent proxy disabled (safer default) - Agent S2 will run without system-wide traffic interception"
    log_info "To enable proxy for enhanced security monitoring, set AGENT_S2_ENABLE_PROXY=true"
fi

# Ensure proper directory permissions
log_info "Setting up container directories..."
mkdir -p /tmp/runtime-agents2 /var/log/supervisor
chmod 700 /tmp/runtime-agents2 2>/dev/null || log_warn "Failed to set permissions on /tmp/runtime-agents2"
chmod 755 /var/log/supervisor 2>/dev/null || log_warn "Failed to set permissions on /var/log/supervisor"

# Try to change ownership, but don't fail if we can't
chown -R agents2:agents2 /tmp/runtime-agents2 2>/dev/null || log_warn "Failed to change ownership of /tmp/runtime-agents2"
chown -R agents2:agents2 /var/log/supervisor 2>/dev/null || {
    log_warn "Failed to change ownership of /var/log/supervisor - setting world writable"
    chmod 777 /var/log/supervisor 2>/dev/null || log_warn "Failed to set permissions on /var/log/supervisor"
}

log_info "Root initialization complete - starting supervisor..."

# Start supervisor as agents2 user
# Note: supervisor will handle user switching via its config files
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisor.conf