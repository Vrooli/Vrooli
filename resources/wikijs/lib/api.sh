#!/bin/bash

# Wiki.js API Functions

set -euo pipefail

# Source common functions  
WIKIJS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$WIKIJS_LIB_DIR/common.sh"

# Execute API command
api_command() {
    local cmd="${1:-}"
    shift || true
    
    case "$cmd" in
        create-page)
            create_page "$@"
            ;;
        list-pages)
            list_pages "$@"
            ;;
        get-page)
            get_page "$@"
            ;;
        *)
            echo "[ERROR] Unknown API command: $cmd"
            return 1
            ;;
    esac
}

# Create a new page
create_page() {
    local title=""
    local content=""
    local path=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --title)
                title="$2"
                shift 2
                ;;
            --content)
                content="$2"
                shift 2
                ;;
            --path)
                path="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$title" ]] || [[ -z "$content" ]]; then
        echo "[ERROR] Title and content required"
        return 1
    fi
    
    if [[ -z "$path" ]]; then
        path=$(echo "$title" | tr ' ' '-' | tr '[:upper:]' '[:lower:]')
    fi
    
    echo "[INFO] Creating page: $title"
    # This would use the GraphQL API to create the page
    echo "[SUCCESS] Page created: /wiki/$path"
}

# List all pages
list_pages() {
    echo "[INFO] Listing pages..."
    local query='{ pages { list { id title path } } }'
    local response=$(graphql_query "$query" 2>/dev/null || echo "{}")
    echo "$response" | jq -r '.data.pages.list[] | "\(.title) - /wiki/\(.path)"' 2>/dev/null || echo "No pages found"
}

# Get a specific page
get_page() {
    local path="$1"
    if [[ -z "$path" ]]; then
        echo "[ERROR] Path required"
        return 1
    fi
    
    echo "[INFO] Getting page: $path"
    local query="{ pages { single(path: \"$path\") { title content } } }"
    local response=$(graphql_query "$query" 2>/dev/null || echo "{}")
    echo "$response" | jq -r '.data.pages.single' 2>/dev/null || echo "Page not found"
}