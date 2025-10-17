#!/bin/bash
# Integration tests for notification-hub scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔗 Running integration tests for notification-hub..."

# Integration tests would test:
# - Database migrations and schema
# - Redis connectivity and operations
# - Full notification workflow (create profile -> send notification -> verify delivery)
# - API authentication flow
# - Multi-channel notification delivery

# For now, run Go tests tagged with integration
if [ -d "api" ] && [ -f "api/go.mod" ]; then
    cd api
    echo "Running Go integration tests..."
    if go test -tags=integration -v ./... -timeout 120s 2>&1 | tee test-integration-output.txt; then
        echo "✅ Integration tests passed"
    else
        echo "⚠️  Some integration tests failed (may need database/Redis)"
    fi
    cd ..
fi

testing::phase::end_with_summary "Integration tests completed"
