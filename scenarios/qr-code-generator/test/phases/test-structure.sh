#!/bin/bash

set -e

echo "=== Test Structure Phase ==="

# Check required files and directories
if [ -f "../PRD.md" ]; then
    echo "✓ PRD.md present"
else
    echo "✗ Missing PRD.md"
    exit 1
fi

if [ -d "../cli" ] && [ -f "../cli/qr-code-generator" ]; then
    echo "✓ CLI directory and executable present"
else
    echo "✗ Missing CLI"
    exit 1
fi

if [ -f "../test/run-tests.sh" ]; then
    echo "✓ Test runner present"
else
    echo "✗ Missing test runner"
    exit 1
fi

echo "✓ Structure tests passed"
