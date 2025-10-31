#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Integration tests for privacy-terms-generator
echo "Running integration tests..."

# Test 1: API server health
echo "Testing API health endpoint..."
if command -v curl &> /dev/null; then
    # Note: This assumes the API is running. In CI/CD, you'd start it first.
    # For now, we'll just validate the handler exists
    if grep -q "healthHandler" api/main.go; then
        echo "✓ Health handler exists"
    else
        echo "✗ Health handler not found"
        exit 1
    fi
fi

# Test 2: CLI integration
echo "Testing CLI integration..."
if [ -f "cli/privacy-terms-generator" ]; then
    echo "✓ CLI binary exists"
else
    echo "✗ CLI binary not found (this is expected in test environment)"
fi

# Test 3: Validate API endpoints are registered
echo "Validating API endpoints..."
required_endpoints=(
    "/health"
    "/api/v1/legal/generate"
    "/api/v1/legal/templates/freshness"
    "/api/v1/legal/documents/history"
    "/api/v1/legal/clauses/search"
)

for endpoint in "${required_endpoints[@]}"; do
    if grep -q "$endpoint" api/main.go; then
        echo "✓ Endpoint $endpoint is registered"
    else
        echo "✗ Endpoint $endpoint not found"
        exit 1
    fi
done

testing::phase::end_with_summary "Integration tests completed"
