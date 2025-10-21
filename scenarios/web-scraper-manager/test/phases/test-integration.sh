#!/bin/bash
# Integration tests for Web Scraper Manager

set -e

echo "Running integration tests for Web Scraper Manager..."

# Test database integration
echo "✓ Testing database integration..."
if ! bash test/test-database-connection.sh > /dev/null 2>&1; then
    echo "❌ Database integration test failed"
    exit 1
fi

# Test storage integration
echo "✓ Testing storage integration..."
if ! bash test/test-storage-buckets.sh > /dev/null 2>&1; then
    echo "❌ Storage integration test failed"
    exit 1
fi

# Test CLI integration
echo "✓ Testing CLI integration..."
if [ -f "cli/web-scraper-manager.bats" ]; then
    cd cli
    if ! bats web-scraper-manager.bats > /dev/null 2>&1; then
        echo "❌ CLI integration tests failed"
        exit 1
    fi
    cd ..
else
    echo "⚠️  No CLI tests found"
fi

echo "✅ Integration tests passed"
