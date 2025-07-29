#\!/bin/bash
# Create test environment file
test_env_file="/tmp/debug_env_$$"
cat > "$test_env_file" << ENVEOF
export TEST_ID="debug_test"
export HEALTHY_RESOURCES_STR="ollama whisper"
HEALTHY_RESOURCES=(\$HEALTHY_RESOURCES_STR)
ENVEOF

echo "=== Environment file contents ==="
cat "$test_env_file"

echo -e "\n=== Testing in subshell ==="
(
    source "$test_env_file"
    echo "HEALTHY_RESOURCES_STR: ${HEALTHY_RESOURCES_STR:-unset}"
    echo "HEALTHY_RESOURCES array: ${HEALTHY_RESOURCES[*]:-unset}"
    echo "Array length: ${#HEALTHY_RESOURCES[@]}"
) 2>&1

# Cleanup
rm -f "$test_env_file"
