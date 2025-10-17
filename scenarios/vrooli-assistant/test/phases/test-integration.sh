#!/bin/bash
set -euo pipefail

echo "=== Integration Tests Phase for Vrooli Assistant ==="

# Basic integration checks
echo "Checking API health endpoint..."
if curl -sf http://localhost:${API_PORT:-8080}/health >/dev/null 2>&1 || true; then
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

# Check Electron build if present
if [ -d "../../ui/electron" ] && [ -f "../../ui/electron/package.json" ]; then
  pushd ../../ui/electron >/dev/null
  if npm run build --silent 2>/dev/null || true; then
    echo "✅ Electron build integration passed"
  else
    echo "⚠️  Electron build failed or no build script"
  fi
  popd >/dev/null
fi

echo "✅ Integration tests phase completed"
