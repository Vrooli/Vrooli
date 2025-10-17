#!/usr/bin/env bash
# Splink Unit Tests - Library and configuration validation (<60s)

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Test results
PASSED=0
FAILED=0

# Helper function for test assertions
assert() {
    local test_name="$1"
    local condition="$2"
    
    echo -n "  $test_name... "
    if eval "$condition"; then
        echo "✓"
        ((PASSED++))
    else
        echo "✗"
        ((FAILED++))
    fi
}

echo "Running Splink Unit Tests"
echo "========================"

# Test 1: File Structure
echo "1. File Structure Tests"

assert "CLI script exists" \
    "[[ -f ${RESOURCE_DIR}/cli.sh ]]"

assert "Core library exists" \
    "[[ -f ${RESOURCE_DIR}/lib/core.sh ]]"

assert "Test library exists" \
    "[[ -f ${RESOURCE_DIR}/lib/test.sh ]]"

assert "Configuration files exist" \
    "[[ -f ${RESOURCE_DIR}/config/defaults.sh && \
        -f ${RESOURCE_DIR}/config/runtime.json && \
        -f ${RESOURCE_DIR}/config/schema.json ]]"

assert "Test structure complete" \
    "[[ -f ${RESOURCE_DIR}/test/run-tests.sh && \
        -d ${RESOURCE_DIR}/test/phases ]]"

assert "Documentation exists" \
    "[[ -f ${RESOURCE_DIR}/README.md && \
        -f ${RESOURCE_DIR}/PRD.md ]]"

# Test 2: Configuration Validation
echo ""
echo "2. Configuration Tests"

assert "Port configuration correct" \
    "[[ '${SPLINK_PORT}' == '8096' ]]"

assert "Backend configuration set" \
    "[[ -n '${SPLINK_BACKEND}' ]]"

assert "Data directory configured" \
    "[[ -n '${DATA_DIR}' ]]"

# Test 3: JSON Schema Validation
echo ""
echo "3. Schema Validation Tests"

assert "Runtime JSON is valid" \
    "python3 -m json.tool ${RESOURCE_DIR}/config/runtime.json &>/dev/null || \
     jq '.' ${RESOURCE_DIR}/config/runtime.json &>/dev/null"

assert "Schema JSON is valid" \
    "python3 -m json.tool ${RESOURCE_DIR}/config/schema.json &>/dev/null || \
     jq '.' ${RESOURCE_DIR}/config/schema.json &>/dev/null"

# Test 4: Script Syntax
echo ""
echo "4. Script Syntax Tests"

assert "CLI script has valid syntax" \
    "bash -n ${RESOURCE_DIR}/cli.sh"

assert "Core library has valid syntax" \
    "bash -n ${RESOURCE_DIR}/lib/core.sh"

assert "Test library has valid syntax" \
    "bash -n ${RESOURCE_DIR}/lib/test.sh"

assert "Test runner has valid syntax" \
    "bash -n ${RESOURCE_DIR}/test/run-tests.sh"

# Test 5: Executable Permissions
echo ""
echo "5. Permission Tests"

assert "CLI script is executable" \
    "[[ -x ${RESOURCE_DIR}/cli.sh ]]"

assert "Test runner is executable" \
    "[[ -x ${RESOURCE_DIR}/test/run-tests.sh ]]"

# Test 6: Function Availability
echo ""
echo "6. Function Availability Tests"

assert "Core functions can be sourced" \
    "source ${RESOURCE_DIR}/lib/core.sh && type -t install_dependencies &>/dev/null"

assert "Test functions can be sourced" \
    "source ${RESOURCE_DIR}/lib/test.sh && type -t run_tests &>/dev/null"

# Test 7: CLI Command Structure
echo ""
echo "7. CLI Command Tests"

assert "Help command exists" \
    "${RESOURCE_DIR}/cli.sh help &>/dev/null"

assert "Info command exists" \
    "${RESOURCE_DIR}/cli.sh info &>/dev/null"

# Test 8: Port Registry Integration
echo ""
echo "8. Port Registry Tests"

assert "Port registered in registry" \
    "grep -q 'splink.*8096' ${VROOLI_ROOT:-$HOME/Vrooli}/scripts/resources/port_registry.sh"

# Summary
echo ""
echo "========================"
echo "Unit Test Summary:"
echo "  Passed: $PASSED"
echo "  Failed: $FAILED"
echo "========================"

if [[ $FAILED -gt 0 ]]; then
    exit 1
fi

exit 0