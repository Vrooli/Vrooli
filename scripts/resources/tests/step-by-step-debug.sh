#!/bin/bash

set -euo pipefail

echo "=== Step-by-Step Debug of Ollama Test ==="

# Initial setup (like the framework does)
export TEST_ID="debug_$(date +%s)"
export TEST_TIMEOUT="60"
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
export HEALTHY_RESOURCES_STR="ollama"
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)

echo "Step 1: Initial environment"
echo "  HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"

# Change directory like the framework does
cd "/home/matthalloran8/Vrooli/scripts/resources/tests/single/ai"
echo "Step 2: Changed directory to $(pwd)"
echo "  HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"

# Reproduce the test file's initial setup
echo "Step 3: Setting test metadata (like ollama.test.sh does)"
TEST_RESOURCE="ollama"
TEST_TIMEOUT="${TEST_TIMEOUT:-60}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"
echo "  HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"

# Reproduce SCRIPT_DIR calculation
echo "Step 4: SCRIPT_DIR calculation"
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
echo "  SCRIPT_DIR: $SCRIPT_DIR"
echo "  HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"

# Source assertions.sh (like test does)
echo "Step 5: Sourcing assertions.sh"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
echo "  HEALTHY_RESOURCES after sourcing assertions.sh: ${HEALTHY_RESOURCES[*]}"

# Source cleanup.sh (like test does)
echo "Step 6: Sourcing cleanup.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
echo "  HEALTHY_RESOURCES after sourcing cleanup.sh: ${HEALTHY_RESOURCES[*]}"

# Test require_resource at this point
echo "Step 7: Testing require_resource"
require_resource "$TEST_RESOURCE"

echo "✅ All steps completed successfully!"