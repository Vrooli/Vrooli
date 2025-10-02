#!/bin/bash
set -euo pipefail

echo "=== Running test-dependencies.sh ==="

# Check required dependencies for scenario-authenticator

echo "Checking Go installation..."
if command -v go >/dev/null 2>&1; then
  GO_VERSION=$(go version | awk '{print $3}')
  echo "✓ Go is installed: $GO_VERSION"
else
  echo "✗ Go is not installed"
  exit 1
fi

echo ""
echo "Checking Node.js installation (for UI)..."
if command -v node >/dev/null 2>&1; then
  NODE_VERSION=$(node --version)
  echo "✓ Node.js is installed: $NODE_VERSION"
else
  echo "✗ Node.js is not installed"
  exit 1
fi

echo ""
echo "Checking required Go modules..."
if [ -f "api/go.mod" ]; then
  echo "✓ Go modules file exists"
  cd api && go mod download && cd ..
  echo "✓ Go dependencies downloaded"
else
  echo "✗ Go modules file missing"
  exit 1
fi

echo ""
echo "Checking required resources..."
REQUIRED_RESOURCES=("postgres" "redis")
for resource in "${REQUIRED_RESOURCES[@]}"; do
  if vrooli resource status "$resource" 2>&1 | grep -qi "running\|healthy"; then
    echo "✓ Resource $resource is available"
  else
    echo "⚠ Resource $resource is not running (will be started during integration tests)"
  fi
done

echo ""
echo "✅ test-dependencies.sh completed successfully"
