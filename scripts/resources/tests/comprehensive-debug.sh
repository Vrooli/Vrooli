#!/bin/bash

set -euo pipefail

echo "=== Comprehensive Test Debug ==="

# Set up environment exactly as framework
export TEST_ID="debug_$(date +%s)"
export TEST_TIMEOUT="60"
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
export HEALTHY_RESOURCES_STR="ollama"
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)

echo "=== Environment Check ==="
echo "SCRIPT_DIR: $SCRIPT_DIR"
echo "HEALTHY_RESOURCES_STR: $HEALTHY_RESOURCES_STR"
echo "HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"
echo "Array length: ${#HEALTHY_RESOURCES[@]}"

echo ""
echo "=== Testing assertions.sh sourcing ==="
if [[ -f "$SCRIPT_DIR/framework/helpers/assertions.sh" ]]; then
    echo "✅ assertions.sh file exists"
    source "$SCRIPT_DIR/framework/helpers/assertions.sh"
    echo "✅ assertions.sh sourced successfully"
    
    echo "=== Testing require_resource directly ==="
    require_resource "ollama"
    echo "✅ require_resource worked!"
else
    echo "❌ assertions.sh file not found"
fi

echo ""
echo "=== Testing in subshell (like framework does) ==="
(
    echo "In subshell:"
    echo "  HEALTHY_RESOURCES_STR: ${HEALTHY_RESOURCES_STR:-NOT_SET}"
    echo "  HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]:-NOT_SET}"
    
    source "$SCRIPT_DIR/framework/helpers/assertions.sh"
    echo "  About to call require_resource..."
    require_resource "ollama"
    echo "  ✅ require_resource succeeded in subshell!"
)

echo ""
echo "=== Testing ollama test file directly ==="
cd "/home/matthalloran8/Vrooli/scripts/resources/tests/single/ai"

# Run just the setup part
echo "Testing setup_test function:"
(
    source "$SCRIPT_DIR/framework/helpers/assertions.sh"
    source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
    
    # Mock the functions that might not exist
    register_cleanup_handler() { echo "Mock cleanup handler registered"; }
    require_tools() { echo "Mock tools check: $*"; }
    
    # Call setup_test
    TEST_RESOURCE="ollama"
    setup_test() {
        echo "🔧 Setting up Ollama integration test..."
        register_cleanup_handler
        require_resource "$TEST_RESOURCE"
        require_tools "curl" "jq"
        echo "✓ Test setup complete"
    }
    
    setup_test
)

echo "=== Debug Complete ==="