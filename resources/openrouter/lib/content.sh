#!/bin/bash
# OpenRouter content management library

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_LIB_DIR="${APP_ROOT}/resources/openrouter/lib"
OPENROUTER_RESOURCE_DIR="${APP_ROOT}/resources/openrouter"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${OPENROUTER_LIB_DIR}/core.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Agent management is now handled via unified system (see cli.sh)

# Content directory for storing prompts and configurations
OPENROUTER_CONTENT_DIR="${var_ROOT_DIR}/data/resources/openrouter/content"

# Initialize content directory
openrouter::content::init() {
    mkdir -p "${OPENROUTER_CONTENT_DIR}/prompts"
    mkdir -p "${OPENROUTER_CONTENT_DIR}/configs"
    mkdir -p "${OPENROUTER_CONTENT_DIR}/routes"
}

# Main content management function
openrouter::content() {
    local action="${1:-}"
    shift
    
    case "$action" in
        add)
            openrouter::content::add "$@"
            ;;
        list)
            openrouter::content::list "$@"
            ;;
        get)
            openrouter::content::get "$@"
            ;;
        remove)
            openrouter::content::remove "$@"
            ;;
        execute)
            openrouter::content::execute "$@"
            ;;
        *)
            echo "Usage: content <add|list|get|remove|execute> [options]"
            echo ""
            echo "Commands:"
            echo "  add --file <file> [--type <type>] [--name <name>]  Add content from file"
            echo "  list [--type <type>]                                List stored content"
            echo "  get --name <name>                                   Get specific content"
            echo "  remove --name <name>                                Remove content"
            echo "  execute --name <name> [--model <model>]             Execute prompt"
            echo ""
            echo "Content types: prompt, config, route"
            return 1
            ;;
    esac
}

# Add content from file
openrouter::content::add() {
    local file=""
    local type="prompt"
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        echo "Error: --file is required"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "Error: File not found: $file"
        return 1
    fi
    
    # Auto-generate name if not provided
    if [[ -z "$name" ]]; then
        name="$(basename "$file" | sed 's/\.[^.]*$//')"
    fi
    
    # Initialize content directory
    openrouter::content::init
    
    # Determine target directory based on type
    local target_dir="${OPENROUTER_CONTENT_DIR}/${type}s"
    if [[ ! -d "$target_dir" ]]; then
        echo "Error: Invalid type: $type (valid: prompt, config, route)"
        return 1
    fi
    
    # Copy file to content directory
    local target_file="${target_dir}/${name}.$(echo "$file" | awk -F. '{print $NF}')"
    cp "$file" "$target_file"
    
    echo "✓ Added ${type}: ${name}"
    return 0
}

# List stored content
openrouter::content::list() {
    local type=""
    local format="text"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
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
    
    openrouter::content::init
    
    if [[ "$format" == "json" ]]; then
        local json_output='{"content":[]}'
        
        # List specific type or all types
        if [[ -n "$type" ]]; then
            local dir="${OPENROUTER_CONTENT_DIR}/${type}s"
            if [[ -d "$dir" ]] && [[ "$(ls -A "$dir" 2>/dev/null)" ]]; then
                for file in "$dir"/*; do
                    [[ -f "$file" ]] || continue
                    local name="$(basename "$file" | sed 's/\.[^.]*$//')"
                    json_output=$(echo "$json_output" | jq ".content += [{\"type\": \"$type\", \"name\": \"$name\", \"path\": \"$file\"}]")
                done
            fi
        else
            for content_type in prompt config route; do
                local dir="${OPENROUTER_CONTENT_DIR}/${content_type}s"
                if [[ -d "$dir" ]] && [[ "$(ls -A "$dir" 2>/dev/null)" ]]; then
                    for file in "$dir"/*; do
                        [[ -f "$file" ]] || continue
                        local name="$(basename "$file" | sed 's/\.[^.]*$//')"
                        json_output=$(echo "$json_output" | jq ".content += [{\"type\": \"$content_type\", \"name\": \"$name\", \"path\": \"$file\"}]")
                    done
                fi
            done
        fi
        
        echo "$json_output"
    else
        echo "OpenRouter Content:"
        echo ""
        
        # List specific type or all types
        if [[ -n "$type" ]]; then
            local dir="${OPENROUTER_CONTENT_DIR}/${type}s"
            if [[ -d "$dir" ]] && [[ "$(ls -A "$dir" 2>/dev/null)" ]]; then
                echo "${type^}s:"
                for file in "$dir"/*; do
                    [[ -f "$file" ]] || continue
                    echo "  - $(basename "$file" | sed 's/\.[^.]*$//')"
                done
            else
                echo "No ${type}s found"
            fi
        else
            for content_type in prompt config route; do
                local dir="${OPENROUTER_CONTENT_DIR}/${content_type}s"
                if [[ -d "$dir" ]] && [[ "$(ls -A "$dir" 2>/dev/null)" ]]; then
                    echo "${content_type^}s:"
                    for file in "$dir"/*; do
                        [[ -f "$file" ]] || continue
                        echo "  - $(basename "$file" | sed 's/\.[^.]*$//')"
                    done
                    echo ""
                fi
            done
        fi
    fi
}

# Get specific content
openrouter::content::get() {
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
        echo "Error: --name is required"
        return 1
    fi
    
    openrouter::content::init
    
    # Search for content in all directories
    for content_type in prompt config route; do
        local dir="${OPENROUTER_CONTENT_DIR}/${content_type}s"
        if [[ -d "$dir" ]]; then
            for file in "$dir"/*; do
                [[ -f "$file" ]] || continue
                local file_name="$(basename "$file" | sed 's/\.[^.]*$//')"
                if [[ "$file_name" == "$name" ]]; then
                    cat "$file"
                    return 0
                fi
            done
        fi
    done
    
    echo "Error: Content not found: $name"
    return 1
}

# Remove content
openrouter::content::remove() {
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
        echo "Error: --name is required"
        return 1
    fi
    
    openrouter::content::init
    
    # Search and remove content from all directories
    local found=false
    for content_type in prompt config route; do
        local dir="${OPENROUTER_CONTENT_DIR}/${content_type}s"
        if [[ -d "$dir" ]]; then
            for file in "$dir"/*; do
                [[ -f "$file" ]] || continue
                local file_name="$(basename "$file" | sed 's/\.[^.]*$//')"
                if [[ "$file_name" == "$name" ]]; then
                    rm "$file"
                    echo "✓ Removed ${content_type}: ${name}"
                    found=true
                    break 2
                fi
            done
        fi
    done
    
    if [[ "$found" == "false" ]]; then
        echo "Error: Content not found: $name"
        return 1
    fi
    
    return 0
}

# Execute a prompt with OpenRouter
openrouter::content::execute() {
    local name=""
    local model="${OPENROUTER_DEFAULT_MODEL:-openai/gpt-3.5-turbo}"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --model)
                model="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name is required"
        return 1
    fi
    
    # Register agent if agent tracking is available
    local agent_id=""
    if declare -f agents::generate_id >/dev/null 2>&1 && declare -f agents::register >/dev/null 2>&1; then
        agent_id=$(agents::generate_id)
        local command="content execute --name $name --model $model"
        agents::register "$agent_id" "$$" "$command" || {
            log::warn "Failed to register agent, continuing without tracking"
        }
        openrouter::setup_agent_cleanup "$agent_id" 2>/dev/null || true
    fi
    
    # Get the prompt content
    local prompt_file=""
    local dir="${OPENROUTER_CONTENT_DIR}/prompts"
    if [[ -d "$dir" ]]; then
        for file in "$dir"/*; do
            [[ -f "$file" ]] || continue
            local file_name="$(basename "$file" | sed 's/\.[^.]*$//')"
            if [[ "$file_name" == "$name" ]]; then
                prompt_file="$file"
                break
            fi
        done
    fi
    
    if [[ -z "$prompt_file" ]] || [[ ! -f "$prompt_file" ]]; then
        echo "Error: Prompt not found: $name"
        return 1
    fi
    
    # Load API key
    local api_key
    api_key=$(openrouter::get_api_key)
    if [[ -z "$api_key" ]] || [[ "$api_key" == "sk-or-v1-placeholder" ]]; then
        echo "Error: Valid OpenRouter API key not configured"
        return 1
    fi
    
    # Read prompt content
    local prompt_content
    prompt_content=$(cat "$prompt_file")
    
    # Execute prompt via OpenRouter API with timeout
    local response
    local timeout="${OPENROUTER_TIMEOUT:-30}"
    
    # Use Cloudflare Gateway if configured
    local api_url
    api_url=$(openrouter::cloudflare::get_gateway_url "$OPENROUTER_API_BASE" "$model")
    
    response=$(timeout "$timeout" curl -s -X POST "${api_url}/chat/completions" \
        -H "Authorization: Bearer ${api_key}" \
        -H "Content-Type: application/json" \
        -d "$(jq -n \
            --arg model "$model" \
            --arg content "$prompt_content" \
            '{
                model: $model,
                messages: [{role: "user", content: $content}],
                max_tokens: 1000
            }')")
    
    # Check for errors
    if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
        echo "Error: $(echo "$response" | jq -r '.error.message // .error')"
        return 1
    fi
    
    # Extract and display the response
    echo "$response" | jq -r '.choices[0].message.content // "No response generated"'
    return 0
}