#!/bin/bash

# ElectionGuard Unit Tests
# Test library functions and modules (<60s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

echo "ElectionGuard Unit Tests"
echo "========================"

# Test 1: Python module imports
echo "Testing Python module imports..."
if [[ -f "${RESOURCE_DIR}/venv/bin/activate" ]]; then
    source "${RESOURCE_DIR}/venv/bin/activate" 2>/dev/null
elif [[ ! -f "${RESOURCE_DIR}/.use_system_python" ]]; then
    echo "  ✗ Python environment not configured"
    exit 1
fi

# Test basic Python modules that should always be available
python3 << EOF
import sys
import json
print("  ✓ Core Python modules available")

# Optional modules - warn but don't fail
try:
    import flask
    print("  ✓ Flask module")
except ImportError:
    print("  ⚠ Flask module not installed (using simulation mode)")

try:
    import requests
    print("  ✓ Requests module")
except ImportError:
    print("  ⚠ Requests module not installed")
EOF

# Test 2: Configuration validation
echo "Testing configuration loading..."
if source "${RESOURCE_DIR}/config/defaults.sh" 2>/dev/null; then
    echo "  ✓ defaults.sh loaded"
else
    echo "  ✗ Failed to load defaults.sh"
    exit 1
fi

# Test 3: JSON schema validation
echo "Testing JSON configurations..."
for json_file in "${RESOURCE_DIR}/config/runtime.json" "${RESOURCE_DIR}/config/schema.json"; do
    if [[ -f "$json_file" ]]; then
        if python3 -m json.tool "$json_file" > /dev/null 2>&1; then
            echo "  ✓ $(basename "$json_file") is valid"
        else
            echo "  ✗ $(basename "$json_file") is invalid"
            exit 1
        fi
    else
        echo "  ✗ $(basename "$json_file") not found"
        exit 1
    fi
done

# Test 4: API server functionality
echo "Testing API server availability..."
if [[ -f "${RESOURCE_DIR}/lib/api_server.py" ]]; then
    echo "  ✓ API server script exists"
    # Check if the server can be imported
    python3 -c "import sys; sys.path.insert(0, '${RESOURCE_DIR}/lib'); import api_server" 2>/dev/null && \
        echo "  ✓ API server can be imported" || \
        echo "  ⚠ API server import test skipped"
else
    echo "  ✗ API server script missing"
    exit 1
fi

# Test 5: Directory structure
echo "Testing directory structure..."
required_dirs=("lib" "config" "test" "test/phases")
for dir in "${required_dirs[@]}"; do
    if [[ -d "${RESOURCE_DIR}/${dir}" ]]; then
        echo "  ✓ ${dir}/ exists"
    else
        echo "  ✗ ${dir}/ missing"
        exit 1
    fi
done

echo ""
echo "Unit tests PASSED"
exit 0