#!/usr/bin/env bash
# QwenCoder Unit Tests - Library function validation (<60s)

set -euo pipefail

# Setup
readonly PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly TEST_DIR="$(dirname "${PHASES_DIR}")"
readonly RESOURCE_DIR="$(dirname "${TEST_DIR}")"

# Source libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

echo "Running QwenCoder unit tests..."

# Test counter
tests_passed=0
tests_failed=0

# Test 1: qwencoder_get_port function
echo -n "1. Testing qwencoder_get_port... "
port=$(qwencoder_get_port)
if [[ "${port}" =~ ^[0-9]+$ ]] && [[ ${port} -ge 1024 ]] && [[ ${port} -le 65535 ]]; then
    echo " (port: ${port})"
    ((tests_passed++))
else
    echo " (invalid port: ${port})"
    ((tests_failed++))
fi

# Test 2: qwencoder_get_model_path function
echo -n "2. Testing qwencoder_get_model_path... "
model_path=$(qwencoder_get_model_path "qwencoder-1.5b")
if [[ -n "${model_path}" ]] && [[ "${model_path}" == *"models"* ]]; then
    echo ""
    ((tests_passed++))
else
    echo ""
    ((tests_failed++))
fi

# Test 3: qwencoder_validate_model function
echo -n "3. Testing qwencoder_validate_model... "
if qwencoder_validate_model "qwencoder-1.5b"; then
    echo " (valid model)"
    ((tests_passed++))
else
    echo " (validation failed)"
    ((tests_failed++))
fi

# Test 4: qwencoder_get_config function
echo -n "4. Testing qwencoder_get_config... "
if config=$(qwencoder_get_config); then
    if echo "${config}" | jq -e '.port' > /dev/null 2>&1; then
        echo ""
        ((tests_passed++))
    else
        echo " (invalid JSON)"
        ((tests_failed++))
    fi
else
    echo " (function failed)"
    ((tests_failed++))
fi

# Test 5: qwencoder_is_port_available function
echo -n "5. Testing qwencoder_is_port_available... "
# Test with a likely available high port
if qwencoder_is_port_available 54321; then
    echo ""
    ((tests_passed++))
else
    echo ""
    ((tests_failed++))
fi

# Test 6: Model size validation
echo -n "6. Testing model size calculations... "
sizes=("0.5b" "1.5b" "7b" "32b")
size_valid=0
for size in "${sizes[@]}"; do
    if qwencoder_get_model_size_gb "qwencoder-${size}" > /dev/null 2>&1; then
        ((size_valid++))
    fi
done

if [[ ${size_valid} -eq ${#sizes[@]} ]]; then
    echo " (${size_valid}/${#sizes[@]})"
    ((tests_passed++))
else
    echo " (${size_valid}/${#sizes[@]})"
    ((tests_failed++))
fi

# Summary
echo ""
echo "Unit Test Results:"
echo "=================="
echo "Passed: ${tests_passed}"
echo "Failed: ${tests_failed}"

if [[ ${tests_failed} -eq 0 ]]; then
    echo "Status:  All unit tests passed!"
    exit 0
else
    echo "Status:  Some tests failed"
    exit 1
fi