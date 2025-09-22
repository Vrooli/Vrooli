#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== Running Research Assistant Tests ==="

# Run all test phases
${SCRIPT_DIR}/test/phases/test-structure.sh
${SCRIPT_DIR}/test/phases/test-dependencies.sh
${SCRIPT_DIR}/test/phases/test-unit.sh
${SCRIPT_DIR}/test/phases/test-integration.sh
${SCRIPT_DIR}/test/phases/test-performance.sh
${SCRIPT_DIR}/test/phases/test-business.sh

echo "✅ All test phases completed successfully"