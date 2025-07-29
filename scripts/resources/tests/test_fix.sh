#\!/bin/bash
# Test the fixed environment propagation
test_env_file="/tmp/test_fix_env_$$"
cat > "$test_env_file" << ENVEOF
export TEST_ID="test_fix"
export HEALTHY_RESOURCES_STR="ollama whisper"
# Recreate the array from the string
HEALTHY_RESOURCES=(\$HEALTHY_RESOURCES_STR)
ENVEOF

echo "=== Environment file contents ==="
cat "$test_env_file"

echo -e "\n=== Testing fixed version ==="
test_script="/tmp/test_fix_script_$$"
cat > "$test_script" << TESTEOF
#\!/bin/bash
echo "HEALTHY_RESOURCES_STR: \${HEALTHY_RESOURCES_STR:-unset}"
echo "HEALTHY_RESOURCES array: \${HEALTHY_RESOURCES[*]:-unset}"
echo "Array length: \${#HEALTHY_RESOURCES[@]}"

if [[ -n "\${HEALTHY_RESOURCES:-}" ]]; then
    echo "✓ HEALTHY_RESOURCES array is properly set"
    for resource in \${HEALTHY_RESOURCES[@]}; do
        echo "  - \$resource"
    done
else
    echo "✗ HEALTHY_RESOURCES array is empty or unset"
fi
TESTEOF
chmod +x "$test_script"

log_file="/tmp/test_fix_log_$$"
(
    source "$test_env_file"
    timeout 30 bash "$test_script"
) > "$log_file" 2>&1
exit_code=$?

echo "Exit code: $exit_code"
echo "Output:"
cat "$log_file"

# Cleanup
rm -f "$test_env_file" "$test_script" "$log_file"
