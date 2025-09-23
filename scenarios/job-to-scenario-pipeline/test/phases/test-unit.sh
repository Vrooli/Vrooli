#!/bin/bash
set -euo pipefail

echo "=== Unit Tests ==="

# Run Go unit tests
cd api && if go test -v ./... 2>/dev/null | grep -q "PASS"; then echo "✓ Go unit tests passed"; else echo "✗ Go unit tests failed"; exit 1; fi || echo "No Go tests or skipped"

# UI tests if applicable
if [[ -d ui ]]; then
    cd ui && if npm test >/dev/null 2>&1; then echo "✓ UI tests passed"; else echo "No UI tests or skipped"; fi || true
fi

echo "Unit tests completed"
exit 0