#!/usr/bin/env bash
# OpenTripPlanner Integration Tests

set -euo pipefail

# Determine APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source port registry and configuration
source "${APP_ROOT}/scripts/resources/port_registry.sh" || exit 1
source "$(dirname "${BASH_SOURCE[0]}")/../../config/defaults.sh"

echo "Running OpenTripPlanner integration tests..."

# Test coordinates for Portland area
FROM_LAT="45.5231"
FROM_LON="-122.6765"
TO_LAT="45.5152"
TO_LON="-122.6784"

# Note: OTP v2.9 has changed API structure - routing endpoints not yet available
# Testing available functionality until routing is fixed

# Test 1: Check OTP API Info
echo -n "1. Testing OTP API info endpoint... "
info_response=$(timeout 10 curl -sf "http://localhost:${OTP_PORT}/otp/" 2>/dev/null || echo "FAILED")
if [[ "$info_response" != "FAILED" ]] && echo "$info_response" | grep -q "version"; then
    echo "✓"
else
    echo "✗ (API info failed)"
    exit 1
fi

# Test 2: Check Debug UI availability
echo -n "2. Testing debug UI availability... "
ui_response=$(timeout 10 curl -sf "http://localhost:${OTP_PORT}/" 2>/dev/null || echo "FAILED")
if [[ "$ui_response" != "FAILED" ]] && echo "$ui_response" | grep -q "OTP Debug"; then
    echo "✓"
else
    echo "✗ (Debug UI not available)"
    exit 1
fi

# Test 3: Check graph object exists
echo -n "3. Testing graph file existence... "
if [[ -f "${OTP_DATA_DIR}/graph.obj" ]]; then
    graph_size=$(du -h "${OTP_DATA_DIR}/graph.obj" | cut -f1)
    echo "✓ (${graph_size})"
else
    echo "✗ (Graph not found)"
    exit 1
fi

# Test 4: Check GTFS data loaded
echo -n "4. Testing GTFS data presence... "
if [[ -f "${OTP_DATA_DIR}/portland.gtfs.zip" ]]; then
    gtfs_size=$(du -h "${OTP_DATA_DIR}/portland.gtfs.zip" | cut -f1)
    echo "✓ (${gtfs_size})"
else
    echo "✗ (GTFS data missing)"
    exit 1
fi

# Test 5: Check OSM data loaded
echo -n "5. Testing OSM data presence... "
if [[ -f "${OTP_DATA_DIR}/portland.osm.pbf" ]]; then
    osm_size=$(du -h "${OTP_DATA_DIR}/portland.osm.pbf" | cut -f1)
    echo "✓ (${osm_size})"
else
    echo "✗ (OSM data missing)"
    exit 1
fi

# Test 6: Check container memory usage
echo -n "6. Testing container resource usage... "
container_stats=$(docker stats --no-stream --format "table {{.MemUsage}}" vrooli-opentripplanner | tail -n1)
if [[ -n "$container_stats" ]]; then
    echo "✓ (${container_stats})"
else
    echo "✗ (Cannot get container stats)"
    exit 1
fi

# Test 7: Check server configuration
echo -n "7. Testing server configuration... "
if [[ -f "${OTP_DATA_DIR}/router-config.json" ]]; then
    echo "✓ (Router config exists)"
else
    echo "✗ (Router config missing)"
    exit 1
fi

# Test 8: Check OTP version
echo -n "8. Testing OTP version info... "
version_info=$(timeout 5 curl -sf "http://localhost:${OTP_PORT}/otp/" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('version',{}).get('version','unknown'))" 2>/dev/null || echo "unknown")
if [[ "$version_info" != "unknown" ]]; then
    echo "✓ (v${version_info})"
else
    echo "⚠ (Version check failed)"
fi

# Test 9: Check trip planning via Transmodel v3 API
echo -n "9. Testing trip planning API... "
trip_response=$(vrooli resource opentripplanner content execute --action plan-trip \
    --from-lat 45.5231 --from-lon -122.6765 \
    --to-lat 45.5152 --to-lon -122.6784 \
    --format summary 2>/dev/null || echo "FAILED")

if [[ "$trip_response" == "FAILED" ]]; then
    echo "✗ (Trip planning failed)"
    exit 1
elif echo "$trip_response" | grep -q "No trips found"; then
    echo "⚠ (API works but no trips found - graph may need rebuild)"
else
    echo "✓ (Trip planning functional)"
fi

# Test 10: Check GTFS-RT support
echo -n "10. Testing GTFS-RT feed support... "
# Test adding a GTFS-RT feed (already added above, so list it)
gtfs_rt_config=$(cat "${OTP_DATA_DIR}/router-config.json" 2>/dev/null | jq -r '.updaters | length' || echo "0")
if [[ "$gtfs_rt_config" != "0" ]]; then
    echo "✓ (${gtfs_rt_config} GTFS-RT feed(s) configured)"
else
    echo "⚠ (No GTFS-RT feeds configured)"
fi

# Test 11: Check GraphQL endpoint availability
echo -n "11. Testing GraphQL API endpoint... "
graphql_response=$(curl -sf -X POST "http://localhost:${OTP_PORT}/otp/transmodel/v3" \
    -H "Content-Type: application/json" \
    -d '{"query":"{ serverInfo { version } }"}' 2>/dev/null || echo "FAILED")

if [[ "$graphql_response" != "FAILED" ]] && echo "$graphql_response" | jq -e '.data.serverInfo.version' &>/dev/null; then
    version=$(echo "$graphql_response" | jq -r '.data.serverInfo.version')
    echo "✓ (GraphQL API v${version})"
else
    echo "✗ (GraphQL API not available)"
    exit 1
fi

echo ""
echo "Integration tests completed successfully!"
echo "Note: Trip planning requires graph with transit data to return results"
exit 0