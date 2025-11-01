#!/bin/bash
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Running phased test suite for $(basename "$SCENARIO_DIR")"
"$SCENARIO_DIR/test/run-tests.sh" "$@"
