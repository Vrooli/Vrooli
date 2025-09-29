#!/usr/bin/env bash

# Integration Tests for Device Sync Hub
# This runs the full integration test suite
set -uo pipefail

# Test environment
export SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Run the full integration test suite
echo "Running full integration test suite..."
exec "$SCENARIO_DIR/test/integration.sh"