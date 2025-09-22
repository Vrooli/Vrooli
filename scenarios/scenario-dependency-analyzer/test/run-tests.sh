#!/bin/bash
set -euo pipefail

echo "🚀 Starting comprehensive tests for scenario-dependency-analyzer"

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$BASE_DIR"

PHASES_DIR="test/phases"
if [ ! -d "$PHASES_DIR" ]; then
  echo "❌ Test phases directory not found: $BASE_DIR/$PHASES_DIR"
  exit 1
fi

failed=0
for phase in "$PHASES_DIR"/test-*.sh; do
  if [ -f "$phase" ]; then
    echo ""
    echo "=== Running $phase ==="
    if bash "$phase"; then
      echo "✅ $phase passed"
    else
      echo "❌ $phase failed"
      failed=1
    fi
  fi
done

if [ $failed -eq 0 ]; then
  echo "🎉 All tests passed!"
  exit 0
else
  echo "💥 Some tests failed!"
  exit 1
fi