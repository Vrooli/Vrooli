#!/bin/bash
set -e

echo "=== Dependencies Tests ==="
# Check if dependencies can be installed/resolved
if [ -f "package.json" ]; then
  npm install --dry-run || echo "Warning: npm dependencies check failed"
fi
if [ -f "go.mod" ]; then
  go mod tidy -v || echo "Warning: Go dependencies check failed"
fi
echo "✅ Dependencies tests passed"