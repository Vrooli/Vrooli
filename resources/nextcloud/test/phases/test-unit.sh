#!/usr/bin/env bash
# Nextcloud Unit Tests - Library function validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Running Nextcloud unit tests..."

# Test 1: Configuration loading
echo -n "  Testing configuration loading... "
if [[ -n "${NEXTCLOUD_PORT}" ]] && [[ "${NEXTCLOUD_PORT}" -eq 8086 ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Configuration not loaded correctly"
    exit 1
fi

# Test 2: Runtime.json exists and is valid
echo -n "  Testing runtime.json validity... "
if jq -e . "${RESOURCE_DIR}/config/runtime.json" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: runtime.json is not valid JSON"
    exit 1
fi

# Test 3: Schema.json exists and is valid
echo -n "  Testing schema.json validity... "
if jq -e . "${RESOURCE_DIR}/config/schema.json" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: schema.json is not valid JSON"
    exit 1
fi

# Test 4: CLI script is executable
echo -n "  Testing CLI script permissions... "
if [[ -x "${RESOURCE_DIR}/cli.sh" ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: cli.sh is not executable"
    exit 1
fi

# Test 5: Help command works
echo -n "  Testing help command... "
if "${RESOURCE_DIR}/cli.sh" help | grep -q "Nextcloud Resource Management"; then
    echo "✓"
else
    echo "✗"
    echo "Error: Help command not working"
    exit 1
fi

# Test 6: Info command returns valid JSON
echo -n "  Testing info command... "
if "${RESOURCE_DIR}/cli.sh" info --json | jq -e . > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: Info command not returning valid JSON"
    exit 1
fi

echo "All unit tests passed!"
exit 0
