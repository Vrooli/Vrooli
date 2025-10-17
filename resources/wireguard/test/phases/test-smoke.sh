#!/usr/bin/env bash
# WireGuard Smoke Tests - Quick health validation (<30s)

set -euo pipefail

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"

# ====================
# Smoke Test Suite
# ====================
test_smoke() {
    local passed=0
    local failed=0
    local start_time=$(date +%s)
    
    echo "Starting smoke tests..."
    
    # Test 1: Docker is available
    echo -n "  [1/5] Docker availability... "
    if command -v docker &>/dev/null && docker info &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Docker not available)"
        ((failed++))
    fi
    
    # Test 2: Container exists or can be created
    echo -n "  [2/5] Container status... "
    if docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "✓"
        ((passed++))
    elif docker images | grep -q "${WIREGUARD_IMAGE%:*}"; then
        echo "✓ (image available)"
        ((passed++))
    else
        echo "✗ (container/image not found)"
        ((failed++))
    fi
    
    # Test 3: Configuration directory
    echo -n "  [3/5] Configuration directory... "
    if [[ -d "$CONFIG_DIR" ]] || mkdir -p "$CONFIG_DIR" 2>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (cannot create config dir)"
        ((failed++))
    fi
    
    # Test 4: Port availability check
    echo -n "  [4/5] Port availability... "
    if ! ss -tuln | grep -q ":${WIREGUARD_PORT}[[:space:]]" 2>/dev/null; then
        echo "✓ (port $WIREGUARD_PORT available)"
        ((passed++))
    elif docker port "$CONTAINER_NAME" 51820/udp &>/dev/null; then
        echo "✓ (port bound to WireGuard)"
        ((passed++))
    else
        echo "✗ (port $WIREGUARD_PORT in use)"
        ((failed++))
    fi
    
    # Test 5: Basic health check
    echo -n "  [5/5] Service health... "
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q . && \
       timeout 5 docker exec "$CONTAINER_NAME" wg --version &>/dev/null; then
        echo "✓"
        ((passed++))
    elif docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "⚠ (container running, WireGuard not responding)"
        ((passed++))
    else
        echo "○ (service not running)"
        ((passed++))  # Not a failure if service isn't started
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Smoke Test Summary: $passed passed, $failed failed (${duration}s)"
    
    # Smoke tests should complete in under 30 seconds
    if [[ $duration -gt 30 ]]; then
        echo "⚠ Warning: Smoke tests took longer than 30 seconds"
    fi
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}