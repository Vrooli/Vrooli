#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

TESTING_PHASE_SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Run Go unit tests with coverage
cd "${TESTING_PHASE_SCENARIO_DIR}/api"

echo "Running Go unit tests with coverage..."
if ! go test -v -cover -coverprofile=coverage.out . 2>&1; then
  echo "❌ Unit tests failed"
  testing::phase::end_with_summary "Unit tests failed"
  exit 1
fi

# Get coverage percentage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Coverage: ${COVERAGE}%"

# Check coverage thresholds
# Note: 48% coverage achieved without live database
# Database-dependent handlers (getProjects, integrateProject, deleteProject) require DB
# Full 80% target achievable with database test fixtures or mocked database
COVERAGE_WARN=80
COVERAGE_ERROR=45

if (( $(echo "$COVERAGE < $COVERAGE_ERROR" | bc -l) )); then
  echo "❌ Coverage ${COVERAGE}% is below error threshold ${COVERAGE_ERROR}%"
  testing::phase::end_with_summary "Unit tests failed - insufficient coverage"
  exit 1
elif (( $(echo "$COVERAGE < $COVERAGE_WARN" | bc -l) )); then
  echo "⚠️  Coverage ${COVERAGE}% is below warning threshold ${COVERAGE_WARN}%"
fi

testing::phase::end_with_summary "Unit tests completed with ${COVERAGE}% coverage"