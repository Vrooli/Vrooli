#!/usr/bin/env bats
# PostGIS Integration Tests

# Test setup
setup() {
    # Get the test directory
    TEST_DIR="$(builtin cd "${BATS_TEST_DIRNAME%/*}" && builtin pwd)"
    POSTGIS_CLI="${TEST_DIR}/cli.sh"
    TEST_FIXTURES="${TEST_DIR}/../../../__test/fixtures/data/spatial"
    
    # Ensure PostGIS is installed and running
    run "${POSTGIS_CLI}" status --format json
    [ "$status" -eq 0 ]
    
    # Check if healthy
    local healthy=$(echo "$output" | jq -r '.healthy')
    if [[ "$healthy" != "true" ]]; then
        skip "PostGIS is not healthy"
    fi
}

@test "PostGIS: Status check returns valid JSON" {
    run "${POSTGIS_CLI}" status --format json
    [ "$status" -eq 0 ]
    
    # Validate JSON structure
    echo "$output" | jq -e '.name == "postgis"'
    echo "$output" | jq -e '.category == "storage"'
    echo "$output" | jq -e '.healthy'
}

@test "PostGIS: Status check returns valid text format" {
    run "${POSTGIS_CLI}" status
    [ "$status" -eq 0 ]
    
    # Check for expected text output
    [[ "$output" == *"PostGIS Status Report"* ]]
    [[ "$output" == *"Basic Status:"* ]]
    [[ "$output" == *"Container Info:"* ]]
}

@test "PostGIS: Can inject SQL file with spatial data" {
    # Create a temporary test database
    local test_db="test_postgis_${RANDOM}"
    
    # Enable PostGIS in test database
    run "${POSTGIS_CLI}" enable-database "${test_db}"
    [ "$status" -eq 0 ]
    
    # Inject test spatial data
    run "${POSTGIS_CLI}" inject "${TEST_FIXTURES}/test_spatial_data.sql" "${test_db}"
    [ "$status" -eq 0 ]
    
    # Verify data was inserted
    run docker exec postgis-main psql -U vrooli -d "${test_db}" -t -c "SELECT COUNT(*) FROM test_locations;"
    [ "$status" -eq 0 ]
    local count=$(echo "$output" | xargs)
    [ "$count" -eq 5 ]
    
    # Clean up test database
    run docker exec postgis-main psql -U vrooli -d postgres -c "DROP DATABASE IF EXISTS ${test_db};"
}

@test "PostGIS: Spatial queries work correctly" {
    # Create a temporary test database
    local test_db="test_spatial_${RANDOM}"
    
    # Enable PostGIS in test database
    run "${POSTGIS_CLI}" enable-database "${test_db}"
    [ "$status" -eq 0 ]
    
    # Create simple test data
    run docker exec postgis-main psql -U vrooli -d "${test_db}" -c "
        CREATE TABLE test_points (id SERIAL PRIMARY KEY, location GEOMETRY(POINT, 4326));
        INSERT INTO test_points (location) VALUES 
            (ST_GeomFromText('POINT(0 0)', 4326)),
            (ST_GeomFromText('POINT(1 1)', 4326));
    "
    [ "$status" -eq 0 ]
    
    # Test spatial function
    run docker exec postgis-main psql -U vrooli -d "${test_db}" -t -c "
        SELECT ST_Distance(
            ST_GeomFromText('POINT(0 0)', 4326)::geography,
            ST_GeomFromText('POINT(1 1)', 4326)::geography
        ) / 1000;
    "
    [ "$status" -eq 0 ]
    
    # Distance should be approximately 157km
    local distance=$(echo "$output" | xargs | cut -d'.' -f1)
    [ "$distance" -ge 155 ] && [ "$distance" -le 159 ]
    
    # Clean up test database
    run docker exec postgis-main psql -U vrooli -d postgres -c "DROP DATABASE IF EXISTS ${test_db};"
}

@test "PostGIS: Examples command works" {
    run "${POSTGIS_CLI}" examples
    [ "$status" -eq 0 ]
    
    # Check for example content
    [[ "$output" == *"PostGIS Example Queries"* ]]
    [[ "$output" == *"ST_"* ]]  # Should contain spatial functions
}

@test "PostGIS: Help command shows usage" {
    run "${POSTGIS_CLI}" help
    [ "$status" -eq 0 ]
    
    # Check for usage content
    [[ "$output" == *"PostGIS Resource CLI"* ]]
    [[ "$output" == *"Commands:"* ]]
    [[ "$output" == *"status"* ]]
    [[ "$output" == *"inject"* ]]
}