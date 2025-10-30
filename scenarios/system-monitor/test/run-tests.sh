#!/bin/bash
set -euo pipefail

echo "Running comprehensive tests for System Monitor"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASE_DIR="${SCRIPT_DIR}/phases"

if [[ ! -d "${PHASE_DIR}" ]]; then
  echo "Missing test phases directory: ${PHASE_DIR}" >&2
  exit 1
fi

run_phase() {
  local script_name="$1"
  bash "${PHASE_DIR}/${script_name}"
}

echo "=== Running test phases ==="
run_phase "test-unit.sh"
run_phase "test-business.sh"
run_phase "test-dependencies.sh"
run_phase "test-integration.sh"
run_phase "test-performance.sh"
run_phase "test-structure.sh"

echo "All tests completed successfully!"
