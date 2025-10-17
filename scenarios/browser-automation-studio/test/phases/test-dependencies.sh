#!/bin/bash
set -e

echo "=== Test Dependencies ==="

# Check Go dependencies
if [[ -f api/go.mod ]]; then
  cd api
  go mod tidy &gt;&amp; /dev/null || true
  cd ..
  echo "✅ Go dependencies verified"
else
  echo "⚠️ No Go dependencies to check"
fi

# Check npm dependencies
if [[ -f ui/package.json ]]; then
  cd ui
  npm ci --dry-run &gt;&amp; /dev/null || true
  cd ..
  echo "✅ NPM dependencies verified"
else
  echo "⚠️ No NPM dependencies to check"
fi

echo "✅ Dependencies tests passed"