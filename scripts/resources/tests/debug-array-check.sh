#!/bin/bash

echo "=== Array Variable Debug ==="

# Set up the array as the framework does
export HEALTHY_RESOURCES_STR="ollama"
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)

echo "HEALTHY_RESOURCES_STR: '$HEALTHY_RESOURCES_STR'"
echo "HEALTHY_RESOURCES array: '${HEALTHY_RESOURCES[*]}'"
echo "HEALTHY_RESOURCES first element: '${HEALTHY_RESOURCES[0]}'"
echo "HEALTHY_RESOURCES as string: '${HEALTHY_RESOURCES:-}'"
echo "HEALTHY_RESOURCES length: ${#HEALTHY_RESOURCES[@]}"

echo ""
echo "=== Testing require_resource logic ==="

# Test the condition from require_resource
if [[ -n "${HEALTHY_RESOURCES:-}" ]]; then
    echo "✅ HEALTHY_RESOURCES is considered set"
    echo "Value being checked: '${HEALTHY_RESOURCES:-}'"
    
    echo "Looping through array:"
    for healthy_resource in ${HEALTHY_RESOURCES[@]}; do
        echo "  - '$healthy_resource'"
        if [[ "$healthy_resource" == "ollama" ]]; then
            echo "  ✅ Found ollama!"
        fi
    done
else
    echo "❌ HEALTHY_RESOURCES is considered unset"
fi