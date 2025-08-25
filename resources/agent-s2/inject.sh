#!/usr/bin/env bash
set -euo pipefail

# Agent-S2 Data Injection Adapter
# This script handles injection of configurations and automations into Agent-S2
# Part of the Vrooli resource data injection system

export DESCRIPTION="Inject configurations and automations into Agent-S2 web automation agent"

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
AGENT_S2_SCRIPT_DIR="${APP_ROOT}/resources/agent-s2"

# Source var.sh first to get proper directory variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Source Agent-S2 configuration if available
if [[ -f "${AGENT_S2_SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${AGENT_S2_SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default Agent-S2 settings  
readonly DEFAULT_AGENT_S2_HOST="http://localhost:4113"
readonly DEFAULT_AGENT_S2_DATA_DIR="${HOME}/.agent-s2"
readonly DEFAULT_AGENT_S2_PROFILES_DIR="${DEFAULT_AGENT_S2_DATA_DIR}/profiles"
readonly DEFAULT_AGENT_S2_WORKFLOWS_DIR="${DEFAULT_AGENT_S2_DATA_DIR}/workflows"

# Agent-S2 settings (can be overridden by environment)
AGENT_S2_HOST="${AGENT_S2_HOST:-$DEFAULT_AGENT_S2_HOST}"
AGENT_S2_DATA_DIR="${AGENT_S2_DATA_DIR:-$DEFAULT_AGENT_S2_DATA_DIR}"
AGENT_S2_PROFILES_DIR="${AGENT_S2_PROFILES_DIR:-$DEFAULT_AGENT_S2_PROFILES_DIR}"
AGENT_S2_WORKFLOWS_DIR="${AGENT_S2_WORKFLOWS_DIR:-$DEFAULT_AGENT_S2_WORKFLOWS_DIR}"

# Operation tracking
declare -a AGENT_S2_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
inject::usage() {
    cat << EOF
Agent-S2 Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects browser profiles, workflows, and configurations into Agent-S2 based on 
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
      "profiles": [
        {
          "name": "stealth_profile",
          "user_agent": "Mozilla/5.0...",
          "viewport": {
            "width": 1920,
            "height": 1080
          },
          "features": {
            "webgl": true,
            "canvas": true,
            "audio": true
          }
        }
      ],
      "workflows": [
        {
          "name": "search_workflow",
          "file": "path/to/workflow.json",
          "autoload": true
        }
      ],
      "proxy_configs": [
        {
          "name": "residential",
          "type": "http",
          "host": "proxy.example.com",
          "port": 8080
        }
      ],
      "configurations": [
        {
          "key": "headless",
          "value": false
        },
        {
          "key": "stealth_mode",
          "value": true
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"profiles": [{"name": "default", "user_agent": "Mozilla/5.0..."}]}'
    
    # Inject browser profiles and workflows
    $0 --inject '{"profiles": [{"name": "stealth"}], "workflows": [{"name": "search", "file": "workflows/search.json"}]}'
    
    # Configure proxy settings
    $0 --inject '{"proxy_configs": [{"name": "residential", "type": "http", "host": "proxy.com", "port": 8080}]}'

EOF
}

#######################################
# Check if Agent-S2 is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
inject::check_accessibility() {
    # Check if Agent-S2 is running
    if curl -s --max-time 5 "${AGENT_S2_HOST}/health" 2>/dev/null | grep -q "healthy"; then
        log::debug "Agent-S2 is accessible at $AGENT_S2_HOST"
        return 0
    else
        log::error "Agent-S2 is not accessible at $AGENT_S2_HOST"
        log::info "Ensure Agent-S2 is running: ./resources/agent-s2/cli.sh start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    AGENT_S2_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Agent-S2 rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
inject::execute_rollback() {
    if [[ ${#AGENT_S2_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Agent-S2 rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Agent-S2 rollback actions..."
    
    local success_count=0
    local total_count=${#AGENT_S2_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#AGENT_S2_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${AGENT_S2_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Agent-S2 rollback completed: $success_count/$total_count actions successful"
    AGENT_S2_ROLLBACK_ACTIONS=()
}

#######################################
# Validate profile configuration
# Arguments:
#   $1 - profiles configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject::validate_profiles() {
    local profiles_config="$1"
    
    log::debug "Validating profile configurations..."
    
    # Check if profiles is an array
    local profiles_type
    profiles_type=$(echo "$profiles_config" | jq -r 'type')
    
    if [[ "$profiles_type" != "array" ]]; then
        log::error "Profiles configuration must be an array, got: $profiles_type"
        return 1
    fi
    
    # Validate each profile
    local profile_count
    profile_count=$(echo "$profiles_config" | jq 'length')
    
    for ((i=0; i<profile_count; i++)); do
        local profile
        profile=$(echo "$profiles_config" | jq -c ".[$i]")
        
        # Check required fields
        local name
        name=$(echo "$profile" | jq -r '.name // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Profile at index $i missing required 'name' field"
            return 1
        fi
        
        log::debug "Profile '$name' configuration is valid"
    done
    
    log::success "All profile configurations are valid"
    return 0
}

#######################################
# Validate workflow configuration
# Arguments:
#   $1 - workflows configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject::validate_workflows() {
    local workflows_config="$1"
    
    log::debug "Validating workflow configurations..."
    
    # Check if workflows is an array
    local workflows_type
    workflows_type=$(echo "$workflows_config" | jq -r 'type')
    
    if [[ "$workflows_type" != "array" ]]; then
        log::error "Workflows configuration must be an array, got: $workflows_type"
        return 1
    fi
    
    # Validate each workflow
    local workflow_count
    workflow_count=$(echo "$workflows_config" | jq 'length')
    
    for ((i=0; i<workflow_count; i++)); do
        local workflow
        workflow=$(echo "$workflows_config" | jq -c ".[$i]")
        
        # Check required fields
        local name file
        name=$(echo "$workflow" | jq -r '.name // empty')
        file=$(echo "$workflow" | jq -r '.file // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Workflow at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "Workflow '$name' missing required 'file' field"
            return 1
        fi
        
        # Check if file exists
        local workflow_path="$var_ROOT_DIR/$file"
        if [[ ! -f "$workflow_path" ]]; then
            log::error "Workflow file not found: $workflow_path"
            return 1
        fi
        
        log::debug "Workflow '$name' configuration is valid"
    done
    
    log::success "All workflow configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject::validate_config() {
    local config="$1"
    
    log::info "Validating Agent-S2 injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Agent-S2 injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_profiles has_workflows has_proxy has_configurations
    has_profiles=$(echo "$config" | jq -e '.profiles' >/dev/null 2>&1 && echo "true" || echo "false")
    has_workflows=$(echo "$config" | jq -e '.workflows' >/dev/null 2>&1 && echo "true" || echo "false")
    has_proxy=$(echo "$config" | jq -e '.proxy_configs' >/dev/null 2>&1 && echo "true" || echo "false")
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_profiles" == "false" && "$has_workflows" == "false" && "$has_proxy" == "false" && "$has_configurations" == "false" ]]; then
        log::error "Agent-S2 injection configuration must have 'profiles', 'workflows', 'proxy_configs', or 'configurations'"
        return 1
    fi
    
    # Validate profiles if present
    if [[ "$has_profiles" == "true" ]]; then
        local profiles
        profiles=$(echo "$config" | jq -c '.profiles')
        
        if ! inject::validate_profiles "$profiles"; then
            return 1
        fi
    fi
    
    # Validate workflows if present
    if [[ "$has_workflows" == "true" ]]; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        
        if ! inject::validate_workflows "$workflows"; then
            return 1
        fi
    fi
    
    log::success "Agent-S2 injection configuration is valid"
    return 0
}

#######################################
# Create browser profile
# Arguments:
#   $1 - profile configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::create_profile() {
    local profile_config="$1"
    
    local name
    name=$(echo "$profile_config" | jq -r '.name')
    
    log::info "Creating browser profile: $name"
    
    # Create profiles directory if it doesn't exist
    mkdir -p "$AGENT_S2_PROFILES_DIR"
    
    # Save profile configuration
    local profile_file="${AGENT_S2_PROFILES_DIR}/${name}.json"
    echo "$profile_config" > "$profile_file"
    
    # Send profile to Agent-S2 if it's running
    if inject::check_accessibility; then
        curl -s -X POST "${AGENT_S2_HOST}/api/profiles" \
            -H "Content-Type: application/json" \
            -d "@${profile_file}" >/dev/null 2>&1 || true
    fi
    
    log::success "Created browser profile: $name"
    
    # Add rollback action
    inject::add_rollback_action \
        "Remove browser profile: $name" \
        "rm -f '${profile_file}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Install workflow
# Arguments:
#   $1 - workflow configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::install_workflow() {
    local workflow_config="$1"
    
    local name file autoload
    name=$(echo "$workflow_config" | jq -r '.name')
    file=$(echo "$workflow_config" | jq -r '.file')
    autoload=$(echo "$workflow_config" | jq -r '.autoload // false')
    
    log::info "Installing workflow: $name"
    
    # Resolve file path
    local workflow_path="$var_ROOT_DIR/$file"
    
    # Create workflows directory if it doesn't exist
    mkdir -p "$AGENT_S2_WORKFLOWS_DIR"
    
    # Copy workflow file
    local dest_file="${AGENT_S2_WORKFLOWS_DIR}/${name}.json"
    cp "$workflow_path" "$dest_file"
    
    # Create metadata file
    local metadata_file="${AGENT_S2_WORKFLOWS_DIR}/${name}.meta.json"
    cat > "$metadata_file" << EOF
{
  "name": "$name",
  "autoload": $autoload,
  "installed": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    # Load workflow if Agent-S2 is running and autoload is enabled
    if [[ "$autoload" == "true" ]] && inject::check_accessibility; then
        curl -s -X POST "${AGENT_S2_HOST}/api/workflows/load" \
            -H "Content-Type: application/json" \
            -d "{\"workflow\": \"$name\"}" >/dev/null 2>&1 || true
    fi
    
    log::success "Installed workflow: $name"
    
    # Add rollback actions
    inject::add_rollback_action \
        "Remove workflow: $name" \
        "rm -f '${dest_file}' '${metadata_file}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Configure proxy
# Arguments:
#   $1 - proxy configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::configure_proxy() {
    local proxy_config="$1"
    
    local name
    name=$(echo "$proxy_config" | jq -r '.name')
    
    log::info "Configuring proxy: $name"
    
    # Create proxy configs directory
    local proxy_dir="${AGENT_S2_DATA_DIR}/proxies"
    mkdir -p "$proxy_dir"
    
    # Save proxy configuration
    local proxy_file="${proxy_dir}/${name}.json"
    echo "$proxy_config" > "$proxy_file"
    
    # Apply proxy if Agent-S2 is running
    if inject::check_accessibility; then
        curl -s -X POST "${AGENT_S2_HOST}/api/proxy/configure" \
            -H "Content-Type: application/json" \
            -d "@${proxy_file}" >/dev/null 2>&1 || true
    fi
    
    log::success "Configured proxy: $name"
    
    # Add rollback action
    inject::add_rollback_action \
        "Remove proxy configuration: $name" \
        "rm -f '${proxy_file}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Inject profiles
# Arguments:
#   $1 - profiles configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_profiles() {
    local profiles_config="$1"
    
    log::info "Injecting Agent-S2 browser profiles..."
    
    local profile_count
    profile_count=$(echo "$profiles_config" | jq 'length')
    
    if [[ "$profile_count" -eq 0 ]]; then
        log::info "No profiles to inject"
        return 0
    fi
    
    local failed_profiles=()
    
    for ((i=0; i<profile_count; i++)); do
        local profile
        profile=$(echo "$profiles_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$profile" | jq -r '.name')
        
        if ! inject::create_profile "$profile"; then
            failed_profiles+=("$name")
        fi
    done
    
    if [[ ${#failed_profiles[@]} -eq 0 ]]; then
        log::success "All profiles injected successfully"
        return 0
    else
        log::error "Failed to inject profiles: ${failed_profiles[*]}"
        return 1
    fi
}

#######################################
# Inject workflows
# Arguments:
#   $1 - workflows configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_workflows() {
    local workflows_config="$1"
    
    log::info "Injecting Agent-S2 workflows..."
    
    local workflow_count
    workflow_count=$(echo "$workflows_config" | jq 'length')
    
    if [[ "$workflow_count" -eq 0 ]]; then
        log::info "No workflows to inject"
        return 0
    fi
    
    local failed_workflows=()
    
    for ((i=0; i<workflow_count; i++)); do
        local workflow
        workflow=$(echo "$workflows_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$workflow" | jq -r '.name')
        
        if ! inject::install_workflow "$workflow"; then
            failed_workflows+=("$name")
        fi
    done
    
    if [[ ${#failed_workflows[@]} -eq 0 ]]; then
        log::success "All workflows injected successfully"
        return 0
    else
        log::error "Failed to inject workflows: ${failed_workflows[*]}"
        return 1
    fi
}

#######################################
# Inject proxy configurations
# Arguments:
#   $1 - proxy configs JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_proxies() {
    local proxies_config="$1"
    
    log::info "Injecting Agent-S2 proxy configurations..."
    
    local proxy_count
    proxy_count=$(echo "$proxies_config" | jq 'length')
    
    if [[ "$proxy_count" -eq 0 ]]; then
        log::info "No proxy configurations to inject"
        return 0
    fi
    
    local failed_proxies=()
    
    for ((i=0; i<proxy_count; i++)); do
        local proxy
        proxy=$(echo "$proxies_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$proxy" | jq -r '.name')
        
        if ! inject::configure_proxy "$proxy"; then
            failed_proxies+=("$name")
        fi
    done
    
    if [[ ${#failed_proxies[@]} -eq 0 ]]; then
        log::success "All proxy configurations injected successfully"
        return 0
    else
        log::error "Failed to inject proxy configurations: ${failed_proxies[*]}"
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
inject::apply_configurations() {
    local configurations="$1"
    
    log::info "Applying Agent-S2 configurations..."
    
    local config_count
    config_count=$(echo "$configurations" | jq 'length')
    
    if [[ "$config_count" -eq 0 ]]; then
        log::info "No configurations to apply"
        return 0
    fi
    
    # Save configuration settings
    local config_file="${AGENT_S2_DATA_DIR}/config.json"
    echo "$configurations" > "$config_file"
    
    # Apply configurations if Agent-S2 is running
    if inject::check_accessibility; then
        for ((i=0; i<config_count; i++)); do
            local config_item
            config_item=$(echo "$configurations" | jq -c ".[$i]")
            
            curl -s -X POST "${AGENT_S2_HOST}/api/config" \
                -H "Content-Type: application/json" \
                -d "$config_item" >/dev/null 2>&1 || true
        done
    fi
    
    log::success "Applied configuration settings"
    
    # Add rollback action
    inject::add_rollback_action \
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
inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Agent-S2"
    
    # Note: Agent-S2 might not need to be running for file-based injection
    # but we check anyway for API-based operations
    inject::check_accessibility || true
    
    # Clear previous rollback actions
    AGENT_S2_ROLLBACK_ACTIONS=()
    
    # Inject profiles if present
    local has_profiles
    has_profiles=$(echo "$config" | jq -e '.profiles' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_profiles" == "true" ]]; then
        local profiles
        profiles=$(echo "$config" | jq -c '.profiles')
        
        if ! inject::inject_profiles "$profiles"; then
            log::error "Failed to inject profiles"
            inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject workflows if present
    local has_workflows
    has_workflows=$(echo "$config" | jq -e '.workflows' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_workflows" == "true" ]]; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        
        if ! inject::inject_workflows "$workflows"; then
            log::error "Failed to inject workflows"
            inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject proxy configurations if present
    local has_proxy
    has_proxy=$(echo "$config" | jq -e '.proxy_configs' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_proxy" == "true" ]]; then
        local proxies
        proxies=$(echo "$config" | jq -c '.proxy_configs')
        
        if ! inject::inject_proxies "$proxies"; then
            log::error "Failed to inject proxy configurations"
            inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply configurations if present
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')
        
        if ! inject::apply_configurations "$configurations"; then
            log::error "Failed to apply configurations"
            inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ Agent-S2 data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Agent-S2 injection status"
    
    # Check Agent-S2 accessibility
    local is_running=false
    if inject::check_accessibility; then
        is_running=true
        log::success "✅ Agent-S2 is running"
    else
        log::warn "⚠️  Agent-S2 is not running"
    fi
    
    # Check profiles
    if [[ -d "$AGENT_S2_PROFILES_DIR" ]]; then
        local profile_count
        profile_count=$(find "$AGENT_S2_PROFILES_DIR" -name "*.json" -not -name "*.meta.json" 2>/dev/null | wc -l)
        
        if [[ "$profile_count" -gt 0 ]]; then
            log::info "Found $profile_count browser profiles"
        else
            log::info "No browser profiles found"
        fi
    else
        log::info "Profiles directory does not exist"
    fi
    
    # Check workflows
    if [[ -d "$AGENT_S2_WORKFLOWS_DIR" ]]; then
        local workflow_count
        workflow_count=$(find "$AGENT_S2_WORKFLOWS_DIR" -name "*.json" -not -name "*.meta.json" 2>/dev/null | wc -l)
        
        if [[ "$workflow_count" -gt 0 ]]; then
            log::info "Found $workflow_count workflows"
            
            # Check for autoload workflows
            local autoload_count
            autoload_count=$(grep -l '"autoload": true' "${AGENT_S2_WORKFLOWS_DIR}"/*.meta.json 2>/dev/null | wc -l)
            
            if [[ "$autoload_count" -gt 0 ]]; then
                log::info "  - $autoload_count workflows set to autoload"
            fi
        else
            log::info "No workflows found"
        fi
    else
        log::info "Workflows directory does not exist"
    fi
    
    # Check proxy configurations
    local proxy_dir="${AGENT_S2_DATA_DIR}/proxies"
    if [[ -d "$proxy_dir" ]]; then
        local proxy_count
        proxy_count=$(find "$proxy_dir" -name "*.json" 2>/dev/null | wc -l)
        
        if [[ "$proxy_count" -gt 0 ]]; then
            log::info "Found $proxy_count proxy configurations"
        else
            log::info "No proxy configurations found"
        fi
    fi
    
    # Test API if running
    if [[ "$is_running" == true ]]; then
        log::info "Testing Agent-S2 API..."
        
        if curl -s "${AGENT_S2_HOST}/api/status" 2>/dev/null | jq -e '.status' >/dev/null 2>&1; then
            log::success "✅ Agent-S2 API is responding"
        else
            log::error "❌ Agent-S2 API not responding properly"
        fi
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            inject::validate_config "$config"
            ;;
        "--inject")
            inject::inject_data "$config"
            ;;
        "--status")
            inject::check_status "$config"
            ;;
        "--rollback")
            inject::execute_rollback
            ;;
        "--help")
            inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        inject::usage
        exit 1
    fi
    
    inject::main "$@"
fi