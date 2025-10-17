#!/usr/bin/env bash
################################################################################
# HTTP Request Tool Executor
# 
# Handles HTTP requests with comprehensive options and security
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Execute HTTP request tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   JSON result with HTTP response information
#######################################
tool_http_request::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Parse arguments
    local url method headers body timeout follow_redirects verify_ssl user_agent auth
    url=$(echo "$arguments" | jq -r '.url')
    method=$(echo "$arguments" | jq -r '.method // "GET"')
    headers=$(echo "$arguments" | jq -r '.headers // {}')
    body=$(echo "$arguments" | jq -r '.body // empty')
    timeout=$(echo "$arguments" | jq -r '.timeout // 30')
    follow_redirects=$(echo "$arguments" | jq -r '.follow_redirects // true')
    verify_ssl=$(echo "$arguments" | jq -r '.verify_ssl // true')
    user_agent=$(echo "$arguments" | jq -r '.user_agent // "Codex-HTTP-Client/1.0"')
    auth=$(echo "$arguments" | jq -r '.auth // {}')
    
    # Validate required parameters
    if [[ "$url" == "null" || -z "$url" ]]; then
        echo '{"success": false, "error": "URL is required"}'
        return 1
    fi
    
    # Validate URL format
    if ! [[ "$url" =~ ^https?:// ]]; then
        echo '{"success": false, "error": "Invalid URL format. Must start with http:// or https://"}'
        return 1
    fi
    
    # Security check for internal/localhost URLs in production
    if [[ "$context" != "direct-system" ]]; then
        if [[ "$url" =~ (localhost|127\.0\.0\.1|0\.0\.0\.0|10\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[0-1])\.) ]]; then
            log::warn "⚠️  WARNING: Request to internal/localhost URL blocked for security"
            echo '{"success": false, "error": "Requests to internal/localhost URLs are not allowed in sandbox mode"}'
            return 1
        fi
    fi
    
    # Build curl command
    local curl_command="curl"
    local curl_args=()
    
    # Method
    if [[ "$method" != "GET" ]]; then
        curl_args+=("-X" "$method")
    fi
    
    # Timeout
    curl_args+=("--max-time" "$timeout")
    
    # User agent
    curl_args+=("-A" "$user_agent")
    
    # SSL verification
    if [[ "$verify_ssl" == "false" ]]; then
        curl_args+=("--insecure")
    fi
    
    # Follow redirects
    if [[ "$follow_redirects" == "true" ]]; then
        curl_args+=("--location")
    fi
    
    # Include response headers
    curl_args+=("--include" "--silent" "--show-error")
    
    # Add custom headers
    if [[ "$headers" != "{}" && "$headers" != "null" ]]; then
        while IFS="=" read -r key value; do
            [[ -z "$key" ]] && continue
            # Remove quotes from jq output
            key=$(echo "$key" | sed 's/^"//;s/"$//')
            value=$(echo "$value" | sed 's/^"//;s/"$//')
            curl_args+=("-H" "$key: $value")
        done < <(echo "$headers" | jq -r 'to_entries[] | "\(.key)=\(.value)"' 2>/dev/null || true)
    fi
    
    # Handle authentication
    if [[ "$auth" != "{}" && "$auth" != "null" ]]; then
        local auth_type
        auth_type=$(echo "$auth" | jq -r '.type // empty')
        
        case "$auth_type" in
            "basic")
                local username password
                username=$(echo "$auth" | jq -r '.username // empty')
                password=$(echo "$auth" | jq -r '.password // empty')
                if [[ -n "$username" && -n "$password" ]]; then
                    curl_args+=("--user" "$username:$password")
                fi
                ;;
            "bearer")
                local token
                token=$(echo "$auth" | jq -r '.token // empty')
                if [[ -n "$token" ]]; then
                    curl_args+=("-H" "Authorization: Bearer $token")
                fi
                ;;
            "api_key")
                local key header
                key=$(echo "$auth" | jq -r '.key // empty')
                header=$(echo "$auth" | jq -r '.header // "X-API-Key"')
                if [[ -n "$key" ]]; then
                    curl_args+=("-H" "$header: $key")
                fi
                ;;
        esac
    fi
    
    # Add request body for POST/PUT/PATCH
    if [[ "$method" =~ ^(POST|PUT|PATCH)$ ]] && [[ -n "$body" && "$body" != "null" ]]; then
        curl_args+=("--data" "$body")
        # Set content-type if not already specified
        if ! echo "$headers" | jq -e '.["Content-Type"] or .["content-type"]' >/dev/null 2>&1; then
            if [[ "$body" =~ ^\{.*\}$ ]]; then
                curl_args+=("-H" "Content-Type: application/json")
            else
                curl_args+=("-H" "Content-Type: application/x-www-form-urlencoded")
            fi
        fi
    fi
    
    # Add URL as final argument
    curl_args+=("$url")
    
    # Execute request
    local start_time end_time duration
    start_time=$(date +%s.%3N)
    
    log::debug "Executing HTTP request: $method $url"
    local response exit_code
    response=$(curl "${curl_args[@]}" 2>&1)
    exit_code=$?
    
    end_time=$(date +%s.%3N)
    duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "unknown")
    
    if [[ $exit_code -eq 0 ]]; then
        # Parse response headers and body
        local headers_end body_start status_line status_code
        headers_end=$(echo "$response" | grep -n "^$" | head -1 | cut -d: -f1)
        
        if [[ -n "$headers_end" ]]; then
            body_start=$((headers_end + 1))
            status_line=$(echo "$response" | head -1)
            
            # Extract status code
            if [[ "$status_line" =~ HTTP/[0-9.]+ ([0-9]+) ]]; then
                status_code="${BASH_REMATCH[1]}"
            else
                status_code="unknown"
            fi
            
            # Extract response headers
            local response_headers
            response_headers=$(echo "$response" | sed -n "2,${headers_end}p" | sed '/^$/d')
            
            # Extract response body
            local response_body
            response_body=$(echo "$response" | tail -n +$body_start)
            
            # Determine if request was successful
            local success="true"
            if [[ "$status_code" =~ ^[45] ]]; then
                success="false"
            fi
            
            # Get response size
            local response_size
            response_size=$(echo -n "$response_body" | wc -c)
            
            # Build JSON response
            cat << EOF
{
  "success": $success,
  "status_code": $status_code,
  "status_text": $(echo "$status_line" | cut -d' ' -f3- | jq -R -s . | tr -d '\n'),
  "headers": $(echo "$response_headers" | awk -F': ' '{if($2) print "\"" $1 "\": \"" $2 "\""}' | sed 's/$/,/' | sed '$s/,$//' | sed '1i{' | sed '$a}'),
  "body": $(echo "$response_body" | jq -R -s .),
  "size": $response_size,
  "duration_ms": $(echo "$duration * 1000" | bc -l 2>/dev/null || echo "null"),
  "method": "$method",
  "url": "$url"
}
EOF
        else
            # No header/body separation found - treat entire response as body
            cat << EOF
{
  "success": true,
  "status_code": 200,
  "status_text": "OK",
  "headers": {},
  "body": $(echo "$response" | jq -R -s .),
  "size": $(echo -n "$response" | wc -c),
  "duration_ms": $(echo "$duration * 1000" | bc -l 2>/dev/null || echo "null"),
  "method": "$method",
  "url": "$url"
}
EOF
        fi
    else
        # Handle curl errors
        local error_message="HTTP request failed"
        case $exit_code in
            6) error_message="Could not resolve host" ;;
            7) error_message="Failed to connect to host" ;;
            28) error_message="Timeout reached" ;;
            35) error_message="SSL handshake failed" ;;
            52) error_message="Empty response from server" ;;
            *) error_message="HTTP request failed (curl exit code: $exit_code)" ;;
        esac
        
        cat << EOF
{
  "success": false,
  "error": "$error_message",
  "curl_exit_code": $exit_code,
  "details": $(echo "$response" | jq -R -s .),
  "duration_ms": $(echo "$duration * 1000" | bc -l 2>/dev/null || echo "null"),
  "method": "$method",
  "url": "$url"
}
EOF
        return 1
    fi
    
    return 0
}

# Export function
export -f tool_http_request::execute