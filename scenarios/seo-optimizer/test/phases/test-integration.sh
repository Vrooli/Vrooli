#!/bin/bash
# Integration tests for seo-optimizer scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running SEO Optimizer integration tests..."

# Check if API is running
if ! curl -s http://localhost:${API_PORT:-8080}/health &> /dev/null; then
    echo "⚠️  API not running, integration tests skipped"
    testing::phase::end_with_summary "Integration tests skipped (API not running)"
    exit 0
fi

# Test health endpoint
echo "Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s http://localhost:${API_PORT:-8080}/health)
if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    echo "✓ Health check passed"
else
    echo "✗ Health check failed"
    exit 1
fi

# Test SEO audit endpoint
echo "Testing SEO audit endpoint..."
AUDIT_RESPONSE=$(curl -s -X POST http://localhost:${API_PORT:-8080}/api/seo-audit \
    -H "Content-Type: application/json" \
    -d '{"url":"https://example.com","depth":2}')

if echo "$AUDIT_RESPONSE" | grep -q "url"; then
    echo "✓ SEO audit endpoint working"
else
    echo "✗ SEO audit endpoint failed"
    exit 1
fi

# Test content optimization endpoint
echo "Testing content optimization endpoint..."
OPTIMIZE_RESPONSE=$(curl -s -X POST http://localhost:${API_PORT:-8080}/api/content-optimize \
    -H "Content-Type: application/json" \
    -d '{"content":"Test content for SEO optimization","target_keywords":"SEO, optimization"}')

if echo "$OPTIMIZE_RESPONSE" | grep -q "content_analysis"; then
    echo "✓ Content optimization endpoint working"
else
    echo "✗ Content optimization endpoint failed"
    exit 1
fi

# Test keyword research endpoint
echo "Testing keyword research endpoint..."
KEYWORD_RESPONSE=$(curl -s -X POST http://localhost:${API_PORT:-8080}/api/keyword-research \
    -H "Content-Type: application/json" \
    -d '{"seed_keyword":"digital marketing"}')

if echo "$KEYWORD_RESPONSE" | grep -q "keywords"; then
    echo "✓ Keyword research endpoint working"
else
    echo "✗ Keyword research endpoint failed"
    exit 1
fi

# Test competitor analysis endpoint
echo "Testing competitor analysis endpoint..."
COMPETITOR_RESPONSE=$(curl -s -X POST http://localhost:${API_PORT:-8080}/api/competitor-analysis \
    -H "Content-Type: application/json" \
    -d '{"your_url":"https://example.com","competitor_url":"https://example.org"}')

if echo "$COMPETITOR_RESPONSE" | grep -q "comparison"; then
    echo "✓ Competitor analysis endpoint working"
else
    echo "✗ Competitor analysis endpoint failed"
    exit 1
fi

testing::phase::end_with_summary "Integration tests completed"
