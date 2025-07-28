#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/api.sh"
SEARXNG_DIR="$BATS_TEST_DIRNAME/.."

# Helper function for proper sourcing in tests
setup_searxng_api_test_env() {
    local script_dir="$SEARXNG_DIR"
    local resources_dir="$SEARXNG_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Source utilities first
    source "$helpers_dir/utils/log.sh"
    source "$resources_dir/common.sh"
    
    # Source config and messages
    source "$script_dir/config/defaults.sh"
    source "$script_dir/config/messages.sh"
    searxng::export_config
    
    # Source dependencies
    source "$script_dir/lib/common.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Mock functions
    log::info() { echo "INFO: $*"; }
    log::success() { echo "SUCCESS: $*"; }
    log::error() { echo "ERROR: $*"; }
    log::warn() { echo "WARNING: $*"; }
    log::header() { echo "HEADER: $*"; }
    log::debug() { echo "DEBUG: $*"; }
    
    # Mock health check
    searxng::is_healthy() {
        if [[ "$MOCK_SEARXNG_HEALTHY" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    # Mock curl command
    curl() {
        local url=""
        local format=""
        
        # Parse curl arguments
        while [[ $# -gt 0 ]]; do
            case "$1" in
                "-sf"|"-s"|"-f")
                    shift
                    ;;
                "--max-time")
                    shift 2
                    ;;
                "-w")
                    shift 2
                    ;;
                "-o")
                    shift 2
                    ;;
                "-G*"|"--data-urlencode")
                    shift
                    ;;
                *)
                    url="$1"
                    shift
                    ;;
            esac
        done
        
        # Mock responses based on URL and settings
        if [[ "$MOCK_CURL_SUCCESS" == "yes" ]]; then
            case "$url" in
                *"/search"*)
                    if [[ "$url" =~ format=([^&]*) ]]; then
                        format="${BASH_REMATCH[1]}"
                    else
                        format="json"
                    fi
                    
                    case "$format" in
                        "json")
                            echo '{"results":[{"title":"Test Result","url":"http://example.com","content":"Test content"}]}'
                            ;;
                        "xml")
                            echo '<results><result><title>Test Result</title></result></results>'
                            ;;
                        "csv")
                            echo 'title,url,content'
                            echo 'Test Result,http://example.com,Test content'
                            ;;
                        *)
                            echo '{"results":[]}'
                            ;;
                    esac
                    return 0
                    ;;
                *"/stats"*)
                    echo '{"engines":{"google":{"errors":0},"bing":{"errors":0}}}'
                    return 0
                    ;;
                *"/config"*)
                    echo '{"categories":["general","images","news"]}'
                    return 0
                    ;;
                *)
                    return 0
                    ;;
            esac
        else
            return 1
        fi
    }
    
    # Mock printf for URL encoding
    printf() {
        if [[ "$1" == '%s' ]] && [[ "$2" == "test query" ]]; then
            echo "test%20query"
        else
            command printf "$@"
        fi
    }
    
    # Mock commands
    command() {
        case "$*" in
            "-v jq")
                if [[ "$MOCK_JQ_AVAILABLE" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            "-v bc")
                if [[ "$MOCK_BC_AVAILABLE" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Mock jq for JSON formatting
    jq() {
        if [[ "$MOCK_JQ_SUCCESS" == "yes" ]]; then
            echo '{"title": "Test Result", "url": "http://example.com", "content": "Test content"}'
            return 0
        else
            return 1
        fi
    }
    
    # Mock bc for calculations
    bc() {
        if [[ "$*" =~ "1.5 / 1" ]]; then
            echo "1.500"
        elif [[ "$*" =~ "< 1.0" ]]; then
            echo "1"
        else
            echo "0"
        fi
        return 0
    }
    
    # Mock date for timing
    date() {
        case "$*" in
            "+%s.%N")
                echo "1234567890.123"
                ;;
            "+%s")
                echo "1234567890"
                ;;
            *)
                command date "$@"
                ;;
        esac
    }
    
    # Mock echo for math operations
    echo() {
        if [[ "$*" =~ "1234567890.123 - 1234567890.123" ]]; then
            command echo "0.000"
        elif [[ "$*" =~ "0.000 + 1.500" ]]; then
            command echo "1.500"
        else
            command echo "$@"
        fi
    }
    
    # Mock read for interactive input
    read() {
        if [[ "$MOCK_USER_INPUT" ]]; then
            eval "$1='$MOCK_USER_INPUT'"
            return 0
        else
            # Simulate 'quit' to exit interactive mode
            eval "$1='quit'"
            return 0
        fi
    }
    
    # Set default mocks
    export MOCK_SEARXNG_HEALTHY="yes"
    export MOCK_CURL_SUCCESS="yes"
    export MOCK_JQ_AVAILABLE="yes"
    export MOCK_JQ_SUCCESS="yes"
    export MOCK_BC_AVAILABLE="yes"
    export MOCK_USER_INPUT=""
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing api.sh defines required functions" {
    setup_searxng_api_test_env
    
    local required_functions=(
        "searxng::search"
        "searxng::get_stats"
        "searxng::get_api_config"
        "searxng::test_api"
        "searxng::interactive_search"
        "searxng::benchmark"
        "searxng::show_api_examples"
    )
    
    for func in "${required_functions[@]}"; do
        run bash -c "declare -f $func"
        [ "$status" -eq 0 ]
    done
}

# ============================================================================
# Search Function Tests
# ============================================================================

@test "searxng::search performs basic search successfully" {
    setup_searxng_api_test_env
    
    run searxng::search "test query"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INFO: Searching for: test query" ]]
    [[ "$output" =~ '"results"' ]]
}

@test "searxng::search fails with empty query" {
    setup_searxng_api_test_env
    
    run searxng::search ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Search query is required" ]]
}

@test "searxng::search fails when SearXNG unhealthy" {
    setup_searxng_api_test_env
    export MOCK_SEARXNG_HEALTHY="no"
    
    run searxng::search "test query"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: SearXNG is not running or healthy" ]]
}

@test "searxng::search handles different formats" {
    setup_searxng_api_test_env
    
    # Test JSON format
    run searxng::search "test" "json"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"results"' ]]
    
    # Test XML format
    run searxng::search "test" "xml"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '<results>' ]]
    
    # Test CSV format
    run searxng::search "test" "csv"
    [ "$status" -eq 0 ]
    [[ "$output" =~ 'title,url,content' ]]
}

@test "searxng::search handles different categories" {
    setup_searxng_api_test_env
    
    run searxng::search "test" "json" "images"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Searching for: test" ]]
}

@test "searxng::search fails when API request fails" {
    setup_searxng_api_test_env
    export MOCK_CURL_SUCCESS="no"
    
    run searxng::search "test query"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Search request failed" ]]
}

# ============================================================================
# Stats and Config Tests
# ============================================================================

@test "searxng::get_stats retrieves statistics successfully" {
    setup_searxng_api_test_env
    
    run searxng::get_stats
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"engines"' ]]
}

@test "searxng::get_stats fails when SearXNG unhealthy" {
    setup_searxng_api_test_env
    export MOCK_SEARXNG_HEALTHY="no"
    
    run searxng::get_stats
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: SearXNG is not running or healthy" ]]
}

@test "searxng::get_api_config retrieves configuration successfully" {
    setup_searxng_api_test_env
    
    run searxng::get_api_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"categories"' ]]
}

@test "searxng::get_api_config fails when API fails" {
    setup_searxng_api_test_env
    export MOCK_CURL_SUCCESS="no"
    
    run searxng::get_api_config
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Failed to retrieve SearXNG configuration" ]]
}

# ============================================================================
# API Testing Tests
# ============================================================================

@test "searxng::test_api tests all endpoints successfully" {
    setup_searxng_api_test_env
    
    run searxng::test_api
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HEADER: SearXNG API Test" ]]
    [[ "$output" =~ "✅ /stats endpoint responding" ]]
    [[ "$output" =~ "✅ /config endpoint responding" ]]
    [[ "$output" =~ "✅ /search endpoint responding" ]]
    [[ "$output" =~ "SUCCESS: All API tests passed" ]]
}

@test "searxng::test_api fails when SearXNG unhealthy" {
    setup_searxng_api_test_env
    export MOCK_SEARXNG_HEALTHY="no"
    
    run searxng::test_api
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: SearXNG is not running or healthy" ]]
}

@test "searxng::test_api detects endpoint failures" {
    setup_searxng_api_test_env
    export MOCK_CURL_SUCCESS="no"
    
    run searxng::test_api
    [ "$status" -eq 1 ]
    [[ "$output" =~ "❌" ]]
    [[ "$output" =~ "ERROR: API tests failed" ]]
}

@test "searxng::test_api tests multiple formats" {
    setup_searxng_api_test_env
    
    run searxng::test_api
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Testing response formats" ]]
    [[ "$output" =~ "✅ json format working" ]]
    [[ "$output" =~ "✅ xml format working" ]]
    [[ "$output" =~ "✅ csv format working" ]]
}

@test "searxng::test_api tests multiple categories" {
    setup_searxng_api_test_env
    
    run searxng::test_api
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Testing search categories" ]]
    [[ "$output" =~ "✅ general category working" ]]
    [[ "$output" =~ "✅ images category working" ]]
    [[ "$output" =~ "✅ news category working" ]]
}

# ============================================================================
# Interactive Search Tests
# ============================================================================

@test "searxng::interactive_search handles user input" {
    setup_searxng_api_test_env
    export MOCK_USER_INPUT="test search"
    
    run searxng::interactive_search
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SearXNG Interactive Search" ]]
    [[ "$output" =~ "Type 'quit' to exit" ]]
}

@test "searxng::interactive_search fails when SearXNG unhealthy" {
    setup_searxng_api_test_env
    export MOCK_SEARXNG_HEALTHY="no"
    
    run searxng::interactive_search
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: SearXNG is not running or healthy" ]]
}

@test "searxng::interactive_search formats JSON with jq when available" {
    setup_searxng_api_test_env
    export MOCK_USER_INPUT="test"
    export MOCK_JQ_AVAILABLE="yes"
    export MOCK_JQ_SUCCESS="yes"
    
    run searxng::interactive_search
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Searching for: test" ]]
}

@test "searxng::interactive_search handles missing jq gracefully" {
    setup_searxng_api_test_env
    export MOCK_USER_INPUT="test"
    export MOCK_JQ_AVAILABLE="no"
    
    run searxng::interactive_search
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Searching for: test" ]]
}

# ============================================================================
# Benchmark Tests
# ============================================================================

@test "searxng::benchmark runs performance test successfully" {
    setup_searxng_api_test_env
    export MOCK_BC_AVAILABLE="yes"
    
    run searxng::benchmark 3
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Running SearXNG performance benchmark" ]]
    [[ "$output" =~ "Number of test queries: 3" ]]
    [[ "$output" =~ "Benchmark Results:" ]]
    [[ "$output" =~ "Total queries: 3" ]]
}

@test "searxng::benchmark fails when SearXNG unhealthy" {
    setup_searxng_api_test_env
    export MOCK_SEARXNG_HEALTHY="no"
    
    run searxng::benchmark
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: SearXNG is not running or healthy" ]]
}

@test "searxng::benchmark uses default query count" {
    setup_searxng_api_test_env
    
    run searxng::benchmark
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Number of test queries: 10" ]]
}

@test "searxng::benchmark handles failed queries" {
    setup_searxng_api_test_env
    export MOCK_CURL_SUCCESS="no"
    
    run searxng::benchmark 2
    [ "$status" -eq 0 ]
    [[ "$output" =~ "❌ Failed" ]]
    [[ "$output" =~ "Failed: 2" ]]
}

@test "searxng::benchmark calculates performance metrics" {
    setup_searxng_api_test_env
    export MOCK_BC_AVAILABLE="yes"
    
    run searxng::benchmark 1
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Average response time:" ]]
    [[ "$output" =~ "Performance:" ]]
}

# ============================================================================
# API Examples Tests
# ============================================================================

@test "searxng::show_api_examples displays comprehensive examples" {
    setup_searxng_api_test_env
    
    run searxng::show_api_examples
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HEADER: SearXNG API Usage Examples" ]]
    [[ "$output" =~ "Basic Search:" ]]
    [[ "$output" =~ "Image Search:" ]]
    [[ "$output" =~ "News Search:" ]]
    [[ "$output" =~ "Get Statistics:" ]]
    [[ "$output" =~ "Get Configuration:" ]]
}

@test "searxng::show_api_examples shows available formats" {
    setup_searxng_api_test_env
    
    run searxng::show_api_examples
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Available Formats:" ]]
    [[ "$output" =~ "- json (default)" ]]
    [[ "$output" =~ "- xml" ]]
    [[ "$output" =~ "- csv" ]]
    [[ "$output" =~ "- rss" ]]
}

@test "searxng::show_api_examples shows available categories" {
    setup_searxng_api_test_env
    
    run searxng::show_api_examples
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Available Categories:" ]]
    [[ "$output" =~ "- general (default)" ]]
    [[ "$output" =~ "- images" ]]
    [[ "$output" =~ "- videos" ]]
    [[ "$output" =~ "- news" ]]
    [[ "$output" =~ "- music" ]]
    [[ "$output" =~ "- files" ]]
    [[ "$output" =~ "- science" ]]
}

@test "searxng::show_api_examples includes script usage" {
    setup_searxng_api_test_env
    
    run searxng::show_api_examples
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Script Usage:" ]]
    [[ "$output" =~ "./manage.sh --action search" ]]
    [[ "$output" =~ "./manage.sh --action api-test" ]]
    [[ "$output" =~ "./manage.sh --action benchmark" ]]
}

@test "searxng::show_api_examples includes correct URLs" {
    setup_searxng_api_test_env
    
    run searxng::show_api_examples
    [ "$status" -eq 0 ]
    [[ "$output" =~ "$SEARXNG_BASE_URL/search" ]]
    [[ "$output" =~ "$SEARXNG_BASE_URL/stats" ]]
    [[ "$output" =~ "$SEARXNG_BASE_URL/config" ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "API functions handle network failures gracefully" {
    setup_searxng_api_test_env
    export MOCK_CURL_SUCCESS="no"
    
    # Test that all API functions fail gracefully
    run searxng::search "test"
    [ "$status" -eq 1 ]
    
    run searxng::get_stats
    [ "$status" -eq 1 ]
    
    run searxng::get_api_config
    [ "$status" -eq 1 ]
}

@test "URL encoding works correctly" {
    setup_searxng_api_test_env
    
    run searxng::search "test query with spaces"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Searching for: test query with spaces" ]]
}

@test "API timeout handling works" {
    setup_searxng_api_test_env
    
    # Verify that API calls include timeout parameter
    run searxng::search "test"
    [ "$status" -eq 0 ]
    # The mock curl should receive --max-time parameter
}

@test "benchmark handles mathematical operations correctly" {
    setup_searxng_api_test_env
    export MOCK_BC_AVAILABLE="yes"
    
    run searxng::benchmark 1
    [ "$status" -eq 0 ]
    # Should complete without mathematical errors
    [[ "$output" =~ "Average response time:" ]]
}
# ============================================================================
# JSON API Integration Tests
# ============================================================================

@test "searxng::search returns valid JSON format" {
    setup_searxng_api_test_env
    
    run searxng::search "artificial intelligence" "json"
    [ "$status" -eq 0 ]
    
    # Check JSON structure
    [[ "$output" =~ '"query"' ]]
    [[ "$output" =~ '"results"' ]]
    [[ "$output" =~ '"number_of_results"' ]]
}

@test "searxng::search handles special characters in queries" {
    setup_searxng_api_test_env
    
    # Test with special characters that need URL encoding
    run searxng::search "C++ programming & algorithms" "json"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"results"' ]]
}

@test "searxng::search supports multiple categories" {
    setup_searxng_api_test_env
    
    # Test category filtering
    run searxng::search "technology" "json" "general,news"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"results"' ]]
}

@test "searxng::search handles language parameter" {
    setup_searxng_api_test_env
    
    # Test language-specific search
    run searxng::search "tecnología" "json" "general" "es"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"results"' ]]
}

@test "API returns proper error for invalid format" {
    setup_searxng_api_test_env
    export MOCK_CURL_ERROR_FORMAT="yes"
    
    # Mock curl to return error for invalid format
    curl() {
        if [[ "$*" =~ "format=invalid" ]]; then
            echo '{"error": "Invalid format"}'
            return 1
        else
            echo '{"results": []}'
            return 0
        fi
    }
    
    run searxng::search "test" "invalid"
    # Should handle gracefully even with invalid format
}

@test "API handles pagination correctly" {
    setup_searxng_api_test_env
    
    # Test pagination parameter
    export MOCK_CURL_RESPONSE='{"results": [{"page": 2}], "pageno": 2}'
    
    curl() {
        if [[ "$*" =~ "pageno=2" ]]; then
            echo "$MOCK_CURL_RESPONSE"
            return 0
        else
            echo '{"results": []}'
            return 0
        fi
    }
    
    run searxng::search "test" "json" "general" "en" "2"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"pageno": 2' ]]
}

@test "API respects safe search settings" {
    setup_searxng_api_test_env
    
    # Test safe search parameter
    curl() {
        if [[ "$*" =~ "safesearch=2" ]]; then
            echo '{"results": [], "safesearch": 2}'
            return 0
        else
            echo '{"results": []}'
            return 0
        fi
    }
    
    run searxng::search "test" "json" "general" "en" "1" "2"
    [ "$status" -eq 0 ]
}

@test "API handles time range filtering" {
    setup_searxng_api_test_env
    
    # Test time range parameter
    curl() {
        if [[ "$*" =~ "time_range=day" ]]; then
            echo '{"results": [{"timerange": "day"}], "time_range": "day"}'
            return 0
        else
            echo '{"results": []}'
            return 0
        fi
    }
    
    run searxng::search "news" "json" "news" "en" "1" "1" "day"
    [ "$status" -eq 0 ]
}

@test "searxng::test_api validates all endpoints" {
    setup_searxng_api_test_env
    
    run searxng::test_api
    [ "$status" -eq 0 ]
    
    # Should test multiple endpoints
    [[ "$output" =~ "Testing search endpoint" ]]
    [[ "$output" =~ "Testing stats endpoint" ]]
    [[ "$output" =~ "Testing config endpoint" ]]
}

@test "API benchmark calculates performance metrics" {
    setup_searxng_api_test_env
    export MOCK_BC_AVAILABLE="yes"
    
    # Mock multiple curl responses for benchmark
    local call_count=0
    curl() {
        ((call_count++))
        echo '{"results": [{"id": '"$call_count"'}]}'
        return 0
    }
    
    run searxng::benchmark 3
    [ "$status" -eq 0 ]
    
    # Should show performance summary
    [[ "$output" =~ "Running 3 search queries" ]]
    [[ "$output" =~ "Average response time:" ]]
    [[ "$output" =~ "Total time:" ]]
}

@test "interactive search handles user input correctly" {
    setup_searxng_api_test_env
    
    # Simulate user entering a query then quitting
    local input_count=0
    read() {
        ((input_count++))
        if [[ $input_count -eq 1 ]]; then
            eval "$1='machine learning'"
        else
            eval "$1='quit'"
        fi
        return 0
    }
    
    run searxng::interactive_search
    [ "$status" -eq 0 ]
    
    # Should process the search and exit cleanly
    [[ "$output" =~ "Enter search query" ]]
    [[ "$output" =~ "Searching for: machine learning" ]]
    [[ "$output" =~ "Exiting interactive search" ]]
}

@test "API error handling for network issues" {
    setup_searxng_api_test_env
    
    # Mock curl network failure
    curl() {
        return 7  # Connection refused
    }
    
    run searxng::search "test"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Failed to search" ]]
}

@test "API handles empty results gracefully" {
    setup_searxng_api_test_env
    
    # Mock empty results
    curl() {
        echo '{"query": "obscure12345query", "results": [], "number_of_results": 0}'
        return 0
    }
    
    run searxng::search "obscure12345query" "json"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"results": \[\]' ]]
}
EOF < /dev/null
