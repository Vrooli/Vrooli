#!/bin/bash
# Run text-tools CLI BATS tests through the shared test type interface
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"
CLI_DIR="$SCENARIO_DIR/cli"

if ! command -v bats >/dev/null 2>&1; then
  echo "⚠️  BATS not installed; skipping CLI tests" >&2
  exit 0
fi

if [ ! -x "$CLI_DIR/text-tools" ]; then
  echo "⚠️  CLI binary not found or not executable at $CLI_DIR/text-tools; skipping CLI tests" >&2
  exit 0
fi

if [ ! -f "$CLI_DIR/text-tools.bats" ]; then
  echo "⚠️  CLI BATS suite not found; skipping CLI tests" >&2
  exit 0
fi

echo "🧪 Running text-tools CLI BATS suite..."
if bats "$CLI_DIR/text-tools.bats"; then
  echo "✅ CLI tests passed"
  exit 0
else
  echo "❌ CLI tests failed" >&2
  exit 1
fi
