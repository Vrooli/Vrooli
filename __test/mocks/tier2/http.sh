#!/usr/bin/env bash
# HTTP Mock - Tier 2 (Stateful)
# 
# Provides stateful HTTP client mocking for testing:
# - HTTP command interception (curl, wget, nc)
# - Endpoint state management (healthy, unhealthy, timeout)
# - Custom responses, status codes, and headers
# - Error injection for resilience testing
# - Call tracking and basic analytics
#
# Coverage: ~80% of common HTTP testing operations in 500 lines

# === Configuration ===
declare -gA HTTP_RESPONSES=()           # URL -> response_body
declare -gA HTTP_STATUS_CODES=()        # URL -> status_code
declare -gA HTTP_CALL_COUNT=()          # URL -> call_count
declare -gA HTTP_HEADERS=()             # URL -> "header1:value1|header2:value2"
declare -gA HTTP_METHODS=()             # URL -> allowed_methods (comma separated)
declare -gA HTTP_CONFIG=(               # Global configuration
    [default_status]="200"
    [default_response]="{\"status\":\"ok\"}"
    [timeout]="10"
    [error_mode]=""
    [debug]="false"
)

# Debug mode
declare -g HTTP_DEBUG="${HTTP_DEBUG:-}"

# === Helper Functions ===
http_debug() {
    [[ -n "$HTTP_DEBUG" || "${HTTP_CONFIG[debug]}" == "true" ]] && echo "[MOCK:HTTP] $*" >&2
}

http_check_error() {
    case "${HTTP_CONFIG[error_mode]}" in
        "connection_refused")
            echo "curl: (7) Failed to connect to localhost port 80: Connection refused" >&2
            return 7
            ;;
        "timeout")
            echo "curl: (28) Operation timed out" >&2
            return 28
            ;;
        "dns_failure")
            echo "curl: (6) Could not resolve host: localhost" >&2
            return 6
            ;;
        "ssl_error")
            echo "curl: (60) SSL certificate problem: unable to get local issuer certificate" >&2
            return 60
            ;;
        "network_unreachable")
            echo "curl: (7) Network is unreachable" >&2
            return 7
            ;;
    esac
    return 0
}

http_normalize_url() {
    local url="$1"
    # Remove trailing slashes and normalize
    url="${url%/}"
    echo "$url"
}

http_extract_base_url() {
    local url="$1"
    # Extract base URL for pattern matching
    if [[ "$url" =~ ^(https?://[^/]+) ]]; then
        echo "${BASH_REMATCH[1]}"
    else
        echo "$url"
    fi
}

http_increment_call_count() {
    local url="$1"
    local count="${HTTP_CALL_COUNT[$url]:-0}"
    ((count++))
    HTTP_CALL_COUNT[$url]="$count"
    http_debug "Call count for $url: $count"
}

# === HTTP Command Mocks ===

# Main curl mock
curl() {
    http_debug "curl called with: $*"
    
    if ! http_check_error; then
        return $?
    fi
    
    # Parse curl arguments (simplified but essential)
    local url="" method="GET" data="" output_file="" headers=()
    local silent=false fail_on_error=false follow_redirects=false
    local write_out="" include_headers=false max_time=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X|--request) method="$2"; shift 2 ;;
            -d|--data) data="$2"; method="POST"; shift 2 ;;
            -H|--header) headers+=("$2"); shift 2 ;;
            -s|--silent) silent=true; shift ;;
            -f|--fail) fail_on_error=true; shift ;;
            -L|--location) follow_redirects=true; shift ;;
            -o|--output) output_file="$2"; shift 2 ;;
            -w|--write-out) write_out="$2"; shift 2 ;;
            -I|--head) method="HEAD"; shift ;;
            -i|--include) include_headers=true; shift ;;
            -m|--max-time) max_time="$2"; shift 2 ;;
            --connect-timeout) shift 2 ;; # Ignore but consume
            -k|--insecure) shift ;; # Ignore SSL verification
            http*) url="$1"; shift ;;
            *) shift ;; # Ignore unknown options
        esac
    done
    
    if [[ -z "$url" ]]; then
        echo "curl: try 'curl --help' for more information" >&2
        return 2
    fi
    
    # Handle HTTP request
    http_handle_request "$method" "$url" "$data" "$output_file" "$silent" "$fail_on_error" "$write_out" "$include_headers"
}

# Simplified wget mock
wget() {
    http_debug "wget called with: $*"
    
    if ! http_check_error; then
        return $?
    fi
    
    local url="" output_file="" quiet=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -O) output_file="$2"; shift 2 ;;
            -q|--quiet) quiet=true; shift ;;
            -o) shift 2 ;; # Ignore log file
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$url" ]]; then
        echo "wget: missing URL" >&2
        return 1
    fi
    
    # Simulate wget behavior (always success for basic cases)
    local normalized_url=$(http_normalize_url "$url")
    http_increment_call_count "$normalized_url"
    
    local response="${HTTP_RESPONSES[$normalized_url]:-${HTTP_CONFIG[default_response]}}"
    local status="${HTTP_STATUS_CODES[$normalized_url]:-${HTTP_CONFIG[default_status]}}"
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    elif [[ "$quiet" != "true" ]]; then
        echo "$response"
    fi
    
    # Return success for 2xx status codes
    [[ "$status" =~ ^2[0-9][0-9]$ ]] && return 0 || return 1
}

# Network connectivity mock
nc() {
    http_debug "nc called with: $*"
    
    if ! http_check_error; then
        return $?
    fi
    
    local host="" port="" timeout=10
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -w) timeout="$2"; shift 2 ;;
            -z) shift ;; # Zero-I/O mode (just check connection)
            -*) shift ;;
            *) 
                if [[ -z "$host" ]]; then
                    host="$1"
                elif [[ -z "$port" ]]; then
                    port="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Simulate basic connectivity check
    local base_url="http://$host:$port"
    http_increment_call_count "$base_url"
    
    # Check if we have a configured response (indicates "service is up")
    if [[ -n "${HTTP_RESPONSES[$base_url]}" || -n "${HTTP_STATUS_CODES[$base_url]}" ]]; then
        http_debug "nc: Connection to $host:$port successful"
        return 0
    else
        http_debug "nc: Connection to $host:$port failed"
        return 1
    fi
}

# === HTTP Request Handler ===
http_handle_request() {
    local method="$1" url="$2" data="$3" output_file="$4" silent="$5" fail_on_error="$6" write_out="$7" include_headers="$8"
    
    local normalized_url=$(http_normalize_url "$url")
    http_increment_call_count "$normalized_url"
    
    http_debug "Processing: $method $normalized_url"
    
    # Check method restrictions
    local allowed_methods="${HTTP_METHODS[$normalized_url]}"
    if [[ -n "$allowed_methods" && ! "$allowed_methods" =~ $method ]]; then
        local status="405"
        local response="{\"error\":\"Method Not Allowed\",\"method\":\"$method\"}"
        http_output_response "$response" "$status" "" "$output_file" "$silent" "$fail_on_error" "$write_out" "$include_headers"
        return 0
    fi
    
    # Get response configuration
    local response="${HTTP_RESPONSES[$normalized_url]}"
    local status="${HTTP_STATUS_CODES[$normalized_url]:-${HTTP_CONFIG[default_status]}}"
    local headers="${HTTP_HEADERS[$normalized_url]:-}"
    
    # Generate default response if none configured
    if [[ -z "$response" ]]; then
        response=$(http_generate_default_response "$method" "$normalized_url" "$data")
    fi
    
    # Output response
    http_debug "About to output: response='$response', status='$status', silent='$silent'"
    http_output_response "$response" "$status" "$headers" "$output_file" "$silent" "$fail_on_error" "$write_out" "$include_headers"
    
    # Return appropriate exit code
    if [[ "$status" =~ ^[45][0-9][0-9]$ && "$fail_on_error" == "true" ]]; then
        return 22 # HTTP error
    fi
    return 0
}

http_generate_default_response() {
    local method="$1" url="$2" data="$3"
    
    # Generate contextual default responses
    case "$url" in
        */health|*/healthz|*/health-check)
            echo "{\"status\":\"healthy\",\"timestamp\":\"$(date -Iseconds)\"}"
            ;;
        */metrics)
            echo "# HELP mock_requests_total Total number of requests
# TYPE mock_requests_total counter
mock_requests_total{method=\"$method\"} ${HTTP_CALL_COUNT[$url]:-1}"
            ;;
        */status)
            echo "{\"service\":\"mock\",\"status\":\"running\",\"method\":\"$method\"}"
            ;;
        */version|*/v1|*/api/v*)
            echo "{\"version\":\"1.0.0\",\"api\":\"mock\",\"method\":\"$method\"}"
            ;;
        *)
            if [[ "$method" == "POST" && -n "$data" ]]; then
                echo "{\"received\":true,\"method\":\"$method\",\"data\":\"$(echo "$data" | head -c 100)\"}"
            else
                echo "{\"message\":\"Mock response\",\"url\":\"$url\",\"method\":\"$method\"}"
            fi
            ;;
    esac
}

http_output_response() {
    local response="$1" status="$2" headers="$3" output_file="$4" silent="$5" fail_on_error="$6" write_out="$7" include_headers="$8"
    
    # Build full response
    local full_response=""
    
    # Add headers if requested
    if [[ "$include_headers" == "true" ]]; then
        full_response="HTTP/1.1 $status OK\r\n"
        full_response+="Content-Type: application/json\r\n"
        if [[ -n "$headers" ]]; then
            IFS='|' read -ra header_array <<< "$headers"
            for header in "${header_array[@]}"; do
                full_response+="$header\r\n"
            done
        fi
        full_response+="\r\n$response"
    else
        full_response="$response"
    fi
    
    # Output response
    if [[ -n "$output_file" ]]; then
        echo -e "$full_response" > "$output_file"
        http_debug "Written response to: $output_file"
    else
        # Always output the response body (silent only suppresses curl's progress info, not the response)
        echo -e "$full_response"
    fi
    
    # Handle write-out format
    if [[ -n "$write_out" ]]; then
        case "$write_out" in
            "%{http_code}") echo "$status" ;;
            "%{response_code}") echo "$status" ;;
            "%{url_effective}") echo "$normalized_url" ;;
            *) echo "$status" ;; # Default to status code
        esac
    fi
}

# === Configuration Functions ===
http_mock_set_response() {
    local url="$1" response="$2"
    local normalized_url=$(http_normalize_url "$url")
    HTTP_RESPONSES[$normalized_url]="$response"
    http_debug "Set response for $normalized_url"
}

http_mock_set_status() {
    local url="$1" status="$2"
    local normalized_url=$(http_normalize_url "$url")
    HTTP_STATUS_CODES[$normalized_url]="$status"
    http_debug "Set status $status for $normalized_url"
}

http_mock_set_headers() {
    local url="$1" headers="$2"
    local normalized_url=$(http_normalize_url "$url")
    HTTP_HEADERS[$normalized_url]="$headers"
    http_debug "Set headers for $normalized_url"
}

http_mock_set_methods() {
    local url="$1" methods="$2"
    local normalized_url=$(http_normalize_url "$url")
    HTTP_METHODS[$normalized_url]="$methods"
    http_debug "Set allowed methods [$methods] for $normalized_url"
}

http_mock_set_error() {
    HTTP_CONFIG[error_mode]="$1"
    http_debug "Set error mode: $1"
}

http_mock_set_endpoint_healthy() {
    local url="$1"
    http_mock_set_response "$url" "{\"status\":\"healthy\"}"
    http_mock_set_status "$url" "200"
}

http_mock_set_endpoint_unhealthy() {
    local url="$1"
    http_mock_set_response "$url" "{\"status\":\"unhealthy\",\"error\":\"Service degraded\"}"
    http_mock_set_status "$url" "503"
}

http_mock_set_endpoint_down() {
    local url="$1"
    http_mock_set_response "$url" "{\"error\":\"Service not found\"}"
    http_mock_set_status "$url" "404"
}

# === State Management ===
http_mock_reset() {
    http_debug "Resetting mock state (called from: ${BASH_SOURCE[1]:-unknown}:${BASH_LINENO[0]:-unknown})"
    
    HTTP_RESPONSES=()
    HTTP_STATUS_CODES=()
    HTTP_CALL_COUNT=()
    HTTP_HEADERS=()
    HTTP_METHODS=()
    HTTP_CONFIG[error_mode]=""
    HTTP_CONFIG[debug]="false"
    
    # Initialize defaults
    http_mock_init_defaults
}

http_mock_init_defaults() {
    # Common health endpoints
    http_mock_set_endpoint_healthy "http://localhost:8080/health"
    http_mock_set_endpoint_healthy "http://localhost:9090/health"
    http_mock_set_endpoint_healthy "http://127.0.0.1:8080/health"
    
    # Default API responses
    HTTP_RESPONSES["http://localhost:8080/api/status"]="{\"status\":\"running\"}"
    HTTP_STATUS_CODES["http://localhost:8080/api/status"]="200"
    
    HTTP_RESPONSES["http://localhost:8080/metrics"]="mock_requests_total 1"
    HTTP_STATUS_CODES["http://localhost:8080/metrics"]="200"
}

http_mock_get_call_count() {
    local url="$1"
    local normalized_url=$(http_normalize_url "$url")
    echo "${HTTP_CALL_COUNT[$normalized_url]:-0}"
}

http_mock_dump_state() {
    echo "=== HTTP Mock State ==="
    echo "Error Mode: ${HTTP_CONFIG[error_mode]:-none}"
    echo "Debug: ${HTTP_CONFIG[debug]}"
    echo "Responses: ${#HTTP_RESPONSES[@]}"
    for url in "${!HTTP_RESPONSES[@]}"; do
        local response="${HTTP_RESPONSES[$url]}"
        local status="${HTTP_STATUS_CODES[$url]:-200}"
        local count="${HTTP_CALL_COUNT[$url]:-0}"
        echo "  $url: [$status] $count calls - ${response:0:50}..."
    done
    echo "Call Counts: ${#HTTP_CALL_COUNT[@]} URLs tracked"
    echo "=================="
}

# === Convention-based Test Functions ===
test_http_connection() {
    http_debug "Testing connection..."
    
    # Test basic HTTP request
    local result
    result=$(curl -s http://localhost:8080/health 2>&1)
    
    if [[ "$result" =~ "status" ]]; then
        http_debug "Connection test passed"
        return 0
    else
        http_debug "Connection test failed: $result"
        return 1
    fi
}

test_http_health() {
    http_debug "Testing health..."
    
    # Test connection
    test_http_connection || return 1
    
    # Test different HTTP methods
    curl -s -X GET http://localhost:8080/health >/dev/null 2>&1 || return 1
    curl -s -X POST http://localhost:8080/api/status -d '{"test":true}' >/dev/null 2>&1 || return 1
    
    # Test error handling
    local old_error="${HTTP_CONFIG[error_mode]}"
    http_mock_set_error "connection_refused"
    if curl -s http://localhost:8080/health >/dev/null 2>&1; then
        http_mock_set_error "$old_error"
        return 1
    fi
    http_mock_set_error "$old_error"
    
    http_debug "Health test passed"
    return 0
}

test_http_basic() {
    http_debug "Testing basic operations..."
    
    # Test different response types
    local health_result
    health_result=$(curl -s http://localhost:8080/health 2>&1)
    [[ "$health_result" =~ "healthy" ]] || return 1
    
    # Test POST with data
    local post_result
    post_result=$(curl -s -X POST http://localhost:8080/api/test -d '{"key":"value"}' 2>&1)
    [[ "$post_result" =~ "received" ]] || return 1
    
    # Test call counting
    local initial_count=$(http_mock_get_call_count "http://localhost:8080/health")
    curl -s http://localhost:8080/health >/dev/null 2>&1
    local new_count=$(http_mock_get_call_count "http://localhost:8080/health")
    [[ "$new_count" -gt "$initial_count" ]] || return 1
    
    # Test wget
    wget -q http://localhost:8080/health >/dev/null 2>&1 || return 1
    
    # Test nc connectivity
    nc -w 1 -z localhost 8080 >/dev/null 2>&1 || return 1
    
    http_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f curl wget nc
export -f test_http_connection test_http_health test_http_basic
export -f http_mock_reset http_mock_set_error http_mock_set_response
export -f http_mock_set_status http_mock_set_headers http_mock_set_methods
export -f http_mock_set_endpoint_healthy http_mock_set_endpoint_unhealthy http_mock_set_endpoint_down
export -f http_mock_get_call_count http_mock_dump_state
export -f http_debug http_check_error

# Initialize with defaults
http_mock_reset
http_debug "HTTP Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
