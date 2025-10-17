#!/bin/bash

# ElectionGuard Smoke Tests
# Quick validation that service is healthy (<30s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Run smoke tests
echo "ElectionGuard Smoke Tests"
echo "========================="

# Test 1: Installation check
echo -n "Checking installation... "
if [[ -f "${RESOURCE_DIR}/.installed" ]]; then
    echo "✓"
else
    echo "✗"
    echo "  ERROR: ElectionGuard not installed"
    exit 1
fi

# Test 2: Python environment
echo -n "Checking Python environment... "
if [[ -d "${RESOURCE_DIR}/venv" ]]; then
    echo "✓"
else
    echo "✗"
    echo "  ERROR: Virtual environment not found"
    exit 1
fi

# Test 3: Required modules
echo -n "Checking required Python modules... "
source "${RESOURCE_DIR}/venv/bin/activate" 2>/dev/null
if python3 -c "import electionguard" 2>/dev/null; then
    echo "✓"
else
    echo "✗"
    echo "  ERROR: ElectionGuard module not installed"
    exit 1
fi

# Test 4: Service health
echo -n "Checking service health... "
if timeout 5 curl -sf "http://localhost:${ELECTIONGUARD_PORT}/health" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "  WARNING: Service not responding (may not be started)"
fi

# Test 5: Configuration files
echo -n "Checking configuration files... "
if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]] && \
   [[ -f "${RESOURCE_DIR}/config/defaults.sh" ]] && \
   [[ -f "${RESOURCE_DIR}/config/schema.json" ]]; then
    echo "✓"
else
    echo "✗"
    echo "  ERROR: Configuration files missing"
    exit 1
fi

echo ""
echo "Smoke tests PASSED"
exit 0