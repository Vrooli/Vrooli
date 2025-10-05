#!/bin/bash
# Integration testing phase for react-component-library scenario

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔗 Running React Component Library integration tests..."
echo "======================================================="

# Check for required resources
echo "📊 Checking resource availability..."

resources_available=0
resources_checked=0

# Check PostgreSQL
echo -n "  PostgreSQL: "
resources_checked=$((resources_checked + 1))
if command -v resource-postgres &> /dev/null; then
    if resource-postgres status &> /dev/null; then
        echo "✅ Available"
        resources_available=$((resources_available + 1))
    else
        echo "⚠️  CLI available but service not running"
    fi
else
    echo "❌ Not available"
fi

# Check Qdrant
echo -n "  Qdrant: "
resources_checked=$((resources_checked + 1))
if command -v resource-qdrant &> /dev/null; then
    if resource-qdrant status &> /dev/null; then
        echo "✅ Available"
        resources_available=$((resources_available + 1))
    else
        echo "⚠️  CLI available but service not running"
    fi
else
    echo "❌ Not available"
fi

# Check MinIO
echo -n "  MinIO: "
resources_checked=$((resources_checked + 1))
if command -v resource-minio &> /dev/null; then
    if resource-minio status &> /dev/null; then
        echo "✅ Available"
        resources_available=$((resources_available + 1))
    else
        echo "⚠️  CLI available but service not running"
    fi
else
    echo "❌ Not available"
fi

echo ""
echo "Resource availability: $resources_available/$resources_checked"

if [ $resources_available -lt 1 ]; then
    echo "⚠️  WARNING: Limited resource availability may cause some tests to fail or be skipped"
fi

# Run integration tests via Go test
echo ""
echo "🧪 Running integration test suite..."
cd api

go test -v -tags=integration -timeout=120s \
    -run "TestComponentLifecycle|TestComponentSearch|TestComponentAnalytics" \
    ./... 2>&1 | tee ../test-integration.log

test_result=${PIPESTATUS[0]}

echo ""
if [ $test_result -eq 0 ]; then
    echo "✅ Integration tests passed"
else
    echo "❌ Integration tests failed (exit code: $test_result)"
    echo ""
    echo "📝 Check test-integration.log for details"
fi

cd ..

testing::phase::end_with_summary "Integration tests completed"

exit $test_result
