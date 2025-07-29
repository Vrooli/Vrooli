#!/bin/bash

set -euo pipefail

echo "=== Debug Resource Check ==="

# Set up environment exactly
export TEST_ID="debug_$(date +%s)"
export TEST_TIMEOUT="60"
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
export HEALTHY_RESOURCES_STR="ollama"
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)

echo "Before sourcing assertions.sh:"
echo "  HEALTHY_RESOURCES_STR: '$HEALTHY_RESOURCES_STR'"
echo "  HEALTHY_RESOURCES: '${HEALTHY_RESOURCES[*]}'"
echo "  Array length: ${#HEALTHY_RESOURCES[@]}"

# Source the assertions file
echo "Sourcing assertions.sh..."
source "$SCRIPT_DIR/framework/helpers/assertions.sh"

echo "After sourcing assertions.sh:"
echo "  HEALTHY_RESOURCES_STR: '$HEALTHY_RESOURCES_STR'"
echo "  HEALTHY_RESOURCES: '${HEALTHY_RESOURCES[*]}'"
echo "  Array length: ${#HEALTHY_RESOURCES[@]}"

# Test the require_resource function step by step
echo ""
echo "=== Testing require_resource function step by step ==="

# Manually reproduce the require_resource logic
resource="ollama"
echo "Checking for resource: '$resource'"

# Check if array is set
if [[ -n "${HEALTHY_RESOURCES:-}" ]]; then
    echo "✅ HEALTHY_RESOURCES is set: '${HEALTHY_RESOURCES:-}'"
    
    echo "Iterating through array:"
    for healthy_resource in ${HEALTHY_RESOURCES[@]}; do
        echo "  Checking: '$healthy_resource' against '$resource'"
        if [[ "$healthy_resource" == "$resource" ]]; then
            echo "  ✅ Match found!"
        else
            echo "  ❌ No match"
        fi
    done
else
    echo "❌ HEALTHY_RESOURCES is not set or empty"
fi

echo ""
echo "=== Calling require_resource directly ==="
require_resource "ollama"