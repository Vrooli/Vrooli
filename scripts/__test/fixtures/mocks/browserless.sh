#!/usr/bin/env bash
# Browserless Mock Implementation
# 
# Provides comprehensive mock for Browserless operations including:
# - curl API endpoint interception
# - Docker container state simulation
# - Health check endpoints (/pressure, /chrome/*)
# - API response simulation (screenshot, pdf, scrape, function)
# - Container lifecycle management
# - Network and port management
#
# This mock follows the same standards as other updated mocks with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${BROWSERLESS_MOCK_LOADED:-}" ]] && return 0
declare -g BROWSERLESS_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"

# Global configuration
declare -g BROWSERLESS_MOCK_STATE_DIR="${BROWSERLESS_MOCK_STATE_DIR:-/tmp/browserless-mock-state}"
declare -g BROWSERLESS_MOCK_DEBUG="${BROWSERLESS_MOCK_DEBUG:-}"

# Global state arrays
declare -gA BROWSERLESS_MOCK_CONFIG=(           # Browserless configuration
    [container_name]="vrooli-browserless"
    [port]="3001"
    [base_url]="http://localhost:3001"
    [image]="ghcr.io/browserless/chromium"
    [max_browsers]="5"
    [timeout]="30000"
    [headless]="yes"
    [health_status]="healthy"
    [server_status]="running"
    [error_mode]=""
    [network_name]="vrooli-network"
    [data_dir]="/tmp/browserless-data"
    [version]="latest"
)

declare -gA BROWSERLESS_MOCK_CONTAINERS=()      # Container state tracking
declare -gA BROWSERLESS_MOCK_API_RESPONSES=()   # Custom API responses
declare -gA BROWSERLESS_MOCK_NETWORK_STATE=()   # Network existence tracking
declare -gA BROWSERLESS_MOCK_PRESSURE_DATA=()   # Browser pool pressure data
declare -gA BROWSERLESS_MOCK_ERROR_INJECTION=() # Error injection for testing
declare -gA BROWSERLESS_MOCK_REQUEST_LOG=()     # Log of API requests made

# Initialize state directory
mkdir -p "$BROWSERLESS_MOCK_STATE_DIR"

# Default pressure data
BROWSERLESS_MOCK_PRESSURE_DATA=(
    [running]="1"
    [queued]="0"
    [maxConcurrent]="5"
    [isAvailable]="true"
    [cpu]="0.15"
    [memory]="0.45"
    [recentlyRejected]="0"
)

# State persistence functions
mock::browserless::save_state() {
    local state_file="$BROWSERLESS_MOCK_STATE_DIR/browserless-state.sh"
    {
        echo "# Browserless mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p BROWSERLESS_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA BROWSERLESS_MOCK_CONFIG=()"
        declare -p BROWSERLESS_MOCK_CONTAINERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA BROWSERLESS_MOCK_CONTAINERS=()"
        declare -p BROWSERLESS_MOCK_API_RESPONSES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA BROWSERLESS_MOCK_API_RESPONSES=()"
        declare -p BROWSERLESS_MOCK_NETWORK_STATE 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA BROWSERLESS_MOCK_NETWORK_STATE=()"
        declare -p BROWSERLESS_MOCK_PRESSURE_DATA 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA BROWSERLESS_MOCK_PRESSURE_DATA=()"
        declare -p BROWSERLESS_MOCK_ERROR_INJECTION 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA BROWSERLESS_MOCK_ERROR_INJECTION=()"
        declare -p BROWSERLESS_MOCK_REQUEST_LOG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA BROWSERLESS_MOCK_REQUEST_LOG=()"
    } > "$state_file"
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless" "Saved Browserless state to $state_file"
    fi
}

mock::browserless::load_state() {
    local state_file="$BROWSERLESS_MOCK_STATE_DIR/browserless-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        
        # Use centralized logging if available
        if declare -f mock::log_state >/dev/null 2>&1; then
            mock::log_state "browserless" "Loaded Browserless state from $state_file"
        fi
    fi
}

# Automatically load state when sourced
mock::browserless::load_state

# ----------------------------
# Core Mock Functions
# ----------------------------

# Reset all mock state
mock::browserless::reset() {
    declare -gA BROWSERLESS_MOCK_CONFIG=(
        [container_name]="vrooli-browserless"
        [port]="3001"
        [base_url]="http://localhost:3001"
        [image]="ghcr.io/browserless/chromium"
        [max_browsers]="5"
        [timeout]="30000"
        [headless]="yes"
        [health_status]="healthy"
        [server_status]="running"
        [error_mode]=""
        [network_name]="vrooli-network"
        [data_dir]="/tmp/browserless-data"
        [version]="latest"
    )
    
    declare -gA BROWSERLESS_MOCK_CONTAINERS=()
    declare -gA BROWSERLESS_MOCK_API_RESPONSES=()
    declare -gA BROWSERLESS_MOCK_NETWORK_STATE=()
    declare -gA BROWSERLESS_MOCK_ERROR_INJECTION=()
    declare -gA BROWSERLESS_MOCK_REQUEST_LOG=()
    
    declare -gA BROWSERLESS_MOCK_PRESSURE_DATA=(
        [running]="1"
        [queued]="0"
        [maxConcurrent]="5"
        [isAvailable]="true"
        [cpu]="0.15"
        [memory]="0.45"
        [recentlyRejected]="0"
    )
    
    mock::browserless::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless" "Reset all Browserless mock state"
    fi
}

# Set container state
mock::browserless::set_container_state() {
    local name="$1"
    local state="$2"
    local image="${3:-${BROWSERLESS_MOCK_CONFIG[image]}}"
    
    # Handle empty names by using a placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    BROWSERLESS_MOCK_CONTAINERS["$name"]="$state|$image"
    mock::browserless::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless_container_state" "$name" "$state"
    fi
}

# Set API response
mock::browserless::set_api_response() {
    local endpoint="$1"
    local response="$2"
    
    BROWSERLESS_MOCK_API_RESPONSES["$endpoint"]="$response"
    mock::browserless::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless_api_response" "$endpoint" "response_set"
    fi
}

# Set pressure data
mock::browserless::set_pressure() {
    local running="${1:-1}"
    local queued="${2:-0}"
    local available="${3:-true}"
    local cpu="${4:-0.15}"
    local memory="${5:-0.45}"
    
    # Handle empty values with defaults
    [[ -z "$running" ]] && running="1"
    [[ -z "$queued" ]] && queued="0"
    [[ -z "$available" ]] && available="true"
    [[ -z "$cpu" ]] && cpu="0.15"
    [[ -z "$memory" ]] && memory="0.45"
    
    BROWSERLESS_MOCK_PRESSURE_DATA[running]="$running"
    BROWSERLESS_MOCK_PRESSURE_DATA[queued]="$queued"
    BROWSERLESS_MOCK_PRESSURE_DATA[isAvailable]="$available"
    BROWSERLESS_MOCK_PRESSURE_DATA[cpu]="$cpu"
    BROWSERLESS_MOCK_PRESSURE_DATA[memory]="$memory"
    
    mock::browserless::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless_pressure" "set" "$running/$queued/$available"
    fi
}

# Inject error for specific endpoint
mock::browserless::inject_error() {
    local endpoint="$1"
    local error_type="${2:-generic}"
    
    BROWSERLESS_MOCK_ERROR_INJECTION["$endpoint"]="$error_type"
    mock::browserless::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless_error_injection" "$endpoint" "$error_type"
    fi
}

# Set health status
mock::browserless::set_health_status() {
    local status="${1:-healthy}"
    BROWSERLESS_MOCK_CONFIG[health_status]="$status"
    mock::browserless::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless_health" "set" "$status"
    fi
}

# Set server status
mock::browserless::set_server_status() {
    local status="${1:-running}"
    BROWSERLESS_MOCK_CONFIG[server_status]="$status"
    mock::browserless::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless_server" "set" "$status"
    fi
}

# ----------------------------
# curl Interceptor
# ----------------------------

curl() {
    # Use centralized logging if available
    if declare -f mock::log_and_verify >/dev/null 2>&1; then
        mock::log_and_verify "curl" "$@"
    fi
    
    # Always reload state at the beginning to handle BATS subshells
    mock::browserless::load_state
    
    # Parse curl arguments
    local url="" method="GET" output_file="" data="" headers=() max_time=""
    local write_out="" silent=false show_error=false fail_on_error=false
    local args=("$@")
    local i=0
    
    while [[ $i -lt ${#args[@]} ]]; do
        case "${args[$i]}" in
            -X)
                method="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            --output|-o)
                output_file="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            -d|--data)
                data="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            -H|--header)
                headers+=("${args[$((i+1))]}")
                i=$((i + 2))
                ;;
            --max-time)
                max_time="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            --write-out)
                write_out="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            -s|--silent)
                silent=true
                i=$((i + 1))
                ;;
            -S|--show-error)
                show_error=true
                i=$((i + 1))
                ;;
            -f|--fail)
                fail_on_error=true
                i=$((i + 1))
                ;;
            http*)
                url="${args[$i]}"
                i=$((i + 1))
                ;;
            *)
                i=$((i + 1))
                ;;
        esac
    done
    
    # Check if this is a Browserless API request
    if [[ "$url" == *"/pressure"* || "$url" == *"/chrome/"* || "$url" =~ :300[0-9] ]]; then
        mock::browserless::handle_api_request "$method" "$url" "$data" "$output_file" "$write_out" "$fail_on_error"
        return $?
    fi
    
    # For non-Browserless requests, simulate a generic successful response
    if [[ -n "$output_file" ]]; then
        echo "Mock curl response" > "$output_file"
    else
        echo "Mock curl response"
    fi
    
    [[ -n "$write_out" ]] && echo "200"
    mock::browserless::save_state
    return 0
}

# Handle Browserless API requests
mock::browserless::handle_api_request() {
    local method="$1"
    local url="$2"
    local data="$3"
    local output_file="$4"
    local write_out="$5"
    local fail_on_error="$6"
    
    # Log the request with microsecond precision to avoid collisions
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S.%N' | head -c 23)
    BROWSERLESS_MOCK_REQUEST_LOG["$timestamp"]="$method $url"
    
    # Extract endpoint from URL
    local endpoint=""
    if [[ "$url" == *"/pressure" ]]; then
        endpoint="pressure"
    elif [[ "$url" == *"/chrome/screenshot" ]]; then
        endpoint="screenshot"
    elif [[ "$url" == *"/chrome/pdf" ]]; then
        endpoint="pdf"
    elif [[ "$url" == *"/chrome/content" ]]; then
        endpoint="content"
    elif [[ "$url" == *"/chrome/function" ]]; then
        endpoint="function"
    else
        endpoint="unknown"
    fi
    
    # Check for injected errors
    if [[ -n "${BROWSERLESS_MOCK_ERROR_INJECTION[$endpoint]}" ]]; then
        mock::browserless::simulate_error "$endpoint" "${BROWSERLESS_MOCK_ERROR_INJECTION[$endpoint]}" "$output_file" "$write_out"
        return $?
    fi
    
    # Check server status
    if [[ "${BROWSERLESS_MOCK_CONFIG[server_status]}" != "running" ]]; then
        [[ "$fail_on_error" == "true" ]] && return 7  # Connection failed
        [[ -n "$write_out" ]] && echo "000"
        echo "curl: (7) Failed to connect to localhost port ${BROWSERLESS_MOCK_CONFIG[port]}: Connection refused" >&2
        return 7
    fi
    
    # Check health status
    if [[ "${BROWSERLESS_MOCK_CONFIG[health_status]}" != "healthy" ]]; then
        [[ "$fail_on_error" == "true" ]] && return 22  # HTTP error
        [[ -n "$write_out" ]] && echo "503"
        echo "Service Unavailable" >&2
        return 22
    fi
    
    # Generate response based on endpoint
    case "$endpoint" in
        "pressure")
            mock::browserless::generate_pressure_response "$output_file"
            ;;
        "screenshot")
            mock::browserless::generate_screenshot_response "$data" "$output_file"
            ;;
        "pdf")
            mock::browserless::generate_pdf_response "$data" "$output_file"
            ;;
        "content")
            mock::browserless::generate_content_response "$data" "$output_file"
            ;;
        "function")
            mock::browserless::generate_function_response "$data" "$output_file"
            ;;
        *)
            echo "Unknown endpoint" >&2
            [[ -n "$write_out" ]] && echo "404"
            return 22
            ;;
    esac
    
    [[ -n "$write_out" ]] && echo "200"
    mock::browserless::save_state
    return 0
}

# Simulate various error conditions
mock::browserless::simulate_error() {
    local endpoint="$1"
    local error_type="$2"
    local output_file="$3"
    local write_out="$4"
    
    case "$error_type" in
        "connection_timeout")
            echo "curl: (28) Operation timed out after 30000 milliseconds" >&2
            [[ -n "$write_out" ]] && echo "000"
            return 28
            ;;
        "connection_refused")
            echo "curl: (7) Failed to connect to localhost port ${BROWSERLESS_MOCK_CONFIG[port]}: Connection refused" >&2
            [[ -n "$write_out" ]] && echo "000"
            return 7
            ;;
        "http_500")
            echo "Internal Server Error" > "${output_file:-/dev/stdout}"
            [[ -n "$write_out" ]] && echo "500"
            return 22
            ;;
        "http_429")
            echo "Too Many Requests - Browser limit exceeded" > "${output_file:-/dev/stdout}"
            [[ -n "$write_out" ]] && echo "429"
            return 22
            ;;
        "http_400")
            echo "Bad Request - Invalid parameters" > "${output_file:-/dev/stdout}"
            [[ -n "$write_out" ]] && echo "400"
            return 22
            ;;
        *)
            echo "Generic error occurred" >&2
            [[ -n "$write_out" ]] && echo "500"
            return 1
            ;;
    esac
}

# Generate pressure endpoint response
mock::browserless::generate_pressure_response() {
    local output_file="$1"
    
    local response=$(cat <<EOF
{
  "running": ${BROWSERLESS_MOCK_PRESSURE_DATA[running]},
  "queued": ${BROWSERLESS_MOCK_PRESSURE_DATA[queued]},
  "maxConcurrent": ${BROWSERLESS_MOCK_PRESSURE_DATA[maxConcurrent]},
  "isAvailable": ${BROWSERLESS_MOCK_PRESSURE_DATA[isAvailable]},
  "recentlyRejected": ${BROWSERLESS_MOCK_PRESSURE_DATA[recentlyRejected]},
  "cpu": ${BROWSERLESS_MOCK_PRESSURE_DATA[cpu]},
  "memory": ${BROWSERLESS_MOCK_PRESSURE_DATA[memory]}
}
EOF
)
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# Generate screenshot response
mock::browserless::generate_screenshot_response() {
    local data="$1"
    local output_file="$2"
    
    # Check for custom response
    if [[ -n "${BROWSERLESS_MOCK_API_RESPONSES[screenshot]}" ]]; then
        echo "${BROWSERLESS_MOCK_API_RESPONSES[screenshot]}" > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    # Generate mock PNG data (minimal valid PNG header)
    local png_data=$'\x89PNG\x0D\x0A\x1A\x0A\x00\x00\x00\x0DIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xDE'
    png_data+=$'\x00\x00\x00\x0CIDAT\x08\x1dc\xF8\x00\x00\x00\x01\x00\x01'
    png_data+=$'\x00\x00\x00\x00IEND\xAEB`\x82'
    
    if [[ -n "$output_file" ]]; then
        printf "%s" "$png_data" > "$output_file"
    else
        printf "%s" "$png_data"
    fi
}

# Generate PDF response
mock::browserless::generate_pdf_response() {
    local data="$1"
    local output_file="$2"
    
    # Check for custom response
    if [[ -n "${BROWSERLESS_MOCK_API_RESPONSES[pdf]}" ]]; then
        echo "${BROWSERLESS_MOCK_API_RESPONSES[pdf]}" > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    # Generate mock PDF data (minimal valid PDF)
    local pdf_data="%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]>>endobj
xref
0 4
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
trailer<</Size 4/Root 1 0 R>>
startxref
190
%%EOF"
    
    if [[ -n "$output_file" ]]; then
        echo "$pdf_data" > "$output_file"
    else
        echo "$pdf_data"
    fi
}

# Generate content scraping response
mock::browserless::generate_content_response() {
    local data="$1"
    local output_file="$2"
    
    # Check for custom response
    if [[ -n "${BROWSERLESS_MOCK_API_RESPONSES[content]}" ]]; then
        echo "${BROWSERLESS_MOCK_API_RESPONSES[content]}" > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    # Parse URL from data if provided
    local url="https://example.com"
    if [[ -n "$data" ]] && command -v jq &>/dev/null; then
        url=$(echo "$data" | jq -r '.url // "https://example.com"' 2>/dev/null || echo "https://example.com")
    fi
    
    local response="<!DOCTYPE html>
<html>
<head>
    <title>Mock Page Content</title>
</head>
<body>
    <h1>Mock scraped content from $url</h1>
    <p>This is mock HTML content returned by the browserless mock.</p>
    <p>Original URL: $url</p>
</body>
</html>"
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# Generate function execution response
mock::browserless::generate_function_response() {
    local data="$1"
    local output_file="$2"
    
    # Check for custom response
    if [[ -n "${BROWSERLESS_MOCK_API_RESPONSES[function]}" ]]; then
        echo "${BROWSERLESS_MOCK_API_RESPONSES[function]}" > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    local response='{"title":"Mock Page Title","url":"https://example.com","viewport":{"width":1280,"height":720},"result":"Mock function execution completed successfully"}'
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# ----------------------------
# Test Helper Functions
# ----------------------------

# Create a running Browserless scenario
mock::browserless::scenario::create_running_service() {
    local container_name="${1:-${BROWSERLESS_MOCK_CONFIG[container_name]}}"
    
    mock::browserless::set_container_state "$container_name" "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::set_server_status "running"
    mock::browserless::set_pressure "2" "0" "true" "0.25" "0.35"
    
    BROWSERLESS_MOCK_NETWORK_STATE["${BROWSERLESS_MOCK_CONFIG[network_name]}"]="created"
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless_scenario" "created_running_service" "$container_name"
    fi
}

# Create a stopped Browserless scenario
mock::browserless::scenario::create_stopped_service() {
    local container_name="${1:-${BROWSERLESS_MOCK_CONFIG[container_name]}}"
    
    mock::browserless::set_container_state "$container_name" "stopped"
    mock::browserless::set_health_status "unhealthy"
    mock::browserless::set_server_status "stopped"
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless_scenario" "created_stopped_service" "$container_name"
    fi
}

# Create an overloaded service scenario
mock::browserless::scenario::create_overloaded_service() {
    local container_name="${1:-${BROWSERLESS_MOCK_CONFIG[container_name]}}"
    
    mock::browserless::set_container_state "$container_name" "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::set_server_status "running"
    mock::browserless::set_pressure "5" "3" "false" "0.85" "0.90"
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "browserless_scenario" "created_overloaded_service" "$container_name"
    fi
}

# ----------------------------
# Assertion Helper Functions
# ----------------------------

mock::browserless::assert::container_running() {
    local name="${1:-${BROWSERLESS_MOCK_CONFIG[container_name]}}"
    
    # Handle empty names by using the placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    local state="${BROWSERLESS_MOCK_CONTAINERS[$name]%%|*}"
    
    if [[ "$state" != "running" ]]; then
        echo "ASSERTION FAILED: Browserless container '$name' is not running (state: ${state:-not found})" >&2
        return 1
    fi
    return 0
}

mock::browserless::assert::container_stopped() {
    local name="${1:-${BROWSERLESS_MOCK_CONFIG[container_name]}}"
    
    # Handle empty names by using the placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    local state="${BROWSERLESS_MOCK_CONTAINERS[$name]%%|*}"
    
    if [[ "$state" != "stopped" ]]; then
        echo "ASSERTION FAILED: Browserless container '$name' is not stopped (state: ${state:-not found})" >&2
        return 1
    fi
    return 0
}

mock::browserless::assert::healthy() {
    if [[ "${BROWSERLESS_MOCK_CONFIG[health_status]}" != "healthy" ]]; then
        echo "ASSERTION FAILED: Browserless is not healthy (status: ${BROWSERLESS_MOCK_CONFIG[health_status]})" >&2
        return 1
    fi
    return 0
}

mock::browserless::assert::pressure_available() {
    if [[ "${BROWSERLESS_MOCK_PRESSURE_DATA[isAvailable]}" != "true" ]]; then
        echo "ASSERTION FAILED: Browserless is not available (isAvailable: ${BROWSERLESS_MOCK_PRESSURE_DATA[isAvailable]})" >&2
        return 1
    fi
    return 0
}

mock::browserless::assert::api_called() {
    local endpoint="$1"
    local found=false
    
    # Load state to ensure we have the latest request log in subshells
    mock::browserless::load_state
    
    for timestamp in "${!BROWSERLESS_MOCK_REQUEST_LOG[@]}"; do
        if [[ "${BROWSERLESS_MOCK_REQUEST_LOG[$timestamp]}" == *"$endpoint"* ]]; then
            found=true
            break
        fi
    done
    
    if [[ "$found" != "true" ]]; then
        echo "ASSERTION FAILED: API endpoint '$endpoint' was not called" >&2
        return 1
    fi
    return 0
}

# Get helper functions
mock::browserless::get::container_state() {
    local name="${1:-${BROWSERLESS_MOCK_CONFIG[container_name]}}"
    
    # Handle empty names by using the placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    echo "${BROWSERLESS_MOCK_CONTAINERS[$name]%%|*}"
}

mock::browserless::get::pressure_data() {
    local key="$1"
    echo "${BROWSERLESS_MOCK_PRESSURE_DATA[$key]:-}"
}

mock::browserless::get::config() {
    local key="$1"
    echo "${BROWSERLESS_MOCK_CONFIG[$key]:-}"
}

# Debug helper
mock::browserless::debug::dump_state() {
    echo "=== Browserless Mock State Dump ==="
    echo "Configuration:"
    for key in "${!BROWSERLESS_MOCK_CONFIG[@]}"; do
        echo "  $key: ${BROWSERLESS_MOCK_CONFIG[$key]}"
    done
    
    echo "Containers:"
    for name in "${!BROWSERLESS_MOCK_CONTAINERS[@]}"; do
        echo "  $name: ${BROWSERLESS_MOCK_CONTAINERS[$name]}"
    done
    
    echo "Pressure Data:"
    for key in "${!BROWSERLESS_MOCK_PRESSURE_DATA[@]}"; do
        echo "  $key: ${BROWSERLESS_MOCK_PRESSURE_DATA[$key]}"
    done
    
    echo "Error Injections:"
    for endpoint in "${!BROWSERLESS_MOCK_ERROR_INJECTION[@]}"; do
        echo "  $endpoint: ${BROWSERLESS_MOCK_ERROR_INJECTION[$endpoint]}"
    done
    
    echo "API Request Log:"
    for timestamp in "${!BROWSERLESS_MOCK_REQUEST_LOG[@]}"; do
        echo "  $timestamp: ${BROWSERLESS_MOCK_REQUEST_LOG[$timestamp]}"
    done
    echo "=========================="
}

# ----------------------------
# Export functions for subshells
# ----------------------------
export -f curl
export -f mock::browserless::reset
export -f mock::browserless::set_container_state
export -f mock::browserless::set_api_response
export -f mock::browserless::set_pressure
export -f mock::browserless::inject_error
export -f mock::browserless::set_health_status
export -f mock::browserless::set_server_status
export -f mock::browserless::handle_api_request
export -f mock::browserless::simulate_error
export -f mock::browserless::generate_pressure_response
export -f mock::browserless::generate_screenshot_response
export -f mock::browserless::generate_pdf_response
export -f mock::browserless::generate_content_response
export -f mock::browserless::generate_function_response

# Export scenario functions
export -f mock::browserless::scenario::create_running_service
export -f mock::browserless::scenario::create_stopped_service
export -f mock::browserless::scenario::create_overloaded_service

# Export assertion functions
export -f mock::browserless::assert::container_running
export -f mock::browserless::assert::container_stopped
export -f mock::browserless::assert::healthy
export -f mock::browserless::assert::pressure_available
export -f mock::browserless::assert::api_called

# Export getter functions
export -f mock::browserless::get::container_state
export -f mock::browserless::get::pressure_data
export -f mock::browserless::get::config

# Export debug functions
export -f mock::browserless::debug::dump_state

echo "[MOCK] Browserless mocks loaded successfully"