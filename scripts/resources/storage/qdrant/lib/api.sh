#!/usr/bin/env bash
# Qdrant API Interface
# Provides HTTP request handling for Qdrant API operations using shared frameworks

set -euo pipefail

# Get directory of this script
QDRANT_API_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "${QDRANT_API_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"

# Source configuration
# shellcheck disable=SC1091
source "${QDRANT_API_DIR}/../config/defaults.sh"
qdrant::export_config

#######################################
# Make authenticated API request to Qdrant
# Arguments:
#   $1 - HTTP method (GET, POST, PUT, DELETE)
#   $2 - API endpoint (e.g., /collections)
#   $3 - Request body (optional, JSON string)
#   $4 - Additional headers (optional)
# Returns:
#   Response body via stdout
#   0 for successful request, 1 for failure
#######################################
qdrant::api::request() {
    local method="${1:-GET}"
    local endpoint="${2:-/}"
    local body="${3:-}"
    local extra_headers="${4:-}"
    
    # Ensure endpoint starts with /
    if [[ "$endpoint" != /* ]]; then
        endpoint="/$endpoint"
    fi
    
    # Build full URL
    local url="${QDRANT_BASE_URL}${endpoint}"
    
    # Build curl command
    local curl_args=("-s" "-X" "$method")
    
    # Add API key if configured
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        curl_args+=("-H" "api-key: ${QDRANT_API_KEY}")
    fi
    
    # Add extra headers if provided
    if [[ -n "$extra_headers" ]]; then
        curl_args+=("-H" "$extra_headers")
    fi
    
    # Add body for write operations
    if [[ -n "$body" ]]; then
        curl_args+=("-H" "Content-Type: application/json")
        curl_args+=("-d" "$body")
    fi
    
    # Add timeout
    curl_args+=("--max-time" "30")
    
    # Make the request
    local response
    response=$(curl "${curl_args[@]}" "$url" 2>/dev/null || echo "ERROR: Connection failed")
    
    # Log debug info if verbose
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        log::debug "API Request: $method $url"
        [[ -n "$response" ]] && log::debug "Response: ${response:0:200}..."
    fi
    
    # Check for connection errors
    if [[ "$response" == "ERROR: Connection failed" ]]; then
        log::error "Failed to connect to Qdrant API at $url"
        return 1
    fi
    
    # Output response
    echo "$response"
    
    # Return success - let caller check response content for errors
    return 0
}

#######################################
# Test Qdrant API connectivity
# Returns:
#   0 if API is accessible
#   1 if API is not accessible
#######################################
qdrant::api::test() {
    local response
    
    # Try to get basic info
    response=$(qdrant::api::request "GET" "/" 2>/dev/null || true)
    
    # Check if we got a valid Qdrant response
    if echo "$response" | grep -q "qdrant\|version"; then
        log::success "Qdrant API is accessible"
        return 0
    elif echo "$response" | grep -q "ERROR:"; then
        log::error "Qdrant API connection failed"
        return 1
    else
        log::error "Qdrant API returned unexpected response: ${response:0:100}..."
        return 1
    fi
}

#######################################
# Get Qdrant cluster info
# Returns:
#   Cluster information as JSON
#######################################
qdrant::api::cluster_info() {
    local response
    response=$(qdrant::api::request "GET" "/cluster" 2>/dev/null || true)
    
    # Check for error responses
    if echo "$response" | grep -q "ERROR:"; then
        log::error "Failed to get cluster info"
        return 1
    fi
    
    echo "$response"
    return 0
}

#######################################
# Get Qdrant telemetry data
# Returns:
#   Telemetry data as JSON
#######################################
qdrant::api::telemetry() {
    local response
    response=$(qdrant::api::request "GET" "/telemetry" 2>/dev/null || true)
    
    if [[ $? -eq 0 ]] || [[ $? -eq 200 ]]; then
        echo "$response"
        return 0
    fi
    
    log::error "Failed to get telemetry data"
    return 1
}

#######################################
# List all collections
# Returns:
#   List of collections as JSON
#######################################
qdrant::api::list_collections() {
    local response
    response=$(qdrant::api::request "GET" "/collections" 2>/dev/null || true)
    
    # Check if we got a valid response
    if [[ -n "$response" ]] && echo "$response" | jq -e '.result' >/dev/null 2>&1; then
        echo "$response"
        return 0
    fi
    
    log::error "Failed to list collections"
    return 1
}

#######################################
# Get collection info
# Arguments:
#   $1 - Collection name
# Returns:
#   Collection info as JSON
#######################################
qdrant::api::get_collection() {
    local collection="${1:-}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local response
    response=$(qdrant::api::request "GET" "/collections/${collection}" 2>/dev/null || true)
    
    # Check for error responses
    if echo "$response" | grep -q "ERROR:"; then
        log::error "Failed to get collection info: $collection"
        return 1
    elif echo "$response" | grep -q "not found"; then
        log::error "Collection '$collection' not found"
        return 1
    elif echo "$response" | grep -q '"status":"ok"\|"result"'; then
        echo "$response"
        return 0
    else
        log::error "Failed to get collection info: $collection"
        return 1
    fi
}

#######################################
# Create a new collection
# Arguments:
#   $1 - Collection name
#   $2 - Vector size
#   $3 - Distance metric (Cosine, Euclid, Dot)
# Returns:
#   Creation response as JSON
#######################################
qdrant::api::create_collection() {
    local collection="${1:-}"
    local vector_size="${2:-1536}"
    local distance="${3:-Cosine}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local config
    config=$(cat <<EOF
{
    "vectors": {
        "size": ${vector_size},
        "distance": "${distance}"
    }
}
EOF
    )
    
    local response
    response=$(qdrant::api::request "PUT" "/collections/${collection}" "$config" 2>/dev/null || true)
    
    # Check for error responses
    if echo "$response" | grep -q "ERROR:"; then
        log::error "Failed to create collection: $collection"
        return 1
    elif echo "$response" | grep -q '"status":"ok"\|"result"'; then
        echo "$response"
        return 0
    else
        log::error "Failed to create collection: $collection"
        echo "$response"
        return 1
    fi
}

#######################################
# Delete a collection
# Arguments:
#   $1 - Collection name
# Returns:
#   Deletion response as JSON
#######################################
qdrant::api::delete_collection() {
    local collection="${1:-}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local response
    response=$(qdrant::api::request "DELETE" "/collections/${collection}" 2>/dev/null || true)
    local http_code=$?
    
    if [[ $http_code -eq 0 ]] || [[ $http_code -eq 200 ]] || [[ $http_code -eq 204 ]]; then
        echo "$response"
        return 0
    elif [[ $http_code -eq 404 ]]; then
        log::error "Collection '$collection' not found"
        return 1
    else
        log::error "Failed to delete collection (HTTP code: $http_code)"
        return 1
    fi
}

#######################################
# Create a snapshot of a collection
# Arguments:
#   $1 - Collection name
# Returns:
#   Snapshot info as JSON
#######################################
qdrant::api::create_snapshot() {
    local collection="${1:-}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local response
    response=$(qdrant::api::request "POST" "/collections/${collection}/snapshots" 2>/dev/null || true)
    local http_code=$?
    
    if [[ $http_code -eq 0 ]] || [[ $http_code -eq 200 ]] || [[ $http_code -eq 201 ]]; then
        echo "$response"
        return 0
    else
        log::error "Failed to create snapshot (HTTP code: $http_code)"
        return 1
    fi
}

#######################################
# List snapshots for a collection
# Arguments:
#   $1 - Collection name
# Returns:
#   List of snapshots as JSON
#######################################
qdrant::api::list_snapshots() {
    local collection="${1:-}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local response
    response=$(qdrant::api::request "GET" "/collections/${collection}/snapshots" 2>/dev/null || true)
    local http_code=$?
    
    if [[ $http_code -eq 0 ]] || [[ $http_code -eq 200 ]]; then
        echo "$response"
        return 0
    else
        log::error "Failed to list snapshots (HTTP code: $http_code)"
        return 1
    fi
}

#######################################
# Export all API functions
#######################################
export -f qdrant::api::request
export -f qdrant::api::test
export -f qdrant::api::cluster_info
export -f qdrant::api::telemetry
export -f qdrant::api::list_collections
export -f qdrant::api::get_collection
export -f qdrant::api::create_collection
export -f qdrant::api::delete_collection
export -f qdrant::api::create_snapshot
export -f qdrant::api::list_snapshots