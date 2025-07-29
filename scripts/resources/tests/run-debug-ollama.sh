#!/bin/bash

set -euo pipefail

echo "=== Running Debug Ollama Test ==="

# Set up environment
export TEST_ID="debug_$(date +%s)"
export TEST_TIMEOUT="60"
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
export HEALTHY_RESOURCES_STR="ollama"
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)

echo "Environment set up:"
echo "  HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"

# Change to test directory
cd "/home/matthalloran8/Vrooli/scripts/resources/tests/single/ai"

echo "=== Running debug ollama test ==="
bash "/home/matthalloran8/Vrooli/scripts/resources/tests/debug-ollama-test.sh"