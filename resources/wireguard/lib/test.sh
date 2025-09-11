#!/usr/bin/env bash
# WireGuard Test Functions Library

set -euo pipefail

# Source core functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/core.sh"

# ====================
# Test Command Handler
# ====================
handle_test_command() {
    local subcommand="${1:-all}"
    shift || true
    
    case "$subcommand" in
        smoke)
            run_smoke_tests "$@"
            ;;
        integration)
            run_integration_tests "$@"
            ;;
        unit)
            run_unit_tests "$@"
            ;;
        all)
            run_all_tests "$@"
            ;;
        *)
            echo "Error: Unknown test subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# ====================
# Smoke Tests (<30s)
# ====================
run_smoke_tests() {
    echo "Running WireGuard smoke tests..."
    echo "================================"
    
    local passed=0
    local failed=0
    
    # Test 1: Container is running
    echo -n "Test: Container running... "
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 2: WireGuard command available
    echo -n "Test: WireGuard command available... "
    if timeout 5 docker exec "$CONTAINER_NAME" wg --version &>/dev/null; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 3: Health check passes
    echo -n "Test: Health check... "
    if check_health; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 4: Config directory exists
    echo -n "Test: Config directory exists... "
    if [[ -d "$CONFIG_DIR" ]]; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 5: Port is exposed
    echo -n "Test: UDP port exposed... "
    if docker port "$CONTAINER_NAME" 51820/udp &>/dev/null; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    echo ""
    echo "Smoke Test Results: $passed passed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

# ====================
# Integration Tests (<120s)
# ====================
run_integration_tests() {
    echo "Running WireGuard integration tests..."
    echo "======================================"
    
    local passed=0
    local failed=0
    
    # Test 1: Can generate keys
    echo -n "Test: Key generation... "
    if docker exec "$CONTAINER_NAME" sh -c "wg genkey | wg pubkey" &>/dev/null; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 2: Can create tunnel config
    echo -n "Test: Create tunnel config... "
    local test_tunnel="test-tunnel-$$"
    if add_tunnel_config "$test_tunnel" &>/dev/null; then
        echo "✓ PASSED"
        ((passed++))
        # Cleanup
        remove_tunnel_config "$test_tunnel" &>/dev/null || true
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 3: Can list configs
    echo -n "Test: List tunnel configs... "
    if list_tunnel_configs &>/dev/null; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 4: Container restart works
    echo -n "Test: Container restart... "
    if docker restart "$CONTAINER_NAME" &>/dev/null && sleep 5 && check_health; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 5: Kernel modules or userspace works
    echo -n "Test: WireGuard implementation... "
    if docker exec "$CONTAINER_NAME" sh -c "lsmod | grep wireguard || which wireguard-go" &>/dev/null; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    echo ""
    echo "Integration Test Results: $passed passed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

# ====================
# Unit Tests (<60s)
# ====================
run_unit_tests() {
    echo "Running WireGuard unit tests..."
    echo "================================"
    
    local passed=0
    local failed=0
    
    # Test 1: Port validation
    echo -n "Test: Port number validation... "
    if [[ "$WIREGUARD_PORT" -gt 0 ]] && [[ "$WIREGUARD_PORT" -lt 65536 ]]; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 2: Network CIDR validation
    echo -n "Test: Network CIDR validation... "
    if echo "$WIREGUARD_NETWORK" | grep -qE '^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}/[0-9]{1,2}$'; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 3: Container name validation
    echo -n "Test: Container name validation... "
    if [[ -n "$CONTAINER_NAME" ]]; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 4: Image name validation
    echo -n "Test: Image name validation... "
    if [[ -n "$WIREGUARD_IMAGE" ]]; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    echo ""
    echo "Unit Test Results: $passed passed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

# ====================
# Run All Tests
# ====================
run_all_tests() {
    echo "Running all WireGuard tests..."
    echo "=============================="
    echo ""
    
    local overall_result=0
    
    # Run smoke tests
    if ! run_smoke_tests; then
        overall_result=1
    fi
    echo ""
    
    # Run integration tests
    if ! run_integration_tests; then
        overall_result=1
    fi
    echo ""
    
    # Run unit tests
    if ! run_unit_tests; then
        overall_result=1
    fi
    
    echo ""
    echo "=============================="
    if [[ $overall_result -eq 0 ]]; then
        echo "All tests PASSED"
    else
        echo "Some tests FAILED"
    fi
    
    return $overall_result
}