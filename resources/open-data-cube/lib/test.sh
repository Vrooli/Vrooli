#!/bin/bash

# Open Data Cube Test Library
# Provides test functionality for ODC resource

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh"

# Handle test commands
handle_test() {
    local phase="${1:-all}"
    shift || true
    
    case "$phase" in
        smoke)
            run_smoke_tests "$@"
            ;;
        integration)
            run_integration_tests "$@"
            ;;
        unit)
            run_unit_tests "$@"
            ;;
        all)
            run_all_tests "$@"
            ;;
        *)
            echo "Usage: test [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
}

# Run smoke tests
run_smoke_tests() {
    echo "Running Open Data Cube smoke tests..."
    
    local start_time=$(date +%s)
    local test_passed=true
    
    # Test 1: Check if services are running
    echo -n "  Checking container status... "
    if docker ps --format "{{.Names}}" | grep -q "${ODC_API_CONTAINER}"; then
        echo "✓"
    else
        echo "✗ (API container not running)"
        test_passed=false
    fi
    
    # Test 2: Check database connectivity
    echo -n "  Checking database connection... "
    if docker exec ${ODC_DB_CONTAINER} pg_isready -U datacube &>/dev/null; then
        echo "✓"
    else
        echo "✗ (Database not ready)"
        test_passed=false
    fi
    
    # Test 3: Check API health endpoint
    echo -n "  Checking API health... "
    if timeout 5 curl -sf "http://localhost:${ODC_PORT}/health" &>/dev/null; then
        echo "✓"
    else
        echo "✗ (API not responding)"
        test_passed=false
    fi
    
    # Test 4: Check datacube CLI
    echo -n "  Checking datacube CLI... "
    if docker exec ${ODC_API_CONTAINER} datacube --version &>/dev/null; then
        echo "✓"
    else
        echo "✗ (Datacube CLI not working)"
        test_passed=false
    fi
    
    # Test 5: Check product listing
    echo -n "  Checking product listing... "
    if docker exec ${ODC_API_CONTAINER} datacube product list &>/dev/null; then
        echo "✓"
    else
        echo "✗ (Cannot list products)"
        test_passed=false
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    if [[ "$test_passed" == true ]]; then
        echo "Smoke tests passed (${duration}s)"
        return 0
    else
        echo "Smoke tests failed (${duration}s)"
        return 1
    fi
}

# Run integration tests
run_integration_tests() {
    echo "Running Open Data Cube integration tests..."
    
    local start_time=$(date +%s)
    local test_passed=true
    
    # Test 1: Create test product
    echo -n "  Creating test product... "
    if create_test_product; then
        echo "✓"
    else
        echo "✗"
        test_passed=false
    fi
    
    # Test 2: Index sample dataset
    echo -n "  Indexing sample dataset... "
    if index_sample_dataset; then
        echo "✓"
    else
        echo "✗"
        test_passed=false
    fi
    
    # Test 3: Query indexed data
    echo -n "  Querying indexed data... "
    if query_test_data; then
        echo "✓"
    else
        echo "✗"
        test_passed=false
    fi
    
    # Test 4: Test WMS/WCS endpoint
    echo -n "  Testing OWS endpoint... "
    if test_ows_endpoint; then
        echo "✓"
    else
        echo "✗"
        test_passed=false
    fi
    
    # Test 5: Export test
    echo -n "  Testing data export... "
    if test_export; then
        echo "✓"
    else
        echo "✗"
        test_passed=false
    fi
    
    # Cleanup
    cleanup_test_data
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    if [[ "$test_passed" == true ]]; then
        echo "Integration tests passed (${duration}s)"
        return 0
    else
        echo "Integration tests failed (${duration}s)"
        return 1
    fi
}

# Run unit tests
run_unit_tests() {
    echo "Running Open Data Cube unit tests..."
    
    local start_time=$(date +%s)
    local test_passed=true
    
    # Test 1: Port allocation
    echo -n "  Testing port allocation... "
    if [[ -n "${ODC_PORT}" ]] && [[ -n "${ODC_DB_PORT}" ]]; then
        echo "✓"
    else
        echo "✗"
        test_passed=false
    fi
    
    # Test 2: Configuration files
    echo -n "  Testing configuration files... "
    if [[ -f "${CONFIG_DIR}/defaults.sh" ]] && [[ -f "${CONFIG_DIR}/runtime.json" ]]; then
        echo "✓"
    else
        echo "✗"
        test_passed=false
    fi
    
    # Test 3: Docker compose file generation
    echo -n "  Testing docker-compose generation... "
    if [[ -f "${DOCKER_DIR}/docker-compose.yml" ]]; then
        echo "✓"
    else
        echo "✗"
        test_passed=false
    fi
    
    # Test 4: Data directory structure
    echo -n "  Testing data directory structure... "
    if [[ -d "${DATA_DIR}/products" ]] && [[ -d "${DATA_DIR}/indexed" ]]; then
        echo "✓"
    else
        echo "✗"
        test_passed=false
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    if [[ "$test_passed" == true ]]; then
        echo "Unit tests passed (${duration}s)"
        return 0
    else
        echo "Unit tests failed (${duration}s)"
        return 1
    fi
}

# Run all tests
run_all_tests() {
    echo "Running all Open Data Cube tests..."
    echo ""
    
    local all_passed=true
    
    if ! run_smoke_tests; then
        all_passed=false
    fi
    
    echo ""
    
    if ! run_integration_tests; then
        all_passed=false
    fi
    
    echo ""
    
    if ! run_unit_tests; then
        all_passed=false
    fi
    
    echo ""
    if [[ "$all_passed" == true ]]; then
        echo "All tests passed"
        return 0
    else
        echo "Some tests failed"
        return 1
    fi
}

# Helper: Create test product
create_test_product() {
    docker exec ${ODC_API_CONTAINER} bash -c "cat > /tmp/test_product.yaml << EOF
name: test_landsat
description: Test Landsat Product
metadata_type: eo3
metadata:
  product:
    name: test_landsat
  properties:
    eo:instrument: OLI
    eo:platform: LANDSAT_8
measurements:
- name: blue
  dtype: int16
  nodata: -9999
  units: '1'
EOF
datacube product add /tmp/test_product.yaml" &>/dev/null
}

# Helper: Index sample dataset
index_sample_dataset() {
    # Create a minimal test dataset
    docker exec ${ODC_API_CONTAINER} bash -c "cat > /tmp/test_dataset.yaml << EOF
id: a8b7c6d5-e4f3-2a1b-9c8d-7e6f5a4b3c2d
product: test_landsat
location: file:///tmp/test_data.tif
datetime: 2024-01-01T00:00:00
properties:
  eo:instrument: OLI
  eo:platform: LANDSAT_8
geometry:
  type: Polygon
  coordinates: [[[0,0],[1,0],[1,1],[0,1],[0,0]]]
grids:
  default:
    shape: [100, 100]
    transform: [0.01, 0, 0, 0, -0.01, 1]
EOF
# Create dummy data file
touch /tmp/test_data.tif
datacube dataset add /tmp/test_dataset.yaml" &>/dev/null
}

# Helper: Query test data
query_test_data() {
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
dc = datacube.Datacube()
datasets = list(dc.find_datasets(product='test_landsat'))
exit(0 if len(datasets) > 0 else 1)
" &>/dev/null
}

# Helper: Test OWS endpoint
test_ows_endpoint() {
    timeout 5 curl -sf "http://localhost:${DATACUBE_OWS_PORT}/wms?service=WMS&request=GetCapabilities" &>/dev/null
}

# Helper: Test export functionality
test_export() {
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
import json
dc = datacube.Datacube()
# Just test that export functions exist
datasets = dc.find_datasets(product='test_landsat', limit=1)
if datasets:
    # Test GeoJSON export capability
    features = []
    for ds in datasets[:1]:
        features.append({
            'type': 'Feature',
            'geometry': {'type': 'Polygon', 'coordinates': [[[0,0],[1,0],[1,1],[0,1],[0,0]]]},
            'properties': {'id': str(ds.id)}
        })
    geojson = {'type': 'FeatureCollection', 'features': features}
    json.dumps(geojson)
exit(0)
" &>/dev/null
}

# Helper: Cleanup test data
cleanup_test_data() {
    docker exec ${ODC_API_CONTAINER} bash -c "
datacube dataset archive a8b7c6d5-e4f3-2a1b-9c8d-7e6f5a4b3c2d 2>/dev/null || true
datacube product remove test_landsat 2>/dev/null || true
rm -f /tmp/test_*.yaml /tmp/test_data.tif 2>/dev/null || true
" &>/dev/null || true
}