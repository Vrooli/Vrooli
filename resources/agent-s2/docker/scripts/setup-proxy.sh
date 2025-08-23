#!/bin/bash
# Setup transparent proxy for Agent S2 security

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Global flag to track if NAT rules were created
NAT_RULES_CREATED=false

# Automatic cleanup trap - ensures NAT rules are cleaned up on exit
cleanup_on_exit() {
    local exit_code=$?
    if [[ "$NAT_RULES_CREATED" == "true" ]]; then
        log_warn "Script exiting - cleaning up NAT rules..."
        cleanup_proxy 2>/dev/null || true
    fi
    exit $exit_code
}

# Set trap for cleanup on exit, interrupt, or termination
trap cleanup_on_exit EXIT INT TERM

# Check if running as root (required for iptables)
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_warn "Not running as root - some features may not work"
        return 1
    fi
    return 0
}

# Enable IP forwarding
setup_ip_forwarding() {
    log_info "Enabling IP forwarding..."
    sysctl -w net.ipv4.ip_forward=1 > /dev/null
    sysctl -w net.ipv6.conf.all.forwarding=1 > /dev/null
    
    # Make persistent
    echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
    echo "net.ipv6.conf.all.forwarding=1" >> /etc/sysctl.conf
}

# Setup iptables rules for transparent proxy
setup_iptables() {
    local proxy_port="${PROXY_PORT:-8085}"
    local proxy_user="agents2"
    
    log_info "Setting up iptables rules for transparent proxy on port $proxy_port..."
    
    # CRITICAL: Verify proxy service is running before creating redirects
    if ! verify_proxy_service; then
        log_error "Aborting iptables setup - proxy service verification failed"
        return 1
    fi
    
    # Create nat table rules for HTTP/HTTPS redirection
    # HTTP (port 80)
    iptables -t nat -A OUTPUT -p tcp --dport 80 -m owner ! --uid-owner $proxy_user -j REDIRECT --to-port $proxy_port
    
    # HTTPS (port 443)  
    iptables -t nat -A OUTPUT -p tcp --dport 443 -m owner ! --uid-owner $proxy_user -j REDIRECT --to-port $proxy_port
    
    # Also handle common alternative ports
    iptables -t nat -A OUTPUT -p tcp --dport 8080 -m owner ! --uid-owner $proxy_user -j REDIRECT --to-port $proxy_port
    iptables -t nat -A OUTPUT -p tcp --dport 8443 -m owner ! --uid-owner $proxy_user -j REDIRECT --to-port $proxy_port
    
    # Exclude local traffic
    iptables -t nat -I OUTPUT -o lo -j RETURN
    
    # Mark that NAT rules were successfully created (for cleanup trap)
    NAT_RULES_CREATED=true
    
    log_info "iptables rules configured successfully"
}

# Install mitmproxy CA certificate
install_ca_certificate() {
    log_info "Setting up mitmproxy CA certificate..."
    
    # Run mitmdump once to generate certificates
    su - agents2 -c "mitmdump -n" > /dev/null 2>&1 || true
    
    local ca_cert="/home/agents2/.mitmproxy/mitmproxy-ca-cert.pem"
    
    if [[ ! -f "$ca_cert" ]]; then
        log_error "mitmproxy CA certificate not found at $ca_cert"
        return 1
    fi
    
    # Install in system certificate store
    cp "$ca_cert" /usr/local/share/ca-certificates/mitmproxy-ca.crt
    update-ca-certificates
    
    # Install in Firefox (if profiles exist)
    local firefox_dir="/home/agents2/.mozilla/firefox"
    if [[ -d "$firefox_dir" ]]; then
        log_info "Installing certificate in Firefox profiles..."
        
        # Find all Firefox profiles
        for profile in "$firefox_dir"/*.default* ; do
            if [[ -d "$profile" ]]; then
                log_info "Installing in profile: $(basename "$profile")"
                
                # Import certificate using certutil
                certutil -A -n "mitmproxy" -t "CT,c,c" -i "$ca_cert" -d "sql:$profile" 2>/dev/null || {
                    log_warn "Failed to install in profile: $profile"
                }
            fi
        done
    else
        log_warn "No Firefox profiles found"
    fi
    
    log_info "CA certificate installation complete"
}

# Create Firefox policies to trust the CA
setup_firefox_policies() {
    log_info "Setting up Firefox policies..."
    
    local policies_dir="/usr/lib/firefox/distribution"
    mkdir -p "$policies_dir"
    
    cat > "$policies_dir/policies.json" << 'EOF'
{
  "policies": {
    "Certificates": {
      "ImportEnterpriseRoots": true,
      "Install": ["/home/agents2/.mitmproxy/mitmproxy-ca-cert.pem"]
    },
    "DisableFirefoxStudies": true,
    "DisableTelemetry": true,
    "DisablePocket": true,
    "OverrideFirstRunPage": "",
    "OverridePostUpdatePage": "",
    "DontCheckDefaultBrowser": true,
    "SecurityDevices": {
      "mitmproxy": "/home/agents2/.mitmproxy/mitmproxy-ca-cert.pem"
    }
  }
}
EOF
    
    chown -R agents2:agents2 "$policies_dir"
}

# Verify proxy setup
verify_setup() {
    log_info "Verifying proxy setup..."
    
    # Check iptables rules
    if iptables -t nat -L OUTPUT -n | grep -q "REDIRECT.*tcp.*dpt:80.*redir ports ${PROXY_PORT:-8085}"; then
        log_info "✓ HTTP redirect rule found"
    else
        log_warn "✗ HTTP redirect rule not found"
    fi
    
    if iptables -t nat -L OUTPUT -n | grep -q "REDIRECT.*tcp.*dpt:443.*redir ports ${PROXY_PORT:-8085}"; then
        log_info "✓ HTTPS redirect rule found"
    else
        log_warn "✗ HTTPS redirect rule not found"
    fi
    
    # Check CA certificate
    if [[ -f "/usr/local/share/ca-certificates/mitmproxy-ca.crt" ]]; then
        log_info "✓ CA certificate installed in system store"
    else
        log_warn "✗ CA certificate not in system store"
    fi
}

# Verify proxy service is listening before creating NAT rules
verify_proxy_service() {
    local proxy_port="${PROXY_PORT:-8085}"
    local max_wait="${PROXY_VERIFY_TIMEOUT:-30}"
    
    log_info "Verifying proxy service is listening on port $proxy_port..."
    
    # Wait for the service to be ready
    local waited=0
    while [[ $waited -lt $max_wait ]]; do
        if ss -tlnp 2>/dev/null | grep -q ":$proxy_port "; then
            log_info "✓ Proxy service confirmed listening on port $proxy_port"
            return 0
        fi
        
        if [[ $waited -eq 0 ]]; then
            log_info "Waiting for proxy service to start listening..."
        fi
        
        sleep 2
        waited=$((waited + 2))
    done
    
    log_error "✗ Proxy service not listening on port $proxy_port after ${max_wait}s"
    log_error "Cannot create NAT redirection rules - would cause traffic hijacking!"
    log_info "Check if mitmproxy/Agent-S2 is running correctly"
    return 1
}

# Cleanup function
cleanup_proxy() {
    log_info "Cleaning up proxy configuration..."
    
    # Remove iptables rules
    iptables -t nat -D OUTPUT -p tcp --dport 80 -m owner ! --uid-owner agents2 -j REDIRECT --to-port ${PROXY_PORT:-8085} 2>/dev/null || true
    iptables -t nat -D OUTPUT -p tcp --dport 443 -m owner ! --uid-owner agents2 -j REDIRECT --to-port ${PROXY_PORT:-8085} 2>/dev/null || true
    iptables -t nat -D OUTPUT -p tcp --dport 8080 -m owner ! --uid-owner agents2 -j REDIRECT --to-port ${PROXY_PORT:-8085} 2>/dev/null || true
    iptables -t nat -D OUTPUT -p tcp --dport 8443 -m owner ! --uid-owner agents2 -j REDIRECT --to-port ${PROXY_PORT:-8085} 2>/dev/null || true
    iptables -t nat -D OUTPUT -o lo -j RETURN 2>/dev/null || true
    
    log_info "Proxy cleanup complete"
}

# Main setup function
main() {
    local action="${1:-setup}"
    
    case "$action" in
        setup)
            log_info "Starting Agent S2 proxy setup..."
            
            if check_root; then
                setup_ip_forwarding
                setup_iptables
            else
                log_warn "Skipping iptables setup (requires root)"
            fi
            
            install_ca_certificate
            setup_firefox_policies
            verify_setup
            
            log_info "Proxy setup complete!"
            ;;
            
        cleanup)
            if check_root; then
                cleanup_proxy
            else
                log_error "Cleanup requires root privileges"
                exit 1
            fi
            ;;
            
        verify)
            verify_setup
            ;;
            
        *)
            echo "Usage: $0 {setup|cleanup|verify}"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"