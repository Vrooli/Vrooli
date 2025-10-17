#!/bin/bash

# LlamaIndex Injection Functions
# Handles injection of documents and indices into LlamaIndex

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LLAMAINDEX_INJECT_LIB_DIR="${APP_ROOT}/resources/llamaindex/lib"
LLAMAINDEX_INJECT_DIR="${APP_ROOT}/resources/llamaindex"
LLAMAINDEX_INJECT_ROOT_DIR="${APP_ROOT}/resources"
LLAMAINDEX_INJECT_SCRIPTS_DIR="${APP_ROOT}/scripts"

# Source dependencies
source "$LLAMAINDEX_INJECT_SCRIPTS_DIR/lib/utils/var.sh" || exit 1
source "$LLAMAINDEX_INJECT_SCRIPTS_DIR/lib/utils/log.sh" || exit 1
source "$LLAMAINDEX_INJECT_SCRIPTS_DIR/resources/port_registry.sh" || exit 1
source "$LLAMAINDEX_INJECT_LIB_DIR/core.sh" || exit 1

# Configuration
LLAMAINDEX_PORT="$(ports::get_resource_port "llamaindex" 8091)"
LLAMAINDEX_API_URL="http://localhost:$LLAMAINDEX_PORT"

#######################################
# Inject documents into LlamaIndex
# Arguments:
#   $1 - Source directory containing documents
#   $2 - Index name (optional, defaults to "default")
# Returns:
#   0 if successful, 1 otherwise
#######################################
llamaindex::inject() {
    local source_dir="${1:-}"
    local index_name="${2:-default}"
    
    if [[ -z "$source_dir" ]]; then
        log::error "Source directory is required"
        return 1
    fi
    
    if [[ ! -d "$source_dir" ]]; then
        log::error "Source directory does not exist: $source_dir"
        return 1
    fi
    
    # Check if LlamaIndex is running
    if ! llamaindex::is_running; then
        log::error "LlamaIndex is not running"
        return 1
    fi
    
    log::info "Injecting documents from $source_dir into index '$index_name'"
    
    # Find all text files in the directory
    local files=()
    while IFS= read -r -d '' file; do
        files+=("$file")
    done < <(find "$source_dir" -type f \( -name "*.txt" -o -name "*.md" -o -name "*.json" -o -name "*.py" -o -name "*.js" -o -name "*.ts" \) -print0)
    
    if [[ ${#files[@]} -eq 0 ]]; then
        log::warning "No documents found in $source_dir"
        return 0
    fi
    
    log::info "Found ${#files[@]} documents to index"
    
    # Prepare documents for indexing
    local documents="[]"
    for file in "${files[@]}"; do
        local content
        content=$(cat "$file" | jq -Rs . 2>/dev/null) || continue
        local metadata=$(jq -n --arg path "$file" --arg name "$(basename "$file")" \
            '{source: $path, filename: $name}')
        
        documents=$(echo "$documents" | jq ". += [{text: $content, metadata: $metadata}]")
    done
    
    # Create index with documents
    local response
    response=$(curl -s -X POST "$LLAMAINDEX_API_URL/index/create" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"$index_name\", \"documents\": $documents}" 2>/dev/null)
    
    if echo "$response" | grep -q "success\|created"; then
        log::success "Successfully indexed ${#files[@]} documents into '$index_name'"
        return 0
    else
        log::error "Failed to create index: $response"
        return 1
    fi
}

#######################################
# List all indices
# Returns:
#   0 if successful, 1 otherwise
#######################################
llamaindex::list_indices() {
    if ! llamaindex::is_running; then
        log::error "LlamaIndex is not running"
        return 1
    fi
    
    local response
    response=$(curl -s "$LLAMAINDEX_API_URL/indices" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq -r '.indices[]' 2>/dev/null || echo "$response"
        return 0
    else
        log::error "Failed to list indices"
        return 1
    fi
}

#######################################
# Query an index
# Arguments:
#   $1 - Query text
#   $2 - Index name (optional, defaults to "default")
# Returns:
#   0 if successful, 1 otherwise
#######################################
llamaindex::query() {
    local query="${1:-}"
    local index_name="${2:-default}"
    
    if [[ -z "$query" ]]; then
        log::error "Query text is required"
        return 1
    fi
    
    if ! llamaindex::is_running; then
        log::error "LlamaIndex is not running"
        return 1
    fi
    
    local response
    response=$(curl -s -X POST "$LLAMAINDEX_API_URL/query" \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"$query\", \"index_name\": \"$index_name\"}" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq . 2>/dev/null || echo "$response"
        return 0
    else
        log::error "Failed to query index"
        return 1
    fi
}

# Export functions
export -f llamaindex::inject
export -f llamaindex::list_indices
export -f llamaindex::query