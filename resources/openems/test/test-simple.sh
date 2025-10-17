#!/bin/bash
set -e

echo "Simple OpenEMS Test"
echo "=================="

RESOURCE_DIR="/home/matthalloran8/Vrooli/resources/openems"

# Test 1: CLI exists
if [[ -x "$RESOURCE_DIR/cli.sh" ]]; then
    echo "✅ CLI exists"
else
    echo "❌ CLI missing"
fi

# Test 2: Directories exist
if [[ -d "$RESOURCE_DIR/lib" && -d "$RESOURCE_DIR/config" ]]; then
    echo "✅ Directories exist"
else
    echo "❌ Directories missing"
fi

# Test 3: Config files exist
if [[ -f "$RESOURCE_DIR/config/defaults.sh" && -f "$RESOURCE_DIR/config/runtime.json" ]]; then
    echo "✅ Config files exist"
else
    echo "❌ Config files missing"
fi

# Test 4: Help command works
if $RESOURCE_DIR/cli.sh help >/dev/null 2>&1; then
    echo "✅ Help command works"
else
    echo "❌ Help command failed"
fi

echo ""
echo "✅ Simple test complete"