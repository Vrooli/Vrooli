#!/bin/bash
set -e

echo "=== Running Unit Tests ==="

# Run Go unit tests
if [ -d "api" ] && [ -f "api/go.mod" ]; then
    echo "Running Go unit tests..."
    cd api

    # Run tests with coverage
    if ! go test -v -race -coverprofile=coverage.out ./...; then
        echo "❌ Go unit tests failed"
        exit 1
    fi

    # Show coverage summary
    go tool cover -func=coverage.out | tail -1

    cd ..
    echo "✓ Go unit tests passed"
else
    echo "⚠️  No Go tests found"
fi

echo "✅ Unit tests passed"
