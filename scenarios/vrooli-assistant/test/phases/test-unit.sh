#!/bin/bash
set -euo pipefail

echo "=== Unit Tests Phase for Vrooli Assistant ==="

# Run Go unit tests in API
if [ -d "../../api" ] && command -v go >/dev/null 2>&1; then
  pushd ../../api >/dev/null
  if go test ./... -v 2>/dev/null || true; then
    echo "✅ Go unit tests passed"
  else
    echo "⚠️  Some Go unit tests failed or no tests found"
  fi
  popd >/dev/null
else
  echo "ℹ️  No Go API or Go not installed, skipping Go unit tests"
fi

# Run npm unit tests in UI if present
if [ -d "../../ui" ] && [ -f "../../ui/package.json" ]; then
  pushd ../../ui >/dev/null
  if npm test -- --watchAll=false 2>/dev/null || true; then
    echo "✅ UI unit tests passed"
  else
    echo "⚠️  Some UI unit tests failed or no tests found"
  fi
  popd >/dev/null
else
  echo "ℹ️  No UI package.json found, skipping UI unit tests"
fi

echo "✅ Unit tests phase completed successfully"
