#!/usr/bin/env bash
################################################################################
# Download File Tool Executor
# 
# Downloads files from URLs with progress tracking and validation
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Execute download file tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   JSON result with download information
#######################################
tool_download_file::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Parse arguments
    local url destination overwrite timeout verify_ssl headers progress
    url=$(echo "$arguments" | jq -r '.url')
    destination=$(echo "$arguments" | jq -r '.destination')
    overwrite=$(echo "$arguments" | jq -r '.overwrite // false')
    timeout=$(echo "$arguments" | jq -r '.timeout // 300')
    verify_ssl=$(echo "$arguments" | jq -r '.verify_ssl // true')
    headers=$(echo "$arguments" | jq -r '.headers // {}')
    progress=$(echo "$arguments" | jq -r '.progress // true')
    
    # Validate required parameters
    if [[ "$url" == "null" || -z "$url" ]]; then
        echo '{"success": false, "error": "URL is required"}'
        return 1
    fi
    
    if [[ "$destination" == "null" || -z "$destination" ]]; then
        echo '{"success": false, "error": "Destination path is required"}'
        return 1
    fi
    
    # Validate URL format
    if ! [[ "$url" =~ ^https?:// ]]; then
        echo '{"success": false, "error": "Invalid URL format. Must start with http:// or https://"}'
        return 1
    fi
    
    # Security check for internal/localhost URLs in sandbox mode
    if [[ "$context" != "direct-system" ]]; then
        if [[ "$url" =~ (localhost|127\.0\.0\.1|0\.0\.0\.0|10\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[0-1])\.) ]]; then
            log::warn "⚠️  WARNING: Download from internal/localhost URL blocked for security"
            echo '{"success": false, "error": "Downloads from internal/localhost URLs are not allowed in sandbox mode"}'
            return 1
        fi
    fi
    
    # Check if destination file already exists
    if [[ -f "$destination" ]] && [[ "$overwrite" != "true" ]]; then
        echo '{"success": false, "error": "Destination file already exists. Set overwrite=true to replace it."}'
        return 1
    fi
    
    # Create destination directory if it doesn't exist
    local dest_dir
    dest_dir=$(dirname "$destination")
    if [[ ! -d "$dest_dir" ]]; then
        if ! mkdir -p "$dest_dir"; then
            echo '{"success": false, "error": "Cannot create destination directory: '"$dest_dir"'"}'
            return 1
        fi
    fi
    
    # Check write permissions
    if ! touch "$destination" 2>/dev/null; then
        echo '{"success": false, "error": "Cannot write to destination: '"$destination"'"}'
        return 1
    fi
    
    # Build curl command for download
    local curl_args=()
    
    # Output to file
    curl_args+=("-o" "$destination")
    
    # Timeout
    curl_args+=("--max-time" "$timeout")
    
    # SSL verification
    if [[ "$verify_ssl" == "false" ]]; then
        curl_args+=("--insecure")
    fi
    
    # Follow redirects
    curl_args+=("--location")
    
    # Progress bar
    if [[ "$progress" == "true" ]]; then
        curl_args+=("--progress-bar")
    else
        curl_args+=("--silent")
    fi
    
    # Show errors
    curl_args+=("--show-error")
    
    # Continue partial downloads
    curl_args+=("--continue-at" "-")
    
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
    
    # Add URL as final argument
    curl_args+=("$url")
    
    # Get initial file info for HEAD request
    local content_length content_type last_modified
    local head_response
    head_response=$(curl --head --silent --max-time 10 "${curl_args[@]:2}" 2>/dev/null || echo "")
    
    if [[ -n "$head_response" ]]; then
        content_length=$(echo "$head_response" | grep -i "content-length:" | head -1 | awk '{print $2}' | tr -d '\r\n' || echo "unknown")
        content_type=$(echo "$head_response" | grep -i "content-type:" | head -1 | cut -d' ' -f2- | tr -d '\r\n' || echo "unknown")
        last_modified=$(echo "$head_response" | grep -i "last-modified:" | head -1 | cut -d' ' -f2- | tr -d '\r\n' || echo "unknown")
    else
        content_length="unknown"
        content_type="unknown"  
        last_modified="unknown"
    fi
    
    # Execute download
    local start_time end_time duration
    start_time=$(date +%s.%3N)
    
    log::info "Downloading file from: $url"
    log::info "Destination: $destination"
    if [[ "$content_length" != "unknown" ]]; then
        log::info "Expected size: $content_length bytes"
    fi
    
    local curl_output exit_code
    curl_output=$(curl "${curl_args[@]}" 2>&1)
    exit_code=$?
    
    end_time=$(date +%s.%3N)
    duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "unknown")
    
    if [[ $exit_code -eq 0 ]]; then
        # Verify file was downloaded
        if [[ -f "$destination" ]]; then
            # Get actual file info
            local file_size file_hash
            file_size=$(stat -c%s "$destination" 2>/dev/null || echo "0")
            file_hash=$(sha256sum "$destination" 2>/dev/null | cut -d' ' -f1 || echo "unknown")
            
            # Check if download was complete
            local complete="true"
            if [[ "$content_length" != "unknown" && "$content_length" -gt 0 ]]; then
                if [[ "$file_size" -ne "$content_length" ]]; then
                    complete="false"
                fi
            fi
            
            # Determine file type from extension and content-type
            local file_extension
            file_extension=$(basename "$destination" | sed 's/.*\\.//')
            
            cat << EOF
{
  "success": true,
  "url": "$url",
  "destination": "$destination",
  "file_size": $file_size,
  "expected_size": $([ "$content_length" = "unknown" ] && echo "null" || echo "$content_length"),
  "complete": $complete,
  "file_hash": "$file_hash",
  "content_type": "$content_type",
  "file_extension": "$file_extension",
  "last_modified": "$last_modified",
  "duration_seconds": $(echo "$duration" | bc -l 2>/dev/null || echo "null"),
  "download_speed_bps": $(if [[ "$duration" != "unknown" && "$file_size" -gt 0 ]]; then echo "scale=2; $file_size / $duration" | bc -l; else echo "null"; fi)
}
EOF
        else
            echo '{"success": false, "error": "Download completed but file not found at destination"}'
            return 1
        fi
    else
        # Clean up partial download on failure
        [[ -f "$destination" ]] && rm -f "$destination"
        
        # Handle curl errors
        local error_message="Download failed"
        case $exit_code in
            6) error_message="Could not resolve host" ;;
            7) error_message="Failed to connect to host" ;;
            18) error_message="Partial file transfer" ;;
            22) error_message="HTTP error response from server" ;;
            28) error_message="Timeout reached" ;;
            35) error_message="SSL handshake failed" ;;
            36) error_message="Failed to continue download" ;;
            78) error_message="Remote file not found" ;;
            *) error_message="Download failed (curl exit code: $exit_code)" ;;
        esac
        
        cat << EOF
{
  "success": false,
  "error": "$error_message",
  "curl_exit_code": $exit_code,
  "details": $(echo "$curl_output" | jq -R -s .),
  "duration_seconds": $(echo "$duration" | bc -l 2>/dev/null || echo "null"),
  "url": "$url",
  "destination": "$destination"
}
EOF
        return 1
    fi
    
    return 0
}

# Export function
export -f tool_download_file::execute