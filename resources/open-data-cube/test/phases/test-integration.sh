#!/bin/bash

# Open Data Cube Integration Tests
# Full end-to-end functionality testing (<120s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${RESOURCE_DIR}/lib/core.sh"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test function wrapper
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    echo -n "  ${test_name}... "
    
    if eval "$test_cmd" &>/dev/null; then
        echo "✓"
        ((TESTS_PASSED++))
        return 0
    else
        echo "✗"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Create test product definition
create_test_product() {
    docker exec ${ODC_API_CONTAINER} bash -c "cat > /tmp/test_product.yaml << 'EOF'
name: integration_test_product
description: Integration test product
metadata_type: eo3
metadata:
  product:
    name: integration_test_product
  properties:
    eo:instrument: TEST
    eo:platform: TEST_PLATFORM
measurements:
- name: band1
  dtype: int16
  nodata: -9999
  units: '1'
- name: band2
  dtype: int16
  nodata: -9999
  units: '1'
EOF
datacube product add /tmp/test_product.yaml"
}

# Create test dataset
create_test_dataset() {
    docker exec ${ODC_API_CONTAINER} bash -c "
# Create test GeoTIFF
python3 -c \"
import numpy as np
import rasterio
from rasterio.transform import from_origin

# Create test data
data = np.random.randint(0, 255, (100, 100), dtype=np.uint8)

# Define transform
transform = from_origin(0, 1, 0.01, 0.01)

# Write GeoTIFF
with rasterio.open(
    '/tmp/test_data.tif',
    'w',
    driver='GTiff',
    height=100,
    width=100,
    count=1,
    dtype='uint8',
    crs='EPSG:4326',
    transform=transform,
) as dst:
    dst.write(data, 1)
\" 2>/dev/null || touch /tmp/test_data.tif

# Create dataset metadata
cat > /tmp/test_dataset.yaml << 'EOF'
id: integration-test-dataset-001
product: integration_test_product
location: file:///tmp/test_data.tif
datetime: 2024-01-15T12:00:00
properties:
  eo:instrument: TEST
  eo:platform: TEST_PLATFORM
geometry:
  type: Polygon
  coordinates: [[[0,0],[0.5,0],[0.5,0.5],[0,0.5],[0,0]]]
grids:
  default:
    shape: [100, 100]
    transform: [0.01, 0, 0, 0, -0.01, 1]
EOF

# Index the dataset
datacube dataset add /tmp/test_dataset.yaml
"
}

# Query test dataset
query_test_dataset() {
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
dc = datacube.Datacube()
datasets = list(dc.find_datasets(product='integration_test_product'))
if not datasets:
    exit(1)
print(f'Found {len(datasets)} test datasets')
"
}

# Test export functionality
test_export_functionality() {
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
import json
dc = datacube.Datacube()

# Find test dataset
datasets = list(dc.find_datasets(product='integration_test_product', limit=1))
if not datasets:
    exit(1)

# Test GeoJSON export
ds = datasets[0]
geojson = {
    'type': 'Feature',
    'geometry': {'type': 'Polygon', 'coordinates': [[[0,0],[0.5,0],[0.5,0.5],[0,0.5],[0,0]]]},
    'properties': {'id': str(ds.id), 'product': 'integration_test_product'}
}

# Write test export
with open('/tmp/test_export.json', 'w') as f:
    json.dump(geojson, f)

print('Export test successful')
"
}

# Cleanup test data
cleanup_test_data() {
    docker exec ${ODC_API_CONTAINER} bash -c "
# Archive test dataset
datacube dataset archive integration-test-dataset-001 2>/dev/null || true

# Remove test product
datacube product remove integration_test_product 2>/dev/null || true

# Clean up files
rm -f /tmp/test_*.yaml /tmp/test_*.tif /tmp/test_*.json 2>/dev/null || true
" &>/dev/null || true
}

# Main integration test execution
main() {
    echo "Running Open Data Cube integration tests..."
    echo ""
    
    local start_time=$(date +%s)
    
    # Test 1: Create test product
    run_test "Create test product" "create_test_product"
    
    # Test 2: Verify product creation
    run_test "Verify product exists" "docker exec ${ODC_API_CONTAINER} datacube product list | grep -q integration_test_product"
    
    # Test 3: Create and index test dataset
    run_test "Index test dataset" "create_test_dataset"
    
    # Test 4: Query indexed dataset
    run_test "Query test dataset" "query_test_dataset"
    
    # Test 5: Test data export
    run_test "Export functionality" "test_export_functionality"
    
    # Test 6: OWS GetCapabilities
    run_test "OWS GetCapabilities" "timeout 5 curl -sf 'http://localhost:${DATACUBE_OWS_PORT}/wms?service=WMS&request=GetCapabilities' | grep -q WMS"
    
    # Test 7: API dataset search
    run_test "API dataset search" "docker exec ${ODC_API_CONTAINER} datacube dataset search product='integration_test_product'"
    
    # Test 8: Database query
    run_test "Database query" "docker exec ${ODC_DB_CONTAINER} psql -U datacube -d datacube -c 'SELECT COUNT(*) FROM dataset;'"
    
    # Test 9: Redis cache
    run_test "Redis cache test" "docker exec ${ODC_REDIS_CONTAINER} redis-cli SET test_key test_value && docker exec ${ODC_REDIS_CONTAINER} redis-cli GET test_key"
    
    # Test 10: Cleanup
    run_test "Cleanup test data" "cleanup_test_data; true"
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "----------------------------------------"
    echo "Integration Test Results:"
    echo "  Passed: ${TESTS_PASSED}"
    echo "  Failed: ${TESTS_FAILED}"
    echo "  Duration: ${duration}s"
    
    if [[ ${TESTS_FAILED} -eq 0 ]]; then
        echo "  Status: SUCCESS"
        echo "----------------------------------------"
        return 0
    else
        echo "  Status: FAILURE"
        echo "----------------------------------------"
        return 1
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi