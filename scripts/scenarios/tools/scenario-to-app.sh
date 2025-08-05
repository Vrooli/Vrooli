#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Scenario-to-App Converter (Resource-Based Architecture)
# 
# Converts Vrooli scenarios into running applications using existing resource
# infrastructure. This approach leverages proven manage.sh and inject.sh scripts
# instead of generating Docker configurations.
#
# Architecture Philosophy:
# - Use existing resource management (manage.sh for startup/teardown)
# - Use existing data injection (inject.sh for initialization)
# - Scenarios orchestrate proven local resources, don't replace them
# - Custom Docker only for edge cases (scenario-provided)
#
# Usage:
#   ./scenario-to-app.sh <scenario-name>
#   ./scenario-to-app.sh ai-content-assistant-example
#   ./scenario-to-app.sh multi-modal-ai-assistant --verbose
#
################################################################################

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SCENARIOS_DIR="${PROJECT_ROOT}/scripts/scenarios/core"
RESOURCES_DIR="${PROJECT_ROOT}/scripts/resources"

# Source utilities or define fallback logging
if [[ -f "${PROJECT_ROOT}/scripts/helpers/utils/log.sh" ]]; then
    source "${PROJECT_ROOT}/scripts/helpers/utils/log.sh"
fi

# Ensure logging functions are available (define if not already defined)
if ! type log_info >/dev/null 2>&1; then
    log_info() { echo "[$(date +'%H:%M:%S')] INFO: $*"; }
fi
if ! type log_error >/dev/null 2>&1; then
    log_error() { echo "[$(date +'%H:%M:%S')] ERROR: $*" >&2; }
fi
if ! type log_success >/dev/null 2>&1; then
    log_success() { echo "[$(date +'%H:%M:%S')] SUCCESS: $*"; }
fi
if ! type log_warning >/dev/null 2>&1; then
    log_warning() { echo "[$(date +'%H:%M:%S')] WARNING: $*"; }
fi

# Global variables
SCENARIO_NAME=""
SCENARIO_PATH=""
SERVICE_JSON=""
VERBOSE=false
DRY_RUN=false
CLEANUP_ON_ERROR=true

# Track started resources for cleanup
declare -a STARTED_RESOURCES=()

################################################################################
# Helper Functions
################################################################################

show_usage() {
    cat << EOF
Usage: $0 <scenario-name> [options]

Convert a validated scenario into a running application using existing resource infrastructure.

Arguments:
  scenario-name     Name of the scenario (e.g., ai-content-assistant-example)

Options:
  --verbose         Enable verbose output
  --dry-run         Show what would be done without executing
  --no-cleanup      Don't cleanup on error (for debugging)
  --help           Show this help message

Examples:
  $0 ai-content-assistant-example
  $0 multi-modal-ai-assistant --verbose
  $0 research-assistant --dry-run

EOF
}

parse_args() {
    if [[ $# -eq 0 ]]; then
        show_usage
        exit 1
    fi

    # Check for help first
    if [[ "$1" == "--help" ]]; then
        show_usage
        exit 0
    fi

    SCENARIO_NAME="$1"
    shift

    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose)
                VERBOSE=true
                ;;
            --dry-run)
                DRY_RUN=true
                ;;
            --no-cleanup)
                CLEANUP_ON_ERROR=false
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
        shift
    done
}

# Cleanup function for error handling
cleanup_on_error() {
    if [[ "$CLEANUP_ON_ERROR" != "true" ]]; then
        log_warning "Cleanup disabled - resources may still be running"
        return 0
    fi

    if [[ ${#STARTED_RESOURCES[@]} -gt 0 ]]; then
        log_warning "Cleaning up started resources..."
        for resource in "${STARTED_RESOURCES[@]}"; do
            stop_resource "$resource" || true
        done
    fi
}

# Set up error handling
trap cleanup_on_error EXIT

################################################################################
# Validation Functions
################################################################################

validate_scenario() {
    log_info "Validating scenario: $SCENARIO_NAME"
    
    # Find scenario directory
    SCENARIO_PATH="${SCENARIOS_DIR}/${SCENARIO_NAME}"
    if [[ ! -d "$SCENARIO_PATH" ]]; then
        log_error "Scenario directory not found: $SCENARIO_PATH"
        exit 1
    fi

    # Check for service.json
    local service_json_path="${SCENARIO_PATH}/.vrooli/service.json"
    if [[ ! -f "$service_json_path" ]]; then
        log_error "service.json not found: $service_json_path"
        exit 1
    fi

    # Validate JSON syntax
    if ! SERVICE_JSON=$(cat "$service_json_path") || ! echo "$SERVICE_JSON" | jq empty 2>/dev/null; then
        log_error "Invalid JSON in service.json"
        exit 1
    fi

    # Check for optional scenario files
    local optional_files=(
        "README.md"
        "test.sh"
    )
    
    for file in "${optional_files[@]}"; do
        if [[ ! -f "${SCENARIO_PATH}/${file}" ]]; then
            [[ "$VERBOSE" == "true" ]] && log_info "Optional file not present: $file"
        fi
    done

    log_success "Scenario validation passed"
}

################################################################################
# Resource Management Functions
################################################################################

# Extract required resources from service.json
get_required_resources() {
    echo "$SERVICE_JSON" | jq -r '
        .resources | 
        to_entries[] | 
        .value | 
        to_entries[] | 
        select(.value.enabled == true and (.value.required // false) == true) | 
        .key
    ' 2>/dev/null | sort -u
}

# Find resource management script
find_resource_script() {
    local resource="$1"
    local script_name="$2"  # manage.sh or inject.sh
    
    # Search for resource directory
    local resource_dir
    resource_dir=$(find "$RESOURCES_DIR" -type d -name "$resource" 2>/dev/null | head -1)
    
    if [[ -z "$resource_dir" ]]; then
        return 1
    fi
    
    local script_path="${resource_dir}/${script_name}"
    if [[ -x "$script_path" ]]; then
        echo "$script_path"
        return 0
    fi
    
    return 1
}

# Start a resource using its manage.sh script
start_resource() {
    local resource="$1"
    
    log_info "Starting resource: $resource"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would start resource: $resource"
        return 0
    fi
    
    local manage_script
    if ! manage_script=$(find_resource_script "$resource" "manage.sh"); then
        log_error "No manage.sh script found for resource: $resource"
        return 1
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Using script: $manage_script"
    fi
    
    # Start the resource
    if "$manage_script" --action start; then
        STARTED_RESOURCES+=("$resource")
        log_success "Started resource: $resource"
        return 0
    else
        log_error "Failed to start resource: $resource"
        return 1
    fi
}

# Stop a resource using its manage.sh script
stop_resource() {
    local resource="$1"
    
    local manage_script
    if ! manage_script=$(find_resource_script "$resource" "manage.sh"); then
        log_warning "No manage.sh script found for resource: $resource"
        return 1
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Stopping resource: $resource using $manage_script"
    fi
    
    "$manage_script" --action stop || true
}

# Wait for resource to be healthy
wait_for_resource() {
    local resource="$1"
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for $resource to be ready..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would wait for resource: $resource"
        return 0
    fi
    
    local manage_script
    if ! manage_script=$(find_resource_script "$resource" "manage.sh"); then
        log_warning "No health check available for resource: $resource"
        sleep 5  # Basic fallback
        return 0
    fi
    
    while (( attempt < max_attempts )); do
        if "$manage_script" --action status >/dev/null 2>&1; then
            log_success "$resource is ready"
            return 0
        fi
        
        sleep 2
        ((attempt++))
        
        if [[ "$VERBOSE" == "true" ]] && (( attempt % 10 == 0 )); then
            log_info "Still waiting for $resource... (${attempt}/${max_attempts})"
        fi
    done
    
    log_error "$resource failed to become ready after $((max_attempts * 2)) seconds"
    return 1
}

# Inject data into a resource using its inject.sh script
inject_resource_data() {
    local resource="$1"
    
    log_info "Injecting data for resource: $resource"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would inject data for resource: $resource"
        return 0
    fi
    
    local inject_script
    if ! inject_script=$(find_resource_script "$resource" "inject.sh"); then
        log_info "No inject.sh script found for resource: $resource (this is OK)"
        return 0
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Using injection script: $inject_script"
    fi
    
    # Change to scenario directory so injection script can find files
    local old_pwd="$PWD"
    cd "$SCENARIO_PATH"
    
    if "$inject_script" --scenario-dir "$SCENARIO_PATH"; then
        log_success "Data injection completed for: $resource"
    else
        log_error "Data injection failed for: $resource"
        cd "$old_pwd"
        return 1
    fi
    
    cd "$old_pwd"
    return 0
}

################################################################################
# Application Orchestration
################################################################################

run_scenario_startup() {
    log_info "Running scenario startup scripts..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would run scenario startup scripts"
        return 0
    fi
    
    # Check for deployment startup script
    local startup_script="${SCENARIO_PATH}/deployment/startup.sh"
    if [[ -x "$startup_script" ]]; then
        log_info "Running deployment startup script..."
        if "$startup_script"; then
            log_success "Deployment startup completed"
        else
            log_error "Deployment startup failed"
            return 1
        fi
    else
        log_info "No deployment startup script found (this is OK)"
    fi
    
    return 0
}

get_access_urls() {
    log_info "Application access points:"
    
    # Extract URLs from service.json if available
    local app_url
    app_url=$(echo "$SERVICE_JSON" | jq -r '.deployment.urls.application // empty' 2>/dev/null)
    if [[ -n "$app_url" ]]; then
        log_info "  Application: $app_url"
    fi
    
    # Common resource URLs
    local required_resources
    mapfile -t required_resources < <(get_required_resources)
    
    for resource in "${required_resources[@]}"; do
        case "$resource" in
            postgres|postgresql)
                log_info "  PostgreSQL: localhost:5432"
                ;;
            n8n)
                log_info "  n8n Workflows: http://localhost:5678"
                ;;
            windmill)
                log_info "  Windmill UI: http://localhost:8000"
                ;;
            ollama)
                log_info "  Ollama API: http://localhost:11434"
                ;;
            whisper)
                log_info "  Whisper API: http://localhost:9000"
                ;;
            minio)
                log_info "  MinIO Console: http://localhost:9001"
                ;;
            qdrant)
                log_info "  Qdrant API: http://localhost:6333"
                ;;
            redis)
                log_info "  Redis: localhost:6379"
                ;;
        esac
    done
}

################################################################################
# Main Execution
################################################################################

main() {
    # Parse command line arguments
    parse_args "$@"
    
    # Show header
    log_info "🚀 Vrooli Scenario-to-App Converter (Resource-Based)"
    log_info "Scenario: $SCENARIO_NAME"
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN MODE - No actual changes will be made"
    fi
    echo ""
    
    # Phase 1: Validation
    log_info "Phase 1: Scenario Validation"
    validate_scenario
    echo ""
    
    # Phase 2: Resource Analysis
    log_info "Phase 2: Resource Analysis"
    local required_resources
    mapfile -t required_resources < <(get_required_resources)
    
    if [[ ${#required_resources[@]} -eq 0 ]]; then
        log_warning "No required resources found in service.json"
        exit 0
    fi
    
    log_info "Required resources: ${required_resources[*]}"
    echo ""
    
    # Phase 3: Resource Startup
    log_info "Phase 3: Resource Startup"
    for resource in "${required_resources[@]}"; do
        start_resource "$resource" || exit 1
        wait_for_resource "$resource" || exit 1
    done
    echo ""
    
    # Phase 4: Data Injection
    log_info "Phase 4: Data Injection"
    for resource in "${required_resources[@]}"; do
        inject_resource_data "$resource" || exit 1
    done
    echo ""
    
    # Phase 5: Application Startup
    log_info "Phase 5: Application Startup"
    run_scenario_startup || exit 1
    echo ""
    
    # Phase 6: Success Summary
    log_info "Phase 6: Application Ready"
    local scenario_display_name
    scenario_display_name=$(echo "$SERVICE_JSON" | jq -r '.service.displayName // .service.name' 2>/dev/null)
    
    log_success "✅ $scenario_display_name is now running!"
    echo ""
    get_access_urls
    echo ""
    log_info "To stop the application:"
    log_info "  Use Ctrl+C or run the individual resource stop commands"
    echo ""
    
    # Disable cleanup on successful completion
    trap - EXIT
    
    # Keep running (similar to develop.sh)
    log_info "Application is running. Press Ctrl+C to stop all services."
    
    # Handle shutdown gracefully
    shutdown() {
        log_info "Shutting down services..."
        cleanup_on_error
        exit 0
    }
    
    trap shutdown SIGINT SIGTERM
    
    # Keep running
    while true; do
        sleep 1
    done
}

# Execute main function
main "$@"