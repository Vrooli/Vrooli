#!/bin/bash
set -euo pipefail

# Comprehensive test runner for the scenario

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../" && pwd)"
cd "$SCENARIO_DIR"

echo "=== Running comprehensive tests for App Issue Tracker ==="

# Run all phase tests
PHASES_DIR="test/phases"
if [ -d "$PHASES_DIR" ]; then
  for phase in "$PHASES_DIR"/test-*.sh; do
    if [ -f "$phase" ]; then
      echo ""
      echo "Running phase: $(basename "$phase")"
      bash "$phase"
    fi
  done
else
  echo "✗ Phases directory missing: $PHASES_DIR"
  exit 1
fi

# Additional scenario-specific tests
if [ -f test/test-investigation-workflow.sh ]; then
  echo ""
  echo "Running investigation workflow test"
  bash test/test-investigation-workflow.sh
fi

if [ -f test/test-search-endpoint.sh ]; then
  echo ""
  echo "Running search endpoint test"
  bash test/test-search-endpoint.sh
fi

echo ""
echo "✅ All tests completed successfully!"
