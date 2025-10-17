#!/usr/bin/env bash
# SearXNG Mock - Tier 2 (Stateful)
# 
# Provides stateful SearXNG search engine mocking for testing:
# - Search query execution
# - Engine management
# - Results caching
# - Settings configuration
# - Error injection for resilience testing
#
# Coverage: ~80% of common SearXNG operations in 400 lines

# === Configuration ===
declare -gA SEARXNG_SEARCHES=()           # Query -> "results|timestamp"
declare -gA SEARXNG_ENGINES=()            # Engine_name -> "enabled|timeout"
declare -gA SEARXNG_CACHE=()              # Cache_key -> "result|expiry"
declare -gA SEARXNG_CONFIG=(              # Service configuration
    [status]="running"
    [port]="8080"
    [default_lang]="en"
    [safe_search]="moderate"
    [error_mode]=""
    [version]="2023.11.1"
)

# Debug mode
declare -g SEARXNG_DEBUG="${SEARXNG_DEBUG:-}"

# === Helper Functions ===
searxng_debug() {
    [[ -n "$SEARXNG_DEBUG" ]] && echo "[MOCK:SEARXNG] $*" >&2
}

searxng_check_error() {
    case "${SEARXNG_CONFIG[error_mode]}" in
        "service_down")
            echo "Error: SearXNG service is not running" >&2
            return 1
            ;;
        "no_results")
            echo "Error: No results found" >&2
            return 1
            ;;
        "engine_error")
            echo "Error: Search engine failed" >&2
            return 1
            ;;
    esac
    return 0
}

# === Main SearXNG Command ===
searxng() {
    searxng_debug "searxng called with: $*"
    
    if ! searxng_check_error; then
        return $?
    fi
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        search)
            searxng_cmd_search "$@"
            ;;
        engines)
            searxng_cmd_engines "$@"
            ;;
        settings)
            searxng_cmd_settings "$@"
            ;;
        cache)
            searxng_cmd_cache "$@"
            ;;
        status)
            searxng_cmd_status "$@"
            ;;
        start|stop|restart)
            searxng_cmd_service "$command" "$@"
            ;;
        *)
            echo "SearXNG CLI - Privacy-respecting Search Engine"
            echo "Commands:"
            echo "  search   - Perform search"
            echo "  engines  - Manage search engines"
            echo "  settings - Configure settings"
            echo "  cache    - Manage cache"
            echo "  status   - Show service status"
            echo "  start    - Start service"
            echo "  stop     - Stop service"
            echo "  restart  - Restart service"
            ;;
    esac
}

# === Search Command ===
searxng_cmd_search() {
    local query="$*"
    
    [[ -z "$query" ]] && { echo "Error: search query required" >&2; return 1; }
    
    # Check cache
    if [[ -n "${SEARXNG_CACHE[$query]}" ]]; then
        local cache_data="${SEARXNG_CACHE[$query]}"
        IFS='|' read -r result expiry <<< "$cache_data"
        if [[ $(date +%s) -lt $expiry ]]; then
            echo "Results (cached):"
            echo "$result"
            return 0
        fi
    fi
    
    # Store search
    SEARXNG_SEARCHES[$query]="3|$(date +%s)"
    
    echo "Searching for: $query"
    echo ""
    echo "Results:"
    echo "1. Example Result - $query"
    echo "   https://example.com/$query"
    echo "   This is a sample result for your search query."
    echo ""
    echo "2. Another Result - Learn about $query"
    echo "   https://wiki.example.org/$query"
    echo "   Comprehensive information about $query."
    echo ""
    echo "3. Third Result - $query Documentation"
    echo "   https://docs.example.com/$query"
    echo "   Official documentation and guides."
    
    # Cache result
    local cache_expiry=$(($(date +%s) + 300))
    SEARXNG_CACHE[$query]="3 results found|$cache_expiry"
}

# === Engines Management ===
searxng_cmd_engines() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Search engines:"
            if [[ ${#SEARXNG_ENGINES[@]} -eq 0 ]]; then
                echo "  google - Enabled (2s timeout)"
                echo "  duckduckgo - Enabled (2s timeout)"
                echo "  bing - Enabled (3s timeout)"
                echo "  wikipedia - Enabled (2s timeout)"
            else
                for engine in "${!SEARXNG_ENGINES[@]}"; do
                    local data="${SEARXNG_ENGINES[$engine]}"
                    IFS='|' read -r enabled timeout <<< "$data"
                    echo "  $engine - $enabled (${timeout}s timeout)"
                done
            fi
            ;;
        enable)
            local engine="${1:-}"
            [[ -z "$engine" ]] && { echo "Error: engine name required" >&2; return 1; }
            SEARXNG_ENGINES[$engine]="Enabled|2"
            echo "Engine '$engine' enabled"
            ;;
        disable)
            local engine="${1:-}"
            [[ -z "$engine" ]] && { echo "Error: engine name required" >&2; return 1; }
            SEARXNG_ENGINES[$engine]="Disabled|2"
            echo "Engine '$engine' disabled"
            ;;
        *)
            echo "Usage: searxng engines {list|enable|disable} [engine]"
            return 1
            ;;
    esac
}

# === Settings Command ===
searxng_cmd_settings() {
    local key="${1:-}"
    local value="${2:-}"
    
    if [[ -z "$key" ]]; then
        echo "Settings:"
        for k in "${!SEARXNG_CONFIG[@]}"; do
            [[ "$k" != "error_mode" ]] && echo "  $k: ${SEARXNG_CONFIG[$k]}"
        done
    elif [[ -z "$value" ]]; then
        echo "${SEARXNG_CONFIG[$key]:-}"
    else
        SEARXNG_CONFIG[$key]="$value"
        echo "Set $key = $value"
    fi
}

# === Cache Management ===
searxng_cmd_cache() {
    local action="${1:-list}"
    
    case "$action" in
        list)
            echo "Cache entries: ${#SEARXNG_CACHE[@]}"
            ;;
        clear)
            SEARXNG_CACHE=()
            echo "Cache cleared"
            ;;
        *)
            echo "Usage: searxng cache {list|clear}"
            return 1
            ;;
    esac
}

# === Status Command ===
searxng_cmd_status() {
    echo "SearXNG Status"
    echo "=============="
    echo "Service: ${SEARXNG_CONFIG[status]}"
    echo "Port: ${SEARXNG_CONFIG[port]}"
    echo "Language: ${SEARXNG_CONFIG[default_lang]}"
    echo "Safe Search: ${SEARXNG_CONFIG[safe_search]}"
    echo "Version: ${SEARXNG_CONFIG[version]}"
    echo ""
    echo "Searches: ${#SEARXNG_SEARCHES[@]}"
    echo "Cache Entries: ${#SEARXNG_CACHE[@]}"
    echo "Engines: ${#SEARXNG_ENGINES[@]}"
}

# === Service Management ===
searxng_cmd_service() {
    local action="$1"
    
    case "$action" in
        start)
            if [[ "${SEARXNG_CONFIG[status]}" == "running" ]]; then
                echo "SearXNG is already running"
            else
                SEARXNG_CONFIG[status]="running"
                echo "SearXNG started on port ${SEARXNG_CONFIG[port]}"
            fi
            ;;
        stop)
            SEARXNG_CONFIG[status]="stopped"
            echo "SearXNG stopped"
            ;;
        restart)
            SEARXNG_CONFIG[status]="stopped"
            SEARXNG_CONFIG[status]="running"
            echo "SearXNG restarted"
            ;;
    esac
}

# === HTTP API Mock ===
curl() {
    searxng_debug "curl called with: $*"
    
    local url="" method="GET" data=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X) method="$2"; shift 2 ;;
            -d|--data) data="$2"; shift 2 ;;
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    if [[ "$url" =~ localhost:8080 || "$url" =~ 127.0.0.1:8080 ]]; then
        searxng_handle_api "$method" "$url" "$data"
        return $?
    fi
    
    echo "curl: Not a SearXNG endpoint"
    return 0
}

searxng_handle_api() {
    local method="$1" url="$2" data="$3"
    
    case "$url" in
        */search)
            echo '{"results":[{"title":"Result 1","url":"https://example.com","content":"Sample content"}],"number_of_results":3}'
            ;;
        */config)
            echo '{"engines":["google","duckduckgo","bing"],"language":"'${SEARXNG_CONFIG[default_lang]}'"}'
            ;;
        */health)
            if [[ "${SEARXNG_CONFIG[status]}" == "running" ]]; then
                echo '{"status":"healthy","version":"'${SEARXNG_CONFIG[version]}'"}'
            else
                echo '{"status":"unhealthy","error":"Service not running"}'
            fi
            ;;
        *)
            echo '{"status":"ok","version":"'${SEARXNG_CONFIG[version]}'"}'
            ;;
    esac
}

# === Mock Control Functions ===
searxng_mock_reset() {
    searxng_debug "Resetting mock state"
    
    SEARXNG_SEARCHES=()
    SEARXNG_ENGINES=()
    SEARXNG_CACHE=()
    SEARXNG_CONFIG[error_mode]=""
    SEARXNG_CONFIG[status]="running"
    
    # Initialize defaults
    searxng_mock_init_defaults
}

searxng_mock_init_defaults() {
    SEARXNG_ENGINES["google"]="Enabled|2"
    SEARXNG_ENGINES["duckduckgo"]="Enabled|2"
    SEARXNG_ENGINES["bing"]="Enabled|3"
}

searxng_mock_set_error() {
    SEARXNG_CONFIG[error_mode]="$1"
    searxng_debug "Set error mode: $1"
}

searxng_mock_dump_state() {
    echo "=== SearXNG Mock State ==="
    echo "Status: ${SEARXNG_CONFIG[status]}"
    echo "Port: ${SEARXNG_CONFIG[port]}"
    echo "Searches: ${#SEARXNG_SEARCHES[@]}"
    echo "Cache: ${#SEARXNG_CACHE[@]}"
    echo "Engines: ${#SEARXNG_ENGINES[@]}"
    echo "Error Mode: ${SEARXNG_CONFIG[error_mode]:-none}"
    echo "====================="
}

# === Convention-based Test Functions ===
test_searxng_connection() {
    searxng_debug "Testing connection..."
    
    local result
    result=$(curl -s http://localhost:8080/health 2>&1)
    
    if [[ "$result" =~ "status" ]]; then
        searxng_debug "Connection test passed"
        return 0
    else
        searxng_debug "Connection test failed"
        return 1
    fi
}

test_searxng_health() {
    searxng_debug "Testing health..."
    
    test_searxng_connection || return 1
    
    searxng engines list >/dev/null 2>&1 || return 1
    searxng cache list >/dev/null 2>&1 || return 1
    
    searxng_debug "Health test passed"
    return 0
}

test_searxng_basic() {
    searxng_debug "Testing basic operations..."
    
    searxng search "test query" >/dev/null 2>&1 || return 1
    searxng engines enable startpage >/dev/null 2>&1 || return 1
    searxng settings default_lang fr >/dev/null 2>&1 || return 1
    
    searxng_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f searxng curl
export -f test_searxng_connection test_searxng_health test_searxng_basic
export -f searxng_mock_reset searxng_mock_set_error searxng_mock_dump_state
export -f searxng_debug searxng_check_error

# Initialize with defaults
searxng_mock_reset
searxng_debug "SearXNG Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
