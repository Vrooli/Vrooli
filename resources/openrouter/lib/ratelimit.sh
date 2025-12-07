#!/bin/bash
# OpenRouter rate limit handling and request queuing

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_LIB_DIR="${APP_ROOT}/resources/openrouter/lib"
OPENROUTER_RESOURCE_DIR="${APP_ROOT}/resources/openrouter"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${OPENROUTER_LIB_DIR}/core.sh"
source "${OPENROUTER_RESOURCE_DIR}/config/defaults.sh"

# Rate limit tracking directory
RATE_LIMIT_DIR="${var_ROOT_DIR}/data/openrouter/ratelimits"

# Initialize rate limit tracking
openrouter::ratelimit::init() {
    mkdir -p "$RATE_LIMIT_DIR"
    
    # Create rate limit state file if it doesn't exist
    local state_file="$RATE_LIMIT_DIR/state.json"
    if [[ ! -f "$state_file" ]]; then
        echo '{"last_request": 0, "request_count": 0, "window_start": 0, "rate_limited": false, "retry_after": 0}' > "$state_file"
    fi
}

# Check if we're rate limited
openrouter::ratelimit::check() {
    openrouter::ratelimit::init
    
    local state_file="$RATE_LIMIT_DIR/state.json"
    local current_time=$(date +%s)
    
    # Read current state
    local rate_limited=$(jq -r '.rate_limited // false' "$state_file")
    local retry_after=$(jq -r '.retry_after // 0' "$state_file")
    
    # Check if we're still rate limited
    if [[ "$rate_limited" == "true" ]]; then
        if [[ $current_time -lt $retry_after ]]; then
            local wait_time=$((retry_after - current_time))
            log::warn "Rate limited. Retry after $wait_time seconds"
            return 1
        else
            # Rate limit period has passed
            jq '.rate_limited = false | .retry_after = 0' "$state_file" > "${state_file}.tmp" && mv "${state_file}.tmp" "$state_file"
        fi
    fi
    
    return 0
}

# Update rate limit state based on response headers
openrouter::ratelimit::update() {
    local response_headers="${1}"
    local response_code="${2}"
    
    openrouter::ratelimit::init
    
    local state_file="$RATE_LIMIT_DIR/state.json"
    local current_time=$(date +%s)
    
    # Check for rate limit response (429)
    if [[ "$response_code" == "429" ]]; then
        # Extract retry-after header if available
        local retry_after_header=$(echo "$response_headers" | grep -i "retry-after:" | cut -d: -f2 | tr -d ' \r')
        local retry_after=$((current_time + ${retry_after_header:-60}))  # Default to 60 seconds
        
        jq --argjson retry_after "$retry_after" \
           '.rate_limited = true | .retry_after = $retry_after' "$state_file" > "${state_file}.tmp" && mv "${state_file}.tmp" "$state_file"
        
        log::warn "Rate limit hit. Will retry after $(date -d @$retry_after 2>/dev/null || date -r $retry_after)"
        return 1
    fi
    
    # Update successful request tracking
    jq --argjson current_time "$current_time" \
       '.last_request = $current_time | .request_count += 1' "$state_file" > "${state_file}.tmp" && mv "${state_file}.tmp" "$state_file"
    
    return 0
}

# Request queue implementation
openrouter::ratelimit::queue_request() {
    local request_data="${1}"
    local priority="${2:-normal}"  # high, normal, low
    
    local queue_dir="$RATE_LIMIT_DIR/queue"
    mkdir -p "$queue_dir"
    
    # Generate unique request ID
    local request_id=$(date +%s%N)-$$
    local queue_file="$queue_dir/${priority}-${request_id}.json"
    
    # Save request to queue
    echo "$request_data" > "$queue_file"
    
    log::info "Request queued with ID: $request_id"
    echo "$request_id"
}

# Process queued requests
openrouter::ratelimit::process_queue() {
    local queue_dir="$RATE_LIMIT_DIR/queue"
    
    if [[ ! -d "$queue_dir" ]]; then
        return 0
    fi
    
    # Process high priority first, then normal, then low
    shopt -s nullglob
    for priority in high normal low; do
        for queue_file in "$queue_dir"/${priority}-*.json; do
            [[ -f "$queue_file" ]] || continue
            
            # Check if we can make a request
            if ! openrouter::ratelimit::check; then
                log::info "Rate limited. Queue processing paused."
                return 1
            fi
            
            # Process the request
            local request_data=$(cat "$queue_file")
            log::info "Processing queued request: $(basename "$queue_file")"
            
            # Execute the request (would need to be implemented based on request type)
            # For now, just remove from queue
            rm -f "$queue_file"
        done
    done
    
    return 0
}

# Execute request with rate limit handling
openrouter::ratelimit::execute_with_retry() {
    local url="${1}"
    local method="${2:-GET}"
    local data="${3:-}"
    local headers="${4:-}"
    local max_retries="${5:-3}"
    local backoff_base="${6:-2}"  # Exponential backoff base in seconds
    
    local retry_count=0
    local backoff=$backoff_base
    
    while [[ $retry_count -lt $max_retries ]]; do
        # Check if we're rate limited
        if ! openrouter::ratelimit::check; then
            # Queue the request for later
            local request_json=$(jq -n \
                --arg url "$url" \
                --arg method "$method" \
                --arg data "$data" \
                --arg headers "$headers" \
                '{url: $url, method: $method, data: $data, headers: $headers}')
            
            openrouter::ratelimit::queue_request "$request_json" "normal"
            return 2  # Special code for queued
        fi
        
        # Execute the request with header capture
        local temp_headers=$(mktemp)
        local response
        local curl_cmd="curl -s -D $temp_headers"
        
        # Add method
        case "$method" in
            POST) curl_cmd="$curl_cmd -X POST" ;;
            PUT) curl_cmd="$curl_cmd -X PUT" ;;
            DELETE) curl_cmd="$curl_cmd -X DELETE" ;;
        esac
        
        # Add headers
        if [[ -n "$headers" ]]; then
            while IFS= read -r header; do
                [[ -n "$header" ]] && curl_cmd="$curl_cmd -H '$header'"
            done <<< "$headers"
        fi
        
        # Add data for POST/PUT
        if [[ -n "$data" ]] && [[ "$method" =~ ^(POST|PUT)$ ]]; then
            curl_cmd="$curl_cmd -d '$data'"
        fi
        
        # Execute request
        response=$(eval "$curl_cmd '$url'" 2>/dev/null)
        local curl_exit_code=$?
        
        # Get response code from headers
        local response_code=$(grep "^HTTP/" "$temp_headers" | tail -1 | cut -d' ' -f2)
        
        # Update rate limit state
        if openrouter::ratelimit::update "$(<$temp_headers)" "$response_code"; then
            # Success
            rm -f "$temp_headers"
            echo "$response"
            return 0
        fi
        
        # Rate limited - apply exponential backoff
        ((retry_count++))
        if [[ $retry_count -lt $max_retries ]]; then
            log::info "Retry $retry_count/$max_retries after ${backoff}s backoff"
            sleep $backoff
            backoff=$((backoff * backoff_base))
        fi
        
        rm -f "$temp_headers"
    done
    
    log::error "Max retries ($max_retries) exceeded"
    return 1
}

# Get rate limit status
openrouter::ratelimit::status() {
    openrouter::ratelimit::init
    
    local state_file="$RATE_LIMIT_DIR/state.json"
    local queue_dir="$RATE_LIMIT_DIR/queue"
    
    # Count queued requests
    local queued_high=$(find "$queue_dir" -name "high-*.json" 2>/dev/null | wc -l)
    local queued_normal=$(find "$queue_dir" -name "normal-*.json" 2>/dev/null | wc -l)
    local queued_low=$(find "$queue_dir" -name "low-*.json" 2>/dev/null | wc -l)
    
    # Get current state
    local state=$(cat "$state_file")
    
    # Add queue counts to state
    echo "$state" | jq \
        --argjson high "$queued_high" \
        --argjson normal "$queued_normal" \
        --argjson low "$queued_low" \
        '. + {queue: {high: $high, normal: $normal, low: $low, total: ($high + $normal + $low)}}'
}

# Export functions
export -f openrouter::ratelimit::init
export -f openrouter::ratelimit::check
export -f openrouter::ratelimit::update
export -f openrouter::ratelimit::queue_request
export -f openrouter::ratelimit::process_queue
export -f openrouter::ratelimit::execute_with_retry
export -f openrouter::ratelimit::status
