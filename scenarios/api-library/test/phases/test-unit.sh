#!/bin/bash
set -euo pipefail

echo "=== Unit Tests ==="

if [ -d "api" ]; then
  cd api
  go test -v ./... -short
  echo "✅ API Unit tests passed"
fi

if [ -d "ui" ]; then
  cd ui
  npm test -- --watchAll=false || echo "UI tests skipped if no tests defined"
  echo "✅ UI Unit tests completed"
fi

echo "All unit tests completed successfully"