#!/usr/bin/env bash
# Qdrant Mock Implementation
# 
# Provides comprehensive mock for Qdrant vector database operations including:
# - curl API endpoint interception (REST and telemetry)
# - Docker container state simulation
# - Collection management (create, delete, list, info)
# - Snapshot operations (create, restore, list)
# - Health check and metrics endpoints
# - API key authentication support
# - Cluster information simulation
# - Container lifecycle management
#
# This mock follows the same standards as other updated mocks with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${QDRANT_MOCK_LOADED:-}" ]] && return 0
declare -g QDRANT_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"

# Global configuration
declare -g QDRANT_MOCK_STATE_DIR="${QDRANT_MOCK_STATE_DIR:-/tmp/qdrant-mock-state}"
declare -g QDRANT_MOCK_DEBUG="${QDRANT_MOCK_DEBUG:-}"

# Global state arrays
declare -gA QDRANT_MOCK_CONFIG=(           # Qdrant configuration
    [container_name]="qdrant"
    [rest_port]="6333"
    [grpc_port]="6334"
    [base_url]="http://localhost:6333"
    [image]="qdrant/qdrant:latest"
    [api_key]=""
    [health_status]="healthy"
    [server_status]="running"
    [error_mode]=""
    [network_name]="qdrant-network"
    [data_dir]="/tmp/qdrant-data"
    [snapshots_dir]="/tmp/qdrant-snapshots"
    [version]="v1.11.0"
)

declare -gA QDRANT_MOCK_CONTAINERS=()      # Container state tracking
declare -gA QDRANT_MOCK_COLLECTIONS=()     # Collection definitions: name -> "size|distance|vectors_count|points_count"
declare -gA QDRANT_MOCK_SNAPSHOTS=()       # Snapshot tracking: name -> "timestamp|collections"
declare -gA QDRANT_MOCK_API_RESPONSES=()   # Custom API responses
declare -gA QDRANT_MOCK_NETWORK_STATE=()   # Network existence tracking
declare -gA QDRANT_MOCK_CLUSTER_INFO=()    # Cluster peer information
declare -gA QDRANT_MOCK_ERROR_INJECTION=() # Error injection for testing
declare -gA QDRANT_MOCK_REQUEST_LOG=()     # Log of API requests made
declare -gA QDRANT_MOCK_METRICS=()         # Service metrics data

# Initialize state directory
mkdir -p "$QDRANT_MOCK_STATE_DIR"

# Default cluster info
QDRANT_MOCK_CLUSTER_INFO=(
    [peer_id]="00000000-0000-0000-0000-000000000000"
    [peers_count]="1"
    [raft_info]="leader"
    [consensus]="joint"
    [status]="enabled"
)

# Default metrics
QDRANT_MOCK_METRICS=(
    [requests_total]="1234"
    [requests_avg_duration_seconds]="0.002"
    [requests_max_duration_seconds]="0.150"
    [grpc_requests_total]="567"
    [grpc_requests_avg_duration_seconds]="0.001"
    [rest_requests_total]="667"
    [rest_requests_avg_duration_seconds]="0.003"
)

# State persistence functions
mock::qdrant::save_state() {
    local state_file="$QDRANT_MOCK_STATE_DIR/qdrant-state.sh"
    {
        echo "# Qdrant mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p QDRANT_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_CONFIG=()"
        declare -p QDRANT_MOCK_CONTAINERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_CONTAINERS=()"
        declare -p QDRANT_MOCK_COLLECTIONS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_COLLECTIONS=()"
        declare -p QDRANT_MOCK_SNAPSHOTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_SNAPSHOTS=()"
        declare -p QDRANT_MOCK_API_RESPONSES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_API_RESPONSES=()"
        declare -p QDRANT_MOCK_NETWORK_STATE 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_NETWORK_STATE=()"
        declare -p QDRANT_MOCK_CLUSTER_INFO 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_CLUSTER_INFO=()"
        declare -p QDRANT_MOCK_ERROR_INJECTION 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_ERROR_INJECTION=()"
        declare -p QDRANT_MOCK_REQUEST_LOG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_REQUEST_LOG=()"
        declare -p QDRANT_MOCK_METRICS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QDRANT_MOCK_METRICS=()"
    } > "$state_file"
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant" "Saved Qdrant state to $state_file"
    fi
}

mock::qdrant::load_state() {
    local state_file="$QDRANT_MOCK_STATE_DIR/qdrant-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        
        # Use centralized logging if available
        if declare -f mock::log_state >/dev/null 2>&1; then
            mock::log_state "qdrant" "Loaded Qdrant state from $state_file"
        fi
    fi
}

# Automatically load state when sourced
mock::qdrant::load_state

# ----------------------------
# Core Mock Functions
# ----------------------------

# Reset all mock state
mock::qdrant::reset() {
    declare -gA QDRANT_MOCK_CONFIG=(
        [container_name]="qdrant"
        [rest_port]="6333"
        [grpc_port]="6334"
        [base_url]="http://localhost:6333"
        [image]="qdrant/qdrant:latest"
        [api_key]=""
        [health_status]="healthy"
        [server_status]="running"
        [error_mode]=""
        [network_name]="qdrant-network"
        [data_dir]="/tmp/qdrant-data"
        [snapshots_dir]="/tmp/qdrant-snapshots"
        [version]="v1.11.0"
    )
    
    declare -gA QDRANT_MOCK_CONTAINERS=()
    declare -gA QDRANT_MOCK_COLLECTIONS=()
    declare -gA QDRANT_MOCK_SNAPSHOTS=()
    declare -gA QDRANT_MOCK_API_RESPONSES=()
    declare -gA QDRANT_MOCK_NETWORK_STATE=()
    declare -gA QDRANT_MOCK_ERROR_INJECTION=()
    declare -gA QDRANT_MOCK_REQUEST_LOG=()
    
    declare -gA QDRANT_MOCK_CLUSTER_INFO=(
        [peer_id]="00000000-0000-0000-0000-000000000000"
        [peers_count]="1"
        [raft_info]="leader"
        [consensus]="joint"
        [status]="enabled"
    )
    
    declare -gA QDRANT_MOCK_METRICS=(
        [requests_total]="1234"
        [requests_avg_duration_seconds]="0.002"
        [requests_max_duration_seconds]="0.150"
        [grpc_requests_total]="567"
        [grpc_requests_avg_duration_seconds]="0.001"
        [rest_requests_total]="667"
        [rest_requests_avg_duration_seconds]="0.003"
    )
    
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant" "Reset all Qdrant mock state"
    fi
}

# Set container state
mock::qdrant::set_container_state() {
    local name="$1"
    local state="$2"
    local image="${3:-${QDRANT_MOCK_CONFIG[image]}}"
    
    # Handle empty names by using a placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    QDRANT_MOCK_CONTAINERS["$name"]="$state|$image"
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_container_state" "$name" "$state"
    fi
}

# Create collection
mock::qdrant::create_collection() {
    local name="$1"
    local size="${2:-1536}"
    local distance="${3:-Cosine}"
    local vectors_count="${4:-0}"
    local points_count="${5:-0}"
    
    # Handle empty names by using a placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    QDRANT_MOCK_COLLECTIONS["$name"]="$size|$distance|$vectors_count|$points_count"
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_collection" "$name" "created"
    fi
}

# Delete collection
mock::qdrant::delete_collection() {
    local name="$1"
    
    # Handle empty names by using a placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    unset QDRANT_MOCK_COLLECTIONS["$name"]
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_collection" "$name" "deleted"
    fi
}

# Set API response
mock::qdrant::set_api_response() {
    local endpoint="$1"
    local response="$2"
    
    QDRANT_MOCK_API_RESPONSES["$endpoint"]="$response"
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_api_response" "$endpoint" "response_set"
    fi
}

# Set API key
mock::qdrant::set_api_key() {
    local api_key="$1"
    QDRANT_MOCK_CONFIG[api_key]="$api_key"
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_api_key" "set" "${api_key:0:10}..."
    fi
}

# Inject error for specific endpoint
mock::qdrant::inject_error() {
    local endpoint="$1"
    local error_type="${2:-generic}"
    
    QDRANT_MOCK_ERROR_INJECTION["$endpoint"]="$error_type"
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_error_injection" "$endpoint" "$error_type"
    fi
}

# Set health status
mock::qdrant::set_health_status() {
    local status="${1:-healthy}"
    QDRANT_MOCK_CONFIG[health_status]="$status"
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_health" "set" "$status"
    fi
}

# Set server status
mock::qdrant::set_server_status() {
    local status="${1:-running}"
    QDRANT_MOCK_CONFIG[server_status]="$status"
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_server" "set" "$status"
    fi
}

# Create snapshot
mock::qdrant::create_snapshot() {
    local name="$1"
    local collections="${2:-all}"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    QDRANT_MOCK_SNAPSHOTS["$name"]="$timestamp|$collections"
    mock::qdrant::save_state
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_snapshot" "$name" "created"
    fi
}

# ----------------------------
# curl Interceptor
# ----------------------------

curl() {
    # Use centralized logging if available
    if declare -f mock::log_and_verify >/dev/null 2>&1; then
        mock::log_and_verify "curl" "$@"
    fi
    
    # Always reload state at the beginning to handle BATS subshells
    mock::qdrant::load_state
    
    # Parse curl arguments
    local url="" method="GET" output_file="" data="" headers=() max_time=""
    local write_out="" silent=false show_error=false fail_on_error=false
    local args=("$@")
    local i=0
    
    while [[ $i -lt ${#args[@]} ]]; do
        case "${args[$i]}" in
            -X)
                method="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            --output|-o)
                output_file="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            -d|--data)
                data="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            -H|--header)
                headers+=("${args[$((i+1))]}")
                i=$((i + 2))
                ;;
            --max-time)
                max_time="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            --write-out|-w)
                write_out="${args[$((i+1))]}"
                i=$((i + 2))
                ;;
            -s|--silent)
                silent=true
                i=$((i + 1))
                ;;
            -S|--show-error)
                show_error=true
                i=$((i + 1))
                ;;
            -f|--fail)
                fail_on_error=true
                i=$((i + 1))
                ;;
            --connect-timeout)
                # Skip connect-timeout
                i=$((i + 2))
                ;;
            http*)
                url="${args[$i]}"
                i=$((i + 1))
                ;;
            *)
                i=$((i + 1))
                ;;
        esac
    done
    
    # Check if this is a Qdrant API request (port 6333 or 6334, or /collections, /cluster, etc.)
    if [[ "$url" == *":6333"* || "$url" == *":6334"* || "$url" == *"/collections"* || "$url" == *"/cluster"* || "$url" == *"/telemetry"* || "$url" == *"/metrics"* ]]; then
        mock::qdrant::handle_api_request "$method" "$url" "$data" "$output_file" "$write_out" "$fail_on_error" "${headers[@]}"
        return $?
    fi
    
    # For non-Qdrant requests, simulate a generic successful response
    if [[ -n "$output_file" ]]; then
        echo "Mock curl response" > "$output_file"
    else
        echo "Mock curl response"
    fi
    
    [[ -n "$write_out" ]] && echo "200"
    mock::qdrant::save_state
    return 0
}

# Handle Qdrant API requests
mock::qdrant::handle_api_request() {
    local method="$1"
    local url="$2"
    local data="$3"
    local output_file="$4"
    local write_out="$5"
    local fail_on_error="$6"
    shift 6
    local headers=("$@")
    
    # Log the request with microsecond precision to avoid collisions
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S.%N' | head -c 23)
    QDRANT_MOCK_REQUEST_LOG["$timestamp"]="$method $url"
    
    # Check for API key authentication if configured
    if [[ -n "${QDRANT_MOCK_CONFIG[api_key]}" ]]; then
        local auth_provided=false
        for header in "${headers[@]}"; do
            if [[ "$header" == "api-key: ${QDRANT_MOCK_CONFIG[api_key]}" ]]; then
                auth_provided=true
                break
            fi
        done
        
        if [[ "$auth_provided" != "true" ]]; then
            [[ "$fail_on_error" == "true" ]] && return 22  # HTTP error
            [[ -n "$write_out" ]] && echo "401"
            echo '{"status":{"error":"Unauthorized"},"time":0.0}' > "${output_file:-/dev/stdout}"
            return 22
        fi
    fi
    
    # Extract endpoint from URL
    local endpoint=""
    if [[ "$url" == *"/collections/"* ]]; then
        # Collection-specific endpoints
        local collection_name="${url#*/collections/}"
        collection_name="${collection_name%%/*}"
        
        if [[ "$url" == *"/cluster" ]]; then
            endpoint="collection_cluster:$collection_name"
        else
            endpoint="collection:$collection_name"
        fi
    elif [[ "$url" == *"/collections" ]]; then
        endpoint="collections"
    elif [[ "$url" == *"/cluster" ]]; then
        endpoint="cluster"
    elif [[ "$url" == *"/telemetry" ]]; then
        endpoint="telemetry"
    elif [[ "$url" == *"/metrics" ]]; then
        endpoint="metrics"
    elif [[ "$url" == *":6333/" ]] || [[ "$url" == *":6333" ]]; then
        endpoint="health"
    else
        endpoint="unknown"
    fi
    
    # Check for injected errors
    if [[ -n "${QDRANT_MOCK_ERROR_INJECTION[$endpoint]}" ]]; then
        mock::qdrant::simulate_error "$endpoint" "${QDRANT_MOCK_ERROR_INJECTION[$endpoint]}" "$output_file" "$write_out"
        return $?
    fi
    
    # Check server status
    if [[ "${QDRANT_MOCK_CONFIG[server_status]}" != "running" ]]; then
        [[ "$fail_on_error" == "true" ]] && return 7  # Connection failed
        [[ -n "$write_out" ]] && echo "000"
        echo "curl: (7) Failed to connect to localhost port ${QDRANT_MOCK_CONFIG[rest_port]}: Connection refused" >&2
        return 7
    fi
    
    # Check health status
    if [[ "${QDRANT_MOCK_CONFIG[health_status]}" != "healthy" && "$endpoint" != "health" ]]; then
        [[ "$fail_on_error" == "true" ]] && return 22  # HTTP error
        [[ -n "$write_out" ]] && echo "503"
        echo "Service Unavailable" >&2
        return 22
    fi
    
    # Generate response based on endpoint and method
    case "$endpoint" in
        "health")
            mock::qdrant::generate_health_response "$output_file"
            ;;
        "cluster")
            mock::qdrant::generate_cluster_response "$output_file"
            ;;
        "telemetry")
            mock::qdrant::generate_telemetry_response "$output_file"
            ;;
        "metrics")
            mock::qdrant::generate_metrics_response "$output_file"
            ;;
        "collections")
            if [[ "$method" == "GET" ]]; then
                mock::qdrant::generate_collections_list_response "$output_file"
            else
                echo '{"status":"ok","time":0.0}' > "${output_file:-/dev/stdout}"
            fi
            ;;
        collection:*)
            local collection_name="${endpoint#collection:}"
            if [[ "$method" == "GET" ]]; then
                mock::qdrant::generate_collection_info_response "$collection_name" "$output_file"
            elif [[ "$method" == "PUT" ]]; then
                mock::qdrant::handle_create_collection "$collection_name" "$data" "$output_file"
            elif [[ "$method" == "DELETE" ]]; then
                mock::qdrant::handle_delete_collection "$collection_name" "$output_file"
            else
                echo '{"status":"ok","time":0.0}' > "${output_file:-/dev/stdout}"
            fi
            ;;
        collection_cluster:*)
            local collection_name="${endpoint#collection_cluster:}"
            mock::qdrant::generate_collection_cluster_response "$collection_name" "$output_file"
            ;;
        *)
            echo "Unknown endpoint" >&2
            [[ -n "$write_out" ]] && echo "404"
            return 22
            ;;
    esac
    
    [[ -n "$write_out" ]] && echo "200"
    mock::qdrant::save_state
    return 0
}

# Simulate various error conditions
mock::qdrant::simulate_error() {
    local endpoint="$1"
    local error_type="$2"
    local output_file="$3"
    local write_out="$4"
    
    case "$error_type" in
        "connection_timeout")
            echo "curl: (28) Operation timed out after 10000 milliseconds" >&2
            [[ -n "$write_out" ]] && echo "000"
            return 28
            ;;
        "connection_refused")
            echo "curl: (7) Failed to connect to localhost port ${QDRANT_MOCK_CONFIG[rest_port]}: Connection refused" >&2
            [[ -n "$write_out" ]] && echo "000"
            return 7
            ;;
        "http_500")
            echo '{"status":{"error":"Internal Server Error"},"time":0.0}' > "${output_file:-/dev/stdout}"
            [[ -n "$write_out" ]] && echo "500"
            return 22
            ;;
        "http_401")
            echo '{"status":{"error":"Unauthorized"},"time":0.0}' > "${output_file:-/dev/stdout}"
            [[ -n "$write_out" ]] && echo "401"
            return 22
            ;;
        "http_400")
            echo '{"status":{"error":"Bad Request"},"time":0.0}' > "${output_file:-/dev/stdout}"
            [[ -n "$write_out" ]] && echo "400"
            return 22
            ;;
        *)
            echo "Generic error occurred" >&2
            [[ -n "$write_out" ]] && echo "500"
            return 1
            ;;
    esac
}

# Generate health/version response
mock::qdrant::generate_health_response() {
    local output_file="$1"
    
    # Check for custom response
    if [[ -n "${QDRANT_MOCK_API_RESPONSES[health]}" ]]; then
        echo "${QDRANT_MOCK_API_RESPONSES[health]}" > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    local response=$(cat <<EOF
{
  "title": "qdrant - vector search engine",
  "version": "${QDRANT_MOCK_CONFIG[version]}",
  "commit": "1234567890abcdef"
}
EOF
)
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# Generate cluster response
mock::qdrant::generate_cluster_response() {
    local output_file="$1"
    
    # Check for custom response
    if [[ -n "${QDRANT_MOCK_API_RESPONSES[cluster]}" ]]; then
        echo "${QDRANT_MOCK_API_RESPONSES[cluster]}" > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    local response=$(cat <<EOF
{
  "result": {
    "status": "${QDRANT_MOCK_CLUSTER_INFO[status]}",
    "peer_id": "${QDRANT_MOCK_CLUSTER_INFO[peer_id]}",
    "peers": {
      "${QDRANT_MOCK_CLUSTER_INFO[peer_id]}": {
        "uri": "http://127.0.0.1:6335"
      }
    },
    "raft_info": {
      "term": 1,
      "commit": 0,
      "pending_operations": 0,
      "leader": "${QDRANT_MOCK_CLUSTER_INFO[peer_id]}",
      "role": "${QDRANT_MOCK_CLUSTER_INFO[raft_info]}",
      "is_voter": true
    },
    "consensus_thread_status": {
      "consensus_thread_status": "${QDRANT_MOCK_CLUSTER_INFO[consensus]}",
      "last_update": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)"
    }
  },
  "status": "ok",
  "time": 0.000012
}
EOF
)
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# Generate telemetry response
mock::qdrant::generate_telemetry_response() {
    local output_file="$1"
    
    # Check for custom response
    if [[ -n "${QDRANT_MOCK_API_RESPONSES[telemetry]}" ]]; then
        echo "${QDRANT_MOCK_API_RESPONSES[telemetry]}" > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    local response=$(cat <<EOF
{
  "result": {
    "id": "${QDRANT_MOCK_CLUSTER_INFO[peer_id]}",
    "app": {
      "name": "qdrant",
      "version": "${QDRANT_MOCK_CONFIG[version]}",
      "commit": "1234567890abcdef"
    },
    "collections": {
      "count": ${#QDRANT_MOCK_COLLECTIONS[@]},
      "vectors": 100000,
      "indexed_vectors": 95000,
      "points": 100000,
      "segments": 10
    },
    "cluster": {
      "enabled": true,
      "nodes_count": ${QDRANT_MOCK_CLUSTER_INFO[peers_count]}
    },
    "requests": {
      "rest": {
        "total": ${QDRANT_MOCK_METRICS[rest_requests_total]},
        "avg_duration_seconds": ${QDRANT_MOCK_METRICS[rest_requests_avg_duration_seconds]}
      },
      "grpc": {
        "total": ${QDRANT_MOCK_METRICS[grpc_requests_total]},
        "avg_duration_seconds": ${QDRANT_MOCK_METRICS[grpc_requests_avg_duration_seconds]}
      }
    }
  },
  "status": "ok",
  "time": 0.000015
}
EOF
)
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# Generate metrics response (Prometheus format)
mock::qdrant::generate_metrics_response() {
    local output_file="$1"
    
    # Check for custom response
    if [[ -n "${QDRANT_MOCK_API_RESPONSES[metrics]}" ]]; then
        echo "${QDRANT_MOCK_API_RESPONSES[metrics]}" > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    local response=$(cat <<EOF
# HELP qdrant_app_info Information about Qdrant server
# TYPE qdrant_app_info counter
qdrant_app_info{version="${QDRANT_MOCK_CONFIG[version]}"} 1

# HELP qdrant_rest_requests_total Total number of REST requests
# TYPE qdrant_rest_requests_total counter
qdrant_rest_requests_total ${QDRANT_MOCK_METRICS[rest_requests_total]}

# HELP qdrant_rest_requests_duration_seconds REST request duration
# TYPE qdrant_rest_requests_duration_seconds histogram
qdrant_rest_requests_duration_seconds_sum ${QDRANT_MOCK_METRICS[rest_requests_avg_duration_seconds]}
qdrant_rest_requests_duration_seconds_count ${QDRANT_MOCK_METRICS[rest_requests_total]}

# HELP qdrant_grpc_requests_total Total number of gRPC requests
# TYPE qdrant_grpc_requests_total counter
qdrant_grpc_requests_total ${QDRANT_MOCK_METRICS[grpc_requests_total]}

# HELP qdrant_grpc_requests_duration_seconds gRPC request duration
# TYPE qdrant_grpc_requests_duration_seconds histogram
qdrant_grpc_requests_duration_seconds_sum ${QDRANT_MOCK_METRICS[grpc_requests_avg_duration_seconds]}
qdrant_grpc_requests_duration_seconds_count ${QDRANT_MOCK_METRICS[grpc_requests_total]}

# HELP qdrant_collections_total Total number of collections
# TYPE qdrant_collections_total gauge
qdrant_collections_total ${#QDRANT_MOCK_COLLECTIONS[@]}
EOF
)
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# Generate collections list response
mock::qdrant::generate_collections_list_response() {
    local output_file="$1"
    
    # Check for custom response
    if [[ -n "${QDRANT_MOCK_API_RESPONSES[collections]}" ]]; then
        echo "${QDRANT_MOCK_API_RESPONSES[collections]}" > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    # Build collections array
    local collections_json="["
    local first=true
    for collection_name in "${!QDRANT_MOCK_COLLECTIONS[@]}"; do
        if [[ "$first" != "true" ]]; then
            collections_json+=","
        fi
        collections_json+="{\"name\":\"$collection_name\"}"
        first=false
    done
    collections_json+="]"
    
    local response=$(cat <<EOF
{
  "result": {
    "collections": $collections_json
  },
  "status": "ok",
  "time": 0.000025
}
EOF
)
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# Generate collection info response
mock::qdrant::generate_collection_info_response() {
    local collection_name="$1"
    local output_file="$2"
    
    # Check if collection exists
    if [[ -z "${QDRANT_MOCK_COLLECTIONS[$collection_name]}" ]]; then
        echo '{"status":{"error":"Collection not found"},"time":0.0}' > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    # Parse collection data
    IFS='|' read -r size distance vectors_count points_count <<< "${QDRANT_MOCK_COLLECTIONS[$collection_name]}"
    
    # Set defaults if empty
    vectors_count="${vectors_count:-0}"
    points_count="${points_count:-0}"
    
    local response=$(cat <<EOF
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
        "replication_factor": 1,
        "write_consistency_factor": 1,
        "on_disk_payload": true
      },
      "hnsw_config": {
        "m": 16,
        "ef_construct": 100,
        "full_scan_threshold": 10000,
        "max_indexing_threads": 0,
        "on_disk": false
      },
      "optimizer_config": {
        "deleted_threshold": 0.2,
        "vacuum_min_vector_number": 1000,
        "default_segment_number": 0,
        "max_segment_size": null,
        "memmap_threshold": 100000,
        "indexing_threshold": 20000,
        "flush_interval_sec": 5,
        "max_optimization_threads": null
      },
      "wal_config": {
        "wal_capacity_mb": 32,
        "wal_segments_ahead": 0
      }
    },
    "payload_schema": {}
  },
  "status": "ok",
  "time": 0.000035
}
EOF
)
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# Generate collection cluster response
mock::qdrant::generate_collection_cluster_response() {
    local collection_name="$1"
    local output_file="$2"
    
    # Check if collection exists
    if [[ -z "${QDRANT_MOCK_COLLECTIONS[$collection_name]}" ]]; then
        echo '{"status":{"error":"Collection not found"},"time":0.0}' > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    local response=$(cat <<EOF
{
  "result": {
    "peer_id": "${QDRANT_MOCK_CLUSTER_INFO[peer_id]}",
    "shard_count": 1,
    "local_shards": [
      {
        "shard_id": 0,
        "points_count": 0,
        "state": "Active"
      }
    ],
    "remote_shards": [],
    "shard_transfers": []
  },
  "status": "ok",
  "time": 0.000018
}
EOF
)
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    else
        echo "$response"
    fi
}

# Handle create collection request
mock::qdrant::handle_create_collection() {
    local collection_name="$1"
    local data="$2"
    local output_file="$3"
    
    # Check if collection already exists
    if [[ -n "${QDRANT_MOCK_COLLECTIONS[$collection_name]}" ]]; then
        echo '{"status":{"error":"Collection already exists"},"time":0.0}' > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    # Parse vector size and distance from data if available
    local size="1536"
    local distance="Cosine"
    
    # Try to parse JSON data if available
    if [[ -n "$data" ]]; then
        # Try using jq if available
        if command -v jq &>/dev/null; then
            size=$(echo "$data" | jq -r '.vectors.size // 1536' 2>/dev/null || echo "1536")
            distance=$(echo "$data" | jq -r '.vectors.distance // "Cosine"' 2>/dev/null || echo "Cosine")
        else
            # Fallback to simple grep/sed parsing
            if [[ "$data" =~ \"size\"[[:space:]]*:[[:space:]]*([0-9]+) ]]; then
                size="${BASH_REMATCH[1]}"
            fi
            if [[ "$data" =~ \"distance\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
                distance="${BASH_REMATCH[1]}"
            fi
        fi
    fi
    
    # Create the collection
    mock::qdrant::create_collection "$collection_name" "$size" "$distance" "0" "0"
    
    echo '{"result":true,"status":"ok","time":0.012345}' > "${output_file:-/dev/stdout}"
}

# Handle delete collection request
mock::qdrant::handle_delete_collection() {
    local collection_name="$1"
    local output_file="$2"
    
    # Check if collection exists
    if [[ -z "${QDRANT_MOCK_COLLECTIONS[$collection_name]}" ]]; then
        echo '{"status":{"error":"Collection not found"},"time":0.0}' > "${output_file:-/dev/stdout}"
        return 0
    fi
    
    # Delete the collection
    mock::qdrant::delete_collection "$collection_name"
    
    echo '{"result":true,"status":"ok","time":0.002345}' > "${output_file:-/dev/stdout}"
}

# ----------------------------
# Test Helper Functions
# ----------------------------

# Create a running Qdrant scenario
mock::qdrant::scenario::create_running_service() {
    local container_name="${1:-${QDRANT_MOCK_CONFIG[container_name]}}"
    
    mock::qdrant::set_container_state "$container_name" "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::set_server_status "running"
    
    # Create default collections
    mock::qdrant::create_collection "agent_memory" "1536" "Cosine" "1000" "1000"
    mock::qdrant::create_collection "code_embeddings" "768" "Dot" "500" "500"
    mock::qdrant::create_collection "document_chunks" "1536" "Cosine" "2000" "2000"
    
    QDRANT_MOCK_NETWORK_STATE["${QDRANT_MOCK_CONFIG[network_name]}"]="created"
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_scenario" "created_running_service" "$container_name"
    fi
}

# Create a stopped Qdrant scenario
mock::qdrant::scenario::create_stopped_service() {
    local container_name="${1:-${QDRANT_MOCK_CONFIG[container_name]}}"
    
    mock::qdrant::set_container_state "$container_name" "stopped"
    mock::qdrant::set_health_status "unhealthy"
    mock::qdrant::set_server_status "stopped"
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_scenario" "created_stopped_service" "$container_name"
    fi
}

# Create an authenticated service scenario
mock::qdrant::scenario::create_authenticated_service() {
    local api_key="${1:-secure-test-api-key}"
    
    mock::qdrant::scenario::create_running_service
    mock::qdrant::set_api_key "$api_key"
    
    # Use centralized logging if available
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "qdrant_scenario" "created_authenticated_service" "with_api_key"
    fi
}

# ----------------------------
# Assertion Helper Functions
# ----------------------------

mock::qdrant::assert::container_running() {
    local name="${1:-${QDRANT_MOCK_CONFIG[container_name]}}"
    
    # Handle empty names by using the placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    local state="${QDRANT_MOCK_CONTAINERS[$name]%%|*}"
    
    if [[ "$state" != "running" ]]; then
        echo "ASSERTION FAILED: Qdrant container '$name' is not running (state: ${state:-not found})" >&2
        return 1
    fi
    return 0
}

mock::qdrant::assert::container_stopped() {
    local name="${1:-${QDRANT_MOCK_CONFIG[container_name]}}"
    
    # Handle empty names by using the placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    local state="${QDRANT_MOCK_CONTAINERS[$name]%%|*}"
    
    if [[ "$state" != "stopped" ]]; then
        echo "ASSERTION FAILED: Qdrant container '$name' is not stopped (state: ${state:-not found})" >&2
        return 1
    fi
    return 0
}

mock::qdrant::assert::healthy() {
    if [[ "${QDRANT_MOCK_CONFIG[health_status]}" != "healthy" ]]; then
        echo "ASSERTION FAILED: Qdrant is not healthy (status: ${QDRANT_MOCK_CONFIG[health_status]})" >&2
        return 1
    fi
    return 0
}

mock::qdrant::assert::collection_exists() {
    local name="$1"
    
    if [[ -z "${QDRANT_MOCK_COLLECTIONS[$name]}" ]]; then
        echo "ASSERTION FAILED: Collection '$name' does not exist" >&2
        return 1
    fi
    return 0
}

mock::qdrant::assert::collection_not_exists() {
    local name="$1"
    
    if [[ -n "${QDRANT_MOCK_COLLECTIONS[$name]}" ]]; then
        echo "ASSERTION FAILED: Collection '$name' should not exist" >&2
        return 1
    fi
    return 0
}

mock::qdrant::assert::api_called() {
    local endpoint="$1"
    local found=false
    
    # Load state to ensure we have the latest request log in subshells
    mock::qdrant::load_state
    
    for timestamp in "${!QDRANT_MOCK_REQUEST_LOG[@]}"; do
        if [[ "${QDRANT_MOCK_REQUEST_LOG[$timestamp]}" == *"$endpoint"* ]]; then
            found=true
            break
        fi
    done
    
    if [[ "$found" != "true" ]]; then
        echo "ASSERTION FAILED: API endpoint '$endpoint' was not called" >&2
        return 1
    fi
    return 0
}

# Get helper functions
mock::qdrant::get::container_state() {
    local name="${1:-${QDRANT_MOCK_CONFIG[container_name]}}"
    
    # Handle empty names by using the placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    echo "${QDRANT_MOCK_CONTAINERS[$name]%%|*}"
}

mock::qdrant::get::collection_info() {
    local name="$1"
    
    # Handle empty names by using a placeholder
    [[ -z "$name" ]] && name="__empty__"
    
    echo "${QDRANT_MOCK_COLLECTIONS[$name]:-}"
}

mock::qdrant::get::collection_count() {
    echo "${#QDRANT_MOCK_COLLECTIONS[@]}"
}

mock::qdrant::get::config() {
    local key="$1"
    echo "${QDRANT_MOCK_CONFIG[$key]:-}"
}

# Debug helper
mock::qdrant::debug::dump_state() {
    echo "=== Qdrant Mock State Dump ==="
    echo "Configuration:"
    for key in "${!QDRANT_MOCK_CONFIG[@]}"; do
        echo "  $key: ${QDRANT_MOCK_CONFIG[$key]}"
    done
    
    echo "Containers:"
    for name in "${!QDRANT_MOCK_CONTAINERS[@]}"; do
        echo "  $name: ${QDRANT_MOCK_CONTAINERS[$name]}"
    done
    
    echo "Collections:"
    for name in "${!QDRANT_MOCK_COLLECTIONS[@]}"; do
        echo "  $name: ${QDRANT_MOCK_COLLECTIONS[$name]}"
    done
    
    echo "Snapshots:"
    for name in "${!QDRANT_MOCK_SNAPSHOTS[@]}"; do
        echo "  $name: ${QDRANT_MOCK_SNAPSHOTS[$name]}"
    done
    
    echo "Error Injections:"
    for endpoint in "${!QDRANT_MOCK_ERROR_INJECTION[@]}"; do
        echo "  $endpoint: ${QDRANT_MOCK_ERROR_INJECTION[$endpoint]}"
    done
    
    echo "API Request Log:"
    for timestamp in "${!QDRANT_MOCK_REQUEST_LOG[@]}"; do
        echo "  $timestamp: ${QDRANT_MOCK_REQUEST_LOG[$timestamp]}"
    done
    echo "=========================="
}

# ----------------------------
# Export functions for subshells
# ----------------------------
export -f curl
export -f mock::qdrant::reset
export -f mock::qdrant::set_container_state
export -f mock::qdrant::create_collection
export -f mock::qdrant::delete_collection
export -f mock::qdrant::set_api_response
export -f mock::qdrant::set_api_key
export -f mock::qdrant::inject_error
export -f mock::qdrant::set_health_status
export -f mock::qdrant::set_server_status
export -f mock::qdrant::create_snapshot
export -f mock::qdrant::handle_api_request
export -f mock::qdrant::simulate_error
export -f mock::qdrant::generate_health_response
export -f mock::qdrant::generate_cluster_response
export -f mock::qdrant::generate_telemetry_response
export -f mock::qdrant::generate_metrics_response
export -f mock::qdrant::generate_collections_list_response
export -f mock::qdrant::generate_collection_info_response
export -f mock::qdrant::generate_collection_cluster_response
export -f mock::qdrant::handle_create_collection
export -f mock::qdrant::handle_delete_collection

# Export scenario functions
export -f mock::qdrant::scenario::create_running_service
export -f mock::qdrant::scenario::create_stopped_service
export -f mock::qdrant::scenario::create_authenticated_service

# Export assertion functions
export -f mock::qdrant::assert::container_running
export -f mock::qdrant::assert::container_stopped
export -f mock::qdrant::assert::healthy
export -f mock::qdrant::assert::collection_exists
export -f mock::qdrant::assert::collection_not_exists
export -f mock::qdrant::assert::api_called

# Export getter functions
export -f mock::qdrant::get::container_state
export -f mock::qdrant::get::collection_info
export -f mock::qdrant::get::collection_count
export -f mock::qdrant::get::config

# Export debug functions
export -f mock::qdrant::debug::dump_state

echo "[MOCK] Qdrant mocks loaded successfully"