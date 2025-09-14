#!/usr/bin/env bash
################################################################################
# PostGIS Visualization Test Phase
# 
# Tests P2 visualization capabilities
################################################################################

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
source "${SCRIPT_DIR}/../../config/defaults.sh"

echo "🧪 Testing PostGIS Visualization Features..."

# Test GeoJSON generation
echo "Testing GeoJSON generation..."
vrooli resource postgis visualization geojson "SELECT name, geom FROM test_points" /tmp/viz_test.geojson
if [[ -f /tmp/viz_test.geojson ]]; then
    echo "✅ GeoJSON generation works"
else
    echo "❌ GeoJSON generation failed"
    exit 1
fi

# Test viewer generation
echo "Testing HTML viewer generation..."
vrooli resource postgis visualization viewer /tmp/viz_test.geojson /tmp/viz_test.html
if [[ -f /tmp/viz_test.html ]]; then
    echo "✅ HTML viewer generation works"
else
    echo "❌ HTML viewer generation failed"
    exit 1
fi

# Clean up test files
rm -f /tmp/viz_test.geojson /tmp/viz_test.html

echo "✅ All visualization tests passed"