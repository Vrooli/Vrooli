#!/usr/bin/env bash
################################################################################
# PostGIS Integration Tests
# End-to-end functionality testing (<120s)
################################################################################

set -euo pipefail

# Determine script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "$SCRIPT_DIR/../../../../" && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test utility functions
test::suite() { echo -e "\n╔════════════════════════════════════╗\n║ $* ║\n╚════════════════════════════════════════╝"; }
test::start() { echo -n "  Testing $*... "; }
test::pass() { echo -e "✅ $*"; }
test::fail() { echo -e "❌ $*"; }
test::warning() { echo -e "⚠️  $*"; }
test::success() { echo -e "\n✅ $*"; }
test::error() { echo -e "\n❌ $*"; }
log::info() { echo -e "ℹ️  $*"; }
log::success() { test::success "$@"; }
log::error() { test::error "$@"; }

# Test configuration
POSTGIS_PORT="${POSTGIS_PORT:-5434}"
POSTGIS_CONTAINER="${POSTGIS_CONTAINER:-postgis-main}"
TEST_DB="test_spatial_$(date +%s)"

# Cleanup function
cleanup() {
    # Drop test database if it exists
    docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -c "DROP DATABASE IF EXISTS ${TEST_DB};" &>/dev/null || true
    
    # Remove test files
    rm -f /tmp/test_geom.sql /tmp/test_export.* 2>/dev/null || true
}

# Set trap for cleanup
trap cleanup EXIT

# Test functions
test_lifecycle() {
    test::start "Resource lifecycle"
    
    # Test restart
    if vrooli resource postgis manage restart; then
        sleep 5
        if docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
            test::pass "Restart successful"
        else
            test::fail "Container not running after restart"
            return 1
        fi
    else
        test::fail "Restart command failed"
        return 1
    fi
}

test_database_operations() {
    test::start "Database operations"
    
    # Create test database
    if docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -c "CREATE DATABASE ${TEST_DB};" &>/dev/null; then
        # Enable PostGIS in test database
        if docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -c "CREATE EXTENSION postgis;" &>/dev/null; then
            test::pass "Database creation and PostGIS enabling successful"
        else
            test::fail "Failed to enable PostGIS in test database"
            return 1
        fi
    else
        test::fail "Failed to create test database"
        return 1
    fi
}

test_spatial_data_types() {
    test::start "Spatial data types"
    
    # Create table with geometry column
    local create_sql="CREATE TABLE test_locations (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100),
        location GEOMETRY(Point, 4326)
    );"
    
    if docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -c "$create_sql" &>/dev/null; then
        # Insert test data
        local insert_sql="INSERT INTO test_locations (name, location) VALUES 
            ('Point A', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326)),
            ('Point B', ST_GeomFromText('POINT(-74.006020 40.712776)', 4326));"
        
        if docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -c "$insert_sql" &>/dev/null; then
            test::pass "Spatial data types working"
        else
            test::fail "Failed to insert spatial data"
            return 1
        fi
    else
        test::fail "Failed to create table with geometry column"
        return 1
    fi
}

test_spatial_queries() {
    test::start "Spatial queries"
    
    # Test distance calculation
    local distance_sql="SELECT ST_Distance(
        a.location::geography,
        b.location::geography
    ) AS distance_meters
    FROM test_locations a, test_locations b
    WHERE a.name = 'Point A' AND b.name = 'Point B';"
    
    local distance
    distance=$(docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -t -c "$distance_sql" 2>/dev/null | xargs)
    
    if [[ -n "$distance" ]]; then
        test::pass "Spatial queries working (distance: ${distance}m)"
    else
        test::fail "Spatial query failed"
        return 1
    fi
}

test_spatial_indexing() {
    test::start "Spatial indexing"
    
    # Create spatial index
    local index_sql="CREATE INDEX idx_location ON test_locations USING GIST(location);"
    
    if docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -c "$index_sql" &>/dev/null; then
        # Verify index is used
        local explain_sql="EXPLAIN SELECT * FROM test_locations WHERE ST_DWithin(location, ST_GeomFromText('POINT(-74 40)', 4326), 1);"
        local explain_result
        explain_result=$(docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -c "$explain_sql" 2>/dev/null)
        
        if echo "$explain_result" | grep -q "idx_location"; then
            test::pass "Spatial index created and used"
        else
            test::pass "Spatial index created (usage not verified)"
        fi
    else
        test::fail "Failed to create spatial index"
        return 1
    fi
}

test_content_management() {
    test::start "Content management"
    
    # Create test SQL file
    cat > /tmp/test_geom.sql << 'EOF'
CREATE TABLE IF NOT EXISTS test_content (
    id SERIAL PRIMARY KEY,
    geom GEOMETRY(Point, 4326)
);
INSERT INTO test_content (geom) VALUES 
    (ST_GeomFromText('POINT(0 0)', 4326));
EOF
    
    # Test content add
    if vrooli resource postgis content add /tmp/test_geom.sql "${TEST_DB}"; then
        # Verify table was created
        local table_exists
        table_exists=$(docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -t -c \
            "SELECT 1 FROM information_schema.tables WHERE table_name = 'test_content';" 2>/dev/null | xargs)
        
        if [[ "$table_exists" == "1" ]]; then
            test::pass "Content management working"
        else
            test::fail "Content added but table not found"
            return 1
        fi
    else
        test::fail "Content add failed"
        return 1
    fi
}

test_geospatial_functions() {
    test::start "Advanced geospatial functions"
    
    # Test buffer operation
    local buffer_sql="SELECT ST_Area(ST_Buffer(ST_GeomFromText('POINT(0 0)', 4326)::geography, 1000));"
    local area
    area=$(docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -t -c "$buffer_sql" 2>/dev/null | xargs)
    
    if [[ -n "$area" ]] && (( $(echo "$area > 0" | bc -l) )); then
        test::pass "Buffer operation working (area: ${area}m²)"
    else
        test::fail "Buffer operation failed"
        return 1
    fi
    
    # Test intersection
    local intersect_sql="SELECT ST_Intersects(
        ST_GeomFromText('POLYGON((0 0, 1 0, 1 1, 0 1, 0 0))', 4326),
        ST_GeomFromText('POINT(0.5 0.5)', 4326)
    );"
    
    local intersects
    intersects=$(docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -t -c "$intersect_sql" 2>/dev/null | xargs)
    
    if [[ "$intersects" == "t" ]]; then
        test::pass "Intersection detection working"
    else
        test::fail "Intersection detection failed"
        return 1
    fi
}

test_performance() {
    test::start "Performance benchmarks"
    
    # Insert many points for performance testing
    local bulk_insert="INSERT INTO test_locations (name, location)
    SELECT 
        'Point_' || generate_series,
        ST_SetSRID(ST_MakePoint(
            random() * 360 - 180,
            random() * 180 - 90
        ), 4326)
    FROM generate_series(1, 1000);"
    
    local start_time
    start_time=$(date +%s%N)
    if docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "${TEST_DB}" -c "$bulk_insert" &>/dev/null; then
        local end_time
        end_time=$(date +%s%N)
        local duration=$(((end_time - start_time) / 1000000))
        
        if [[ $duration -lt 5000 ]]; then
            test::pass "Bulk insert performance acceptable (${duration}ms for 1000 points)"
        else
            test::warning "Bulk insert slow (${duration}ms for 1000 points)"
        fi
    else
        test::fail "Bulk insert failed"
        return 1
    fi
}

# Main test execution
main() {
    test::suite "PostGIS Integration Tests"
    
    local failed=0
    
    # Run tests
    test_lifecycle || ((failed++))
    test_database_operations || ((failed++))
    test_spatial_data_types || ((failed++))
    test_spatial_queries || ((failed++))
    test_spatial_indexing || ((failed++))
    test_content_management || ((failed++))
    test_geospatial_functions || ((failed++))
    test_performance || ((failed++))
    
    # Summary
    echo
    if [[ $failed -eq 0 ]]; then
        test::success "All integration tests passed!"
        return 0
    else
        test::error "$failed integration tests failed"
        return 1
    fi
}

# Run tests
main "$@"