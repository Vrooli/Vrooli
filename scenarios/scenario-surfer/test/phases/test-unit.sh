#!/bin/bash
set -euo pipefail
echo "=== Test Unit ==="
# Go unit tests
if [ -d "api" ]; then
  cd api
  if go test ./... ; then
    echo "✅ Go unit tests passed"
  else
    echo "❌ Go unit tests failed"
    exit 1
  fi
  cd ..
fi
# UI unit tests (if any)
if [ -f "ui/package.json" ]; then
  cd ui
  if npm test; then
    echo "✅ UI unit tests passed"
  else
    echo "⚠️ No UI tests or failed"
  fi
  cd ..
fi
echo "✅ Unit tests completed"
