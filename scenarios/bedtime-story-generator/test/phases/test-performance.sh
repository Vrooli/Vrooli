#!/bin/bash
# Performance tests for bedtime-story-generator

set -e

echo "🚀 Running performance tests for bedtime-story-generator..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
API_DIR="${SCENARIO_DIR}/api"

echo "📊 Running Go performance benchmarks..."
cd "${API_DIR}"

# Run benchmarks
go test -bench=. -benchmem -benchtime=1s ./... 2>&1 | tee bench-output.txt

# Run performance tests (without -short flag)
echo "🔍 Running performance tests..."
go test -v -run="Performance|Concurrency" ./... 2>&1 | tee perf-test-output.txt

# Check if performance tests passed
if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo "✅ Performance tests passed"
else
    echo "❌ Performance tests failed"
    exit 1
fi

echo "✅ All performance tests completed successfully"
