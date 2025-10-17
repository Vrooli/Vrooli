#!/bin/bash

# GeoNode Integration Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CLI="${RESOURCE_DIR}/cli.sh"

# Test utilities
source "${RESOURCE_DIR}/lib/core.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Test counter
TESTS_RUN=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    ((TESTS_RUN++))
    echo -n "  Testing ${test_name}... "
    
    if eval "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Create test data
create_test_geojson() {
    cat > /tmp/test_points.geojson << 'EOF'
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "properties": {
        "name": "Test Point 1",
        "description": "First test location"
      },
      "geometry": {
        "type": "Point",
        "coordinates": [-74.006, 40.7128]
      }
    },
    {
      "type": "Feature",
      "properties": {
        "name": "Test Point 2",
        "description": "Second test location"
      },
      "geometry": {
        "type": "Point",
        "coordinates": [-73.9776, 40.7614]
      }
    }
  ]
}
EOF
}

# Main integration tests
main() {
    echo "Running GeoNode integration tests..."
    echo ""
    
    # Load configuration
    load_config
    
    # Check if service is running
    if [[ "$(get_status)" != "running" ]]; then
        echo "GeoNode is not running. Starting service..."
        "${CLI}" manage start --wait || {
            echo -e "${RED}Failed to start GeoNode${NC}"
            exit 1
        }
    fi
    
    # Create test data
    create_test_geojson
    
    echo "Lifecycle tests:"
    
    # Test restart
    run_test "Service restart" \
        "${CLI} manage restart"
    
    # Wait for services to be ready
    sleep 10
    
    run_test "Health after restart" \
        "timeout 5 curl -sf http://localhost:${GEONODE_PORT}/health"
    
    echo ""
    echo "Data management tests:"
    
    # Test layer upload
    run_test "Upload GeoJSON layer" \
        "${CLI} content add-layer /tmp/test_points.geojson"
    
    # Test layer listing
    run_test "List layers" \
        "${CLI} content list-layers"
    
    # Test API authentication
    run_test "API with authentication" \
        "curl -sf -u ${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD} http://localhost:${GEONODE_PORT}/api/v2/layers/ | grep -q 'objects'"
    
    echo ""
    echo "Service interaction tests:"
    
    # Test GeoServer WMS capabilities
    run_test "GeoServer WMS GetCapabilities" \
        "timeout 5 curl -sf http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/wms?service=WMS&version=1.1.0&request=GetCapabilities | grep -q 'WMS_Capabilities'"
    
    # Test GeoServer WFS capabilities
    run_test "GeoServer WFS GetCapabilities" \
        "timeout 5 curl -sf http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/wfs?service=WFS&version=1.0.0&request=GetCapabilities | grep -q 'WFS_Capabilities'"
    
    # Test Django admin interface
    run_test "Django admin login page" \
        "timeout 5 curl -sf http://localhost:${GEONODE_PORT}/admin/ | grep -q 'Django'"
    
    echo ""
    echo "Container health tests:"
    
    # Check all containers are healthy
    run_test "Django container healthy" \
        "docker ps --filter name=geonode-django --filter health=healthy --format '{{.Names}}' | grep -q geonode-django"
    
    run_test "GeoServer container healthy" \
        "docker ps --filter name=geonode-geoserver --filter health=healthy --format '{{.Names}}' | grep -q geonode-geoserver"
    
    run_test "Database container healthy" \
        "docker ps --filter name=geonode-postgres --filter health=healthy --format '{{.Names}}' | grep -q geonode-postgres"
    
    run_test "Redis container healthy" \
        "docker ps --filter name=geonode-redis --filter health=healthy --format '{{.Names}}' | grep -q geonode-redis"
    
    # Cleanup
    rm -f /tmp/test_points.geojson
    
    echo ""
    echo "Integration test summary:"
    echo "  Tests run: ${TESTS_RUN}"
    echo "  Tests failed: ${TESTS_FAILED}"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}Integration tests failed${NC}"
        exit 1
    else
        echo -e "${GREEN}All integration tests passed${NC}"
        exit 0
    fi
}

main "$@"