#!/bin/bash
set -e

# CLI tests for text-tools scenario
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR/cli"

echo "=== CLI Tests ==="

# Run BATS test suite
if [ -f "text-tools.bats" ]; then
    echo "Running BATS CLI tests..."
    bats text-tools.bats || {
        echo "❌ CLI tests failed"
        exit 1
    }
    echo "✅ CLI tests passed"
else
    echo "⚠️  No BATS tests found"
fi

# End test phase with summary
testing::phase::end_with_summary "CLI tests completed"
