#!/bin/bash

set -e

echo "Running all tests for QR Code Generator"

# Get the scenario root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$SCENARIO_ROOT"

bash test/phases/test-structure.sh
bash test/phases/test-dependencies.sh
bash test/phases/test-unit.sh
bash test/phases/test-integration.sh
bash test/phases/test-performance.sh
bash test/phases/test-business.sh

echo "All tests completed."
