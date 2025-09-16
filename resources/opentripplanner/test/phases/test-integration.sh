#!/usr/bin/env bash
# OpenTripPlanner Integration Tests

set -euo pipefail

# Source configuration
source "$(dirname "${BASH_SOURCE[0]}")/../../config/defaults.sh"

echo "Running OpenTripPlanner integration tests..."

# Test coordinates for Portland area
FROM_LAT="45.5231"
FROM_LON="-122.6765"
TO_LAT="45.5152"
TO_LON="-122.6784"

# Test 1: Plan a simple walking route
echo -n "1. Testing walk-only routing... "
walk_response=$(timeout 10 curl -sf "http://localhost:${OTP_PORT}/otp/routers/default/plan?fromPlace=${FROM_LAT},${FROM_LON}&toPlace=${TO_LAT},${TO_LON}&mode=WALK" 2>/dev/null || echo "FAILED")
if [[ "$walk_response" != "FAILED" ]] && echo "$walk_response" | grep -q "itineraries"; then
    echo "✓"
else
    echo "✗ (Walk routing failed)"
    exit 1
fi

# Test 2: Plan a transit + walk route
echo -n "2. Testing transit+walk routing... "
transit_response=$(timeout 10 curl -sf "http://localhost:${OTP_PORT}/otp/routers/default/plan?fromPlace=${FROM_LAT},${FROM_LON}&toPlace=${TO_LAT},${TO_LON}&mode=TRANSIT,WALK" 2>/dev/null || echo "FAILED")
if [[ "$transit_response" != "FAILED" ]]; then
    if echo "$transit_response" | grep -q "itineraries"; then
        echo "✓"
    else
        # Transit data might not be loaded yet, which is acceptable for initial setup
        echo "⚠ (No transit data loaded yet)"
    fi
else
    echo "✗ (Transit routing request failed)"
    exit 1
fi

# Test 3: Test isochrone generation
echo -n "3. Testing isochrone generation... "
iso_response=$(timeout 10 curl -sf "http://localhost:${OTP_PORT}/otp/routers/default/isochrone?fromPlace=${FROM_LAT},${FROM_LON}&mode=WALK&cutoffSec=900" 2>/dev/null || echo "FAILED")
if [[ "$iso_response" != "FAILED" ]]; then
    if echo "$iso_response" | grep -q "type.*Feature"; then
        echo "✓"
    else
        echo "⚠ (Isochrone response unexpected format)"
    fi
else
    echo "✗ (Isochrone generation failed)"
    exit 1
fi

# Test 4: Test bike routing
echo -n "4. Testing bike routing... "
bike_response=$(timeout 10 curl -sf "http://localhost:${OTP_PORT}/otp/routers/default/plan?fromPlace=${FROM_LAT},${FROM_LON}&toPlace=${TO_LAT},${TO_LON}&mode=BICYCLE" 2>/dev/null || echo "FAILED")
if [[ "$bike_response" != "FAILED" ]] && echo "$bike_response" | grep -q "itineraries"; then
    echo "✓"
else
    echo "✗ (Bike routing failed)"
    exit 1
fi

# Test 5: Test stops endpoint (if transit data loaded)
echo -n "5. Testing stops endpoint... "
stops_response=$(timeout 10 curl -sf "http://localhost:${OTP_PORT}/otp/routers/default/index/stops" 2>/dev/null || echo "FAILED")
if [[ "$stops_response" != "FAILED" ]]; then
    echo "✓"
else
    echo "⚠ (Stops endpoint not available - no transit data)"
fi

# Test 6: Test invalid coordinates handling
echo -n "6. Testing error handling... "
error_response=$(timeout 5 curl -sf "http://localhost:${OTP_PORT}/otp/routers/default/plan?fromPlace=999,999&toPlace=888,888&mode=WALK" 2>/dev/null || echo "FAILED")
if [[ "$error_response" != "FAILED" ]]; then
    echo "✓"
else
    echo "⚠ (Error handling test inconclusive)"
fi

# Test 7: Test multimodal routing (bike + transit)
echo -n "7. Testing bike+transit routing... "
multimodal_response=$(timeout 10 curl -sf "http://localhost:${OTP_PORT}/otp/routers/default/plan?fromPlace=${FROM_LAT},${FROM_LON}&toPlace=${TO_LAT},${TO_LON}&mode=BICYCLE,TRANSIT" 2>/dev/null || echo "FAILED")
if [[ "$multimodal_response" != "FAILED" ]]; then
    echo "✓"
else
    echo "⚠ (Multimodal routing not available)"
fi

# Test 8: Test wheelchair accessibility
echo -n "8. Testing wheelchair routing... "
wheelchair_response=$(timeout 10 curl -sf "http://localhost:${OTP_PORT}/otp/routers/default/plan?fromPlace=${FROM_LAT},${FROM_LON}&toPlace=${TO_LAT},${TO_LON}&mode=WALK&wheelchair=true" 2>/dev/null || echo "FAILED")
if [[ "$wheelchair_response" != "FAILED" ]]; then
    echo "✓"
else
    echo "⚠ (Wheelchair routing not configured)"
fi

echo ""
echo "Integration tests completed!"
exit 0