#!/bin/bash

set -e

echo "=== Test Integration Phase ==="

# Placeholder for integration tests
# Example: Test full pipeline from input to output

../cli/qr-code-generator "integration test" test-integration.png
if [ -f "test-integration.png" ]; then
    rm test-integration.png
    echo "✓ Integration test passed"
else
    echo "✗ Integration test failed"
    exit 1
fi
