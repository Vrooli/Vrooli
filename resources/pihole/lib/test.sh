#!/usr/bin/env bash
# Pi-hole Test Library - Testing functions for Pi-hole resource
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh"

# Run smoke tests (quick health check)
test_smoke() {
    echo "Running Pi-hole smoke tests..."
    local failed=0
    
    # Test 1: Container exists
    echo -n "Test: Container exists... "
    if docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 2: Container is running
    echo -n "Test: Container running... "
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 3: DNS port accessible
    echo -n "Test: DNS port accessible... "
    # Load port configuration if exists
    local dns_port="${PIHOLE_DNS_PORT}"
    if [[ -f "${PIHOLE_DATA_DIR}/.port_config" ]]; then
        source "${PIHOLE_DATA_DIR}/.port_config"
        dns_port="${DNS_PORT:-${PIHOLE_DNS_PORT}}"
    fi
    if timeout 5 nc -z localhost "${dns_port}" 2>/dev/null; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 4: API port accessible
    echo -n "Test: API port accessible... "
    if timeout 5 nc -z localhost "${PIHOLE_API_PORT}" 2>/dev/null; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 5: Health check passes
    echo -n "Test: Health check... "
    if check_health; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 6: Web interface responds
    echo -n "Test: Web interface responds... "
    if timeout 5 curl -If "http://localhost:${PIHOLE_API_PORT}/admin/" 2>/dev/null | grep -q "HTTP/1.1"; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Summary
    echo ""
    if [[ $failed -eq 0 ]]; then
        echo "All smoke tests passed!"
        return 0
    else
        echo "Failed $failed smoke tests"
        return 1
    fi
}

# Run integration tests
test_integration() {
    echo "Running Pi-hole integration tests..."
    local failed=0
    
    # Get actual DNS port from configuration
    local dns_port="${PIHOLE_DNS_PORT}"
    if [[ -f "${PIHOLE_DATA_DIR}/.port_config" ]]; then
        source "${PIHOLE_DATA_DIR}/.port_config"
        dns_port="${DNS_PORT:-${PIHOLE_DNS_PORT}}"
    fi
    
    # Test 1: DNS resolution works
    echo -n "Test: DNS resolution... "
    if timeout 5 dig @localhost -p ${dns_port} google.com +short >/dev/null 2>&1; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 2: Known ad domain is blocked
    echo -n "Test: Ad domain blocked... "
    local result
    result=$(timeout 5 dig @localhost -p ${dns_port} doubleclick.net +short 2>/dev/null || echo "failed")
    if [[ "$result" == "0.0.0.0" ]] || [[ "$result" == "" ]]; then
        echo "PASS"
    else
        echo "FAIL (result: $result)"
        ((failed++))
    fi
    
    # Test 3: Statistics API works
    echo -n "Test: Statistics API... "
    if timeout 5 curl -sf "http://localhost:${PIHOLE_API_PORT}/admin/api.php?summary" | jq -e '.domains_being_blocked' >/dev/null 2>&1; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 4: Blacklist management
    echo -n "Test: Blacklist add/remove... "
    local test_domain="test-blocked-domain.com"
    if docker exec "${CONTAINER_NAME}" pihole -b "$test_domain" >/dev/null 2>&1 && \
       docker exec "${CONTAINER_NAME}" pihole -b -d "$test_domain" >/dev/null 2>&1; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 5: Whitelist management
    echo -n "Test: Whitelist add/remove... "
    local test_domain="test-allowed-domain.com"
    if docker exec "${CONTAINER_NAME}" pihole -w "$test_domain" >/dev/null 2>&1 && \
       docker exec "${CONTAINER_NAME}" pihole -w -d "$test_domain" >/dev/null 2>&1; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 6: Query logging
    echo -n "Test: Query logging... "
    if docker exec "${CONTAINER_NAME}" test -f /var/log/pihole/pihole.log; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 7: Gravity update
    echo -n "Test: Gravity update... "
    if docker exec "${CONTAINER_NAME}" pihole -g --skip-download >/dev/null 2>&1; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Summary
    echo ""
    if [[ $failed -eq 0 ]]; then
        echo "All integration tests passed!"
        return 0
    else
        echo "Failed $failed integration tests"
        return 1
    fi
}

# Run unit tests
test_unit() {
    echo "Running Pi-hole unit tests..."
    local failed=0
    
    # Test 1: Configuration files exist
    echo -n "Test: Configuration directory... "
    if [[ -d "${PIHOLE_DATA_DIR}/etc-pihole" ]]; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 2: Password file exists
    echo -n "Test: Password file... "
    if [[ -f "${PIHOLE_DATA_DIR}/.webpassword" ]]; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 3: DNS port configuration
    echo -n "Test: DNS port config... "
    if [[ "${PIHOLE_DNS_PORT}" == "53" ]]; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 4: API port configuration
    echo -n "Test: API port config... "
    if [[ "${PIHOLE_API_PORT}" == "8087" ]]; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Test 5: Container name
    echo -n "Test: Container name... "
    if [[ "${CONTAINER_NAME}" == "vrooli-pihole" ]]; then
        echo "PASS"
    else
        echo "FAIL"
        ((failed++))
    fi
    
    # Summary
    echo ""
    if [[ $failed -eq 0 ]]; then
        echo "All unit tests passed!"
        return 0
    else
        echo "Failed $failed unit tests"
        return 1
    fi
}

# Run all tests
test_all() {
    echo "Running all Pi-hole tests..."
    echo "=============================="
    
    local total_failed=0
    
    # Run smoke tests
    echo ""
    if ! test_smoke; then
        ((total_failed++))
    fi
    
    # Run integration tests
    echo ""
    if ! test_integration; then
        ((total_failed++))
    fi
    
    # Run unit tests
    echo ""
    if ! test_unit; then
        ((total_failed++))
    fi
    
    # Final summary
    echo ""
    echo "=============================="
    if [[ $total_failed -eq 0 ]]; then
        echo "All test suites passed!"
        return 0
    else
        echo "Failed $total_failed test suites"
        return 1
    fi
}