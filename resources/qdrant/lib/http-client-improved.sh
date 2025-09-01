#!/usr/bin/env bash
################################################################################
# Improved HTTP Client for Qdrant - Connection Pooling & Retry Logic
# 
# Provides resilient HTTP communication with Qdrant API including:
# - Connection pooling and reuse
# - Exponential backoff retry logic  
# - Circuit breaker pattern
# - Request/response logging
# - Performance metrics
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source logging utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
source "${var_LIB_UTILS_DIR}/log.sh" 2>/dev/null || echo() { printf "%s\n" "$*"; }

# Configuration
QDRANT_BASE_URL="${QDRANT_BASE_URL:-http://localhost:6333}"
HTTP_TIMEOUT="${HTTP_TIMEOUT:-30}"
MAX_RETRIES="${MAX_RETRIES:-3}"
INITIAL_RETRY_DELAY="${INITIAL_RETRY_DELAY:-1}"
MAX_RETRY_DELAY="${MAX_RETRY_DELAY:-30}"
CIRCUIT_BREAKER_THRESHOLD="${CIRCUIT_BREAKER_THRESHOLD:-5}"
CIRCUIT_BREAKER_RESET_TIME="${CIRCUIT_BREAKER_RESET_TIME:-60}"

# Global state for connection pooling and circuit breaker
declare -A CONNECTION_STATS
declare -A CIRCUIT_BREAKER_STATE
CURL_CONFIG_FILE=""
REQUEST_ID_COUNTER=0

#######################################
# Initialize HTTP client
#######################################
http_client::init() {
    # Create curl config for connection reuse
    CURL_CONFIG_FILE=$(mktemp)
    cat > "$CURL_CONFIG_FILE" << 'EOF'
# Qdrant HTTP Client Configuration
user-agent = "Qdrant-Embeddings/1.0"
connect-timeout = 10
max-time = 30
retry = 0
tcp-nodelay
tcp-keepalive
keepalive-time = 300
max-redirs = 3
location
compressed
EOF
    
    # Initialize circuit breaker
    CIRCUIT_BREAKER_STATE["failures"]=0
    CIRCUIT_BREAKER_STATE["state"]="closed"  # closed, open, half-open
    CIRCUIT_BREAKER_STATE["next_attempt"]=0
    
    log::debug "HTTP client initialized with connection pooling"
}

#######################################
# Cleanup HTTP client resources
#######################################
http_client::cleanup() {
    if [[ -n "$CURL_CONFIG_FILE" && -f "$CURL_CONFIG_FILE" ]]; then
        rm -f "$CURL_CONFIG_FILE"
    fi
}

#######################################
# Check circuit breaker state
# Returns: 0 if request allowed, 1 if blocked
#######################################
http_client::check_circuit_breaker() {
    local current_time=$(date +%s)
    local failures=${CIRCUIT_BREAKER_STATE["failures"]}
    local state=${CIRCUIT_BREAKER_STATE["state"]}
    local next_attempt=${CIRCUIT_BREAKER_STATE["next_attempt"]}
    
    case "$state" in
        "closed")
            return 0
            ;;
        "open")
            if [[ $current_time -ge $next_attempt ]]; then
                CIRCUIT_BREAKER_STATE["state"]="half-open"
                log::debug "Circuit breaker: half-open state"
                return 0
            else
                log::warn "Circuit breaker: blocking request (open state)"
                return 1
            fi
            ;;
        "half-open")
            return 0
            ;;
    esac
}

#######################################
# Update circuit breaker after request
# Arguments:
#   $1 - success (true/false)
#######################################
http_client::update_circuit_breaker() {
    local success="$1"
    local current_time=$(date +%s)
    
    if [[ "$success" == "true" ]]; then
        # Reset on success
        CIRCUIT_BREAKER_STATE["failures"]=0
        CIRCUIT_BREAKER_STATE["state"]="closed"
        log::debug "Circuit breaker: reset to closed state"
    else
        # Increment failures
        local failures=$((${CIRCUIT_BREAKER_STATE["failures"]} + 1))
        CIRCUIT_BREAKER_STATE["failures"]=$failures
        
        if [[ $failures -ge $CIRCUIT_BREAKER_THRESHOLD ]]; then
            CIRCUIT_BREAKER_STATE["state"]="open"
            CIRCUIT_BREAKER_STATE["next_attempt"]=$((current_time + CIRCUIT_BREAKER_RESET_TIME))
            log::warn "Circuit breaker: opened due to $failures failures"
        fi
    fi
}

#######################################
# Perform HTTP request with retry logic
# Arguments:
#   $1 - HTTP method (GET, POST, PUT, DELETE)
#   $2 - URL path (relative to base URL)
#   $3 - Request body (optional, for POST/PUT)
#   $4 - Description for logging (optional)
# Returns: 0 on success, 1 on failure
# Outputs: Response body
#######################################
http_client::request() {
    local method="$1"
    local path="$2"
    local body="${3:-}"
    local description="${4:-$method $path}"
    
    # Initialize if not done
    if [[ -z "$CURL_CONFIG_FILE" ]]; then
        http_client::init
    fi
    
    # Check circuit breaker
    if ! http_client::check_circuit_breaker; then
        return 1
    fi
    
    local url="${QDRANT_BASE_URL}${path}"
    local request_id=$((++REQUEST_ID_COUNTER))
    local attempt=1
    local delay=$INITIAL_RETRY_DELAY
    
    log::debug "HTTP[$request_id]: $description -> $method $url"
    
    while [[ $attempt -le $((MAX_RETRIES + 1)) ]]; do
        local start_time=$(date +%s%3N)
        local curl_args=()
        
        # Build curl command
        curl_args+=(
            "--config" "$CURL_CONFIG_FILE"
            "--silent"
            "--fail"
            "--show-error"
            "--request" "$method"
            "--header" "Content-Type: application/json"
        )
        
        # Add body for POST/PUT
        if [[ -n "$body" && ("$method" == "POST" || "$method" == "PUT") ]]; then
            curl_args+=("--data" "$body")
        fi
        
        curl_args+=("$url")
        
        # Execute request
        local response=""
        local curl_exit_code=0
        
        if response=$(curl "${curl_args[@]}" 2>/dev/null); then
            curl_exit_code=0
        else
            curl_exit_code=$?
        fi
        
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        
        # Check result
        if [[ $curl_exit_code -eq 0 && -n "$response" ]]; then
            # Success
            log::debug "HTTP[$request_id]: ✅ Success in ${duration}ms (attempt $attempt)"
            
            # Update stats
            CONNECTION_STATS["${method}_success"]=$((${CONNECTION_STATS["${method}_success"]:-0} + 1))
            CONNECTION_STATS["total_time"]=$((${CONNECTION_STATS["total_time"]:-0} + duration))
            
            # Update circuit breaker
            http_client::update_circuit_breaker "true"
            
            # Output response
            echo "$response"
            return 0
        else
            # Failure
            log::debug "HTTP[$request_id]: ❌ Failure (attempt $attempt/$((MAX_RETRIES + 1)), exit=$curl_exit_code)"
            
            # Update stats
            CONNECTION_STATS["${method}_failure"]=$((${CONNECTION_STATS["${method}_failure"]:-0} + 1))
            
            # Check if we should retry
            if [[ $attempt -le $MAX_RETRIES ]]; then
                log::debug "HTTP[$request_id]: Retrying in ${delay}s..."
                sleep "$delay"
                
                # Exponential backoff with jitter
                local jitter=$((RANDOM % 1000))  # 0-999ms
                delay=$(( (delay * 2) + (jitter / 1000) ))
                
                # Cap delay
                if [[ $delay -gt $MAX_RETRY_DELAY ]]; then
                    delay=$MAX_RETRY_DELAY
                fi
                
                ((attempt++))
            else
                # Max retries exceeded
                log::error "HTTP[$request_id]: Failed after $((MAX_RETRIES + 1)) attempts"
                
                # Update circuit breaker
                http_client::update_circuit_breaker "false"
                
                return 1
            fi
        fi
    done
}

#######################################
# Convenience methods for common operations
#######################################

http_client::get() {
    http_client::request "GET" "$1" "" "${2:-GET $1}"
}

http_client::post() {
    http_client::request "POST" "$1" "$2" "${3:-POST $1}"
}

http_client::put() {
    http_client::request "PUT" "$1" "$2" "${3:-PUT $1}"
}

http_client::delete() {
    http_client::request "DELETE" "$1" "" "${2:-DELETE $1}"
}

#######################################
# Get connection statistics
#######################################
http_client::stats() {
    echo "=== HTTP Client Statistics ==="
    echo "Circuit Breaker: ${CIRCUIT_BREAKER_STATE["state"]} (failures: ${CIRCUIT_BREAKER_STATE["failures"]})"
    echo
    
    local total_requests=0
    local total_success=0
    local total_failures=0
    
    for method in GET POST PUT DELETE; do
        local success=${CONNECTION_STATS["${method}_success"]:-0}
        local failure=${CONNECTION_STATS["${method}_failure"]:-0}
        local total=$((success + failure))
        
        if [[ $total -gt 0 ]]; then
            local success_rate=$(( (success * 100) / total ))
            echo "$method: $total requests ($success success, $failure failures) - ${success_rate}% success rate"
        fi
        
        total_requests=$((total_requests + total))
        total_success=$((total_success + success))
        total_failures=$((total_failures + failure))
    done
    
    if [[ $total_requests -gt 0 ]]; then
        local overall_success_rate=$(( (total_success * 100) / total_requests ))
        local avg_time=$((${CONNECTION_STATS["total_time"]:-0} / total_success))
        echo
        echo "Overall: $total_requests requests - ${overall_success_rate}% success rate"
        echo "Average response time: ${avg_time}ms"
    fi
}

#######################################
# Reset statistics
#######################################
http_client::reset_stats() {
    unset CONNECTION_STATS
    declare -gA CONNECTION_STATS
    log::info "HTTP client statistics reset"
}

# Cleanup on exit
trap http_client::cleanup EXIT