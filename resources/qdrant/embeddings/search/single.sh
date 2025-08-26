#!/usr/bin/env bash
# Single App Search for Qdrant Embeddings
# Enables semantic search within a single app's collections

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Define paths from APP_ROOT
QDRANT_DIR="${APP_ROOT}/resources/qdrant"
EMBEDDINGS_DIR="${QDRANT_DIR}/embeddings"
SEARCH_DIR="${EMBEDDINGS_DIR}/search"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source Qdrant libraries
source "${QDRANT_DIR}/lib/core.sh"

# Default settings
DEFAULT_MODEL="mxbai-embed-large"
DEFAULT_LIMIT=10
DEFAULT_MIN_SCORE=0.5
QDRANT_URL="${QDRANT_URL:-http://localhost:6333}"

#######################################
# Search a single app's collections
# Arguments:
#   $1 - Query text
#   $2 - App ID
#   $3 - Type filter (optional: all|workflows|scenarios|knowledge|code|resources)
#   $4 - Limit (optional, default: 10)
#   $5 - Min score (optional, default: 0.5)
# Returns: JSON results
#######################################
qdrant::search::single_app() {
    local query="$1"
    local app_id="$2"
    local type="${3:-all}"
    local limit="${4:-$DEFAULT_LIMIT}"
    local min_score="${5:-$DEFAULT_MIN_SCORE}"
    
    if [[ -z "$query" ]] || [[ -z "$app_id" ]]; then
        log::error "Query and app ID are required"
        return 1
    fi
    
    log::debug "Searching app '$app_id' for: $query"
    
    # Generate query embedding
    local query_embedding=$(qdrant::search::generate_embedding "$query")
    if [[ -z "$query_embedding" ]]; then
        log::error "Failed to generate query embedding"
        return 1
    fi
    
    # Determine which collections to search
    local collections=()
    case "$type" in
        all)
            collections=(
                "${app_id}-workflows"
                "${app_id}-scenarios"
                "${app_id}-knowledge"
                "${app_id}-code"
                "${app_id}-resources"
            )
            ;;
        workflows)
            collections=("${app_id}-workflows")
            ;;
        scenarios)
            collections=("${app_id}-scenarios")
            ;;
        knowledge|docs|documentation)
            collections=("${app_id}-knowledge")
            ;;
        code)
            collections=("${app_id}-code")
            ;;
        resources)
            collections=("${app_id}-resources")
            ;;
        *)
            log::error "Invalid type: $type"
            return 1
            ;;
    esac
    
    # Search each collection and aggregate results
    local all_results="[]"
    local search_start=$(date +%s%3N)
    
    for collection in "${collections[@]}"; do
        # Check if collection exists
        if ! qdrant::search::collection_exists "$collection"; then
            log::debug "Collection $collection does not exist, skipping"
            continue
        fi
        
        # Perform vector search
        local results=$(qdrant::search::vector_search \
            "$collection" \
            "$query_embedding" \
            "$limit" \
            "$min_score")
        
        if [[ -n "$results" ]] && [[ "$results" != "[]" ]]; then
            # Add app_id and collection to each result
            results=$(echo "$results" | jq --arg app "$app_id" --arg coll "$collection" \
                '[.[] | . + {app_id: $app, collection: $coll}]')
            
            # Merge with all results
            all_results=$(echo "$all_results $results" | jq -s 'add')
        fi
    done
    
    # Sort by score and limit
    all_results=$(echo "$all_results" | jq --argjson limit "$limit" \
        'sort_by(-.score) | .[0:$limit]')
    
    # Calculate search time
    local search_end=$(date +%s%3N)
    local search_time=$((search_end - search_start))
    
    # Build final response
    jq -n \
        --arg query "$query" \
        --arg app_id "$app_id" \
        --arg type "$type" \
        --argjson results "$all_results" \
        --arg search_time "$search_time" \
        --arg result_count "$(echo "$all_results" | jq 'length')" \
        '{
            query: $query,
            app_id: $app_id,
            type: $type,
            results: $results,
            search_time_ms: ($search_time | tonumber),
            result_count: ($result_count | tonumber),
            timestamp: now | strftime("%Y-%m-%dT%H:%M:%SZ")
        }'
}

#######################################
# Perform vector search on a collection
# Arguments:
#   $1 - Collection name
#   $2 - Query embedding (JSON array)
#   $3 - Limit
#   $4 - Min score
# Returns: JSON array of results
#######################################
qdrant::search::vector_search() {
    local collection="$1"
    local embedding="$2"
    local limit="$3"
    local min_score="$4"
    
    # Build search request
    local request=$(jq -n \
        --argjson vector "$embedding" \
        --argjson limit "$limit" \
        --argjson score "$min_score" \
        '{
            vector: $vector,
            limit: $limit,
            score_threshold: $score,
            with_payload: true,
            with_vector: false
        }')
    
    # Perform search via Qdrant API
    local response=$(curl -s -X POST \
        "${QDRANT_URL}/collections/${collection}/points/search" \
        -H "Content-Type: application/json" \
        -d "$request" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        echo "[]"
        return
    fi
    
    # Extract and format results
    echo "$response" | jq -r '.result // []' | jq '[.[] | {
        id: .id,
        score: .score,
        content: .payload.content,
        type: .payload.type,
        metadata: .payload
    }]'
}

#######################################
# Search with advanced filters
# Arguments:
#   $1 - Query text
#   $2 - App ID
#   $3 - JSON filter object
# Returns: Filtered JSON results
#######################################
qdrant::search::filtered() {
    local query="$1"
    local app_id="$2"
    local filters="${3:-{}}"
    
    # Extract filter parameters
    local type=$(echo "$filters" | jq -r '.type // "all"')
    local limit=$(echo "$filters" | jq -r '.limit // 10')
    local min_score=$(echo "$filters" | jq -r '.min_score // 0.5')
    local after_date=$(echo "$filters" | jq -r '.after_date // null')
    local before_date=$(echo "$filters" | jq -r '.before_date // null')
    local tags=$(echo "$filters" | jq -r '.tags // []')
    
    # Perform base search
    local results=$(qdrant::search::single_app "$query" "$app_id" "$type" "$limit" "$min_score")
    
    # Apply additional filters
    if [[ "$after_date" != "null" ]] || [[ "$before_date" != "null" ]]; then
        results=$(echo "$results" | jq \
            --arg after "$after_date" \
            --arg before "$before_date" \
            '.results |= map(select(
                (if $after != "null" then .metadata.modified >= $after else true end) and
                (if $before != "null" then .metadata.modified <= $before else true end)
            ))')
    fi
    
    # Filter by tags if provided
    if [[ "$(echo "$tags" | jq 'length')" -gt 0 ]]; then
        results=$(echo "$results" | jq \
            --argjson tags "$tags" \
            '.results |= map(select(
                .metadata.tags as $mtags |
                any($tags[]; . as $tag | $mtags | index($tag))
            ))')
    fi
    
    echo "$results"
}

#######################################
# Generate embedding for query text
# Arguments:
#   $1 - Query text
#   $2 - Model name (optional)
# Returns: Embedding vector as JSON array
#######################################
qdrant::search::generate_embedding() {
    local text="$1"
    local model="${2:-$DEFAULT_MODEL}"
    
    # Generate embedding using Ollama
    local embedding=$(ollama embeddings "$model" "$text" 2>/dev/null | jq -c '.embedding // empty')
    
    if [[ -z "$embedding" ]]; then
        # Fallback to alternative method
        embedding=$(curl -s -X POST http://localhost:11434/api/embeddings \
            -d "{\"model\": \"$model\", \"prompt\": \"$text\"}" 2>/dev/null | \
            jq -c '.embedding // empty')
    fi
    
    echo "$embedding"
}

#######################################
# Check if collection exists
# Arguments:
#   $1 - Collection name
# Returns: 0 if exists, 1 otherwise
#######################################
qdrant::search::collection_exists() {
    local collection="$1"
    
    local response=$(curl -s -X GET \
        "${QDRANT_URL}/collections/${collection}" 2>/dev/null)
    
    if echo "$response" | jq -e '.result' >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

#######################################
# Search and explain results
# Arguments:
#   $1 - Query text
#   $2 - App ID
#   $3 - Type filter (optional)
# Returns: Human-readable results
#######################################
qdrant::search::explain() {
    local query="$1"
    local app_id="$2"
    local type="${3:-all}"
    
    echo "=== Search Results ==="
    echo "Query: \"$query\""
    echo "App: $app_id"
    echo "Type: $type"
    echo
    
    local results=$(qdrant::search::single_app "$query" "$app_id" "$type")
    local count=$(echo "$results" | jq -r '.result_count')
    
    if [[ "$count" -eq 0 ]]; then
        echo "No results found."
        return 0
    fi
    
    echo "Found $count result(s):"
    echo
    
    echo "$results" | jq -r '.results[] | 
        "Score: \(.score | tostring[0:4])\n" +
        "Type: \(.type)\n" +
        "Content: \(.content | split("\n")[0:3] | join("\n") | .[0:200])\n" +
        (if .metadata.source_file then "File: \(.metadata.source_file)\n" else "" end) +
        "---"'
}

#######################################
# Get similar items (more like this)
# Arguments:
#   $1 - Item ID
#   $2 - Collection name
#   $3 - Limit (optional, default: 5)
# Returns: Similar items
#######################################
qdrant::search::similar() {
    local item_id="$1"
    local collection="$2"
    local limit="${3:-5}"
    
    # Get the item's vector
    local response=$(curl -s -X GET \
        "${QDRANT_URL}/collections/${collection}/points/${item_id}" 2>/dev/null)
    
    local vector=$(echo "$response" | jq -c '.result.vector // empty')
    
    if [[ -z "$vector" ]]; then
        log::error "Could not find item: $item_id"
        return 1
    fi
    
    # Search for similar items (excluding the original)
    local results=$(qdrant::search::vector_search "$collection" "$vector" "$((limit + 1))" "0.0")
    
    # Remove the original item from results
    echo "$results" | jq --arg id "$item_id" '[.[] | select(.id != $id)][0:'"$limit"']'
}

#######################################
# Batch search multiple queries
# Arguments:
#   $1 - JSON array of queries
#   $2 - App ID
#   $3 - Type filter (optional)
# Returns: Array of results
#######################################
qdrant::search::batch() {
    local queries="$1"
    local app_id="$2"
    local type="${3:-all}"
    
    if [[ -z "$queries" ]] || [[ "$queries" == "[]" ]]; then
        echo "[]"
        return 0
    fi
    
    local results="[]"
    
    echo "$queries" | jq -r '.[]' | while read -r query; do
        local result=$(qdrant::search::single_app "$query" "$app_id" "$type")
        results=$(echo "$results" | jq --argjson r "$result" '. + [$r]')
    done
    
    echo "$results"
}

#######################################
# Analyze search patterns
# Arguments:
#   $1 - App ID
# Returns: Search analytics
#######################################
qdrant::search::analytics() {
    local app_id="$1"
    
    echo "=== Search Analytics for $app_id ==="
    echo
    
    # Count items in each collection
    local collections=(
        "${app_id}-workflows"
        "${app_id}-scenarios"
        "${app_id}-knowledge"
        "${app_id}-code"
        "${app_id}-resources"
    )
    
    echo "Collection Sizes:"
    for collection in "${collections[@]}"; do
        if qdrant::search::collection_exists "$collection"; then
            local count=$(curl -s -X GET \
                "${QDRANT_URL}/collections/${collection}" 2>/dev/null | \
                jq -r '.result.points_count // 0')
            echo "  • ${collection}: $count items"
        else
            echo "  • ${collection}: not created"
        fi
    done
    echo
    
    # Sample search to test connectivity
    echo "Search Capability: "
    local test_result=$(qdrant::search::single_app "test" "$app_id" "all" "1" "0.0" 2>/dev/null)
    if [[ -n "$test_result" ]]; then
        echo "  ✅ Search system operational"
    else
        echo "  ❌ Search system not responding"
    fi
}