#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running integration tests..."

# Test API is running
if ! curl -sf http://localhost:${API_PORT}/health &> /dev/null; then
    echo "❌ API is not running. Start with 'vrooli scenario start palette-gen'"
    exit 1
fi

# Test palette generation
echo "Testing palette generation..."
response=$(curl -sf -X POST http://localhost:${API_PORT}/generate \
    -H 'Content-Type: application/json' \
    -d '{"theme":"ocean","style":"vibrant","num_colors":5}')

if ! echo "$response" | jq -e '.success' &> /dev/null; then
    echo "❌ Palette generation failed"
    exit 1
fi

echo "✓ Palette generation working"

# Test accessibility check
echo "Testing accessibility check..."
response=$(curl -sf -X POST http://localhost:${API_PORT}/accessibility \
    -H 'Content-Type: application/json' \
    -d '{"foreground":"#000000","background":"#FFFFFF"}')

if ! echo "$response" | jq -e '.success' &> /dev/null; then
    echo "❌ Accessibility check failed"
    exit 1
fi

echo "✓ Accessibility check working"

# Test harmony analysis
echo "Testing harmony analysis..."
response=$(curl -sf -X POST http://localhost:${API_PORT}/harmony \
    -H 'Content-Type: application/json' \
    -d '{"colors":["#FF0000","#00FF00","#0000FF"]}')

if ! echo "$response" | jq -e '.success' &> /dev/null; then
    echo "❌ Harmony analysis failed"
    exit 1
fi

echo "✓ Harmony analysis working"

# Test colorblind simulation
echo "Testing colorblind simulation..."
response=$(curl -sf -X POST http://localhost:${API_PORT}/colorblind \
    -H 'Content-Type: application/json' \
    -d '{"colors":["#FF0000","#00FF00"],"type":"protanopia"}')

if ! echo "$response" | jq -e '.success' &> /dev/null; then
    echo "❌ Colorblind simulation failed"
    exit 1
fi

echo "✓ Colorblind simulation working"

# Test export
echo "Testing export..."
response=$(curl -sf -X POST http://localhost:${API_PORT}/export \
    -H 'Content-Type: application/json' \
    -d '{"palette":["#FF0000","#00FF00","#0000FF"],"format":"css"}')

if ! echo "$response" | jq -e '.success' &> /dev/null; then
    echo "❌ Export failed"
    exit 1
fi

echo "✓ Export working"

# Test history
echo "Testing history..."
response=$(curl -sf http://localhost:${API_PORT}/history)

if ! echo "$response" | jq -e '.success' &> /dev/null; then
    echo "❌ History failed"
    exit 1
fi

echo "✓ History working"

# Test suggest
echo "Testing suggest..."
response=$(curl -sf -X POST http://localhost:${API_PORT}/suggest \
    -H 'Content-Type: application/json' \
    -d '{"use_case":"corporate website"}')

if ! echo "$response" | jq -e '.success' &> /dev/null; then
    echo "❌ Suggest failed"
    exit 1
fi

echo "✓ Suggest working"

testing::phase::end_with_summary "Integration tests completed"
