#!/usr/bin/env bash
# WireGuard Unit Tests - Library function validation (<60s)

set -euo pipefail

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"

# ====================
# Unit Test Suite
# ====================
test_unit() {
    local passed=0
    local failed=0
    local start_time=$(date +%s)
    
    echo "Starting unit tests..."
    
    # Test 1: Port number validation
    echo -n "  [1/8] Port number validation... "
    if [[ "$WIREGUARD_PORT" =~ ^[0-9]+$ ]] && \
       [[ "$WIREGUARD_PORT" -ge 1024 ]] && \
       [[ "$WIREGUARD_PORT" -le 65535 ]]; then
        echo "✓ (port: $WIREGUARD_PORT)"
        ((passed++))
    else
        echo "✗ (invalid port: $WIREGUARD_PORT)"
        ((failed++))
    fi
    
    # Test 2: Network CIDR validation
    echo -n "  [2/8] Network CIDR validation... "
    if echo "$WIREGUARD_NETWORK" | grep -qE '^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$'; then
        # Additional validation for IP octets
        local ip="${WIREGUARD_NETWORK%/*}"
        local valid_ip=true
        IFS='.' read -ra OCTETS <<< "$ip"
        for octet in "${OCTETS[@]}"; do
            if [[ $octet -gt 255 ]]; then
                valid_ip=false
                break
            fi
        done
        
        if [[ "$valid_ip" == true ]]; then
            echo "✓ (network: $WIREGUARD_NETWORK)"
            ((passed++))
        else
            echo "✗ (invalid IP in CIDR: $WIREGUARD_NETWORK)"
            ((failed++))
        fi
    else
        echo "✗ (invalid CIDR format: $WIREGUARD_NETWORK)"
        ((failed++))
    fi
    
    # Test 3: Container name validation
    echo -n "  [3/8] Container name validation... "
    if [[ "$CONTAINER_NAME" =~ ^[a-zA-Z0-9][a-zA-Z0-9_.-]*$ ]]; then
        echo "✓ (name: $CONTAINER_NAME)"
        ((passed++))
    else
        echo "✗ (invalid name: $CONTAINER_NAME)"
        ((failed++))
    fi
    
    # Test 4: Image name validation
    echo -n "  [4/8] Image name validation... "
    if [[ -n "$WIREGUARD_IMAGE" ]] && [[ "$WIREGUARD_IMAGE" =~ ^[a-zA-Z0-9].*:.+$ ]]; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (invalid image: $WIREGUARD_IMAGE)"
        ((failed++))
    fi
    
    # Test 5: Config directory path validation
    echo -n "  [5/8] Config directory path... "
    if [[ -n "$CONFIG_DIR" ]] && [[ "$CONFIG_DIR" =~ ^/ ]]; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (invalid path: $CONFIG_DIR)"
        ((failed++))
    fi
    
    # Test 6: Keepalive interval validation
    echo -n "  [6/8] Keepalive interval... "
    if [[ "$WIREGUARD_KEEPALIVE" =~ ^[0-9]+$ ]] && \
       [[ "$WIREGUARD_KEEPALIVE" -ge 0 ]] && \
       [[ "$WIREGUARD_KEEPALIVE" -le 3600 ]]; then
        echo "✓ (keepalive: ${WIREGUARD_KEEPALIVE}s)"
        ((passed++))
    else
        echo "✗ (invalid keepalive: $WIREGUARD_KEEPALIVE)"
        ((failed++))
    fi
    
    # Test 7: DNS servers validation
    echo -n "  [7/8] DNS servers validation... "
    if echo "$WIREGUARD_DNS" | grep -qE '^([0-9]{1,3}\.){3}[0-9]{1,3}(,([0-9]{1,3}\.){3}[0-9]{1,3})*$'; then
        echo "✓ (DNS: $WIREGUARD_DNS)"
        ((passed++))
    else
        echo "✗ (invalid DNS: $WIREGUARD_DNS)"
        ((failed++))
    fi
    
    # Test 8: Runtime configuration file
    echo -n "  [8/8] Runtime configuration... "
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        if jq empty "${SCRIPT_DIR}/config/runtime.json" 2>/dev/null; then
            echo "✓"
            ((passed++))
        else
            echo "✗ (invalid JSON)"
            ((failed++))
        fi
    else
        echo "✗ (file not found)"
        ((failed++))
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Unit Test Summary: $passed passed, $failed failed (${duration}s)"
    
    # Unit tests should complete in under 60 seconds
    if [[ $duration -gt 60 ]]; then
        echo "⚠ Warning: Unit tests took longer than 60 seconds"
    fi
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}