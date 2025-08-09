#!/usr/bin/env bash
set -euo pipefail

# Claude Code Data Injection Adapter
# This script handles injection of templates, prompts, and configurations into Claude Code
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject templates, prompts, and configurations into Claude Code assistant"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get proper directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Source Claude Code configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default Claude Code settings
readonly DEFAULT_CLAUDE_CODE_DATA_DIR="${HOME}/.claude-code"
readonly DEFAULT_CLAUDE_CODE_TEMPLATES_DIR="${DEFAULT_CLAUDE_CODE_DATA_DIR}/templates"
readonly DEFAULT_CLAUDE_CODE_PROMPTS_DIR="${DEFAULT_CLAUDE_CODE_DATA_DIR}/prompts"
readonly DEFAULT_CLAUDE_CODE_SESSIONS_DIR="${DEFAULT_CLAUDE_CODE_DATA_DIR}/sessions"

# Claude Code settings (can be overridden by environment)
CLAUDE_CODE_DATA_DIR="${CLAUDE_CODE_DATA_DIR:-$DEFAULT_CLAUDE_CODE_DATA_DIR}"
CLAUDE_CODE_TEMPLATES_DIR="${CLAUDE_CODE_TEMPLATES_DIR:-$DEFAULT_CLAUDE_CODE_TEMPLATES_DIR}"
CLAUDE_CODE_PROMPTS_DIR="${CLAUDE_CODE_PROMPTS_DIR:-$DEFAULT_CLAUDE_CODE_PROMPTS_DIR}"
CLAUDE_CODE_SESSIONS_DIR="${CLAUDE_CODE_SESSIONS_DIR:-$DEFAULT_CLAUDE_CODE_SESSIONS_DIR}"

# Operation tracking
declare -a CLAUDE_CODE_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
claude_code_inject::usage() {
    cat << EOF
Claude Code Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects templates, prompts, sessions, and configurations into Claude Code based on 
    scenario configuration. Supports validation, injection, status checks, 
    and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected data
    --rollback    Rollback injected data
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "templates": [
        {
          "name": "react_component",
          "file": "path/to/template.js",
          "description": "React component template",
          "tags": ["react", "frontend"]
        }
      ],
      "prompts": [
        {
          "name": "code_review",
          "file": "path/to/prompt.md",
          "category": "review"
        }
      ],
      "sessions": [
        {
          "name": "project_setup",
          "file": "path/to/session.json",
          "autoload": true
        }
      ],
      "api_keys": [
        {
          "provider": "anthropic",
          "key": "sk-...",
          "encrypted": false
        }
      ],
      "configurations": [
        {
          "key": "model",
          "value": "claude-3-sonnet"
        },
        {
          "key": "temperature",
          "value": 0.7
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"templates": [{"name": "test", "file": "template.js"}]}'
    
    # Inject templates and prompts
    $0 --inject '{"templates": [{"name": "react", "file": "templates/react.js"}]}'
    
    # Configure API keys
    $0 --inject '{"api_keys": [{"provider": "anthropic", "key": "sk-...", "encrypted": false}]}'

EOF
}

#######################################
# Check if Claude Code is accessible
# Returns:
#   0 if accessible (config exists), 1 otherwise
#######################################
claude_code_inject::check_accessibility() {
    # Claude Code doesn't have a running service, check for CLI installation
    if system::is_command "claude-code" || system::is_command "claude"; then
        log::debug "Claude Code CLI is installed"
        return 0
    else
        # Check if data directory exists as alternative indicator
        if [[ -d "$CLAUDE_CODE_DATA_DIR" ]]; then
            log::debug "Claude Code data directory exists"
            return 0
        else
            log::warn "Claude Code CLI not found and data directory doesn't exist"
            log::info "Install Claude Code: ./scripts/resources/agents/claude-code/manage.sh --action install"
            return 1
        fi
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
claude_code_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    CLAUDE_CODE_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Claude Code rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
claude_code_inject::execute_rollback() {
    if [[ ${#CLAUDE_CODE_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Claude Code rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Claude Code rollback actions..."
    
    local success_count=0
    local total_count=${#CLAUDE_CODE_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#CLAUDE_CODE_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${CLAUDE_CODE_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Claude Code rollback completed: $success_count/$total_count actions successful"
    CLAUDE_CODE_ROLLBACK_ACTIONS=()
}

#######################################
# Validate template configuration
# Arguments:
#   $1 - templates configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
claude_code_inject::validate_templates() {
    local templates_config="$1"
    
    log::debug "Validating template configurations..."
    
    # Check if templates is an array
    local templates_type
    templates_type=$(echo "$templates_config" | jq -r 'type')
    
    if [[ "$templates_type" != "array" ]]; then
        log::error "Templates configuration must be an array, got: $templates_type"
        return 1
    fi
    
    # Validate each template
    local template_count
    template_count=$(echo "$templates_config" | jq 'length')
    
    for ((i=0; i<template_count; i++)); do
        local template
        template=$(echo "$templates_config" | jq -c ".[$i]")
        
        # Check required fields
        local name file
        name=$(echo "$template" | jq -r '.name // empty')
        file=$(echo "$template" | jq -r '.file // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Template at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "Template '$name' missing required 'file' field"
            return 1
        fi
        
        # Check if file exists
        local template_path="$var_ROOT_DIR/$file"
        if [[ ! -f "$template_path" ]]; then
            log::error "Template file not found: $template_path"
            return 1
        fi
        
        log::debug "Template '$name' configuration is valid"
    done
    
    log::success "All template configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
claude_code_inject::validate_config() {
    local config="$1"
    
    log::info "Validating Claude Code injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Claude Code injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_templates has_prompts has_sessions has_api_keys has_configurations
    has_templates=$(echo "$config" | jq -e '.templates' >/dev/null 2>&1 && echo "true" || echo "false")
    has_prompts=$(echo "$config" | jq -e '.prompts' >/dev/null 2>&1 && echo "true" || echo "false")
    has_sessions=$(echo "$config" | jq -e '.sessions' >/dev/null 2>&1 && echo "true" || echo "false")
    has_api_keys=$(echo "$config" | jq -e '.api_keys' >/dev/null 2>&1 && echo "true" || echo "false")
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_templates" == "false" && "$has_prompts" == "false" && "$has_sessions" == "false" && "$has_api_keys" == "false" && "$has_configurations" == "false" ]]; then
        log::error "Claude Code injection configuration must have 'templates', 'prompts', 'sessions', 'api_keys', or 'configurations'"
        return 1
    fi
    
    # Validate templates if present
    if [[ "$has_templates" == "true" ]]; then
        local templates
        templates=$(echo "$config" | jq -c '.templates')
        
        if ! claude_code_inject::validate_templates "$templates"; then
            return 1
        fi
    fi
    
    # Validate prompts if present
    if [[ "$has_prompts" == "true" ]]; then
        local prompts
        prompts=$(echo "$config" | jq -c '.prompts')
        
        local prompt_count
        prompt_count=$(echo "$prompts" | jq 'length')
        
        for ((i=0; i<prompt_count; i++)); do
            local prompt
            prompt=$(echo "$prompts" | jq -c ".[$i]")
            
            local name file
            name=$(echo "$prompt" | jq -r '.name // empty')
            file=$(echo "$prompt" | jq -r '.file // empty')
            
            if [[ -z "$name" ]]; then
                log::error "Prompt at index $i missing required 'name' field"
                return 1
            fi
            
            if [[ -z "$file" ]]; then
                log::error "Prompt '$name' missing required 'file' field"
                return 1
            fi
            
            # Check if file exists
            local prompt_path="$var_ROOT_DIR/$file"
            if [[ ! -f "$prompt_path" ]]; then
                log::error "Prompt file not found: $prompt_path"
                return 1
            fi
        done
    fi
    
    log::success "Claude Code injection configuration is valid"
    return 0
}

#######################################
# Install template
# Arguments:
#   $1 - template configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::install_template() {
    local template_config="$1"
    
    local name file description tags
    name=$(echo "$template_config" | jq -r '.name')
    file=$(echo "$template_config" | jq -r '.file')
    description=$(echo "$template_config" | jq -r '.description // ""')
    tags=$(echo "$template_config" | jq -r '.tags // []')
    
    log::info "Installing template: $name"
    
    # Resolve file path
    local template_path="$var_ROOT_DIR/$file"
    
    # Create templates directory if it doesn't exist
    mkdir -p "$CLAUDE_CODE_TEMPLATES_DIR"
    
    # Determine file extension
    local extension="${file##*.}"
    local dest_file="${CLAUDE_CODE_TEMPLATES_DIR}/${name}.${extension}"
    
    # Copy template file
    cp "$template_path" "$dest_file"
    
    # Create metadata file
    local metadata_file="${CLAUDE_CODE_TEMPLATES_DIR}/${name}.meta.json"
    cat > "$metadata_file" << EOF
{
  "name": "$name",
  "description": "$description",
  "tags": $tags,
  "extension": "$extension",
  "installed": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    log::success "Installed template: $name"
    
    # Add rollback actions
    claude_code_inject::add_rollback_action \
        "Remove template: $name" \
        "rm -f '${dest_file}' '${metadata_file}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Install prompt
# Arguments:
#   $1 - prompt configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::install_prompt() {
    local prompt_config="$1"
    
    local name file category
    name=$(echo "$prompt_config" | jq -r '.name')
    file=$(echo "$prompt_config" | jq -r '.file')
    category=$(echo "$prompt_config" | jq -r '.category // "general"')
    
    log::info "Installing prompt: $name"
    
    # Resolve file path
    local prompt_path="$var_ROOT_DIR/$file"
    
    # Create prompts directory with category subdirectory
    local category_dir="${CLAUDE_CODE_PROMPTS_DIR}/${category}"
    mkdir -p "$category_dir"
    
    # Copy prompt file
    local dest_file="${category_dir}/${name}.md"
    cp "$prompt_path" "$dest_file"
    
    # Create metadata file
    local metadata_file="${category_dir}/${name}.meta.json"
    cat > "$metadata_file" << EOF
{
  "name": "$name",
  "category": "$category",
  "installed": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    log::success "Installed prompt: $name"
    
    # Add rollback actions
    claude_code_inject::add_rollback_action \
        "Remove prompt: $name" \
        "rm -f '${dest_file}' '${metadata_file}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Install session
# Arguments:
#   $1 - session configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::install_session() {
    local session_config="$1"
    
    local name file autoload
    name=$(echo "$session_config" | jq -r '.name')
    file=$(echo "$session_config" | jq -r '.file')
    autoload=$(echo "$session_config" | jq -r '.autoload // false')
    
    log::info "Installing session: $name"
    
    # Resolve file path
    local session_path="$var_ROOT_DIR/$file"
    
    # Create sessions directory
    mkdir -p "$CLAUDE_CODE_SESSIONS_DIR"
    
    # Copy session file
    local dest_file="${CLAUDE_CODE_SESSIONS_DIR}/${name}.json"
    cp "$session_path" "$dest_file"
    
    # Create metadata file
    local metadata_file="${CLAUDE_CODE_SESSIONS_DIR}/${name}.meta.json"
    cat > "$metadata_file" << EOF
{
  "name": "$name",
  "autoload": $autoload,
  "installed": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    # Set as default session if autoload is true
    if [[ "$autoload" == "true" ]]; then
        echo "$name" > "${CLAUDE_CODE_SESSIONS_DIR}/.default_session"
    fi
    
    log::success "Installed session: $name"
    
    # Add rollback actions
    claude_code_inject::add_rollback_action \
        "Remove session: $name" \
        "rm -f '${dest_file}' '${metadata_file}' 2>/dev/null || true"
    
    if [[ "$autoload" == "true" ]]; then
        claude_code_inject::add_rollback_action \
            "Remove default session marker" \
            "rm -f '${CLAUDE_CODE_SESSIONS_DIR}/.default_session' 2>/dev/null || true"
    fi
    
    return 0
}

#######################################
# Configure API keys
# Arguments:
#   $1 - API keys configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::configure_api_keys() {
    local api_keys_config="$1"
    
    log::info "Configuring API keys..."
    
    local key_count
    key_count=$(echo "$api_keys_config" | jq 'length')
    
    if [[ "$key_count" -eq 0 ]]; then
        log::info "No API keys to configure"
        return 0
    fi
    
    # Create secure keys directory
    local keys_dir="${CLAUDE_CODE_DATA_DIR}/.keys"
    mkdir -p "$keys_dir"
    chmod 700 "$keys_dir"
    
    for ((i=0; i<key_count; i++)); do
        local key_config
        key_config=$(echo "$api_keys_config" | jq -c ".[$i]")
        
        local provider key encrypted
        provider=$(echo "$key_config" | jq -r '.provider')
        key=$(echo "$key_config" | jq -r '.key')
        encrypted=$(echo "$key_config" | jq -r '.encrypted // false')
        
        log::info "Configuring API key for: $provider"
        
        # Save key (consider encryption in production)
        local key_file="${keys_dir}/${provider}.key"
        
        if [[ "$encrypted" == "true" ]]; then
            # In production, decrypt the key here
            echo "$key" > "$key_file"
        else
            # Store plaintext key (not recommended for production)
            echo "$key" > "$key_file"
        fi
        
        chmod 600 "$key_file"
        
        # Add rollback action
        claude_code_inject::add_rollback_action \
            "Remove API key: $provider" \
            "rm -f '${key_file}' 2>/dev/null || true"
    done
    
    log::success "Configured API keys"
    return 0
}

#######################################
# Inject templates
# Arguments:
#   $1 - templates configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::inject_templates() {
    local templates_config="$1"
    
    log::info "Injecting Claude Code templates..."
    
    local template_count
    template_count=$(echo "$templates_config" | jq 'length')
    
    if [[ "$template_count" -eq 0 ]]; then
        log::info "No templates to inject"
        return 0
    fi
    
    local failed_templates=()
    
    for ((i=0; i<template_count; i++)); do
        local template
        template=$(echo "$templates_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$template" | jq -r '.name')
        
        if ! claude_code_inject::install_template "$template"; then
            failed_templates+=("$name")
        fi
    done
    
    if [[ ${#failed_templates[@]} -eq 0 ]]; then
        log::success "All templates injected successfully"
        return 0
    else
        log::error "Failed to inject templates: ${failed_templates[*]}"
        return 1
    fi
}

#######################################
# Inject prompts
# Arguments:
#   $1 - prompts configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::inject_prompts() {
    local prompts_config="$1"
    
    log::info "Injecting Claude Code prompts..."
    
    local prompt_count
    prompt_count=$(echo "$prompts_config" | jq 'length')
    
    if [[ "$prompt_count" -eq 0 ]]; then
        log::info "No prompts to inject"
        return 0
    fi
    
    local failed_prompts=()
    
    for ((i=0; i<prompt_count; i++)); do
        local prompt
        prompt=$(echo "$prompts_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$prompt" | jq -r '.name')
        
        if ! claude_code_inject::install_prompt "$prompt"; then
            failed_prompts+=("$name")
        fi
    done
    
    if [[ ${#failed_prompts[@]} -eq 0 ]]; then
        log::success "All prompts injected successfully"
        return 0
    else
        log::error "Failed to inject prompts: ${failed_prompts[*]}"
        return 1
    fi
}

#######################################
# Inject sessions
# Arguments:
#   $1 - sessions configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::inject_sessions() {
    local sessions_config="$1"
    
    log::info "Injecting Claude Code sessions..."
    
    local session_count
    session_count=$(echo "$sessions_config" | jq 'length')
    
    if [[ "$session_count" -eq 0 ]]; then
        log::info "No sessions to inject"
        return 0
    fi
    
    local failed_sessions=()
    
    for ((i=0; i<session_count; i++)); do
        local session
        session=$(echo "$sessions_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$session" | jq -r '.name')
        
        if ! claude_code_inject::install_session "$session"; then
            failed_sessions+=("$name")
        fi
    done
    
    if [[ ${#failed_sessions[@]} -eq 0 ]]; then
        log::success "All sessions injected successfully"
        return 0
    else
        log::error "Failed to inject sessions: ${failed_sessions[*]}"
        return 1
    fi
}

#######################################
# Apply configurations
# Arguments:
#   $1 - configurations JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::apply_configurations() {
    local configurations="$1"
    
    log::info "Applying Claude Code configurations..."
    
    local config_count
    config_count=$(echo "$configurations" | jq 'length')
    
    if [[ "$config_count" -eq 0 ]]; then
        log::info "No configurations to apply"
        return 0
    fi
    
    # Save configuration settings
    local config_file="${CLAUDE_CODE_DATA_DIR}/config.json"
    echo "$configurations" > "$config_file"
    
    log::success "Saved configuration settings"
    
    # Add rollback action
    claude_code_inject::add_rollback_action \
        "Remove configuration file" \
        "rm -f '${config_file}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Claude Code"
    
    # Check Claude Code accessibility (installation)
    claude_code_inject::check_accessibility || true
    
    # Clear previous rollback actions
    CLAUDE_CODE_ROLLBACK_ACTIONS=()
    
    # Inject templates if present
    local has_templates
    has_templates=$(echo "$config" | jq -e '.templates' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_templates" == "true" ]]; then
        local templates
        templates=$(echo "$config" | jq -c '.templates')
        
        if ! claude_code_inject::inject_templates "$templates"; then
            log::error "Failed to inject templates"
            claude_code_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject prompts if present
    local has_prompts
    has_prompts=$(echo "$config" | jq -e '.prompts' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_prompts" == "true" ]]; then
        local prompts
        prompts=$(echo "$config" | jq -c '.prompts')
        
        if ! claude_code_inject::inject_prompts "$prompts"; then
            log::error "Failed to inject prompts"
            claude_code_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject sessions if present
    local has_sessions
    has_sessions=$(echo "$config" | jq -e '.sessions' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_sessions" == "true" ]]; then
        local sessions
        sessions=$(echo "$config" | jq -c '.sessions')
        
        if ! claude_code_inject::inject_sessions "$sessions"; then
            log::error "Failed to inject sessions"
            claude_code_inject::execute_rollback
            return 1
        fi
    fi
    
    # Configure API keys if present
    local has_api_keys
    has_api_keys=$(echo "$config" | jq -e '.api_keys' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_api_keys" == "true" ]]; then
        local api_keys
        api_keys=$(echo "$config" | jq -c '.api_keys')
        
        if ! claude_code_inject::configure_api_keys "$api_keys"; then
            log::error "Failed to configure API keys"
            claude_code_inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply configurations if present
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')
        
        if ! claude_code_inject::apply_configurations "$configurations"; then
            log::error "Failed to apply configurations"
            claude_code_inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ Claude Code data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
claude_code_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Claude Code injection status"
    
    # Check Claude Code installation
    if claude_code_inject::check_accessibility; then
        log::success "✅ Claude Code is installed"
    else
        log::warn "⚠️  Claude Code is not installed"
    fi
    
    # Check templates
    if [[ -d "$CLAUDE_CODE_TEMPLATES_DIR" ]]; then
        local template_count
        template_count=$(find "$CLAUDE_CODE_TEMPLATES_DIR" -type f ! -name "*.meta.json" 2>/dev/null | wc -l)
        
        if [[ "$template_count" -gt 0 ]]; then
            log::info "Found $template_count templates"
        else
            log::info "No templates found"
        fi
    else
        log::info "Templates directory does not exist"
    fi
    
    # Check prompts
    if [[ -d "$CLAUDE_CODE_PROMPTS_DIR" ]]; then
        local prompt_count
        prompt_count=$(find "$CLAUDE_CODE_PROMPTS_DIR" -name "*.md" 2>/dev/null | wc -l)
        
        if [[ "$prompt_count" -gt 0 ]]; then
            log::info "Found $prompt_count prompts"
            
            # List categories
            local categories
            categories=$(find "$CLAUDE_CODE_PROMPTS_DIR" -maxdepth 1 -type d ! -path "$CLAUDE_CODE_PROMPTS_DIR" 2>/dev/null | wc -l)
            
            if [[ "$categories" -gt 0 ]]; then
                log::info "  - Organized in $categories categories"
            fi
        else
            log::info "No prompts found"
        fi
    else
        log::info "Prompts directory does not exist"
    fi
    
    # Check sessions
    if [[ -d "$CLAUDE_CODE_SESSIONS_DIR" ]]; then
        local session_count
        session_count=$(find "$CLAUDE_CODE_SESSIONS_DIR" -name "*.json" ! -name "*.meta.json" 2>/dev/null | wc -l)
        
        if [[ "$session_count" -gt 0 ]]; then
            log::info "Found $session_count sessions"
            
            # Check for default session
            if [[ -f "${CLAUDE_CODE_SESSIONS_DIR}/.default_session" ]]; then
                local default_session
                default_session=$(cat "${CLAUDE_CODE_SESSIONS_DIR}/.default_session")
                log::info "  - Default session: $default_session"
            fi
        else
            log::info "No sessions found"
        fi
    else
        log::info "Sessions directory does not exist"
    fi
    
    # Check API keys
    local keys_dir="${CLAUDE_CODE_DATA_DIR}/.keys"
    if [[ -d "$keys_dir" ]]; then
        local key_count
        key_count=$(find "$keys_dir" -name "*.key" 2>/dev/null | wc -l)
        
        if [[ "$key_count" -gt 0 ]]; then
            log::info "Found $key_count API key configurations"
        else
            log::info "No API keys configured"
        fi
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
claude_code_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        claude_code_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            claude_code_inject::validate_config "$config"
            ;;
        "--inject")
            claude_code_inject::inject_data "$config"
            ;;
        "--status")
            claude_code_inject::check_status "$config"
            ;;
        "--rollback")
            claude_code_inject::execute_rollback
            ;;
        "--help")
            claude_code_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            claude_code_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        claude_code_inject::usage
        exit 1
    fi
    
    claude_code_inject::main "$@"
fi