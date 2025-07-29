#\!/bin/bash
# Create test environment file
test_env_file="/tmp/debug_timeout_env_$$"
cat > "$test_env_file" << ENVEOF
export TEST_ID="debug_test"
export HEALTHY_RESOURCES_STR="ollama whisper"  
HEALTHY_RESOURCES=(\$HEALTHY_RESOURCES_STR)
ENVEOF

# Create a minimal test script
test_script="/tmp/debug_test_script_$$"
cat > "$test_script" << TESTEOF
#\!/bin/bash
echo "=== Inside test script ==="
echo "HEALTHY_RESOURCES_STR: \${HEALTHY_RESOURCES_STR:-unset}"
echo "HEALTHY_RESOURCES array: \${HEALTHY_RESOURCES[*]:-unset}"
echo "Array length: \${#HEALTHY_RESOURCES[@]}"

# Test the require_resource function path
if [[ -n "\${HEALTHY_RESOURCES:-}" ]]; then
    echo "HEALTHY_RESOURCES variable exists"
    for resource in \${HEALTHY_RESOURCES[@]}; do
        echo "  Found resource: \$resource"
    done
else
    echo "HEALTHY_RESOURCES variable is empty or unset"
fi
TESTEOF
chmod +x "$test_script"

echo "=== Testing with timeout (like framework does) ==="
log_file="/tmp/debug_timeout_log_$$"
(
    source "$test_env_file"
    timeout 30 bash "$test_script"
) > "$log_file" 2>&1
exit_code=$?

echo "Exit code: $exit_code"
echo "Log contents:"
cat "$log_file"

# Cleanup
rm -f "$test_env_file" "$test_script" "$log_file"
