#!/bin/bash
# Unit tests for mind-maps scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Run tests with -tags=testing flag
echo "🧪 Running Go unit tests with testing tag..."
cd api
go test -tags=testing -v -coverprofile=coverage.out -covermode=atomic

# Check coverage
COVERAGE=$(go tool cover -func=coverage.out | grep "total:" | awk '{print $3}' | sed 's/%//')
echo "📊 Go Test Coverage: ${COVERAGE}%"

if (( $(echo "$COVERAGE < 50" | bc -l) )); then
    echo "❌ ERROR: Go test coverage (${COVERAGE}%) is below error threshold (50%)"
    exit 1
fi

if (( $(echo "$COVERAGE < 80" | bc -l) )); then
    echo "⚠️  WARNING: Go test coverage (${COVERAGE}%) is below warning threshold (80%)"
fi

cd ..

testing::phase::end_with_summary "Unit tests completed"
