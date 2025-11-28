#!/bin/bash
# Orchestrates language unit tests with coverage thresholds.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/unit.sh"

# Set DATABASE_URL for auto-campaigns tests
# Extract postgres credentials from running container
POSTGRES_USER=$(docker exec vrooli-postgres-main env | grep "^POSTGRES_USER=" | cut -d'=' -f2)
POSTGRES_PASSWORD=$(docker exec vrooli-postgres-main env | grep "^POSTGRES_PASSWORD=" | cut -d'=' -f2)
POSTGRES_PORT="5433"  # vrooli-postgres-main is exposed on 5433
export DATABASE_URL="postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/tidiness-manager?sslmode=disable"

# Run Go tests with extended timeout for comprehensive test suite (300s)
echo "🧪 Running all unit tests..."
echo ""

# Change to scenario root directory for tests
cd "$(dirname "${BASH_SOURCE[0]}")/../.."

if testing::unit::run_go_tests --dir api --timeout 300 --coverage-warn 80 --coverage-error 50; then
    echo "✅ Go tests passed"
    GO_PASSED=true
else
    echo "❌ Go tests failed"
    GO_PASSED=false
fi

# Run Node tests
if testing::unit::run_node_tests --dir ui --coverage-warn 80 --coverage-error 50; then
    echo "✅ Node tests passed"
    NODE_PASSED=true
else
    echo "❌ Node tests failed"
    NODE_PASSED=false
fi

# Exit with failure if any tests failed
if [ "$GO_PASSED" = false ] || [ "$NODE_PASSED" = false ]; then
    exit 1
fi
