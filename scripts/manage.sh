#!/usr/bin/env bash
set -euo pipefail

#######################################
# Application Management Script
# 
# Single entry point for all lifecycle operations.
# All behavior is controlled by .vrooli/service.json
#######################################

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source utilities
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/utils/json.sh"

#######################################
# Execute lifecycle phase
# Simple function that runs steps from service.json
#######################################
manage::execute_phase() {
    local phase="$1"
    shift  # Remove phase from arguments
    
    # Get phase description
    local description
    description=$(json::get_value ".lifecycle.${phase}.description" "")
    
    log::header "Starting phase: $phase"
    [[ -n "$description" ]] && log::info "$description"
    
    # Parse common arguments into environment variables
    for arg in "$@"; do
        case "$arg" in
            --target=*) export TARGET="${arg#*=}" ;;
            --environment=*) export ENVIRONMENT="${arg#*=}" ;;
            --dry-run) export DRY_RUN="true" ;;
            *) ;; # Other args available via $@
        esac
    done
    
    # Export phase info for scripts
    export LIFECYCLE_PHASE="$phase"
    export LIFECYCLE_ARGS="$*"
    
    # Get and execute steps
    local steps
    steps=$(json::get_value ".lifecycle.${phase}.steps" "[]")
    
    if [[ "$steps" == "[]" ]]; then
        log::warning "No steps defined for phase: $phase"
        return 0
    fi
    
    # Execute each step
    local step_count=0
    local total_steps
    total_steps=$(echo "$steps" | jq 'length')
    
    echo "$steps" | jq -c '.[]' | while IFS= read -r step; do
        step_count=$((step_count + 1))
        
        local name=$(echo "$step" | jq -r '.name // "unnamed"')
        local run=$(echo "$step" | jq -r '.run // ""')
        local desc=$(echo "$step" | jq -r '.description // ""')
        
        [[ -z "$run" ]] && continue
        
        log::info "[$step_count/$total_steps] Executing: $name"
        [[ -n "$desc" ]] && echo "  → $desc" >&2
        
        if [[ "${DRY_RUN:-}" == "true" ]]; then
            echo "[DRY RUN] Would execute: $run" >&2
        else
            # Execute in project root for consistency
            (cd "$var_ROOT_DIR" && eval "$run") || {
                local exit_code=$?
                log::error "Step '$name' failed with exit code $exit_code"
                return $exit_code
            }
        fi
    done
    
    log::success "Phase '$phase' completed successfully"
    return 0
}

#######################################
# Display help with dynamic app info
#######################################
manage::show_help() {
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
    $0 <phase> [options]
    $0 --help | -h
    $0 --list-phases

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
    
    Run '$0 <phase> --help' for phase-specific options.

EXAMPLES:
    $0 setup --target native-linux
    $0 develop --detached yes
    $0 build --environment production --dry-run
    $0 test --coverage yes
    $0 deploy --source k8s --version 2.0.0

CONFIGURATION:
    Lifecycle phases and steps: $var_ROOT_DIR/.vrooli/service.json
    Project root: $var_ROOT_DIR

EOF
}

#######################################
# List available phases
#######################################
manage::list_phases() {
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

#######################################
# Main execution
#######################################
manage::main() {
    local phase="${1:-}"
    
    # Check for --dry-run early
    local dry_run_flag="false"
    for arg in "$@"; do
        [[ "$arg" == "--dry-run" ]] && {
            dry_run_flag="true"
            export DRY_RUN="true"
            break
        }
    done
    
    # Handle special flags
    case "$phase" in
        --help|-h|"")
            manage::show_help
            exit 0
            ;;
        --list-phases|--list)
            manage::list_phases
            exit 0
            ;;
    esac
    
    # Validate service.json exists
    if ! json::validate_config; then
        log::error "No valid service.json found in this directory"
        echo "Create .vrooli/service.json with lifecycle configuration"
        exit 1
    fi
    
    # Validate phase exists
    if ! json::path_exists ".lifecycle.${phase}"; then
        log::error "Phase '$phase' not found in service.json"
        echo
        manage::list_phases
        echo
        echo "Run '$0 --help' for usage information"
        exit 1
    fi
    
    # Show dry-run banner
    if [[ "$dry_run_flag" == "true" ]]; then
        echo "═══════════════════════════════════════════════════════" >&2
        echo "DRY RUN MODE: No actual changes will be made" >&2
        echo "═══════════════════════════════════════════════════════" >&2
        echo >&2
    fi
    
    # Export common environment variables
    export var_ROOT_DIR
    export ENVIRONMENT="${ENVIRONMENT:-development}"
    export LOCATION="${LOCATION:-Local}"
    export TARGET="${TARGET:-docker}"
    
    # Remove phase from arguments
    shift
    
    # Execute phase directly (no more external executor!)
    [[ "$dry_run_flag" == "true" ]] && \
        log::info "[DRY RUN] Executing phase '$phase'..." || \
        log::info "Executing phase '$phase'..."
    
    manage::execute_phase "$phase" "$@"
}

# Execute
manage::main "$@"