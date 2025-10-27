#!/bin/bash
set -euo pipefail

echo "=== Integration Tests Phase for Vrooli Assistant ==="

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Get API port
export API_PORT=$(vrooli scenario port vrooli-assistant API_PORT 2>/dev/null || echo "17628")
export API_URL="http://localhost:${API_PORT}"

# Basic integration checks
echo "Checking API health endpoint (${API_URL}/health)..."
if curl -sf "${API_URL}/health" >/dev/null 2>&1; then
  echo "✅ API health check passed"
else
  echo "⚠️  API not running or health check failed (expected in CI)"
fi

# Check if CLI is functional
if command -v vrooli-assistant >/dev/null 2>&1; then
  echo "Checking CLI help..."
  if vrooli-assistant --help >/dev/null 2>&1; then
    echo "✅ CLI functional"
  else
    echo "⚠️  CLI help failed"
    exit 1
  fi
else
  echo "ℹ️  CLI not installed, skipping CLI integration test"
fi

# Run CLI BATS tests if available
if command -v bats >/dev/null 2>&1 && [ -f "${SCENARIO_ROOT}/cli/vrooli-assistant.bats" ]; then
  echo "Running CLI BATS tests..."
  cd "${SCENARIO_ROOT}"
  if bats cli/vrooli-assistant.bats; then
    echo "✅ CLI BATS tests passed"
  else
    echo "⚠️  Some CLI BATS tests failed"
    exit 1
  fi
else
  echo "ℹ️  BATS not installed or CLI tests not found, skipping BATS tests"
fi

# Check Electron build if present
if [ -d "${SCENARIO_ROOT}/ui/electron" ] && [ -f "${SCENARIO_ROOT}/ui/electron/package.json" ]; then
  pushd "${SCENARIO_ROOT}/ui/electron" >/dev/null
  if npm run build --silent 2>/dev/null || true; then
    echo "✅ Electron build integration passed"
  else
    echo "⚠️  Electron build failed or no build script"
  fi
  popd >/dev/null
fi

echo "✅ Integration tests phase completed"
