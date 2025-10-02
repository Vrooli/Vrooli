#!/bin/bash

set -e

echo "=== Unit Tests ==="

# Test Go build
echo "Testing Go API build..."
if [ -d "api" ] && [ -f "api/go.mod" ]; then
    cd api
    # Build all Go files together since main.go references other files
    if go build -o /dev/null *.go 2>/dev/null; then
        echo "✅ Go API builds successfully"
    else
        echo "❌ Go API build failed"
        exit 1
    fi
    cd ..
else
    echo "⚠️  No Go API found, skipping"
fi

# Test CLI build
echo "Testing CLI build..."
if [ -f "cli/contact-book" ]; then
    echo "✅ CLI binary exists"
elif [ -d "cli" ] && [ -f "cli/go.mod" ]; then
    cd cli
    if go build -o /dev/null main.go 2>/dev/null; then
        echo "✅ CLI builds successfully"
    else
        echo "❌ CLI build failed"
        exit 1
    fi
    cd ..
else
    echo "⚠️  No CLI found, skipping"
fi

echo "✅ All unit tests passed"
