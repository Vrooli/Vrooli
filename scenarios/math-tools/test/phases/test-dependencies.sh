#!/bin/bash
# Test Phase: Dependency Check
# Validates all required dependencies are available

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

echo "Checking dependencies..."

# Check Go
if command -v go &>/dev/null; then
    GO_VERSION=$(go version)
    echo "✓ Go installed: $GO_VERSION"
else
    echo "✗ Go not found"
    exit 1
fi

# Check Go modules
if [[ -f "api/go.mod" ]]; then
    echo "✓ Go modules configured"
    cd api
    if go mod verify &>/dev/null; then
        echo "✓ Go modules verified"
    else
        echo "⚠ Go module verification warnings (non-critical)"
    fi
    cd ..
fi

# Check required tools
required_tools=("curl" "jq")
for tool in "${required_tools[@]}"; do
    if command -v "$tool" &>/dev/null; then
        echo "✓ $tool available"
    else
        echo "✗ $tool not found"
        exit 1
    fi
done

# Check required resources
echo "Checking required resources..."
required_resources=("postgres" "redis")
for resource in "${required_resources[@]}"; do
    if vrooli resource status "$resource" &>/dev/null; then
        echo "✓ Resource available: $resource"
    else
        echo "⚠ Resource not running (may be started by lifecycle): $resource"
    fi
done

echo "✓ Dependency check completed"
exit 0
