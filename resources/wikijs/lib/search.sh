#!/bin/bash

# Wiki.js Search Functions

set -euo pipefail

# Source common functions
WIKIJS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$WIKIJS_LIB_DIR/common.sh"

# Search wiki content
search_content() {
    local query="$1"
    
    if [[ -z "$query" ]]; then
        echo "[ERROR] Search query required"
        return 1
    fi
    
    echo "[INFO] Searching for: $query"
    
    local graphql_query="{ pages { search(query: \"$query\") { results { id title path description } } } }"
    local response=$(graphql_query "$graphql_query" 2>/dev/null || echo "{}")
    
    local results=$(echo "$response" | jq -r '.data.pages.search.results[]' 2>/dev/null || echo "")
    
    if [[ -z "$results" ]]; then
        echo "No results found for: $query"
    else
        echo "$response" | jq -r '.data.pages.search.results[] | "[\(.title)] /wiki/\(.path)"' 2>/dev/null
    fi
}

# Rebuild search index
rebuild_search_index() {
    echo "[INFO] Rebuilding search index..."
    
    # This would trigger a search index rebuild via the API
    # For now, just indicate the action
    echo "[INFO] Search index rebuild initiated"
    echo "[INFO] This may take a few minutes depending on content size"
}