#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== Running tests for $(basename "$SCENARIO_DIR") ==="

# Run all phase tests
for phase in "$SCRIPT_DIR"/phases/*.sh; do
    if [[ -f "$phase" ]]; then
        echo "Running phase: $(basename "$phase")"
        bash "$phase"
        echo "✓ $(basename "$phase") completed"
    fi
done

echo "✅ All test phases completed successfully"