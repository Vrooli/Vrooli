#!/bin/bash

set -euo pipefail

echo "=== Testing Ollama Test File Directly ==="

# Set up environment
export TEST_ID="direct_test_$(date +%s)"
export TEST_TIMEOUT="60"
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
export HEALTHY_RESOURCES_STR="ollama"
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)

echo "Environment set up:"
echo "  SCRIPT_DIR: $SCRIPT_DIR"
echo "  HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"

# Change to test directory
cd "/home/matthalloran8/Vrooli/scripts/resources/tests/single/ai"

echo "=== Running ollama test with 30 second timeout ==="
timeout 30 bash ollama.test.sh