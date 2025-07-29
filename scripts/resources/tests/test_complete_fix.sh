#\!/bin/bash
# Test the complete fix - simulate the framework behavior
SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"

# Create test environment file (like runner.sh does)
test_env_file="/tmp/test_complete_fix_env_$$"
cat > "$test_env_file" << ENVEOF
export TEST_ID="test_complete_fix"
export TEST_TIMEOUT="30"
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="$SCRIPT_DIR"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
# Export healthy resources as a space-separated string
export HEALTHY_RESOURCES_STR="ollama whisper browserless"
# Note: Individual tests will use HEALTHY_RESOURCES_STR directly via require_resource()
ENVEOF

echo "=== Environment file contents ==="
cat "$test_env_file"

# Create minimal test script that mimics Ollama test behavior
test_script="/tmp/test_complete_fix_script_$$"
cat > "$test_script" << TESTEOF
#\!/bin/bash
set -euo pipefail

# Source framework helpers (like real tests do)
source "\$SCRIPT_DIR/framework/helpers/assertions.sh"

echo "=== Testing require_resource function ==="
echo "HEALTHY_RESOURCES_STR: \${HEALTHY_RESOURCES_STR:-unset}"

# Test requiring ollama (should work)
echo "Testing ollama..."
require_resource "ollama"

# Test requiring unknown resource (should skip)
echo "Testing unknown resource..."
require_resource "unknown" || echo "Correctly skipped unknown resource"

echo "✓ All tests completed successfully"
TESTEOF
chmod +x "$test_script"

echo -e "\n=== Testing complete fix ==="
log_file="/tmp/test_complete_fix_log_$$"
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
