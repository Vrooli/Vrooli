#!/bin/bash
set -euo pipefail

echo "=== Unit Tests ==="

if [ -d api ]; then
  cd api && go test -v ./... -short || { echo "Unit tests failed ❌"; exit 1; }
  echo "✅ Unit tests completed"
else
  echo "No API directory, skipping unit tests"
fi

if [ -d ui ] && [ -f ui/package.json ]; then
  cd ui && npm test || { echo "UI tests failed ❌"; exit 1; } || echo "No UI tests configured, skipping"
  echo "✅ UI unit tests completed"
fi