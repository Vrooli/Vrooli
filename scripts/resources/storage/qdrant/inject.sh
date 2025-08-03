#!/usr/bin/env bash
set -euo pipefail

# Qdrant Vector Database Injection Adapter
# This script handles injection of collections and vectors into Qdrant
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject collections and vector data into Qdrant vector database"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common utilities
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"

# Source Qdrant configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default Qdrant settings
readonly DEFAULT_QDRANT_HOST="http://localhost:6333"

# Qdrant settings (can be overridden by environment)
QDRANT_HOST="${QDRANT_HOST:-$DEFAULT_QDRANT_HOST}"

# Operation tracking
declare -a QDRANT_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
qdrant_inject::usage() {
    cat << EOF
Qdrant Vector Database Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects collections and vector data into Qdrant based on scenario configuration.
    Supports validation, injection, status checks, and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected collections
    --rollback    Rollback injected collections
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "collections": [
        {
          "name": "documents",
          "size": 384,
          "distance": "Cosine",
          "on_disk": false
        }
      ],
      "vectors": [
        {
          "collection": "documents",
          "vectors": "path/to/vectors.json",
          "batch_size": 100
        }
      ],
      "indices": [
        {
          "collection": "documents",
          "field": "category",
          "type": "keyword"
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"collections": [{"name": "embeddings", "size": 768, "distance": "Cosine"}]}'
    
    # Create collections and inject vectors
    $0 --inject '{"collections": [{"name": "docs", "size": 384}], "vectors": [{"collection": "docs", "vectors": "data/vectors.json"}]}'

EOF
}

#######################################
# Check if Qdrant is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
qdrant_inject::check_accessibility() {
    # Check if Qdrant is running
    if curl -s --max-time 5 "${QDRANT_HOST}/collections" >/dev/null 2>&1; then
        log::debug "Qdrant is accessible at $QDRANT_HOST"
        return 0
    else
        log::error "Qdrant is not accessible at $QDRANT_HOST"
        log::info "Ensure Qdrant is running: ./scripts/resources/storage/qdrant/manage.sh --action start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
qdrant_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    QDRANT_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Qdrant rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
qdrant_inject::execute_rollback() {
    if [[ ${#QDRANT_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Qdrant rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Qdrant rollback actions..."
    
    local success_count=0
    local total_count=${#QDRANT_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#QDRANT_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${QDRANT_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Qdrant rollback completed: $success_count/$total_count actions successful"
    QDRANT_ROLLBACK_ACTIONS=()
}

#######################################
# Validate collection configuration
# Arguments:
#   $1 - collections configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
qdrant_inject::validate_collections() {
    local collections_config="$1"
    
    log::debug "Validating collection configurations..."
    
    # Check if collections is an array
    local collections_type
    collections_type=$(echo "$collections_config" | jq -r 'type')
    
    if [[ "$collections_type" != "array" ]]; then
        log::error "Collections configuration must be an array, got: $collections_type"
        return 1
    fi
    
    # Validate each collection
    local collection_count
    collection_count=$(echo "$collections_config" | jq 'length')
    
    for ((i=0; i<collection_count; i++)); do
        local collection
        collection=$(echo "$collections_config" | jq -c ".[$i]")
        
        # Check required fields
        local name size
        name=$(echo "$collection" | jq -r '.name // empty')
        size=$(echo "$collection" | jq -r '.size // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Collection at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$size" ]]; then
            log::error "Collection '$name' missing required 'size' field"
            return 1
        fi
        
        # Validate size is a positive integer
        if ! [[ "$size" =~ ^[0-9]+$ ]] || [[ "$size" -le 0 ]]; then
            log::error "Collection '$name' has invalid size: $size (must be positive integer)"
            return 1
        fi
        
        # Validate distance metric if specified
        local distance
        distance=$(echo "$collection" | jq -r '.distance // "Cosine"')
        
        case "$distance" in
            Cosine|Dot|Euclidean|Manhattan)
                log::debug "Collection '$name' has valid distance metric: $distance"
                ;;
            *)
                log::error "Collection '$name' has invalid distance metric: $distance"
                return 1
                ;;
        esac
        
        log::debug "Collection '$name' configuration is valid"
    done
    
    log::success "All collection configurations are valid"
    return 0
}

#######################################
# Validate vectors configuration
# Arguments:
#   $1 - vectors configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
qdrant_inject::validate_vectors() {
    local vectors_config="$1"
    
    log::debug "Validating vector configurations..."
    
    # Check if vectors is an array
    local vectors_type
    vectors_type=$(echo "$vectors_config" | jq -r 'type')
    
    if [[ "$vectors_type" != "array" ]]; then
        log::error "Vectors configuration must be an array, got: $vectors_type"
        return 1
    fi
    
    # Validate each vector batch
    local vector_count
    vector_count=$(echo "$vectors_config" | jq 'length')
    
    for ((i=0; i<vector_count; i++)); do
        local vector_batch
        vector_batch=$(echo "$vectors_config" | jq -c ".[$i]")
        
        # Check required fields
        local collection vectors_file
        collection=$(echo "$vector_batch" | jq -r '.collection // empty')
        vectors_file=$(echo "$vector_batch" | jq -r '.vectors // empty')
        
        if [[ -z "$collection" ]]; then
            log::error "Vector batch at index $i missing required 'collection' field"
            return 1
        fi
        
        if [[ -z "$vectors_file" ]]; then
            log::error "Vector batch for collection '$collection' missing required 'vectors' field"
            return 1
        fi
        
        # Check if vectors file exists
        local vectors_path="$VROOLI_PROJECT_ROOT/$vectors_file"
        if [[ ! -f "$vectors_path" ]]; then
            log::error "Vectors file not found: $vectors_path"
            return 1
        fi
        
        log::debug "Vector batch for collection '$collection' is valid"
    done
    
    log::success "All vector configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
qdrant_inject::validate_config() {
    local config="$1"
    
    log::info "Validating Qdrant injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Qdrant injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_collections has_vectors has_indices
    has_collections=$(echo "$config" | jq -e '.collections' >/dev/null 2>&1 && echo "true" || echo "false")
    has_vectors=$(echo "$config" | jq -e '.vectors' >/dev/null 2>&1 && echo "true" || echo "false")
    has_indices=$(echo "$config" | jq -e '.indices' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_collections" == "false" && "$has_vectors" == "false" && "$has_indices" == "false" ]]; then
        log::error "Qdrant injection configuration must have 'collections', 'vectors', or 'indices'"
        return 1
    fi
    
    # Validate collections if present
    if [[ "$has_collections" == "true" ]]; then
        local collections
        collections=$(echo "$config" | jq -c '.collections')
        
        if ! qdrant_inject::validate_collections "$collections"; then
            return 1
        fi
    fi
    
    # Validate vectors if present
    if [[ "$has_vectors" == "true" ]]; then
        local vectors
        vectors=$(echo "$config" | jq -c '.vectors')
        
        if ! qdrant_inject::validate_vectors "$vectors"; then
            return 1
        fi
    fi
    
    log::success "Qdrant injection configuration is valid"
    return 0
}

#######################################
# Create collection
# Arguments:
#   $1 - collection configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
qdrant_inject::create_collection() {
    local collection_config="$1"
    
    local name size distance on_disk
    name=$(echo "$collection_config" | jq -r '.name')
    size=$(echo "$collection_config" | jq -r '.size')
    distance=$(echo "$collection_config" | jq -r '.distance // "Cosine"')
    on_disk=$(echo "$collection_config" | jq -r '.on_disk // false')
    
    log::info "Creating collection: $name (size: $size, distance: $distance)"
    
    # Prepare collection configuration
    local collection_json=$(cat <<EOF
{
    "vectors": {
        "size": $size,
        "distance": "$distance",
        "on_disk": $on_disk
    }
}
EOF
)
    
    # Create collection via Qdrant API
    local response
    response=$(curl -s -X PUT "${QDRANT_HOST}/collections/${name}" \
        -H "Content-Type: application/json" \
        -d "$collection_json" 2>&1)
    
    if [[ $? -eq 0 ]] && echo "$response" | jq -e '.result' >/dev/null 2>&1; then
        log::success "Created collection: $name"
        
        # Add rollback action
        qdrant_inject::add_rollback_action \
            "Delete collection: $name" \
            "curl -s -X DELETE '${QDRANT_HOST}/collections/${name}' >/dev/null 2>&1"
        
        return 0
    else
        # Check if collection already exists
        if curl -s "${QDRANT_HOST}/collections/${name}" | jq -e '.result' >/dev/null 2>&1; then
            log::warn "Collection '$name' already exists"
            return 0
        else
            log::error "Failed to create collection: $name"
            return 1
        fi
    fi
}

#######################################
# Insert vectors into collection
# Arguments:
#   $1 - vector batch configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
qdrant_inject::insert_vectors() {
    local vector_config="$1"
    
    local collection vectors_file batch_size
    collection=$(echo "$vector_config" | jq -r '.collection')
    vectors_file=$(echo "$vector_config" | jq -r '.vectors')
    batch_size=$(echo "$vector_config" | jq -r '.batch_size // 100')
    
    # Resolve vectors file path
    local vectors_path="$VROOLI_PROJECT_ROOT/$vectors_file"
    
    log::info "Inserting vectors into collection '$collection' from: $vectors_file"
    
    # Load vectors from file
    local vectors_data
    vectors_data=$(cat "$vectors_path")
    
    # Validate vectors data is valid JSON
    if ! echo "$vectors_data" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in vectors file: $vectors_file"
        return 1
    fi
    
    # Insert vectors via Qdrant API
    local response
    response=$(curl -s -X PUT "${QDRANT_HOST}/collections/${collection}/points" \
        -H "Content-Type: application/json" \
        -d "$vectors_data" 2>&1)
    
    if [[ $? -eq 0 ]] && echo "$response" | jq -e '.result' >/dev/null 2>&1; then
        log::success "Inserted vectors into collection: $collection"
        
        # Note: Rollback for vectors is complex as it requires tracking point IDs
        # For now, we'll note that the entire collection can be deleted if needed
        
        return 0
    else
        log::error "Failed to insert vectors into collection: $collection"
        return 1
    fi
}

#######################################
# Create index
# Arguments:
#   $1 - index configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
qdrant_inject::create_index() {
    local index_config="$1"
    
    local collection field index_type
    collection=$(echo "$index_config" | jq -r '.collection')
    field=$(echo "$index_config" | jq -r '.field')
    index_type=$(echo "$index_config" | jq -r '.type // "keyword"')
    
    log::info "Creating index on collection '$collection' field '$field' (type: $index_type)"
    
    # Create index via Qdrant API
    local index_json=$(cat <<EOF
{
    "field_name": "$field",
    "field_schema": {
        "type": "$index_type"
    }
}
EOF
)
    
    local response
    response=$(curl -s -X PUT "${QDRANT_HOST}/collections/${collection}/index" \
        -H "Content-Type: application/json" \
        -d "$index_json" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "Created index on collection '$collection' field '$field'"
        return 0
    else
        log::error "Failed to create index on collection '$collection' field '$field'"
        return 1
    fi
}

#######################################
# Inject collections
# Arguments:
#   $1 - collections configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
qdrant_inject::inject_collections() {
    local collections_config="$1"
    
    log::info "Creating Qdrant collections..."
    
    local collection_count
    collection_count=$(echo "$collections_config" | jq 'length')
    
    if [[ "$collection_count" -eq 0 ]]; then
        log::info "No collections to create"
        return 0
    fi
    
    local failed_collections=()
    
    for ((i=0; i<collection_count; i++)); do
        local collection
        collection=$(echo "$collections_config" | jq -c ".[$i]")
        
        local collection_name
        collection_name=$(echo "$collection" | jq -r '.name')
        
        if ! qdrant_inject::create_collection "$collection"; then
            failed_collections+=("$collection_name")
        fi
    done
    
    if [[ ${#failed_collections[@]} -eq 0 ]]; then
        log::success "All collections created successfully"
        return 0
    else
        log::error "Failed to create collections: ${failed_collections[*]}"
        return 1
    fi
}

#######################################
# Inject vectors
# Arguments:
#   $1 - vectors configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
qdrant_inject::inject_vector_data() {
    local vectors_config="$1"
    
    log::info "Inserting vectors into Qdrant..."
    
    local vector_count
    vector_count=$(echo "$vectors_config" | jq 'length')
    
    if [[ "$vector_count" -eq 0 ]]; then
        log::info "No vectors to insert"
        return 0
    fi
    
    local failed_batches=()
    
    for ((i=0; i<vector_count; i++)); do
        local vector_batch
        vector_batch=$(echo "$vectors_config" | jq -c ".[$i]")
        
        local collection
        collection=$(echo "$vector_batch" | jq -r '.collection')
        
        if ! qdrant_inject::insert_vectors "$vector_batch"; then
            failed_batches+=("$collection")
        fi
    done
    
    if [[ ${#failed_batches[@]} -eq 0 ]]; then
        log::success "All vectors inserted successfully"
        return 0
    else
        log::error "Failed to insert vectors for collections: ${failed_batches[*]}"
        return 1
    fi
}

#######################################
# Inject indices
# Arguments:
#   $1 - indices configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
qdrant_inject::inject_indices() {
    local indices_config="$1"
    
    log::info "Creating Qdrant indices..."
    
    local index_count
    index_count=$(echo "$indices_config" | jq 'length')
    
    if [[ "$index_count" -eq 0 ]]; then
        log::info "No indices to create"
        return 0
    fi
    
    local failed_indices=()
    
    for ((i=0; i<index_count; i++)); do
        local index
        index=$(echo "$indices_config" | jq -c ".[$i]")
        
        local collection field
        collection=$(echo "$index" | jq -r '.collection')
        field=$(echo "$index" | jq -r '.field')
        
        if ! qdrant_inject::create_index "$index"; then
            failed_indices+=("${collection}.${field}")
        fi
    done
    
    if [[ ${#failed_indices[@]} -eq 0 ]]; then
        log::success "All indices created successfully"
        return 0
    else
        log::error "Failed to create indices: ${failed_indices[*]}"
        return 1
    fi
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
qdrant_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Qdrant"
    
    # Check Qdrant accessibility
    if ! qdrant_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    QDRANT_ROLLBACK_ACTIONS=()
    
    # Inject collections if present
    local has_collections
    has_collections=$(echo "$config" | jq -e '.collections' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_collections" == "true" ]]; then
        local collections
        collections=$(echo "$config" | jq -c '.collections')
        
        if ! qdrant_inject::inject_collections "$collections"; then
            log::error "Failed to create collections"
            qdrant_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject vectors if present
    local has_vectors
    has_vectors=$(echo "$config" | jq -e '.vectors' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_vectors" == "true" ]]; then
        local vectors
        vectors=$(echo "$config" | jq -c '.vectors')
        
        if ! qdrant_inject::inject_vector_data "$vectors"; then
            log::error "Failed to insert vectors"
            qdrant_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject indices if present
    local has_indices
    has_indices=$(echo "$config" | jq -e '.indices' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_indices" == "true" ]]; then
        local indices
        indices=$(echo "$config" | jq -c '.indices')
        
        if ! qdrant_inject::inject_indices "$indices"; then
            log::error "Failed to create indices"
            qdrant_inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ Qdrant data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
qdrant_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Qdrant injection status"
    
    # Check Qdrant accessibility
    if ! qdrant_inject::check_accessibility; then
        return 1
    fi
    
    # Check collection status
    local has_collections
    has_collections=$(echo "$config" | jq -e '.collections' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_collections" == "true" ]]; then
        local collections
        collections=$(echo "$config" | jq -c '.collections')
        
        log::info "Checking collection status..."
        
        local collection_count
        collection_count=$(echo "$collections" | jq 'length')
        
        for ((i=0; i<collection_count; i++)); do
            local collection
            collection=$(echo "$collections" | jq -c ".[$i]")
            
            local name
            name=$(echo "$collection" | jq -r '.name')
            
            # Check if collection exists
            local response
            response=$(curl -s "${QDRANT_HOST}/collections/${name}" 2>/dev/null)
            
            if echo "$response" | jq -e '.result' >/dev/null 2>&1; then
                local point_count
                point_count=$(echo "$response" | jq -r '.result.points_count // 0')
                log::success "✅ Collection '$name' exists (points: $point_count)"
            else
                log::error "❌ Collection '$name' not found"
            fi
        done
    fi
    
    # Get overall cluster info
    log::info "Fetching cluster information..."
    
    local cluster_info
    cluster_info=$(curl -s "${QDRANT_HOST}/collections" 2>/dev/null)
    
    if echo "$cluster_info" | jq -e '.result.collections' >/dev/null 2>&1; then
        local total_collections
        total_collections=$(echo "$cluster_info" | jq '.result.collections | length')
        log::info "Total collections in cluster: $total_collections"
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
qdrant_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        qdrant_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            qdrant_inject::validate_config "$config"
            ;;
        "--inject")
            qdrant_inject::inject_data "$config"
            ;;
        "--status")
            qdrant_inject::check_status "$config"
            ;;
        "--rollback")
            qdrant_inject::execute_rollback
            ;;
        "--help")
            qdrant_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            qdrant_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        qdrant_inject::usage
        exit 1
    fi
    
    qdrant_inject::main "$@"
fi