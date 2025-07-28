#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/api.sh"
SEARXNG_DIR="$BATS_TEST_DIRNAME/.."

# ============================================================================
# Test Helpers
# ============================================================================

# Run a command in a clean bash environment with mocks
run_with_mocks() {
    local cmd="$1"
    shift
    local extra_setup="$*"
    
    run bash -c "
        # Set up test environment
        export SEARXNG_TEST_MODE=yes
        export SEARXNG_PORT=8100
        export SEARXNG_BASE_URL='http://localhost:8100'
        export SEARXNG_DATA_DIR='\$HOME/.searxng'
        
        # Mock variables
        export MOCK_SEARXNG_HEALTHY=yes
        export MOCK_CURL_SUCCESS=yes
        export MOCK_JQ_AVAILABLE=yes
        export MOCK_API_RESPONSE='{\"results\": [{\"title\": \"Test Result\", \"url\": \"http://example.com\", \"content\": \"Test content\"}], \"number_of_results\": 1}'
        
        # Mock logging functions FIRST (before sourcing)
        log::info() { echo \"INFO: \$*\"; }
        log::error() { echo \"ERROR: \$*\" >&2; return 1; }
        log::success() { echo \"SUCCESS: \$*\"; }
        log::warn() { echo \"WARNING: \$*\" >&2; }
        log::debug() { :; }  # Silent debug
        log::header() { echo \"=== \$* ===\"; }
        
        # Mock other required functions
        searxng::is_healthy() { [[ \"\$MOCK_SEARXNG_HEALTHY\" == \"yes\" ]]; }
        searxng::validate_url() { return 0; }  # Always valid in tests
        
        # Mock command function
        command() {
            case \"\$1 \$2\" in
                \"which jq\") [[ \"\$MOCK_JQ_AVAILABLE\" == \"yes\" ]] && echo \"/usr/bin/jq\" || return 1 ;;
                \"which curl\") echo \"/usr/bin/curl\" ;;
                \"-v jq\") [[ \"\$MOCK_JQ_AVAILABLE\" == \"yes\" ]] && echo \"/usr/bin/jq\" || return 1 ;;
                \"-v curl\") echo \"/usr/bin/curl\" ;;
                *) /usr/bin/command \"\$@\" ;;
            esac
        }
        
        # Mock curl function
        curl() {
            if [[ \"\$MOCK_CURL_SUCCESS\" == \"yes\" ]]; then
                echo \"\$MOCK_API_RESPONSE\"
                return 0
            else
                echo \"curl: (7) Failed to connect\" >&2
                return 7
            fi
        }
        
        # Mock jq function - simulates jq behavior for our test data
        jq() {
            if [[ \"\$MOCK_JQ_AVAILABLE\" == \"yes\" ]]; then
                # Try to use real jq if available
                if command -v /usr/bin/jq >/dev/null 2>&1; then
                    /usr/bin/jq \"\$@\"
                else
                    # Fallback to simple mock for common patterns
                    case \"\$1\" in
                        \"-r\")
                            shift
                            case \"\$1\" in
                                \".results[0].url // empty\")
                                    echo \"http://example.com\"
                                    ;;
                                \".results[:*] | .[] | .title\")
                                    echo \"Test Result\"
                                    ;;
                                \".results | .[] | .title\")
                                    echo \"Test Result\"
                                    ;;
                                *)
                                    echo \"\$MOCK_API_RESPONSE\"
                                    ;;
                            esac
                            ;;
                        \"-c\")
                            echo \"\$MOCK_API_RESPONSE\"
                            ;;
                        *)
                            echo \"\$MOCK_API_RESPONSE\"
                            ;;
                    esac
                fi
                return 0
            else
                echo \"jq: command not found\" >&2
                return 127
            fi
        }
        
        # Additional setup if provided
        $extra_setup
        
        # Source the script
        source '$SCRIPT_PATH' || { echo \"Failed to source api.sh\" >&2; exit 1; }
        
        # Run the command
        $cmd
    "
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
        run bash -c "source '$SCRIPT_PATH' && declare -F '$func'"
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
    run_with_mocks "searxng::search 'test query'"
    echo "Status: $status" >&2
    echo "Output: $output" >&2
    [ "$status" -eq 0 ]
    [[ "$output" =~ "results" ]]
}

@test "searxng::search fails with empty query" {
    run_with_mocks "searxng::search ''"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

@test "searxng::search fails when SearXNG unhealthy" {
    run_with_mocks "searxng::search 'test'" "export MOCK_SEARXNG_HEALTHY=no"
    [ "$status" -eq 1 ]
}

@test "searxng::search handles pagination" {
    run_with_mocks "searxng::search 'test' '' '' 2"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "results" ]]
}

@test "searxng::search handles time range" {
    run_with_mocks "searxng::search 'test' 'json' 'general' 'en' '1' '1' 'day'"
    [ "$status" -eq 0 ]
}

@test "searxng::search handles safe search" {
    run_with_mocks "searxng::search 'test' '' '' 1 '' 1"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Format Output Tests
# ============================================================================

@test "searxng::format_output handles json format" {
    run_with_mocks "searxng::format_output \"\$MOCK_API_RESPONSE\" \"json\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "results" ]]
}

@test "searxng::format_output handles title-only format" {
    run_with_mocks "searxng::format_output \"\$MOCK_API_RESPONSE\" \"title-only\""
    [ "$status" -eq 0 ]
    [[ "$output" == "Test Result" ]]
}

@test "searxng::format_output handles csv format" {
    run_with_mocks "searxng::format_output \"\$MOCK_API_RESPONSE\" \"csv\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "title,url,content,engine" ]]
    [[ "$output" =~ "Test Result" ]]
}

@test "searxng::format_output handles markdown format" {
    run_with_mocks "searxng::format_output \"\$MOCK_API_RESPONSE\" \"markdown\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "# Search Results" ]]
    [[ "$output" =~ "## Test Result" ]]
}

@test "searxng::format_output applies limit" {
    run_with_mocks "
        export MOCK_API_RESPONSE='{\"results\": [{\"title\": \"Result 1\"}, {\"title\": \"Result 2\"}, {\"title\": \"Result 3\"}]}'
        searxng::format_output \"\$MOCK_API_RESPONSE\" \"title-only\" \"2\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Result 1" ]]
    [[ "$output" =~ "Result 2" ]]
    [[ ! "$output" =~ "Result 3" ]]
}

# ============================================================================
# Headlines Function Tests
# ============================================================================

@test "searxng::headlines fetches general headlines" {
    run_with_mocks "searxng::headlines"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Getting latest headlines" ]]
}

@test "searxng::headlines fetches topic-specific headlines" {
    run_with_mocks "searxng::headlines 'technology'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "technology" ]]
}

# ============================================================================
# Lucky Search Tests
# ============================================================================

@test "searxng::lucky returns first result URL" {
    run_with_mocks "searxng::lucky 'test query'"
    [ "$status" -eq 0 ]
    [[ "$output" == "http://example.com" ]]
}

@test "searxng::lucky fails with empty query" {
    run_with_mocks "searxng::lucky ''"
    [ "$status" -eq 1 ]
}

@test "searxng::lucky handles no results" {
    run_with_mocks "searxng::lucky 'test'" "export MOCK_API_RESPONSE='{\"results\": []}'"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No results found" ]]
}

# ============================================================================
# Batch Search Tests
# ============================================================================

@test "searxng::batch_search_queries processes multiple queries" {
    run_with_mocks "searxng::batch_search_queries 'query1,query2'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting batch search with 2 queries" ]]
    [[ "$output" =~ "batch_1_query1.json" ]]
    [[ "$output" =~ "batch_2_query2.json" ]]
}

@test "searxng::batch_search_file processes queries from file" {
    run_with_mocks "
        echo -e 'query1\\nquery2' > /tmp/test_queries.txt
        searxng::batch_search_file /tmp/test_queries.txt
        rm -f /tmp/test_queries.txt
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Processing 2 queries..." ]]
}

# ============================================================================
# API Status Tests  
# ============================================================================

@test "searxng::get_stats retrieves statistics" {
    run_with_mocks "searxng::get_stats" "export MOCK_API_RESPONSE='{\"stats\": \"test\"}'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "stats" ]]
}

@test "searxng::get_api_config retrieves configuration" {
    run_with_mocks "searxng::get_api_config" "export MOCK_API_RESPONSE='{\"config\": \"test\"}'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "config" ]]
}

@test "searxng::test_api performs comprehensive test" {
    run_with_mocks "searxng::test_api"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SearXNG API Test" ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "functions handle curl failures gracefully" {
    run_with_mocks "searxng::search 'test'" "export MOCK_CURL_SUCCESS=no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR" ]]
}

@test "functions handle missing jq gracefully" {
    run_with_mocks "searxng::search 'test'" "export MOCK_JQ_AVAILABLE=no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "jq is required" ]]
}

# Clean up any temporary files
teardown() {
    rm -f /tmp/test_queries.txt
    rm -f batch_*.json
    rm -f results_*.json
}