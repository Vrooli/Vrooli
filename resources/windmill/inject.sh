#!/usr/bin/env bash
set -euo pipefail

# Windmill Data Injection Adapter
# This script handles injection of scripts, apps, and resources into Windmill
# Part of the Vrooli resource data injection system

export DESCRIPTION="Inject scripts, apps, and resources into Windmill workflow platform"

# Source var.sh first with relative path
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../.." && builtin pwd)}"
INJECT_SCRIPT_DIR="${APP_ROOT}/resources/windmill"
# shellcheck disable=SC1091
source "${INJECT_SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Source Windmill configuration if available
if [[ -f "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/config/defaults.sh" 2>/dev/null || true
    
    # Export configuration if function exists
    if declare -f windmill::export_config >/dev/null 2>&1; then
        windmill::export_config 2>/dev/null || true
    fi
fi

# Default Windmill API settings
if [[ -z "${WINDMILL_BASE_URL:-}" ]]; then
    WINDMILL_BASE_URL="http://localhost:5681"
fi
readonly WINDMILL_API_BASE="$WINDMILL_BASE_URL/api"

# Default workspace
readonly DEFAULT_WORKSPACE="f/vrooli"

# Operation tracking
declare -a WINDMILL_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
windmill_inject::usage() {
    cat << EOF
Windmill Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects scripts, apps, and resources into Windmill based on scenario
    configuration. Supports validation, injection, status checks, and rollback.

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
          "path": "f/vrooli/script_name",
          "file": "path/to/script.ts",
          "summary": "Script description",
          "schema": "path/to/schema.json",
          "language": "typescript"
        }
      ],
      "apps": [
        {
          "path": "f/vrooli/app_name",
          "file": "path/to/app.json",
          "summary": "App description"
        }
      ],
      "resources": [
        {
          "name": "resource_name",
          "type": "postgres",
          "config": { ... }
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"scripts": [{"path": "f/test", "file": "test.ts", "language": "typescript"}]}'
    
    # Inject data
    $0 --inject '{"scripts": [{"path": "f/test", "file": "test.ts", "summary": "Test script"}]}'

EOF
}

#######################################
# Check if Windmill is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
windmill_inject::check_accessibility() {
    if ! system::is_command "curl"; then
        log::error "curl command not available"
        log::info "Please install curl to use Windmill injection features"
        return 1
    fi
    
    if ! system::is_command "jq"; then
        log::error "jq command not available"
        log::info "Please install jq to use Windmill injection features"
        return 1
    fi
    
    # Check health endpoint
    if curl -s --max-time 5 "$WINDMILL_API_BASE/version" >/dev/null 2>&1; then
        log::debug "Windmill is accessible at $WINDMILL_BASE_URL"
        return 0
    fi
    
    log::error "Windmill is not accessible at $WINDMILL_BASE_URL"
    log::info "Ensure Windmill is running: ${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/manage.sh --action start"
    return 1
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
windmill_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    WINDMILL_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Windmill rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
windmill_inject::execute_rollback() {
    if [[ ${#WINDMILL_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Windmill rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Windmill rollback actions..."
    
    local success_count=0
    local total_count=${#WINDMILL_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#WINDMILL_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${WINDMILL_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Windmill rollback completed: $success_count/$total_count actions successful"
    WINDMILL_ROLLBACK_ACTIONS=()
}

#######################################
# Detect script language from file extension
# Arguments:
#   $1 - file path
# Returns:
#   Language string
#######################################
windmill_inject::detect_language() {
    local file_path="$1"
    local extension="${file_path##*.}"
    
    case "$extension" in
        ts|tsx)
            echo "typescript"
            ;;
        js|jsx)
            echo "javascript"
            ;;
        py)
            echo "python3"
            ;;
        sh|bash)
            echo "bash"
            ;;
        go)
            echo "go"
            ;;
        rs)
            echo "rust"
            ;;
        *)
            echo "typescript"  # Default
            ;;
    esac
}

#######################################
# Validate script configuration
# Arguments:
#   $1 - scripts configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
windmill_inject::validate_scripts() {
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
    
    for ((i=0; i<script_count; i++)); do
        local script
        script=$(echo "$scripts_config" | jq -c ".[$i]")
        
        # Check required fields
        local path file
        path=$(echo "$script" | jq -r '.path // empty')
        file=$(echo "$script" | jq -r '.file // empty')
        
        if [[ -z "$path" ]]; then
            log::error "Script at index $i missing required 'path' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "Script '$path' missing required 'file' field"
            return 1
        fi
        
        # Check if file exists
        local script_file="$var_ROOT_DIR/$file"
        if [[ ! -f "$script_file" ]]; then
            log::error "Script file not found: $script_file"
            return 1
        fi
        
        log::debug "Script '$path' configuration is valid"
    done
    
    log::success "All script configurations are valid"
    return 0
}

#######################################
# Validate app configuration
# Arguments:
#   $1 - apps configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
windmill_inject::validate_apps() {
    local apps_config="$1"
    
    log::debug "Validating app configurations..."
    
    # Check if apps is an array
    local apps_type
    apps_type=$(echo "$apps_config" | jq -r 'type')
    
    if [[ "$apps_type" != "array" ]]; then
        log::error "Apps configuration must be an array, got: $apps_type"
        return 1
    fi
    
    # Validate each app
    local app_count
    app_count=$(echo "$apps_config" | jq 'length')
    
    for ((i=0; i<app_count; i++)); do
        local app
        app=$(echo "$apps_config" | jq -c ".[$i]")
        
        # Check required fields
        local path file
        path=$(echo "$app" | jq -r '.path // empty')
        file=$(echo "$app" | jq -r '.file // empty')
        
        if [[ -z "$path" ]]; then
            log::error "App at index $i missing required 'path' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "App '$path' missing required 'file' field"
            return 1
        fi
        
        # Check if file exists
        local app_file="$var_ROOT_DIR/$file"
        if [[ ! -f "$app_file" ]]; then
            log::error "App file not found: $app_file"
            return 1
        fi
        
        # Validate app file is valid JSON
        if ! jq . "$app_file" >/dev/null 2>&1; then
            log::error "App file contains invalid JSON: $app_file"
            return 1
        fi
        
        log::debug "App '$path' configuration is valid"
    done
    
    log::success "All app configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
windmill_inject::validate_config() {
    local config="$1"
    
    log::info "Validating Windmill injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Windmill injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_scripts has_apps has_resources
    has_scripts=$(echo "$config" | jq -e '.scripts' >/dev/null 2>&1 && echo "true" || echo "false")
    has_apps=$(echo "$config" | jq -e '.apps' >/dev/null 2>&1 && echo "true" || echo "false")
    has_resources=$(echo "$config" | jq -e '.resources' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_scripts" == "false" && "$has_apps" == "false" && "$has_resources" == "false" ]]; then
        log::error "Windmill injection configuration must have 'scripts', 'apps', or 'resources'"
        return 1
    fi
    
    # Validate scripts if present
    if [[ "$has_scripts" == "true" ]]; then
        local scripts
        scripts=$(echo "$config" | jq -c '.scripts')
        
        if ! windmill_inject::validate_scripts "$scripts"; then
            return 1
        fi
    fi
    
    # Validate apps if present
    if [[ "$has_apps" == "true" ]]; then
        local apps
        apps=$(echo "$config" | jq -c '.apps')
        
        if ! windmill_inject::validate_apps "$apps"; then
            return 1
        fi
    fi
    
    # Note: Resource validation would require more complex logic
    if [[ "$has_resources" == "true" ]]; then
        log::warn "Resource injection validation not yet implemented"
    fi
    
    log::success "Windmill injection configuration is valid"
    return 0
}

#######################################
# Import script into Windmill
# Arguments:
#   $1 - script configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
windmill_inject::import_script() {
    local script_config="$1"
    
    local path file summary schema language
    path=$(echo "$script_config" | jq -r '.path')
    file=$(echo "$script_config" | jq -r '.file')
    summary=$(echo "$script_config" | jq -r '.summary // "Imported script"')
    schema=$(echo "$script_config" | jq -r '.schema // empty')
    language=$(echo "$script_config" | jq -r '.language // empty')
    
    log::info "Importing script: $path"
    
    # Resolve file path
    local script_file="$var_ROOT_DIR/$file"
    
    # Read script content
    local script_content
    script_content=$(cat "$script_file")
    
    # Detect language if not specified
    if [[ -z "$language" ]]; then
        language=$(windmill_inject::detect_language "$file")
    fi
    
    # Prepare script payload
    local payload
    payload=$(jq -n \
        --arg path "$path" \
        --arg content "$script_content" \
        --arg language "$language" \
        --arg summary "$summary" \
        '{
            path: $path,
            content: $content,
            language: $language,
            summary: $summary,
            is_template: false,
            kind: "script"
        }')
    
    # Add schema if provided
    if [[ -n "$schema" ]]; then
        local schema_file="$var_ROOT_DIR/$schema"
        if [[ -f "$schema_file" ]]; then
            local schema_content
            schema_content=$(cat "$schema_file")
            payload=$(echo "$payload" | jq --argjson schema "$schema_content" '. + {schema: $schema}')
        fi
    fi
    
    # Import script via API
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$WINDMILL_API_BASE/w/$DEFAULT_WORKSPACE/scripts/create" 2>/dev/null); then
        
        log::success "Imported script: $path"
        
        # Add rollback action
        windmill_inject::add_rollback_action \
            "Remove script: $path" \
            "curl -s -X DELETE '$WINDMILL_API_BASE/w/$DEFAULT_WORKSPACE/scripts/p/$path' >/dev/null 2>&1"
        
        return 0
    else
        log::error "Failed to import script: $path"
        return 1
    fi
}

#######################################
# Import app into Windmill
# Arguments:
#   $1 - app configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
windmill_inject::import_app() {
    local app_config="$1"
    
    local path file summary
    path=$(echo "$app_config" | jq -r '.path')
    file=$(echo "$app_config" | jq -r '.file')
    summary=$(echo "$app_config" | jq -r '.summary // "Imported app"')
    
    log::info "Importing app: $path"
    
    # Resolve file path
    local app_file="$var_ROOT_DIR/$file"
    
    # Read app configuration
    local app_content
    app_content=$(cat "$app_file")
    
    # Prepare app payload
    local payload
    payload=$(jq -n \
        --arg path "$path" \
        --argjson value "$app_content" \
        --arg summary "$summary" \
        '{
            path: $path,
            value: $value,
            summary: $summary,
            policy: {
                execution_mode: "anonymous",
                triggerables: {}
            }
        }')
    
    # Import app via API
    if curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$WINDMILL_API_BASE/w/$DEFAULT_WORKSPACE/apps/create" >/dev/null 2>&1; then
        
        log::success "Imported app: $path"
        
        # Add rollback action
        windmill_inject::add_rollback_action \
            "Remove app: $path" \
            "curl -s -X DELETE '$WINDMILL_API_BASE/w/$DEFAULT_WORKSPACE/apps/$path' >/dev/null 2>&1"
        
        return 0
    else
        log::error "Failed to import app: $path"
        return 1
    fi
}

#######################################
# Inject scripts into Windmill
# Arguments:
#   $1 - scripts configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
windmill_inject::inject_scripts() {
    local scripts_config="$1"
    
    log::info "Injecting scripts into Windmill..."
    
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
        
        local script_path
        script_path=$(echo "$script" | jq -r '.path')
        
        if ! windmill_inject::import_script "$script"; then
            failed_scripts+=("$script_path")
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
# Inject apps into Windmill
# Arguments:
#   $1 - apps configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
windmill_inject::inject_apps() {
    local apps_config="$1"
    
    log::info "Injecting apps into Windmill..."
    
    local app_count
    app_count=$(echo "$apps_config" | jq 'length')
    
    if [[ "$app_count" -eq 0 ]]; then
        log::info "No apps to inject"
        return 0
    fi
    
    local failed_apps=()
    
    for ((i=0; i<app_count; i++)); do
        local app
        app=$(echo "$apps_config" | jq -c ".[$i]")
        
        local app_path
        app_path=$(echo "$app" | jq -r '.path')
        
        if ! windmill_inject::import_app "$app"; then
            failed_apps+=("$app_path")
        fi
    done
    
    if [[ ${#failed_apps[@]} -eq 0 ]]; then
        log::success "All apps injected successfully"
        return 0
    else
        log::error "Failed to inject apps: ${failed_apps[*]}"
        return 1
    fi
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
windmill_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Windmill"
    
    # Check Windmill accessibility
    if ! windmill_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    WINDMILL_ROLLBACK_ACTIONS=()
    
    # Inject scripts if present
    local has_scripts
    has_scripts=$(echo "$config" | jq -e '.scripts' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_scripts" == "true" ]]; then
        local scripts
        scripts=$(echo "$config" | jq -c '.scripts')
        
        if ! windmill_inject::inject_scripts "$scripts"; then
            log::error "Failed to inject scripts"
            windmill_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject apps if present
    local has_apps
    has_apps=$(echo "$config" | jq -e '.apps' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_apps" == "true" ]]; then
        local apps
        apps=$(echo "$config" | jq -c '.apps')
        
        if ! windmill_inject::inject_apps "$apps"; then
            log::error "Failed to inject apps"
            windmill_inject::execute_rollback
            return 1
        fi
    fi
    
    # Note: Resource injection not yet implemented
    local has_resources
    has_resources=$(echo "$config" | jq -e '.resources' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_resources" == "true" ]]; then
        log::warn "Resource injection not yet implemented for Windmill"
    fi
    
    log::success "✅ Windmill data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
windmill_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Windmill injection status"
    
    # Check Windmill accessibility
    if ! windmill_inject::check_accessibility; then
        return 1
    fi
    
    # Check script status
    local has_scripts
    has_scripts=$(echo "$config" | jq -e '.scripts' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_scripts" == "true" ]]; then
        local scripts
        scripts=$(echo "$config" | jq -c '.scripts')
        
        log::info "Checking script status..."
        
        local script_count
        script_count=$(echo "$scripts" | jq 'length')
        
        for ((i=0; i<script_count; i++)); do
            local script
            script=$(echo "$scripts" | jq -c ".[$i]")
            
            local path
            path=$(echo "$script" | jq -r '.path')
            
            # Check if script exists in Windmill
            if curl -s "$WINDMILL_API_BASE/w/$DEFAULT_WORKSPACE/scripts/get/$path" >/dev/null 2>&1; then
                log::success "✅ Script '$path' found"
            else
                log::error "❌ Script '$path' not found"
            fi
        done
    fi
    
    # Check app status
    local has_apps
    has_apps=$(echo "$config" | jq -e '.apps' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_apps" == "true" ]]; then
        local apps
        apps=$(echo "$config" | jq -c '.apps')
        
        log::info "Checking app status..."
        
        local app_count
        app_count=$(echo "$apps" | jq 'length')
        
        for ((i=0; i<app_count; i++)); do
            local app
            app=$(echo "$apps" | jq -c ".[$i]")
            
            local path
            path=$(echo "$app" | jq -r '.path')
            
            # Check if app exists in Windmill
            if curl -s "$WINDMILL_API_BASE/w/$DEFAULT_WORKSPACE/apps/get/$path" >/dev/null 2>&1; then
                log::success "✅ App '$path' found"
            else
                log::error "❌ App '$path' not found"
            fi
        done
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
windmill_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        windmill_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            windmill_inject::validate_config "$config"
            ;;
        "--inject")
            windmill_inject::inject_data "$config"
            ;;
        "--status")
            windmill_inject::check_status "$config"
            ;;
        "--rollback")
            windmill_inject::execute_rollback
            ;;
        "--help")
            windmill_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            windmill_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        windmill_inject::usage
        exit 1
    fi
    
    windmill_inject::main "$@"
fi