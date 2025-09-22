#!/bin/bash
set -euo pipefail

echo "=== Test Dependencies ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Check Go module
if [[ ! -f "${SCENARIO_DIR}/api/go.mod" ]]; then
  echo "❌ Missing api/go.mod"
  exit 1
fi

cd "${SCENARIO_DIR}/api" && go mod tidy >/dev/null 2>&1 || { echo "❌ Go dependencies check failed"; exit 1; }

# Check CLI install script
if [[ ! -f "${SCENARIO_DIR}/cli/install.sh" ]]; then
  echo "❌ Missing cli/install.sh"
  exit 1
fi

# Check service.json schema
if [[ ! -f "${SCENARIO_DIR}/.vrooli/service.json" ]]; then
  echo "❌ Missing .vrooli/service.json"
  exit 1
fi

echo "✅ Dependencies tests passed"