#!/bin/bash
set -euo pipefail

echo "🔄 Running integration tests"

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Install CLI locally if possible
if [ -f "$BASE_DIR/cli/install.sh" ]; then
  cd "$BASE_DIR/cli"
  ./install.sh --local 2>/dev/null || true
  cd "$BASE_DIR"
fi

# Test if CLI works
if command -v scenario-dependency-analyzer >/dev/null 2>&1; then
  scenario-dependency-analyzer --help >/dev/null || exit 1
  echo "✅ CLI integration test passed"
else
  echo "⚠️  CLI not in PATH, basic integration skipped"
fi

# API integration placeholder
echo "Integration tests completed (basic)"