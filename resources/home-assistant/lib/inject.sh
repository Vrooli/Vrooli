#!/bin/bash
# Home Assistant Injection Functions

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HOME_ASSISTANT_INJECT_DIR="${APP_ROOT}/resources/home-assistant/lib"

# Source dependencies
source "${HOME_ASSISTANT_INJECT_DIR}/core.sh"
source "${HOME_ASSISTANT_INJECT_DIR}/health.sh"

#######################################
# Inject automation configuration into Home Assistant
# Arguments:
#   file_path: Path to YAML file with automation
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::inject() {
    local file_path="$1"
    
    if [[ -z "$file_path" ]]; then
        log::error "No file path provided"
        return 1
    fi
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    # Initialize environment
    home_assistant::init
    
    # Check if Home Assistant is running
    if ! home_assistant::health::is_healthy; then
        log::error "Home Assistant is not healthy. Please start it first."
        return 1
    fi
    
    local file_extension="${file_path##*.}"
    local file_name=$(basename "$file_path")
    
    case "$file_extension" in
        yaml|yml)
            log::info "Injecting YAML automation: $file_name"
            home_assistant::inject::yaml "$file_path"
            ;;
        json)
            log::info "Injecting JSON configuration: $file_name"
            home_assistant::inject::json "$file_path"
            ;;
        py)
            log::info "Injecting Python script: $file_name"
            home_assistant::inject::python "$file_path"
            ;;
        *)
            log::error "Unsupported file type: .$file_extension"
            log::info "Supported types: .yaml, .yml, .json, .py"
            return 1
            ;;
    esac
}

#######################################
# Inject YAML automation
# Arguments:
#   file_path: Path to YAML file
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::inject::yaml() {
    local file_path="$1"
    local file_name=$(basename "$file_path")
    
    # Determine target directory based on content
    local target_dir="$HOME_ASSISTANT_CONFIG_DIR/automations"
    
    # Check if it's a package file
    if grep -q "^[a-z_]*:" "$file_path" | head -1 | grep -qE "^(automation|script|scene|sensor|binary_sensor|switch|light|climate|cover|fan|lock|media_player|notify):"; then
        target_dir="$HOME_ASSISTANT_CONFIG_DIR/packages"
    fi
    
    # Create target directory
    mkdir -p "$target_dir"
    
    # Copy file to Home Assistant config
    if cp "$file_path" "$target_dir/$file_name"; then
        log::success "Injected YAML file to: $target_dir/$file_name"
        
        # Reload automations if it's an automation file
        if [[ "$target_dir" == *"automations"* ]]; then
            home_assistant::reload_automations
        fi
        
        return 0
    else
        log::error "Failed to copy YAML file"
        return 1
    fi
}

#######################################
# Inject JSON configuration
# Arguments:
#   file_path: Path to JSON file
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::inject::json() {
    local file_path="$1"
    local file_name=$(basename "$file_path")
    
    # Store JSON configs in a dedicated directory
    local target_dir="$HOME_ASSISTANT_CONFIG_DIR/json_configs"
    mkdir -p "$target_dir"
    
    # Copy file
    if cp "$file_path" "$target_dir/$file_name"; then
        log::success "Injected JSON configuration to: $target_dir/$file_name"
        
        # Try to convert to YAML if it's an automation
        if jq -e '.automation' "$file_path" >/dev/null 2>&1; then
            log::info "Detected automation in JSON, converting to YAML..."
            home_assistant::convert_json_to_yaml "$file_path" "$HOME_ASSISTANT_CONFIG_DIR/automations/${file_name%.json}.yaml"
        fi
        
        return 0
    else
        log::error "Failed to copy JSON file"
        return 1
    fi
}

#######################################
# Inject Python script
# Arguments:
#   file_path: Path to Python file
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::inject::python() {
    local file_path="$1"
    local file_name=$(basename "$file_path")
    
    # Python scripts go to the python_scripts directory
    local target_dir="$HOME_ASSISTANT_CONFIG_DIR/python_scripts"
    mkdir -p "$target_dir"
    
    # Copy file
    if cp "$file_path" "$target_dir/$file_name"; then
        log::success "Injected Python script to: $target_dir/$file_name"
        
        # Enable python_script integration if not already enabled
        home_assistant::enable_python_scripts
        
        return 0
    else
        log::error "Failed to copy Python script"
        return 1
    fi
}

#######################################
# List injected files
# Returns: List of injected files
#######################################
home_assistant::inject::list() {
    home_assistant::init
    
    log::header "Injected Home Assistant Files"
    
    # List automations
    if [[ -d "$HOME_ASSISTANT_CONFIG_DIR/automations" ]]; then
        local count=$(find "$HOME_ASSISTANT_CONFIG_DIR/automations" -type f -name "*.yaml" -o -name "*.yml" | wc -l)
        log::info "📋 Automations: $count files"
        find "$HOME_ASSISTANT_CONFIG_DIR/automations" -type f -name "*.yaml" -o -name "*.yml" -exec basename {} \; | sed 's/^/    - /'
    fi
    
    # List packages
    if [[ -d "$HOME_ASSISTANT_CONFIG_DIR/packages" ]]; then
        local count=$(find "$HOME_ASSISTANT_CONFIG_DIR/packages" -type f -name "*.yaml" -o -name "*.yml" | wc -l)
        log::info "📦 Packages: $count files"
        find "$HOME_ASSISTANT_CONFIG_DIR/packages" -type f -name "*.yaml" -o -name "*.yml" -exec basename {} \; | sed 's/^/    - /'
    fi
    
    # List Python scripts
    if [[ -d "$HOME_ASSISTANT_CONFIG_DIR/python_scripts" ]]; then
        local count=$(find "$HOME_ASSISTANT_CONFIG_DIR/python_scripts" -type f -name "*.py" | wc -l)
        log::info "🐍 Python Scripts: $count files"
        find "$HOME_ASSISTANT_CONFIG_DIR/python_scripts" -type f -name "*.py" -exec basename {} \; | sed 's/^/    - /'
    fi
    
    # List JSON configs
    if [[ -d "$HOME_ASSISTANT_CONFIG_DIR/json_configs" ]]; then
        local count=$(find "$HOME_ASSISTANT_CONFIG_DIR/json_configs" -type f -name "*.json" | wc -l)
        log::info "📄 JSON Configs: $count files"
        find "$HOME_ASSISTANT_CONFIG_DIR/json_configs" -type f -name "*.json" -exec basename {} \; | sed 's/^/    - /'
    fi
}

#######################################
# Clear all injected files
# Arguments:
#   --force: Force clear without confirmation
# Returns: 0 on success
#######################################
home_assistant::inject::clear() {
    local force="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$force" != "true" ]]; then
        log::error "Clear requires --force flag for safety"
        return 1
    fi
    
    home_assistant::init
    
    log::warning "Clearing all injected files..."
    
    # Clear directories
    rm -rf "$HOME_ASSISTANT_CONFIG_DIR/automations"/*
    rm -rf "$HOME_ASSISTANT_CONFIG_DIR/packages"/*
    rm -rf "$HOME_ASSISTANT_CONFIG_DIR/python_scripts"/*
    rm -rf "$HOME_ASSISTANT_CONFIG_DIR/json_configs"/*
    
    log::success "Cleared all injected files"
    
    # Reload Home Assistant if running
    if home_assistant::health::is_healthy; then
        home_assistant::reload_automations
    fi
    
    return 0
}

#######################################
# Helper: Reload automations
#######################################
home_assistant::reload_automations() {
    log::info "Reloading Home Assistant automations..."
    
    # This requires authentication, so we'll just restart the container for now
    # In production, you'd use the API with proper auth
    docker restart "$HOME_ASSISTANT_CONTAINER_NAME" >/dev/null 2>&1
    
    # Wait for it to come back up
    home_assistant::health::wait_for_healthy 30
}

#######################################
# Helper: Convert JSON to YAML
#######################################
home_assistant::convert_json_to_yaml() {
    local json_file="$1"
    local yaml_file="$2"
    
    # This is a simplified conversion - real implementation would need proper YAML library
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "
import json
import yaml
import sys

with open('$json_file', 'r') as f:
    data = json.load(f)

with open('$yaml_file', 'w') as f:
    yaml.dump(data, f, default_flow_style=False)
" 2>/dev/null
    fi
}

#######################################
# Helper: Enable Python scripts integration
#######################################
home_assistant::enable_python_scripts() {
    local config_file="$HOME_ASSISTANT_CONFIG_DIR/configuration.yaml"
    
    # Check if python_script is already enabled
    if ! grep -q "^python_script:" "$config_file" 2>/dev/null; then
        log::info "Enabling python_script integration..."
        echo -e "\n# Enable Python scripts\npython_script:" >> "$config_file"
        
        # Restart to apply configuration
        docker restart "$HOME_ASSISTANT_CONTAINER_NAME" >/dev/null 2>&1
        home_assistant::health::wait_for_healthy 30
    fi
}

# Export functions
export -f home_assistant::inject
export -f home_assistant::inject::yaml
export -f home_assistant::inject::json
export -f home_assistant::inject::python
export -f home_assistant::inject::list
export -f home_assistant::inject::clear