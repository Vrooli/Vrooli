#!/bin/bash
# Debug script to test execution

echo "DEBUG: Starting debug test"
export HEALTHY_RESOURCES_STR="ollama"
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)

echo "DEBUG: HEALTHY_RESOURCES array: ${HEALTHY_RESOURCES[*]}"
echo "DEBUG: Array length: ${#HEALTHY_RESOURCES[@]}"

# Test require_resource function
source framework/helpers/assertions.sh

echo "DEBUG: Testing require_resource function"
require_resource "ollama"
echo "DEBUG: require_resource completed with exit code: $?"

echo "DEBUG: Test completed"