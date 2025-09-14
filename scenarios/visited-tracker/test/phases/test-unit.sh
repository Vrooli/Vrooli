#!/bin/bash
# Simplified Unit Tests Phase - <60 seconds  
# Runs unit tests for Go and Node.js components
set -uo pipefail

# Setup paths and utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Unit Tests Phase (Target: <60s) ==="
start_time=$(date +%s)

error_count=0
test_count=0
skipped_count=0

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UNIT_DIR="$TEST_DIR/unit"

echo "🧪 Running unit tests from: $TEST_DIR/unit"

# Run Go unit tests
echo ""
echo "🔍 Running Go unit tests..."
if [ -d "api" ] && [ -f "api/go.mod" ]; then
    if [ -x "$UNIT_DIR/go.sh" ]; then
        if "$UNIT_DIR/go.sh" >/dev/null 2>&1; then
            log::success "✅ Go unit tests passed"
            ((test_count++))
        else
            log::error "❌ Go unit tests failed"
            ((error_count++))
        fi
    else
        log::warning "ℹ️  Go test runner not found, skipping Go tests"
        ((skipped_count++))
    fi
else
    log::warning "ℹ️  No Go code found, skipping Go tests"
    ((skipped_count++))
fi

# Run Node.js unit tests (simplified)
echo ""
echo "🔍 Running Node.js unit tests..."
if [ -d "ui" ] && [ -f "ui/package.json" ]; then
    cd ui
    # Check if test script exists and jest is installed
    if jq -e '.scripts.test' package.json >/dev/null 2>&1 && [ -d "node_modules/.bin" ] && [ -f "node_modules/.bin/jest" ]; then
        echo "📦 Running Node.js tests..."
        set +e  # Temporarily disable exit on error
        npm test --passWithNoTests --silent >/dev/null 2>&1
        npm_exit_code=$?
        set -e  # Re-enable exit on error
        
        if [ $npm_exit_code -eq 0 ]; then
            log::success "✅ Node.js unit tests passed"
            ((test_count++))
        else
            log::error "❌ Node.js unit tests failed"
            ((error_count++))
        fi
    else
        log::warning "ℹ️  No test script or Jest not installed, skipping Node.js tests"
        ((skipped_count++))
    fi
    cd ..
else
    log::warning "ℹ️  No Node.js code found, skipping Node.js tests"
    ((skipped_count++))
fi

# Performance and summary
end_time=$(date +%s)
duration=$((end_time - start_time))
total_tests=$((test_count + error_count))

echo ""
echo "📊 Unit Test Summary:"
echo "   Tests run: $test_count"
echo "   Tests failed: $error_count" 
echo "   Tests skipped: $skipped_count"
echo "   Duration: ${duration}s"

if [ $error_count -eq 0 ]; then
    if [ $test_count -gt 0 ]; then
        log::success "✅ All unit tests passed in ${duration}s"
    else
        log::warning "⚠️  No unit tests were executed in ${duration}s"
    fi
else
    log::error "❌ Unit tests failed with $error_count failures in ${duration}s"
fi

if [ $duration -gt 60 ]; then
    log::warning "⚠️  Unit tests phase exceeded 60s target"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi