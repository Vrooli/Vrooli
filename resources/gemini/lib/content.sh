#!/bin/bash
# Gemini content management functionality

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
GEMINI_CONTENT_DIR="${APP_ROOT}/resources/gemini/lib"

# Source dependencies
source "${GEMINI_CONTENT_DIR}/core.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Content storage directory
GEMINI_CONTENT_STORAGE="${VROOLI_DATA_DIR:-${HOME}/Vrooli/data}/gemini/content"

# Initialize content storage
gemini::content::init() {
    mkdir -p "${GEMINI_CONTENT_STORAGE}/prompts"
    mkdir -p "${GEMINI_CONTENT_STORAGE}/templates"
    mkdir -p "${GEMINI_CONTENT_STORAGE}/functions"
}

# Add content
gemini::content::add() {
    local name="${1:-}"
    local type="${2:-prompt}"
    local file="${3:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Content name required"
        return 1
    fi
    
    if [[ -z "$file" ]] || [[ "$file" == "-" ]]; then
        # Read from stdin
        local content
        content=$(cat)
    elif [[ -f "$file" ]]; then
        local content
        content=$(cat "$file")
    else
        log::error "File not found: $file"
        return 1
    fi
    
    # Initialize storage
    gemini::content::init
    
    # Determine storage path based on type
    local storage_path
    case "$type" in
        prompt|prompts)
            storage_path="${GEMINI_CONTENT_STORAGE}/prompts/${name}.txt"
            ;;
        template|templates)
            storage_path="${GEMINI_CONTENT_STORAGE}/templates/${name}.json"
            ;;
        function|functions)
            storage_path="${GEMINI_CONTENT_STORAGE}/functions/${name}.json"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
    
    # Save content
    echo "$content" > "$storage_path"
    
    log::success "Added $type: $name"
    return 0
}

# List content
gemini::content::list() {
    local type="${1:-all}"
    local format="${2:-text}"
    
    # Initialize storage
    gemini::content::init
    
    local results=()
    
    if [[ "$type" == "all" ]] || [[ "$type" == "prompt" ]] || [[ "$type" == "prompts" ]]; then
        shopt -s nullglob
        for file in "${GEMINI_CONTENT_STORAGE}/prompts"/*.txt; do
            [[ -f "$file" ]] && results+=("prompt:$(basename "$file" .txt)")
        done
        shopt -u nullglob
    fi
    
    if [[ "$type" == "all" ]] || [[ "$type" == "template" ]] || [[ "$type" == "templates" ]]; then
        shopt -s nullglob
        for file in "${GEMINI_CONTENT_STORAGE}/templates"/*.json; do
            [[ -f "$file" ]] && results+=("template:$(basename "$file" .json)")
        done
        shopt -u nullglob
    fi
    
    if [[ "$type" == "all" ]] || [[ "$type" == "function" ]] || [[ "$type" == "functions" ]]; then
        shopt -s nullglob
        for file in "${GEMINI_CONTENT_STORAGE}/functions"/*.json; do
            [[ -f "$file" ]] && results+=("function:$(basename "$file" .json)")
        done
        shopt -u nullglob
    fi
    
    if [[ ${#results[@]} -eq 0 ]]; then
        if [[ "$format" == "json" ]]; then
            echo '{"content":[]}'
        else
            echo "No content found"
        fi
        return 0
    fi
    
    if [[ "$format" == "json" ]]; then
        printf '{"content":['
        local first=true
        for item in "${results[@]}"; do
            IFS=: read -r item_type item_name <<< "$item"
            if [[ "$first" != true ]]; then
                printf ','
            fi
            printf '{"type":"%s","name":"%s"}' "$item_type" "$item_name"
            first=false
        done
        printf ']}\n'
    else
        for item in "${results[@]}"; do
            echo "$item"
        done
    fi
}

# Get content
gemini::content::get() {
    local name="${1:-}"
    local type="${2:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Content name required"
        return 1
    fi
    
    # Initialize storage
    gemini::content::init
    
    # Try to find the content
    local file_path=""
    
    if [[ -z "$type" ]] || [[ "$type" == "prompt" ]] || [[ "$type" == "prompts" ]]; then
        if [[ -f "${GEMINI_CONTENT_STORAGE}/prompts/${name}.txt" ]]; then
            file_path="${GEMINI_CONTENT_STORAGE}/prompts/${name}.txt"
        fi
    fi
    
    if [[ -z "$file_path" ]] && ([[ -z "$type" ]] || [[ "$type" == "template" ]] || [[ "$type" == "templates" ]]); then
        if [[ -f "${GEMINI_CONTENT_STORAGE}/templates/${name}.json" ]]; then
            file_path="${GEMINI_CONTENT_STORAGE}/templates/${name}.json"
        fi
    fi
    
    if [[ -z "$file_path" ]] && ([[ -z "$type" ]] || [[ "$type" == "function" ]] || [[ "$type" == "functions" ]]); then
        if [[ -f "${GEMINI_CONTENT_STORAGE}/functions/${name}.json" ]]; then
            file_path="${GEMINI_CONTENT_STORAGE}/functions/${name}.json"
        fi
    fi
    
    if [[ -z "$file_path" ]] || [[ ! -f "$file_path" ]]; then
        log::error "Content not found: $name"
        return 1
    fi
    
    cat "$file_path"
    return 0
}

# Remove content
gemini::content::remove() {
    local name="${1:-}"
    local type="${2:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Content name required"
        return 1
    fi
    
    # Initialize storage
    gemini::content::init
    
    local removed=false
    
    if [[ -z "$type" ]] || [[ "$type" == "prompt" ]] || [[ "$type" == "prompts" ]]; then
        if [[ -f "${GEMINI_CONTENT_STORAGE}/prompts/${name}.txt" ]]; then
            rm -f "${GEMINI_CONTENT_STORAGE}/prompts/${name}.txt"
            removed=true
        fi
    fi
    
    if [[ -z "$type" ]] || [[ "$type" == "template" ]] || [[ "$type" == "templates" ]]; then
        if [[ -f "${GEMINI_CONTENT_STORAGE}/templates/${name}.json" ]]; then
            rm -f "${GEMINI_CONTENT_STORAGE}/templates/${name}.json"
            removed=true
        fi
    fi
    
    if [[ -z "$type" ]] || [[ "$type" == "function" ]] || [[ "$type" == "functions" ]]; then
        if [[ -f "${GEMINI_CONTENT_STORAGE}/functions/${name}.json" ]]; then
            rm -f "${GEMINI_CONTENT_STORAGE}/functions/${name}.json"
            removed=true
        fi
    fi
    
    if [[ "$removed" == true ]]; then
        log::success "Removed content: $name"
        return 0
    else
        log::error "Content not found: $name"
        return 1
    fi
}

# Execute content (generate using stored prompt/template)
gemini::content::execute() {
    local name="${1:-}"
    local model="${2:-${GEMINI_DEFAULT_MODEL}}"
    shift 2
    local params="$@"
    
    if [[ -z "$name" ]]; then
        log::error "Content name required"
        return 1
    fi
    
    # Get content
    local content
    content=$(gemini::content::get "$name") || return 1
    
    # Determine content type
    local content_type="prompt"
    if [[ "$content" =~ ^\{ ]]; then
        content_type="template"
    fi
    
    # Initialize API
    gemini::init >/dev/null 2>&1
    
    if [[ "$content_type" == "template" ]]; then
        # Execute template with parameters
        # Templates can include placeholders that get replaced
        for param in $params; do
            if [[ "$param" =~ ^([^=]+)=(.*)$ ]]; then
                local key="${BASH_REMATCH[1]}"
                local value="${BASH_REMATCH[2]}"
                content="${content//\{\{$key\}\}/$value}"
            fi
        done
        
        # Execute the processed template
        echo "$content" | gemini::generate_from_json
    else
        # Execute as a simple prompt
        gemini::generate "$content" "$model"
    fi
}

# Main content management handler
gemini::content() {
    local action="${1:-list}"
    shift
    
    case "$action" in
        add)
            gemini::content::add "$@"
            ;;
        list)
            gemini::content::list "$@"
            ;;
        get)
            gemini::content::get "$@"
            ;;
        remove|rm|delete)
            gemini::content::remove "$@"
            ;;
        execute|exec|run)
            gemini::content::execute "$@"
            ;;
        *)
            log::error "Unknown content action: $action"
            echo "Available actions: add, list, get, remove, execute"
            return 1
            ;;
    esac
}

# Export functions
export -f gemini::content
export -f gemini::content::init
export -f gemini::content::add
export -f gemini::content::list
export -f gemini::content::get
export -f gemini::content::remove
export -f gemini::content::execute
