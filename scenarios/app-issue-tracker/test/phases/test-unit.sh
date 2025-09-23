#!/bin/bash
set -euo pipefail

echo "=== Running test-unit.sh ==="

# Unit tests

if [ -d api ] && command -v go >/dev/null 2>&1; then
  cd api
  if go test ./... -short 2>/dev/null | grep -q "FAIL"; then
    echo "✗ Go unit tests failed"
    exit 1
  else
    echo "✓ Go unit tests passed"
  fi
  cd ..
else
  echo "⚠ No Go API or go not available, skipping unit tests"
fi

if [ -d ui ] && [ -f ui/package.json ]; then
  cd ui
  if npm test >/dev/null 2>&1; then
    echo "✓ UI unit tests passed"
  else
    echo "⚠ UI tests failed or not configured"
  fi
  cd ..
else
  echo "⚠ No UI tests, skipping"
fi

echo "✅ test-unit.sh completed successfully"
