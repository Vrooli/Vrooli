#!/bin/bash

# Set up test environment
export SEARXNG_TEST_MODE=yes
export SEARXNG_PORT=8100
export SEARXNG_BASE_URL='http://localhost:8100'
export SEARXNG_DATA_DIR='$HOME/.searxng'
export SEARXNG_API_TIMEOUT=10
export SEARXNG_DEFAULT_LANG=en

# Mock variables
export MOCK_SEARXNG_HEALTHY=yes
export MOCK_CURL_SUCCESS=yes
export MOCK_JQ_AVAILABLE=yes
export MOCK_API_RESPONSE='{"results": [{"title": "Test Result", "url": "http://example.com", "content": "Test content"}], "number_of_results": 1}'

# Mock logging functions
log::info() { echo "INFO: $*"; }
log::error() { echo "ERROR: $*" >&2; return 1; }
log::success() { echo "SUCCESS: $*"; }
log::warn() { echo "WARNING: $*" >&2; }
log::debug() { :; }
log::header() { echo "=== $* ==="; }

# Mock other required functions
searxng::is_healthy() { [[ "$MOCK_SEARXNG_HEALTHY" == "yes" ]]; }
searxng::validate_url() { return 0; }

# Mock command function
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

# Mock jq function (passthrough to real jq if available)
jq() {
    if [[ "$MOCK_JQ_AVAILABLE" == "yes" ]]; then
        # Simulate jq parsing for our mock data
        case "$1" in
            "-r")
                shift
                case "$1" in
                    ".results[0].url // empty")
                        echo "http://example.com"
                        ;;
                    ".results[0].title")
                        echo "Test Result"
                        ;;
                    *)
                        echo "$MOCK_API_RESPONSE"
                        ;;
                esac
                ;;
            *)
                echo "$MOCK_API_RESPONSE"
                ;;
        esac
        return 0
    else
        echo "jq: command not found" >&2
        return 127
    fi
}

# Source the script
source lib/api.sh || { echo "Failed to source api.sh" >&2; exit 1; }

# Test lucky search
echo "Testing searxng::lucky..."

# Test the curl and jq pipeline directly
echo -e "\nTest 1: Direct curl output:"
curl -sf --max-time 10 "http://localhost:8100/search?q=test&format=json"

echo -e "\n\nTest 2: Curl piped to jq:"
curl -sf --max-time 10 "http://localhost:8100/search?q=test&format=json" | jq -r '.results[0].url // empty' 2>&1

echo -e "\n\nTest 3: Full lucky search:"
searxng::lucky 'test query'
echo "Exit code: $?"