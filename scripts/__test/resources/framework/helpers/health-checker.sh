#!/bin/bash
# ====================================================================
# Standardized Health Check Functions
# ====================================================================
#
# Provides consistent health checking with retry logic, exponential
# backoff, and standardized timeout handling for all resources.
#
# Functions:
#   - check_resource_health_standardized() - Main health check with retries
#   - get_health_timeout()                 - Get appropriate timeout for resource
#   - perform_http_health_check()          - Standardized HTTP health check
#   - perform_health_check_with_retry()    - Health check with retry logic
#
# ====================================================================

# Source health check configuration
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
if [[ -f "$SCRIPT_DIR/config/health-checks.conf" ]]; then
    source "$SCRIPT_DIR/config/health-checks.conf"
else
    # Fallback defaults if config file is missing
    DEFAULT_HEALTH_TIMEOUT=8
    HEALTH_CHECK_RETRIES=2
    HEALTH_CHECK_RETRY_DELAY=3
    EXTENDED_HEALTH_TIMEOUT=15
    QUICK_HEALTH_TIMEOUT=5
    HEALTH_CHECK_DEBUG=false
fi

# Source logging functions
if [[ -f "$SCRIPT_DIR/helpers/logging.sh" ]]; then
    source "$SCRIPT_DIR/helpers/logging.sh"
fi

# Get appropriate timeout for a specific resource
get_health_timeout() {
    local resource="$1"
    
    # Check if resource needs extended timeout
    if [[ " ${EXTENDED_TIMEOUT_RESOURCES[*]} " == *" $resource "* ]]; then
        echo "$EXTENDED_HEALTH_TIMEOUT"
        return
    fi
    
    # Check if resource needs quick timeout
    if [[ " ${QUICK_TIMEOUT_RESOURCES[*]} " == *" $resource "* ]]; then
        echo "$QUICK_HEALTH_TIMEOUT"
        return
    fi
    
    # Default timeout
    echo "$DEFAULT_HEALTH_TIMEOUT"
}

# Perform standardized HTTP health check
perform_http_health_check() {
    local url="$1"
    local timeout="$2"
    local expected_status="${3:-200}"
    
    if [[ "$HEALTH_CHECK_DEBUG" == "true" ]]; then
        log_debug "Health check: $url (timeout: ${timeout}s)"
    fi
    
    local response
    local http_code
    
    # Perform HTTP request with timeout
    response=$(curl -s --max-time "$timeout" -w "%{http_code}" "$url" 2>/dev/null)
    local curl_exit_code=$?
    
    # Extract HTTP status code (last 3 characters)
    if [[ ${#response} -ge 3 ]]; then
        http_code="${response: -3}"
        response="${response%???}"  # Remove status code from response
    else
        http_code="000"
    fi
    
    # Evaluate health check result
    if [[ $curl_exit_code -eq 0 ]]; then
        if [[ "$http_code" -ge 200 && "$http_code" -lt 400 ]]; then
            if [[ "$HEALTH_CHECK_DEBUG" == "true" ]]; then
                log_debug "Health check passed: HTTP $http_code"
            fi
            echo "healthy"
            return 0
        else
            if [[ "$HEALTH_CHECK_DEBUG" == "true" ]]; then
                log_debug "Health check failed: HTTP $http_code"
            fi
            echo "unhealthy"
            return 1
        fi
    else
        if [[ "$HEALTH_CHECK_DEBUG" == "true" ]]; then
            log_debug "Health check failed: curl exit code $curl_exit_code"
        fi
        if [[ $curl_exit_code -eq 124 || $curl_exit_code -eq 28 ]]; then
            echo "timeout"
        else
            echo "unreachable"
        fi
        return 1
    fi
}

# Perform health check with retry logic and exponential backoff
perform_health_check_with_retry() {
    local resource="$1"
    local health_check_func="$2"
    shift 2
    local args=("$@")
    
    local timeout
    timeout=$(get_health_timeout "$resource")
    
    local attempt=0
    local max_attempts=$((HEALTH_CHECK_RETRIES + 1))
    local delay="$HEALTH_CHECK_RETRY_DELAY"
    
    while [[ $attempt -lt $max_attempts ]]; do
        if [[ "$HEALTH_CHECK_DEBUG" == "true" && $attempt -gt 0 ]]; then
            log_debug "Health check retry $attempt/$HEALTH_CHECK_RETRIES for $resource"
        fi
        
        # Perform the health check
        local result
        result=$($health_check_func "$timeout" "${args[@]}" 2>/dev/null)
        local exit_code=$?
        
        if [[ $exit_code -eq 0 && "$result" == "healthy" ]]; then
            if [[ "$HEALTH_CHECK_DEBUG" == "true" && $attempt -gt 0 ]]; then
                log_debug "Health check succeeded on retry $attempt for $resource"
            fi
            echo "healthy"
            return 0
        fi
        
        attempt=$((attempt + 1))
        
        # If we have more attempts, wait before retry
        if [[ $attempt -lt $max_attempts ]]; then
            if [[ "$HEALTH_CHECK_DEBUG" == "true" ]]; then
                log_debug "Health check failed for $resource (attempt $attempt), retrying in ${delay}s..."
            fi
            sleep "$delay"
            # Exponential backoff: double the delay for next retry
            delay=$((delay * 2))
        fi
    done
    
    # All attempts failed
    if [[ "$HEALTH_CHECK_DEBUG" == "true" ]]; then
        log_debug "Health check failed for $resource after $max_attempts attempts"
    fi
    echo "unhealthy"
    return 1
}

# Standardized health check for any resource
check_resource_health_standardized() {
    local resource="$1"
    local port="$2"
    
    if [[ "$HEALTH_CHECK_DEBUG" == "true" ]]; then
        log_debug "Starting standardized health check for $resource"
    fi
    
    # Resource-specific health check logic
    case "$resource" in
        "ollama")
            perform_health_check_with_retry "$resource" "check_ollama_health_std" "$port"
            ;;
        "whisper")
            perform_health_check_with_retry "$resource" "check_whisper_health_std" "$port"
            ;;
        "n8n")
            perform_health_check_with_retry "$resource" "check_n8n_health_std" "$port"
            ;;
        "browserless")
            perform_health_check_with_retry "$resource" "check_browserless_health_std" "$port"
            ;;
        "minio")
            perform_health_check_with_retry "$resource" "check_minio_health_std" "$port"
            ;;
        "qdrant")
            perform_health_check_with_retry "$resource" "check_qdrant_health_std" "$port"
            ;;
        "searxng")
            perform_health_check_with_retry "$resource" "check_searxng_health_std" "$port"
            ;;
        "vault")
            perform_health_check_with_retry "$resource" "check_vault_health_std" "$port"
            ;;
        "windmill")
            perform_health_check_with_retry "$resource" "check_windmill_health_std" "$port"
            ;;
        "unstructured-io")
            perform_health_check_with_retry "$resource" "check_unstructured_health_std" "$port"
            ;;
        "questdb")
            perform_health_check_with_retry "$resource" "check_questdb_health_std" "$port"
            ;;
        "redis")
            perform_health_check_with_retry "$resource" "check_redis_health_std" "$port"
            ;;
        *)
            perform_health_check_with_retry "$resource" "check_generic_health_std" "$port"
            ;;
    esac
}

# Standardized health check functions for specific resources
check_ollama_health_std() {
    local timeout="$1"
    local port="$2"
    
    local result
    result=$(perform_http_health_check "http://localhost:${port}/api/tags" "$timeout")
    
    # Additional validation for Ollama - check if response contains models
    if [[ "$result" == "healthy" ]]; then
        local response
        response=$(curl -s --max-time "$timeout" "http://localhost:${port}/api/tags" 2>/dev/null)
        if echo "$response" | jq -e '.models' >/dev/null 2>&1; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "$result"
    fi
}

check_whisper_health_std() {
    local timeout="$1" 
    local port="$2"
    
    # Whisper doesn't have a /health endpoint, so check /openapi.json instead
    local result
    result=$(perform_http_health_check "http://localhost:${port}/openapi.json" "$timeout")
    
    # If openapi.json fails, fallback to root endpoint
    if [[ "$result" != "healthy" ]]; then
        result=$(perform_http_health_check "http://localhost:${port}/" "$timeout")
    fi
    
    echo "$result"
}

check_n8n_health_std() {
    local timeout="$1"
    local port="$2"
    perform_http_health_check "http://localhost:${port}/healthz" "$timeout"
}

check_browserless_health_std() {
    local timeout="$1"
    local port="$2"
    
    local result
    result=$(perform_http_health_check "http://localhost:${port}/pressure" "$timeout")
    
    # Additional validation for Browserless - check if response contains pressure info
    if [[ "$result" == "healthy" ]]; then
        local response
        response=$(curl -s --max-time "$timeout" "http://localhost:${port}/pressure" 2>/dev/null)
        if echo "$response" | jq -e '.pressure' >/dev/null 2>&1; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "$result"
    fi
}

check_minio_health_std() {
    local timeout="$1"
    local port="$2"
    perform_http_health_check "http://localhost:${port}/minio/health/live" "$timeout"
}

check_qdrant_health_std() {
    local timeout="$1"
    local port="$2"
    perform_http_health_check "http://localhost:${port}/" "$timeout"
}

check_searxng_health_std() {
    local timeout="$1"
    local port="$2"
    perform_http_health_check "http://localhost:${port}/healthz" "$timeout"
}

check_vault_health_std() {
    local timeout="$1"
    local port="$2"
    perform_http_health_check "http://localhost:${port}/v1/sys/health" "$timeout"
}

check_windmill_health_std() {
    local timeout="$1"
    local port="$2"
    perform_http_health_check "http://localhost:${port}/api/version" "$timeout"
}

check_unstructured_health_std() {
    local timeout="$1"
    local port="$2"
    perform_http_health_check "http://localhost:${port}/healthcheck" "$timeout"
}

check_questdb_health_std() {
    local timeout="$1"
    local port="$2"
    
    # Try multiple endpoints for QuestDB
    local result
    result=$(perform_http_health_check "http://localhost:${port}/status" "$timeout")
    if [[ "$result" != "healthy" ]]; then
        result=$(perform_http_health_check "http://localhost:${port}/" "$timeout")
    fi
    echo "$result"
}

check_redis_health_std() {
    local timeout="$1"
    local port="$2"
    
    # If no port provided, try to find Redis container and get its port
    if [[ -z "$port" ]]; then
        # Find Redis container (might be named differently like vrooli-redis-resource)
        local redis_container
        redis_container=$(docker ps --format "{{.Names}}" 2>/dev/null | grep -E "(redis|^redis$)" | head -1)
        
        if [[ -n "$redis_container" ]]; then
            # Get the port mapping
            local port_mapping
            port_mapping=$(docker port "$redis_container" 2>/dev/null | head -1)
            if [[ -n "$port_mapping" ]]; then
                # Extract host port from format: 6379/tcp -> 0.0.0.0:6380
                port=$(echo "$port_mapping" | sed 's/.*://g')
            fi
        fi
        
        # Fallback to default Redis port
        if [[ -z "$port" ]]; then
            port="6380"  # Vrooli's default Redis port
        fi
    fi
    
    # Redis uses its own protocol, not HTTP. Test with Redis PING command
    # Using netcat to send Redis protocol PING command
    local response
    response=$(timeout "$timeout" bash -c 'echo -e "*1\r\n\$4\r\nPING\r\n" | nc localhost '"$port"' 2>/dev/null' | head -1 | tr -d '\r\n')
    if [[ "$response" == "+PONG" ]]; then
        echo "healthy"
    else
        echo "unhealthy"
    fi
}

check_generic_health_std() {
    local timeout="$1"
    local port="$2"
    
    # Try common health endpoints
    for endpoint in "/" "/health" "/healthz" "/ping"; do
        local result
        result=$(perform_http_health_check "http://localhost:${port}${endpoint}" "$timeout")
        if [[ "$result" == "healthy" ]]; then
            echo "healthy"
            return 0
        fi
    done
    
    echo "unreachable"
}