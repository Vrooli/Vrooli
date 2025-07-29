#!/bin/bash

set -euo pipefail

# Test the execute_test_file function directly
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VERBOSE=true
TEST_TIMEOUT=30
CLEANUP=true

# Simulate healthy resources
HEALTHY_RESOURCES=("ollama")

# Source the runner
source "$SCRIPT_DIR/framework/runner.sh"

echo "=== Testing execute_test_file function ==="
echo "HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"
echo "About to call execute_test_file..."

# Call the function directly
execute_test_file "/home/matthalloran8/Vrooli/scripts/resources/tests/single/ai/ollama.test.sh" "debug_test"
echo "Function returned with code: $?"