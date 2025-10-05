#!/bin/bash
set -e

# Text Tools Test Runner - Orchestrates all test phases
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"

SCENARIO_DIR="$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)"
TEST_PHASES_DIR="${SCENARIO_DIR}/test/phases"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Text Tools - Comprehensive Test Suite"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Default to running all phases
PHASES="${1:-unit cli api integration}"

for phase in $PHASES; do
    phase_script="${TEST_PHASES_DIR}/test-${phase}.sh"

    if [ -f "$phase_script" ]; then
        echo ""
        echo "▶ Running ${phase} tests..."
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        bash "$phase_script" || {
            echo "❌ ${phase} tests failed"
            exit 1
        }
    else
        echo "⚠️  Skipping ${phase} - test script not found: $phase_script"
    fi
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ All tests passed successfully!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
