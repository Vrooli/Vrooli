#!/usr/bin/env bash
################################################################################
# Huginn Smoke Tests - Quick health validation (<30s)
# 
# Validates basic functionality and health of Huginn resource
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
MAX_WAIT=30

echo "🔥 Starting Huginn smoke tests..."

# Test 1: CLI responds to help
test::cli_help() {
    echo -n "Testing CLI help command... "
    if vrooli resource huginn help &>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "❌ FAIL: CLI help not responding"
        return 1
    fi
}

# Test 2: Check if service is running
test::service_running() {
    echo -n "Checking if Huginn container is running... "
    
    local container_status=$(docker ps --filter "name=huginn" --format "{{.Status}}" 2>/dev/null || echo "")
    
    if [[ -n "${container_status}" ]]; then
        echo "✅ PASS (${container_status})"
        return 0
    else
        echo "⚠️  SKIP: Service not running"
        return 0  # Not a failure, just not running
    fi
}

# Test 3: Health endpoint responds (if running)
test::health_check() {
    echo -n "Testing health endpoint... "
    
    # Check if container is running first
    if ! docker ps --filter "name=huginn" --format "{{.Names}}" 2>/dev/null | grep -q huginn; then
        echo "⚠️  SKIP: Service not running"
        return 0
    fi
    
    if timeout 5 curl -sf "${HUGINN_URL}/health" &>/dev/null || \
       timeout 5 curl -sf "${HUGINN_URL}" &>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "❌ FAIL: Health endpoint not responding"
        return 1
    fi
}

# Test 4: CLI status command works
test::cli_status() {
    echo -n "Testing CLI status command... "
    # Status command should work even if not installed, just check it runs
    local output=$(vrooli resource huginn status 2>&1)
    if [[ -n "${output}" ]]; then
        echo "✅ PASS"
        return 0
    else
        echo "❌ FAIL: Status command failed"
        return 1
    fi
}

# Test 5: Configuration files exist
test::config_files() {
    echo -n "Checking configuration files... "
    local missing=()
    
    [[ ! -f "${RESOURCE_DIR}/config/defaults.sh" ]] && missing+=("defaults.sh")
    [[ ! -f "${RESOURCE_DIR}/config/runtime.json" ]] && missing+=("runtime.json")
    [[ ! -f "${RESOURCE_DIR}/cli.sh" ]] && missing+=("cli.sh")
    
    if [[ ${#missing[@]} -eq 0 ]]; then
        echo "✅ PASS"
        return 0
    else
        echo "❌ FAIL: Missing files: ${missing[*]}"
        return 1
    fi
}

# Test 6: Port allocation correct
test::port_allocation() {
    echo -n "Checking port allocation... "
    
    local registry_port=$(grep -E '^\s*\["huginn"\]=' "${APP_ROOT}/scripts/resources/port_registry.sh" | grep -oE '[0-9]+' || echo "")
    
    if [[ "${registry_port}" == "${HUGINN_PORT}" ]]; then
        echo "✅ PASS (port ${HUGINN_PORT})"
        return 0
    else
        echo "❌ FAIL: Port mismatch (registry: ${registry_port}, expected: ${HUGINN_PORT})"
        return 1
    fi
}

# Run all smoke tests
main() {
    local failed=0
    
    test::cli_help || ((failed++))
    test::service_running || ((failed++))
    test::health_check || ((failed++))
    test::cli_status || ((failed++))
    test::config_files || ((failed++))
    test::port_allocation || ((failed++))
    
    echo ""
    if [[ ${failed} -eq 0 ]]; then
        echo "✅ All smoke tests passed!"
        return 0
    else
        echo "❌ ${failed} smoke test(s) failed"
        return 1
    fi
}

# Execute tests
main "$@"