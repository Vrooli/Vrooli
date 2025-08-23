#!/bin/bash

# Wiki.js Content Injection Functions

set -euo pipefail

# Source common functions
WIKIJS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$WIKIJS_LIB_DIR/common.sh"

# Inject content into Wiki.js
inject_content() {
    local file=""
    local title=""
    local path=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --title)
                title="$2"
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
    
    if [[ -z "$file" ]]; then
        echo "[ERROR] File required: --file <path>"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "[ERROR] File not found: $file"
        return 1
    fi
    
    if [[ -z "$title" ]]; then
        title=$(basename "$file" .md)
    fi
    
    if [[ -z "$path" ]]; then
        path=$(echo "$title" | tr ' ' '-' | tr '[:upper:]' '[:lower:]')
    fi
    
    echo "[INFO] Injecting content: $title"
    
    # Copy file to content directory
    cp "$file" "$WIKIJS_DATA_DIR/content/${path}.md"
    
    echo "[SUCCESS] Content injected: $title"
    echo "[INFO] Path: /wiki/${path}"
}