#!/bin/bash
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "=== Running comprehensive tests for audio-intelligence-platform ==="
# Run all test phases
${SCRIPT_DIR}/phases/test-structure.sh
${SCRIPT_DIR}/phases/test-dependencies.sh
${SCRIPT_DIR}/phases/test-unit.sh
${SCRIPT_DIR}/phases/test-integration.sh
${SCRIPT_DIR}/phases/test-performance.sh
${SCRIPT_DIR}/phases/test-business.sh
echo "✅ All test phases completed successfully"