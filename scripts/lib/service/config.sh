#!/usr/bin/env bash
# This script initializes .vrooli configuration files from examples
# It handles environment variable substitution and smart defaults

LIB_SERVICE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${LIB_SERVICE_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_REPOSITORY_FILE}"

# Simple environment check function (universal replacement for env::in_development)
config::is_development() {
    local environment="${1:-${ENVIRONMENT:-development}}"
    environment=$(echo "$environment" | tr '[:upper:]' '[:lower:]')
    [[ "$environment" == "development" || "$environment" == "dev" ]]
}

config::init() {
    log::header "Initializing .vrooli configuration files..."
    
    local vrooli_dir="${var_VROOLI_CONFIG_DIR}"
    local config_created=false
    
    # Ensure .vrooli directory exists
    if [[ ! -d "${vrooli_dir}" ]]; then
        log::info "Creating .vrooli directory..."
        mkdir -p "${vrooli_dir}"
    fi
    
    # Check for old configuration files and warn user
    if [[ -f "${vrooli_dir}/resources.local.json" ]] || [[ -f "${vrooli_dir}/execution.local.json" ]]; then
        log::warning "Found old configuration files (resources.local.json/execution.local.json)"
        log::info "These files have been replaced by the unified service.json format."
        log::info "Please manually remove the old files after confirming service.json is working correctly."
    fi
    
    # Initialize service config (unified configuration)
    if config::init_service_config "${vrooli_dir}"; then
        config_created=true
    fi
    
    if [[ "${config_created}" == "true" ]]; then
        log::success "✅ Configuration files initialized"
        log::info "Please review and customize the generated config files in .vrooli/"
    else
        log::info "All configuration files already exist"
    fi
    
    # Handle repository configuration and hooks
    config::handle_repository_setup
    
    return 0
}

config::init_service_config() {
    local vrooli_dir="$1"
    local example_file="${var_EXAMPLES_DIR}/complete/ai-assistant.service.json"
    local local_file="${var_SERVICE_JSON_FILE}"
    
    # Skip if service.json already exists
    if [[ -f "${local_file}" ]]; then
        log::info "service.json already exists, skipping"
        return 1
    fi
    
    # Check if example file exists
    if [[ ! -f "${example_file}" ]]; then
        log::warning "service example not found at ${example_file}, skipping service config"
        return 1
    fi
    
    log::info "Creating service.json from example..."
    
    # Copy the example file
    cp "${example_file}" "${local_file}"
    
    # Process environment variable placeholders
    local has_missing_vars=false
    local missing_vars=()
    
    # Find all \${VAR_NAME} placeholders
    local placeholders=$(grep -o '\${[^}]*}' "${local_file}" | sort -u || true)
    
    if [[ -n "${placeholders}" ]]; then
        log::info "Processing environment variable placeholders..."
        
        while IFS= read -r placeholder; do
            # Extract variable name (remove \${ and })
            local var_name="${placeholder:2:-1}"
            local var_value="${!var_name:-}"
            
            if [[ -n "${var_value}" ]]; then
                # Replace placeholder with actual value
                # Use | as delimiter since values might contain /
                sed -i "s|${placeholder}|${var_value}|g" "${local_file}"
                log::info "Substituted ${var_name}"
            else
                has_missing_vars=true
                missing_vars+=("${var_name}")
            fi
        done <<< "${placeholders}"
    fi
    
    # Handle missing environment variables
    if [[ "${has_missing_vars}" == "true" ]]; then
        log::warning "Some environment variables are not set: ${missing_vars[*]}"
        log::info "Resources requiring these variables will remain disabled"
        
        # Create a comment in the service config about missing vars
        if command -v jq &> /dev/null; then
            local comment=$(printf "Missing environment variables: %s. Set these in your .env file to enable related resources." "${missing_vars[*]}")
            jq --arg comment "$comment" '._warning = $comment | . as $orig | {_warning: ._warning} + ($orig | del(._warning))' "${local_file}" > "${local_file}.tmp" && \
                mv "${local_file}.tmp" "${local_file}"
        fi
    fi
    
    # For development environment, enable by default with safe settings
    if config::is_development; then
        log::info "Development environment detected - enabling local execution and checking for running services"
        
        # Use jq if available for complex JSON manipulation
        if command -v jq &> /dev/null; then
            # Enable execution with safe defaults
            jq '.execution.enabled = true | .execution.environments.development.runtimes.languages |= map(if .name == "javascript" then .enabled = true else . end)' "${local_file}" > "${local_file}.tmp" && \
                mv "${local_file}.tmp" "${local_file}"
            
            # Check if Ollama is running and enable it
            if curl -s http://localhost:11434/api/tags &> /dev/null; then
                log::info "Ollama detected - enabling in config"
                jq '.resources.ai.ollama.enabled = true' "${local_file}" > "${local_file}.tmp" && \
                    mv "${local_file}.tmp" "${local_file}"
            else
                log::info "Ollama not detected - you can install it with: ./scripts/resources/ai/ollama/manage.sh --action install"
            fi
            
            # Could add more service detection here...
        else
            # Fallback to sed (less robust but works)
            sed -i 's/"enabled": false/"enabled": true/' "${local_file}"
        fi
        
        log::success "Enabled local execution for development"
        log::warning "⚠️  Local code execution is now enabled. Review security settings in service.json"
    else
        log::info "Non-development environment - execution remains disabled by default"
        log::info "To enable, edit service.json and set 'execution.enabled: true'"
    fi
    
    return 0
}

config::handle_repository_setup() {
    log::header "Configuring repository settings..."
    
    # Check if we're in a git repository
    if [[ -d "${var_ROOT_DIR}/.git" ]]; then
        log::info "Git repository detected"
        
        # Display repository information
        if repository::info; then
            log::success "Repository configuration loaded"
        fi
        
        # If this is a fresh clone (checking for node_modules as indicator)
        if [[ ! -d "${var_ROOT_DIR}/node_modules" ]]; then
            log::info "Fresh repository detected - running postClone hook"
            if repository::run_hook "postClone"; then
                log::success "PostClone hook completed"
            else
                log::warning "PostClone hook failed - you may need to run setup commands manually"
            fi
        fi
        
        # Configure git remotes for mirrors if available
        local mirrors
        mirrors=$(repository::get_mirrors)
        if [[ -n "$mirrors" ]]; then
            log::info "Configuring mirror remotes..."
            local index=1
            for mirror in $mirrors; do
                local remote_name="mirror${index}"
                if ! git remote get-url "$remote_name" &>/dev/null; then
                    git remote add "$remote_name" "$mirror"
                    log::success "Added remote '$remote_name': $mirror"
                else
                    log::info "Remote '$remote_name' already exists"
                fi
                ((index++))
            done
        fi
        
        # Set up branch tracking
        local branch
        branch=$(repository::get_branch)
        local current_branch
        current_branch=$(git branch --show-current 2>/dev/null || echo "")
        
        if [[ -n "$current_branch" && "$current_branch" != "$branch" ]]; then
            log::warning "Current branch ($current_branch) differs from configured branch ($branch)"
            log::info "You may want to switch branches: git checkout $branch"
        fi
    else
        log::info "Not in a git repository - skipping repository configuration"
    fi
}


# If this script is run directly, invoke its main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    config::init "$@"
fi