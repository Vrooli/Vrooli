#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=== Running Vrooli Bridge Tests ==="

"${SCRIPT_DIR}/phases/test-structure.sh"
"${SCRIPT_DIR}/phases/test-dependencies.sh"
"${SCRIPT_DIR}/phases/test-unit.sh"
"${SCRIPT_DIR}/phases/test-integration.sh"
"${SCRIPT_DIR}/phases/test-performance.sh"
"${SCRIPT_DIR}/phases/test-business.sh"

echo "✅ All Vrooli Bridge tests passed!"