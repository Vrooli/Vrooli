#!/bin/bash
set -euo pipefail

echo "=== Unit Tests ==="

# Go unit tests
if [ -d "api" ]; then
  cd api && go test -v ./... -short || exit 1
  cd ..
fi

# UI unit tests if Jest or similar is present
if [ -f "ui/package.json" ]; then
  cd ui
  if jq -e '.scripts."test"' package.json > /dev/null 2>&1; then
    npm test || exit 1
  else
    echo "No UI unit tests defined"
  fi
  cd ..
fi

echo "✅ Unit tests passed"