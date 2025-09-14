#!/usr/bin/env bash
# Cline Content Management Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLINE_LIB_DIR="${APP_ROOT}/resources/cline/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CLINE_LIB_DIR}/common.sh"
# shellcheck disable=SC1091
source "${CLINE_LIB_DIR}/agents.sh" 2>/dev/null || true

#######################################
# Add content to Cline (templates, configs, etc.)
# Args: <type> <name> [file]
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::add() {
    local type="${1:-}"
    local name="${2:-}"
    local file="${3:-}"
    
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Usage: content add <type> <name> [file]"
        log::info "Types: template, config, prompt"
        return 1
    fi
    
    # Ensure directories exist
    mkdir -p "$CLINE_DATA_DIR/$type"
    
    case "$type" in
        template|templates)
            if [[ -n "$file" ]] && [[ -f "$file" ]]; then
                cp "$file" "$CLINE_DATA_DIR/templates/$name"
                log::success "Template added: $name"
            else
                log::error "Template file not found: $file"
                return 1
            fi
            ;;
        config|configs)
            if [[ -n "$file" ]] && [[ -f "$file" ]]; then
                if jq empty "$file" 2>/dev/null; then
                    cp "$file" "$CLINE_DATA_DIR/configs/$name.json"
                    log::success "Configuration added: $name"
                else
                    log::error "Invalid JSON configuration file"
                    return 1
                fi
            else
                log::error "Configuration file not found: $file"
                return 1
            fi
            ;;
        prompt|prompts)
            if [[ -n "$file" ]] && [[ -f "$file" ]]; then
                cp "$file" "$CLINE_DATA_DIR/prompts/$name"
                log::success "Prompt added: $name"
            else
                log::error "Prompt file not found: $file"
                return 1
            fi
            ;;
        *)
            log::error "Unknown content type: $type"
            log::info "Available types: template, config, prompt"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# List content in Cline
# Args: [type]
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::list() {
    local type="${1:-all}"
    
    if [[ ! -d "$CLINE_DATA_DIR" ]]; then
        log::info "No content directory found. Use 'content add' to add content."
        return 0
    fi
    
    case "$type" in
        all)
            log::info "Cline Content:"
            for content_type in templates configs prompts; do
                if [[ -d "$CLINE_DATA_DIR/$content_type" ]]; then
                    local count=$(find "$CLINE_DATA_DIR/$content_type" -type f 2>/dev/null | wc -l)
                    if [[ $count -gt 0 ]]; then
                        echo "  $content_type/ ($count files)"
                        find "$CLINE_DATA_DIR/$content_type" -type f -printf "    %f\n" 2>/dev/null | sort
                    fi
                fi
            done
            ;;
        template|templates)
            if [[ -d "$CLINE_DATA_DIR/templates" ]]; then
                log::info "Templates:"
                find "$CLINE_DATA_DIR/templates" -type f -printf "  %f\n" 2>/dev/null | sort
            else
                log::info "No templates found"
            fi
            ;;
        config|configs)
            if [[ -d "$CLINE_DATA_DIR/configs" ]]; then
                log::info "Configurations:"
                find "$CLINE_DATA_DIR/configs" -type f -printf "  %f\n" 2>/dev/null | sort
            else
                log::info "No configurations found"
            fi
            ;;
        prompt|prompts)
            if [[ -d "$CLINE_DATA_DIR/prompts" ]]; then
                log::info "Prompts:"
                find "$CLINE_DATA_DIR/prompts" -type f -printf "  %f\n" 2>/dev/null | sort
            else
                log::info "No prompts found"
            fi
            ;;
        *)
            log::error "Unknown content type: $type"
            log::info "Available types: all, template, config, prompt"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Get content from Cline
# Args: <type> <name>
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::get() {
    local type="${1:-}"
    local name="${2:-}"
    
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Usage: content get <type> <name>"
        log::info "Types: template, config, prompt"
        return 1
    fi
    
    local file_path
    case "$type" in
        template|templates)
            file_path="$CLINE_DATA_DIR/templates/$name"
            ;;
        config|configs)
            file_path="$CLINE_DATA_DIR/configs/$name.json"
            if [[ ! -f "$file_path" ]]; then
                file_path="$CLINE_DATA_DIR/configs/$name"
            fi
            ;;
        prompt|prompts)
            file_path="$CLINE_DATA_DIR/prompts/$name"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
    
    if [[ -f "$file_path" ]]; then
        log::info "Content: $type/$name"
        echo "---"
        cat "$file_path"
    else
        log::error "Content not found: $type/$name"
        return 1
    fi
    
    return 0
}

#######################################
# Remove content from Cline
# Args: <type> <name>
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::remove() {
    local type="${1:-}"
    local name="${2:-}"
    
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Usage: content remove <type> <name>"
        log::info "Types: template, config, prompt"
        return 1
    fi
    
    local file_path
    case "$type" in
        template|templates)
            file_path="$CLINE_DATA_DIR/templates/$name"
            ;;
        config|configs)
            file_path="$CLINE_DATA_DIR/configs/$name.json"
            if [[ ! -f "$file_path" ]]; then
                file_path="$CLINE_DATA_DIR/configs/$name"
            fi
            ;;
        prompt|prompts)
            file_path="$CLINE_DATA_DIR/prompts/$name"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
    
    if [[ -f "$file_path" ]]; then
        rm "$file_path"
        log::success "Content removed: $type/$name"
    else
        log::error "Content not found: $type/$name"
        return 1
    fi
    
    return 0
}

#######################################
# Setup agent cleanup on signals
# Arguments:
#   $1 - Agent ID
#######################################
cline::setup_agent_cleanup() {
    local agent_id="$1"
    
    # Export the agent ID so trap can access it
    export CLINE_CURRENT_AGENT_ID="$agent_id"
    
    # Cleanup function that uses the exported variable
    cline::agent_cleanup() {
        if [[ -n "${CLINE_CURRENT_AGENT_ID:-}" ]] && type -t agents::unregister &>/dev/null; then
            agents::unregister "${CLINE_CURRENT_AGENT_ID}" >/dev/null 2>&1
        fi
        exit 0
    }
    
    # Register cleanup for common signals
    trap 'cline::agent_cleanup' EXIT SIGTERM SIGINT
}

#######################################
# Execute content using Cline (business functionality)
# Args: <action> [args...]
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::execute() {
    local action="${1:-}"
    shift || true
    
    # Register agent if agent management is available
    local agent_id=""
    if type -t agents::register &>/dev/null; then
        agent_id=$(agents::generate_id)
        local command_string="resource-cline content execute $action $*"
        if agents::register "$agent_id" $$ "$command_string"; then
            log::debug "Registered agent: $agent_id"
            
            # Set up signal handler for cleanup
            cline::setup_agent_cleanup "$agent_id"
        fi
    fi
    
    case "$action" in
        open|launch)
            # Open VS Code with Cline if available
            if cline::check_vscode && cline::is_installed; then
                log::info "Opening VS Code with Cline..."
                code --command "cline.openChat" 2>/dev/null || code
            else
                log::error "VS Code or Cline extension not available"
                return 1
            fi
            ;;
        configure)
            # Configure Cline with current settings
            cline::configure
            ;;
        chat)
            # Information about starting a chat session
            log::info "To start a Cline chat session:"
            log::info "1. Open VS Code"
            log::info "2. Press Cmd/Ctrl+Shift+P"
            log::info "3. Type 'Cline: Open Chat'"
            log::info "4. Select your provider and model"
            log::info "5. Start coding with AI assistance!"
            ;;
        prompt)
            # Send a prompt directly to the AI provider from terminal
            local prompt="$*"
            if [[ -z "$prompt" ]]; then
                log::error "Please provide a prompt"
                return 1
            fi
            
            local provider=$(cline::get_provider)
            log::info "Sending prompt to $provider..."
            
            # For now, echo the prompt - in future this would integrate with the API
            log::info "Prompt: $prompt"
            log::info "Note: Direct API integration coming soon. Use VS Code for now."
            ;;
        provider)
            # Switch or show provider
            local new_provider="${1:-}"
            if [[ -n "$new_provider" ]]; then
                cline::config_provider "$new_provider"
            else
                log::info "Current provider: $(cline::get_provider)"
                log::info "Available: ollama, openrouter, anthropic, openai, google"
            fi
            ;;
        models)
            # List available models for the current provider
            local provider=$(cline::get_provider)
            log::info "Models for $provider:"
            
            case "$provider" in
                ollama)
                    if timeout 5 curl -sf http://localhost:11434/api/tags 2>/dev/null | jq -r '.models[].name' 2>/dev/null; then
                        :
                    else
                        log::warn "Cannot fetch Ollama models (is Ollama running?)"
                    fi
                    ;;
                openrouter)
                    log::info "Popular OpenRouter models:"
                    log::info "  - anthropic/claude-3.5-sonnet"
                    log::info "  - openai/gpt-4-turbo"
                    log::info "  - meta-llama/llama-3.1-70b"
                    log::info "  - google/gemini-pro-1.5"
                    ;;
                *)
                    log::info "Model listing not implemented for $provider"
                    ;;
            esac
            ;;
        status)
            # Show detailed status for execution context
            cline::status
            ;;
        *)
            log::error "Unknown execute action: $action"
            log::info "Available actions: open, configure, chat, prompt, provider, models, status"
            return 1
            ;;
    esac
    
    # Unregister agent on success
    if [[ -n "$agent_id" ]] && type -t agents::unregister &>/dev/null; then
        agents::unregister "$agent_id" >/dev/null 2>&1
    fi
    
    return 0
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-list}" in
        add)
            shift
            cline::content::add "$@"
            ;;
        list)
            shift
            cline::content::list "$@"
            ;;
        get)
            shift
            cline::content::get "$@"
            ;;
        remove)
            shift
            cline::content::remove "$@"
            ;;
        execute)
            shift
            cline::content::execute "$@"
            ;;
        *)
            echo "Usage: $0 [add|list|get|remove|execute] [args...]"
            exit 1
            ;;
    esac
fi