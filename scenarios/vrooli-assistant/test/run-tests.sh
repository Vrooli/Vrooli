#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="$SCRIPT_DIR/phases"

echo "=== Running Vrooli Assistant Standard Tests ==="

if [ ! -d "$PHASES_DIR" ]; then
  echo "❌ Test phases directory not found: $PHASES_DIR"
  exit 1
fi

cd "$PHASES_DIR"

for phase in test-*.sh; do
  if [ -f "$phase" ]; then
    echo ""
    echo "🔄 Running $phase..."
    if bash "$phase"; then
      echo "✅ $phase completed successfully"
    else
      echo "❌ $phase failed"
      exit 1
    fi
  else
    echo "⚠️  Phase script not found: $phase"
  fi
done

echo ""
echo "🎉 All standard test phases completed successfully!"
