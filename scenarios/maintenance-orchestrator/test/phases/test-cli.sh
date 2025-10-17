#!/bin/bash
set -euo pipefail

echo "=== CLI Tests (BATS) ==="

SCENARIO_NAME="maintenance-orchestrator"
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/cli"
BATS_FILE="${CLI_DIR}/maintenance-orchestrator.bats"

# Check if BATS is installed
if ! command -v bats &> /dev/null; then
    echo "❌ BATS not installed"
    echo "   Install with: npm install -g bats"
    exit 1
fi

# Check if BATS test file exists
if [ ! -f "$BATS_FILE" ]; then
    echo "❌ BATS test file not found: $BATS_FILE"
    exit 1
fi

# Verify scenario is running
if ! vrooli scenario port "$SCENARIO_NAME" API_PORT > /dev/null 2>&1; then
    echo "⚠️  $SCENARIO_NAME is not running"
    echo "   Start with: vrooli scenario run $SCENARIO_NAME"
    exit 1
fi

# Run BATS tests
echo "Running BATS test suite..."
cd "$CLI_DIR"
if bats maintenance-orchestrator.bats; then
    echo "✅ All CLI tests passed"
    exit 0
else
    echo "❌ Some CLI tests failed"
    exit 1
fi
