#!/usr/bin/env bash
################################################################################
# OWASP ZAP Resource - Unit Tests
# Tests individual functions and components
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ZAP_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "Running OWASP ZAP unit tests..."

# Test 1: Configuration files exist
echo -n "Test 1 - Configuration files: "
if [[ -f "${ZAP_DIR}/config/defaults.sh" ]] && \
   [[ -f "${ZAP_DIR}/config/runtime.json" ]] && \
   [[ -f "${ZAP_DIR}/config/schema.json" ]]; then
    echo "PASS"
else
    echo "FAIL - Missing configuration files"
    exit 1
fi

# Test 2: Library files exist
echo -n "Test 2 - Library files: "
if [[ -f "${ZAP_DIR}/lib/core.sh" ]] && \
   [[ -f "${ZAP_DIR}/lib/test.sh" ]]; then
    echo "PASS"
else
    echo "FAIL - Missing library files"
    exit 1
fi

# Test 3: Default port configuration
echo -n "Test 3 - Port configuration: "
source "${ZAP_DIR}/config/defaults.sh"
if [[ "${ZAP_API_PORT}" == "8180" ]] && [[ "${ZAP_PROXY_PORT}" == "8181" ]]; then
    echo "PASS"
else
    echo "FAIL - Incorrect port configuration"
    exit 1
fi

# Test 4: Runtime.json valid JSON
echo -n "Test 4 - Runtime.json validity: "
if jq empty "${ZAP_DIR}/config/runtime.json" 2>/dev/null; then
    echo "PASS"
else
    echo "FAIL - Invalid JSON in runtime.json"
    exit 1
fi

# Test 5: Schema.json valid JSON
echo -n "Test 5 - Schema.json validity: "
if jq empty "${ZAP_DIR}/config/schema.json" 2>/dev/null; then
    echo "PASS"
else
    echo "FAIL - Invalid JSON in schema.json"
    exit 1
fi

# Test 6: CLI script has correct shebang
echo -n "Test 6 - CLI shebang: "
if head -n1 "${ZAP_DIR}/cli.sh" | grep -q "^#!/usr/bin/env bash"; then
    echo "PASS"
else
    echo "FAIL - Incorrect shebang in cli.sh"
    exit 1
fi

# Test 7: Test runner is executable
echo -n "Test 7 - Test runner executable: "
if [[ -x "${ZAP_DIR}/test/run-tests.sh" ]]; then
    echo "PASS"
else
    echo "FAIL - Test runner not executable"
    exit 1
fi

# Test 8: Documentation files exist
echo -n "Test 8 - Documentation files: "
if [[ -f "${ZAP_DIR}/README.md" ]] && [[ -f "${ZAP_DIR}/PRD.md" ]]; then
    echo "PASS"
else
    echo "FAIL - Missing documentation files"
    exit 1
fi

echo "All unit tests passed!"
exit 0