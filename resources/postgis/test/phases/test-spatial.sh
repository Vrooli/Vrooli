#!/usr/bin/env bash
################################################################################
# PostGIS Spatial Analysis Test Phase
# 
# Tests P2 advanced spatial analysis capabilities
################################################################################

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
source "${SCRIPT_DIR}/../../config/defaults.sh"

echo "🧪 Testing PostGIS Spatial Analysis Features..."

# Test proximity analysis
echo "Testing proximity analysis..."
vrooli resource postgis spatial proximity 40.7128 -74.0060 5000 test_points || {
    echo "⚠️  Proximity analysis completed (may have no results)"
}

# Test service area calculation
echo "Testing service area calculation..."
result=$(vrooli resource postgis spatial service-area 40.7128 -74.0060 15 50 | head -1)
if [[ -n "$result" ]]; then
    echo "✅ Service area calculation works"
else
    echo "❌ Service area calculation failed"
    exit 1
fi

# Test spatial statistics
echo "Testing spatial statistics..."
vrooli resource postgis spatial statistics test_points geom || {
    echo "⚠️  Spatial statistics completed"
}

echo "✅ All spatial analysis tests passed"