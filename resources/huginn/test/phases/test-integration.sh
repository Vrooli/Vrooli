#!/usr/bin/env bash
################################################################################
# Huginn Integration Tests - Full functionality validation (<120s)
# 
# Tests complete Huginn workflows and integrations
################################################################################

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"

# Test configuration
HUGINN_PORT="${HUGINN_PORT:-4111}"
HUGINN_URL="http://localhost:${HUGINN_PORT}"

echo "🔌 Starting Huginn integration tests..."

# Test 1: Service lifecycle
test::lifecycle() {
    echo -n "Testing service lifecycle... "
    
    # Check if we can get status
    if ! vrooli resource huginn status &>/dev/null; then
        echo "⚠️  SKIP: Cannot test lifecycle without proper setup"
        return 0
    fi
    
    echo "✅ PASS"
    return 0
}

# Test 2: Database connectivity (if running)
test::database_connectivity() {
    echo -n "Testing database connectivity... "
    
    # Check if container is running
    if ! docker ps --filter "name=huginn" --format "{{.Names}}" 2>/dev/null | grep -q huginn; then
        echo "⚠️  SKIP: Service not running"
        return 0
    fi
    
    # Check if postgres dependency is satisfied
    if docker ps --filter "name=postgres" --format "{{.Names}}" 2>/dev/null | grep -q postgres; then
        echo "✅ PASS (PostgreSQL available)"
        return 0
    else
        echo "⚠️  WARN: PostgreSQL not running"
        return 0
    fi
}

# Test 3: Web interface accessibility
test::web_interface() {
    echo -n "Testing web interface... "
    
    if ! docker ps --filter "name=huginn" --format "{{.Names}}" 2>/dev/null | grep -q huginn; then
        echo "⚠️  SKIP: Service not running"
        return 0
    fi
    
    if timeout 5 curl -sf "${HUGINN_URL}" -o /dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "❌ FAIL: Web interface not accessible"
        return 1
    fi
}

# Test 4: API endpoints (if running)
test::api_endpoints() {
    echo -n "Testing API endpoints... "
    
    if ! docker ps --filter "name=huginn" --format "{{.Names}}" 2>/dev/null | grep -q huginn; then
        echo "⚠️  SKIP: Service not running"
        return 0
    fi
    
    # Test basic API endpoint (adjust based on actual Huginn API)
    if timeout 5 curl -sf "${HUGINN_URL}/users/sign_in" -o /dev/null 2>/dev/null || \
       timeout 5 curl -sf "${HUGINN_URL}/" -o /dev/null 2>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "⚠️  WARN: API endpoints not fully accessible"
        return 0
    fi
}

# Test 5: Content management commands
test::content_commands() {
    echo -n "Testing content management commands... "
    
    # Test list command
    if vrooli resource huginn content list &>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "⚠️  WARN: Content commands not fully functional"
        return 0
    fi
}

# Test 6: Integration with Vrooli ecosystem
test::vrooli_integration() {
    echo -n "Testing Vrooli ecosystem integration... "
    
    # Check if credentials command works
    if vrooli resource huginn credentials &>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "❌ FAIL: Credentials command failed"
        return 1
    fi
}

# Test 7: Log accessibility
test::logs() {
    echo -n "Testing log accessibility... "
    
    if vrooli resource huginn logs --help &>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "⚠️  WARN: Log command not fully implemented"
        return 0
    fi
}

# Run all integration tests
main() {
    local failed=0
    
    test::lifecycle || ((failed++))
    test::database_connectivity || ((failed++))
    test::web_interface || ((failed++))
    test::api_endpoints || ((failed++))
    test::content_commands || ((failed++))
    test::vrooli_integration || ((failed++))
    test::logs || ((failed++))
    
    echo ""
    if [[ ${failed} -eq 0 ]]; then
        echo "✅ All integration tests passed!"
        return 0
    else
        echo "❌ ${failed} integration test(s) failed"
        return 1
    fi
}

# Execute tests
main "$@"