#!/bin/bash

set -euo pipefail

echo "=== Direct Execution Test ==="

# Create environment file exactly like the framework
test_env_file="/tmp/vrooli_test_env_direct_$$"
cat > "$test_env_file" << EOF
export TEST_ID="direct_test_$(date +%s)"
export TEST_TIMEOUT="30"
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
export HEALTHY_RESOURCES_STR="ollama"
EOF

echo 'HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)' >> "$test_env_file"

echo "Environment file created:"
cat "$test_env_file"
echo "=== End of environment file ==="

# Test execution
test_file="/home/matthalloran8/Vrooli/scripts/resources/tests/single/ai/ollama.test.sh"
test_log_file="/tmp/vrooli_test_direct_$$.log"

echo "About to run test in subshell..."
echo "Test file: $test_file"
echo "Log file: $test_log_file"

# Run exactly like the framework
(
    echo "=== SUBSHELL START ===" >&2
    source "$test_env_file"
    echo "Environment sourced successfully" >&2
    echo "HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]}" >&2
    cd "$(dirname "$test_file")"
    echo "Changed directory to: $(pwd)" >&2
    echo "About to run timeout command..." >&2
    timeout 30 bash "$(basename "$test_file")"
    echo "Test execution completed" >&2
) > "$test_log_file" 2>&1
exit_code=$?

echo "Subshell completed with exit code: $exit_code"
echo "Log file contents:"
cat "$test_log_file"

# Cleanup
rm -f "$test_env_file" "$test_log_file"