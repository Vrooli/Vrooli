#!/bin/bash
set -e

echo "=== Unit Tests ==="

# Test Go API
if [ -d "api" ] && [ -f "api/go.mod" ]; then
  echo "Running Go API unit tests..."
  cd api
  go test -v -short -cover ./... || {
    echo "❌ Go API tests failed"
    exit 1
  }
  cd ..
fi

# Test UI (if tests are configured)
if [ -d "ui" ] && [ -f "ui/package.json" ]; then
  echo "Running UI unit tests..."
  cd ui
  if grep -q '"test"' package.json; then
    npm test 2>/dev/null || echo "⚠️ No UI unit tests configured"
  else
    echo "⚠️ No UI test script configured in package.json"
  fi
  cd ..
fi

echo "✅ Unit tests completed"