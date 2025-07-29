#!/bin/bash

set -euo pipefail

echo "=== Manual Test Execution ==="

# Set up environment exactly as the framework does
export TEST_ID="manual_test_$(date +%s)"
export TEST_TIMEOUT="60"
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
export HEALTHY_RESOURCES_STR="ollama"
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)

echo "Environment set up:"
echo "  SCRIPT_DIR: $SCRIPT_DIR"
echo "  HEALTHY_RESOURCES_STR: $HEALTHY_RESOURCES_STR"
echo "  HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"

echo "=== Changing to test directory ==="
cd "/home/matthalloran8/Vrooli/scripts/resources/tests/single/ai"
pwd

echo "=== Running ollama test directly ==="
timeout 60 bash ollama.test.sh