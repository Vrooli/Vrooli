#!/usr/bin/env bats
# ====================================================================
# Tests for Cache Manager - Layer 1 Syntax Validation
# ====================================================================

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../../lib/utils/var.sh"
# shellcheck disable=SC1091  
source "${var_TEST_DIR}/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Source the cache manager
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/cache-manager.sh"
    
    # Create test environment
    export TEST_CACHE_DIR="${BATS_TEST_TMPDIR}/cache_test"
    mkdir -p "$TEST_CACHE_DIR"
    
    # Initialize cache manager with test directory
    cache_manager::init
    
    # Create test file for testing
    export TEST_FILE="$TEST_CACHE_DIR/test_resource.sh"
    echo "#!/usr/bin/env bash" > "$TEST_FILE"
    echo "echo 'test script'" >> "$TEST_FILE"
}

teardown() {
    # Cleanup cache manager
    if command -v cache_manager::cleanup &>/dev/null; then
        cache_manager::cleanup 2>/dev/null || true
    fi
    
    vrooli_cleanup_test
}

@test "cache_manager::init: initializes cache successfully" {
    # Should be already initialized in setup, test cache directory exists
    [[ -d "$CACHE_DIR" ]]
}

@test "cache_manager::generate_key: generates valid cache key from file" {
    run cache_manager::generate_key "test_resource" "$TEST_FILE"
    
    assert_success
    assert_output --regexp "^syntax-test_resource-[a-f0-9]{64}$"
}

@test "cache_manager::generate_key: fails for non-existent file" {
    run cache_manager::generate_key "test_resource" "/non/existent/file"
    
    assert_failure
    assert_output --partial "File not found"
}

@test "cache_manager::get_file_path: returns correct path" {
    local test_key="syntax-test-abc123"
    
    run cache_manager::get_file_path "$test_key"
    
    assert_success
    assert_output "${CACHE_DIR}/${test_key}.json"
}

@test "cache_manager::set and get: stores and retrieves cache" {
    local test_result='{"status": "passed", "details": "test passed", "duration_ms": 100}'
    
    # Set cache
    run cache_manager::set "test_resource" "$TEST_FILE" "$test_result"
    assert_success
    
    # Get cache
    run cache_manager::get "test_resource" "$TEST_FILE"
    assert_success
    assert_output "$test_result"
}

@test "cache_manager::get: returns failure for cache miss" {
    # Try to get cache that doesn't exist
    run cache_manager::get "nonexistent_resource" "$TEST_FILE"
    
    assert_failure
}

@test "cache_manager::is_valid: validates cache file timestamp" {
    local test_result='{"status": "passed", "details": "test passed", "duration_ms": 100}'
    
    # Set cache
    cache_manager::set "test_resource" "$TEST_FILE" "$test_result"
    
    # Get cache file path
    local cache_key
    cache_key=$(cache_manager::generate_key "test_resource" "$TEST_FILE")
    local cache_file
    cache_file=$(cache_manager::get_file_path "$cache_key")
    
    # Should be valid (just created)
    run cache_manager::is_valid "$cache_file"
    assert_success
}

@test "cache_manager::create_result_json: creates valid JSON" {
    run cache_manager::create_result_json "passed" "All tests passed" 150
    
    assert_success
    assert_output --partial '"status": "passed"'
    assert_output --partial '"details": "All tests passed"'
    assert_output --partial '"duration_ms": 150'
    assert_output --partial '"cache_version": "1.0"'
}

@test "cache_manager::parse_result: parses cached JSON correctly" {
    local test_json='{"status": "passed", "details": "Test successful", "duration_ms": 100, "timestamp": "2023-01-01T12:00:00Z"}'
    
    run cache_manager::parse_result "$test_json"
    
    assert_success
    # Check stderr for details output
    [[ "$stderr" == "Test successful" ]]
}

@test "cache_manager::parse_result: handles failed status" {
    local test_json='{"status": "failed", "details": "Test failed", "duration_ms": 100, "timestamp": "2023-01-01T12:00:00Z"}'
    
    run cache_manager::parse_result "$test_json"
    
    assert_failure
    # Check stderr for details output
    [[ "$stderr" == "Test failed" ]]
}

@test "cache_manager::get_stats: returns cache statistics" {
    # Set some cache entries
    local test_result='{"status": "passed", "details": "test", "duration_ms": 100}'
    cache_manager::set "test1" "$TEST_FILE" "$test_result"
    cache_manager::set "test2" "$TEST_FILE" "$test_result"
    
    run cache_manager::get_stats
    
    assert_success
    assert_output --partial '"cache_hits":'
    assert_output --partial '"cache_misses":'
    assert_output --partial '"total_entries":'
    assert_output --partial '"cache_directory":'
}

@test "cache_manager::clear_all: removes all cache entries" {
    # Create some cache entries
    local test_result='{"status": "passed", "details": "test", "duration_ms": 100}'
    cache_manager::set "test1" "$TEST_FILE" "$test_result"
    cache_manager::set "test2" "$TEST_FILE" "$test_result"
    
    # Clear all cache
    run cache_manager::clear_all
    assert_success
    assert_output --partial "Cleared"
    
    # Verify cache is empty
    run cache_manager::get "test1" "$TEST_FILE"
    assert_failure
}

@test "cache_manager::validate_integrity: validates cache files" {
    # Create valid cache entry
    local test_result='{"status": "passed", "details": "test", "duration_ms": 100, "timestamp": "2023-01-01T12:00:00Z"}'
    cache_manager::set "test_resource" "$TEST_FILE" "$test_result"
    
    run cache_manager::validate_integrity
    
    assert_success
    assert_output --partial "Cache Integrity Report:"
    assert_output --partial "Valid files: 1"
    assert_output --partial "Invalid files: 0"
}