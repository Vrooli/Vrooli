#!/bin/bash
set -euo pipefail

echo "=== Running test-dependencies.sh ==="

# Check dependencies

if command -v go >/dev/null 2>&1; then
  echo "✓ Go available"
else
  echo "✗ Go not found"
  exit 1
fi

if [ -f api/go.mod ]; then
  cd api && go mod tidy >/dev/null 2>&1 && echo "✓ Go modules tidy"
else
  echo "⚠ No Go API, skipping module check"
fi

if [ -f ui/package.json ]; then
  cd ui && npm install --dry-run >/dev/null 2>&1 && echo "✓ NPM dependencies ok"
else
  echo "⚠ No UI, skipping NPM check"
fi

echo "✅ test-dependencies.sh completed successfully"
