#!/usr/bin/env bash
#######################################
# Help and Documentation Library
# Provides help system and phase information for the Vrooli platform
#######################################

set -euo pipefail

#######################################
# Display help with dynamic app info
# Returns:
#   0 on success
#######################################
help::show_app_help() {
    local script_name="${1:-manage.sh}"
    
    # Get app info from service.json
    local app_name app_version app_desc
    
    if json::validate_config 2>/dev/null; then
        app_name=$(json::get_value '.service.displayName // .service.name' 'Application')
        app_version=$(json::get_value '.version // .service.version' 'unknown')
        app_desc=$(json::get_value '.service.description' '')
    else
        app_name="Application"
        app_version="unknown"
        app_desc=""
    fi
    
    cat << EOF
$app_name v$app_version
$([ -n "$app_desc" ] && echo "$app_desc" && echo)

USAGE:
    $script_name <phase> [options]
    $script_name --help | -h
    $script_name --list-phases

PHASES:
    Phases are defined in .vrooli/service.json under 'lifecycle'.
    Common phases include:
    
    setup       Prepare the environment for development/production
    develop     Start the development environment  
    build       Build the application artifacts
    test        Run tests
    deploy      Deploy the application
    
    Custom phases can be defined in service.json.

OPTIONS:
    Options vary by phase. Common options include:
    
    --target <target>        Deployment target (native-linux, docker, k8s, etc.)
    --environment <env>      Environment (development, production, staging)
    --dry-run               Preview what would be executed without making changes
    --yes                   Skip confirmation prompts
    
    Run '$script_name <phase> --help' for phase-specific options.

EXAMPLES:
    $script_name setup --target native-linux
    $script_name develop --detached yes
    $script_name build --environment production --dry-run
    $script_name test --coverage yes
    $script_name deploy --source k8s --version 2.0.0

CONFIGURATION:
    Lifecycle phases and steps: $var_ROOT_DIR/.vrooli/service.json
    Project root: $var_ROOT_DIR

EOF
}

#######################################
# List available phases from service.json
# Returns:
#   0 on success, 1 on failure
#######################################
help::list_phases() {
    if ! json::validate_config; then
        log::error "Cannot list phases - invalid or missing service.json"
        return 1
    fi
    
    echo "Available lifecycle phases:"
    echo
    
    local phases
    phases=$(json::list_lifecycle_phases)
    
    if [[ -z "$phases" ]]; then
        echo "  (No lifecycle phases configured)"
        return 0
    fi
    
    while IFS= read -r phase; do
        [[ -z "$phase" ]] && continue
        local description
        description=$(json::get_value ".lifecycle.${phase}.description" "")
        printf "  %-12s  %s\n" "$phase" "$description"
    done <<< "$phases"
}