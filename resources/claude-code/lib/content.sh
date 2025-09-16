#!/usr/bin/env bash
set -euo pipefail

# Claude Code Content Management
# Replaces the inject pattern with clearer content operations

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLAUDE_CODE_SCRIPT_DIR="${APP_ROOT}/resources/claude-code"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091  
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Default paths
readonly DEFAULT_CLAUDE_CODE_DATA_DIR="${HOME}/.claude-code"
readonly DEFAULT_CLAUDE_CODE_TEMPLATES_DIR="${DEFAULT_CLAUDE_CODE_DATA_DIR}/templates"
readonly DEFAULT_CLAUDE_CODE_PROMPTS_DIR="${DEFAULT_CLAUDE_CODE_DATA_DIR}/prompts"
readonly DEFAULT_CLAUDE_CODE_SESSIONS_DIR="${DEFAULT_CLAUDE_CODE_DATA_DIR}/sessions"

# Override with environment variables if set
CLAUDE_CODE_DATA_DIR="${CLAUDE_CODE_DATA_DIR:-$DEFAULT_CLAUDE_CODE_DATA_DIR}"
CLAUDE_CODE_TEMPLATES_DIR="${CLAUDE_CODE_TEMPLATES_DIR:-$DEFAULT_CLAUDE_CODE_TEMPLATES_DIR}"
CLAUDE_CODE_PROMPTS_DIR="${CLAUDE_CODE_PROMPTS_DIR:-$DEFAULT_CLAUDE_CODE_PROMPTS_DIR}"
CLAUDE_CODE_SESSIONS_DIR="${CLAUDE_CODE_SESSIONS_DIR:-$DEFAULT_CLAUDE_CODE_SESSIONS_DIR}"

#######################################
# Add content to Claude Code
# Arguments:
#   --type <type> - Content type (template|prompt|config)
#   --name <name> - Content name
#   --file <file> - File to add
#   --data <data> - Raw data (alternative to file)
# Returns:
#   0 on success, 1 on failure
#######################################
claude_code::content::add() {
    local type=""
    local name=""
    local file=""
    local data=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            --file)
                file="$2"
                shift 2
                ;;
            --data)
                data="$2"
                shift 2
                ;;
            *)
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Missing required arguments: --type and --name"
        return 1
    fi
    
    if [[ -z "$file" ]] && [[ -z "$data" ]]; then
        log::error "Must provide either --file or --data"
        return 1
    fi
    
    # Determine target directory based on type
    local target_dir=""
    case "$type" in
        template)
            target_dir="$CLAUDE_CODE_TEMPLATES_DIR"
            ;;
        prompt)
            target_dir="$CLAUDE_CODE_PROMPTS_DIR"
            ;;
        config)
            target_dir="$CLAUDE_CODE_DATA_DIR/configs"
            ;;
        *)
            log::error "Invalid type: $type (must be template, prompt, or config)"
            return 1
            ;;
    esac
    
    # Create target directory if needed
    mkdir -p "$target_dir"
    
    # Save content
    local target_file="$target_dir/$name"
    if [[ -n "$file" ]]; then
        if [[ ! -f "$file" ]]; then
            log::error "Source file not found: $file"
            return 1
        fi
        cp "$file" "$target_file"
        log::success "Added $type '$name' from file"
    else
        echo "$data" > "$target_file"
        log::success "Added $type '$name' from data"
    fi
    
    return 0
}

#######################################
# List content in Claude Code
# Arguments:
#   --type <type> - Content type (template|prompt|config|all)
#   --format <format> - Output format (text|json)
# Returns:
#   0 on success, 1 on failure
#######################################
claude_code::content::list() {
    local type="all"
    local format="text"
    
    # Parse arguments
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
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    local results=()
    
    # Helper function to list directory contents
    list_directory() {
        local dir="$1"
        local content_type="$2"
        
        if [[ -d "$dir" ]]; then
            for file in "$dir"/*; do
                if [[ -f "$file" ]]; then
                    local basename=$(basename "$file")
                    if [[ "$format" == "json" ]]; then
                        results+=("{\"type\":\"$content_type\",\"name\":\"$basename\",\"path\":\"$file\"}")
                    else
                        results+=("$content_type: $basename")
                    fi
                fi
            done
        fi
    }
    
    # List based on type
    case "$type" in
        template)
            list_directory "$CLAUDE_CODE_TEMPLATES_DIR" "template"
            ;;
        prompt)
            list_directory "$CLAUDE_CODE_PROMPTS_DIR" "prompt"
            ;;
        config)
            list_directory "$CLAUDE_CODE_DATA_DIR/configs" "config"
            ;;
        all)
            list_directory "$CLAUDE_CODE_TEMPLATES_DIR" "template"
            list_directory "$CLAUDE_CODE_PROMPTS_DIR" "prompt"
            list_directory "$CLAUDE_CODE_DATA_DIR/configs" "config"
            ;;
        *)
            log::error "Invalid type: $type"
            return 1
            ;;
    esac
    
    # Output results
    if [[ "$format" == "json" ]]; then
        echo -n "["
        local first=true
        for item in "${results[@]}"; do
            if [[ "$first" == true ]]; then
                echo -n "$item"
                first=false
            else
                echo -n ",$item"
            fi
        done
        echo "]"
    else
        if [[ ${#results[@]} -eq 0 ]]; then
            echo "No content found"
        else
            for item in "${results[@]}"; do
                echo "$item"
            done
        fi
    fi
    
    return 0
}

#######################################
# Get content from Claude Code
# Arguments:
#   --type <type> - Content type (template|prompt|config)
#   --name <name> - Content name
# Returns:
#   0 on success, 1 on failure
#######################################
claude_code::content::get() {
    local type=""
    local name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Missing required arguments: --type and --name"
        return 1
    fi
    
    # Determine source directory based on type
    local source_dir=""
    case "$type" in
        template)
            source_dir="$CLAUDE_CODE_TEMPLATES_DIR"
            ;;
        prompt)
            source_dir="$CLAUDE_CODE_PROMPTS_DIR"
            ;;
        config)
            source_dir="$CLAUDE_CODE_DATA_DIR/configs"
            ;;
        *)
            log::error "Invalid type: $type"
            return 1
            ;;
    esac
    
    # Get content
    local source_file="$source_dir/$name"
    if [[ ! -f "$source_file" ]]; then
        log::error "$type '$name' not found"
        return 1
    fi
    
    cat "$source_file"
    return 0
}

#######################################
# Remove content from Claude Code
# Arguments:
#   --type <type> - Content type (template|prompt|config)
#   --name <name> - Content name
# Returns:
#   0 on success, 1 on failure
#######################################
claude_code::content::remove() {
    local type=""
    local name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Missing required arguments: --type and --name"
        return 1
    fi
    
    # Determine target directory based on type
    local target_dir=""
    case "$type" in
        template)
            target_dir="$CLAUDE_CODE_TEMPLATES_DIR"
            ;;
        prompt)
            target_dir="$CLAUDE_CODE_PROMPTS_DIR"
            ;;
        config)
            target_dir="$CLAUDE_CODE_DATA_DIR/configs"
            ;;
        *)
            log::error "Invalid type: $type"
            return 1
            ;;
    esac
    
    # Remove content
    local target_file="$target_dir/$name"
    if [[ ! -f "$target_file" ]]; then
        log::error "$type '$name' not found"
        return 1
    fi
    
    rm -f "$target_file"
    log::success "Removed $type '$name'"
    return 0
}

#######################################
# Execute a template or prompt
# Arguments:
#   --type <type> - Content type (template|prompt)
#   --name <name> - Content name
#   --model <model> - Model to use (optional)
#   --session <session> - Session to use (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
# Internal execution function (wrapped by agent manager)
claude_code::content::execute_internal() {
    local type="$1"
    local name="$2"
    local model="$3"
    local session="$4"
    
    # Get content
    local content
    content=$(claude_code::content::get --type "$type" --name "$name")
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Build Claude command
    local claude_cmd="claude"
    
    if [[ -n "$model" ]]; then
        claude_cmd="$claude_cmd --model $model"
    fi
    
    if [[ -n "$session" ]]; then
        claude_cmd="$claude_cmd --session $session"
    fi
    
    # Execute based on type
    if [[ "$type" == "template" ]]; then
        # Templates are code files to be processed
        echo "$content" | $claude_cmd --execute
    else
        # Prompts are text to be sent as messages
        echo "$content" | $claude_cmd
    fi
    
    return $?
}

#######################################
claude_code::content::execute() {
    local type=""
    local name=""
    local model=""
    local session=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            --model)
                model="$2"
                shift 2
                ;;
            --session)
                session="$2"
                shift 2
                ;;
            *)
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Missing required arguments: --type and --name"
        return 1
    fi
    
    # Only templates and prompts can be executed
    if [[ "$type" != "template" ]] && [[ "$type" != "prompt" ]]; then
        log::error "Can only execute templates and prompts"
        return 1
    fi
    
    # Use agent wrapper for execution (AI processing operation)
    if type -t agents::with_agent &>/dev/null; then
        agents::with_agent "content-execute" "claude_code::content::execute_internal" "$type" "$name" "$model" "$session"
    else
        # Fallback if agent management not available
        claude_code::content::execute_internal "$type" "$name" "$model" "$session"
    fi
}

#######################################
# Main content management entry point
# Arguments:
#   $1 - Subcommand (add|list|get|remove|execute)
#   $@ - Additional arguments
# Returns:
#   0 on success, 1 on failure
#######################################
claude_code::content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add)
            claude_code::content::add "$@"
            ;;
        list)
            claude_code::content::list "$@"
            ;;
        get)
            claude_code::content::get "$@"
            ;;
        remove)
            claude_code::content::remove "$@"
            ;;
        execute)
            claude_code::content::execute "$@"
            ;;
        *)
            log::error "Invalid subcommand: $subcommand"
            echo "Usage: content <add|list|get|remove|execute> [options]"
            return 1
            ;;
    esac
}