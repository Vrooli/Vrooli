#!/bin/bash

set -e

echo "=== Test Business Phase ==="

# Business logic test: Ensure QR generation works for valid business input
../cli/qr-code-generator "Business test input" business-test.png
if [ -f "business-test.png" ]; then
    rm business-test.png
    echo "✓ Business test passed"
else
    echo "✗ Business test failed"
    exit 1
fi
