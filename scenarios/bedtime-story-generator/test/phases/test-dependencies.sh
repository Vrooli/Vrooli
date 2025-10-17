#!/bin/bash
set -e

echo "=== Phase 2: Dependencies Tests ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [ -d "$SCENARIO_DIR/api" ]; then
  (cd "$SCENARIO_DIR/api" && go mod tidy >/dev/null 2>&1 && go mod verify >/dev/null 2>&1)
  echo "✅ API dependencies verified"
fi

if [ -d "$SCENARIO_DIR/ui" ]; then
  (cd "$SCENARIO_DIR/ui" && npm install --dry-run >/dev/null 2>&1)
  echo "✅ UI dependencies verified"
fi

echo "✅ Dependencies tests passed"
