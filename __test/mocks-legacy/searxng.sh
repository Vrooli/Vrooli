#!/usr/bin/env bash
# SearXNG Mock System
# Comprehensive mock for SearXNG metasearch engine with Docker simulation
# Follows the standards established in http.sh and filesystem.sh

# Prevent duplicate loading in same shell, but allow state loading in subshells
if [[ "${SEARXNG_MOCK_LOADED:-}" == "true" ]]; then
    # Already loaded, but still load state if available
    if [[ -f "$SEARXNG_MOCK_STATE_FILE" && -s "$SEARXNG_MOCK_STATE_FILE" ]]; then
        _searxng_mock_load_state
    fi
    return 0
fi
export SEARXNG_MOCK_LOADED="true"

# ----------------------------
# State Management
# ----------------------------
declare -gA MOCK_SEARXNG_STATE=(
    [container_status]="not_installed"  # not_installed, stopped, running, unhealthy
    [health_status]="unknown"           # unknown, healthy, unhealthy
    [port]="8200"
    [base_url]="http://localhost:8200"
    [install_attempts]=0
    [start_attempts]=0
    [search_count]=0
    [last_query]=""
    [api_calls]=0
)

declare -gA MOCK_SEARXNG_SEARCH_RESULTS=()
declare -gA MOCK_SEARXNG_ENGINES=()
declare -gA MOCK_SEARXNG_ERRORS=()
declare -gA MOCK_SEARXNG_RESPONSE_TIMES=()
declare -gA MOCK_SEARXNG_CONFIG=()

# State file for subshell persistence
export SEARXNG_MOCK_STATE_FILE="${SEARXNG_MOCK_STATE_FILE:-/tmp/searxng_mock_state_$$}"

# ----------------------------
# State Persistence (for subshells)
# ----------------------------
_searxng_mock_save_state() {
    {
        # Declare associative arrays first
        echo "declare -gA MOCK_SEARXNG_STATE"
        echo "declare -gA MOCK_SEARXNG_SEARCH_RESULTS"
        echo "declare -gA MOCK_SEARXNG_ENGINES"
        echo "declare -gA MOCK_SEARXNG_ERRORS"
        echo "declare -gA MOCK_SEARXNG_RESPONSE_TIMES"
        echo "declare -gA MOCK_SEARXNG_CONFIG"
        echo
        
        # Save all state arrays with proper escaping for JSON content
        for key in "${!MOCK_SEARXNG_STATE[@]}"; do
            printf 'MOCK_SEARXNG_STATE[%s]=%q\n' "$key" "${MOCK_SEARXNG_STATE[$key]}"
        done
        for key in "${!MOCK_SEARXNG_SEARCH_RESULTS[@]}"; do
            # Use printf %q to properly quote/escape the JSON content
            printf 'MOCK_SEARXNG_SEARCH_RESULTS[%s]=%q\n' "$key" "${MOCK_SEARXNG_SEARCH_RESULTS[$key]}"
        done
        for key in "${!MOCK_SEARXNG_ENGINES[@]}"; do
            printf 'MOCK_SEARXNG_ENGINES[%s]=%q\n' "$key" "${MOCK_SEARXNG_ENGINES[$key]}"
        done
        for key in "${!MOCK_SEARXNG_ERRORS[@]}"; do
            printf 'MOCK_SEARXNG_ERRORS[%s]=%q\n' "$key" "${MOCK_SEARXNG_ERRORS[$key]}"
        done
        for key in "${!MOCK_SEARXNG_RESPONSE_TIMES[@]}"; do
            printf 'MOCK_SEARXNG_RESPONSE_TIMES[%s]=%q\n' "$key" "${MOCK_SEARXNG_RESPONSE_TIMES[$key]}"
        done
        for key in "${!MOCK_SEARXNG_CONFIG[@]}"; do
            printf 'MOCK_SEARXNG_CONFIG[%s]=%q\n' "$key" "${MOCK_SEARXNG_CONFIG[$key]}"
        done
    } > "$SEARXNG_MOCK_STATE_FILE"
}

_searxng_mock_load_state() {
    if [[ -f "$SEARXNG_MOCK_STATE_FILE" ]]; then
        # Use source to properly load arrays
        source "$SEARXNG_MOCK_STATE_FILE" 2>/dev/null || true
    fi
}

_searxng_mock_init_state_file() {
    mkdir -p "$(dirname "$SEARXNG_MOCK_STATE_FILE")"
    # Only initialize if file doesn't exist or is empty
    if [[ ! -f "$SEARXNG_MOCK_STATE_FILE" || ! -s "$SEARXNG_MOCK_STATE_FILE" ]]; then
        touch "$SEARXNG_MOCK_STATE_FILE"
        _searxng_mock_save_state
    fi
}

# Initialize state file (won't overwrite existing state)
_searxng_mock_init_state_file

# If state file exists and has content, load it
if [[ -f "$SEARXNG_MOCK_STATE_FILE" && -s "$SEARXNG_MOCK_STATE_FILE" ]]; then
    _searxng_mock_load_state
fi

# ----------------------------
# Utilities
# ----------------------------
_mock_searxng_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

_mock_searxng_response_id() {
    echo "srx$(date +%s)$(( RANDOM % 9999 ))"
}

_mock_searxng_generate_result() {
    local query="$1"
    local index="$2"
    local category="${3:-general}"
    
    # Escape JSON special characters
    local escaped_query="${query//\\/\\\\}"
    escaped_query="${escaped_query//\"/\\\"}"
    
    local title="Result $index for: $escaped_query"
    local url_part=$(echo "$query" | tr ' ' '-' | tr -cd '[:alnum:]-')
    [[ -z "$url_part" ]] && url_part="search"
    local url="https://example$index.com/$url_part"
    local content="This is a mock search result for '$escaped_query'. Result number $index from mock SearXNG engine."
    local engine="mock-engine-$((index % 3 + 1))"
    
    # Generate score ensuring it has leading zero
    local score
    score=$(echo "scale=2; 1.0 - $index * 0.1" | bc 2>/dev/null || echo "0.9")
    # Ensure leading zero for numbers like .5 -> 0.5
    if [[ "$score" =~ ^\. ]]; then
        score="0$score"
    fi

    cat <<EOF
{
  "title": "$title",
  "url": "$url",
  "content": "$content",
  "engine": "$engine",
  "score": $score,
  "category": "$category",
  "publishedDate": "$(_mock_searxng_timestamp)"
}
EOF
}

# ----------------------------
# Public API
# ----------------------------
mock::searxng::reset() {
    # Reset all state
    MOCK_SEARXNG_STATE=(
        [container_status]="not_installed"
        [health_status]="unknown"
        [port]="8200"
        [base_url]="http://localhost:8200"
        [install_attempts]=0
        [start_attempts]=0
        [search_count]=0
        [last_query]=""
        [api_calls]=0
    )
    
    MOCK_SEARXNG_SEARCH_RESULTS=()
    MOCK_SEARXNG_ENGINES=()
    MOCK_SEARXNG_ERRORS=()
    MOCK_SEARXNG_RESPONSE_TIMES=()
    MOCK_SEARXNG_CONFIG=()
    
    # Save state for subshells
    _searxng_mock_save_state
    
    echo "[MOCK] SearXNG state reset"
}

mock::searxng::set_container_status() {
    local status="$1"
    MOCK_SEARXNG_STATE[container_status]="$status"
    
    # Update health status based on container status
    case "$status" in
        "running")
            MOCK_SEARXNG_STATE[health_status]="healthy"
            ;;
        "unhealthy")
            MOCK_SEARXNG_STATE[health_status]="unhealthy"
            ;;
        *)
            MOCK_SEARXNG_STATE[health_status]="unknown"
            ;;
    esac
    
    _searxng_mock_save_state
    
    # Use centralized logging if available
    if command -v mock::log_state &>/dev/null; then
        mock::log_state "searxng_container" "status" "$status"
    fi
}

mock::searxng::set_search_result() {
    local query="$1"
    local result="$2"
    local key=$(echo "$query" | tr ' ' '_' | tr '[:upper:]' '[:lower:]')
    
    MOCK_SEARXNG_SEARCH_RESULTS["$key"]="$result"
    _searxng_mock_save_state
}

mock::searxng::set_response_time() {
    local endpoint="$1"
    local time_ms="$2"
    
    MOCK_SEARXNG_RESPONSE_TIMES["$endpoint"]="$time_ms"
    _searxng_mock_save_state
}

mock::searxng::inject_error() {
    local operation="$1"
    local error_type="$2"
    
    MOCK_SEARXNG_ERRORS["$operation"]="$error_type"
    _searxng_mock_save_state
}

mock::searxng::set_config() {
    local key="$1"
    local value="$2"
    
    MOCK_SEARXNG_CONFIG["$key"]="$value"
    _searxng_mock_save_state
}

# ----------------------------
# Core Command Mocks
# ----------------------------

# Mock the searxng::is_installed function
searxng::is_installed() {
    _searxng_mock_load_state
    
    # Use centralized logging and verification if available
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::is_installed" "$*"
    fi
    
    local status="${MOCK_SEARXNG_STATE[container_status]}"
    [[ "$status" != "not_installed" ]]
}

# Mock the searxng::is_running function
searxng::is_running() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::is_running" "$*"
    fi
    
    local status="${MOCK_SEARXNG_STATE[container_status]}"
    [[ "$status" == "running" || "$status" == "unhealthy" ]]
}

# Mock the searxng::is_healthy function
searxng::is_healthy() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::is_healthy" "$*"
    fi
    
    local health="${MOCK_SEARXNG_STATE[health_status]}"
    [[ "$health" == "healthy" ]]
}

# Mock the searxng::get_status function
searxng::get_status() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::get_status" "$*"
    fi
    
    local container_status="${MOCK_SEARXNG_STATE[container_status]}"
    local health_status="${MOCK_SEARXNG_STATE[health_status]}"
    
    case "$container_status" in
        "not_installed")
            echo "not_installed"
            ;;
        "stopped")
            echo "stopped"
            ;;
        "running")
            if [[ "$health_status" == "healthy" ]]; then
                echo "healthy"
            else
                echo "unhealthy"
            fi
            ;;
        "unhealthy")
            echo "unhealthy"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# Mock the searxng::install function
searxng::install() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::install" "$*"
    fi
    
    # Check for injected errors
    if [[ -n "${MOCK_SEARXNG_ERRORS[install]}" ]]; then
        local error_type="${MOCK_SEARXNG_ERRORS[install]}"
        echo "[MOCK] Installation failed: $error_type" >&2
        return 1
    fi
    
    # Increment install attempts
    local attempts="${MOCK_SEARXNG_STATE[install_attempts]:-0}"
    MOCK_SEARXNG_STATE[install_attempts]=$((attempts + 1))
    
    # Simulate installation
    echo "[MOCK] Installing SearXNG..."
    MOCK_SEARXNG_STATE[container_status]="stopped"
    MOCK_SEARXNG_STATE[health_status]="unknown"
    
    _searxng_mock_save_state
    
    echo "[MOCK] SearXNG installed successfully"
    return 0
}

# Mock the searxng::start_container function
searxng::start_container() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::start_container" "$*"
    fi
    
    # Check if installed
    if [[ "${MOCK_SEARXNG_STATE[container_status]}" == "not_installed" ]]; then
        echo "[MOCK] SearXNG is not installed" >&2
        return 1
    fi
    
    # Check for errors
    if [[ -n "${MOCK_SEARXNG_ERRORS[start]}" ]]; then
        local error_type="${MOCK_SEARXNG_ERRORS[start]}"
        echo "[MOCK] Start failed: $error_type" >&2
        return 1
    fi
    
    # Increment start attempts
    local attempts="${MOCK_SEARXNG_STATE[start_attempts]:-0}"
    MOCK_SEARXNG_STATE[start_attempts]=$((attempts + 1))
    
    # Start container
    echo "[MOCK] Starting SearXNG container..."
    MOCK_SEARXNG_STATE[container_status]="running"
    MOCK_SEARXNG_STATE[health_status]="healthy"
    
    _searxng_mock_save_state
    
    echo "[MOCK] SearXNG started successfully"
    return 0
}

# Mock the searxng::stop_container function
searxng::stop_container() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::stop_container" "$*"
    fi
    
    if [[ "${MOCK_SEARXNG_STATE[container_status]}" == "not_installed" ]]; then
        echo "[MOCK] SearXNG is not installed" >&2
        return 1
    fi
    
    echo "[MOCK] Stopping SearXNG container..."
    MOCK_SEARXNG_STATE[container_status]="stopped"
    MOCK_SEARXNG_STATE[health_status]="unknown"
    
    _searxng_mock_save_state
    
    echo "[MOCK] SearXNG stopped"
    return 0
}

# Mock the searxng::search function
searxng::search() {
    _searxng_mock_load_state
    
    local query="$1"
    local format="${2:-json}"
    local category="${3:-general}"
    local language="${4:-en}"
    local pageno="${5:-1}"
    local safesearch="${6:-1}"
    local time_range="${7:-}"
    local output_format="${8:-json}"
    local limit="${9:-}"
    local save_file="${10:-}"
    local append_file="${11:-}"
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::search" "query=$query format=$format category=$category"
    fi
    
    # Check if healthy
    if [[ "${MOCK_SEARXNG_STATE[health_status]}" != "healthy" ]]; then
        echo "[MOCK] SearXNG is not running or healthy" >&2
        return 1
    fi
    
    # Update state
    local count="${MOCK_SEARXNG_STATE[search_count]}"
    MOCK_SEARXNG_STATE[search_count]=$((count + 1))
    MOCK_SEARXNG_STATE[last_query]="$query"
    
    # Check for custom results  
    local query_key
    if [[ -z "$query" ]]; then
        query_key="_empty_query_"
    else
        query_key=$(echo "$query" | tr ' ' '_' | tr '[:upper:]' '[:lower:]')
    fi
    local custom_result="${MOCK_SEARXNG_SEARCH_RESULTS[$query_key]:-}"
    
    local search_result
    local num_results=0
    if [[ -n "$custom_result" ]]; then
        search_result="$custom_result"
    else
        # Generate mock results
        local results_json="["
        num_results=$((RANDOM % 10 + 1))
        [[ -n "$limit" ]] && num_results=$(( limit < num_results ? limit : num_results ))
        
        # Handle empty query properly for JSON
        local escaped_query="${query//\"/\\\"}"
        
        for ((i=1; i<=num_results; i++)); do
            [[ $i -gt 1 ]] && results_json+=","
            results_json+=$(_mock_searxng_generate_result "$escaped_query" "$i" "$category")
        done
        results_json+="]"
        
        search_result=$(cat <<EOF
{
  "query": "$escaped_query",
  "number_of_results": $num_results,
  "results": $results_json,
  "suggestions": [],
  "answers": [],
  "infoboxes": [],
  "paging": true,
  "results_on_new_tab": false,
  "search": {
    "q": "$escaped_query",
    "categories": ["$category"],
    "language": "$language",
    "pageno": $pageno,
    "safesearch": $safesearch,
    "time_range": "$time_range"
  }
}
EOF
)
    fi
    
    # Handle different output formats
    case "$format" in
        "json")
            # Already in JSON format
            ;;
        "xml")
            search_result='<?xml version="1.0" encoding="UTF-8"?><results><query>'$query'</query><count>'$num_results'</count></results>'
            ;;
        "csv")
            search_result="title,url,content,engine"$'\n'"Mock Result,$query,https://example.com,mock-engine"
            ;;
        "rss")
            search_result='<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>Search: '$query'</title></channel></rss>'
            ;;
    esac
    
    # Save to file if requested
    if [[ -n "$save_file" ]]; then
        echo "$search_result" > "$save_file"
        echo "[MOCK] Results saved to: $save_file"
    elif [[ -n "$append_file" ]]; then
        # For JSONL format, compact the JSON to a single line
        if command -v jq >/dev/null 2>&1 && [[ "$format" == "json" ]]; then
            # Compact JSON for JSONL format
            echo "$search_result" | jq -c '.' >> "$append_file"
        else
            echo "$search_result" >> "$append_file"
        fi
        echo "[MOCK] Results appended to: $append_file"
    else
        echo "$search_result"
    fi
    
    _searxng_mock_save_state
    return 0
}

# Mock the searxng::get_stats function
searxng::get_stats() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::get_stats" "$*"
    fi
    
    if [[ "${MOCK_SEARXNG_STATE[health_status]}" != "healthy" ]]; then
        echo "[MOCK] SearXNG is not running or healthy" >&2
        return 1
    fi
    
    # Increment API calls
    local api_calls="${MOCK_SEARXNG_STATE[api_calls]}"
    MOCK_SEARXNG_STATE[api_calls]=$((api_calls + 1))
    
    # Generate mock stats
    cat <<EOF
{
  "engines": [
    {"name": "google", "time": 0.523, "errors": 0},
    {"name": "duckduckgo", "time": 0.412, "errors": 0},
    {"name": "bing", "time": 0.634, "errors": 1}
  ],
  "timing": {
    "total": 1.234,
    "engine": 0.634,
    "processing": 0.600
  },
  "search_count": ${MOCK_SEARXNG_STATE[search_count]},
  "uptime": "12h 34m 56s",
  "version": "1.1.0"
}
EOF
    
    _searxng_mock_save_state
    return 0
}

# Mock the searxng::get_api_config function
searxng::get_api_config() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::get_api_config" "$*"
    fi
    
    if [[ "${MOCK_SEARXNG_STATE[health_status]}" != "healthy" ]]; then
        echo "[MOCK] SearXNG is not running or healthy" >&2
        return 1
    fi
    
    # Generate mock config
    cat <<EOF
{
  "categories": ["general", "images", "videos", "news", "music", "files", "science"],
  "engines": [
    {"name": "google", "enabled": true, "shortcut": "go"},
    {"name": "duckduckgo", "enabled": true, "shortcut": "ddg"},
    {"name": "bing", "enabled": false, "shortcut": "bi"}
  ],
  "locales": ["en", "de", "fr", "es", "it"],
  "doi_resolvers": ["sci-hub.tw"],
  "default_doi_resolver": "sci-hub.tw",
  "public_instance": false,
  "safe_search": ${MOCK_SEARXNG_CONFIG[safe_search]:-1},
  "autocomplete": "google",
  "default_lang": "${MOCK_SEARXNG_CONFIG[default_lang]:-en}",
  "server": {
    "port": ${MOCK_SEARXNG_STATE[port]},
    "bind_address": "0.0.0.0",
    "secret_key": "mock-secret-key",
    "base_url": "${MOCK_SEARXNG_STATE[base_url]}"
  }
}
EOF
    
    return 0
}

# Mock benchmark function
searxng::benchmark() {
    _searxng_mock_load_state
    
    local num_queries="${1:-10}"
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::benchmark" "num_queries=$num_queries"
    fi
    
    if [[ "${MOCK_SEARXNG_STATE[health_status]}" != "healthy" ]]; then
        echo "[MOCK] SearXNG is not running or healthy" >&2
        return 1
    fi
    
    echo "[MOCK] Running SearXNG performance benchmark..."
    echo "[MOCK] Number of test queries: $num_queries"
    echo
    echo "Running benchmark..."
    
    local total_time=0
    for ((i=1; i<=num_queries; i++)); do
        local response_time=$(echo "scale=3; 0.1 + $RANDOM / 32768 * 0.9" | bc 2>/dev/null || echo "0.5")
        total_time=$(echo "$total_time + $response_time" | bc 2>/dev/null || echo "$total_time")
        printf "  Query %2d: ✅ %.3fs - test query %d\n" "$i" "$response_time" "$i"
    done
    
    local avg_time=$(echo "scale=3; $total_time / $num_queries" | bc 2>/dev/null || echo "0.5")
    
    echo
    echo "Benchmark Results:"
    echo "  Total queries: $num_queries"
    echo "  Successful: $num_queries"
    echo "  Failed: 0"
    echo "  Average response time: ${avg_time}s"
    echo "  Total time: ${total_time}s"
    echo "  Performance: ✅ Excellent (< 1s average)"
    
    return 0
}

# Mock test_api function
searxng::test_api() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::test_api" "$*"
    fi
    
    if [[ "${MOCK_SEARXNG_STATE[health_status]}" != "healthy" ]]; then
        echo "[MOCK] SearXNG is not running or healthy" >&2
        return 1
    fi
    
    echo "[MOCK] SearXNG API Test"
    echo "Testing /stats endpoint..."
    echo "  ✅ /stats endpoint responding"
    echo "Testing /config endpoint..."
    echo "  ✅ /config endpoint responding"
    echo "Testing /search endpoint..."
    echo "  ✅ /search endpoint responding"
    echo "Testing response formats..."
    echo "  ✅ json format working"
    echo "  ✅ xml format working"
    echo "  ✅ csv format working"
    echo "Testing search categories..."
    echo "  ✅ general category working"
    echo "  ✅ images category working"
    echo "  ✅ news category working"
    echo
    echo "[MOCK] All API tests passed"
    
    return 0
}

# Mock show_status function
searxng::show_status() {
    _searxng_mock_load_state
    
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "searxng::show_status" "$*"
    fi
    
    echo "[MOCK] SearXNG Status Report"
    echo "Status: $(searxng::get_status)"
    echo "Port: ${MOCK_SEARXNG_STATE[port]}"
    echo "Base URL: ${MOCK_SEARXNG_STATE[base_url]}"
    echo "Container: searxng"
    echo "Health: ${MOCK_SEARXNG_STATE[health_status]}"
    echo "Search Count: ${MOCK_SEARXNG_STATE[search_count]}"
    echo "Last Query: ${MOCK_SEARXNG_STATE[last_query]}"
    echo "API Calls: ${MOCK_SEARXNG_STATE[api_calls]}"
    
    return 0
}

# ----------------------------
# Test Helper Functions
# ----------------------------
mock::searxng::assert::is_installed() {
    _searxng_mock_load_state
    
    if ! searxng::is_installed; then
        echo "ASSERTION FAILED: SearXNG is not installed" >&2
        return 1
    fi
    return 0
}

mock::searxng::assert::is_running() {
    _searxng_mock_load_state
    
    if ! searxng::is_running; then
        echo "ASSERTION FAILED: SearXNG is not running" >&2
        return 1
    fi
    return 0
}

mock::searxng::assert::is_healthy() {
    _searxng_mock_load_state
    
    if ! searxng::is_healthy; then
        echo "ASSERTION FAILED: SearXNG is not healthy" >&2
        return 1
    fi
    return 0
}

mock::searxng::assert::search_count() {
    _searxng_mock_load_state
    
    local expected="$1"
    local actual="${MOCK_SEARXNG_STATE[search_count]}"
    
    if [[ "$actual" != "$expected" ]]; then
        echo "ASSERTION FAILED: Search count is $actual, expected $expected" >&2
        return 1
    fi
    return 0
}

mock::searxng::assert::last_query() {
    _searxng_mock_load_state
    
    local expected="$1"
    local actual="${MOCK_SEARXNG_STATE[last_query]}"
    
    if [[ "$actual" != "$expected" ]]; then
        echo "ASSERTION FAILED: Last query is '$actual', expected '$expected'" >&2
        return 1
    fi
    return 0
}

mock::searxng::get::search_count() {
    _searxng_mock_load_state
    echo "${MOCK_SEARXNG_STATE[search_count]}"
}

mock::searxng::get::container_status() {
    _searxng_mock_load_state
    echo "${MOCK_SEARXNG_STATE[container_status]}"
}

mock::searxng::debug::dump_state() {
    _searxng_mock_load_state
    
    echo "=== SearXNG Mock State Dump ==="
    echo "Container Status: ${MOCK_SEARXNG_STATE[container_status]}"
    echo "Health Status: ${MOCK_SEARXNG_STATE[health_status]}"
    echo "Port: ${MOCK_SEARXNG_STATE[port]}"
    echo "Base URL: ${MOCK_SEARXNG_STATE[base_url]}"
    echo "Install Attempts: ${MOCK_SEARXNG_STATE[install_attempts]}"
    echo "Start Attempts: ${MOCK_SEARXNG_STATE[start_attempts]}"
    echo "Search Count: ${MOCK_SEARXNG_STATE[search_count]}"
    echo "Last Query: ${MOCK_SEARXNG_STATE[last_query]}"
    echo "API Calls: ${MOCK_SEARXNG_STATE[api_calls]}"
    
    if [[ ${#MOCK_SEARXNG_ERRORS[@]} -gt 0 ]]; then
        echo "Errors:"
        for key in "${!MOCK_SEARXNG_ERRORS[@]}"; do
            echo "  $key: ${MOCK_SEARXNG_ERRORS[$key]}"
        done
    fi
    
    if [[ ${#MOCK_SEARXNG_CONFIG[@]} -gt 0 ]]; then
        echo "Config:"
        for key in "${!MOCK_SEARXNG_CONFIG[@]}"; do
            echo "  $key: ${MOCK_SEARXNG_CONFIG[$key]}"
        done
    fi
    
    echo "=========================="
}

# ----------------------------
# Scenario Builders
# ----------------------------
mock::searxng::scenario::healthy_instance() {
    mock::searxng::reset
    mock::searxng::set_container_status "running"
    MOCK_SEARXNG_STATE[health_status]="healthy"
    _searxng_mock_save_state
    echo "[MOCK] Created healthy SearXNG instance scenario"
}

mock::searxng::scenario::unhealthy_instance() {
    mock::searxng::reset
    mock::searxng::set_container_status "unhealthy"
    MOCK_SEARXNG_STATE[health_status]="unhealthy"
    _searxng_mock_save_state
    echo "[MOCK] Created unhealthy SearXNG instance scenario"
}

mock::searxng::scenario::not_installed() {
    mock::searxng::reset
    echo "[MOCK] Created not installed scenario"
}

mock::searxng::scenario::with_search_results() {
    mock::searxng::scenario::healthy_instance
    
    # Add some predefined search results
    mock::searxng::set_search_result "test" '{"query":"test","number_of_results":3,"results":[{"title":"Test Result 1","url":"https://test1.com","content":"First test result","engine":"google"}]}'
    mock::searxng::set_search_result "docker" '{"query":"docker","number_of_results":5,"results":[{"title":"Docker Documentation","url":"https://docs.docker.com","content":"Official Docker docs","engine":"duckduckgo"}]}'
    
    echo "[MOCK] Added predefined search results"
}

# ----------------------------
# Export Functions
# ----------------------------
export -f searxng::is_installed searxng::is_running searxng::is_healthy searxng::get_status
export -f searxng::install searxng::start_container searxng::stop_container
export -f searxng::search searxng::get_stats searxng::get_api_config
export -f searxng::benchmark searxng::test_api searxng::show_status

export -f _mock_searxng_timestamp _mock_searxng_response_id _mock_searxng_generate_result
export -f _searxng_mock_save_state _searxng_mock_load_state _searxng_mock_init_state_file

export -f mock::searxng::reset mock::searxng::set_container_status
export -f mock::searxng::set_search_result mock::searxng::set_response_time
export -f mock::searxng::inject_error mock::searxng::set_config

export -f mock::searxng::assert::is_installed mock::searxng::assert::is_running
export -f mock::searxng::assert::is_healthy mock::searxng::assert::search_count
export -f mock::searxng::assert::last_query

export -f mock::searxng::get::search_count mock::searxng::get::container_status
export -f mock::searxng::debug::dump_state

export -f mock::searxng::scenario::healthy_instance mock::searxng::scenario::unhealthy_instance
export -f mock::searxng::scenario::not_installed mock::searxng::scenario::with_search_results

echo "[MOCK] SearXNG mocks loaded successfully"