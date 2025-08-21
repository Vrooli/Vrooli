#!/usr/bin/env bash
set -euo pipefail

DESCRIPTION="Content management for Qdrant vector database"

# Get script directory and source frameworks
QDRANT_CONTENT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source shared frameworks
# shellcheck disable=SC1091
source "${QDRANT_CONTENT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/format.sh"

# Source Qdrant lib functions
for lib_file in "${QDRANT_CONTENT_DIR}/"{core,api,collections}.sh; do
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

# Content storage directory
QDRANT_CONTENT_DIR="${var_SCRIPTS_RESOURCES_DIR}/storage/qdrant/content"
mkdir -p "$QDRANT_CONTENT_DIR"

#######################################
# Add content to Qdrant
# Arguments:
#   --file: Path to content file
#   --name: Optional name for the content
#   --type: Content type (collection, vectors, operation)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::content::add() {
    local file=""
    local name=""
    local type="auto"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            *)
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        log::error "File path required for adding content"
        echo "Usage: resource-qdrant content add --file <path> [--name <name>] [--type <type>]"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Generate name from filename if not provided
    if [[ -z "$name" ]]; then
        name=$(basename "$file" | sed 's/\.[^.]*$//')
    fi
    
    # Auto-detect type if not specified
    if [[ "$type" == "auto" ]]; then
        local content
        content=$(cat "$file")
        
        if echo "$content" | jq -e '.collections' &>/dev/null; then
            type="collections"
        elif echo "$content" | jq -e '.points' &>/dev/null; then
            type="vectors"
        elif echo "$content" | jq -e '.operations' &>/dev/null; then
            type="operations"
        else
            log::error "Unable to auto-detect content type. Please specify --type"
            return 1
        fi
    fi
    
    # Process based on type
    case "$type" in
        collections)
            log::info "Adding collections from $file"
            qdrant::content::process_collections "$file"
            ;;
        vectors)
            log::info "Adding vectors from $file"
            qdrant::content::process_vectors "$file"
            ;;
        operations)
            log::info "Executing operations from $file"
            qdrant::content::process_operations "$file"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
    
    # Store content metadata
    local metadata_file="$QDRANT_CONTENT_DIR/${name}.meta"
    cat > "$metadata_file" <<EOF
{
    "name": "$name",
    "type": "$type",
    "file": "$file",
    "added": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    log::success "Content added successfully: $name"
}

#######################################
# List content in Qdrant
# Arguments:
#   --type: Filter by content type
#   --format: Output format (text/json)
# Returns: 0 on success
#######################################
qdrant::content::list() {
    local type_filter=""
    local format="text"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type_filter="$2"
                shift 2
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local content_list=()
    
    # List stored content metadata
    if [[ -d "$QDRANT_CONTENT_DIR" ]]; then
        shopt -s nullglob
        for meta_file in "$QDRANT_CONTENT_DIR"/*.meta; do
            [[ -f "$meta_file" ]] || continue
            
            local metadata
            metadata=$(cat "$meta_file")
            
            if [[ -n "$type_filter" ]]; then
                local content_type
                content_type=$(echo "$metadata" | jq -r '.type')
                [[ "$content_type" == "$type_filter" ]] || continue
            fi
            
            content_list+=("$metadata")
        done
    fi
    
    # Also list actual collections from Qdrant
    local collections
    collections=$(qdrant::list_collections 2>/dev/null || echo "[]")
    
    if [[ "$format" == "json" ]]; then
        echo "{\"stored_content\": [$(IFS=,; echo "${content_list[*]}")], \"collections\": $collections}"
    else
        echo "Stored Content:"
        for content in "${content_list[@]}"; do
            echo "  - $(echo "$content" | jq -r '.name') ($(echo "$content" | jq -r '.type'))"
        done
        
        echo ""
        echo "Active Collections:"
        echo "$collections" | jq -r '.[] | "  - \(.)"'
    fi
}

#######################################
# Get specific content from Qdrant
# Arguments:
#   --name: Content name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::content::get() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Name required for getting content"
        echo "Usage: resource-qdrant content get --name <name>"
        return 1
    fi
    
    local metadata_file="$QDRANT_CONTENT_DIR/${name}.meta"
    if [[ ! -f "$metadata_file" ]]; then
        log::error "Content not found: $name"
        return 1
    fi
    
    cat "$metadata_file"
}

#######################################
# Remove content from Qdrant
# Arguments:
#   --name: Content name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::content::remove() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Name required for removing content"
        echo "Usage: resource-qdrant content remove --name <name>"
        return 1
    fi
    
    local metadata_file="$QDRANT_CONTENT_DIR/${name}.meta"
    if [[ ! -f "$metadata_file" ]]; then
        log::error "Content not found: $name"
        return 1
    fi
    
    # Get metadata
    local metadata
    metadata=$(cat "$metadata_file")
    local type
    type=$(echo "$metadata" | jq -r '.type')
    
    # For collections, try to delete from Qdrant
    if [[ "$type" == "collections" ]]; then
        # Try to delete the collection from Qdrant
        qdrant::delete_collection "$name" 2>/dev/null || true
    fi
    
    # Remove metadata
    rm -f "$metadata_file"
    
    log::success "Content removed: $name"
}

#######################################
# Execute content operations
# Arguments:
#   --file: Path to operations file
# Returns: 0 on success, 1 on failure
#######################################
qdrant::content::execute() {
    local file=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        log::error "File path required for executing operations"
        echo "Usage: resource-qdrant content execute --file <path>"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    qdrant::content::process_operations "$file"
}

#######################################
# Process collections from file
# Arguments:
#   $1: File path
# Returns: 0 on success, 1 on failure
#######################################
qdrant::content::process_collections() {
    local file="$1"
    local content
    content=$(cat "$file")
    
    # Check if it's the inject format
    if echo "$content" | jq -e '.collections' &>/dev/null; then
        # Process collections array
        local collections
        collections=$(echo "$content" | jq -c '.collections[]')
        
        while IFS= read -r collection; do
            local name
            name=$(echo "$collection" | jq -r '.name')
            local size
            size=$(echo "$collection" | jq -r '.size // 128')
            local distance
            distance=$(echo "$collection" | jq -r '.distance // "Cosine"')
            
            log::info "Creating collection: $name"
            qdrant::create_collection "$name" "$size" "$distance"
        done <<< "$collections"
        
        # Process vectors if present
        if echo "$content" | jq -e '.vectors' &>/dev/null; then
            qdrant::content::process_vectors_internal "$content"
        fi
    else
        log::error "Invalid collections format"
        return 1
    fi
}

#######################################
# Process vectors from file
# Arguments:
#   $1: File path
# Returns: 0 on success, 1 on failure
#######################################
qdrant::content::process_vectors() {
    local file="$1"
    local content
    content=$(cat "$file")
    
    qdrant::content::process_vectors_internal "$content"
}

#######################################
# Internal function to process vectors
# Arguments:
#   $1: Content JSON
# Returns: 0 on success, 1 on failure
#######################################
qdrant::content::process_vectors_internal() {
    local content="$1"
    
    if echo "$content" | jq -e '.vectors' &>/dev/null; then
        local vectors
        vectors=$(echo "$content" | jq -c '.vectors[]')
        
        while IFS= read -r vector_group; do
            local collection
            collection=$(echo "$vector_group" | jq -r '.collection')
            local points
            points=$(echo "$vector_group" | jq -c '.points')
            
            if [[ -n "$collection" ]] && [[ -n "$points" ]]; then
                log::info "Adding points to collection: $collection"
                
                # Use Qdrant API to add points
                local response
                response=$(curl -s -X PUT \
                    "${QDRANT_BASE_URL}/collections/${collection}/points" \
                    -H "Content-Type: application/json" \
                    -d "{\"points\": $points}")
                
                if echo "$response" | jq -e '.status == "ok"' &>/dev/null; then
                    log::success "Points added to $collection"
                else
                    log::error "Failed to add points to $collection: $response"
                fi
            fi
        done <<< "$vectors"
    elif echo "$content" | jq -e '.points' &>/dev/null; then
        # Direct points format
        local collection
        collection=$(echo "$content" | jq -r '.collection // "default"')
        local points
        points=$(echo "$content" | jq -c '.points')
        
        log::info "Adding points to collection: $collection"
        
        local response
        response=$(curl -s -X PUT \
            "${QDRANT_BASE_URL}/collections/${collection}/points" \
            -H "Content-Type: application/json" \
            -d "{\"points\": $points}")
        
        if echo "$response" | jq -e '.status == "ok"' &>/dev/null; then
            log::success "Points added to $collection"
        else
            log::error "Failed to add points to $collection: $response"
        fi
    else
        log::error "Invalid vectors format"
        return 1
    fi
}

#######################################
# Process operations from file
# Arguments:
#   $1: File path
# Returns: 0 on success, 1 on failure
#######################################
qdrant::content::process_operations() {
    local file="$1"
    local content
    content=$(cat "$file")
    
    if echo "$content" | jq -e '.operations' &>/dev/null; then
        local operations
        operations=$(echo "$content" | jq -c '.operations[]')
        
        while IFS= read -r operation; do
            local op_type
            op_type=$(echo "$operation" | jq -r '.type')
            
            case "$op_type" in
                create_collection)
                    local name size distance
                    name=$(echo "$operation" | jq -r '.name')
                    size=$(echo "$operation" | jq -r '.size // 128')
                    distance=$(echo "$operation" | jq -r '.distance // "Cosine"')
                    qdrant::create_collection "$name" "$size" "$distance"
                    ;;
                delete_collection)
                    local name
                    name=$(echo "$operation" | jq -r '.name')
                    qdrant::delete_collection "$name"
                    ;;
                upsert_points)
                    local collection points
                    collection=$(echo "$operation" | jq -r '.collection')
                    points=$(echo "$operation" | jq -c '.points')
                    
                    local response
                    response=$(curl -s -X PUT \
                        "${QDRANT_BASE_URL}/collections/${collection}/points" \
                        -H "Content-Type: application/json" \
                        -d "{\"points\": $points}")
                    
                    if echo "$response" | jq -e '.status == "ok"' &>/dev/null; then
                        log::success "Operation completed: $op_type"
                    else
                        log::error "Operation failed: $op_type - $response"
                    fi
                    ;;
                *)
                    log::warn "Unknown operation type: $op_type"
                    ;;
            esac
        done <<< "$operations"
    else
        log::error "Invalid operations format"
        return 1
    fi
}

# Backward compatibility wrapper for inject
qdrant::inject() {
    log::warn "The 'inject' command is deprecated. Please use 'content add' instead."
    
    # Try to get file from INJECTION_CONFIG or first argument
    local file=""
    if [[ -n "${INJECTION_CONFIG:-}" ]]; then
        # Save config to temp file
        file="/tmp/qdrant_inject_$$.json"
        echo "$INJECTION_CONFIG" > "$file"
        trap "rm -f $file" EXIT
    elif [[ -n "${1:-}" ]]; then
        file="$1"
    fi
    
    if [[ -n "$file" ]]; then
        qdrant::content::add --file "$file"
    else
        log::error "No file provided for injection"
        return 1
    fi
}