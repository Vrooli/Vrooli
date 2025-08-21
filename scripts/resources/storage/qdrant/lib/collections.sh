#!/usr/bin/env bash
# Qdrant Collection Management
# Functions for managing vector collections

set -euo pipefail

# Get directory of this script
QDRANT_COLLECTIONS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "${QDRANT_COLLECTIONS_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"

# Source configuration and messages
# shellcheck disable=SC1091
source "${QDRANT_COLLECTIONS_DIR}/../config/defaults.sh"
# shellcheck disable=SC1091
source "${QDRANT_COLLECTIONS_DIR}/../config/messages.sh"

# Initialize configuration and messages if not already done
if [[ -z "${QDRANT_CONFIG_EXPORTED:-}" ]]; then
    qdrant::export_config 2>/dev/null || true
fi
if [[ -z "${MSG_COLLECTION_CREATED:-}" ]]; then
    qdrant::messages::init 2>/dev/null || true
fi

# Configuration and messages initialized by CLI
# qdrant::export_config and qdrant::messages::init called by main script

#######################################
# Verify initialization before running collection functions
#######################################
qdrant::collections::verify_init() {
    if [[ -z "${QDRANT_CONFIG_EXPORTED:-}" ]]; then
        log::error "Qdrant configuration not initialized. Run qdrant::export_config first."
        return 1
    fi
    if [[ -z "${MSG_COLLECTION_CREATED:-}" ]]; then
        log::error "Qdrant messages not initialized. Run qdrant::messages::init first."
        return 1
    fi
}

#######################################
# Create a new collection
# Arguments:
#   $1 - Collection name
#   $2 - Vector size (default: 1536)
#   $3 - Distance metric (default: Cosine)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::create() {
    qdrant::collections::verify_init || return 1
    
    local collection_name="$1"
    local vector_size="${2:-1536}"
    local distance_metric="${3:-Cosine}"
    
    if [[ -z "$collection_name" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    # Validate distance metric
    case "$distance_metric" in
        Cosine|Dot|Euclid) ;;
        *)
            log::error "Invalid distance metric: $distance_metric (must be Cosine, Dot, or Euclid)"
            return 1
            ;;
    esac
    
    # Check if collection already exists
    if qdrant::collections::exists "$collection_name"; then
        log::warn "${MSG_COLLECTION_EXISTS}: $collection_name"
        return 0
    fi
    
    # Prepare collection configuration
    local config
    config=$(cat << EOF
{
  "vectors": {
    "size": $vector_size,
    "distance": "$distance_metric"
  },
  "optimizers_config": {
    "deleted_threshold": 0.2,
    "vacuum_min_vector_number": 1000,
    "default_segment_number": 0,
    "max_segment_size": null,
    "memmap_threshold": ${QDRANT_STORAGE_MEMMAP_THRESHOLD},
    "indexing_threshold": ${QDRANT_STORAGE_INDEXING_THRESHOLD},
    "flush_interval_sec": 5,
    "max_optimization_threads": null
  },
  "hnsw_config": {
    "m": 16,
    "ef_construct": 100,
    "full_scan_threshold": 10000,
    "max_indexing_threads": 0,
    "on_disk": false
  }
}
EOF
)
    
    # Create the collection
    local response
    response=$(qdrant::api::request "PUT" "/collections/${collection_name}" "$config" 2>/dev/null)
    
    # Check if request was successful (api::request returns 0 for success)
    if [[ $? -eq 0 ]]; then
        # Check if the response indicates success
        local status
        status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
        
        if [[ "$status" == "ok" ]]; then
            log::success "${MSG_COLLECTION_CREATED}: $collection_name (size: $vector_size, distance: $distance_metric)"
            return 0
        else
            local error_msg
            error_msg=$(echo "$response" | jq -r '.status.error // "Unknown error"' 2>/dev/null)
            log::error "${MSG_COLLECTION_CREATE_FAILED}: $error_msg"
            return 1
        fi
    else
        log::error "${MSG_COLLECTION_CREATE_FAILED}"
        return 1
    fi
}

#######################################
# Delete a collection
# Arguments:
#   $1 - Collection name
#   $2 - Force deletion (yes/no, default: no)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::delete() {
    qdrant::collections::verify_init || return 1
    
    local collection_name="$1"
    local force="${2:-no}"
    
    if [[ -z "$collection_name" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    # Check if collection exists
    if ! qdrant::collections::exists "$collection_name"; then
        log::warn "Collection does not exist: $collection_name"
        return 0
    fi
    
    # Check if collection is empty (unless force is specified)
    if [[ "$force" != "yes" ]]; then
        local vector_count
        vector_count=$(qdrant::collections::get_vector_count "$collection_name" 2>/dev/null || echo "unknown")
        
        if [[ "$vector_count" != "0" && "$vector_count" != "unknown" ]]; then
            log::warn "${MSG_COLLECTION_NOT_EMPTY}: $collection_name ($vector_count vectors)"
            log::info "Use --force yes to delete non-empty collection"
            return 1
        fi
    fi
    
    # Delete the collection
    local response
    response=$(qdrant::api::request "DELETE" "/collections/${collection_name}" 2>/dev/null)
    
    # Check if request was successful
    if [[ $? -eq 0 ]]; then
        local status
        status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
        
        if [[ "$status" == "ok" ]]; then
            log::success "Collection deleted: $collection_name"
            return 0
        else
            local error_msg
            error_msg=$(echo "$response" | jq -r '.status.error // "Unknown error"' 2>/dev/null)
            log::error "${MSG_COLLECTION_DELETE_FAILED}: $error_msg"
            return 1
        fi
    else
        log::error "${MSG_COLLECTION_DELETE_FAILED}"
        return 1
    fi
}

#######################################
# Check if a collection exists
# Arguments:
#   $1 - Collection name
# Returns: 0 if exists, 1 if not
#######################################
qdrant::collections::exists() {
    local collection_name="$1"
    
    local response
    response=$(qdrant::api::request "GET" "/collections/${collection_name}" 2>/dev/null)
    
    # Check if request was successful
    if [[ $? -eq 0 ]]; then
        local status
        status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
        [[ "$status" == "ok" ]]
    else
        return 1
    fi
}

#######################################
# List all collections
# Outputs: Collection information
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::list() {
    qdrant::collections::verify_init || return 1
    
    local response
    response=$(qdrant::api::request "GET" "/collections" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to retrieve collections"
        return 1
    fi
    
    local status
    status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
    
    if [[ "$status" != "ok" ]]; then
        log::error "API returned error status"
        return 1
    fi
    
    echo "=== Qdrant Collections ==="
    echo
    
    # Extract collection names
    local collection_names
    collection_names=$(echo "$response" | jq -r '.result.collections[].name' 2>/dev/null)
    
    if [[ -z "$collection_names" ]]; then
        echo "No collections found"
        return 0
    fi
    
    # Display collection names (simple format for now - detailed info available via collection-info)
    local count=0
    while IFS= read -r collection_name; do
        if [[ -n "$collection_name" ]]; then
            echo "📁 $collection_name"
            count=$((count + 1))
        fi
    done <<< "$collection_names"
    
    echo ""
    echo "Total: $count collections"
    echo ""
    echo "💡 Use 'resource-qdrant collections info <name>' for detailed information"
}

#######################################
# List collection names only (simple format)
# Outputs: Collection names, one per line
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::list_simple() {
    local response
    response=$(qdrant::api::request "GET" "/collections" 2>/dev/null)
    
    # Check if request was successful
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.result.collections[]?.name // empty' 2>/dev/null
    else
        return 1
    fi
}

#######################################
# Get collection information
# Arguments:
#   $1 - Collection name
# Outputs: Collection details
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::info() {
    local collection_name="$1"
    
    if [[ -z "$collection_name" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local response
    response=$(qdrant::api::request "GET" "/collections/${collection_name}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to retrieve collection information for: $collection_name"
        log::info "Check that Qdrant is running and the collection name is correct"
        return 1
    fi
    
    local status
    status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
    
    if [[ "$status" != "ok" ]]; then
        log::error "Collection not found: $collection_name"
        log::info "Use 'resource-qdrant collections list' to see available collections"
        return 1
    fi
    
    echo "=== Collection Information: $collection_name ==="
    echo
    
    # Extract and display information
    echo "$response" | jq -r '
        .result |
        "Status: " + .status +
        "\nVectors Count: " + (.vectors_count | tostring) +
        "\nIndexed Vectors: " + (.indexed_vectors_count | tostring) +
        "\nPoints Count: " + (.points_count | tostring) +
        "\nSegments Count: " + (.segments_count | tostring) +
        "\n\nConfiguration:" +
        "\n  Vector Size: " + (.config.params.vectors.size | tostring) +
        "\n  Distance Metric: " + .config.params.vectors.distance +
        "\n  Replication Factor: " + (.config.params.replication_factor | tostring) +
        "\n  Write Consistency: " + (.config.params.write_consistency_factor | tostring) +
        "\n\nOptimizer Config:" +
        "\n  Deleted Threshold: " + (.config.optimizer_config.deleted_threshold | tostring) +
        "\n  Vacuum Min Vectors: " + (.config.optimizer_config.vacuum_min_vector_number | tostring) +
        "\n  Max Segment Size: " + (.config.optimizer_config.max_segment_size | tostring) +
        "\n\nHNSW Config:" +
        "\n  M: " + (.config.hnsw_config.m | tostring) +
        "\n  EF Construct: " + (.config.hnsw_config.ef_construct | tostring) +
        "\n  Full Scan Threshold: " + (.config.hnsw_config.full_scan_threshold | tostring) +
        "\n  On Disk: " + (.config.hnsw_config.on_disk | tostring)
    ' 2>/dev/null || {
        log::error "Failed to parse collection information"
        return 1
    }
    
    echo
}

#######################################
# Get vector count for a collection
# Arguments:
#   $1 - Collection name
# Outputs: Number of vectors
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::get_vector_count() {
    local collection_name="$1"
    
    local response
    response=$(qdrant::api::request "GET" "/collections/${collection_name}" 2>/dev/null)
    
    # Check if request was successful
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.result.vectors_count // 0' 2>/dev/null || echo "0"
    else
        return 1
    fi
}

#######################################
# Count total number of collections
# Outputs: Number of collections
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::count() {
    local response
    response=$(qdrant::api::request "GET" "/collections" 2>/dev/null)
    
    # Check if request was successful
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.result.collections | length' 2>/dev/null || echo "0"
    else
        return 1
    fi
}

#######################################
# Get index statistics for a collection
# Arguments:
#   $1 - Collection name
# Outputs: Index statistics
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::index_stats() {
    local collection_name="$1"
    
    if [[ -z "$collection_name" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local response
    response=$(qdrant::api::request "GET" "/collections/${collection_name}/cluster" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to retrieve index statistics"
        return 1
    fi
    
    echo "=== Index Statistics: $collection_name ==="
    echo
    
    # Display cluster and indexing information
    echo "$response" | jq -r '
        .result |
        "Peer ID: " + (.peer_id | tostring) +
        "\nShard ID: " + (.shard_id | tostring) +
        "\n\nRemote Shards:"
    ' 2>/dev/null
    
    echo "$response" | jq -r '
        .result.remote_shards[]? |
        "  Shard " + (.shard_id | tostring) + 
        " on peer " + (.peer_id | tostring) + 
        " (state: " + .state + ")"
    ' 2>/dev/null
    
    echo
}

#######################################
# Get index statistics for all collections
# Outputs: Index statistics for all collections
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::index_stats_all() {
    local collections
    collections=$(qdrant::collections::list_simple)
    
    if [[ $? -ne 0 || -z "$collections" ]]; then
        log::info "No collections found"
        return 0
    fi
    
    echo "=== Index Statistics for All Collections ==="
    echo
    
    while IFS= read -r collection_name; do
        if [[ -n "$collection_name" ]]; then
            qdrant::collections::index_stats "$collection_name"
            echo "----------------------------------------"
            echo
        fi
    done <<< "$collections"
}

#######################################
# Recreate a collection with new parameters
# Arguments:
#   $1 - Collection name
#   $2 - New vector size
#   $3 - New distance metric
#   $4 - Force recreation (yes/no, default: no)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::recreate() {
    local collection_name="$1"
    local vector_size="$2"
    local distance_metric="$3"
    local force="${4:-no}"
    
    if [[ -z "$collection_name" || -z "$vector_size" || -z "$distance_metric" ]]; then
        log::error "Collection name, vector size, and distance metric are required"
        return 1
    fi
    
    if ! qdrant::collections::exists "$collection_name"; then
        log::info "Collection does not exist, creating new one"
        qdrant::collections::create "$collection_name" "$vector_size" "$distance_metric"
        return $?
    fi
    
    if [[ "$force" != "yes" ]]; then
        local vector_count
        vector_count=$(qdrant::collections::get_vector_count "$collection_name" 2>/dev/null || echo "unknown")
        
        if [[ "$vector_count" != "0" && "$vector_count" != "unknown" ]]; then
            log::error "Collection contains $vector_count vectors. Use --force yes to recreate."
            return 1
        fi
    fi
    
    # Delete existing collection
    if ! qdrant::collections::delete "$collection_name" "yes"; then
        log::error "Failed to delete existing collection"
        return 1
    fi
    
    # Create new collection
    if qdrant::collections::create "$collection_name" "$vector_size" "$distance_metric"; then
        log::success "Collection recreated: $collection_name"
        return 0
    else
        log::error "Failed to recreate collection"
        return 1
    fi
}

#######################################
# Parse collection creation arguments
# Arguments: All CLI arguments passed to create command
# Sets global variables: PARSED_NAME, PARSED_DIMENSIONS, PARSED_MODEL, PARSED_DISTANCE
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::parse_create_args() {
    local name=""
    local vector_size=""
    local distance="Cosine"
    local model=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --model|-m)
                model="$2"
                shift 2
                ;;
            --dimensions|-d)
                vector_size="$2"
                shift 2
                ;;
            --distance)
                distance="$2"
                shift 2
                ;;
            --help|-h)
                # Help is requested, return special code
                return 2
                ;;
            -*)
                log::error "Unknown option: $1"
                return 1
                ;;
            *)
                if [[ -z "$name" ]]; then
                    name="$1"
                elif [[ -z "$vector_size" ]] && [[ "$1" =~ ^[0-9]+$ ]]; then
                    vector_size="$1"
                elif [[ -z "$model" ]]; then
                    model="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Collection name required"
        return 1
    fi
    
    # Set global variables for caller
    PARSED_NAME="$name"
    PARSED_DIMENSIONS="$vector_size"
    PARSED_MODEL="$model"
    PARSED_DISTANCE="$distance"
    
    return 0
}

#######################################
# Create collection with parsed arguments
# Uses global variables set by parse_create_args
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::create_parsed() {
    if [[ -z "${PARSED_NAME:-}" ]]; then
        log::error "No parsed arguments available. Call parse_create_args first."
        return 1
    fi
    
    # Use model-based creation if model specified
    if [[ -n "${PARSED_MODEL:-}" ]]; then
        qdrant::collections::create_from_model "$PARSED_NAME" "$PARSED_MODEL" "$PARSED_DISTANCE"
    else
        # Use traditional creation with dimensions
        local vector_size="${PARSED_DIMENSIONS:-1536}"
        qdrant::collections::create "$PARSED_NAME" "$vector_size" "$PARSED_DISTANCE"
    fi
}

#######################################
# SMART COLLECTION FEATURES
# Enhanced functions that integrate with model discovery
#######################################

# Source dependencies if not already loaded
if ! command -v qdrant::models::get_model_dimensions >/dev/null 2>&1; then
    # shellcheck disable=SC1091
    source "${QDRANT_COLLECTIONS_DIR}/models.sh" 2>/dev/null || true
fi

#######################################
# Create collection with automatic dimension detection from model
# Arguments:
#   $1 - Collection name
#   $2 - Model name (will detect dimensions automatically)
#   $3 - Distance metric (optional, default: Cosine)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::create_from_model() {
    local collection_name="$1"
    local model_name="$2"
    local distance_metric="${3:-Cosine}"
    
    if [[ -z "$collection_name" ]] || [[ -z "$model_name" ]]; then
        log::error "Collection name and model name are required"
        return 1
    fi
    
    # Get model dimensions
    log::info "Detecting dimensions for model: $model_name"
    local dimensions
    dimensions=$(qdrant::models::get_model_dimensions "$model_name")
    
    if [[ "$dimensions" == "unknown" ]] || [[ -z "$dimensions" ]]; then
        log::error "Cannot determine dimensions for model: $model_name"
        log::info "Please ensure the model is installed and is an embedding model"
        return 1
    fi
    
    log::info "Model '$model_name' has $dimensions dimensions"
    
    # Create collection with detected dimensions
    qdrant::collections::create "$collection_name" "$dimensions" "$distance_metric"
}

#######################################
# Create or ensure collection exists with proper dimensions
# Arguments:
#   $1 - Collection name
#   $2 - Model name or dimensions
#   $3 - Distance metric (optional, default: Cosine)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::ensure() {
    local collection_name="$1"
    local model_or_dims="$2"
    local distance_metric="${3:-Cosine}"
    
    if [[ -z "$collection_name" ]] || [[ -z "$model_or_dims" ]]; then
        log::error "Collection name and model/dimensions are required"
        return 1
    fi
    
    # Check if collection already exists
    if qdrant::collections::exists "$collection_name"; then
        # Validate dimensions match
        local existing_dims
        existing_dims=$(qdrant::collections::get_dimensions "$collection_name")
        
        local required_dims
        if [[ "$model_or_dims" =~ ^[0-9]+$ ]]; then
            required_dims="$model_or_dims"
        else
            required_dims=$(qdrant::models::get_model_dimensions "$model_or_dims")
        fi
        
        if [[ "$existing_dims" == "$required_dims" ]]; then
            log::debug "Collection '$collection_name' exists with correct dimensions ($existing_dims)"
            return 0
        else
            log::error "Collection '$collection_name' exists but has wrong dimensions"
            log::error "Existing: $existing_dims, Required: $required_dims"
            log::info "Consider using a different collection name or recreating with --force"
            return 1
        fi
    fi
    
    # Create collection
    if [[ "$model_or_dims" =~ ^[0-9]+$ ]]; then
        # Direct dimensions provided
        qdrant::collections::create "$collection_name" "$model_or_dims" "$distance_metric"
    else
        # Model name provided
        qdrant::collections::create_from_model "$collection_name" "$model_or_dims" "$distance_metric"
    fi
}

#######################################
# List collections with model compatibility info
# Outputs: Collection information with compatible models
# Returns: 0 on success
#######################################
qdrant::collections::list_with_models() {
    qdrant::collections::verify_init || return 1
    
    local response
    response=$(qdrant::api::request "GET" "/collections" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to retrieve collections"
        return 1
    fi
    
    echo "=== Qdrant Collections with Model Compatibility ==="
    echo
    
    # Get all embedding models
    local embedding_models
    if command -v qdrant::models::get_embedding_models >/dev/null 2>&1; then
        embedding_models=$(qdrant::models::get_embedding_models)
    else
        embedding_models="[]"
    fi
    
    # Extract collection names
    local collection_names
    collection_names=$(echo "$response" | jq -r '.result.collections[].name' 2>/dev/null)
    
    if [[ -z "$collection_names" ]]; then
        echo "No collections found"
        return 0
    fi
    
    while IFS= read -r collection_name; do
        if [[ -n "$collection_name" ]]; then
            # Get collection info
            local coll_info
            coll_info=$(qdrant::api::request "GET" "/collections/${collection_name}" 2>/dev/null)
            
            local dimensions
            dimensions=$(echo "$coll_info" | jq -r '.result.config.params.vectors.size // "unknown"' 2>/dev/null)
            
            local vector_count
            vector_count=$(echo "$coll_info" | jq -r '.result.vectors_count // 0' 2>/dev/null)
            
            echo "📁 $collection_name"
            echo "   Dimensions: $dimensions"
            echo "   Vectors: $vector_count"
            
            # Find compatible models
            if [[ "$dimensions" != "unknown" ]] && [[ "$embedding_models" != "[]" ]]; then
                local compatible_models
                compatible_models=$(echo "$embedding_models" | jq -r ".[] | select(.dimensions == $dimensions) | .name" 2>/dev/null)
                
                if [[ -n "$compatible_models" ]]; then
                    echo "   Compatible Models:"
                    while IFS= read -r model; do
                        echo "     • $model"
                    done <<< "$compatible_models"
                else
                    echo "   ⚠️  No compatible embedding models found"
                fi
            fi
            echo
        fi
    done <<< "$collection_names"
}

#######################################
# Get collection dimensions (already defined in search.sh but adding here for completeness)
# Arguments:
#   $1 - Collection name
# Outputs: Dimension count or "unknown"
# Returns: 0 on success
#######################################
if ! command -v qdrant::collections::get_dimensions >/dev/null 2>&1; then
    qdrant::collections::get_dimensions() {
        local collection="$1"
        
        if [[ -z "$collection" ]]; then
            echo "unknown"
            return 1
        fi
        
        local response
        response=$(qdrant::api::request "GET" "/collections/${collection}" 2>/dev/null)
        
        if [[ -n "$response" ]]; then
            local dimensions
            dimensions=$(echo "$response" | jq -r '.result.config.params.vectors.size // "unknown"' 2>/dev/null)
            
            if [[ "$dimensions" != "unknown" ]] && [[ "$dimensions" != "null" ]]; then
                echo "$dimensions"
                return 0
            fi
        fi
        
        echo "unknown"
        return 1
    }
fi

#######################################
# Migrate collection to new dimensions
# Arguments:
#   $1 - Source collection name
#   $2 - Target collection name
#   $3 - New model name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::collections::migrate() {
    local source_collection="$1"
    local target_collection="$2"
    local new_model="$3"
    
    if [[ -z "$source_collection" ]] || [[ -z "$target_collection" ]] || [[ -z "$new_model" ]]; then
        log::error "Source collection, target collection, and new model are required"
        return 1
    fi
    
    if ! qdrant::collections::exists "$source_collection"; then
        log::error "Source collection does not exist: $source_collection"
        return 1
    fi
    
    if qdrant::collections::exists "$target_collection"; then
        log::error "Target collection already exists: $target_collection"
        log::info "Please choose a different name or delete the existing collection"
        return 1
    fi
    
    # Get new model dimensions
    local new_dimensions
    new_dimensions=$(qdrant::models::get_model_dimensions "$new_model")
    
    if [[ "$new_dimensions" == "unknown" ]]; then
        log::error "Cannot determine dimensions for model: $new_model"
        return 1
    fi
    
    log::info "Migration Plan:"
    log::info "  Source: $source_collection"
    log::info "  Target: $target_collection"
    log::info "  New Model: $new_model ($new_dimensions dimensions)"
    echo
    
    # Create target collection
    log::info "Creating target collection..."
    if ! qdrant::collections::create "$target_collection" "$new_dimensions" "Cosine"; then
        log::error "Failed to create target collection"
        return 1
    fi
    
    log::warn "Note: Automatic vector migration requires re-embedding all data"
    log::info "This feature requires access to original text data in payloads"
    log::info "Manual migration may be needed if original text is not available"
    
    return 0
}

#######################################
# List brief collection info for status displays
# Outputs: Brief collection information
# Returns: 0 on success
#######################################
qdrant::collections::list_brief() {
    local response
    response=$(qdrant::api::request "GET" "/collections" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.result.collections[] | "\(.name)"' 2>/dev/null
    fi
}