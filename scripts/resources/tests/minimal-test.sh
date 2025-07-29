#!/bin/bash

echo "=== Minimal Test Debug ==="
echo "Environment variables:"
echo "  HEALTHY_RESOURCES_STR: ${HEALTHY_RESOURCES_STR:-unset}"
echo "  SCRIPT_DIR: ${SCRIPT_DIR:-unset}"

if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
    echo "  HEALTHY_RESOURCES array: ${HEALTHY_RESOURCES[*]}"
    echo "  Array length: ${#HEALTHY_RESOURCES[@]}"
else
    echo "  HEALTHY_RESOURCES_STR not set!"
fi

echo "Sourcing assertions..."
source "$SCRIPT_DIR/framework/helpers/assertions.sh"

echo "Testing require_resource function..."
require_resource "ollama"
echo "require_resource completed successfully"

echo "=== Test Complete ==="