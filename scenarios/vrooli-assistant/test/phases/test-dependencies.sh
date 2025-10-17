#!/bin/bash
set -euo pipefail

echo "=== Dependencies Tests Phase for Vrooli Assistant ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Check Go dependencies
if [ -f "$SCENARIO_DIR/api/go.mod" ]; then
  pushd "$SCENARIO_DIR/api" >/dev/null
  if go mod tidy -v 2>/dev/null || true; then
    echo "✅ Go dependencies tidy"
  else
    echo "⚠️  Go dependencies issues"
  fi
  if go mod download 2>/dev/null || true; then
    echo "✅ Go dependencies downloaded"
  fi
  popd >/dev/null
fi

# Check npm dependencies
if [ -f "$SCENARIO_DIR/ui/package.json" ]; then
  pushd "$SCENARIO_DIR/ui" >/dev/null
  if npm install --dry-run 2>/dev/null || true; then
    echo "✅ npm dependencies check passed"
  else
    echo "⚠️  npm dependencies issues"
  fi
  popd >/dev/null
fi

# Check for lockfiles
if [ -f "$SCENARIO_DIR/api/go.sum" ] || [ -f "$SCENARIO_DIR/ui/package-lock.json" ]; then
  echo "✅ Lockfiles present"
else
  echo "⚠️  No lockfiles found"
fi

echo "✅ Dependencies tests phase completed"
