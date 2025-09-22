#!/bin/bash

set -e

echo "=== Unit Tests ==="

# Test Go API unit tests
echo "Running Go API unit tests..."
cd api
if [ -f "go.mod" ]; then
    go test -v ./... -short
    echo "✅ Go API unit tests passed"
else
    echo "⚠️  No go.mod found, skipping Go tests"
fi
cd ..

# Test CLI unit tests
echo "Running CLI unit tests..."
cd cli
if [ -f "go.mod" ]; then
    go test -v ./... -short
    echo "✅ CLI unit tests passed"
else
    echo "⚠️  No go.mod found, skipping CLI tests"
fi
cd ..

# Test UI unit tests
echo "Running UI unit tests..."
cd ui
if [ -f "package.json" ] && [ -d "node_modules" ]; then
    if npm list --depth=0 | grep -q "jest\|vitest\|@testing-library"; then
        npm test -- --run --reporter=verbose
        echo "✅ UI unit tests passed"
    else
        echo "⚠️  No testing framework found, skipping UI tests"
    fi
else
    echo "⚠️  UI dependencies not installed, skipping UI tests"
fi
cd ..

echo "=== Unit Tests Complete ==="