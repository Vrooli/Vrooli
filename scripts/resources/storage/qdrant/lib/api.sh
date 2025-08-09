#!/usr/bin/env bash
# Qdrant API Management
# Functions for interacting with Qdrant REST API

#######################################
# Make authenticated API request to Qdrant
# Arguments:
#   $1 - HTTP method (GET, POST, PUT, DELETE)
#   $2 - API endpoint (without base URL)
#   $3 - Request body (optional, for POST/PUT)
# Outputs: API response
# Returns: 0 on success, 1 on failure
#######################################
qdrant::api::request() {
    local method="$1"
    local endpoint="$2"
    local body="${3:-}"
    
    local url="${QDRANT_BASE_URL}${endpoint}"
    local curl_cmd=(
        "curl" "-s" "-X" "$method"
        "--max-time" "${QDRANT_API_TIMEOUT}"
        "-H" "Content-Type: application/json"
    )
    
    # Add authentication header if API key is configured
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        curl_cmd+=("-H" "api-key: ${QDRANT_API_KEY}")
    fi
    
    # Add request body for POST/PUT requests
    if [[ -n "$body" ]]; then
        curl_cmd+=("-d" "$body")
    fi
    
    # Add URL
    curl_cmd+=("$url")
    
    # Execute request
    "${curl_cmd[@]}"
}

#######################################
# Get Qdrant cluster information
# Outputs: Cluster information JSON
# Returns: 0 on success, 1 on failure
#######################################
qdrant::api::get_cluster_info() {
    qdrant::api::request "GET" "/cluster"
}

#######################################
# Get Qdrant telemetry information
# Outputs: Telemetry JSON
# Returns: 0 on success, 1 on failure
#######################################
qdrant::api::get_telemetry() {
    qdrant::api::request "GET" "/telemetry"
}

#######################################
# Get health status
# Returns: 0 if healthy, 1 if not
#######################################
qdrant::api::health_check() {
    local response
    response=$(qdrant::api::request "GET" "/" 2>/dev/null)
    
    if [[ $? -eq 0 && -n "$response" ]]; then
        # Check if response contains version information
        if echo "$response" | jq -e '.version' >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Get detailed health status with error information
# Outputs: Detailed health status message
# Returns: 0 if healthy, 1 if not
#######################################
qdrant::api::detailed_health_check() {
    local auth_header=""
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        auth_header="-H \"api-key: ${QDRANT_API_KEY}\""
    fi
    
    local temp_output=$(mktemp)
    local http_code
    local curl_exit_code
    
    # Use curl with detailed options to capture both status and errors
    http_code=$(eval "curl -s -w \"%{http_code}\" --max-time ${QDRANT_API_TIMEOUT} --connect-timeout 3 ${auth_header} --output \"$temp_output\" \"${QDRANT_BASE_URL}/\" 2>/dev/null")
    curl_exit_code=$?
    
    # Clean up temp file
    rm -f "$temp_output"
    
    # Analyze the results and provide specific error messages
    case $curl_exit_code in
        0)
            # Success - check HTTP status code
            case $http_code in
                200)
                    echo "✅ Healthy"
                    return 0
                    ;;
                401)
                    echo "⚠️  Service running but authentication failed"
                    return 1
                    ;;
                403)
                    echo "⚠️  Service running but access denied"
                    return 1
                    ;;
                5*)
                    echo "⚠️  Service running but internal error (HTTP $http_code)"
                    return 1
                    ;;
                *)
                    echo "⚠️  Service responded with HTTP $http_code"
                    return 1
                    ;;
            esac
            ;;
        7)
            echo "⚠️  Service not running (connection refused)"
            return 1
            ;;
        28)
            echo "⚠️  Service running but health check timed out"
            return 1
            ;;
        6)
            echo "⚠️  Network connectivity problem (DNS resolution failed)"
            return 1
            ;;
        *)
            echo "⚠️  Health check failed (curl error $curl_exit_code)"
            return 1
            ;;
    esac
}

#######################################
# Get service metrics
# Outputs: Service metrics JSON
# Returns: 0 on success, 1 on failure
#######################################
qdrant::api::get_metrics() {
    qdrant::api::request "GET" "/metrics"
}

#######################################
# Test API connectivity with detailed diagnostics
# Outputs: Diagnostic information
# Returns: 0 if successful, 1 if failed
#######################################
qdrant::api::diagnose_connectivity() {
    echo "=== Qdrant API Connectivity Diagnostics ==="
    echo
    
    # Test basic connectivity
    echo "1. Testing basic connectivity..."
    if qdrant::api::health_check; then
        echo "   ✅ API is responding"
    else
        echo "   ❌ API is not responding"
        echo "   Details: $(qdrant::api::detailed_health_check)"
        return 1
    fi
    
    # Test authentication
    echo
    echo "2. Testing authentication..."
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        echo "   ℹ️  API key is configured"
        local auth_test
        if qdrant::api::request "GET" "/cluster" >/dev/null 2>&1; then
            echo "   ✅ Authentication successful"
        else
            echo "   ❌ Authentication failed"
            return 1
        fi
    else
        echo "   ⚠️  No API key configured (unauthenticated access)"
    fi
    
    # Test cluster information
    echo
    echo "3. Testing cluster information retrieval..."
    local cluster_info
    cluster_info=$(qdrant::api::get_cluster_info 2>/dev/null)
    if [[ $? -eq 0 && -n "$cluster_info" ]]; then
        echo "   ✅ Cluster information retrieved"
        local peer_count
        peer_count=$(echo "$cluster_info" | jq -r '.result.peers | length' 2>/dev/null || echo "unknown")
        echo "   ℹ️  Cluster peers: $peer_count"
    else
        echo "   ❌ Failed to retrieve cluster information"
        return 1
    fi
    
    # Test telemetry
    echo
    echo "4. Testing telemetry endpoint..."
    local telemetry
    telemetry=$(qdrant::api::get_telemetry 2>/dev/null)
    if [[ $? -eq 0 && -n "$telemetry" ]]; then
        echo "   ✅ Telemetry endpoint accessible"
    else
        echo "   ⚠️  Telemetry endpoint not accessible (may be disabled)"
    fi
    
    echo
    echo "=== Diagnostics Complete ==="
    return 0
}

#######################################
# Get API version information
# Outputs: Version string
# Returns: 0 on success, 1 on failure
#######################################
qdrant::api::get_version() {
    local response
    response=$(qdrant::api::request "GET" "/" 2>/dev/null)
    
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "$response" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown"
        return 0
    else
        echo "unknown"
        return 1
    fi
}

#######################################
# Wait for API to become available
# Arguments:
#   $1 - Maximum wait time in seconds (optional)
# Returns: 0 if API becomes available, 1 if timeout
#######################################
qdrant::api::wait_for_availability() {
    local max_wait="${1:-$QDRANT_STARTUP_MAX_WAIT}"
    local elapsed=0
    
    while [[ $elapsed -lt $max_wait ]]; do
        if qdrant::api::health_check; then
            return 0
        fi
        
        sleep "${QDRANT_STARTUP_WAIT_INTERVAL}"
        elapsed=$((elapsed + QDRANT_STARTUP_WAIT_INTERVAL))
    done
    
    return 1
}

#######################################
# Get comprehensive service status
# Outputs: Service status information
# Returns: 0 always (informational)
#######################################
qdrant::api::get_service_status() {
    echo "=== Qdrant Service Status ==="
    echo
    
    # Basic connectivity
    echo "API Health: $(qdrant::api::detailed_health_check)"
    
    # Version information
    local version
    version=$(qdrant::api::get_version)
    echo "Version: $version"
    
    # Cluster information
    if qdrant::api::health_check; then
        local cluster_info
        cluster_info=$(qdrant::api::get_cluster_info 2>/dev/null)
        if [[ $? -eq 0 && -n "$cluster_info" ]]; then
            local peer_count
            peer_count=$(echo "$cluster_info" | jq -r '.result.peers | length' 2>/dev/null || echo "unknown")
            echo "Cluster Peers: $peer_count"
        fi
    fi
    
    echo
}