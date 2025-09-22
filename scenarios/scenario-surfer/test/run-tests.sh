#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &amp;&amp; pwd)"
cd "$SCRIPT_DIR"

echo "=== Running Scenario Surfer Tests ==="
echo "1. Dependencies..."
"$SCRIPT_DIR/phases/test-dependencies.sh"
echo "2. Structure..."
"$SCRIPT_DIR/phases/test-structure.sh"
echo "3. Unit..."
"$SCRIPT_DIR/phases/test-unit.sh"
echo "4. Integration..."
"$SCRIPT_DIR/phases/test-integration.sh"
echo "5. Business..."
"$SCRIPT_DIR/phases/test-business.sh"
echo "6. Performance..."
"$SCRIPT_DIR/phases/test-performance.sh"
echo "✅ All tests passed!"
