#!/bin/bash

set -e

echo "=== Dependencies Tests ==="

# Check API health
echo "Checking API health..."

API_PORT="${API_PORT:-15696}"
if curl -sf "http://localhost:${API_PORT}/health" > /dev/null; then
    echo "✅ API health endpoint responsive"
else
    echo "⚠️  API health endpoint not responding (may not be started)"
fi

# Check Go toolchain
echo "Checking Go toolchain..."

if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    echo "✅ Go toolchain available: ${GO_VERSION}"
else
    echo "❌ Go toolchain required but not found"
    exit 1
fi

# Check Node.js toolchain
echo "Checking Node.js toolchain..."

if command -v node &> /dev/null; then
    NODE_VERSION=$(node --version)
    echo "✅ Node.js available: ${NODE_VERSION}"
else
    echo "⚠️  Node.js not found (optional for UI)"
fi

# Check npm
echo "Checking npm..."

if command -v npm &> /dev/null; then
    NPM_VERSION=$(npm --version)
    echo "✅ npm available: ${NPM_VERSION}"
else
    echo "⚠️  npm not found (optional for UI)"
fi

# Check jq for JSON processing
echo "Checking jq..."

if command -v jq &> /dev/null; then
    echo "✅ jq available"
else
    echo "⚠️  jq not found (recommended for JSON processing)"
fi

# Check curl for API testing
echo "Checking curl..."

if command -v curl &> /dev/null; then
    echo "✅ curl available"
else
    echo "❌ curl required for API testing"
    exit 1
fi

echo "=== Dependencies Tests Complete ==="
