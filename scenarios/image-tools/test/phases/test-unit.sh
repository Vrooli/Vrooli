#!/bin/bash
# Test: Unit tests
# Tests individual plugin functionality

set -e

echo "  ✓ Running plugin unit tests..."

cd api

# Test plugin loading
go test -v ./plugins/... 2>&1 | grep -q "no test files" && {
    echo "  ⚠️  No unit tests found for plugins"
} || {
    go test -v ./plugins/... || {
        echo "  ❌ Plugin tests failed"
        exit 1
    }
}

cd ..

echo "  ✓ Unit tests complete"