#!/usr/bin/env bash
# WireGuard Integration Tests - End-to-end functionality (<120s)

set -euo pipefail

# Source configuration and core functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"

# ====================
# Integration Test Suite
# ====================
test_integration() {
    local passed=0
    local failed=0
    local start_time=$(date +%s)
    
    echo "Starting integration tests..."
    
    # Ensure service is running for integration tests
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "  ⚠ Starting WireGuard service for integration tests..."
        if ! start_wireguard --wait; then
            echo "  ✗ Failed to start WireGuard service"
            return 1
        fi
    fi
    
    # Test 1: Key generation capability
    echo -n "  [1/11] Key generation... "
    if docker exec "$CONTAINER_NAME" sh -c "wg genkey | wg pubkey" &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗"
        ((failed++))
    fi
    
    # Test 2: Create tunnel configuration
    echo -n "  [2/11] Create tunnel config... "
    local test_tunnel="integration-test-$$"
    if add_tunnel_config "$test_tunnel" &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗"
        ((failed++))
    fi
    
    # Test 3: List tunnel configurations
    echo -n "  [3/11] List tunnel configs... "
    if list_tunnel_configs | grep -q "$test_tunnel"; then
        echo "✓"
        ((passed++))
    else
        echo "✗"
        ((failed++))
    fi
    
    # Test 4: Get tunnel configuration
    echo -n "  [4/11] Get tunnel config... "
    if get_tunnel_config "$test_tunnel" | grep -q "PrivateKey"; then
        echo "✓"
        ((passed++))
    else
        echo "✗"
        ((failed++))
    fi
    
    # Test 5: Key rotation functionality
    echo -n "  [5/11] Key rotation... "
    local rotation_tunnel="rotation-test-$$"
    if add_tunnel_config "$rotation_tunnel" &>/dev/null && \
       rotate_keys "$rotation_tunnel" &>/dev/null; then
        echo "✓"
        ((passed++))
        remove_tunnel_config "$rotation_tunnel" &>/dev/null
    else
        echo "✗"
        ((failed++))
    fi
    
    # Test 6: Key rotation scheduling
    echo -n "  [6/11] Rotation scheduling... "
    local schedule_tunnel="schedule-test-$$"
    if add_tunnel_config "$schedule_tunnel" &>/dev/null && \
       schedule_rotation "$schedule_tunnel" --interval 7d &>/dev/null; then
        echo "✓"
        ((passed++))
        remove_tunnel_config "$schedule_tunnel" &>/dev/null
    else
        echo "✗"
        ((failed++))
    fi
    
    # Test 7: Rotation status
    echo -n "  [7/11] Rotation status... "
    if rotation_status &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗"
        ((failed++))
    fi
    
    # Test 8: Remove tunnel configuration
    echo -n "  [8/11] Remove tunnel config... "
    if remove_tunnel_config "$test_tunnel" &>/dev/null && \
       ! get_tunnel_config "$test_tunnel" &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗"
        ((failed++))
    fi
    
    # Test 9: Container restart
    echo -n "  [9/11] Container restart... "
    if docker restart "$CONTAINER_NAME" &>/dev/null; then
        sleep 5
        if check_health; then
            echo "✓"
            ((passed++))
        else
            echo "✗ (health check failed after restart)"
            ((failed++))
        fi
    else
        echo "✗ (restart failed)"
        ((failed++))
    fi
    
    # Test 10: Logging functionality
    echo -n "  [10/11] Logging... "
    if docker logs --tail 10 "$CONTAINER_NAME" &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗"
        ((failed++))
    fi
    
    # Test 11: Network capabilities
    echo -n "  [11/11] Network capabilities... "
    if docker exec "$CONTAINER_NAME" sh -c "ip link show | grep -E 'wg|tun'" &>/dev/null || \
       docker exec "$CONTAINER_NAME" sh -c "which wg-quick" &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗"
        ((failed++))
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Integration Test Summary: $passed passed, $failed failed (${duration}s)"
    
    # Integration tests should complete in under 120 seconds
    if [[ $duration -gt 120 ]]; then
        echo "⚠ Warning: Integration tests took longer than 120 seconds"
    fi
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}