#!/bin/bash

# OpenEMS Smoke Tests - Fixed Version
# Quick validation of basic functionality (<30s)

set -eo pipefail  # Remove 'u' to avoid unbound variable issues

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CLI="${RESOURCE_DIR}/cli.sh"

# Set default ports for testing
export OPENEMS_PORT="${OPENEMS_PORT:-8084}"
export OPENEMS_JSONRPC_PORT="${OPENEMS_JSONRPC_PORT:-8085}"
export OPENEMS_MODBUS_PORT="${OPENEMS_MODBUS_PORT:-502}"
export OPENEMS_BACKEND_PORT="${OPENEMS_BACKEND_PORT:-8086}"

# Source configuration if available (but don't fail if it doesn't work)
source "${RESOURCE_DIR}/config/defaults.sh" 2>/dev/null || true

# Test counters
TESTS_RUN=0
TESTS_PASSED=0

echo "🔥 OpenEMS Smoke Tests"
echo "====================="

# Test 1: CLI exists and is executable
echo -n "  Testing CLI accessibility... "
if [[ -x "$CLI" ]]; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 2: Help command works
echo -n "  Testing help command... "
if $CLI help >/dev/null 2>&1; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 3: Info command works
echo -n "  Testing info command... "
if $CLI info >/dev/null 2>&1; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 4: Check Docker availability
echo -n "  Testing Docker availability... "
if docker --version >/dev/null 2>&1; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 5: Port availability check
echo -n "  Testing primary port available... "
if ! netstat -tuln 2>/dev/null | grep -q ":${OPENEMS_PORT} "; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 6: Configuration files exist
echo -n "  Testing config files... "
if [[ -f "${RESOURCE_DIR}/config/defaults.sh" && -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 7: Test directory structure
echo -n "  Testing directory structure... "
if [[ -d "${RESOURCE_DIR}/lib" && -d "${RESOURCE_DIR}/config" && -d "${RESOURCE_DIR}/test" ]]; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 8: Content list command
echo -n "  Testing content list... "
if $CLI content list >/dev/null 2>&1; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 9: Status command (when not running)
echo -n "  Testing status command... "
if $CLI status >/dev/null 2>&1; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 10: Credentials command
echo -n "  Testing credentials display... "
if $CLI credentials >/dev/null 2>&1; then
    echo "✅"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "❌"
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Summary
echo ""
echo "📊 Smoke Test Results"
echo "===================="
echo "Tests Run: $TESTS_RUN"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $((TESTS_RUN - TESTS_PASSED))"

if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
    echo "✅ All smoke tests passed!"
    exit 0
else
    echo "❌ Some smoke tests failed"
    exit 1
fi