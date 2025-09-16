#!/bin/bash

# OpenEMS Smoke Tests - Minimal Version
# Quick validation without sourcing port registry

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CLI="${RESOURCE_DIR}/cli.sh"

# Set ports directly to avoid sourcing issues
export OPENEMS_PORT=8084
export OPENEMS_JSONRPC_PORT=8085
export OPENEMS_MODBUS_PORT=502
export OPENEMS_BACKEND_PORT=8086

# Test counters
TESTS_RUN=0
TESTS_PASSED=0

echo "🔥 OpenEMS Smoke Tests (Minimal)"
echo "================================"

# Test 1: CLI exists
echo -n "  CLI exists... "
if [[ -x "$CLI" ]]; then
    echo "✅"
    ((TESTS_PASSED++))
else
    echo "❌"
fi
((TESTS_RUN++))

# Test 2: Directories exist
echo -n "  Directories exist... "
if [[ -d "${RESOURCE_DIR}/lib" && -d "${RESOURCE_DIR}/config" ]]; then
    echo "✅"
    ((TESTS_PASSED++))
else
    echo "❌"
fi
((TESTS_RUN++))

# Test 3: Config files exist
echo -n "  Config files exist... "
if [[ -f "${RESOURCE_DIR}/config/defaults.sh" && -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
    echo "✅"
    ((TESTS_PASSED++))
else
    echo "❌"
fi
((TESTS_RUN++))

# Test 4: Docker available
echo -n "  Docker available... "
if docker --version >/dev/null 2>&1; then
    echo "✅"
    ((TESTS_PASSED++))
else
    echo "❌"
fi
((TESTS_RUN++))

# Test 5: Port available
echo -n "  Port 8084 available... "
if ! netstat -tuln 2>/dev/null | grep -q ":8084 "; then
    echo "✅"
    ((TESTS_PASSED++))
else
    echo "❌"
fi
((TESTS_RUN++))

# Summary
echo ""
echo "📊 Results: $TESTS_PASSED/$TESTS_RUN passed"

if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
    echo "✅ All tests passed!"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi