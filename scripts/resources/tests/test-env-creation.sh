#!/bin/bash

set -euo pipefail

echo "=== Testing Environment Creation ==="

# Simulate the framework setup
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VERBOSE=true
TEST_TIMEOUT=30
CLEANUP=true
HEALTHY_RESOURCES=("ollama")
test_identifier="test_ollama"

echo "Original HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}"

# Create test-specific environment exactly like the framework
local_test_env_file="/tmp/vrooli_test_env_$$"
cat > "$local_test_env_file" << EOF
export TEST_ID="${test_identifier}_$(date +%s)"
export TEST_TIMEOUT="$TEST_TIMEOUT"
export TEST_VERBOSE="$VERBOSE"
export TEST_CLEANUP="$CLEANUP"
export SCRIPT_DIR="$SCRIPT_DIR"
export RESOURCES_DIR="$RESOURCES_DIR"
# Export healthy resources as a space-separated string
export HEALTHY_RESOURCES_STR="${HEALTHY_RESOURCES[*]}"
EOF

# Add array recreation to the environment file
echo "# Recreate the array from the string" >> "$local_test_env_file"
echo 'HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)' >> "$local_test_env_file"

echo "=== Environment file created ==="
echo "Content:"
cat "$local_test_env_file"
echo "=== End of file ==="

# Test the environment in a subshell
echo "=== Testing in subshell ==="
(
    source "$local_test_env_file"
    echo "In subshell:"
    echo "  HEALTHY_RESOURCES_STR: $HEALTHY_RESOURCES_STR"
    echo "  HEALTHY_RESOURCES array: ${HEALTHY_RESOURCES[*]}"
    echo "  Array length: ${#HEALTHY_RESOURCES[@]}"
    
    # Test require_resource function
    source "$SCRIPT_DIR/framework/helpers/assertions.sh"
    echo "  Testing require_resource..."
    require_resource "ollama"
    echo "  ✅ require_resource succeeded!"
)

echo "=== Test completed ==="

# Cleanup
rm -f "$local_test_env_file"