#!/bin/bash
# Unit tests for Airbyte - Library function validation (<60s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source "${RESOURCE_DIR}/lib/core.sh"

echo "Running Airbyte unit tests..."

# Test 1: Configuration loading
echo -n "  Testing configuration loading... "
source "${RESOURCE_DIR}/config/defaults.sh"
if [[ -n "${AIRBYTE_VERSION}" ]] && \
   [[ -n "${AIRBYTE_WEBAPP_PORT}" ]] && \
   [[ -n "${AIRBYTE_SERVER_PORT}" ]]; then
    echo "OK"
else
    echo "FAILED"
    echo "    Configuration variables not loaded"
    exit 1
fi

# Test 2: Runtime.json validity
echo -n "  Testing runtime.json validity... "
if jq -e '.' "${RESOURCE_DIR}/config/runtime.json" > /dev/null 2>&1; then
    echo "OK"
else
    echo "FAILED"
    echo "    runtime.json is not valid JSON"
    exit 1
fi

# Test 3: Schema.json validity
echo -n "  Testing schema.json validity... "
if jq -e '.' "${RESOURCE_DIR}/config/schema.json" > /dev/null 2>&1; then
    echo "OK"
else
    echo "FAILED"
    echo "    schema.json is not valid JSON"
    exit 1
fi

# Test 4: CLI executable
echo -n "  Testing CLI executable... "
if [[ -x "${RESOURCE_DIR}/cli.sh" ]]; then
    echo "OK"
else
    echo "FAILED"
    echo "    cli.sh is not executable"
    exit 1
fi

# Test 5: Directory structure
echo -n "  Testing directory structure... "
required_dirs=("lib" "config" "test" "test/phases")
missing_dirs=()
for dir in "${required_dirs[@]}"; do
    if [[ ! -d "${RESOURCE_DIR}/${dir}" ]]; then
        missing_dirs+=("$dir")
    fi
done

if [[ ${#missing_dirs[@]} -eq 0 ]]; then
    echo "OK"
else
    echo "FAILED"
    echo "    Missing directories: ${missing_dirs[*]}"
    exit 1
fi

# Test 6: Required files
echo -n "  Testing required files... "
required_files=(
    "cli.sh"
    "lib/core.sh"
    "lib/test.sh"
    "config/defaults.sh"
    "config/runtime.json"
    "config/schema.json"
    "test/run-tests.sh"
    "PRD.md"
    "README.md"
)
missing_files=()
for file in "${required_files[@]}"; do
    if [[ ! -f "${RESOURCE_DIR}/${file}" ]]; then
        missing_files+=("$file")
    fi
done

if [[ ${#missing_files[@]} -eq 0 ]]; then
    echo "OK"
else
    echo "FAILED"
    echo "    Missing files: ${missing_files[*]}"
    exit 1
fi

echo "Unit tests completed successfully"