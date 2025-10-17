#!/bin/bash
set -euo pipefail

echo "=== Dependency Tests ==="

failed=0

# Check Go
if command -v go &> /dev/null; then
  go_version=$(go version)
  echo "✅ Go: $go_version"
else
  echo "❌ Go not found"
  failed=1
fi

# Check Node.js
if command -v node &> /dev/null; then
  node_version=$(node --version)
  echo "✅ Node.js: $node_version"
else
  echo "❌ Node.js not found"
  failed=1
fi

# Check npm
if command -v npm &> /dev/null; then
  npm_version=$(npm --version)
  echo "✅ npm: $npm_version"
else
  echo "❌ npm not found"
  failed=1
fi

# Check jq (required for CLI)
if command -v jq &> /dev/null; then
  jq_version=$(jq --version)
  echo "✅ jq: $jq_version"
else
  echo "❌ jq not found (required for CLI)"
  failed=1
fi

# Check curl
if command -v curl &> /dev/null; then
  curl_version=$(curl --version | head -1)
  echo "✅ curl: $curl_version"
else
  echo "❌ curl not found"
  failed=1
fi

# Check Go modules
if [ -f "api/go.mod" ]; then
  echo "✅ Go modules file exists"
  cd api && go mod verify &> /dev/null && echo "   Go modules verified" || echo "   ⚠️  Go modules verification failed"
  cd ..
fi

# Check UI dependencies
if [ -f "ui/package.json" ]; then
  echo "✅ UI package.json exists"
  if [ -d "ui/node_modules" ]; then
    echo "   Node modules installed"
  else
    echo "   ⚠️  Node modules not installed (run: cd ui && npm install)"
  fi
fi

exit $failed
