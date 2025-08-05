#!/usr/bin/env bats


# Load test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"

# Setup for each test
setup() {
    # Use auto-setup for service tests
    vrooli_auto_setup
    
    # Mock system functions (lightweight)
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    system::is_command() { command -v "$1" >/dev/null 2>&1; }
    
    # Mock basic curl function
    curl() {
        case "$*" in
            *"health"*) echo '{"status":"healthy"}';;
            *) echo '{"success":true}';;
        esac
        return 0
    }
    export -f curl
    
    
    # Path to the script under test
    SCRIPT_PATH="$BATS_TEST_DIRNAME/api.sh"
    SEARXNG_DIR="$BATS_TEST_DIRNAME/.."
    
    # Set up test environment
    export SEARXNG_TEST_MODE=yes
    export SEARXNG_PORT=9200
    export SEARXNG_BASE_URL='http://localhost:9200'
    export SEARXNG_DATA_DIR="$HOME/.searxng"
    
    # Mock variables
    export MOCK_SEARXNG_HEALTHY=yes
    export MOCK_CURL_SUCCESS=yes
    export MOCK_JQ_AVAILABLE=yes
    export MOCK_API_RESPONSE='{"results": [{"title": "Test Result", "url": "http://example.com", "content": "Test content"}], "number_of_results": 1}'
    
    # Mock other required functions
    searxng::is_healthy() { [[ "$MOCK_SEARXNG_HEALTHY" == "yes" ]]; }
    searxng::validate_url() { return 0; }  # Always valid in tests
    
    # Mock command function for jq/curl availability checks
    command() {
        case "$1 $2" in
            "which jq") [[ "$MOCK_JQ_AVAILABLE" == "yes" ]] && echo "/usr/bin/jq" || return 1 ;;
            "which curl") echo "/usr/bin/curl" ;;
            "-v jq") [[ "$MOCK_JQ_AVAILABLE" == "yes" ]] && echo "/usr/bin/jq" || return 1 ;;
            "-v curl") echo "/usr/bin/curl" ;;
            *) /usr/bin/command "$@" ;;
        esac
    }
    
    # Mock curl function
    curl() {
        if [[ "$MOCK_CURL_SUCCESS" == "yes" ]]; then
            echo "$MOCK_API_RESPONSE"
            return 0
        else
            echo "curl: (7) Failed to connect" >&2
            return 7
        fi
    }
    
    # Mock jq function - simulates jq behavior for our test data
    jq() {
        if [[ "$MOCK_JQ_AVAILABLE" == "yes" ]]; then
            # Try to use real jq if available
            if command -v /usr/bin/jq >/dev/null 2>&1; then
                /usr/bin/jq "$@"
            else
                # Fallback to simple mock for common patterns
                case "$1" in
                    "-r")
                        shift
                        local query="$1"
                        case "$query" in
                            ".results[0].url // empty")
                                if [[ "$MOCK_API_RESPONSE" == *"\"results\": []"* ]]; then
                                    # Return empty for no results
                                    :
                                else
                                    echo "http://example.com"
                                fi
                                ;;
                            ".results[:2] | .[] | .title")
                                echo -e "Result 1\nResult 2"
                                ;;
                            ".results | .[] | .title")
                                echo "Test Result"
                                ;;
                            '.results | .[] | "## \(.title)\n\n**URL:** \(.url)\n\n\(.content)\n\n**Source:** \(.engine)\n\n---\n"'*)
                                echo -e "## Test Result\n\n**URL:** http://example.com\n\nTest content\n\n**Source:** \n\n---\n"
                                ;;
                            '.results | .[] | "Title,URL,Description"'*)
                                echo "Title,URL,Description"
                                ;;
                            '.results | .[] | [.title, .url, .content, .engine] | @csv'*)
                                echo '"Test Result","http://example.com","Test content",""'
                                ;;
                            '.query // "Unknown"')
                                echo "Unknown"
                                ;;
                            *)
                                echo "Test Result"
                                ;;
                        esac
                        ;;
                    "-c")
                        echo "$MOCK_API_RESPONSE"
                        ;;
                    *)
                        # Handle basic jq queries
                        case "$1" in
                            "{query: .query, number_of_results: .number_of_results, results: .results[:2]}")
                                echo '{"query":null,"number_of_results":1,"results":[{"title":"Result 1"},{"title":"Result 2"}]}'
                                ;;
                            *)
                                echo "$MOCK_API_RESPONSE"
                                ;;
                        esac
                        ;;
                esac
            fi
            return 0
        else
            echo "jq: command not found" >&2
            return 127
        fi
    }
    
    # Source the script
    source "$SCRIPT_PATH" || { echo "Failed to source api.sh" >&2; exit 1; }
}

# ============================================================================
# Core Function Tests
# ============================================================================

@test "sourcing api.sh defines all required functions" {
    local required_functions=(
        "searxng::search"
        "searxng::format_output"
        "searxng::get_stats"
        "searxng::get_api_config"
        "searxng::test_api"
        "searxng::interactive_search"
        "searxng::benchmark"
        "searxng::show_api_examples"
        "searxng::headlines"
        "searxng::lucky"
        "searxng::batch_search_file"
        "searxng::batch_search_queries"
    )
    
    for func in "${required_functions[@]}"; do
        run bash -c "declare -F '$func'"
        if [ "$status" -ne 0 ]; then
            echo "MISSING FUNCTION: $func"
            return 1
        fi
    done
}

# ============================================================================
# Search Function Tests
# ============================================================================

@test "searxng::search performs basic search successfully" {
    run searxng::search 'test query'
    echo "Status: $status" >&2
    echo "Output: $output" >&2
    [ "$status" -eq 0 ]
    [[ "$output" =~ "results" ]]
}

@test "searxng::search fails with empty query" {
    run searxng::search ''
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

@test "searxng::search fails when SearXNG unhealthy" {
    export MOCK_SEARXNG_HEALTHY=no
    run searxng::search 'test'
    [ "$status" -eq 1 ]
}

@test "searxng::search handles pagination" {
    run searxng::search 'test' '' '' 2
    [ "$status" -eq 0 ]
    [[ "$output" =~ "results" ]]
}

@test "searxng::search handles time range" {
    run searxng::search 'test' 'json' 'general' 'en' '1' '1' 'day'
    [ "$status" -eq 0 ]
}

@test "searxng::search handles safe search" {
    run searxng::search 'test' '' '' 1 '' 1
    [ "$status" -eq 0 ]
}

# ============================================================================
# Format Output Tests
# ============================================================================

@test "searxng::format_output handles json format" {
    run searxng::format_output "$MOCK_API_RESPONSE" "json"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "results" ]]
}

@test "searxng::format_output handles title-only format" {
    run searxng::format_output "$MOCK_API_RESPONSE" "title-only"
    [ "$status" -eq 0 ]
    [[ "$output" == "Test Result" ]]
}

@test "searxng::format_output handles csv format" {
    run searxng::format_output "$MOCK_API_RESPONSE" "csv"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "title,url,content,engine" ]]
    [[ "$output" =~ "Test Result" ]]
}

@test "searxng::format_output handles markdown format" {
    run searxng::format_output "$MOCK_API_RESPONSE" "markdown"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "# Search Results" ]]
    [[ "$output" =~ "## Test Result" ]]
}

@test "searxng::format_output applies limit" {
    export MOCK_API_RESPONSE='{"results": [{"title": "Result 1"}, {"title": "Result 2"}, {"title": "Result 3"}]}'
    run searxng::format_output "$MOCK_API_RESPONSE" "title-only" "2"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Result 1" ]]
    [[ "$output" =~ "Result 2" ]]
    [[ ! "$output" =~ "Result 3" ]]
}

# ============================================================================
# Headlines Function Tests
# ============================================================================

@test "searxng::headlines fetches general headlines" {
    run searxng::headlines
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Getting latest headlines" ]]
}

@test "searxng::headlines fetches topic-specific headlines" {
    run searxng::headlines 'technology'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "technology" ]]
}

# ============================================================================
# Lucky Search Tests
# ============================================================================

@test "searxng::lucky returns first result URL" {
    run searxng::lucky 'test query'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "http://example.com" ]]
}

@test "searxng::lucky fails with empty query" {
    run searxng::lucky ''
    [ "$status" -eq 1 ]
}

@test "searxng::lucky handles no results" {
    export MOCK_API_RESPONSE='{"results": []}'
    run searxng::lucky 'test'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No results found" ]]
}

# ============================================================================
# Batch Search Tests
# ============================================================================

@test "searxng::batch_search_queries processes multiple queries" {
    run searxng::batch_search_queries 'query1,query2'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting batch search with 2 queries" ]]
    [[ "$output" =~ "batch_1_query1.json" ]]
    [[ "$output" =~ "batch_2_query2.json" ]]
}

@test "searxng::batch_search_file processes queries from file" {
    echo -e 'query1\nquery2' > /tmp/test_queries.txt
    run searxng::batch_search_file /tmp/test_queries.txt
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Processing 2 queries..." ]]
    rm -f /tmp/test_queries.txt
}

# ============================================================================
# API Status Tests  
# ============================================================================

@test "searxng::get_stats retrieves statistics" {
    export MOCK_API_RESPONSE='{"stats": "test"}'
    run searxng::get_stats
    [ "$status" -eq 0 ]
    [[ "$output" =~ "stats" ]]
}

@test "searxng::get_api_config retrieves configuration" {
    export MOCK_API_RESPONSE='{"config": "test"}'
    run searxng::get_api_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "config" ]]
}

@test "searxng::test_api performs comprehensive test" {
    run searxng::test_api
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SearXNG API Test" ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "functions handle curl failures gracefully" {
    export MOCK_CURL_SUCCESS=no
    run searxng::search 'test'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

@test "functions handle missing jq gracefully" {
    export MOCK_JQ_AVAILABLE=no
    run searxng::format_output "$MOCK_API_RESPONSE" "title-only"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "jq: command not found" ]]
}

# Clean up any temporary files
teardown() {
    vrooli_cleanup_test
    rm -f /tmp/test_queries.txt
    rm -f batch_*.json
    rm -f results_*.json
}
