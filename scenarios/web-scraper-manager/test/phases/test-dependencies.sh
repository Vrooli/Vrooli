#!/bin/bash
set -e

echo "=== Testing Dependencies ==="

# Test Go dependencies
if [ -d "api" ] && [ -f "api/go.mod" ]; then
    echo "Testing Go module dependencies..."
    cd api
    go mod tidy

    if ! go mod verify; then
        echo "❌ Go dependency verification failed"
        exit 1
    fi
    cd ..
    echo "✓ Go dependencies verified"
fi

# Test Node.js dependencies
if [ -d "ui" ] && [ -f "ui/package.json" ]; then
    echo "Testing Node.js dependencies..."
    if [ ! -d "ui/node_modules" ]; then
        echo "⚠️  Node modules not installed, skipping dependency check"
    else
        echo "✓ Node.js dependencies present"
    fi
fi

# Test CLI dependencies
echo "Testing CLI dependencies..."
if ! command -v jq &> /dev/null; then
    echo "❌ jq not found"
    exit 1
fi
if ! command -v curl &> /dev/null; then
    echo "❌ curl not found"
    exit 1
fi
echo "✓ CLI dependencies (jq, curl) present"

echo "✅ Dependencies tests passed"
