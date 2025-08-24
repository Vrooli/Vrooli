#!/usr/bin/env bash
# Qdrant API Client - Centralized HTTP client with standardized error handling
# Eliminates duplication of qdrant::api::request across multiple files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
API_CLIENT_DIR="${APP_ROOT}/resources/qdrant/lib"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"

#######################################
# Core API request function (replaces qdrant::api::request)
# Arguments:
#   $1 - HTTP method (GET, POST, PUT, DELETE)
#   $2 - API endpoint (starting with /)  
#   $3 - Request body (optional, JSON string)
# Returns: 0 on success, 1 on failure
# Outputs: Response body on success, error message on failure
#######################################
qdrant::client::request() {
    local method="$1"
    local endpoint="$2"
    local body="${3:-}"
    
    if [[ -z "$method" ]] || [[ -z "$endpoint" ]]; then
        log::error "Method and endpoint required"
        return 1
    fi
    
    local url="${QDRANT_BASE_URL}${endpoint}"
    local headers="Content-Type: application/json"
    
    # Add authentication header if API key is configured
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        headers="${headers}\napi-key: ${QDRANT_API_KEY}"
    fi
    
    # Use http-utils framework for the request
    if [[ -n "$body" ]]; then
        http::request "$method" "$url" "$body" "$headers"
    else
        http::request "$method" "$url" "" "$headers"
    fi
}

#######################################
# Check if API response indicates success
# Arguments:
#   $1 - Response body
# Returns: 0 if success, 1 if error
#######################################
qdrant::client::is_success() {
    local response="$1"
    
    if [[ -z "$response" ]]; then
        return 1
    fi
    
    # Check for success indicators
    # 1. Explicit success status
    if echo "$response" | grep -q '"status":"ok"' 2>/dev/null; then
        return 0
    fi
    
    # 2. Result field present (standard Qdrant API response)
    if echo "$response" | jq -e '.result' >/dev/null 2>&1; then
        return 0
    fi
    
    # 3. Valid JSON without error fields (for root endpoint, version info, etc.)
    if echo "$response" | jq . >/dev/null 2>&1; then
        # Check if it's an error response
        if echo "$response" | jq -e '.error // .status.error' >/dev/null 2>&1; then
            return 1  # Has error field
        fi
        # Valid JSON without error - consider it success
        return 0
    fi
    
    return 1
}

#######################################
# Extract error message from API response
# Arguments:
#   $1 - Response body
# Outputs: Error message
#######################################
qdrant::client::extract_error() {
    local response="$1"
    
    if [[ -z "$response" ]]; then
        echo "Empty response"
        return
    fi
    
    # Try to extract structured error
    local error_msg
    error_msg=$(echo "$response" | jq -r '.status.error // .error.message // .message // "Unknown error"' 2>/dev/null)
    
    if [[ "$error_msg" == "null" ]] || [[ -z "$error_msg" ]]; then
        error_msg="API request failed"
    fi
    
    echo "$error_msg"
}

#######################################
# Handle API response with standardized error logging
# Arguments:
#   $1 - Response body
#   $2 - Context message (e.g., "create collection")
# Returns: 0 on success, 1 on error
#######################################
qdrant::client::handle_response() {
    local response="$1"
    local context="${2:-API operation}"
    
    if qdrant::client::is_success "$response"; then
        echo "$response"
        return 0
    else
        local error_msg
        error_msg=$(qdrant::client::extract_error "$response")
        log::error "${context} failed: $error_msg"
        return 1
    fi
}

#######################################
# GET request with error handling
# Arguments:
#   $1 - API endpoint
#   $2 - Context for error messages (optional)
# Returns: 0 on success, 1 on failure
# Outputs: Response body on success
#######################################
qdrant::client::get() {
    local endpoint="$1"
    local context="${2:-GET request}"
    
    local response
    response=$(qdrant::client::request "GET" "$endpoint" 2>/dev/null || true)
    
    qdrant::client::handle_response "$response" "$context"
}

#######################################
# POST request with error handling
# Arguments:
#   $1 - API endpoint
#   $2 - Request body (JSON)
#   $3 - Context for error messages (optional)
# Returns: 0 on success, 1 on failure
# Outputs: Response body on success
#######################################
qdrant::client::post() {
    local endpoint="$1"
    local body="$2"
    local context="${3:-POST request}"
    
    if [[ -z "$body" ]]; then
        log::error "POST request requires body"
        return 1
    fi
    
    local response
    response=$(qdrant::client::request "POST" "$endpoint" "$body" 2>/dev/null || true)
    
    qdrant::client::handle_response "$response" "$context"
}

#######################################
# PUT request with error handling
# Arguments:
#   $1 - API endpoint
#   $2 - Request body (JSON)
#   $3 - Context for error messages (optional)
# Returns: 0 on success, 1 on failure
# Outputs: Response body on success
#######################################
qdrant::client::put() {
    local endpoint="$1"
    local body="$2"
    local context="${3:-PUT request}"
    
    if [[ -z "$body" ]]; then
        log::error "PUT request requires body"
        return 1
    fi
    
    local response
    response=$(qdrant::client::request "PUT" "$endpoint" "$body" 2>/dev/null || true)
    
    qdrant::client::handle_response "$response" "$context"
}

#######################################
# DELETE request with error handling
# Arguments:
#   $1 - API endpoint
#   $2 - Context for error messages (optional)
# Returns: 0 on success, 1 on failure
# Outputs: Response body on success
#######################################
qdrant::client::delete() {
    local endpoint="$1"
    local context="${2:-DELETE request}"
    
    local response
    response=$(qdrant::client::request "DELETE" "$endpoint" 2>/dev/null || true)
    
    qdrant::client::handle_response "$response" "$context"
}

#######################################
# Health check with standardized handling
# Returns: 0 if healthy, 1 if not
#######################################
qdrant::client::health_check() {
    local response
    response=$(qdrant::client::get "/" "health check" 2>/dev/null || true)
    
    if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
        # Check if response contains version information (indicates Qdrant is running)
        if echo "$response" | jq -e '.version' >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Get collections list with error handling
# Returns: 0 on success, 1 on failure
# Outputs: Collections response JSON
#######################################
qdrant::client::get_collections() {
    qdrant::client::get "/collections" "list collections"
}

#######################################
# Get single collection info with error handling
# Arguments:
#   $1 - Collection name
# Returns: 0 on success, 1 on failure  
# Outputs: Collection info JSON
#######################################
qdrant::client::get_collection_info() {
    local collection="$1"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name required"
        return 1
    fi
    
    qdrant::client::get "/collections/${collection}" "get collection info"
}

#######################################
# Create collection with validation
# Arguments:
#   $1 - Collection name
#   $2 - Vector size
#   $3 - Distance metric
# Returns: 0 on success, 1 on failure
#######################################
qdrant::client::create_collection() {
    local collection="$1"
    local vector_size="$2"
    local distance="${3:-Cosine}"
    
    if [[ -z "$collection" ]] || [[ -z "$vector_size" ]]; then
        log::error "Collection name and vector size required"
        return 1
    fi
    
    local config
    config=$(jq -n \
        --argjson size "$vector_size" \
        --arg distance "$distance" \
        '{
            vectors: {
                size: $size,
                distance: $distance
            }
        }')
    
    qdrant::client::put "/collections/${collection}" "$config" "create collection"
}

#######################################
# Delete collection with error handling
# Arguments:
#   $1 - Collection name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::client::delete_collection() {
    local collection="$1"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name required"
        return 1
    fi
    
    qdrant::client::delete "/collections/${collection}" "delete collection"
}

#######################################
# Search in collection with embedding vector
# Arguments:
#   $1 - Collection name
#   $2 - Search vector (JSON array)
#   $3 - Limit (optional, default: 10)
#   $4 - Score threshold (optional)
# Returns: 0 on success, 1 on failure
# Outputs: Search results JSON
#######################################
qdrant::client::search() {
    local collection="$1"
    local vector="$2"
    local limit="${3:-10}"
    local score_threshold="${4:-}"
    
    if [[ -z "$collection" ]] || [[ -z "$vector" ]]; then
        log::error "Collection and vector required"
        return 1
    fi
    
    # Build search request
    local search_request
    search_request=$(jq -n \
        --argjson vector "$vector" \
        --argjson limit "$limit" \
        '{
            vector: $vector,
            limit: $limit,
            with_payload: true,
            with_vector: false
        }')
    
    # Add score threshold if specified
    if [[ -n "$score_threshold" ]]; then
        search_request=$(echo "$search_request" | jq --argjson threshold "$score_threshold" '.score_threshold = $threshold')
    fi
    
    qdrant::client::post "/collections/${collection}/points/search" "$search_request" "search collection"
}

#######################################
# Search in collection with pre-built search request
# Arguments:
#   $1 - Collection name
#   $2 - Search request object (JSON)
# Returns: 0 on success, 1 on failure
# Outputs: Search results JSON  
#######################################
qdrant::client::search_raw() {
    local collection="$1"
    local search_request="$2"
    
    if [[ -z "$collection" ]] || [[ -z "$search_request" ]]; then
        log::error "Collection and search request required"
        return 1
    fi
    
    qdrant::client::post "/collections/${collection}/points/search" "$search_request" "search collection"
}

#######################################
# Insert points into collection
# Arguments:
#   $1 - Collection name
#   $2 - Points data (JSON)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::client::insert_points() {
    local collection="$1"
    local points="$2"
    
    if [[ -z "$collection" ]] || [[ -z "$points" ]]; then
        log::error "Collection and points data required"
        return 1
    fi
    
    qdrant::client::put "/collections/${collection}/points" "$points" "insert points"
}

#######################################
# Upsert single point into collection
# Arguments:
#   $1 - Collection name
#   $2 - Point ID
#   $3 - Vector (JSON array)
#   $4 - Payload (JSON object)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::client::upsert_point() {
    local collection="$1"
    local point_id="$2"
    local vector="$3"
    local payload="$4"
    
    if [[ -z "$collection" ]] || [[ -z "$point_id" ]] || [[ -z "$vector" ]]; then
        log::error "Collection, point ID, and vector required"
        return 1
    fi
    
    local points_data
    points_data=$(jq -n \
        --arg id "$point_id" \
        --argjson vector "$vector" \
        --argjson payload "${payload:-{}}" \
        '{
            points: [{
                id: $id,
                vector: $vector,
                payload: $payload
            }]
        }')
    
    qdrant::client::insert_points "$collection" "$points_data"
}

#######################################
# Get cluster information
# Returns: 0 on success, 1 on failure
# Outputs: Cluster info JSON
#######################################
qdrant::client::get_cluster_info() {
    # Try cluster endpoint first, fallback to root
    local response
    response=$(qdrant::client::get "/cluster" "get cluster info" 2>/dev/null || true)
    
    if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
        echo "$response"
        return 0
    fi
    
    # Fallback to root endpoint for single-node setups
    response=$(qdrant::client::get "/" "get basic info" 2>/dev/null || true)
    
    if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
        # Transform simple response to cluster format
        echo "$response" | jq '{
            result: {
                version: .version,
                peers: [],
                peer_id: "single-node"
            }
        }'
        return 0
    fi
    
    return 1
}

#######################################
# Create collection snapshot
# Arguments:
#   $1 - Collection name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::client::create_snapshot() {
    local collection="$1"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name required"
        return 1
    fi
    
    qdrant::client::post "/collections/${collection}/snapshots" "{}" "create snapshot"
}

#######################################
# List collection snapshots
# Arguments:
#   $1 - Collection name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::client::list_snapshots() {
    local collection="$1"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name required"
        return 1
    fi
    
    qdrant::client::get "/collections/${collection}/snapshots" "list snapshots"
}

#######################################
# Delete collection snapshot
# Arguments:
#   $1 - Collection name
#   $2 - Snapshot name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::client::delete_snapshot() {
    local collection="$1"
    local snapshot="$2"
    
    if [[ -z "$collection" ]] || [[ -z "$snapshot" ]]; then
        log::error "Collection and snapshot name required"
        return 1
    fi
    
    qdrant::client::delete "/collections/${collection}/snapshots/${snapshot}" "delete snapshot"
}

#######################################
# Get Qdrant version information
# Returns: 0 on success, 1 on failure
# Outputs: Version string
#######################################
qdrant::client::get_version() {
    local response
    response=$(qdrant::client::get "/" "get version" 2>/dev/null || true)
    
    if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
        echo "$response" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown"
        return 0
    else
        echo "unknown"
        return 1
    fi
}

#######################################
# Test API connectivity with comprehensive checks
# Returns: 0 if all tests pass, 1 if any fail
#######################################
qdrant::client::test_connectivity() {
    log::info "Testing Qdrant API connectivity..."
    
    # Test 1: Basic health check
    if ! qdrant::client::health_check; then
        log::error "Basic health check failed"
        return 1
    fi
    log::debug "✅ Basic health check passed"
    
    # Test 2: Collections endpoint
    local collections_response
    collections_response=$(qdrant::client::get_collections 2>/dev/null || true)
    if [[ $? -ne 0 ]]; then
        log::error "Collections endpoint test failed"
        return 1
    fi
    log::debug "✅ Collections endpoint accessible"
    
    # Test 3: Cluster endpoint (optional)
    local cluster_response
    cluster_response=$(qdrant::client::get_cluster_info 2>/dev/null || true)
    if [[ $? -eq 0 ]]; then
        log::debug "✅ Cluster endpoint accessible"
    else
        log::debug "⚠️  Cluster endpoint not available (single-node setup)"
    fi
    
    log::success "API connectivity tests passed"
    return 0
}

#######################################
# Convenience function for backwards compatibility
# This maintains the existing qdrant::api::request interface
#######################################
qdrant::api::request() {
    qdrant::client::request "$@"
}