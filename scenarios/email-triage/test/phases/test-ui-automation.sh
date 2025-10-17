#!/bin/bash
# Email Triage UI Automation Tests
# Runs Jest-based UI automation tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
UI_DIR="$SCENARIO_ROOT/ui"

echo "🧪 Running Email Triage UI automation tests..."

# Ensure UI is running
if [ -z "${UI_PORT:-}" ]; then
    echo "❌ UI_PORT environment variable not set"
    exit 1
fi

# Check if UI is accessible
if ! curl -sf "http://localhost:${UI_PORT}" > /dev/null 2>&1; then
    echo "❌ UI not accessible on port ${UI_PORT}"
    exit 1
fi

echo "  ✓ UI server running on port ${UI_PORT}"

# Run Jest tests
cd "$UI_DIR"
echo "  Running Jest UI automation tests..."

if UI_PORT="$UI_PORT" API_PORT="${API_PORT:-19525}" npm test; then
    echo "✅ UI automation tests passed"
else
    echo "❌ UI automation tests failed"
    exit 1
fi
