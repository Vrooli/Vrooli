#!/usr/bin/env bash
#######################################
# Setup State Management Library
# Provides setup state tracking and validation for the Vrooli platform
#######################################

set -euo pipefail

# Global variables for setup artifacts and reasons
SETUP_MISSING_ARTIFACTS=()
SETUP_REASONS=()

#######################################
# Check if setup artifacts exist and are valid
# Sets SETUP_MISSING_ARTIFACTS global variable with list of missing items
# Returns:
#   0 if all artifacts exist
#   1 if any artifacts are missing
#######################################
setup::verify_artifacts() {
    # Reset global variable
    SETUP_MISSING_ARTIFACTS=()
    
    # Check for common build artifacts based on project structure
    local artifacts_missing=false
    
    # Check Go binary if Go project exists
    if [[ -f "api/go.mod" ]]; then
        local binary_name
        binary_name=$(basename "$PWD")
        # Check for common binary naming patterns
        if [[ ! -f "api/${binary_name}-api" ]] && [[ ! -f "api/${binary_name}" ]] && [[ ! -f "api/api-server" ]]; then
            log::debug "Go binary missing for ${binary_name} (checked: ${binary_name}-api, ${binary_name}, api-server)"
            SETUP_MISSING_ARTIFACTS+=("Go binary (api/${binary_name}-api, api/${binary_name}, or api/api-server)")
            artifacts_missing=true
        fi
    fi
    
    # Check Node.js dependencies if package.json exists
    if [[ -f "ui/package.json" ]] && [[ ! -d "ui/node_modules" ]]; then
        log::debug "Node.js dependencies missing in ui/"
        SETUP_MISSING_ARTIFACTS+=("Node.js dependencies (ui/node_modules)")
        artifacts_missing=true
    fi
    
    # Return false if any artifacts missing
    if [[ "$artifacts_missing" == "true" ]]; then
        return 1
    fi
    
    return 0
}

#######################################
# Check if app needs setup based on git commit only
# Sets SETUP_REASONS global array with specific reasons
#
# Simplified logic: Only runs setup when:
# - First run (no state file)
# - Git commit changed
# - Critical artifacts are missing
#
# Local changes don't trigger setup since developers
# know when they need to re-run setup
# Returns:
#   0 if setup is needed
#   1 if setup is not needed
#######################################
setup::is_needed() {
    local state_file="data/.setup-state"
    
    # Reset global array for setup reasons
    SETUP_REASONS=()
    
    # Ensure data directory exists
    mkdir -p "$(dirname "$state_file")"
    
    # No state file = needs setup
    if [[ ! -f "$state_file" ]]; then
        log::debug "No setup state file found, setup needed"
        SETUP_REASONS+=("First run - no previous setup state found")
        return 0
    fi
    
    # Validate state file is proper JSON
    if ! jq empty "$state_file" 2>/dev/null; then
        log::debug "Corrupted setup state file, setup needed"
        SETUP_REASONS+=("Corrupted setup state file - needs regeneration")
        rm -f "$state_file"
        return 0
    fi
    
    # Get current git commit
    local current_commit
    current_commit=$(git::get_commit)
    
    # Get recorded commit
    local recorded_commit
    recorded_commit=$(jq -r '.git_commit // empty' "$state_file" 2>/dev/null || echo "")
    
    # If git commit changed, needs setup
    if [[ "$current_commit" != "$recorded_commit" ]]; then
        log::debug "Git commit changed (${recorded_commit} -> ${current_commit}), setup needed"
        local short_old="${recorded_commit:0:8}"
        local short_new="${current_commit:0:8}"
        SETUP_REASONS+=("Git commit changed: ${short_old} → ${short_new}")
        return 0
    fi
    
    # Verify critical artifacts still exist
    if ! setup::verify_artifacts; then
        log::debug "Setup artifacts missing or invalid, setup needed"
        # Add specific missing artifacts to reasons
        if [[ ${#SETUP_MISSING_ARTIFACTS[@]} -gt 0 ]]; then
            for artifact in "${SETUP_MISSING_ARTIFACTS[@]}"; do
                SETUP_REASONS+=("Missing: $artifact")
            done
        else
            SETUP_REASONS+=("Build artifacts missing or invalid")
        fi
        return 0
    fi
    
    log::debug "Setup state is current"
    return 1  # No setup needed
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
# Mark setup as complete with current git state
#######################################
setup::mark_complete() {
    local state_file="data/.setup-state"
    mkdir -p "$(dirname "$state_file")"
    
    local current_commit setup_steps
    current_commit=$(git::get_commit)
    setup_steps=$(setup::get_steps_list)
    
    cat > "$state_file" << EOF
{
  "setup_version": "1.0.0",
  "git_commit": "$current_commit",
  "setup_completed_at": "$(date -Iseconds)",
  "setup_steps_completed": $setup_steps
}
EOF
    
    log::debug "Setup state marked as complete"
}