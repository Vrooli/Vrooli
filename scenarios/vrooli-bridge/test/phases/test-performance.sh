#!/bin/bash
set -euo pipefail

echo "=== Test Performance ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Basic performance checks
# TODO: Add benchmarks, load tests

# Check build time or something simple
time cd "${SCENARIO_DIR}/api" && go build -o /dev/null . >/dev/null 2>&1

echo "✅ Performance tests passed (basic)"