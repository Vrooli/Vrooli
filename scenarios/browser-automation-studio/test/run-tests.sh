#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &amp;&amp; pwd)"
echo "=== Running All Test Phases for Browser Automation Studio ==="

phases=(test-structure.sh test-dependencies.sh test-unit.sh test-integration.sh test-performance.sh test-business.sh)
for phase in "${phases[@]}"; do
  phase_file="${SCRIPT_DIR}/phases/$phase"
  if [[ -f "$phase_file" ]]; then
    echo ""
    echo "Running $phase..."
    bash "$phase_file"
  else
    echo "⚠️ Phase $phase not found, skipping"
  fi
done

echo ""
echo "🎉 All available test phases completed successfully!"