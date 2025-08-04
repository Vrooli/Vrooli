#!/bin/bash
# HTTP Test Handler - Handles HTTP request/response testing

set -euo pipefail

# Colors for output (only define if not already defined)
if [[ -z "${RED:-}" ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[1;33m'
    readonly BLUE='\033[0;34m'
    readonly NC='\033[0m'
fi

# HTTP test results
HTTP_ERRORS=0
HTTP_WARNINGS=0

# Source common utilities if available
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/../clients/common.sh" ]]; then
    source "$SCRIPT_DIR/../clients/common.sh"
fi

# Print functions
print_http_info() {
    echo -e "${BLUE}[HTTP]${NC} $1"
}

print_http_success() {
    echo -e "${GREEN}[HTTP ✓]${NC} $1"
}

print_http_error() {
    echo -e "${RED}[HTTP ✗]${NC} $1"
    ((HTTP_ERRORS++))
}

print_http_warning() {
    echo -e "${YELLOW}[HTTP ⚠]${NC} $1"
    ((HTTP_WARNINGS++))
}

# Check if service is available
check_service_availability() {
    local service_name="$1"
    local service_url="$2"
    local timeout="${3:-5}"
    
    # Try a simple connectivity check
    if [[ "$service_url" == http* ]]; then
        # HTTP service - try a simple curl check
        curl -s --connect-timeout "$timeout" --max-time "$timeout" "$service_url" >/dev/null 2>&1
    elif [[ "$service_url" == postgresql* ]]; then
        # PostgreSQL - use pg_isready if available
        local host=$(echo "$service_url" | sed 's|.*@\([^:]*\).*|\1|')
        local port=$(echo "$service_url" | sed 's|.*:\([0-9]*\)/.*|\1|')
        if command -v pg_isready >/dev/null 2>&1; then
            pg_isready -h "$host" -p "${port:-5432}" -t "$timeout" >/dev/null 2>&1
        else
            # Fallback to netcat
            nc -z "$host" "${port:-5432}" 2>/dev/null
        fi
    elif [[ "$service_url" == redis* ]]; then
        # Redis - use redis-cli if available
        local host=$(echo "$service_url" | sed 's|redis://\([^:]*\).*|\1|')
        local port=$(echo "$service_url" | sed 's|.*:\([0-9]*\).*|\1|')
        if command -v redis-cli >/dev/null 2>&1; then
            redis-cli -h "$host" -p "${port:-6379}" ping >/dev/null 2>&1
        else
            # Fallback to netcat
            nc -z "$host" "${port:-6379}" 2>/dev/null
        fi
    else
        # Generic check - assume it's reachable
        return 0
    fi
}

# Generate mock HTTP response
mock_http_response() {
    local expected_status="${1:-200}"
    local expected_body="${2:-{\"status\":\"mock\",\"message\":\"Service unavailable, mock response\"}}"
    
    print_http_success "Mock response generated (status: $expected_status)"
    return 0
}

# Make HTTP request with error handling
make_http_request() {
    local method="$1"
    local url="$2"
    local headers="$3"
    local body="$4"
    local timeout="${5:-30}"
    local output_file="${6:-}"
    
    local curl_args=(
        -X "$method"
        -s
        --max-time "$timeout"
        --show-error
        --write-out "HTTPSTATUS:%{http_code}|SIZE:%{size_download}|TIME:%{time_total}"
    )
    
    # Add headers if provided
    if [[ -n "$headers" ]]; then
        while IFS= read -r header; do
            if [[ -n "$header" ]]; then
                curl_args+=(-H "$header")
            fi
        done <<< "$headers"
    fi
    
    # Add body if provided
    if [[ -n "$body" ]]; then
        if [[ -f "$body" ]]; then
            # Body is a file path
            curl_args+=(-T "$body")
        else
            # Body is inline data
            curl_args+=(-d "$body")
        fi
    fi
    
    # Add output file if specified
    if [[ -n "$output_file" ]]; then
        curl_args+=(-o "$output_file")
    fi
    
    # Execute curl request
    curl "${curl_args[@]}" "$url" 2>/dev/null
}

# Parse HTTP response
parse_http_response() {
    local response="$1"
    
    # Extract status code, size, and time from curl output
    local status_line=$(echo "$response" | tail -1)
    local body=$(echo "$response" | head -n -1)
    
    local status_code=$(echo "$status_line" | sed 's/.*HTTPSTATUS:\([0-9]*\).*/\1/')
    local size=$(echo "$status_line" | sed 's/.*SIZE:\([0-9.]*\).*/\1/')
    local time=$(echo "$status_line" | sed 's/.*TIME:\([0-9.]*\).*/\1/')
    
    echo "STATUS:$status_code|BODY:$body|SIZE:$size|TIME:$time"
}

# Validate HTTP response
validate_http_response() {
    local response="$1"
    local expected_status="${2:-200}"
    local expected_contains="${3:-}"
    local expected_not_contains="${4:-}"
    local max_response_time="${5:-10}"
    
    # Parse response components
    local status_code=$(echo "$response" | sed 's/.*STATUS:\([0-9]*\).*/\1/')
    local body=$(echo "$response" | sed 's/.*BODY:\(.*\)|SIZE:.*/\1/')
    local response_time=$(echo "$response" | sed 's/.*TIME:\([0-9.]*\)/\1/')
    
    local validation_passed=true
    
    # Check status code
    if [[ "$status_code" != "$expected_status" ]]; then
        print_http_error "Expected status $expected_status, got $status_code"
        validation_passed=false
    fi
    
    # Check response contains expected content
    if [[ -n "$expected_contains" ]] && [[ "$body" != *"$expected_contains"* ]]; then
        print_http_error "Response does not contain expected text: '$expected_contains'"
        validation_passed=false
    fi
    
    # Check response does not contain unwanted content
    if [[ -n "$expected_not_contains" ]] && [[ "$body" == *"$expected_not_contains"* ]]; then
        print_http_error "Response contains unwanted text: '$expected_not_contains'"
        validation_passed=false
    fi
    
    # Check response time
    if [[ -n "$response_time" ]] && (( $(echo "$response_time > $max_response_time" | bc -l 2>/dev/null || echo "0") )); then
        print_http_warning "Slow response: ${response_time}s (max: ${max_response_time}s)"
    fi
    
    if [[ "$validation_passed" == "true" ]]; then
        print_http_success "Response validation passed (${status_code}, ${response_time}s)"
        return 0
    else
        return 1
    fi
}

# Execute HTTP test from declarative definition
execute_http_test_from_config() {
    local test_config_file="$1"
    
    # Read test configuration from file
    local test_config=""
    if [[ -f "$test_config_file" ]]; then
        test_config=$(cat "$test_config_file")
    else
        test_config="$test_config_file"
    fi
    
    # Extract test parameters (simplified parsing)
    local service=$(echo "$test_config" | grep "service:" | head -1 | cut -d: -f2 | xargs)
    local endpoint=$(echo "$test_config" | grep "endpoint:" | head -1 | cut -d: -f2 | xargs)
    local method=$(echo "$test_config" | grep "method:" | head -1 | cut -d: -f2 | xargs)
    local expected_status=$(echo "$test_config" | grep -A5 "expect:" | grep "status:" | head -1 | cut -d: -f2 | xargs)
    local expected_contains=$(echo "$test_config" | grep "contains:" | cut -d: -f2 | xargs)
    local required=$(echo "$test_config" | grep "required:" | head -1 | cut -d: -f2 | xargs)
    local fallback=$(echo "$test_config" | grep "fallback:" | head -1 | cut -d: -f2 | xargs)
    
    # Default values
    method="${method:-GET}"
    expected_status="${expected_status:-200}"
    required="${required:-true}"
    
    # Source resource validator for get_resource_url function
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -f "$script_dir/../validators/resources.sh" ]]; then
        source "$script_dir/../validators/resources.sh"
    fi
    
    # Get service URL
    local service_url=""
    if declare -f get_resource_url >/dev/null 2>&1; then
        service_url=$(get_resource_url "$service")
    else
        print_http_error "Cannot resolve service URL for: $service"
        return 1
    fi
    
    if [[ -z "$service_url" ]]; then
        if [[ "$required" == "false" ]]; then
            print_http_warning "Optional service not available: $service"
            return 2  # Return 2 for degraded/warning state
        else
            print_http_error "Required service URL not found: $service"
            return 1
        fi
    fi
    
    local full_url="${service_url}${endpoint}"
    
    print_http_info "Testing $method $full_url"
    
    # Check if service is actually reachable
    if ! check_service_availability "$service" "$service_url"; then
        if [[ "$required" == "false" ]]; then
            print_http_warning "Optional service unavailable: $service (continuing with degraded functionality)"
            return 2  # Return 2 for degraded/warning state
        elif [[ "$fallback" == "mock" ]]; then
            print_http_info "Service unavailable, using mock response: $service"
            mock_http_response "$expected_status" "$expected_contains"
            return 2  # Mock response is degraded functionality
        else
            print_http_error "Required service unavailable: $service"
            return 1
        fi
    fi
    
    # Make HTTP request
    local response=$(make_http_request "$method" "$full_url" "" "" 30)
    local parsed_response=$(parse_http_response "$response")
    
    # Validate response
    validate_http_response "$parsed_response" "$expected_status" "$expected_contains" "" 10
}

# Test HTTP endpoint with simple parameters
test_http_endpoint() {
    local url="$1"
    local method="${2:-GET}"
    local expected_status="${3:-200}"
    local expected_contains="${4:-}"
    local headers="${5:-}"
    local body="${6:-}"
    
    print_http_info "Testing $method $url"
    
    # Make request
    local response=$(make_http_request "$method" "$url" "$headers" "$body" 30)
    local parsed_response=$(parse_http_response "$response")
    
    # Validate response
    validate_http_response "$parsed_response" "$expected_status" "$expected_contains"
}

# Test health endpoint for a service
test_service_health() {
    local service_name="$1"
    local service_url="$2"
    
    # Common health endpoints to try
    local health_endpoints=(
        "/health"
        "/healthz"
        "/status"
        "/api/health"
        "/api/v1/health"
        "/"
    )
    
    for endpoint in "${health_endpoints[@]}"; do
        local full_url="${service_url}${endpoint}"
        
        if make_http_request "GET" "$full_url" "" "" 10 >/dev/null 2>&1; then
            print_http_success "$service_name health check passed ($endpoint)"
            return 0
        fi
    done
    
    print_http_error "$service_name health check failed"
    return 1
}

# Test API authentication
test_api_auth() {
    local url="$1"
    local auth_header="$2"
    local expected_status="${3:-200}"
    
    print_http_info "Testing authentication for $url"
    
    # Test without auth (should fail)
    local response_no_auth=$(make_http_request "GET" "$url" "" "" 10)
    local parsed_no_auth=$(parse_http_response "$response_no_auth")
    local status_no_auth=$(echo "$parsed_no_auth" | sed 's/.*STATUS:\([0-9]*\).*/\1/')
    
    if [[ "$status_no_auth" == "401" ]] || [[ "$status_no_auth" == "403" ]]; then
        print_http_success "Unauthorized access correctly blocked"
    else
        print_http_warning "Expected 401/403 without auth, got $status_no_auth"
    fi
    
    # Test with auth (should succeed)
    local response_with_auth=$(make_http_request "GET" "$url" "$auth_header" "" 10)
    local parsed_with_auth=$(parse_http_response "$response_with_auth")
    
    validate_http_response "$parsed_with_auth" "$expected_status"
}

# Test file upload
test_file_upload() {
    local url="$1"
    local file_path="$2"
    local expected_status="${3:-200}"
    local content_type="${4:-application/octet-stream}"
    
    if [[ ! -f "$file_path" ]]; then
        print_http_error "File not found: $file_path"
        return 1
    fi
    
    print_http_info "Testing file upload: $(basename "$file_path") to $url"
    
    local headers="Content-Type: $content_type"
    local response=$(make_http_request "POST" "$url" "$headers" "$file_path" 60)
    local parsed_response=$(parse_http_response "$response")
    
    validate_http_response "$parsed_response" "$expected_status"
}

# Execute HTTP test handler
execute_http_test() {
    local test_name="$1"
    local test_data="$2"
    
    print_http_info "Executing HTTP test: $test_name"
    
    # If test_data is a file path to config, parse it
    if [[ -f "$test_data" ]]; then
        execute_http_test_from_config "$test_data"
    else
        # Assume it's inline test data
        execute_http_test_from_config "$test_data"
    fi
}

# Test common service endpoints
test_ollama_endpoint() {
    local ollama_url="$1"
    
    print_http_info "Testing Ollama service"
    
    # Test health endpoint
    test_service_health "Ollama" "$ollama_url"
    
    # Test models endpoint
    test_http_endpoint "${ollama_url}/api/tags" "GET" "200" "models"
    
    # Test simple generation (if models are available)
    local test_prompt='{"model":"llama2","prompt":"Hello","stream":false}'
    test_http_endpoint "${ollama_url}/api/generate" "POST" "200" "" "Content-Type: application/json" "$test_prompt"
}

test_whisper_endpoint() {
    local whisper_url="$1"
    
    print_http_info "Testing Whisper service"
    
    # Test health endpoint
    test_service_health "Whisper" "$whisper_url"
}

test_comfyui_endpoint() {
    local comfyui_url="$1"
    
    print_http_info "Testing ComfyUI service"
    
    # Test health endpoint
    test_service_health "ComfyUI" "$comfyui_url"
    
    # Test system stats
    test_http_endpoint "${comfyui_url}/system_stats" "GET" "200"
}

# Export functions for use by test runner
export -f execute_http_test
export -f test_http_endpoint
export -f test_service_health
export -f test_api_auth
export -f test_file_upload
export -f test_ollama_endpoint
export -f test_whisper_endpoint
export -f test_comfyui_endpoint