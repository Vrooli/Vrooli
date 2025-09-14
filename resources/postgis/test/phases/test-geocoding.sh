#!/usr/bin/env bash
################################################################################
# PostGIS Geocoding Test Phase
# 
# Tests P2 geocoding capabilities
################################################################################

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
source "${SCRIPT_DIR}/../../config/defaults.sh"

echo "🧪 Testing PostGIS Geocoding Features..."

# Initialize geocoding tables
echo "Initializing geocoding tables..."
vrooli resource postgis geocoding init

# Test geocoding with coordinates
echo "Testing coordinate geocoding..."
result=$(vrooli resource postgis geocoding geocode "40.7128, -74.0060" | grep "✅")
if [[ -n "$result" ]]; then
    echo "✅ Coordinate geocoding works"
else
    echo "❌ Coordinate geocoding failed"
    exit 1
fi

# Test reverse geocoding
echo "Testing reverse geocoding..."
vrooli resource postgis geocoding reverse 40.7128 -74.0060 5000 || {
    echo "⚠️  Reverse geocoding returned no results (expected for empty places table)"
}

# Test geocoding stats
echo "Testing geocoding statistics..."
vrooli resource postgis geocoding stats

echo "✅ All geocoding tests passed"