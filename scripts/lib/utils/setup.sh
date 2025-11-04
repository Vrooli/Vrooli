#!/usr/bin/env bash
#######################################
# Setup State Management Library
# Provides setup state tracking and condition checking for Vrooli platform
#######################################

set -euo pipefail

# Global variables for setup reasons
SETUP_REASONS=()

#######################################
# Check if app needs setup based on service.json conditions
# Sets SETUP_REASONS global array with specific reasons
#
# Logic: Runs checks defined in service.json lifecycle.setup.condition.checks
# Returns:
#   0 if setup is needed
#   1 if setup is not needed
#######################################
setup::is_needed() {
    # Accept optional path parameter for service.json location
    local check_path="${1:-$APP_ROOT}"

    # Reset global array for setup reasons
    SETUP_REASONS=()

    # Track if force setup is active
    local force_setup="${FORCE_SETUP:-false}"
    if [[ "$force_setup" == "true" ]]; then
        log::debug "FORCE_SETUP=true, forcing setup verification"
        SETUP_REASONS+=("Forced rebuild (restart)")
    fi

    # Get service.json path
    local service_json="${check_path}/.vrooli/service.json"

    if [[ ! -f "$service_json" ]]; then
        log::debug "No service.json found at $check_path, assuming setup not needed"
        # Unless FORCE_SETUP is true
        [[ "$force_setup" == "true" ]] && return 0
        return 1
    fi

    # Check if setup has a condition defined
    local has_condition
    has_condition=$(jq -r '.lifecycle.setup.condition // empty' "$service_json" 2>/dev/null)

    if [[ -z "$has_condition" ]]; then
        log::debug "No setup condition defined, setup not needed"
        # Unless FORCE_SETUP is true
        [[ "$force_setup" == "true" ]] && return 0
        return 1
    fi

    # Check for checks array
    local checks
    checks=$(jq -c '.lifecycle.setup.condition.checks // []' "$service_json" 2>/dev/null)

    if [[ "$checks" == "[]" ]]; then
        log::debug "No setup checks defined, setup not needed"
        # Unless FORCE_SETUP is true
        [[ "$force_setup" == "true" ]] && return 0
        return 1
    fi

    # Run each check
    local setup_needed="$force_setup"  # Initialize to force_setup value
    local check_count=0
    
    while IFS= read -r check; do
        ((check_count++))
        
        local check_type
        check_type=$(echo "$check" | jq -r '.type // empty')
        
        if [[ -z "$check_type" ]]; then
            log::warn "Check #$check_count has no type, skipping"
            continue
        fi
        
        # Map check types to checker scripts
        local checker_script=""
        case "$check_type" in
            binaries)
                checker_script="scripts/lib/setup-conditions/binaries-check.sh"
                ;;
            cli)
                checker_script="scripts/lib/setup-conditions/cli-check.sh"
                ;;
            ui-bundle)
                checker_script="scripts/lib/setup-conditions/ui-bundle-check.sh"
                ;;
            resources)
                checker_script="scripts/lib/setup-conditions/resources-check.sh"
                ;;
            dependencies)
                checker_script="scripts/lib/setup-conditions/dependencies-check.sh"
                ;;
            data)
                checker_script="scripts/lib/setup-conditions/data-check.sh"
                ;;
            files)
                checker_script="scripts/lib/setup-conditions/files-check.sh"
                ;;
            directories)
                checker_script="scripts/lib/setup-conditions/directories-check.sh"
                ;;
            *)
                # Try custom check script
                checker_script="scripts/lib/setup-conditions/${check_type}-check.sh"
                ;;
        esac
        
        # Resolve checker script path
        if [[ ! "$checker_script" =~ ^/ ]]; then
# Use VROOLI_ROOT for checker scripts since they're part of the CLI
            checker_script="${VROOLI_ROOT:-$APP_ROOT}/$checker_script"
        fi
        
        if [[ ! -f "$checker_script" ]]; then
            log::warn "Checker script not found: $checker_script (VROOLI_ROOT=${VROOLI_ROOT:-not set}, APP_ROOT=${APP_ROOT:-not set})"
            continue
        fi
        
        # Run the check (returns 0 if setup needed, 1 if not)
        if "$checker_script" "$check" 2>/dev/null; then
            log::debug "Check '$check_type' indicates setup is needed"
            
            # Add descriptive reason based on check type
            case "$check_type" in
                binaries)
                    local targets
                    targets=$(echo "$check" | jq -r '.targets[]?' 2>/dev/null | head -3 | paste -sd, -)
                    SETUP_REASONS+=("Missing binaries: $targets")
                    ;;
                cli)
                    local cmd
                    cmd=$(echo "$check" | jq -r '.command // "unknown"')
                    SETUP_REASONS+=("CLI not installed: $cmd")
                    ;;
                ui-bundle)
                    local bundle_path
                    bundle_path=$(echo "$check" | jq -r '.bundle_path // "ui/dist/index.html"')
                    SETUP_REASONS+=("UI bundle outdated: $bundle_path")
                    ;;
                resources)
                    SETUP_REASONS+=("Resources not populated")
                    ;;
                dependencies)
                    SETUP_REASONS+=("Dependencies not installed")
                    ;;
                data)
                    SETUP_REASONS+=("Data directory missing")
                    ;;
                files)
                    SETUP_REASONS+=("Required files missing")
                    ;;
                directories)
                    local targets
                    targets=$(echo "$check" | jq -r '.targets[]?' 2>/dev/null | head -3 | paste -sd, -)
                    SETUP_REASONS+=("Missing directories: $targets")
                    ;;
                *)
                    SETUP_REASONS+=("Check failed: $check_type")
                    ;;
            esac
            
            setup_needed=true
        else
            log::debug "Check '$check_type' passed"
        fi
    done <<< "$(echo "$checks" | jq -c '.[]' 2>/dev/null)"
    
    if [[ "$setup_needed" == "true" ]]; then
        log::debug "Setup needed based on condition checks"
        return 0
    else
        log::debug "All setup checks passed, no setup needed"
        return 1
    fi
}

#######################################
# Get completed setup steps from service.json for state tracking
# Returns:
#   JSON array of setup step names
#######################################
setup::get_steps_list() {
    local steps
    steps=$(json::get_value ".lifecycle.setup.steps" "[]" 2>/dev/null || echo "[]")
    
    if [[ "$steps" == "[]" ]]; then
        echo "[]"
        return 0
    fi
    
    echo "$steps" | jq -c '[.[].name // "unnamed"]' 2>/dev/null || echo "[]"
}

#######################################
# Mark setup as complete with markers for resource population
# This helps the condition checks know setup has been run
#######################################
setup::mark_complete() {
    local data_dir="${APP_ROOT}/data"
    mkdir -p "$data_dir"
    
    # Create a general setup completion marker
    local setup_steps
    setup_steps=$(setup::get_steps_list)
    
    cat > "$data_dir/.setup-complete" << EOF
{
  "setup_version": "2.0.0",
  "completed_at": "$(date -Iseconds)",
  "steps_completed": $setup_steps
}
EOF
    
    # Also create resource population marker if resources were populated
    # This is checked by the resources-check.sh condition
    if jq -e '.lifecycle.setup.steps[] | select(.name == "populate-resources" or .name == "add-data")' "${SERVICE_JSON:-${APP_ROOT}/.vrooli/service.json}" >/dev/null 2>&1; then
        touch "$data_dir/.resources-populated"
    fi
    
    log::debug "Setup marked as complete"
}