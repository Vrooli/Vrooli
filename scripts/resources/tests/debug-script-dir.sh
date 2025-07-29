#!/bin/bash

set -euo pipefail

echo "=== Debug SCRIPT_DIR Issue ==="

# Set up environment
export TEST_ID="debug_$(date +%s)"
export TEST_TIMEOUT="60" 
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
export HEALTHY_RESOURCES_STR="ollama"
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)

echo "Initial SCRIPT_DIR: '$SCRIPT_DIR'"

# Simulate what the test file does
cd "/home/matthalloran8/Vrooli/scripts/resources/tests/single/ai"
echo "Changed to directory: $(pwd)"

# Test what the SCRIPT_DIR calculation would be from the test file location
NEW_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
echo "Calculated SCRIPT_DIR from test location: '$NEW_SCRIPT_DIR'"

# Test the condition from the test file
CALCULATED_SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
echo "SCRIPT_DIR after ollama test calculation: '$CALCULATED_SCRIPT_DIR'"

# Check if files exist at both locations
echo ""
echo "=== File existence check ==="
echo "Original assertions.sh: $SCRIPT_DIR/framework/helpers/assertions.sh"
if [[ -f "$SCRIPT_DIR/framework/helpers/assertions.sh" ]]; then
    echo "✅ Original assertions.sh exists"
else
    echo "❌ Original assertions.sh missing"
fi

echo "Calculated assertions.sh: $CALCULATED_SCRIPT_DIR/framework/helpers/assertions.sh"
if [[ -f "$CALCULATED_SCRIPT_DIR/framework/helpers/assertions.sh" ]]; then
    echo "✅ Calculated assertions.sh exists"
else
    echo "❌ Calculated assertions.sh missing"
fi

# Test if they're different
if [[ "$SCRIPT_DIR" != "$CALCULATED_SCRIPT_DIR" ]]; then
    echo "⚠️  SCRIPT_DIR values are different!"
    echo "  Original: $SCRIPT_DIR"
    echo "  Calculated: $CALCULATED_SCRIPT_DIR"
fi