#!/bin/bash
set -euo pipefail

echo "=== Test Unit ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Run Go unit tests
cd "${SCENARIO_DIR}/api"
if ! go test ./... -short; then
  echo "❌ Unit tests failed"
  exit 1
fi

# TODO: Add UI unit tests if applicable

echo "✅ Unit tests passed"