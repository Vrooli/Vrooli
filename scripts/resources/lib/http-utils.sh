#!/usr/bin/env bash
# Generic HTTP/API Utility Functions
# Provides reusable HTTP operations for all resource managers

# Source guard to prevent multiple sourcing
[[ -n "${_HTTP_UTILS_SOURCED:-}" ]] && return 0
_HTTP_UTILS_SOURCED=1

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/system/system_commands.sh"

#######################################
# Make HTTP request with error handling
# Args: $1 - method, $2 - url, $3 - data (optional), $4 - headers (optional)
# Returns: Response body via stdout, HTTP code via return value
#######################################
http::request() {
    local method="$1"
    local url="$2"
    local data="${3:-}"
    local headers="${4:-}"
    
    local curl_args=("-s" "-w" "\n__HTTP_CODE__:%{http_code}")
    curl_args+=("-X" "$method")
    
    # Add headers if provided
    if [[ -n "$headers" ]]; then
        while IFS= read -r header; do
            [[ -n "$header" ]] && curl_args+=("-H" "$header")
        done <<< "$headers"
    fi
    
    # Add data for POST/PUT/PATCH requests
    if [[ -n "$data" ]] && [[ "$method" =~ ^(POST|PUT|PATCH)$ ]]; then
        curl_args+=("-d" "$data")
        # Add Content-Type if not already in headers
        if ! echo "$headers" | grep -qi "content-type"; then
            curl_args+=("-H" "Content-Type: application/json")
        fi
    fi
    
    # Add timeout
    curl_args+=("--max-time" "30")
    
    # Make the request
    local response
    response=$(curl "${curl_args[@]}" "$url" 2>/dev/null || echo "__HTTP_CODE__:000")
    
    # Extract HTTP code and body
    local http_code
    http_code=$(echo "$response" | grep "__HTTP_CODE__:" | cut -d':' -f2 | tr -d '\n' | sed 's/[^0-9]//g')
    local body
    body=$(echo "$response" | grep -v "__HTTP_CODE__:")
    
    # Validate HTTP code is numeric
    if [[ ! "$http_code" =~ ^[0-9]+$ ]]; then
        http_code="0"
    fi
    
    # Output body and return appropriate exit code
    echo "$body"
    
    # Return 0 for successful HTTP codes (200-299), 1 for failures
    if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validate JSON response
# Args: $1 - response body, $2 - expected_fields (optional, space-separated)
# Returns: 0 if valid JSON, 1 if invalid
#######################################
http::validate_json() {
    local response="$1"
    local expected_fields="${2:-}"
    
    # Check if jq is available
    if ! system::is_command "jq"; then
        # Basic JSON validation without jq
        if [[ "$response" =~ ^\{.*\}$ ]] || [[ "$response" =~ ^\[.*\]$ ]]; then
            return 0
        else
            return 1
        fi
    fi
    
    # Check if response is valid JSON
    if ! echo "$response" | jq empty 2>/dev/null; then
        return 1
    fi
    
    # Check for expected fields if provided
    if [[ -n "$expected_fields" ]]; then
        for field in $expected_fields; do
            if ! echo "$response" | jq -e ".$field" >/dev/null 2>&1; then
                log::warn "Expected field '$field' not found in response"
            fi
        done
    fi
    
    return 0
}

#######################################
# Extract field from JSON response
# Args: $1 - response body, $2 - field path (jq syntax)
# Returns: Field value via stdout, empty if not found
#######################################
http::extract_json_field() {
    local response="$1"
    local field_path="$2"
    
    # Check if jq is available
    if ! system::is_command "jq"; then
        log::warn "jq not available for JSON parsing"
        echo ""
        return 1
    fi
    
    # Try to extract the field
    if echo "$response" | jq -e "$field_path" >/dev/null 2>&1; then
        echo "$response" | jq -r "$field_path"
    else
        echo ""
    fi
}

#######################################
# Check if API endpoint is accessible
# Args: $1 - url, $2 - expected_code (optional, default 200)
# Returns: 0 if accessible, 1 otherwise
#######################################
http::check_endpoint() {
    local url="$1"
    # The expected_code parameter is kept for compatibility but not used
    # since http::request already returns proper exit codes
    
    # Simply pass through the exit code from http::request
    # (0 for success/200-299, 1 for failure)
    http::request "GET" "$url" >/dev/null 2>&1
}

#######################################
# Make authenticated API request
# Args: $1 - method, $2 - url, $3 - auth_header, $4 - data (optional)
# Returns: Response via stdout, HTTP code via return value
#######################################
http::auth_request() {
    local method="$1"
    local url="$2"
    local auth_header="$3"
    local data="${4:-}"
    
    http::request "$method" "$url" "$data" "$auth_header"
}

#######################################
# Check API response for errors
# Args: $1 - response body, $2 - http_code
# Returns: 0 if no errors, 1 if error found
#######################################
http::check_response_error() {
    local response="$1"
    local http_code="$2"
    
    # Check HTTP error codes
    if [[ "$http_code" -ge 400 ]]; then
        local error_msg="HTTP error $http_code"
        
        # Try to extract error message from JSON
        if system::is_command "jq" && http::validate_json "$response"; then
            local json_error
            json_error=$(echo "$response" | jq -r '.error // .message // .msg // empty' 2>/dev/null)
            if [[ -n "$json_error" ]]; then
                error_msg="$error_msg: $json_error"
            fi
        fi
        
        log::error "$error_msg"
        return 1
    fi
    
    return 0
}

#######################################
# Resolve API key from multiple sources
# Args: $1 - env_var_name, $2 - secrets_key (optional)
# Returns: API key via stdout, empty if not found
#######################################
http::resolve_api_key() {
    local env_var_name="$1"
    local secrets_key="${2:-$env_var_name}"
    
    # Try environment variable first
    local api_key="${!env_var_name:-}"
    
    # Try secrets resolution if available and env is empty
    if [[ -z "$api_key" ]] && command -v secrets::resolve &>/dev/null; then
        if api_key=$(secrets::resolve "$secrets_key" 2>/dev/null); then
            echo "$api_key"
            return 0
        fi
    fi
    
    echo "$api_key"
}

#######################################
# Test API authentication
# Args: $1 - test_url, $2 - auth_header
# Returns: 0 if authenticated, 1 if not
#######################################
http::test_auth() {
    local test_url="$1"
    local auth_header="$2"
    
    local response
    local http_code
    response=$(http::auth_request "GET" "$test_url" "$auth_header")
    http_code=$?
    
    case "$http_code" in
        200|201|204)
            log::success "API authentication successful"
            return 0
            ;;
        401)
            log::error "API authentication failed: Invalid credentials"
            return 1
            ;;
        403)
            log::error "API authentication failed: Insufficient permissions"
            return 1
            ;;
        *)
            log::error "API authentication failed: HTTP $http_code"
            return 1
            ;;
    esac
}

#######################################
# URL encode a string
# Args: $1 - string to encode
# Returns: Encoded string via stdout
#######################################
http::url_encode() {
    local string="$1"
    
    if system::is_command "python3"; then
        python3 -c "import urllib.parse; print(urllib.parse.quote('$string'))"
    elif system::is_command "python"; then
        python -c "import urllib; print urllib.quote('$string')"
    else
        # Basic encoding for common characters
        echo "$string" | sed 's/ /%20/g; s/!/%21/g; s/"/%22/g; s/#/%23/g; s/\$/%24/g; s/&/%26/g'
    fi
}

#######################################
# Parse URL components
# Args: $1 - url, $2 - component (protocol|host|port|path)
# Returns: Component value via stdout
#######################################
http::parse_url() {
    local url="$1"
    local component="$2"
    
    case "$component" in
        protocol)
            echo "$url" | sed 's|://.*||'
            ;;
        host)
            echo "$url" | sed 's|.*://||' | sed 's|/.*||' | sed 's|:.*||'
            ;;
        port)
            local port_part
            port_part=$(echo "$url" | sed 's|.*://||' | sed 's|/.*||' | grep ':' | sed 's|.*:||')
            if [[ -n "$port_part" ]]; then
                echo "$port_part"
            else
                # Default ports
                if [[ "$url" =~ ^https ]]; then
                    echo "443"
                else
                    echo "80"
                fi
            fi
            ;;
        path)
            echo "$url" | sed 's|.*://||' | sed 's|^[^/]*||'
            ;;
    esac
}

#######################################
# Make paginated API requests
# Args: $1 - base_url, $2 - auth_header, $3 - max_pages (optional)
# Returns: Combined results via stdout
#######################################
http::paginated_request() {
    local base_url="$1"
    local auth_header="$2"
    local max_pages="${3:-10}"
    
    local page=1
    local all_results="[]"
    local has_more=true
    
    while [[ "$has_more" == "true" ]] && [[ $page -le $max_pages ]]; do
        local url="$base_url"
        if [[ "$url" =~ \? ]]; then
            url="${url}&page=${page}"
        else
            url="${url}?page=${page}"
        fi
        
        local response
        local http_code
        response=$(http::auth_request "GET" "$url" "$auth_header")
        http_code=$?
        
        if [[ $http_code -ne 200 ]]; then
            break
        fi
        
        # Check if response has data
        if system::is_command "jq"; then
            local data
            data=$(echo "$response" | jq '.data // .' 2>/dev/null)
            
            if [[ "$data" == "[]" ]] || [[ "$data" == "null" ]]; then
                has_more=false
            else
                # Combine results
                all_results=$(echo "$all_results $data" | jq -s 'add')
            fi
        else
            # Without jq, just return first page
            echo "$response"
            return 0
        fi
        
        page=$((page + 1))
    done
    
    echo "$all_results"
}

#######################################
# Retry HTTP request with exponential backoff
# Args: $1 - method, $2 - url, $3 - max_attempts, $4 - data (optional), $5 - headers (optional)
# Returns: Response via stdout, last HTTP code via return value
#######################################
http::retry_request() {
    local method="$1"
    local url="$2"
    local max_attempts="${3:-3}"
    local data="${4:-}"
    local headers="${5:-}"
    
    local attempt=1
    local delay=2
    
    while [ $attempt -le $max_attempts ]; do
        local response
        local http_code
        
        response=$(http::request "$method" "$url" "$data" "$headers")
        http_code=$?
        
        # Success codes
        if [[ $http_code -ge 200 ]] && [[ $http_code -lt 300 ]]; then
            echo "$response"
            return $http_code
        fi
        
        # Don't retry on client errors (except 429)
        if [[ $http_code -ge 400 ]] && [[ $http_code -lt 500 ]] && [[ $http_code -ne 429 ]]; then
            echo "$response"
            return $http_code
        fi
        
        if [ $attempt -lt $max_attempts ]; then
            log::warn "Request failed (attempt $attempt/$max_attempts), retrying in ${delay}s..."
            sleep $delay
            delay=$((delay * 2))  # Exponential backoff
        fi
        
        attempt=$((attempt + 1))
    done
    
    echo "$response"
    return $http_code
}