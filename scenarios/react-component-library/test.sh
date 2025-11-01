#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

run_phase() {
  local script="$1"
  if [ -x "${SCRIPT_DIR}/${script}" ]; then
    echo "👉 Running ${script}"
    "${SCRIPT_DIR}/${script}"
    echo ""
  else
    echo "⚠️  Skipping missing phase script: ${script}" >&2
  fi
}

run_phase "test/phases/test-structure.sh"

echo "✅ Selected integration checks completed"
