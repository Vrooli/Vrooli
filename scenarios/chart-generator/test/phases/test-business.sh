#!/bin/bash
# Business-layer validation: ensure core workflow capabilities are wired correctly

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
SCENARIO_ROOT="${SCENARIO_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)}"

source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/business.sh"

echo "🧪 Running API integration tests..."

# Get API port from vrooli
API_PORT=$(vrooli scenario port chart-generator API_PORT 2>/dev/null || echo "18594")
export API_PORT

# Run focused API integration tests
if [[ -f "${SCENARIO_ROOT}/test/api-integration.test.cjs" ]]; then
  if node "${SCENARIO_ROOT}/test/api-integration.test.cjs"; then
    echo "✅ API integration tests passed"
  else
    echo "❌ API integration tests failed"
    exit 1
  fi
else
  echo "⚠️  API integration test file not found (expected: test/api-integration.test.cjs)"
fi

testing::business::validate_all
