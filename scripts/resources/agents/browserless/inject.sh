#!/usr/bin/env bash
set -euo pipefail

# Browserless Data Injection Adapter
# This script handles injection of configurations and scripts into Browserless
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject configurations and scripts into Browserless headless browser service"

BROWSERLESS_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "$(dirname "$(dirname "$(dirname "${BROWSERLESS_SCRIPT_DIR}")")")/lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Source Browserless configuration if available
if [[ -f "${BROWSERLESS_SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${BROWSERLESS_SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default Browserless settings
readonly DEFAULT_BROWSERLESS_HOST="http://localhost:3001"
readonly DEFAULT_BROWSERLESS_DATA_DIR="${HOME}/.browserless"
readonly DEFAULT_BROWSERLESS_SCRIPTS_DIR="${DEFAULT_BROWSERLESS_DATA_DIR}/scripts"
readonly DEFAULT_BROWSERLESS_FUNCTIONS_DIR="${DEFAULT_BROWSERLESS_DATA_DIR}/functions"

# Browserless settings (can be overridden by environment)
BROWSERLESS_HOST="${BROWSERLESS_HOST:-$DEFAULT_BROWSERLESS_HOST}"
BROWSERLESS_DATA_DIR="${BROWSERLESS_DATA_DIR:-$DEFAULT_BROWSERLESS_DATA_DIR}"
BROWSERLESS_SCRIPTS_DIR="${BROWSERLESS_SCRIPTS_DIR:-$DEFAULT_BROWSERLESS_SCRIPTS_DIR}"
BROWSERLESS_FUNCTIONS_DIR="${BROWSERLESS_FUNCTIONS_DIR:-$DEFAULT_BROWSERLESS_FUNCTIONS_DIR}"

# Operation tracking
declare -a BROWSERLESS_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
browserless_inject::usage() {
    cat << EOF
Browserless Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects scripts, functions, and configurations into Browserless based on 
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
      "scripts": [
        {
          "name": "screenshot_script",
          "file": "path/to/script.js",
          "type": "puppeteer"
        }
      ],
      "functions": [
        {
          "name": "extract_data",
          "file": "path/to/function.js",
          "endpoint": "/function/extract_data"
        }
      ],
      "browser_contexts": [
        {
          "name": "default",
          "viewport": {
            "width": 1920,
            "height": 1080
          },
          "user_agent": "Mozilla/5.0...",
          "extra_headers": {
            "Accept-Language": "en-US,en;q=0.9"
          }
        }
      ],
      "configurations": [
        {
          "key": "max_concurrent_sessions",
          "value": 10
        },
        {
          "key": "preboot_chrome",
          "value": true
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"scripts": [{"name": "test", "file": "test.js", "type": "puppeteer"}]}'
    
    # Inject scripts and functions
    $0 --inject '{"scripts": [{"name": "scraper", "file": "scripts/scraper.js"}]}'
    
    # Configure browser contexts
    $0 --inject '{"browser_contexts": [{"name": "mobile", "viewport": {"width": 375, "height": 667}}]}'

EOF
}

#######################################
# Check if Browserless is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
browserless_inject::check_accessibility() {
    # Check if Browserless is running
    if curl -s --max-time 5 "${BROWSERLESS_HOST}/pressure" 2>/dev/null | grep -q "queued"; then
        log::debug "Browserless is accessible at $BROWSERLESS_HOST"
        return 0
    else
        log::error "Browserless is not accessible at $BROWSERLESS_HOST"
        log::info "Ensure Browserless is running: ./scripts/resources/agents/browserless/manage.sh --action start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
browserless_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    BROWSERLESS_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Browserless rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
browserless_inject::execute_rollback() {
    if [[ ${#BROWSERLESS_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Browserless rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Browserless rollback actions..."
    
    local success_count=0
    local total_count=${#BROWSERLESS_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#BROWSERLESS_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${BROWSERLESS_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Browserless rollback completed: $success_count/$total_count actions successful"
    BROWSERLESS_ROLLBACK_ACTIONS=()
}

#######################################
# Validate script configuration
# Arguments:
#   $1 - scripts configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
browserless_inject::validate_scripts() {
    local scripts_config="$1"
    
    log::debug "Validating script configurations..."
    
    # Check if scripts is an array
    local scripts_type
    scripts_type=$(echo "$scripts_config" | jq -r 'type')
    
    if [[ "$scripts_type" != "array" ]]; then
        log::error "Scripts configuration must be an array, got: $scripts_type"
        return 1
    fi
    
    # Validate each script
    local script_count
    script_count=$(echo "$scripts_config" | jq 'length')
    
    local valid_types=("puppeteer" "playwright" "chrome-cdp")
    
    for ((i=0; i<script_count; i++)); do
        local script
        script=$(echo "$scripts_config" | jq -c ".[$i]")
        
        # Check required fields
        local name file type
        name=$(echo "$script" | jq -r '.name // empty')
        file=$(echo "$script" | jq -r '.file // empty')
        type=$(echo "$script" | jq -r '.type // "puppeteer"')
        
        if [[ -z "$name" ]]; then
            log::error "Script at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "Script '$name' missing required 'file' field"
            return 1
        fi
        
        # Validate script type
        local is_valid=false
        for valid_type in "${valid_types[@]}"; do
            if [[ "$type" == "$valid_type" ]]; then
                is_valid=true
                break
            fi
        done
        
        if [[ "$is_valid" == false ]]; then
            log::error "Invalid script type: $type. Valid types: ${valid_types[*]}"
            return 1
        fi
        
        # Check if file exists
        local script_path="$VROOLI_PROJECT_ROOT/$file"
        if [[ ! -f "$script_path" ]]; then
            log::error "Script file not found: $script_path"
            return 1
        fi
        
        log::debug "Script '$name' configuration is valid"
    done
    
    log::success "All script configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
browserless_inject::validate_config() {
    local config="$1"
    
    log::info "Validating Browserless injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Browserless injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_scripts has_functions has_contexts has_configurations
    has_scripts=$(echo "$config" | jq -e '.scripts' >/dev/null 2>&1 && echo "true" || echo "false")
    has_functions=$(echo "$config" | jq -e '.functions' >/dev/null 2>&1 && echo "true" || echo "false")
    has_contexts=$(echo "$config" | jq -e '.browser_contexts' >/dev/null 2>&1 && echo "true" || echo "false")
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_scripts" == "false" && "$has_functions" == "false" && "$has_contexts" == "false" && "$has_configurations" == "false" ]]; then
        log::error "Browserless injection configuration must have 'scripts', 'functions', 'browser_contexts', or 'configurations'"
        return 1
    fi
    
    # Validate scripts if present
    if [[ "$has_scripts" == "true" ]]; then
        local scripts
        scripts=$(echo "$config" | jq -c '.scripts')
        
        if ! browserless_inject::validate_scripts "$scripts"; then
            return 1
        fi
    fi
    
    # Validate functions if present
    if [[ "$has_functions" == "true" ]]; then
        local functions
        functions=$(echo "$config" | jq -c '.functions')
        
        local func_count
        func_count=$(echo "$functions" | jq 'length')
        
        for ((i=0; i<func_count; i++)); do
            local func
            func=$(echo "$functions" | jq -c ".[$i]")
            
            local name file
            name=$(echo "$func" | jq -r '.name // empty')
            file=$(echo "$func" | jq -r '.file // empty')
            
            if [[ -z "$name" ]]; then
                log::error "Function at index $i missing required 'name' field"
                return 1
            fi
            
            if [[ -z "$file" ]]; then
                log::error "Function '$name' missing required 'file' field"
                return 1
            fi
            
            # Check if file exists
            local func_path="$VROOLI_PROJECT_ROOT/$file"
            if [[ ! -f "$func_path" ]]; then
                log::error "Function file not found: $func_path"
                return 1
            fi
        done
    fi
    
    log::success "Browserless injection configuration is valid"
    return 0
}

#######################################
# Install script
# Arguments:
#   $1 - script configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
browserless_inject::install_script() {
    local script_config="$1"
    
    local name file type
    name=$(echo "$script_config" | jq -r '.name')
    file=$(echo "$script_config" | jq -r '.file')
    type=$(echo "$script_config" | jq -r '.type // "puppeteer"')
    
    log::info "Installing script: $name ($type)"
    
    # Resolve file path
    local script_path="$VROOLI_PROJECT_ROOT/$file"
    
    # Create scripts directory if it doesn't exist
    mkdir -p "$BROWSERLESS_SCRIPTS_DIR"
    
    # Copy script file
    local dest_file="${BROWSERLESS_SCRIPTS_DIR}/${name}.js"
    cp "$script_path" "$dest_file"
    
    # Create metadata file
    local metadata_file="${BROWSERLESS_SCRIPTS_DIR}/${name}.meta.json"
    cat > "$metadata_file" << EOF
{
  "name": "$name",
  "type": "$type",
  "installed": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    log::success "Installed script: $name"
    
    # Add rollback actions
    browserless_inject::add_rollback_action \
        "Remove script: $name" \
        "rm -f '${dest_file}' '${metadata_file}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Install function
# Arguments:
#   $1 - function configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
browserless_inject::install_function() {
    local func_config="$1"
    
    local name file endpoint
    name=$(echo "$func_config" | jq -r '.name')
    file=$(echo "$func_config" | jq -r '.file')
    endpoint=$(echo "$func_config" | jq -r '.endpoint // empty')
    
    log::info "Installing function: $name"
    
    # Resolve file path
    local func_path="$VROOLI_PROJECT_ROOT/$file"
    
    # Create functions directory if it doesn't exist
    mkdir -p "$BROWSERLESS_FUNCTIONS_DIR"
    
    # Copy function file
    local dest_file="${BROWSERLESS_FUNCTIONS_DIR}/${name}.js"
    cp "$func_path" "$dest_file"
    
    # Create metadata file
    local metadata_file="${BROWSERLESS_FUNCTIONS_DIR}/${name}.meta.json"
    cat > "$metadata_file" << EOF
{
  "name": "$name",
  "endpoint": "${endpoint:-/function/$name}",
  "installed": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    log::success "Installed function: $name"
    
    # Add rollback actions
    browserless_inject::add_rollback_action \
        "Remove function: $name" \
        "rm -f '${dest_file}' '${metadata_file}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Create browser context
# Arguments:
#   $1 - context configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
browserless_inject::create_context() {
    local context_config="$1"
    
    local name
    name=$(echo "$context_config" | jq -r '.name')
    
    log::info "Creating browser context: $name"
    
    # Create contexts directory
    local contexts_dir="${BROWSERLESS_DATA_DIR}/contexts"
    mkdir -p "$contexts_dir"
    
    # Save context configuration
    local context_file="${contexts_dir}/${name}.json"
    echo "$context_config" > "$context_file"
    
    log::success "Created browser context: $name"
    
    # Add rollback action
    browserless_inject::add_rollback_action \
        "Remove browser context: $name" \
        "rm -f '${context_file}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Inject scripts
# Arguments:
#   $1 - scripts configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
browserless_inject::inject_scripts() {
    local scripts_config="$1"
    
    log::info "Injecting Browserless scripts..."
    
    local script_count
    script_count=$(echo "$scripts_config" | jq 'length')
    
    if [[ "$script_count" -eq 0 ]]; then
        log::info "No scripts to inject"
        return 0
    fi
    
    local failed_scripts=()
    
    for ((i=0; i<script_count; i++)); do
        local script
        script=$(echo "$scripts_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$script" | jq -r '.name')
        
        if ! browserless_inject::install_script "$script"; then
            failed_scripts+=("$name")
        fi
    done
    
    if [[ ${#failed_scripts[@]} -eq 0 ]]; then
        log::success "All scripts injected successfully"
        return 0
    else
        log::error "Failed to inject scripts: ${failed_scripts[*]}"
        return 1
    fi
}

#######################################
# Inject functions
# Arguments:
#   $1 - functions configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
browserless_inject::inject_functions() {
    local functions_config="$1"
    
    log::info "Injecting Browserless functions..."
    
    local func_count
    func_count=$(echo "$functions_config" | jq 'length')
    
    if [[ "$func_count" -eq 0 ]]; then
        log::info "No functions to inject"
        return 0
    fi
    
    local failed_functions=()
    
    for ((i=0; i<func_count; i++)); do
        local func
        func=$(echo "$functions_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$func" | jq -r '.name')
        
        if ! browserless_inject::install_function "$func"; then
            failed_functions+=("$name")
        fi
    done
    
    if [[ ${#failed_functions[@]} -eq 0 ]]; then
        log::success "All functions injected successfully"
        return 0
    else
        log::error "Failed to inject functions: ${failed_functions[*]}"
        return 1
    fi
}

#######################################
# Inject browser contexts
# Arguments:
#   $1 - contexts configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
browserless_inject::inject_contexts() {
    local contexts_config="$1"
    
    log::info "Injecting browser contexts..."
    
    local context_count
    context_count=$(echo "$contexts_config" | jq 'length')
    
    if [[ "$context_count" -eq 0 ]]; then
        log::info "No browser contexts to inject"
        return 0
    fi
    
    local failed_contexts=()
    
    for ((i=0; i<context_count; i++)); do
        local context
        context=$(echo "$contexts_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$context" | jq -r '.name')
        
        if ! browserless_inject::create_context "$context"; then
            failed_contexts+=("$name")
        fi
    done
    
    if [[ ${#failed_contexts[@]} -eq 0 ]]; then
        log::success "All browser contexts injected successfully"
        return 0
    else
        log::error "Failed to inject browser contexts: ${failed_contexts[*]}"
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
browserless_inject::apply_configurations() {
    local configurations="$1"
    
    log::info "Applying Browserless configurations..."
    
    local config_count
    config_count=$(echo "$configurations" | jq 'length')
    
    if [[ "$config_count" -eq 0 ]]; then
        log::info "No configurations to apply"
        return 0
    fi
    
    # Save configuration settings
    local config_file="${BROWSERLESS_DATA_DIR}/config.json"
    echo "$configurations" > "$config_file"
    
    log::success "Saved configuration settings"
    
    # Add rollback action
    browserless_inject::add_rollback_action \
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
browserless_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Browserless"
    
    # Note: Browserless doesn't need to be running for file-based injection
    browserless_inject::check_accessibility || true
    
    # Clear previous rollback actions
    BROWSERLESS_ROLLBACK_ACTIONS=()
    
    # Inject scripts if present
    local has_scripts
    has_scripts=$(echo "$config" | jq -e '.scripts' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_scripts" == "true" ]]; then
        local scripts
        scripts=$(echo "$config" | jq -c '.scripts')
        
        if ! browserless_inject::inject_scripts "$scripts"; then
            log::error "Failed to inject scripts"
            browserless_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject functions if present
    local has_functions
    has_functions=$(echo "$config" | jq -e '.functions' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_functions" == "true" ]]; then
        local functions
        functions=$(echo "$config" | jq -c '.functions')
        
        if ! browserless_inject::inject_functions "$functions"; then
            log::error "Failed to inject functions"
            browserless_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject browser contexts if present
    local has_contexts
    has_contexts=$(echo "$config" | jq -e '.browser_contexts' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_contexts" == "true" ]]; then
        local contexts
        contexts=$(echo "$config" | jq -c '.browser_contexts')
        
        if ! browserless_inject::inject_contexts "$contexts"; then
            log::error "Failed to inject browser contexts"
            browserless_inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply configurations if present
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')
        
        if ! browserless_inject::apply_configurations "$configurations"; then
            log::error "Failed to apply configurations"
            browserless_inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ Browserless data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
browserless_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Browserless injection status"
    
    # Check Browserless accessibility
    local is_running=false
    if browserless_inject::check_accessibility; then
        is_running=true
        log::success "✅ Browserless is running"
    else
        log::warn "⚠️  Browserless is not running"
    fi
    
    # Check scripts
    if [[ -d "$BROWSERLESS_SCRIPTS_DIR" ]]; then
        local script_count
        script_count=$(find "$BROWSERLESS_SCRIPTS_DIR" -name "*.js" 2>/dev/null | wc -l)
        
        if [[ "$script_count" -gt 0 ]]; then
            log::info "Found $script_count scripts"
        else
            log::info "No scripts found"
        fi
    else
        log::info "Scripts directory does not exist"
    fi
    
    # Check functions
    if [[ -d "$BROWSERLESS_FUNCTIONS_DIR" ]]; then
        local func_count
        func_count=$(find "$BROWSERLESS_FUNCTIONS_DIR" -name "*.js" 2>/dev/null | wc -l)
        
        if [[ "$func_count" -gt 0 ]]; then
            log::info "Found $func_count functions"
        else
            log::info "No functions found"
        fi
    else
        log::info "Functions directory does not exist"
    fi
    
    # Check browser contexts
    local contexts_dir="${BROWSERLESS_DATA_DIR}/contexts"
    if [[ -d "$contexts_dir" ]]; then
        local context_count
        context_count=$(find "$contexts_dir" -name "*.json" 2>/dev/null | wc -l)
        
        if [[ "$context_count" -gt 0 ]]; then
            log::info "Found $context_count browser contexts"
        else
            log::info "No browser contexts found"
        fi
    fi
    
    # Test API if running
    if [[ "$is_running" == true ]]; then
        log::info "Testing Browserless API..."
        
        # Check pressure endpoint
        if curl -s "${BROWSERLESS_HOST}/pressure" 2>/dev/null | jq -e '.running' >/dev/null 2>&1; then
            local running queued
            running=$(curl -s "${BROWSERLESS_HOST}/pressure" | jq -r '.running // 0')
            queued=$(curl -s "${BROWSERLESS_HOST}/pressure" | jq -r '.queued // 0')
            
            log::success "✅ Browserless API is responding"
            log::info "  - Running sessions: $running"
            log::info "  - Queued sessions: $queued"
        else
            log::error "❌ Browserless API not responding properly"
        fi
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
browserless_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    # Handle help first, before checking for config
    if [[ "$action" == "--help" ]]; then
        browserless_inject::usage
        exit 0
    fi
    
    # Handle rollback, which doesn't require config
    if [[ "$action" == "--rollback" ]]; then
        browserless_inject::execute_rollback
        exit 0
    fi
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        browserless_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            browserless_inject::validate_config "$config"
            ;;
        "--inject")
            browserless_inject::inject_data "$config"
            ;;
        "--status")
            browserless_inject::check_status "$config"
            ;;
        *)
            log::error "Unknown action: $action"
            browserless_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        browserless_inject::usage
        exit 1
    fi
    
    browserless_inject::main "$@"
fi