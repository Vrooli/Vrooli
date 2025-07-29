#!/bin/bash
# Simple test to check if the test execution works

set -euo pipefail

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"

echo "Starting simple test"

# Check if we have the environment
echo "HEALTHY_RESOURCES_STR: ${HEALTHY_RESOURCES_STR:-not set}"
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
    echo "HEALTHY_RESOURCES array: ${HEALTHY_RESOURCES[*]}"
fi

# Test require_resource
require_resource "ollama"

echo "Simple test passed"
exit 0