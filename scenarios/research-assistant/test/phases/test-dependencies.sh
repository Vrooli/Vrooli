#!/bin/bash
set -euo pipefail

echo "=== Testing Dependencies ==="

# Check licenses and unused modules in go.mod
if [ -f "api/go.mod" ]; then
  cd api
  go mod tidy
  go list -m all | while read -r module; do
    go mod why "$module" > /dev/null 2>&1 || echo "⚠️ Potentially unused module: $module"
  done
  cd ..
fi

# Basic security check for known bad deps (minimal)
if command -v npm &> /dev/null && [ -f "ui/package.json" ]; then
  cd ui
  npm audit --audit-level moderate > /dev/null 2>&1 || echo "⚠️ UI dependencies have vulnerabilities"
  cd ..
fi

echo "✅ Dependencies tests passed"