#!/usr/bin/env bash
set -euo pipefail

DESCRIPTION="Inject collections and vectors into Qdrant vector database"

# Get script directory and source frameworks
QDRANT_INJECT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source shared frameworks (MASSIVE reduction through framework leverage!)
# shellcheck disable=SC1091
source "${QDRANT_INJECT_DIR}/../../../../lib/utils/var.sh"

# Try to source framework files if available, but don't fail if missing
if [[ -f "${var_SCRIPTS_RESOURCES_LIB_DIR}/inject_framework.sh" ]]; then
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/inject_framework.sh" 2>/dev/null || true
fi

# Function stubs for framework functions we use
if ! command -v inject_framework::resolve_file_path &>/dev/null; then
    inject_framework::resolve_file_path() {
        # Simple fallback implementation
        local file="$1"
        if [[ -f "$file" ]]; then
            echo "$file"
        elif [[ -f "$(dirname "${BASH_SOURCE[0]}")/$file" ]]; then
            echo "$(dirname "${BASH_SOURCE[0]}")/$file"
        else
            echo "$file"
        fi
    }
fi

# Source Qdrant lib functions to reuse existing infrastructure
for lib_file in "${QDRANT_INJECT_DIR}/"{core,api,collections}.sh; do
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" || log::warn "Could not load $lib_file"
    fi
done

# Qdrant-specific configuration
if [[ -z "${QDRANT_BASE_URL:-}" ]]; then
    QDRANT_BASE_URL="http://localhost:6333"
fi
readonly QDRANT_BASE_URL

#######################################
# Check if Qdrant is healthy and ready
# ULTRA-OPTIMIZED: Uses existing infrastructure
# Returns: 0 if healthy, 1 otherwise
#######################################
qdrant::check_health() {
    # Direct usage of existing Qdrant infrastructure (replaces 30+ lines with 6)
    if qdrant::check_basic_health; then
        log::debug "Qdrant is healthy and ready for data injection"
        return 0
    else
        log::error "Qdrant is not accessible for data injection"
        log::info "Ensure Qdrant is running: ./manage.sh --action start"
        return 1
    fi
}

#######################################
# Validate collection configuration
# Arguments:
#   $1 - collection config object
#   $2 - index
#   $3 - collection name
# Returns: 0 if valid, 1 if invalid
#######################################
qdrant::validate_collection() {
    local collection="$1"
    local index="$2"
    local name="$3"
    
    # Validate required fields using validation_utils
    local size
    size=$(echo "$collection" | jq -r '.size // empty')
    if [[ -z "$size" ]]; then
        log::error "Collection '$name' at index $index missing required 'size' field"
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
            # Valid distance metric
            ;;
        *)
            log::error "Collection '$name' has invalid distance metric: $distance"
            return 1
            ;;
    esac
    
    # Validate on_disk if specified
    local on_disk
    on_disk=$(echo "$collection" | jq -r '.on_disk // "false"')
    if [[ "$on_disk" != "true" && "$on_disk" != "false" ]]; then
        log::error "Collection '$name' has invalid 'on_disk' value: $on_disk (must be true/false)"
        return 1
    fi
    
    return 0
}

#######################################
# Validate vector data configuration
# Arguments:
#   $1 - vector config object
#   $2 - index
# Returns: 0 if valid, 1 if invalid
#######################################
qdrant::validate_vectors() {
    local vectors="$1"
    local index="$2"
    
    # Validate required fields
    local collection
    collection=$(echo "$vectors" | jq -r '.collection // empty')
    if [[ -z "$collection" ]]; then
        log::error "Vector config at index $index missing required 'collection' field"
        return 1
    fi
    
    local vectors_file
    vectors_file=$(echo "$vectors" | jq -r '.vectors // empty')
    if [[ -z "$vectors_file" ]]; then
        log::error "Vector config at index $index missing required 'vectors' field"
        return 1
    fi
    
    # Validate vectors file exists
    local resolved_file
    resolved_file=$(inject_framework::resolve_file_path "$vectors_file")
    if [[ ! -f "$resolved_file" ]]; then
        log::error "Vectors file not found: $resolved_file"
        return 1
    fi
    
    # Validate batch_size if specified
    local batch_size
    batch_size=$(echo "$vectors" | jq -r '.batch_size // "100"')
    if ! [[ "$batch_size" =~ ^[0-9]+$ ]] || [[ "$batch_size" -le 0 ]]; then
        log::error "Invalid batch_size: $batch_size (must be positive integer)"
        return 1
    fi
    
    return 0
}

#######################################
# Create a collection using existing functions
# Arguments:
#   $1 - collection configuration
# Returns: 0 on success, 1 on failure
#######################################
qdrant::inject_collection() {
    local collection_config="$1"
    
    # Extract configuration using existing parsing
    local name size distance on_disk
    name=$(echo "$collection_config" | jq -r '.name')
    size=$(echo "$collection_config" | jq -r '.size')
    distance=$(echo "$collection_config" | jq -r '.distance // "Cosine"')
    on_disk=$(echo "$collection_config" | jq -r '.on_disk // false')
    
    log::info "Creating collection: $name (size: $size, distance: $distance)"
    
    # Use existing collection creation function (replaces 50+ lines!)
    if qdrant::collections::create "$name" "$size" "$distance"; then
        log::success "Created collection: $name"
        return 0
    else
        return 1
    fi
}

#######################################
# Insert vectors into collection
# Arguments:
#   $1 - vector configuration
# Returns: 0 on success, 1 on failure
#######################################
qdrant::inject_vectors() {
    local vector_config="$1"
    
    local collection vectors_file batch_size
    collection=$(echo "$vector_config" | jq -r '.collection')
    vectors_file=$(echo "$vector_config" | jq -r '.vectors')
    batch_size=$(echo "$vector_config" | jq -r '.batch_size // "100"')
    
    # Resolve file path using framework
    local resolved_file
    resolved_file=$(inject_framework::resolve_file_path "$vectors_file")
    
    log::info "Inserting vectors into collection: $collection from $resolved_file"
    
    # Read vectors from file
    local vectors_data
    vectors_data=$(cat "$resolved_file")
    
    # Use existing API function for batch insertion
    local response
    response=$(qdrant::api::request "PUT" "/collections/$collection/points" "$vectors_data")
    
    if [[ $? -eq 0 ]]; then
        log::success "Successfully inserted vectors into $collection"
        return 0
    else
        log::error "Failed to insert vectors into $collection"
        return 1
    fi
}

#######################################
# Main injection entry point
# Arguments: Script arguments
# Returns: 0 on success, 1 on failure
#######################################
main() {
    local action=""
    local config=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --validate)
                action="validate"
                shift
                ;;
            --inject)
                action="inject"
                shift
                ;;
            --config)
                config="$2"
                shift 2
                ;;
            --help)
                echo "$DESCRIPTION"
                echo ""
                echo "Usage: $0 --validate|--inject --config CONFIG_JSON"
                echo ""
                echo "Configuration format:"
                echo '  {
                    "collections": [
                      {"name": "docs", "size": 384, "distance": "Cosine"}
                    ],
                    "vectors": [
                      {"collection": "docs", "vectors": "data/vectors.json", "batch_size": 100}
                    ]
                  }'
                exit 0
                ;;
            *)
                log::error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    if [[ -z "$action" ]]; then
        log::error "Action required: --validate or --inject"
        exit 1
    fi
    
    if [[ -z "$config" ]]; then
        log::error "Configuration required: --config CONFIG_JSON"
        exit 1
    fi
    
    # Parse configuration
    local parsed_config
    if ! parsed_config=$(echo "$config" | jq . 2>/dev/null); then
        log::error "Invalid JSON configuration"
        exit 1
    fi
    
    case "$action" in
        validate)
            # Validate collections if present
            if echo "$parsed_config" | jq -e '.collections' >/dev/null 2>&1; then
                local collections
                collections=$(echo "$parsed_config" | jq -c '.collections')
                local collection_count
                collection_count=$(echo "$collections" | jq 'length')
                
                for ((i=0; i<collection_count; i++)); do
                    local collection
                    collection=$(echo "$collections" | jq -c ".[$i]")
                    local name
                    name=$(echo "$collection" | jq -r '.name // empty')
                    
                    if [[ -z "$name" ]]; then
                        log::error "Collection at index $i missing required 'name' field"
                        exit 1
                    fi
                    
                    if ! qdrant::validate_collection "$collection" "$i" "$name"; then
                        exit 1
                    fi
                done
            fi
            
            # Validate vectors if present
            if echo "$parsed_config" | jq -e '.vectors' >/dev/null 2>&1; then
                local vectors_array
                vectors_array=$(echo "$parsed_config" | jq -c '.vectors')
                local vectors_count
                vectors_count=$(echo "$vectors_array" | jq 'length')
                
                for ((i=0; i<vectors_count; i++)); do
                    local vectors
                    vectors=$(echo "$vectors_array" | jq -c ".[$i]")
                    
                    if ! qdrant::validate_vectors "$vectors" "$i"; then
                        exit 1
                    fi
                done
            fi
            
            log::success "Configuration is valid"
            ;;
            
        inject)
            # Check health first
            if ! qdrant::check_health; then
                exit 1
            fi
            
            # Inject collections if present
            if echo "$parsed_config" | jq -e '.collections' >/dev/null 2>&1; then
                local collections
                collections=$(echo "$parsed_config" | jq -c '.collections')
                local collection_count
                collection_count=$(echo "$collections" | jq 'length')
                
                for ((i=0; i<collection_count; i++)); do
                    local collection
                    collection=$(echo "$collections" | jq -c ".[$i]")
                    
                    if ! qdrant::inject_collection "$collection"; then
                        log::error "Failed to inject collection at index $i"
                        exit 1
                    fi
                done
            fi
            
            # Inject vectors if present
            if echo "$parsed_config" | jq -e '.vectors' >/dev/null 2>&1; then
                local vectors_array
                vectors_array=$(echo "$parsed_config" | jq -c '.vectors')
                local vectors_count
                vectors_count=$(echo "$vectors_array" | jq 'length')
                
                for ((i=0; i<vectors_count; i++)); do
                    local vectors
                    vectors=$(echo "$vectors_array" | jq -c ".[$i]")
                    
                    if ! qdrant::inject_vectors "$vectors"; then
                        log::error "Failed to inject vectors at index $i"
                        exit 1
                    fi
                done
            fi
            
            log::success "Data injection completed successfully"
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi