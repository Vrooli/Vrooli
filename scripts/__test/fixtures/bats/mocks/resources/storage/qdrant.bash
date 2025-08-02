#!/usr/bin/env bash
# Qdrant Resource Mock Implementation
# Provides realistic mock responses for Qdrant vector database service

# Prevent duplicate loading
if [[ "${QDRANT_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export QDRANT_MOCK_LOADED="true"

#######################################
# Setup Qdrant mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::qdrant::setup() {
    local state="${1:-healthy}"
    
    # Configure Qdrant-specific environment
    export QDRANT_PORT="${QDRANT_PORT:-6333}"
    export QDRANT_BASE_URL="http://localhost:${QDRANT_PORT}"
    export QDRANT_CONTAINER_NAME="${TEST_NAMESPACE}_qdrant"
    export QDRANT_API_KEY="${QDRANT_API_KEY:-}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$QDRANT_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::qdrant::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::qdrant::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::qdrant::setup_installing_endpoints
            ;;
        "stopped")
            mock::qdrant::setup_stopped_endpoints
            ;;
        *)
            echo "[QDRANT_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[QDRANT_MOCK] Qdrant mock configured with state: $state"
}

#######################################
# Setup healthy Qdrant endpoints
#######################################
mock::qdrant::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/health" \
        '{"status":"ok","version":"1.7.0"}'
    
    # Collections endpoint
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/collections" \
        '{
            "status": "ok",
            "result": {
                "collections": [
                    {
                        "name": "test_collection",
                        "vectors_count": 1000,
                        "points_count": 1000,
                        "segments_count": 1,
                        "config": {
                            "params": {
                                "vectors": {
                                    "size": 384,
                                    "distance": "Cosine"
                                }
                            }
                        }
                    },
                    {
                        "name": "embeddings",
                        "vectors_count": 50000,
                        "points_count": 50000,
                        "segments_count": 4
                    }
                ]
            }
        }'
    
    # Single collection endpoint
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/collections/test_collection" \
        '{
            "status": "ok",
            "result": {
                "status": "green",
                "optimizer_status": "ok",
                "vectors_count": 1000,
                "indexed_vectors_count": 1000,
                "points_count": 1000,
                "segments_count": 1,
                "config": {
                    "params": {
                        "vectors": {
                            "size": 384,
                            "distance": "Cosine"
                        }
                    },
                    "hnsw_config": {
                        "m": 16,
                        "ef_construct": 200,
                        "full_scan_threshold": 10000
                    }
                }
            }
        }'
    
    # Create collection endpoint
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/collections/new_collection" \
        '{"status":"ok","result":true,"time":0.5}' \
        "PUT"
    
    # Points operations
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/collections/test_collection/points" \
        '{"status":"ok","result":{"operation_id":1,"status":"completed"}}' \
        "PUT"
    
    # Search endpoint
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/collections/test_collection/points/search" \
        '{
            "status": "ok",
            "result": [
                {
                    "id": 1,
                    "version": 0,
                    "score": 0.95,
                    "payload": {
                        "text": "Similar document 1",
                        "metadata": {"category": "test"}
                    }
                },
                {
                    "id": 2,
                    "version": 0,
                    "score": 0.87,
                    "payload": {
                        "text": "Similar document 2",
                        "metadata": {"category": "test"}
                    }
                }
            ],
            "time": 0.002
        }' \
        "POST"
    
    # Telemetry endpoint
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/telemetry" \
        '{
            "status": "ok",
            "result": {
                "collections": 2,
                "searches": 1523,
                "updates": 450,
                "deletes": 12
            }
        }'
}

#######################################
# Setup unhealthy Qdrant endpoints
#######################################
mock::qdrant::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/health" \
        '{"status":"error","error":"Storage backend unavailable"}' \
        "GET" \
        "503"
    
    # Collections endpoint returns error
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/collections" \
        '{"status":"error","error":"Service temporarily unavailable"}' \
        "GET" \
        "503"
}

#######################################
# Setup installing Qdrant endpoints
#######################################
mock::qdrant::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/health" \
        '{"status":"installing","progress":75,"current_step":"Building indexes"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$QDRANT_BASE_URL/collections" \
        '{"status":"error","error":"Qdrant is still initializing"}' \
        "GET" \
        "503"
}

#######################################
# Setup stopped Qdrant endpoints
#######################################
mock::qdrant::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$QDRANT_BASE_URL"
}

#######################################
# Mock Qdrant-specific operations
#######################################

# Mock vector insertion
mock::qdrant::insert_vectors() {
    local collection="$1"
    local vector_count="${2:-100}"
    
    echo '{
        "status": "ok",
        "result": {
            "operation_id": '$(date +%s)',
            "status": "completed"
        },
        "time": 0.05
    }'
}

# Mock collection creation with specific config
mock::qdrant::create_collection() {
    local collection_name="$1"
    local vector_size="${2:-384}"
    local distance="${3:-Cosine}"
    
    echo '{
        "status": "ok",
        "result": true,
        "time": 0.1,
        "collection": {
            "name": "'$collection_name'",
            "config": {
                "params": {
                    "vectors": {
                        "size": '$vector_size',
                        "distance": "'$distance'"
                    }
                }
            }
        }
    }'
}

# Mock batch search
mock::qdrant::batch_search() {
    local collection="$1"
    local batch_size="${2:-10}"
    
    local results=()
    for i in $(seq 1 "$batch_size"); do
        results+=("{\"query_id\":$i,\"results\":[{\"id\":$((i*10)),\"score\":0.$((90-i))}]}")
    done
    
    echo "{\"status\":\"ok\",\"result\":[${results[*]}],\"time\":0.01}"
}

# Mock snapshot operations
mock::qdrant::create_snapshot() {
    local collection="$1"
    
    echo '{
        "status": "ok",
        "result": {
            "name": "'$collection'-snapshot-'$(date +%s)'",
            "size": 1048576,
            "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
        }
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::qdrant::setup
export -f mock::qdrant::setup_healthy_endpoints
export -f mock::qdrant::setup_unhealthy_endpoints
export -f mock::qdrant::setup_installing_endpoints
export -f mock::qdrant::setup_stopped_endpoints
export -f mock::qdrant::insert_vectors
export -f mock::qdrant::create_collection
export -f mock::qdrant::batch_search
export -f mock::qdrant::create_snapshot

echo "[QDRANT_MOCK] Qdrant mock implementation loaded"