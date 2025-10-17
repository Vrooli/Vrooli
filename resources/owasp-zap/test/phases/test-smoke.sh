#!/usr/bin/env bash
################################################################################
# OWASP ZAP Resource - Smoke Tests
# Quick validation that the resource can start and respond to health checks
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ZAP_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Use the resource CLI
CLI_PATH="${ZAP_DIR}/cli.sh"

echo "Running OWASP ZAP smoke tests..."

# Test 1: CLI is executable
echo -n "Test 1 - CLI executable: "
if [[ -x "${CLI_PATH}" ]]; then
    echo "PASS"
else
    echo "FAIL - CLI not executable"
    exit 1
fi

# Test 2: Help command works
echo -n "Test 2 - Help command: "
if "${CLI_PATH}" help &>/dev/null; then
    echo "PASS"
else
    echo "FAIL - Help command failed"
    exit 1
fi

# Test 3: Info command works
echo -n "Test 3 - Info command: "
if "${CLI_PATH}" info &>/dev/null; then
    echo "PASS"
else
    echo "FAIL - Info command failed"
    exit 1
fi

# Test 4: Can check status (even if not running)
echo -n "Test 4 - Status command: "
"${CLI_PATH}" status &>/dev/null
STATUS_CODE=$?
if [[ ${STATUS_CODE} -eq 0 ]] || [[ ${STATUS_CODE} -eq 1 ]]; then
    echo "PASS"
else
    echo "FAIL - Status command failed unexpectedly"
    exit 1
fi

# Test 5: Configuration is valid
echo -n "Test 5 - Configuration valid: "
if [[ -f "${ZAP_DIR}/config/defaults.sh" ]] && \
   [[ -f "${ZAP_DIR}/config/runtime.json" ]] && \
   [[ -f "${ZAP_DIR}/config/schema.json" ]]; then
    echo "PASS"
else
    echo "FAIL - Missing configuration files"
    exit 1
fi

echo "All smoke tests passed!"
exit 0