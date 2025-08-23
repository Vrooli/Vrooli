#!/bin/bash
# Clean up all Agent-S2 iptables rules

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[CLEAN]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[CLEAN]${NC} $1"
}

# Clean up all Agent-S2 related iptables rules
cleanup_iptables() {
    log_info "Cleaning up all Agent-S2 iptables rules..."
    
    # Use the sudo command passed from environment or default to sudo
    SUDO_CMD="${SUDO_CMD:-sudo}"
    
    # Count rules removed
    local removed=0
    
    # Remove all OUTPUT rules in nat table that redirect to Agent-S2 ports
    while true; do
        # Find first rule that matches our patterns
        local rule_num=$($SUDO_CMD iptables -t nat -L OUTPUT --line-numbers -n 2>/dev/null | \
            grep -E "REDIRECT.*tcp.*dpt:(80|443|8080|8443).*redir ports (8080|8085)" | \
            head -1 | awk '{print $1}')
        
        if [[ -n "$rule_num" ]]; then
            if $SUDO_CMD iptables -t nat -D OUTPUT "$rule_num" 2>/dev/null; then
                ((removed++))
            else
                break
            fi
        else
            break
        fi
    done
    
    # Also remove loopback exclusion rules
    while $SUDO_CMD iptables -t nat -D OUTPUT -o lo -j RETURN 2>/dev/null; do
        ((removed++))
    done
    
    log_info "Removed $removed iptables rules"
    
    # Verify cleanup
    local remaining=$($SUDO_CMD iptables -t nat -L OUTPUT -n 2>/dev/null | \
        grep -cE "REDIRECT.*tcp.*dpt:(80|443|8080|8443).*redir ports (8080|8085)" || echo "0")
    
    if [[ "$remaining" -eq 0 ]]; then
        log_info "✓ All Agent-S2 iptables rules cleaned up successfully"
    else
        log_warn "⚠ $remaining Agent-S2 iptables rules still remain"
    fi
}

# Run cleanup
cleanup_iptables