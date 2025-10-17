#!/usr/bin/env bash
################################################################################
# Validate URL Tool Executor
# 
# Validates URLs and checks accessibility with detailed analysis
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Execute validate URL tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   JSON result with URL validation information
#######################################
tool_validate_url::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Parse arguments
    local url check_accessibility timeout verify_ssl
    url=$(echo "$arguments" | jq -r '.url')
    check_accessibility=$(echo "$arguments" | jq -r '.check_accessibility // true')
    timeout=$(echo "$arguments" | jq -r '.timeout // 10')
    verify_ssl=$(echo "$arguments" | jq -r '.verify_ssl // true')
    
    # Validate required parameters
    if [[ "$url" == "null" || -z "$url" ]]; then
        echo '{"success": false, "error": "URL is required"}'
        return 1
    fi
    
    # URL format validation
    local url_valid="false"
    local url_scheme="" url_host="" url_port="" url_path=""
    local validation_errors=()
    
    # Basic URL format check
    if [[ "$url" =~ ^(https?|ftp|ftps|file)://([^:/]+)(:([0-9]+))?(/.*)?$ ]]; then
        url_scheme="${BASH_REMATCH[1]}"
        url_host="${BASH_REMATCH[2]}"
        url_port="${BASH_REMATCH[4]}"
        url_path="${BASH_REMATCH[5]}"
        url_valid="true"
        
        # Default ports
        if [[ -z "$url_port" ]]; then
            case "$url_scheme" in
                "http") url_port="80" ;;
                "https") url_port="443" ;;
                "ftp") url_port="21" ;;
                "ftps") url_port="990" ;;
            esac
        fi
    else
        validation_errors+=("Invalid URL format")
    fi
    
    # Host validation
    if [[ -n "$url_host" ]]; then
        # Check for valid hostname/IP format
        if [[ "$url_host" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            # IP address validation
            IFS='.' read -ra IP_PARTS <<< "$url_host"
            for part in "${IP_PARTS[@]}"; do
                if [[ $part -gt 255 || $part -lt 0 ]]; then
                    validation_errors+=("Invalid IP address")
                    break
                fi
            done
        else
            # Hostname validation
            if ! [[ "$url_host" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$ ]]; then
                validation_errors+=("Invalid hostname format")
            fi
        fi
    fi
    
    # Port validation
    if [[ -n "$url_port" && ($url_port -lt 1 || $url_port -gt 65535) ]]; then
        validation_errors+=("Invalid port number")
    fi
    
    # Build validation result
    local validation_result="true"
    if [[ ${#validation_errors[@]} -gt 0 ]]; then
        validation_result="false"
        url_valid="false"
    fi
    
    # Accessibility check
    local accessibility_result=""
    local response_headers=""
    local status_code=""
    local response_time=""
    local ssl_info=""
    local server_info=""
    
    if [[ "$check_accessibility" == "true" && "$url_valid" == "true" && "$url_scheme" =~ ^https?$ ]]; then
        # Security check for internal/localhost URLs in sandbox mode
        if [[ "$context" != "direct-system" ]]; then
            if [[ "$url_host" =~ ^(localhost|127\.0\.0\.1|0\.0\.0\.0)$ ]] || \
               [[ "$url_host" =~ ^10\. ]] || \
               [[ "$url_host" =~ ^192\.168\. ]] || \
               [[ "$url_host" =~ ^172\.(1[6-9]|2[0-9]|3[0-1])\. ]]; then
                accessibility_result="blocked"
                log::debug "Accessibility check blocked for internal/localhost URL"
            fi
        fi
        
        if [[ "$accessibility_result" != "blocked" ]]; then
            # Build curl command for HEAD request
            local curl_args=("--head" "--silent" "--max-time" "$timeout")
            
            if [[ "$verify_ssl" == "false" ]]; then
                curl_args+=("--insecure")
            fi
            
            # Include timing information
            curl_args+=("--write-out" "RESPONSE_TIME:%{time_total}")
            
            # Execute accessibility check
            local start_time end_time curl_output curl_exit_code
            start_time=$(date +%s.%3N)
            curl_output=$(curl "${curl_args[@]}" "$url" 2>/dev/null || echo "")
            curl_exit_code=$?
            end_time=$(date +%s.%3N)
            
            if [[ $curl_exit_code -eq 0 ]]; then
                accessibility_result="accessible"
                
                # Parse response
                response_time=$(echo "$curl_output" | grep "RESPONSE_TIME:" | cut -d: -f2 || echo "unknown")
                curl_output=$(echo "$curl_output" | grep -v "RESPONSE_TIME:")
                
                # Extract status code
                local status_line
                status_line=$(echo "$curl_output" | head -1)
                if [[ "$status_line" =~ HTTP/[0-9.]+\ ([0-9]+) ]]; then
                    status_code="${BASH_REMATCH[1]}"
                fi
                
                # Extract important headers
                response_headers=$(echo "$curl_output" | grep -i -E "^(server|content-type|content-length|last-modified|cache-control|x-powered-by):" || echo "")
                
                # Extract server info
                server_info=$(echo "$response_headers" | grep -i "^server:" | cut -d: -f2- | sed 's/^ *//' || echo "unknown")
                
                # SSL information for HTTPS URLs
                if [[ "$url_scheme" == "https" ]]; then
                    local ssl_output
                    ssl_output=$(curl --insecure --silent --max-time 5 --write-out "SSL_VERIFY:%{ssl_verify_result}" --output /dev/null "$url" 2>/dev/null || echo "")
                    local ssl_verify_result
                    ssl_verify_result=$(echo "$ssl_output" | grep "SSL_VERIFY:" | cut -d: -f2 || echo "unknown")
                    
                    if [[ "$ssl_verify_result" == "0" ]]; then
                        ssl_info="valid"
                    else
                        ssl_info="invalid"
                    fi
                fi
                
            else
                accessibility_result="inaccessible"
                
                # Provide specific error messages based on curl exit code
                case $curl_exit_code in
                    6) accessibility_result="dns_resolution_failed" ;;
                    7) accessibility_result="connection_failed" ;;
                    28) accessibility_result="timeout" ;;
                    35) accessibility_result="ssl_handshake_failed" ;;
                    60|77) accessibility_result="ssl_certificate_problem" ;;
                esac
            fi
        fi
    fi
    
    # URL analysis
    local url_analysis=""
    if [[ "$url_valid" == "true" ]]; then
        local is_secure="false"
        local uses_standard_port="true"
        local has_query_params="false"
        local has_fragment="false"
        
        if [[ "$url_scheme" == "https" ]]; then
            is_secure="true"
        fi
        
        if [[ ("$url_scheme" == "http" && "$url_port" != "80") || \
              ("$url_scheme" == "https" && "$url_port" != "443") ]]; then
            uses_standard_port="false"
        fi
        
        if [[ "$url" == *"?"* ]]; then
            has_query_params="true"
        fi
        
        if [[ "$url" == *"#"* ]]; then
            has_fragment="true"
        fi
        
        url_analysis=$(cat << EOF
{
  "is_secure": $is_secure,
  "uses_standard_port": $uses_standard_port,
  "has_query_parameters": $has_query_params,
  "has_fragment": $has_fragment,
  "estimated_safety": $(if [[ "$is_secure" == "true" && "$uses_standard_port" == "true" ]]; then echo "\"high\""; elif [[ "$is_secure" == "true" ]]; then echo "\"medium\""; else echo "\"low\""; fi)
}
EOF
        )
    fi
    
    # Build comprehensive response
    local errors_json="[]"
    if [[ ${#validation_errors[@]} -gt 0 ]]; then
        errors_json="[$(printf '"%s",' "${validation_errors[@]}" | sed 's/,$//')]"
    fi
    
    cat << EOF
{
  "success": $validation_result,
  "url": "$url",
  "valid": $url_valid,
  "errors": $errors_json,
  "components": {
    "scheme": "$url_scheme",
    "host": "$url_host",
    "port": "$url_port",
    "path": "$url_path"
  },
  "accessibility": {
    "checked": $check_accessibility,
    "result": "$accessibility_result",
    "status_code": $([ -n "$status_code" ] && echo "\"$status_code\"" || echo "null"),
    "response_time": $([ -n "$response_time" ] && echo "$response_time" || echo "null"),
    "server": $([ -n "$server_info" ] && echo "\"$server_info\"" || echo "null"),
    "ssl_status": $([ -n "$ssl_info" ] && echo "\"$ssl_info\"" || echo "null")
  },
  "analysis": $([ -n "$url_analysis" ] && echo "$url_analysis" || echo "null")
}
EOF
    
    return 0
}

# Export function
export -f tool_validate_url::execute