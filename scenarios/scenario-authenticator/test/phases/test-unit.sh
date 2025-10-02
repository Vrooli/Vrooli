#!/bin/bash
set -euo pipefail

echo "=== Running test-unit.sh ==="

# Run Go unit tests for scenario-authenticator
echo "Running Go unit tests..."
cd api

# Run tests with coverage
if go test ./... -v -cover -coverprofile=coverage.out; then
  echo "✓ Go unit tests passed"

  # Show coverage summary
  if [ -f coverage.out ]; then
    echo ""
    echo "Coverage summary:"
    go tool cover -func=coverage.out | grep total || true
  fi
else
  echo "✗ Go unit tests failed"
  exit 1
fi

cd ..

echo ""
echo "✅ test-unit.sh completed successfully"
