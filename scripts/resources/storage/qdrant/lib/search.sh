#!/usr/bin/env bash
# Qdrant Search Management - Semantic search functionality
# Provides text-based and embedding-based search capabilities

set -euo pipefail

# Get directory of this script
QDRANT_SEARCH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "${QDRANT_SEARCH_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"

# Source configuration and dependencies
# shellcheck disable=SC1091
source "${QDRANT_SEARCH_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${QDRANT_SEARCH_DIR}/core.sh"
# shellcheck disable=SC1091
source "${QDRANT_SEARCH_DIR}/collections.sh"
# shellcheck disable=SC1091
source "${QDRANT_SEARCH_DIR}/models.sh"
# shellcheck disable=SC1091
source "${QDRANT_SEARCH_DIR}/embeddings.sh"

# Search configuration
QDRANT_SEARCH_DEFAULT_LIMIT="${QDRANT_SEARCH_DEFAULT_LIMIT:-10}"
QDRANT_SEARCH_DEFAULT_SCORE_THRESHOLD="${QDRANT_SEARCH_DEFAULT_SCORE_THRESHOLD:-0.7}"

#######################################
# Parse search arguments
# Arguments: All CLI arguments passed to search command
# Sets global variables: PARSED_TEXT, PARSED_COLLECTION, PARSED_LIMIT, PARSED_MODEL, PARSED_EMBEDDING
# Returns: 0 on success, 1 on failure
#######################################
qdrant::search::parse_args() {
    local text=""
    local collection=""
    local limit="10"
    local model=""
    local embedding=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --text|-t)
                text="$2"
                shift 2
                ;;
            --collection|-c)
                collection="$2"
                shift 2
                ;;
            --embedding|-e)
                embedding="$2"
                shift 2
                ;;
            --limit|-l)
                limit="$2"
                shift 2
                ;;
            --model|-m)
                model="$2"
                shift 2
                ;;
            *)
                if [[ -z "$text" ]]; then
                    text="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$text" ]] && [[ -z "$embedding" ]]; then
        log::error "Search query required (text or embedding)"
        return 1
    fi
    
    # Set global variables for caller
    PARSED_TEXT="$text"
    PARSED_COLLECTION="$collection"
    PARSED_LIMIT="$limit"
    PARSED_MODEL="$model"
    PARSED_EMBEDDING="$embedding"
    
    return 0
}

#######################################
# Execute search with parsed arguments
# Uses global variables set by parse_args
# Returns: 0 on success, 1 on failure
#######################################
qdrant::search::execute_parsed() {
    if [[ -z "${PARSED_TEXT:-}" ]] && [[ -z "${PARSED_EMBEDDING:-}" ]]; then
        log::error "No parsed arguments available. Call parse_args first."
        return 1
    fi
    
    if [[ -n "${PARSED_EMBEDDING:-}" ]]; then
        # Direct embedding search
        if [[ -z "${PARSED_COLLECTION:-}" ]]; then
            log::error "Collection required for embedding search"
            return 1
        fi
        if command -v qdrant::search::by_vector &>/dev/null; then
            qdrant::search::by_vector "$PARSED_EMBEDDING" "$PARSED_COLLECTION" "$PARSED_LIMIT"
        else
            log::error "Vector search not available"
            return 1
        fi
    else
        # Semantic search with text
        qdrant::search::semantic "$PARSED_TEXT" "$PARSED_COLLECTION" "$PARSED_LIMIT" "$PARSED_MODEL"
    fi
}

#######################################
# Perform semantic search with text query
# Arguments:
#   $1 - Query text
#   $2 - Collection name (optional, auto-detect if single collection)
#   $3 - Limit (optional, default: 10)
#   $4 - Model name (optional, auto-detect based on collection)
# Outputs: Search results in JSON format
# Returns: 0 on success, 1 on failure
#######################################
qdrant::search::semantic() {
    local query_text="$1"
    local collection="${2:-}"
    local limit="${3:-$QDRANT_SEARCH_DEFAULT_LIMIT}"
    local model="${4:-}"
    
    if [[ -z "$query_text" ]]; then
        log::error "Query text is required"
        return 1
    fi
    
    # Auto-detect collection if not specified
    if [[ -z "$collection" ]]; then
        collection=$(qdrant::search::auto_detect_collection)
        if [[ -z "$collection" ]]; then
            log::error "No collection specified and auto-detection failed"
            log::info "Available collections:"
            qdrant::collections::list_simple
            return 1
        fi
        log::info "Auto-selected collection: $collection"
    fi
    
    # Verify collection exists
    if ! qdrant::collections::exists "$collection"; then
        log::error "Collection not found: $collection"
        log::info "Available collections:"
        qdrant::collections::list_simple
        return 1
    fi
    
    # Get collection dimensions
    local collection_dims
    collection_dims=$(qdrant::collections::get_dimensions "$collection")
    
    if [[ "$collection_dims" == "unknown" ]] || [[ -z "$collection_dims" ]]; then
        log::error "Cannot determine dimensions for collection: $collection"
        return 1
    fi
    
    log::debug "Collection '$collection' has $collection_dims dimensions"
    
    # Auto-select model if not specified
    if [[ -z "$model" ]]; then
        model=$(qdrant::models::auto_select "$collection_dims")
        if [[ -z "$model" ]]; then
            log::error "No compatible embedding model found for $collection_dims dimensions"
            log::info "Available models:"
            qdrant::models::list_compatible "$collection_dims"
            return 1
        fi
        log::debug "Using model: $model"
    else
        # Validate model compatibility
        if ! qdrant::models::validate_compatibility "$model" "$collection_dims"; then
            log::error "Model '$model' is not compatible with collection '$collection' ($collection_dims dimensions)"
            log::info "Compatible models:"
            qdrant::models::list_compatible "$collection_dims"
            return 1
        fi
    fi
    
    # Generate embedding for query
    log::info "Generating embedding for query..."
    local query_embedding
    query_embedding=$(qdrant::embeddings::generate "$query_text" "$model")
    
    if [[ -z "$query_embedding" ]] || [[ "$query_embedding" == "null" ]]; then
        log::error "Failed to generate embedding for query"
        return 1
    fi
    
    # Perform vector search
    qdrant::search::by_vector "$query_embedding" "$collection" "$limit"
}

#######################################
# Search by pre-computed embedding vector
# Arguments:
#   $1 - Embedding vector (JSON array)
#   $2 - Collection name
#   $3 - Limit (optional, default: 10)
#   $4 - Score threshold (optional)
# Outputs: Search results in JSON format
# Returns: 0 on success, 1 on failure
#######################################
qdrant::search::by_vector() {
    local embedding="$1"
    local collection="$2"
    local limit="${3:-$QDRANT_SEARCH_DEFAULT_LIMIT}"
    local score_threshold="${4:-}"
    
    if [[ -z "$embedding" ]] || [[ -z "$collection" ]]; then
        log::error "Embedding and collection are required"
        return 1
    fi
    
    # Validate embedding format
    if ! echo "$embedding" | jq -e 'type == "array"' >/dev/null 2>&1; then
        log::error "Invalid embedding format (must be JSON array)"
        return 1
    fi
    
    # Validate embedding dimensions
    if ! qdrant::embeddings::validate_dimensions "$embedding" "$collection"; then
        return 1
    fi
    
    # Build search request
    local search_request
    search_request=$(jq -n \
        --argjson vector "$embedding" \
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
    
    log::info "Searching in collection '$collection'..."
    
    # Perform search via Qdrant API
    local response
    response=$(qdrant::api::request "POST" "/collections/${collection}/points/search" "$search_request" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        log::error "Search request failed"
        return 1
    fi
    
    # Check for errors in response
    local status
    status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
    
    if [[ "$status" != "ok" ]]; then
        local error_msg
        error_msg=$(echo "$response" | jq -r '.status.error // "Unknown error"' 2>/dev/null)
        log::error "Search failed: $error_msg"
        return 1
    fi
    
    # Format and display results
    qdrant::search::format_results "$response"
}

#######################################
# Format search results for display
# Arguments:
#   $1 - Raw search response from Qdrant
# Outputs: Formatted search results
# Returns: 0 on success
#######################################
qdrant::search::format_results() {
    local response="$1"
    
    # Extract results
    local results
    results=$(echo "$response" | jq '.result[]' 2>/dev/null)
    
    if [[ -z "$results" ]]; then
        log::info "No results found"
        echo "[]"
        return 0
    fi
    
    echo "=== Search Results ==="
    echo
    
    local count=0
    while IFS= read -r result; do
        ((count++))
        
        local score
        local id
        local payload
        
        score=$(echo "$result" | jq -r '.score // 0')
        id=$(echo "$result" | jq -r '.id // "unknown"')
        payload=$(echo "$result" | jq '.payload // {}')
        
        echo "📍 Result #$count"
        echo "   ID: $id"
        echo "   Score: $score"
        
        # Display payload if available
        if [[ "$payload" != "{}" ]]; then
            echo "   Data:"
            echo "$payload" | jq -r 'to_entries[] | "     • \(.key): \(.value)"' 2>/dev/null || echo "     $payload"
        fi
        
        echo
    done <<< "$(echo "$response" | jq -c '.result[]' 2>/dev/null)"
    
    echo "Total results: $count"
    
    # Return results as JSON for programmatic use
    echo "$response" | jq '.result'
}

#######################################
# Auto-detect collection for search
# Outputs: Collection name or empty string
# Returns: 0 if found, 1 if not
#######################################
qdrant::search::auto_detect_collection() {
    local collections
    collections=$(qdrant::collections::list_simple 2>/dev/null)
    
    if [[ -z "$collections" ]]; then
        log::debug "No collections found"
        return 1
    fi
    
    local collection_count
    collection_count=$(echo "$collections" | wc -l)
    
    if [[ "$collection_count" -eq 1 ]]; then
        # Single collection - use it
        echo "$collections"
        return 0
    else
        # Multiple collections - try to find a default
        # Prefer collections with standard names
        for default_name in "general_embeddings" "embeddings" "vectors" "default"; do
            if echo "$collections" | grep -q "^${default_name}$"; then
                echo "$default_name"
                return 0
            fi
        done
        
        # No default found
        log::debug "Multiple collections found, cannot auto-select"
        return 1
    fi
}

#######################################
# Search with advanced options
# Arguments:
#   JSON configuration object with search parameters
# Outputs: Search results
# Returns: 0 on success, 1 on failure
#######################################
qdrant::search::advanced() {
    local config="$1"
    
    if [[ -z "$config" ]]; then
        log::error "Search configuration required"
        cat <<EOF
Usage: qdrant::search::advanced '<json_config>'

Example configuration:
{
  "query": "machine learning",      // Text query (or "embedding" for direct vector)
  "collection": "my_collection",    // Target collection
  "limit": 10,                      // Number of results
  "score_threshold": 0.7,           // Minimum similarity score
  "filter": {                       // Optional payload filters
    "must": [
      {"key": "category", "match": {"value": "technology"}}
    ]
  },
  "model": "nomic-embed-text"       // Embedding model (auto-detect if not specified)
}
EOF
        return 1
    fi
    
    # Parse configuration
    local query
    local embedding
    local collection
    local limit
    local score_threshold
    local filter
    local model
    
    query=$(echo "$config" | jq -r '.query // ""')
    embedding=$(echo "$config" | jq '.embedding // null')
    collection=$(echo "$config" | jq -r '.collection // ""')
    limit=$(echo "$config" | jq -r '.limit // 10')
    score_threshold=$(echo "$config" | jq '.score_threshold // null')
    filter=$(echo "$config" | jq '.filter // null')
    model=$(echo "$config" | jq -r '.model // ""')
    
    # Validate required fields
    if [[ -z "$query" ]] && [[ "$embedding" == "null" ]]; then
        log::error "Either 'query' (text) or 'embedding' (vector) is required"
        return 1
    fi
    
    if [[ -z "$collection" ]]; then
        collection=$(qdrant::search::auto_detect_collection)
        if [[ -z "$collection" ]]; then
            log::error "Collection must be specified"
            return 1
        fi
    fi
    
    # Get or generate embedding
    local search_vector
    if [[ "$embedding" != "null" ]]; then
        search_vector="$embedding"
    else
        # Generate embedding from query text
        if [[ -z "$model" ]]; then
            local collection_dims
            collection_dims=$(qdrant::collections::get_dimensions "$collection")
            model=$(qdrant::models::auto_select "$collection_dims")
        fi
        
        search_vector=$(qdrant::embeddings::generate "$query" "$model")
        if [[ -z "$search_vector" ]]; then
            log::error "Failed to generate embedding"
            return 1
        fi
    fi
    
    # Build search request
    local search_request
    search_request=$(jq -n \
        --argjson vector "$search_vector" \
        --argjson limit "$limit" \
        '{
            vector: $vector,
            limit: $limit,
            with_payload: true,
            with_vector: false
        }')
    
    # Add optional parameters
    if [[ "$score_threshold" != "null" ]]; then
        search_request=$(echo "$search_request" | jq --argjson threshold "$score_threshold" '.score_threshold = $threshold')
    fi
    
    if [[ "$filter" != "null" ]]; then
        search_request=$(echo "$search_request" | jq --argjson filter "$filter" '.filter = $filter')
    fi
    
    # Perform search
    local response
    response=$(qdrant::api::request "POST" "/collections/${collection}/points/search" "$search_request" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        log::error "Search request failed"
        return 1
    fi
    
    # Format results
    qdrant::search::format_results "$response"
}

#######################################
# Insert vector with metadata and search
# Arguments:
#   $1 - Text to embed and store
#   $2 - Collection name
#   $3 - Metadata (JSON object, optional)
#   $4 - Model name (optional)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::search::insert_and_search() {
    local text="$1"
    local collection="$2"
    local metadata="${3:-{}}"
    local model="${4:-}"
    
    if [[ -z "$text" ]] || [[ -z "$collection" ]]; then
        log::error "Text and collection are required"
        return 1
    fi
    
    # Ensure collection exists
    if ! qdrant::collections::exists "$collection"; then
        log::warn "Collection '$collection' does not exist"
        
        # Try to create it with auto-detected dimensions
        if [[ -n "$model" ]]; then
            local dims
            dims=$(qdrant::models::get_model_dimensions "$model")
            if [[ "$dims" != "unknown" ]]; then
                log::info "Creating collection with $dims dimensions..."
                if ! qdrant::collections::create "$collection" "$dims" "Cosine"; then
                    log::error "Failed to create collection"
                    return 1
                fi
            fi
        else
            log::error "Please create the collection first or specify a model"
            return 1
        fi
    fi
    
    # Generate embedding
    local embedding
    embedding=$(qdrant::embeddings::generate "$text" "$model")
    
    if [[ -z "$embedding" ]]; then
        log::error "Failed to generate embedding"
        return 1
    fi
    
    # Generate unique ID
    local point_id
    point_id=$(uuidgen 2>/dev/null || echo "$(date +%s)-$$-$RANDOM")
    
    # Build point data
    local point
    point=$(jq -n \
        --arg id "$point_id" \
        --argjson vector "$embedding" \
        --argjson payload "$metadata" \
        '{
            points: [{
                id: $id,
                vector: $vector,
                payload: $payload
            }]
        }')
    
    # Add text to payload
    point=$(echo "$point" | jq --arg text "$text" '.points[0].payload.text = $text')
    
    # Insert into Qdrant
    log::info "Inserting vector into collection '$collection'..."
    local response
    response=$(qdrant::api::request "PUT" "/collections/${collection}/points" "$point" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        log::error "Failed to insert vector"
        return 1
    fi
    
    local status
    status=$(echo "$response" | jq -r '.status // "unknown"')
    
    if [[ "$status" == "ok" ]]; then
        log::success "Vector inserted successfully (ID: $point_id)"
        
        # Perform immediate search to verify
        log::info "Verifying insertion with search..."
        sleep 0.5  # Brief delay for indexing
        qdrant::search::by_vector "$embedding" "$collection" 1
    else
        log::error "Failed to insert vector"
        return 1
    fi
}

#######################################
# Get collection dimensions helper
# Arguments:
#   $1 - Collection name
# Outputs: Dimension count or "unknown"
# Returns: 0 on success
#######################################
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