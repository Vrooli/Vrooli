#!/bin/bash

set -e

echo "=== Test Business Phase ==="

# Test business logic: QR code generation workflow
echo "Testing QR generation workflow..."
qr-generator generate "Business test QR" --output /tmp/business-test.png --size 512

if [ -f "/tmp/business-test.png" ]; then
    rm /tmp/business-test.png
    echo "✓ Business workflow test passed"
else
    echo "✗ Business workflow test failed"
    exit 1
fi

echo "✓ Business tests completed"
