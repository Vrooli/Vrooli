#!/bin/bash
# Integration tests: Direct HTTP API tests (pragmatic approach vs BAS workflows)
# See docs/PROBLEMS.md for BAS format incompatibility details

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
SCENARIO_ROOT="$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)"

source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Export API_PORT for BATS tests
export API_PORT=$(vrooli scenario port visited-tracker API_PORT 2>/dev/null || echo "17693")

echo "🔗 Running API integration tests..."
echo ""

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Run each BATS test file
for test_file in "${SCENARIO_ROOT}/test/api"/*.bats; do
    if [ -f "$test_file" ]; then
        test_name=$(basename "$test_file" .bats)
        echo "  Running: $test_name"

        # Run BATS and capture output
        output=$(bats "$test_file" 2>&1)
        exit_code=$?

        # Parse test count from first line (format: "1..N")
        test_count=$(echo "$output" | head -1 | cut -d'.' -f3)

        if [ "$exit_code" -eq 0 ]; then
            PASSED_TESTS=$((PASSED_TESTS + test_count))
            TOTAL_TESTS=$((TOTAL_TESTS + test_count))
            echo "    ✅ $test_count tests passed"
        else
            # Count failures
            failures=$(echo "$output" | grep -c "^not ok")
            passes=$((test_count - failures))

            PASSED_TESTS=$((PASSED_TESTS + passes))
            FAILED_TESTS=$((FAILED_TESTS + failures))
            TOTAL_TESTS=$((TOTAL_TESTS + test_count))

            echo "    ❌ $failures/$test_count tests failed"
            echo "$output" | grep "^not ok" | head -5
        fi
        echo ""
    fi
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Integration Test Results"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Total:  $TOTAL_TESTS tests"
echo "  Passed: $PASSED_TESTS tests ($(( PASSED_TESTS * 100 / TOTAL_TESTS ))%)"
echo "  Failed: $FAILED_TESTS tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ "$FAILED_TESTS" -gt 0 ]; then
    echo ""
    echo "❌ Integration tests failed"
    exit 1
fi

echo ""
echo "✅ All integration tests passed"
