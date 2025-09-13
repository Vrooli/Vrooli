#!/bin/bash
# VirusTotal Unit Test Phase
# Library function and configuration validation

set -euo pipefail

# Configuration
RESOURCE_NAME="virustotal"
MAX_DURATION=60  # v2.0 contract requirement
RESOURCE_DIR="/home/matthalloran8/Vrooli/resources/${RESOURCE_NAME}"

# Start timer
START_TIME=$(date +%s)

echo "Starting unit tests for ${RESOURCE_NAME}..."

# Test 1: CLI script exists and is executable
echo -n "1. Checking CLI script... "
if [[ -f "${RESOURCE_DIR}/cli.sh" ]] && [[ -x "${RESOURCE_DIR}/cli.sh" ]]; then
    echo "PASS"
else
    echo "FAIL: cli.sh not found or not executable"
    exit 1
fi

# Test 2: Core library exists
echo -n "2. Checking core library... "
if [[ -f "${RESOURCE_DIR}/lib/core.sh" ]]; then
    echo "PASS"
else
    echo "FAIL: lib/core.sh not found"
    exit 1
fi

# Test 3: Test library exists
echo -n "3. Checking test library... "
if [[ -f "${RESOURCE_DIR}/lib/test.sh" ]]; then
    echo "PASS"
else
    echo "FAIL: lib/test.sh not found"
    exit 1
fi

# Test 4: Configuration files exist
echo -n "4. Checking configuration files... "
CONFIG_FILES=("defaults.sh" "runtime.json" "schema.json")
ALL_FOUND=true
for file in "${CONFIG_FILES[@]}"; do
    if [[ ! -f "${RESOURCE_DIR}/config/${file}" ]]; then
        echo "FAIL: config/${file} not found"
        ALL_FOUND=false
        break
    fi
done
if [[ "$ALL_FOUND" == true ]]; then
    echo "PASS"
fi

# Test 5: Validate runtime.json structure
echo -n "5. Validating runtime.json... "
if jq -e '.startup_order and .dependencies and .startup_timeout' \
    "${RESOURCE_DIR}/config/runtime.json" >/dev/null 2>&1; then
    echo "PASS"
    STARTUP_ORDER=$(jq -r '.startup_order' "${RESOURCE_DIR}/config/runtime.json")
    echo "   Startup order: $STARTUP_ORDER"
else
    echo "FAIL: Invalid runtime.json structure"
    exit 1
fi

# Test 6: Validate schema.json
echo -n "6. Validating schema.json... "
if jq -e '.type == "object" and .properties' \
    "${RESOURCE_DIR}/config/schema.json" >/dev/null 2>&1; then
    echo "PASS"
else
    echo "FAIL: Invalid schema.json"
    exit 1
fi

# Test 7: Source defaults.sh
echo -n "7. Testing defaults.sh sourcing... "
(
    source "${RESOURCE_DIR}/config/defaults.sh" 2>/dev/null
    if [[ -n "${VIRUSTOTAL_PORT}" ]] && [[ "${VIRUSTOTAL_PORT}" == "8290" ]]; then
        exit 0
    else
        exit 1
    fi
)
if [ $? -eq 0 ]; then
    echo "PASS"
else
    echo "FAIL: Cannot source defaults.sh or invalid defaults"
    exit 1
fi

# Test 8: Test phase scripts exist
echo -n "8. Checking test phase scripts... "
TEST_PHASES=("test-smoke.sh" "test-integration.sh" "test-unit.sh")
ALL_FOUND=true
for phase in "${TEST_PHASES[@]}"; do
    if [[ ! -f "${RESOURCE_DIR}/test/phases/${phase}" ]]; then
        echo "FAIL: test/phases/${phase} not found"
        ALL_FOUND=false
        break
    fi
done
if [[ "$ALL_FOUND" == true ]]; then
    echo "PASS"
fi

# Test 9: Environment variable handling
echo -n "9. Testing environment variable preservation... "
TEST_KEY="test_key_abc123"
(
    export VIRUSTOTAL_API_KEY="$TEST_KEY"
    source "${RESOURCE_DIR}/config/defaults.sh" 2>/dev/null
    if [[ "${VIRUSTOTAL_API_KEY}" == "$TEST_KEY" ]]; then
        exit 0
    else
        exit 1
    fi
)
if [ $? -eq 0 ]; then
    echo "PASS"
else
    echo "FAIL: Environment variables not preserved"
    exit 1
fi

# Test 10: CLI help command
echo -n "10. Testing CLI help command... "
HELP_OUTPUT=$(cd "${RESOURCE_DIR}" && bash cli.sh help 2>&1)
if echo "$HELP_OUTPUT" | grep -q "VirusTotal Resource"; then
    echo "PASS"
else
    echo "FAIL: Help command not working"
    exit 1
fi

# Test 11: CLI info command structure
echo -n "11. Testing CLI info command... "
# Since the service might not be running, we just test that the command exists
if cd "${RESOURCE_DIR}" && bash -c 'source lib/core.sh; type show_info' >/dev/null 2>&1; then
    echo "PASS"
else
    echo "FAIL: Info command not defined"
    exit 1
fi

# Test 12: Required v2.0 commands present
echo -n "12. Checking v2.0 contract commands... "
REQUIRED_COMMANDS=("info" "manage" "test" "content" "status" "logs" "credentials")
SOURCE_CMD="source ${RESOURCE_DIR}/lib/core.sh"
ALL_FOUND=true
for cmd in "${REQUIRED_COMMANDS[@]}"; do
    case $cmd in
        manage|test|content)
            if ! bash -c "$SOURCE_CMD; type handle_${cmd}" >/dev/null 2>&1; then
                echo "FAIL: handle_${cmd} function not found"
                ALL_FOUND=false
                break
            fi
            ;;
        *)
            if ! bash -c "$SOURCE_CMD; type show_${cmd}" >/dev/null 2>&1; then
                echo "FAIL: show_${cmd} function not found"
                ALL_FOUND=false
                break
            fi
            ;;
    esac
done
if [[ "$ALL_FOUND" == true ]]; then
    echo "PASS"
fi

# End timer and check duration
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo "Unit tests completed in ${DURATION} seconds"

if [ $DURATION -gt $MAX_DURATION ]; then
    echo "ERROR: Unit tests exceeded ${MAX_DURATION} second limit"
    exit 1
fi

echo "All unit tests passed successfully!"
exit 0