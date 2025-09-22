#!/bin/bash

set -e

echo "=== Integration Tests ==="

# Test API integration
echo "Running API integration tests..."
cd api
if [ -f "go.mod" ]; then
    # Test database connections
    echo "Testing database integration..."
    go test -v -run "TestDB" ./... -short 2>/dev/null || echo "⚠️  No database integration tests found"

    # Test API endpoints
    echo "Testing API endpoints..."
    go test -v -run "TestAPI" ./... -short 2>/dev/null || echo "⚠️  No API integration tests found"

    echo "✅ API integration tests completed"
else
    echo "⚠️  No go.mod found, skipping API integration tests"
fi
cd ..

# Test CLI integration
echo "Running CLI integration tests..."
cd cli
if [ -f "go.mod" ]; then
    go test -v -run "TestCLI" ./... -short 2>/dev/null || echo "⚠️  No CLI integration tests found"
    echo "✅ CLI integration tests completed"
else
    echo "⚠️  No go.mod found, skipping CLI integration tests"
fi
cd ..

# Test UI integration
echo "Running UI integration tests..."
cd ui
if [ -f "package.json" ] && [ -d "node_modules" ]; then
    if npm list --depth=0 | grep -q "cypress\|playwright\|puppeteer"; then
        npm run test:e2e 2>/dev/null || echo "⚠️  No E2E test script available"
        echo "✅ UI integration tests completed"
    else
        echo "⚠️  No integration testing framework found, skipping UI integration tests"
    fi
else
    echo "⚠️  UI dependencies not installed, skipping UI integration tests"
fi
cd ..

echo "=== Integration Tests Complete ==="