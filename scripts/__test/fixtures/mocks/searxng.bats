#!/usr/bin/env bats
# SearXNG Mock System Tests
# Comprehensive test coverage for SearXNG mock functionality

setup() {
    # Set up test environment
    export TEST_TEMP_DIR=$(mktemp -d)
    export SEARXNG_MOCK_STATE_FILE="$TEST_TEMP_DIR/searxng_state"
    
    # Source the mock
    source "${BATS_TEST_DIRNAME}/searxng.sh"
    
    # Reset state before each test
    mock::searxng::reset
}

teardown() {
    # Clean up
    [[ -d "$TEST_TEMP_DIR" ]] && rm -rf "$TEST_TEMP_DIR"
}

# ----------------------------
# Container Management Tests
# ----------------------------
@test "searxng mock: initial state is not_installed" {
    run searxng::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "not_installed" ]
    
    run searxng::is_installed
    [ "$status" -eq 1 ]
    
    run searxng::is_running
    [ "$status" -eq 1 ]
    
    run searxng::is_healthy
    [ "$status" -eq 1 ]
}

@test "searxng mock: install transitions to stopped state" {
    run searxng::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing SearXNG" ]]
    [[ "$output" =~ "installed successfully" ]]
    
    run searxng::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "stopped" ]
    
    run searxng::is_installed
    [ "$status" -eq 0 ]
    
    run searxng::is_running
    [ "$status" -eq 1 ]
}

@test "searxng mock: start transitions to running state" {
    # Install first
    searxng::install
    
    run searxng::start_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting SearXNG" ]]
    [[ "$output" =~ "started successfully" ]]
    
    run searxng::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "healthy" ]
    
    run searxng::is_running
    [ "$status" -eq 0 ]
    
    run searxng::is_healthy
    [ "$status" -eq 0 ]
}

@test "searxng mock: cannot start when not installed" {
    run searxng::start_container
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not installed" ]]
}

@test "searxng mock: stop transitions to stopped state" {
    # Install and start
    searxng::install
    searxng::start_container
    
    run searxng::stop_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Stopping SearXNG" ]]
    [[ "$output" =~ "stopped" ]]
    
    run searxng::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "stopped" ]
    
    run searxng::is_running
    [ "$status" -eq 1 ]
}

@test "searxng mock: unhealthy state handling" {
    mock::searxng::set_container_status "unhealthy"
    
    run searxng::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "unhealthy" ]
    
    run searxng::is_running
    [ "$status" -eq 0 ]  # Container is running but unhealthy
    
    run searxng::is_healthy
    [ "$status" -eq 1 ]  # Not healthy
}

# ----------------------------
# Search API Tests
# ----------------------------
@test "searxng mock: search requires healthy state" {
    run searxng::search "test query"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running or healthy" ]]
}

@test "searxng mock: basic search works when healthy" {
    mock::searxng::scenario::healthy_instance
    
    run searxng::search "test query"
    [ "$status" -eq 0 ]
    
    # Verify JSON structure
    echo "$output" | jq -e '.query == "test query"' >/dev/null
    echo "$output" | jq -e '.results | length > 0' >/dev/null
    echo "$output" | jq -e '.results[0] | has("title", "url", "content", "engine")' >/dev/null
}

@test "searxng mock: search with different formats" {
    mock::searxng::scenario::healthy_instance
    
    # JSON format
    run searxng::search "test" "json"
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.query' >/dev/null
    
    # XML format
    run searxng::search "test" "xml"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "<?xml" ]]
    [[ "$output" =~ "<results>" ]]
    
    # CSV format
    run searxng::search "test" "csv"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "title,url,content,engine" ]]
    
    # RSS format
    run searxng::search "test" "rss"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "<rss version" ]]
}

@test "searxng mock: search with categories" {
    mock::searxng::scenario::healthy_instance
    
    local categories=("general" "images" "videos" "news")
    for category in "${categories[@]}"; do
        run searxng::search "test" "json" "$category"
        [ "$status" -eq 0 ]
        echo "$output" | jq -e --arg cat "$category" '.results[0].category == $cat' >/dev/null
    done
}

@test "searxng mock: search updates state counters" {
    mock::searxng::scenario::healthy_instance
    
    # Initial count
    run mock::searxng::get::search_count
    [ "$output" = "0" ]
    
    # Perform searches
    searxng::search "query1" >/dev/null
    searxng::search "query2" >/dev/null
    searxng::search "query3" >/dev/null
    
    # Check count
    run mock::searxng::get::search_count
    [ "$output" = "3" ]
    
    # Check last query
    run mock::searxng::assert::last_query "query3"
    [ "$status" -eq 0 ]
}

@test "searxng mock: custom search results" {
    mock::searxng::scenario::healthy_instance
    
    # Set custom result
    local custom_result='{"query":"custom","number_of_results":1,"results":[{"title":"Custom Result","url":"https://custom.com","content":"Custom content","engine":"custom-engine"}]}'
    mock::searxng::set_search_result "custom" "$custom_result"
    
    run searxng::search "custom"
    [ "$status" -eq 0 ]
    
    # Verify custom result is returned
    echo "$output" | jq -e '.results[0].title == "Custom Result"' >/dev/null
    echo "$output" | jq -e '.results[0].engine == "custom-engine"' >/dev/null
}

@test "searxng mock: search with file output" {
    mock::searxng::scenario::healthy_instance
    
    local save_file="$TEST_TEMP_DIR/results.json"
    
    run searxng::search "test" "json" "general" "en" "1" "1" "" "json" "" "$save_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Results saved to: $save_file" ]]
    
    # Verify file was created and contains valid JSON
    [ -f "$save_file" ]
    jq -e '.query == "test"' "$save_file" >/dev/null
}

@test "searxng mock: search with append file" {
    mock::searxng::scenario::healthy_instance
    
    local append_file="$TEST_TEMP_DIR/append.jsonl"
    
    # Perform multiple searches with append
    searxng::search "query1" "json" "general" "en" "1" "1" "" "json" "" "" "$append_file" >/dev/null
    searxng::search "query2" "json" "general" "en" "1" "1" "" "json" "" "" "$append_file" >/dev/null
    
    # Verify file contains both results
    [ -f "$append_file" ]
    local line_count=$(wc -l < "$append_file")
    [ "$line_count" -eq 2 ]
}

# ----------------------------
# API Endpoint Tests
# ----------------------------
@test "searxng mock: stats endpoint requires healthy state" {
    run searxng::get_stats
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running or healthy" ]]
}

@test "searxng mock: stats endpoint returns valid data" {
    mock::searxng::scenario::healthy_instance
    
    # Perform some searches first
    searxng::search "test1" >/dev/null
    searxng::search "test2" >/dev/null
    
    run searxng::get_stats
    [ "$status" -eq 0 ]
    
    # Verify JSON structure
    echo "$output" | jq -e '.engines | length > 0' >/dev/null
    echo "$output" | jq -e '.timing | has("total", "engine", "processing")' >/dev/null
    echo "$output" | jq -e '.search_count == 2' >/dev/null
}

@test "searxng mock: config endpoint returns valid data" {
    mock::searxng::scenario::healthy_instance
    
    run searxng::get_api_config
    [ "$status" -eq 0 ]
    
    # Verify JSON structure
    echo "$output" | jq -e '.categories | length > 0' >/dev/null
    echo "$output" | jq -e '.engines | length > 0' >/dev/null
    echo "$output" | jq -e '.server | has("port", "bind_address", "secret_key", "base_url")' >/dev/null
}

@test "searxng mock: custom config values" {
    mock::searxng::scenario::healthy_instance
    
    # Set custom config
    mock::searxng::set_config "safe_search" "2"
    mock::searxng::set_config "default_lang" "de"
    
    run searxng::get_api_config
    [ "$status" -eq 0 ]
    
    echo "$output" | jq -e '.safe_search == 2' >/dev/null
    echo "$output" | jq -e '.default_lang == "de"' >/dev/null
}

# ----------------------------
# Benchmark Tests
# ----------------------------
@test "searxng mock: benchmark requires healthy state" {
    run searxng::benchmark 5
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running or healthy" ]]
}

@test "searxng mock: benchmark runs successfully" {
    mock::searxng::scenario::healthy_instance
    
    run searxng::benchmark 5
    [ "$status" -eq 0 ]
    
    # Verify output structure
    [[ "$output" =~ "Running SearXNG performance benchmark" ]]
    [[ "$output" =~ "Number of test queries: 5" ]]
    [[ "$output" =~ "Total queries: 5" ]]
    [[ "$output" =~ "Successful: 5" ]]
    [[ "$output" =~ "Failed: 0" ]]
    [[ "$output" =~ "Average response time:" ]]
    [[ "$output" =~ "Performance: ✅" ]]
}

# ----------------------------
# API Test Function
# ----------------------------
@test "searxng mock: test_api requires healthy state" {
    run searxng::test_api
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running or healthy" ]]
}

@test "searxng mock: test_api reports all endpoints working" {
    mock::searxng::scenario::healthy_instance
    
    run searxng::test_api
    [ "$status" -eq 0 ]
    
    # Verify all endpoints report success
    [[ "$output" =~ "✅ /stats endpoint responding" ]]
    [[ "$output" =~ "✅ /config endpoint responding" ]]
    [[ "$output" =~ "✅ /search endpoint responding" ]]
    [[ "$output" =~ "✅ json format working" ]]
    [[ "$output" =~ "✅ xml format working" ]]
    [[ "$output" =~ "✅ csv format working" ]]
    [[ "$output" =~ "All API tests passed" ]]
}

# ----------------------------
# Status Display Tests
# ----------------------------
@test "searxng mock: show_status displays correct information" {
    mock::searxng::scenario::healthy_instance
    
    # Perform some operations
    searxng::search "test query" >/dev/null
    searxng::get_stats >/dev/null
    
    run searxng::show_status
    [ "$status" -eq 0 ]
    
    # Verify output contains expected information
    [[ "$output" =~ "Status: healthy" ]]
    [[ "$output" =~ "Port: 8200" ]]
    [[ "$output" =~ "Base URL: http://localhost:8200" ]]
    [[ "$output" =~ "Health: healthy" ]]
    [[ "$output" =~ "Search Count: 1" ]]
    [[ "$output" =~ "Last Query: test query" ]]
    [[ "$output" =~ "API Calls: 1" ]]
}

# ----------------------------
# Error Injection Tests
# ----------------------------
@test "searxng mock: inject install error" {
    mock::searxng::inject_error "install" "permission_denied"
    
    run searxng::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Installation failed: permission_denied" ]]
}

@test "searxng mock: inject start error" {
    searxng::install
    mock::searxng::inject_error "start" "port_conflict"
    
    run searxng::start_container
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Start failed: port_conflict" ]]
}

# ----------------------------
# Scenario Tests
# ----------------------------
@test "searxng mock: healthy instance scenario" {
    run mock::searxng::scenario::healthy_instance
    [ "$status" -eq 0 ]
    
    run searxng::is_healthy
    [ "$status" -eq 0 ]
    
    # Should be able to perform all operations
    run searxng::search "test"
    [ "$status" -eq 0 ]
    
    run searxng::get_stats
    [ "$status" -eq 0 ]
    
    run searxng::get_api_config
    [ "$status" -eq 0 ]
}

@test "searxng mock: unhealthy instance scenario" {
    run mock::searxng::scenario::unhealthy_instance
    [ "$status" -eq 0 ]
    
    run searxng::get_status
    [ "$output" = "unhealthy" ]
    
    # Should not be able to perform operations
    run searxng::search "test"
    [ "$status" -eq 1 ]
}

@test "searxng mock: with search results scenario" {
    run mock::searxng::scenario::with_search_results
    [ "$status" -eq 0 ]
    
    # Test predefined result
    run searxng::search "test"
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.results[0].title == "Test Result 1"' >/dev/null
    
    # Test another predefined result
    run searxng::search "docker"
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.results[0].title == "Docker Documentation"' >/dev/null
}

# ----------------------------
# State Persistence Tests
# ----------------------------
@test "searxng mock: state persists across subshells" {
    mock::searxng::scenario::healthy_instance
    
    # Perform operation in main shell
    searxng::search "main shell query" >/dev/null
    
    # Check state in subshell
    local count=$(bash -c "source '${BATS_TEST_DIRNAME}/searxng.sh'; mock::searxng::get::search_count")
    [ "$count" = "1" ]
    
    # Perform operation in subshell
    bash -c "source '${BATS_TEST_DIRNAME}/searxng.sh'; searxng::search 'subshell query' >/dev/null"
    
    # Check updated state
    run mock::searxng::get::search_count
    [ "$output" = "2" ]
}

# ----------------------------
# Assertion Helper Tests
# ----------------------------
@test "searxng mock: assertions work correctly" {
    mock::searxng::scenario::healthy_instance
    
    # Test is_installed assertion
    run mock::searxng::assert::is_installed
    [ "$status" -eq 0 ]
    
    # Test is_running assertion
    run mock::searxng::assert::is_running
    [ "$status" -eq 0 ]
    
    # Test is_healthy assertion
    run mock::searxng::assert::is_healthy
    [ "$status" -eq 0 ]
    
    # Test search count assertion
    searxng::search "test" >/dev/null
    run mock::searxng::assert::search_count 1
    [ "$status" -eq 0 ]
    
    run mock::searxng::assert::search_count 2
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ASSERTION FAILED" ]]
    
    # Test last query assertion
    run mock::searxng::assert::last_query "test"
    [ "$status" -eq 0 ]
    
    run mock::searxng::assert::last_query "wrong"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ASSERTION FAILED" ]]
}

# ----------------------------
# Debug Functions Tests
# ----------------------------
@test "searxng mock: debug dump shows complete state" {
    mock::searxng::scenario::healthy_instance
    
    # Perform various operations
    searxng::search "debug test" >/dev/null
    mock::searxng::inject_error "test" "test_error"
    mock::searxng::set_config "test_key" "test_value"
    
    run mock::searxng::debug::dump_state
    [ "$status" -eq 0 ]
    
    # Verify output contains all state information
    [[ "$output" =~ "=== SearXNG Mock State Dump ===" ]]
    [[ "$output" =~ "Container Status: running" ]]
    [[ "$output" =~ "Health Status: healthy" ]]
    [[ "$output" =~ "Search Count: 1" ]]
    [[ "$output" =~ "Last Query: debug test" ]]
    [[ "$output" =~ "test: test_error" ]]
    [[ "$output" =~ "test_key: test_value" ]]
}

# ----------------------------
# Edge Cases and Error Handling
# ----------------------------
@test "searxng mock: handles empty search query gracefully" {
    mock::searxng::scenario::healthy_instance
    
    run searxng::search ""
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.query == ""' >/dev/null
}

@test "searxng mock: handles special characters in query" {
    mock::searxng::scenario::healthy_instance
    
    local special_query='test & "quotes" <tags> $variable'
    run searxng::search "$special_query"
    [ "$status" -eq 0 ]
    echo "$output" | jq -e --arg q "$special_query" '.query == $q' >/dev/null
}

@test "searxng mock: handles rapid successive operations" {
    mock::searxng::scenario::healthy_instance
    
    # Perform rapid operations
    for i in {1..10}; do
        searxng::search "query$i" >/dev/null
    done
    
    run mock::searxng::get::search_count
    [ "$output" = "10" ]
}

@test "searxng mock: multiple install attempts tracked" {
    # First install
    searxng::install >/dev/null
    
    # Stop and try to install again
    searxng::stop_container >/dev/null
    searxng::install >/dev/null
    
    # Check install attempts
    run bash -c "source '${BATS_TEST_DIRNAME}/searxng.sh'; echo \${MOCK_SEARXNG_STATE[install_attempts]}"
    [ "$output" = "2" ]
}

@test "searxng mock: response time configuration" {
    mock::searxng::scenario::healthy_instance
    
    # Set custom response time
    mock::searxng::set_response_time "search" "250"
    
    # Verify it's stored (would be used by actual implementation)
    run bash -c "source '${BATS_TEST_DIRNAME}/searxng.sh'; echo \${MOCK_SEARXNG_RESPONSE_TIMES[search]}"
    [ "$output" = "250" ]
}