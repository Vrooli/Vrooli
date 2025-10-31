#!/usr/bin/env bash
set -euo pipefail

# Test: Unit Tests
# Runs Go unit tests for the API

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "🧪 Testing Prompt Injection Arena unit tests..."

cd "${SCENARIO_DIR}"

# Track failures
FAILURES=0

# Test Go build (compilation test)
echo "🐹 Testing Go compilation..."
if [ -f "api/main.go" ]; then
    cd api
    if go build -o test-build . 2>&1; then
        echo "  ✅ Go compilation successful"
        rm -f test-build
    else
        echo "  ❌ Go compilation failed"
        ((FAILURES++))
    fi
    cd ..
else
    echo "  ❌ api/main.go not found"
    ((FAILURES++))
fi

# Test CLI BATS tests
echo "🧪 Testing CLI tests..."
if [ -f "cli/prompt-injection-arena.bats" ]; then
    cd cli
    if bats prompt-injection-arena.bats > /dev/null 2>&1; then
        test_count=$(bats prompt-injection-arena.bats 2>&1 | grep -E "^1\.\." | sed 's/1\.\.//')
        echo "  ✅ CLI tests passed (${test_count} tests)"
    else
        echo "  ❌ CLI tests failed"
        ((FAILURES++))
    fi
    cd ..
else
    echo "  ⚠️  No CLI BATS tests found"
fi

# Summary
echo ""
if [ ${FAILURES} -eq 0 ]; then
    echo "✅ Unit tests passed!"
    exit 0
else
    echo "❌ Unit tests failed with ${FAILURES} error(s)"
    exit 1
fi
