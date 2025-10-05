#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running business requirement tests..."

# Test API is running
if ! curl -sf http://localhost:${API_PORT}/health &> /dev/null; then
    echo "❌ API is not running. Start with 'vrooli scenario start palette-gen'"
    exit 1
fi

# Business Requirement 1: Generate palettes with different styles
echo "Testing palette generation with multiple styles..."
STYLES=("vibrant" "pastel" "dark" "minimal" "earthy")

for style in "${STYLES[@]}"; do
    echo "  Testing style: $style"
    response=$(curl -sf -X POST http://localhost:${API_PORT}/generate \
        -H 'Content-Type: application/json' \
        -d "{\"theme\":\"ocean\",\"style\":\"$style\",\"num_colors\":5}")

    if ! echo "$response" | jq -e '.success and (.palette | length == 5)' &> /dev/null; then
        echo "  ❌ Failed to generate $style palette"
        exit 1
    fi
    echo "  ✓ $style palette generated successfully"
done

# Business Requirement 2: WCAG accessibility compliance
echo "Testing WCAG accessibility compliance..."
response=$(curl -sf -X POST http://localhost:${API_PORT}/accessibility \
    -H 'Content-Type: application/json' \
    -d '{"foreground":"#000000","background":"#FFFFFF"}')

if ! echo "$response" | jq -e '.wcag_aa and .wcag_aaa' &> /dev/null; then
    echo "❌ Black on white should pass WCAG AA/AAA"
    exit 1
fi

echo "✓ WCAG accessibility checks working correctly"

# Business Requirement 3: Export to multiple formats
echo "Testing export to multiple formats..."
FORMATS=("css" "json" "scss")

for format in "${FORMATS[@]}"; do
    echo "  Testing format: $format"
    response=$(curl -sf -X POST http://localhost:${API_PORT}/export \
        -H 'Content-Type: application/json' \
        -d "{\"palette\":[\"#FF0000\",\"#00FF00\",\"#0000FF\"],\"format\":\"$format\"}")

    if ! echo "$response" | jq -e '.success' &> /dev/null; then
        echo "  ❌ Failed to export to $format"
        exit 1
    fi
    echo "  ✓ Export to $format successful"
done

# Business Requirement 4: Color harmony analysis
echo "Testing color harmony analysis..."
response=$(curl -sf -X POST http://localhost:${API_PORT}/harmony \
    -H 'Content-Type: application/json' \
    -d '{"colors":["#FF0000","#00FF00","#0000FF"]}')

if ! echo "$response" | jq -e '.success and .score' &> /dev/null; then
    echo "❌ Color harmony analysis failed"
    exit 1
fi

echo "✓ Color harmony analysis working"

# Business Requirement 5: Colorblind simulation
echo "Testing colorblind simulation..."
CB_TYPES=("protanopia" "deuteranopia" "tritanopia")

for cb_type in "${CB_TYPES[@]}"; do
    echo "  Testing colorblindness type: $cb_type"
    response=$(curl -sf -X POST http://localhost:${API_PORT}/colorblind \
        -H 'Content-Type: application/json' \
        -d "{\"colors\":[\"#FF0000\",\"#00FF00\"],\"type\":\"$cb_type\"}")

    if ! echo "$response" | jq -e '.success and (.simulated | length == 2)' &> /dev/null; then
        echo "  ❌ Failed to simulate $cb_type"
        exit 1
    fi
    echo "  ✓ $cb_type simulation successful"
done

# Business Requirement 6: Variable palette sizes
echo "Testing variable palette sizes..."
for size in 3 5 7 10; do
    echo "  Testing palette size: $size"
    response=$(curl -sf -X POST http://localhost:${API_PORT}/generate \
        -H 'Content-Type: application/json' \
        -d "{\"theme\":\"sunset\",\"num_colors\":$size}")

    if ! echo "$response" | jq -e ".palette | length == $size" &> /dev/null; then
        echo "  ❌ Failed to generate palette with $size colors"
        exit 1
    fi
    echo "  ✓ Palette with $size colors generated"
done

# Business Requirement 7: Suggestion system
echo "Testing palette suggestion system..."
response=$(curl -sf -X POST http://localhost:${API_PORT}/suggest \
    -H 'Content-Type: application/json' \
    -d '{"use_case":"corporate website"}')

if ! echo "$response" | jq -e '.success and (.suggestions | length > 0)' &> /dev/null; then
    echo "❌ Suggestion system failed"
    exit 1
fi

echo "✓ Suggestion system working"

# Business Requirement 8: History tracking
echo "Testing history tracking..."

# Generate a palette to add to history
curl -sf -X POST http://localhost:${API_PORT}/generate \
    -H 'Content-Type: application/json' \
    -d '{"theme":"tech","style":"dark","num_colors":5}' &> /dev/null

# Retrieve history
response=$(curl -sf http://localhost:${API_PORT}/history)

if ! echo "$response" | jq -e '.success' &> /dev/null; then
    echo "❌ History tracking failed"
    exit 1
fi

echo "✓ History tracking working"

# Business Requirement 9: Caching (if Redis available)
echo "Testing caching behavior..."
response1=$(curl -sf -X POST http://localhost:${API_PORT}/generate \
    -H 'Content-Type: application/json' \
    -d '{"theme":"ocean","style":"vibrant","num_colors":5}')

# Check if cache header is present
if curl -sf -I -X POST http://localhost:${API_PORT}/generate \
    -H 'Content-Type: application/json' \
    -d '{"theme":"ocean","style":"vibrant","num_colors":5}' | grep -q "X-Cache"; then
    echo "✓ Cache headers present"
else
    echo "⚠️  No cache headers (Redis may not be available)"
fi

echo ""
echo "═══════════════════════════════════════════════════════"
echo "  Business Requirements Validation Complete"
echo "═══════════════════════════════════════════════════════"

testing::phase::end_with_summary "Business tests completed"
