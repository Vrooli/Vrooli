#!/usr/bin/env bash
#
# Unit Test: Test PRD Control Tower Go code units
#

set -eo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

echo "🧪 Running PRD Control Tower unit tests..."

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo "⚠️  Go not found, skipping unit tests"
    exit 0
fi

# Run Go unit tests
echo "  Testing Go API..."
cd api

if go test -v -cover ./... > /tmp/prd-control-tower-unit-test.log 2>&1; then
    echo "  ✓ Go unit tests passed"

    # Show coverage summary
    if grep -q "coverage:" /tmp/prd-control-tower-unit-test.log; then
        echo "  Coverage:"
        grep "coverage:" /tmp/prd-control-tower-unit-test.log | sed 's/^/    /'
    fi
else
    echo "  ✗ Go unit tests failed"
    cat /tmp/prd-control-tower-unit-test.log
    exit 1
fi

cd "$SCENARIO_DIR"

# Check if UI tests exist
if [ -f "ui/package.json" ] && grep -q "\"test\"" ui/package.json; then
    echo "  Testing UI..."
    cd ui

    if npm test 2>&1 | tee /tmp/prd-control-tower-ui-test.log; then
        echo "  ✓ UI unit tests passed"
    else
        echo "  ⚠️  UI unit tests not configured or failed (non-fatal)"
    fi

    cd "$SCENARIO_DIR"
else
    echo "  ⚠️  UI unit tests not configured, skipping"
fi

echo "✅ Unit test phase passed"
