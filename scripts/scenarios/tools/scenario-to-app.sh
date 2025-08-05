#!/usr/bin/env bash

################################################################################
# Scenario-to-App Conversion Script
# 
# Converts Vrooli scenarios into deployable applications using the unified
# service.json format and resource injection engine.
#
# SCHEMA REFERENCE:
#   All service.json files follow the official schema:
#   .vrooli/schemas/service.schema.json
#   
#   Key paths used in this script:
#   - .service.name           (scenario identifier)
#   - .service.displayName    (human-readable name)
#   - .resources.{category}.{resource}  (resource definitions)
#   - .resources.{category}.{resource}.required  (not .optional!)
#   - .resources.{category}.{resource}.initialization  (setup data)
#
# Usage:
#   ./scenario-to-app.sh <scenario-name> [options]
#
# Options:
#   --mode        Deployment mode (local, docker, k8s) [default: local]
#   --validate    Validation mode (none, basic, full) [default: full]
#   --dry-run     Show what would be done without executing
#   --verbose     Enable verbose logging
#   --help        Show this help message
#
################################################################################

set -euo pipefail

# Define log functions first (before any usage)
log_info() { echo "[INFO] $1"; }
log_success() { echo "[SUCCESS] $1"; }
log_warning() { echo "[WARNING] $1"; }
log_error() { echo "[ERROR] $1" >&2; }
log_step() { echo "[STEP] $1"; }
log_phase() { echo "=== $1 ==="; }
log_banner() { echo ""; echo "=== $1 ==="; echo ""; }

# Script location and imports
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SCENARIOS_DIR="${PROJECT_ROOT}/scripts/scenarios/core"
# INJECTION_DIR removed in Phase 3 - direct integration without engine.sh

# Import helper functions (override defaults if available)
if [[ -f "${PROJECT_ROOT}/scripts/helpers/utils/log.sh" ]]; then
    source "${PROJECT_ROOT}/scripts/helpers/utils/log.sh"
fi

if [[ -f "${PROJECT_ROOT}/scripts/helpers/utils/system.sh" ]]; then
    source "${PROJECT_ROOT}/scripts/helpers/utils/system.sh"
fi

# Global variables
DEPLOYMENT_MODE="local"
VALIDATION_MODE="full"
DRY_RUN=false
VERBOSE=false
SCENARIO_NAME=""
SCENARIO_PATH=""
SERVICE_JSON=""
SERVICE_JSON_PATH=""

# Rollback tracking
declare -a ROLLBACK_ACTIONS=()

################################################################################
# Helper Functions
################################################################################

# Show usage information
show_usage() {
    head -n 20 "${BASH_SOURCE[0]}" | grep "^#" | grep -v "^#!/" | sed 's/^# //'
}

# Parse command line arguments
parse_args() {
    local scenario_provided=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --mode)
                DEPLOYMENT_MODE="$2"
                shift 2
                ;;
            --validate)
                VALIDATION_MODE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            --*)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
            *)
                if [[ "$scenario_provided" == true ]]; then
                    log_error "Multiple scenario names provided: '$SCENARIO_NAME' and '$1'"
                    show_usage
                    exit 1
                fi
                SCENARIO_NAME="$1"
                scenario_provided=true
                shift
                ;;
        esac
    done
    
    if [[ "$scenario_provided" == false ]]; then
        log_error "No scenario specified"
        show_usage
        exit 1
    fi
}

# Validate scenario exists and has required files
validate_scenario() {
    SCENARIO_PATH="${SCENARIOS_DIR}/${SCENARIO_NAME}"
    
    if [[ ! -d "$SCENARIO_PATH" ]]; then
        log_error "Scenario not found: $SCENARIO_NAME"
        log_info "Available scenarios:"
        ls -1 "$SCENARIOS_DIR" | sed 's/^/  - /'
        exit 1
    fi
    
    # Check for service.json
    SERVICE_JSON_PATH="${SCENARIO_PATH}/service.json"
    
    if [[ ! -f "$SERVICE_JSON_PATH" ]]; then
        log_error "service.json not found in scenario: $SCENARIO_NAME"
        log_info "All scenarios must have a service.json file"
        log_info "Expected location: $SERVICE_JSON_PATH"
        exit 1
    fi
    
    # Load service.json (already verified to exist)
    log_info "Loading service.json from $SERVICE_JSON_PATH"
    SERVICE_JSON=$(cat "$SERVICE_JSON_PATH")
    log_info "Loaded service.json successfully"
}

# Validate scenario structure based on service.json
validate_structure() {
    if [[ "$VALIDATION_MODE" == "none" ]]; then
        log_info "Skipping structure validation"
        return 0
    fi
    
    log_step "Validating scenario structure..."
    
    # Extract scenario name and version from service.json
    # NOTE: Using official service.schema.json paths (.service.* not .metadata.*)
    local scenario_name
    local scenario_version
    scenario_name=$(echo "$SERVICE_JSON" | jq -r '.service.name // ""')
    scenario_version=$(echo "$SERVICE_JSON" | jq -r '.service.version // ""')
    
    if [[ -z "$scenario_name" ]]; then
        log_error "Invalid service.json: missing service.name (see .vrooli/schemas/service.schema.json)"
        return 1
    fi
    
    [[ "$VERBOSE" == true ]] && log_info "Scenario: $scenario_name v$scenario_version"
    
    # Check for required initialization files
    # NOTE: Initialization data is embedded within each resource definition
    local init_resources
    if ! init_resources=$(echo "$SERVICE_JSON" | jq -r '
      .resources | 
      to_entries[] | 
      .value | 
      to_entries[] | 
      select(.value.initialization != null) | 
      .key
    '); then
        log_error "Failed to extract initialization resources from service.json"
        return 1
    fi
    
    for resource in $init_resources; do
        [[ "$VERBOSE" == true ]] && log_info "Checking initialization data for: $resource"
        
        # Check for database files (find resource across all categories)
        local db_files
        if ! db_files=$(echo "$SERVICE_JSON" | jq -r "
          .resources | 
          to_entries[] | 
          .value | 
          to_entries[] | 
          select(.key == \"$resource\") | 
          .value.initialization.data[]?.file // empty
        "); then
            log_warning "Failed to check database files for resource: $resource"
            continue
        fi
        for file in $db_files; do
            local full_path="${SCENARIO_PATH}/$file"
            if [[ -f "$full_path" ]]; then
                [[ "$VERBOSE" == true ]] && log_success "  ✓ Found: $file"
            else
                log_error "  ✗ Missing: $file"
                return 1
            fi
        done
        
        # Check for workflow files (find resource across all categories)
        local workflow_files
        if ! workflow_files=$(echo "$SERVICE_JSON" | jq -r "
          .resources | 
          to_entries[] | 
          .value | 
          to_entries[] | 
          select(.key == \"$resource\") | 
          .value.initialization.workflows[]?.file // empty
        "); then
            log_warning "Failed to check workflow files for resource: $resource"
            continue
        fi
        for file in $workflow_files; do
            local full_path="${SCENARIO_PATH}/$file"
            if [[ -f "$full_path" ]]; then
                [[ "$VERBOSE" == true ]] && log_success "  ✓ Found: $file"
            else
                log_error "  ✗ Missing: $file"
                return 1
            fi
        done
    done
    
    log_success "Structure validation passed"
}

# Validate required resources are available
validate_resources() {
    if [[ "$VALIDATION_MODE" != "full" ]]; then
        log_info "Skipping resource validation"
        return 0
    fi
    
    log_step "Validating required resources..."
    
    # Extract required resources from service.json
    # NOTE: Using standard service.schema.json structure (.resources.{category}.{resource})
    local required_resources
    if ! required_resources=$(echo "$SERVICE_JSON" | jq -r '
      .resources | 
      to_entries[] | 
      .value | 
      to_entries[] | 
      select(.value.required == true) | 
      .key
    '); then
        log_error "Failed to extract required resources from service.json"
        return 1
    fi
    
    if [[ -z "$required_resources" ]]; then
        log_warning "No required resources defined"
        return 0
    fi
    
    [[ "$VERBOSE" == true ]] && log_info "Required resources: $(echo "$required_resources" | tr '\n' ' ')"
    
    # Check each resource
    for resource in $required_resources; do
        # Check if resource has an inject.sh script
        local inject_script
        inject_script=$(find "${PROJECT_ROOT}/scripts/resources" -name "inject.sh" -path "*/${resource}/*" 2>/dev/null | head -1)
        
        if [[ -n "$inject_script" ]]; then
            [[ "$VERBOSE" == true ]] && log_success "  ✓ Injection handler found for: $resource"
        else
            log_warning "  ⚠ No injection handler for: $resource"
        fi
        
        # Check resource health if injection script supports status checks
        if [[ -x "$inject_script" ]]; then
            # First verify the script supports --status flag
            if "$inject_script" --help 2>/dev/null | grep -q "\-\-status" 2>/dev/null; then
                if "$inject_script" --status >/dev/null 2>&1; then
                    [[ "$VERBOSE" == true ]] && log_success "  ✓ Resource '$resource' is healthy"
                else
                    log_warning "  ⚠ Resource '$resource' health check failed"
                fi
            else
                [[ "$VERBOSE" == true ]] && log_info "  ℹ Resource '$resource' injection script doesn't support health checks"
            fi
        fi
    done
    
    log_success "Resource validation completed"
}

################################################################################
# Safety and Validation Functions
################################################################################

# Validate service.json structure and content
validate_service_config() {
    local config_file="$1"
    local validation_type="${2:-full}"  # basic, full
    
    if [[ ! -f "$config_file" ]]; then
        log_error "Service config file not found: $config_file"
        return 1
    fi
    
    # Basic JSON validation
    if ! jq empty "$config_file" 2>/dev/null; then
        log_error "Service config is not valid JSON: $config_file"
        return 1
    fi
    
    if [[ "$validation_type" == "basic" ]]; then
        return 0
    fi
    
    # Full structure validation
    local config_content
    if ! config_content=$(cat "$config_file"); then
        log_error "Failed to read service config: $config_file"
        return 1
    fi
    
    # Check required top-level structure
    local required_sections=("service" "resources")
    for section in "${required_sections[@]}"; do
        if [[ $(echo "$config_content" | jq -r ".$section // false") == "false" ]]; then
            log_error "Missing required section: $section"
            return 1
        fi
    done
    
    # Check resources structure - dynamically extract categories from config
    # Phase 2 improvement: Support any resource category from service.json
    local resource_categories=()
    if [[ $(echo "$config_content" | jq -r ".resources // false") != "false" ]]; then
        # Extract all category names from the resources section
        if ! mapfile -t resource_categories < <(echo "$config_content" | jq -r '.resources | keys[]'); then
            log_error "Failed to extract resource categories from config"
            return 1
        fi
    fi
    
    # If no categories found, create basic structure  
    if [[ ${#resource_categories[@]} -eq 0 ]]; then
        log_info "No resource categories found - service config may need initialization"
    else
        [[ "$VERBOSE" == true ]] && log_info "Found resource categories: ${resource_categories[*]}"
    fi
    
    [[ "$VERBOSE" == true ]] && log_success "Service config validation passed: $config_file"
    return 0
}

# Create safe backup with metadata
create_safe_backup() {
    local source_file="$1"
    local backup_reason="${2:-manual}"
    
    if [[ ! -f "$source_file" ]]; then
        log_warning "Source file not found for backup: $source_file"
        return 1
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${source_file}.backup.${timestamp}.${backup_reason}"
    
    if cp "$source_file" "$backup_file"; then
        # Create metadata file
        cat > "${backup_file}.meta" << EOF
{
  "source_file": "$source_file",
  "backup_time": "$(date -Iseconds)",
  "backup_reason": "$backup_reason",
  "script_version": "scenario-to-app.sh",
  "user": "$(whoami)",
  "pwd": "$(pwd)"
}
EOF
        [[ "$VERBOSE" == true ]] && log_info "Created backup: $backup_file"
        echo "$backup_file"
        return 0
    else
        log_error "Failed to create backup of: $source_file"
        return 1
    fi
}

# Pre-flight safety checks before any modifications
preflight_safety_check() {
    local service_config="$1"
    
    [[ "$VERBOSE" == true ]] && log_step "Running pre-flight safety checks..."
    
    # Check if we have write permissions
    local config_dir=$(dirname "$service_config")
    if [[ ! -w "$config_dir" ]]; then
        log_error "No write permission to config directory: $config_dir"
        return 1
    fi
    
    # Check for required tools
    local required_tools=("jq" "cp" "mv" "date")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            log_error "Required tool not found: $tool"
            return 1
        fi
    done
    
    # Validate existing config if it exists
    if [[ -f "$service_config" ]]; then
        if ! validate_service_config "$service_config" "full"; then
            log_error "Existing service config is invalid"
            return 1
        fi
    fi
    
    [[ "$VERBOSE" == true ]] && log_success "Pre-flight safety checks passed"
    return 0
}

# REMOVED: generate_resources_config function
# Scenarios are self-contained and use their own service.json as single source of truth
# This aligns with the service.schema.json specification where scenarios
# define their own resource requirements without modifying global configuration

# Rollback system functions
add_rollback_action() {
    local description="$1"
    local command="$2"
    
    ROLLBACK_ACTIONS+=("$description|$command")
    [[ "$VERBOSE" == true ]] && log_info "Added rollback action: $description"
}

execute_rollback() {
    if [[ ${#ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        [[ "$VERBOSE" == true ]] && log_info "No rollback actions to execute"
        return 0
    fi
    
    log_warning "Executing rollback actions..."
    
    local success_count=0
    local total_count=${#ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log_info "Rollback: $description"
        
        if eval "$command" 2>/dev/null; then
            success_count=$((success_count + 1))
            [[ "$VERBOSE" == true ]] && log_success "Rollback completed: $description"
        else
            log_warning "Rollback failed: $description"
        fi
    done
    
    log_info "Rollback completed: $success_count/$total_count actions successful"
    ROLLBACK_ACTIONS=()
}

# Phase 2: Dynamic resource discovery function
find_resource_inject_script() {
    local resource="$1"
    local category="$2"
    
    # Try category-specific location first
    local category_script="${PROJECT_ROOT}/scripts/resources/${category}/${resource}/inject.sh"
    if [[ -x "$category_script" ]]; then
        echo "$category_script"
        return 0
    fi
    
    # Fallback to searching across all resources
    local fallback_script=$(find "${PROJECT_ROOT}/scripts/resources" -name "inject.sh" -path "*/${resource}/*" -executable 2>/dev/null | head -1)
    if [[ -n "$fallback_script" ]]; then
        echo "$fallback_script"
        return 0
    fi
    
    # No injection script found
    return 1
}

# Phase 3: Direct resource injection without engine.sh
inject_resources() {
    log_step "Injecting resource initialization data..."
    
    # Get list of resources to inject (those with initialization data)
    # NOTE: Using standard service.schema.json structure
    local resources
    if ! resources=$(echo "$SERVICE_JSON" | jq -r '
      .resources | 
      to_entries[] | 
      .value | 
      to_entries[] | 
      select(.value.initialization != null) | 
      .key
    '); then
        log_error "Failed to extract resources with initialization data"
        return 1
    fi
    
    if [[ -z "$resources" ]]; then
        log_info "No resources to inject"
        return 0
    fi
    
    # Inject each resource
    for resource in $resources; do
        log_info "Injecting data for: $resource"
        
        # Extract initialization data and category for this specific resource
        local resource_info
        if ! resource_info=$(echo "$SERVICE_JSON" | jq -r --arg res "$resource" '
          .resources | 
          to_entries[] | 
          select(.value | has($res)) | 
          "\(.key)|\(.value[$res].initialization // {})"'
        ); then
            log_error "Failed to extract resource information for: $resource"
            return 1
        fi
        
        if [[ -z "$resource_info" ]]; then
            log_error "Resource '$resource' not found in any category"
            continue
        fi
        
        local category
        local init_data
        IFS='|' read -r category init_data <<< "$resource_info"
        
        # Find injection script for resource using dynamic discovery
        local inject_script=$(find_resource_inject_script "$resource" "$category")
        
        if [[ -z "$inject_script" ]]; then
            log_warning "  No injection handler for: $resource"
            continue
        fi
        
        if [[ -z "$init_data" || "$init_data" == "{}" ]]; then
            log_info "  No initialization data for: $resource"
            continue
        fi
        
        if [[ "$DRY_RUN" == true ]]; then
            log_info "  [DRY RUN] Would run: $inject_script --inject '<initialization_data>'"
            [[ "$VERBOSE" == true ]] && log_info "  [DRY RUN] Initialization data: $init_data"
        else
            # Run injection script with proper JSON data
            if [[ "$VERBOSE" == true ]]; then
                log_info "  Running: $inject_script --inject '<initialization_data>'"
                log_info "  Initialization data: $init_data"
            fi
            
            # Add rollback action before injection (escape JSON properly)
            local escaped_init_data
            escaped_init_data=$(printf '%s' "$init_data" | sed "s/'/'\\\\''/g")
            add_rollback_action "Rollback injection for $resource" "\"$inject_script\" --rollback '$escaped_init_data' || true"
            
            if "$inject_script" --inject "$init_data"; then
                log_success "  ✓ Injected data for: $resource (category: $category)"
            else
                log_error "  ✗ Failed to inject data for: $resource (category: $category)"
                execute_rollback
                return 1
            fi
        fi
    done
    
    log_success "Resource injection completed"
}

# Deploy the application
deploy_application() {
    log_step "Deploying application..."
    
    local startup_script="${SCENARIO_PATH}/deployment/startup.sh"
    
    if [[ -f "$startup_script" && -x "$startup_script" ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log_info "[DRY RUN] Would run: $startup_script deploy"
        else
            [[ "$VERBOSE" == true ]] && log_info "Running deployment script..."
            
            if "$startup_script" deploy; then
                log_success "Application deployed successfully"
            else
                log_error "Application deployment failed"
                return 1
            fi
        fi
    else
        log_warning "Deployment script not found: $startup_script"
        log_info "Skipping deployment step"
    fi
}

# Start monitoring
start_monitoring() {
    log_step "Starting application monitoring..."
    
    local monitor_script="${SCENARIO_PATH}/deployment/monitor.sh"
    
    if [[ -f "$monitor_script" && -x "$monitor_script" ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log_info "[DRY RUN] Would run: $monitor_script start"
        else
            [[ "$VERBOSE" == true ]] && log_info "Starting monitoring..."
            
            if "$monitor_script" start >/dev/null 2>&1; then
                log_success "Monitoring started"
            else
                log_warning "Failed to start monitoring (non-critical)"
            fi
        fi
    else
        [[ "$VERBOSE" == true ]] && log_info "No monitoring script found"
    fi
}

# Run integration tests
run_tests() {
    if [[ "$VALIDATION_MODE" == "none" ]]; then
        log_info "Skipping integration tests"
        return 0
    fi
    
    log_step "Running integration tests..."
    
    local test_script="${SCENARIO_PATH}/test.sh"
    
    if [[ -f "$test_script" && -x "$test_script" ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log_info "[DRY RUN] Would run: $test_script"
        else
            [[ "$VERBOSE" == true ]] && log_info "Running tests..."
            
            if "$test_script"; then
                log_success "Integration tests passed"
            else
                log_error "Integration tests failed"
                return 1
            fi
        fi
    else
        log_warning "Test script not found: $test_script"
    fi
}

# Show deployment summary
show_summary() {
    # NOTE: Using standard service.schema.json paths (.service.* not .metadata.*)
    local scenario_name
    local scenario_id
    local version
    scenario_name=$(echo "$SERVICE_JSON" | jq -r '.service.displayName // .service.name')
    scenario_id=$(echo "$SERVICE_JSON" | jq -r '.service.name')
    version=$(echo "$SERVICE_JSON" | jq -r '.service.version')
    
    echo ""
    log_success "🎉 Deployment Summary"
    echo "════════════════════════════════════════════════════════════════"
    echo "📋 Scenario: $scenario_name ($scenario_id)"
    echo "📌 Version: $version"
    echo "📁 Path: $SCENARIO_PATH"
    echo "🚀 Mode: $DEPLOYMENT_MODE"
    echo "✅ Status: Successfully Deployed"
    echo ""
    
    # Extract endpoints from service.json  
    # NOTE: Using deployment.urls from service.schema.json structure
    local endpoints
    if ! endpoints=$(echo "$SERVICE_JSON" | jq -r '.deployment.urls // {} | to_entries[] | "  \(.key): \(.value)"'); then
        [[ "$VERBOSE" == true ]] && log_info "No deployment URLs found in service.json"
        endpoints=""
    fi
    
    if [[ -n "$endpoints" ]]; then
        log_info "Application Endpoints:"
        echo "$endpoints"
        echo ""
    fi
    
    # Show resource-specific URLs
    # NOTE: Using standard service.schema.json structure for required resources
    local resources
    if ! resources=$(echo "$SERVICE_JSON" | jq -r '
      .resources | 
      to_entries[] | 
      .value | 
      to_entries[] | 
      select(.value.required == true) | 
      .key
    '); then
        log_warning "Failed to extract required resources for summary display"
        resources=""
    fi
    
    log_info "Resource Access:"
    for resource in $resources; do
        case "$resource" in
            n8n)
                echo "  📡 n8n Editor: http://localhost:5678"
                ;;
            windmill)
                echo "  🌪️ Windmill: http://localhost:5681"
                ;;
            postgres)
                echo "  🗄️ Database: postgresql://localhost:5432/${scenario_id//-/_}"
                ;;
            minio)
                echo "  📦 MinIO: http://localhost:9000"
                ;;
        esac
    done
    echo ""
    
    log_info "Management Commands:"
    echo "  📊 Status: $SCENARIO_PATH/deployment/monitor.sh status"
    echo "  📝 Logs: $SCENARIO_PATH/deployment/monitor.sh logs"
    echo "  🧪 Tests: $SCENARIO_PATH/test.sh"
    echo ""
}

################################################################################
# Main Execution
################################################################################

main() {
    # Parse arguments
    parse_args "$@"
    
    # Show banner
    log_banner "Vrooli Scenario-to-App Converter"
    log_info "Converting scenario: $SCENARIO_NAME"
    log_info "Deployment mode: $DEPLOYMENT_MODE"
    log_info "Validation mode: $VALIDATION_MODE"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warning "DRY RUN MODE - No actual changes will be made"
    fi
    
    echo ""
    
    # Phase 1: Validation
    log_phase "Phase 1: Validation"
    if ! validate_scenario; then
        log_error "Scenario validation failed"
        exit 1
    fi
    
    if ! validate_structure; then
        log_error "Structure validation failed"
        exit 1
    fi
    
    if ! validate_resources; then
        log_error "Resource validation failed"
        exit 1
    fi
    
    log_success "Validation completed"
    
    echo ""
    
    # Phase 2: Resource Injection
    log_phase "Phase 2: Resource Injection"
    if ! inject_resources; then
        log_error "Resource injection failed - rollback executed"
        exit 1
    fi
    log_success "Resource injection completed"
    
    echo ""
    
    # Phase 3: Deployment
    log_phase "Phase 3: Deployment"
    if ! deploy_application; then
        log_error "Application deployment failed"
        execute_rollback
        exit 1
    fi
    
    if ! start_monitoring; then
        log_warning "Monitoring startup failed (non-critical)"
    fi
    
    log_success "Deployment completed"
    
    echo ""
    
    # Phase 4: Testing
    log_phase "Phase 4: Testing"
    if ! run_tests; then
        log_error "Integration tests failed"
        execute_rollback
        exit 1
    fi
    log_success "Testing completed"
    
    echo ""
    
    # Summary
    if [[ "$DRY_RUN" == true ]]; then
        log_success "Dry run completed successfully!"
        log_info "Run without --dry-run to perform actual deployment"
    else
        show_summary
    fi
}

# Run main function
main "$@"
