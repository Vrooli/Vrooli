#!/usr/bin/env bash
# Qdrant Mock - Tier 2 (Stateful)
# 
# Provides stateful Qdrant vector database mock for testing:
# - Collection management (create, delete, list, info)
# - curl API endpoint interception
# - Health check and version endpoints
# - Basic cluster information
# - Error injection for resilience testing
#
# Coverage: ~80% of common Qdrant use cases in 500 lines

# === Configuration ===
declare -gA QDRANT_COLLECTIONS=()      # Collection storage: name -> "size|distance|vectors_count|points_count"
declare -gA QDRANT_CONFIG=(             # Server configuration
    [host]="localhost"
    [rest_port]="6333" 
    [grpc_port]="6334"
    [version]="v1.11.0"
    [health_status]="healthy"
    [server_running]="true"
)

# Debug and error modes
declare -g QDRANT_DEBUG="${QDRANT_DEBUG:-}"
declare -g QDRANT_ERROR_MODE="${QDRANT_ERROR_MODE:-}"

# === Helper Functions ===
qdrant_debug() {
    [[ -n "$QDRANT_DEBUG" ]] && echo "[MOCK:QDRANT] $*" >&2
}

qdrant_check_error() {
    case "$QDRANT_ERROR_MODE" in
        "connection_failed")
            echo "curl: (7) Failed to connect to localhost port ${QDRANT_CONFIG[rest_port]}: Connection refused" >&2
            return 7
            ;;
        "connection_timeout")
            echo "curl: (28) Operation timed out after 10000 milliseconds" >&2
            return 28
            ;;
        "server_not_running")
            echo "Service Unavailable" >&2
            return 22
            ;;
        "http_500")
            echo '{"status":{"error":"Internal Server Error"},"time":0.0}'
            return 22
            ;;
        "http_401")
            echo '{"status":{"error":"Unauthorized"},"time":0.0}'
            return 22
            ;;
    esac
    return 0
}

qdrant_parse_collection_data() {
    local data="$1"
    local size="1536"
    local distance="Cosine"
    
    # Simple JSON parsing for vector config
    if [[ "$data" =~ \"size\"[[:space:]]*:[[:space:]]*([0-9]+) ]]; then
        size="${BASH_REMATCH[1]}"
    fi
    if [[ "$data" =~ \"distance\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
        distance="${BASH_REMATCH[1]}"
    fi
    
    echo "$size|$distance"
}

# === API Response Generators ===
qdrant_generate_health_response() {
    cat <<EOF
{
  "title": "qdrant - vector search engine",
  "version": "${QDRANT_CONFIG[version]}",
  "commit": "1234567890abcdef"
}
EOF
}

qdrant_generate_cluster_response() {
    cat <<EOF
{
  "result": {
    "status": "enabled",
    "peer_id": "00000000-0000-0000-0000-000000000000",
    "peers": {
      "00000000-0000-0000-0000-000000000000": {
        "uri": "http://127.0.0.1:6335"
      }
    },
    "raft_info": {
      "term": 1,
      "commit": 0,
      "pending_operations": 0,
      "leader": "00000000-0000-0000-0000-000000000000",
      "role": "leader",
      "is_voter": true
    }
  },
  "status": "ok",
  "time": 0.000012
}
EOF
}

qdrant_generate_collections_list() {
    local collections_json="["
    local first=true
    
    for collection_name in "${!QDRANT_COLLECTIONS[@]}"; do
        if [[ "$first" != "true" ]]; then
            collections_json+=","
        fi
        collections_json+="{\"name\":\"$collection_name\"}"
        first=false
    done
    collections_json+="]"
    
    cat <<EOF
{
  "result": {
    "collections": $collections_json
  },
  "status": "ok",
  "time": 0.000025
}
EOF
}

qdrant_generate_collection_info() {
    local collection_name="$1"
    local collection_data="${QDRANT_COLLECTIONS[$collection_name]}"
    
    if [[ -z "$collection_data" ]]; then
        echo '{"status":{"error":"Collection not found"},"time":0.0}'
        return
    fi
    
    IFS='|' read -r size distance vectors_count points_count <<< "$collection_data"
    vectors_count="${vectors_count:-0}"
    points_count="${points_count:-0}"
    
    cat <<EOF
{
  "result": {
    "status": "green",
    "optimizer_status": "ok",
    "vectors_count": ${vectors_count},
    "indexed_vectors_count": ${vectors_count},
    "points_count": ${points_count},
    "segments_count": 1,
    "config": {
      "params": {
        "vectors": {
          "size": ${size},
          "distance": "${distance}"
        },
        "shard_number": 1,
        "replication_factor": 1
      }
    }
  },
  "status": "ok",
  "time": 0.000035
}
EOF
}

# === curl Interceptor ===
curl() {
    qdrant_debug "curl called with: $*"
    
    # Check for errors first
    if ! qdrant_check_error; then
        return $?
    fi
    
    # Check server status
    if [[ "${QDRANT_CONFIG[server_running]}" != "true" ]]; then
        echo "curl: (7) Failed to connect to localhost port ${QDRANT_CONFIG[rest_port]}: Connection refused" >&2
        return 7
    fi
    
    # Parse curl arguments
    local url="" method="GET" data="" output_file=""
    local args=("$@") i=0
    
    while [[ $i -lt ${#args[@]} ]]; do
        case "${args[$i]}" in
            -X) method="${args[$((i+1))]}"; i=$((i + 2)) ;;
            --output|-o) output_file="${args[$((i+1))]}"; i=$((i + 2)) ;;
            -d|--data) data="${args[$((i+1))]}"; i=$((i + 2)) ;;
            -H|--header) i=$((i + 2)) ;;  # Skip headers for simplicity
            --max-time|--connect-timeout) i=$((i + 2)) ;;  # Skip timeouts
            -s|--silent|-S|--show-error|-f|--fail) i=$((i + 1)) ;;  # Skip flags
            --write-out|-w) i=$((i + 2)) ;;  # Skip write-out
            http*) url="${args[$i]}"; i=$((i + 1)) ;;
            *) i=$((i + 1)) ;;
        esac
    done
    
    # Check if this is a Qdrant request
    if [[ "$url" != *":6333"* && "$url" != *":6334"* && "$url" != *"/collections"* && "$url" != *"/cluster"* ]]; then
        # Non-Qdrant request - pass through with generic response
        if [[ -n "$output_file" ]]; then
            echo "Mock curl response" > "$output_file"
        else
            echo "Mock curl response"
        fi
        return 0
    fi
    
    # Route Qdrant API request
    qdrant_handle_api_request "$method" "$url" "$data" "$output_file"
    return $?
}

qdrant_handle_api_request() {
    local method="$1" url="$2" data="$3" output_file="$4"
    local response=""
    
    qdrant_debug "API request: $method $url"
    
    # Determine endpoint from URL
    if [[ "$url" == *":6333/" ]] || [[ "$url" == *":6333" ]]; then
        # Health/version endpoint
        response="$(qdrant_generate_health_response)"
        
    elif [[ "$url" == *"/cluster" ]]; then
        # Cluster info endpoint
        response="$(qdrant_generate_cluster_response)"
        
    elif [[ "$url" == *"/collections" ]] && [[ "$url" != *"/collections/"* ]]; then
        # Collections list endpoint
        response="$(qdrant_generate_collections_list)"
        
    elif [[ "$url" == *"/collections/"* ]]; then
        # Individual collection endpoint
        local collection_name="${url#*/collections/}"
        collection_name="${collection_name%%/*}"
        
        case "$method" in
            "GET")
                response="$(qdrant_generate_collection_info "$collection_name")"
                ;;
            "PUT")
                qdrant_handle_create_collection "$collection_name" "$data"
                response='{"result":true,"status":"ok","time":0.012345}'
                ;;
            "DELETE")
                qdrant_handle_delete_collection "$collection_name"
                response='{"result":true,"status":"ok","time":0.002345}'
                ;;
            *)
                response='{"status":"ok","time":0.0}'
                ;;
        esac
    else
        # Unknown endpoint
        echo "Unknown endpoint" >&2
        return 22
    fi
    
    # Output response
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
    
    return 0
}

# === Collection Management ===
qdrant_handle_create_collection() {
    local collection_name="$1" data="$2"
    
    # Check if collection already exists
    if [[ -n "${QDRANT_COLLECTIONS[$collection_name]}" ]]; then
        qdrant_debug "Collection $collection_name already exists"
        return 0
    fi
    
    # Parse collection configuration
    local config
    config="$(qdrant_parse_collection_data "$data")"
    local size distance
    IFS='|' read -r size distance <<< "$config"
    
    # Create collection
    QDRANT_COLLECTIONS[$collection_name]="$size|$distance|0|0"
    qdrant_debug "Created collection: $collection_name ($size, $distance)"
}

qdrant_handle_delete_collection() {
    local collection_name="$1"
    
    if [[ -z "${QDRANT_COLLECTIONS[$collection_name]}" ]]; then
        qdrant_debug "Collection $collection_name does not exist"
        return 0
    fi
    
    unset QDRANT_COLLECTIONS[$collection_name]
    qdrant_debug "Deleted collection: $collection_name"
}

# === Convention-based Test Functions ===
test_qdrant_connection() {
    qdrant_debug "Testing connection..."
    
    local result
    result=$(curl -s "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/" 2>&1)
    
    if [[ "$result" =~ "qdrant - vector search engine" ]]; then
        qdrant_debug "Connection test passed"
        return 0
    else
        qdrant_debug "Connection test failed: $result"
        return 1
    fi
}

test_qdrant_health() {
    qdrant_debug "Testing health..."
    
    # Test connection
    test_qdrant_connection || return 1
    
    # Test basic collection operations
    local test_collection="health_test_collection"
    
    # Create collection
    curl -s -X PUT "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/collections/$test_collection" \
        -d '{"vectors":{"size":128,"distance":"Cosine"}}' >/dev/null 2>&1 || return 1
    
    # List collections
    local result
    result=$(curl -s "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/collections" 2>&1)
    [[ "$result" =~ "$test_collection" ]] || return 1
    
    # Get collection info
    curl -s "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/collections/$test_collection" >/dev/null 2>&1 || return 1
    
    # Delete collection
    curl -s -X DELETE "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/collections/$test_collection" >/dev/null 2>&1 || return 1
    
    qdrant_debug "Health test passed"
    return 0
}

test_qdrant_basic() {
    qdrant_debug "Testing basic operations..."
    
    # Test version endpoint
    local version_result
    version_result=$(curl -s "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/" 2>&1)
    [[ "$version_result" =~ "version" ]] || return 1
    
    # Test cluster endpoint
    local cluster_result
    cluster_result=$(curl -s "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/cluster" 2>&1)
    [[ "$cluster_result" =~ "peer_id" ]] || return 1
    
    # Test collections list
    curl -s "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/collections" >/dev/null 2>&1 || return 1
    
    # Test collection lifecycle
    local test_collection="basic_test_collection"
    
    # Create
    curl -s -X PUT "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/collections/$test_collection" \
        -d '{"vectors":{"size":256,"distance":"Dot"}}' >/dev/null 2>&1 || return 1
        
    # Verify creation
    local collections_result
    collections_result=$(curl -s "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/collections" 2>&1)
    [[ "$collections_result" =~ "$test_collection" ]] || return 1
    
    # Get info
    local info_result
    info_result=$(curl -s "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/collections/$test_collection" 2>&1)
    [[ "$info_result" =~ "256" && "$info_result" =~ "Dot" ]] || return 1
    
    # Delete
    curl -s -X DELETE "http://${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]}/collections/$test_collection" >/dev/null 2>&1 || return 1
    
    qdrant_debug "Basic test passed"
    return 0
}

# === State Management ===
qdrant_mock_reset() {
    qdrant_debug "Resetting mock state (called from: ${BASH_SOURCE[1]:-unknown}:${BASH_LINENO[0]:-unknown})"
    
    QDRANT_COLLECTIONS=()
    QDRANT_ERROR_MODE=""
    QDRANT_CONFIG[health_status]="healthy"
    QDRANT_CONFIG[server_running]="true"
    
    # Create default collections for testing
    QDRANT_COLLECTIONS["agent_memory"]="1536|Cosine|0|0"
    QDRANT_COLLECTIONS["code_embeddings"]="768|Dot|0|0"
}

qdrant_mock_set_error() {
    QDRANT_ERROR_MODE="$1"
    qdrant_debug "Set error mode: $1"
}

qdrant_mock_set_health() {
    QDRANT_CONFIG[health_status]="$1"
    qdrant_debug "Set health status: $1"
}

qdrant_mock_set_server_status() {
    QDRANT_CONFIG[server_running]="$1"
    qdrant_debug "Set server running: $1"
}

qdrant_mock_dump_state() {
    echo "=== Qdrant Mock State ==="
    echo "Server: ${QDRANT_CONFIG[host]}:${QDRANT_CONFIG[rest_port]} (running: ${QDRANT_CONFIG[server_running]})"
    echo "Version: ${QDRANT_CONFIG[version]}"
    echo "Collections: ${#QDRANT_COLLECTIONS[@]}"
    for collection in "${!QDRANT_COLLECTIONS[@]}"; do
        echo "  $collection: ${QDRANT_COLLECTIONS[$collection]}"
    done
    echo "Error Mode: ${QDRANT_ERROR_MODE:-none}"
    echo "======================="
}

qdrant_mock_create_collection() {
    local name="${1:-test_collection}"
    local size="${2:-1536}"
    local distance="${3:-Cosine}"
    
    QDRANT_COLLECTIONS[$name]="$size|$distance|0|0"
    qdrant_debug "Added collection: $name ($size, $distance)"
    echo "$name"
}

# === Export Functions ===
export -f curl
export -f test_qdrant_connection
export -f test_qdrant_health
export -f test_qdrant_basic
export -f qdrant_mock_reset
export -f qdrant_mock_set_error
export -f qdrant_mock_set_health
export -f qdrant_mock_set_server_status
export -f qdrant_mock_dump_state
export -f qdrant_mock_create_collection
export -f qdrant_debug
export -f qdrant_check_error
export -f qdrant_handle_api_request

# Initialize with default state
qdrant_mock_reset
qdrant_debug "Qdrant Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
