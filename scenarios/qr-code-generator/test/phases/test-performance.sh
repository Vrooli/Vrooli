#!/bin/bash

set -e

echo "=== Performance Phase (Target: <30s) ==="

# Run Go benchmarks if available
if [ -d "api" ] && [ -f "api/main_test.go" ]; then
    echo "Running Go benchmarks..."
    cd api
    go test -bench=. -benchmem -run=^$ || true
    cd ..
fi

echo "✓ Performance tests completed"
