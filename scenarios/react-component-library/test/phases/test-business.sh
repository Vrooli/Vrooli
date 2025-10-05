#!/bin/bash
# Business logic testing phase for react-component-library scenario

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "💼 Running React Component Library business logic tests..."
echo "=========================================================="

# Run business logic tests via Go test
cd api

echo "🧪 Running business logic test suite..."
echo ""

# Run tests that validate business requirements
go test -v -timeout=120s \
    -run "TestComponent.*|TestSearch.*|TestAnalytics.*|TestLifecycle.*" \
    ./... 2>&1 | tee ../test-business.log

test_result=${PIPESTATUS[0]}

echo ""

if [ $test_result -eq 0 ]; then
    echo "✅ Business logic tests passed"
else
    echo "❌ Business logic tests failed (exit code: $test_result)"
    echo ""
    echo "📝 Check test-business.log for details"
fi

cd ..

testing::phase::end_with_summary "Business logic tests completed"

exit $test_result
