#!/usr/bin/env bash
set -euo pipefail

#######################################
# Universal Application Management Script
# 
# This is the single entry point for all lifecycle operations.
# Works with both Vrooli monorepo and standalone applications.
# 
# Usage:
#   ./scripts/manage.sh <phase> [options]
#   ./scripts/manage.sh setup --target native-linux
#   ./scripts/manage.sh develop --detached yes
#   ./scripts/manage.sh build --environment production
#   
# All behavior is controlled by .vrooli/service.json
#######################################

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PROJECT_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)

# Minimal bootstrap - only source what we absolutely need
if [[ -f "$SCRIPT_DIR/lib/utils/log.sh" ]]; then
    # shellcheck disable=SC1091
    source "$SCRIPT_DIR/lib/utils/log.sh"
else
    # Fallback if log.sh doesn't exist
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warning() { echo "[WARN] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
fi

#######################################
# Display help information
#######################################
manage::show_help() {
    cat << EOF
Universal Application Management Interface

USAGE:
    $0 <phase> [options]
    $0 --help | -h
    $0 --version | -v
    $0 --list-phases
    $0 --check-deps

PHASES:
    Phases are defined in .vrooli/service.json under the 'lifecycle' key.
    Common phases include:
    
    setup       Prepare the environment for development/production
    develop     Start the development environment
    build       Build the application artifacts
    deploy      Deploy the application
    test        Run tests
    
    Custom phases can be defined in service.json.

OPTIONS:
    Options vary by phase. Common options include:
    
    --target <target>        Deployment target (native-linux, docker, k8s, etc.)
    --environment <env>      Environment (development, production, staging)
    --dry-run                Preview what would be executed without making changes
    --yes                    Skip confirmation prompts
    --help                   Show phase-specific help
    
    Run '$0 <phase> --help' for phase-specific options.

EXAMPLES:
    $0 setup --target native-linux
    $0 develop --detached yes
    $0 build --environment production --dry-run
    $0 test --coverage yes
    $0 deploy --source k8s --version 2.0.0 --dry-run
    $0 my-custom-phase --my-option value

CONFIGURATION:
    All lifecycle phases and their steps are defined in:
    $PROJECT_ROOT/.vrooli/service.json
    
    The lifecycle engine at scripts/lib/lifecycle/engine.sh
    executes the configured steps for each phase.

EOF
}

#######################################
# Display version information
#######################################
manage::show_version() {
    local service_json="$PROJECT_ROOT/.vrooli/service.json"
    
    if [[ -f "$service_json" ]]; then
        local app_version
        app_version=$(jq -r '.version // "unknown"' "$service_json" 2>/dev/null || echo "unknown")
        echo "Application version: $app_version"
    else
        echo "Version: unknown (no service.json found)"
    fi
    
    echo "Manage script: 1.0.0"
    echo "Project root: $PROJECT_ROOT"
}

#######################################
# List available phases from service.json
#######################################
manage::list_phases() {
    local service_json="$PROJECT_ROOT/.vrooli/service.json"
    
    if [[ ! -f "$service_json" ]]; then
        log::error "No service.json found at $service_json"
        echo "Cannot list phases without service.json configuration"
        return 1
    fi
    
    echo "Available lifecycle phases:"
    echo
    
    # Check if jq is available
    if command -v jq &> /dev/null; then
        jq -r '.lifecycle | to_entries[] | "  \(.key)\t\(.value.description // "No description")"' "$service_json" 2>/dev/null || {
            log::warning "Failed to parse service.json"
            echo "  (Unable to list phases - invalid JSON or missing lifecycle key)"
        }
    else
        log::warning "jq is not installed - showing raw phase names only"
        grep -o '"[^"]*"[[:space:]]*:' "$service_json" | grep -v "lifecycle" | sed 's/"//g' | sed 's/://' | sed 's/^/  /'
    fi
    
    echo
    echo "Run '$0 <phase> --help' for phase-specific options"
}

#######################################
# Check system dependencies
#######################################
manage::check_dependencies() {
    echo "Checking system dependencies..."
    echo
    
    local missing_deps=()
    local optional_deps=()
    
    # Check required dependencies
    echo "Required dependencies:"
    for cmd in bash jq; do
        if command -v "$cmd" &> /dev/null; then
            echo "  ✓ $cmd - found at $(command -v "$cmd")"
        else
            echo "  ✗ $cmd - NOT FOUND"
            missing_deps+=("$cmd")
        fi
    done
    
    echo
    echo "Optional dependencies (enhance functionality):"
    
    # Check optional dependencies with descriptions
    local -A optional_cmds=(
        [git]="Version control and repository management"
        [docker]="Container runtime for Docker deployments"
        [kubectl]="Kubernetes cluster management"
        [node]="Node.js runtime for JavaScript execution"
        [pnpm]="Fast, disk space efficient package manager"
    )
    
    for cmd in "${!optional_cmds[@]}"; do
        if command -v "$cmd" &> /dev/null; then
            echo "  ✓ $cmd - found"
        else
            echo "  ○ $cmd - not found (${optional_cmds[$cmd]})"
            optional_deps+=("$cmd")
        fi
    done
    
    echo
    
    # Report results
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        echo "ERROR: Missing required dependencies: ${missing_deps[*]}"
        echo
        echo "Installation instructions:"
        
        # Provide platform-specific installation instructions
        if [[ -f /etc/os-release ]]; then
            # shellcheck disable=SC1091
            source /etc/os-release
            case "$ID" in
                ubuntu|debian)
                    echo "  sudo apt-get update && sudo apt-get install -y ${missing_deps[*]}"
                    ;;
                fedora|rhel|centos)
                    echo "  sudo yum install -y ${missing_deps[*]}"
                    ;;
                arch)
                    echo "  sudo pacman -S ${missing_deps[*]}"
                    ;;
                *)
                    echo "  Please install: ${missing_deps[*]}"
                    ;;
            esac
        else
            echo "  Please install: ${missing_deps[*]}"
        fi
        
        return 1
    else
        echo "✓ All required dependencies are installed"
        
        if [[ ${#optional_deps[@]} -gt 0 ]]; then
            echo
            echo "TIP: Install optional dependencies for full functionality:"
            echo "  Missing: ${optional_deps[*]}"
        fi
        
        return 0
    fi
}

#######################################
# Main execution
#######################################
manage::main() {
    local phase="${1:-}"
    
    # Check for --dry-run early to set it globally
    local dry_run_flag="false"
    for arg in "$@"; do
        if [[ "$arg" == "--dry-run" ]]; then
            dry_run_flag="true"
            export DRY_RUN="true"
            break
        fi
    done
    
    # Handle special flags
    case "$phase" in
        --help|-h|"")
            manage::show_help
            exit 0
            ;;
        --version|-v)
            manage::show_version
            exit 0
            ;;
        --list-phases|--list)
            manage::list_phases
            exit 0
            ;;
        --check-deps|--check-dependencies)
            manage::check_dependencies
            exit $?
            ;;
    esac
    
    # Check for service.json
    local service_json="$PROJECT_ROOT/.vrooli/service.json"
    if [[ ! -f "$service_json" ]]; then
        log::error "No service.json found at $service_json"
        log::error "This directory does not appear to be a properly configured application"
        echo
        echo "To initialize a new application, create .vrooli/service.json with lifecycle configuration"
        exit 1
    fi
    
    # Check for lifecycle engine
    local lifecycle_engine="$SCRIPT_DIR/lib/lifecycle/engine.sh"
    if [[ ! -f "$lifecycle_engine" ]]; then
        log::error "Lifecycle engine not found at $lifecycle_engine"
        log::error "The scripts/lib directory may be missing or incomplete"
        
        # Provide helpful message based on context
        if [[ -d "$PROJECT_ROOT/packages" ]]; then
            echo "This appears to be the Vrooli monorepo. Try running:"
            echo "  git restore scripts/lib"
        else
            echo "This appears to be a standalone app. Ensure scripts/lib was properly copied."
        fi
        exit 1
    fi
    
    # Check for required dependencies first
    if ! command -v jq &> /dev/null; then
        log::error "Required dependency 'jq' is not installed"
        echo "Run '$0 --check-deps' to see all dependency requirements"
        exit 1
    fi
    
    # Validate phase exists in service.json
    local phase_exists
    phase_exists=$(jq --arg phase "$phase" '.lifecycle | has($phase)' "$service_json" 2>/dev/null || echo "false")
    
    if [[ "$phase_exists" == "false" ]]; then
        log::error "Phase '$phase' not found in service.json"
        echo
        echo "Available phases:"
        manage::list_phases
        echo
        echo "To see all available options, run: $0 --help"
        exit 1
    fi
    
    # Show dry-run banner if enabled
    if [[ "$dry_run_flag" == "true" ]]; then
        echo "═══════════════════════════════════════════════════════════════" >&2
        echo "DRY RUN MODE: No actual changes will be made" >&2
        echo "═══════════════════════════════════════════════════════════════" >&2
        echo >&2
    fi
    
    # Set VROOLI_CONTEXT if not already set
    # Detect context based on presence of packages directory and other markers
    if [[ -z "${VROOLI_CONTEXT:-}" ]]; then
        if [[ -d "$PROJECT_ROOT/packages" ]] && [[ -f "$PROJECT_ROOT/packages/server/package.json" ]]; then
            export VROOLI_CONTEXT="monorepo"
            log::info "Detected Vrooli monorepo context"
        else
            export VROOLI_CONTEXT="standalone"
            log::info "Detected standalone application context"
        fi
    fi
    
    # Export PROJECT_ROOT for child scripts
    export PROJECT_ROOT
    
    # Set default ENVIRONMENT, LOCATION, and TARGET if not already set (commonly needed by many phases)
    export ENVIRONMENT="${ENVIRONMENT:-development}"
    export LOCATION="${LOCATION:-Local}"
    export TARGET="${TARGET:-docker}"
    
    # Shift phase from arguments and pass remaining to lifecycle engine
    shift
    
    # Delegate to lifecycle engine
    if [[ "$dry_run_flag" == "true" ]]; then
        log::info "[DRY RUN] Would execute phase '$phase' via lifecycle engine..."
    else
        log::info "Executing phase '$phase' via lifecycle engine..."
    fi
    
    exec "$lifecycle_engine" "$phase" "$@"
}

# Execute main with all arguments
manage::main "$@"