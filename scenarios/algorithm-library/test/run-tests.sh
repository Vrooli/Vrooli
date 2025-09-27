#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

SCENARIO_NAME="$(basename "$PWD")"
echo "Running all test phases for $SCENARIO_NAME"

PHASES_DIR="$SCRIPT_DIR/phases"
if [ -d "$PHASES_DIR" ]; then
  for phase in "$PHASES_DIR"/test-*.sh; do
    if [ -f "$phase" ]; then
      echo "=== Running $(basename $phase) ==="
      bash "$phase"
    fi
  done
  echo "All tests passed ✅"
else
  echo "No phases directory found"
  exit 1
fi