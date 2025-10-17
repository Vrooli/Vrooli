#!/usr/bin/env bash
# Mesa Unit Tests
# Library function validation per v2.0 contract (<60s)

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly TEST_DIR

# Source libraries for testing
source "${TEST_DIR}/lib/core.sh"
source "${TEST_DIR}/lib/test.sh"

echo "=== Mesa Unit Tests ==="
echo "Testing library functions..."

# Test 1: Port configuration
echo -n "1. Testing port configuration... "
if [[ "$MESA_PORT" == "9512" ]]; then
    echo "✓"
else
    echo "✗ (Expected 9512, got $MESA_PORT)"
    exit 1
fi

# Test 2: Directory structure
echo -n "2. Testing directory structure... "
required_dirs=("lib" "config" "test" "examples" "templates")
all_exist=true

for dir in "${required_dirs[@]}"; do
    if [[ ! -d "${TEST_DIR}/$dir" ]]; then
        all_exist=false
        break
    fi
done

if $all_exist; then
    echo "✓"
else
    echo "✗ (Missing required directories)"
    exit 1
fi

# Test 3: Configuration files
echo -n "3. Testing configuration files... "
required_files=(
    "config/runtime.json"
    "config/defaults.sh"
    "config/schema.json"
    "cli.sh"
    "lib/core.sh"
    "lib/test.sh"
)

all_exist=true
for file in "${required_files[@]}"; do
    if [[ ! -f "${TEST_DIR}/$file" ]]; then
        all_exist=false
        echo "✗ (Missing: $file)"
        exit 1
    fi
done

if $all_exist; then
    echo "✓"
fi

# Test 4: Runtime configuration validation
echo -n "4. Testing runtime.json structure... "
if jq -e '.startup_order and .dependencies and .startup_timeout' \
    "${TEST_DIR}/config/runtime.json" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗ (Invalid runtime.json)"
    exit 1
fi

# Test 5: Info command output
echo -n "5. Testing info command... "
info_output=$(mesa::show_info 2>&1)
if echo "$info_output" | grep -q "Startup Order"; then
    echo "✓"
else
    echo "✗ (Info command failed)"
    exit 1
fi

# Test 6: JSON info output
echo -n "6. Testing JSON info output... "
json_output=$(mesa::show_info --json 2>&1)
if echo "$json_output" | jq -e '.startup_order' > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗ (JSON info failed)"
    exit 1
fi

# Test 7: Status function
echo -n "7. Testing status function... "
status_output=$(mesa::status 2>&1)
if echo "$status_output" | grep -qE "Mesa Status: (Running|Stopped)"; then
    echo "✓"
else
    echo "✗ (Status function failed)"
    exit 1
fi

# Test 8: Content list function
echo -n "8. Testing content list... "
list_output=$(mesa::content_list 2>&1)
if echo "$list_output" | grep -q "Built-in Models"; then
    echo "✓"
else
    echo "✗ (Content list failed)"
    exit 1
fi

# Test 9: Schema validation
echo -n "9. Testing schema.json validity... "
if jq -e '."$schema"' "${TEST_DIR}/config/schema.json" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗ (Invalid schema.json)"
    exit 1
fi

# Test 10: Exit codes
echo -n "10. Testing exit codes... "
set +e
mesa::show_help > /dev/null 2>&1
help_exit=$?
set -e

if [[ $help_exit -eq 0 ]]; then
    echo "✓"
else
    echo "✗ (Incorrect exit code)"
    exit 1
fi

echo ""
echo "=== Unit Tests Passed ==="
echo "All library functions validated!"
exit 0