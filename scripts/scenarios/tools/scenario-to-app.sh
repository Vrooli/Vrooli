#!/usr/bin/env bash

################################################################################
# Scenario-to-App Conversion Script
# 
# Converts Vrooli scenarios into deployable applications using the unified
# service.json format and resource injection engine.
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
INJECTION_DIR="${PROJECT_ROOT}/scripts/scenarios/injection"
SERVICE_CONFIG="${PROJECT_ROOT}/.vrooli/service.json"

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

################################################################################
# Helper Functions
################################################################################

# Show usage information
show_usage() {
    head -n 20 "${BASH_SOURCE[0]}" | grep "^#" | grep -v "^#!/" | sed 's/^# //'
}

# Parse command line arguments
parse_args() {
    if [[ $# -eq 0 ]]; then
        log_error "No scenario specified"
        show_usage
        exit 1
    fi
    
    SCENARIO_NAME="$1"
    shift
    
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
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
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
    
    # Load service.json
    if [[ -f "$SERVICE_JSON_PATH" ]]; then
        log_info "Loading service.json from $SERVICE_JSON_PATH"
        SERVICE_JSON=$(cat "$SERVICE_JSON_PATH")
        log_info "Loaded service.json successfully"
        [[ "$VERBOSE" == true ]] && log_info "Loaded service.json from $SERVICE_JSON_PATH"
    else
        log_error "Failed to load service.json"
        exit 1
    fi
}

# Validate scenario structure based on service.json
validate_structure() {
    if [[ "$VALIDATION_MODE" == "none" ]]; then
        log_info "Skipping structure validation"
        return 0
    fi
    
    log_step "Validating scenario structure..."
    
    # Extract scenario name and version from service.json
    local scenario_name=$(echo "$SERVICE_JSON" | jq -r '.metadata.name // ""')
    local scenario_version=$(echo "$SERVICE_JSON" | jq -r '.metadata.version // ""')
    
    if [[ -z "$scenario_name" ]]; then
        log_error "Invalid service.json: missing metadata.name"
        return 1
    fi
    
    [[ "$VERBOSE" == true ]] && log_info "Scenario: $scenario_name v$scenario_version"
    
    # Check for required initialization files
    local init_resources=$(echo "$SERVICE_JSON" | jq -r '.spec.scenarios.initialization.resources // {} | keys[]' 2>/dev/null)
    
    for resource in $init_resources; do
        [[ "$VERBOSE" == true ]] && log_info "Checking initialization data for: $resource"
        
        # Check for database files
        local db_files=$(echo "$SERVICE_JSON" | jq -r ".spec.scenarios.initialization.resources.$resource.data[]?.file // empty" 2>/dev/null)
        for file in $db_files; do
            local full_path="${PROJECT_ROOT}/$file"
            if [[ -f "$full_path" ]]; then
                [[ "$VERBOSE" == true ]] && log_success "  ✓ Found: $file"
            else
                log_error "  ✗ Missing: $file"
                return 1
            fi
        done
        
        # Check for workflow files
        local workflow_files=$(echo "$SERVICE_JSON" | jq -r ".spec.scenarios.initialization.resources.$resource.workflows[]?.file // empty" 2>/dev/null)
        for file in $workflow_files; do
            local full_path="${PROJECT_ROOT}/$file"
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
    local required_resources=$(echo "$SERVICE_JSON" | jq -r '.spec.dependencies.resources[] | select(.optional == false) | .name' 2>/dev/null)
    
    if [[ -z "$required_resources" ]]; then
        log_warning "No required resources defined"
        return 0
    fi
    
    [[ "$VERBOSE" == true ]] && log_info "Required resources: $(echo $required_resources | tr '\n' ' ')"
    
    # Check each resource
    for resource in $required_resources; do
        # Check if resource has an inject.sh script
        local inject_script=$(find "${PROJECT_ROOT}/scripts/resources" -name "inject.sh" -path "*/${resource}/*" 2>/dev/null | head -1)
        
        if [[ -n "$inject_script" ]]; then
            [[ "$VERBOSE" == true ]] && log_success "  ✓ Injection handler found for: $resource"
        else
            log_warning "  ⚠ No injection handler for: $resource"
        fi
        
        # TODO: Add actual resource health checks here
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
    
    # Check resources structure
    local resource_categories=("storage" "ai" "automation")
    for category in "${resource_categories[@]}"; do
        if [[ $(echo "$config_content" | jq -r ".resources.$category // false") == "false" ]]; then
            log_warning "Missing resource category: $category (will be created)"
        fi
    done
    
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

# SAFE Generate service.json configuration
generate_resources_config() {
    log_step "Safely updating resources configuration..."
    
    # Run pre-flight safety checks
    if ! preflight_safety_check "$SERVICE_CONFIG"; then
        log_error "Pre-flight safety checks failed"
        return 1
    fi
    
    # Extract resources from scenario service.json
    local resources=$(echo "$SERVICE_JSON" | jq -r '.spec.dependencies.resources[] | .name' 2>/dev/null)
    
    if [[ -z "$resources" ]]; then
        log_warning "No resources defined in service.json"
        return 0
    fi
    
    # SAFETY: Create timestamped backup BEFORE any modifications
    local backup_file=""
    if [[ -f "$SERVICE_CONFIG" ]]; then
        if backup_file=$(create_safe_backup "$SERVICE_CONFIG" "scenario-deployment"); then
            log_info "Created safety backup: $backup_file"
        else
            log_error "Failed to create safety backup"
            return 1
        fi
    fi
    
    # Read existing configuration or create minimal structure
    local existing_config='{}'
    if [[ -f "$SERVICE_CONFIG" ]]; then
        if ! existing_config=$(cat "$SERVICE_CONFIG"); then
            log_error "Failed to read existing service config"
            return 1
        fi
        log_info "Loaded existing service configuration"
    else
        # Create minimal structure for new installations
        existing_config='{"resources": {"storage": {}, "ai": {}, "automation": {}, "agents": {}, "execution": {}, "search": {}}}'
        log_info "Creating new service configuration structure"
    fi
    
    # Process each required resource
    local updated_config="$existing_config"
    
    for resource in $resources; do
        local resource_type=$(echo "$SERVICE_JSON" | jq -r ".spec.dependencies.resources[] | select(.name == \"$resource\") | .type" 2>/dev/null)
        local optional=$(echo "$SERVICE_JSON" | jq -r ".spec.dependencies.resources[] | select(.name == \"$resource\") | .optional" 2>/dev/null)
        
        # Skip optional resources (don't auto-enable them)
        if [[ "$optional" == "true" ]]; then
            [[ "$VERBOSE" == true ]] && log_info "Skipping optional resource: $resource"
            continue
        fi
        
        [[ "$VERBOSE" == true ]] && log_info "Processing required resource: $resource (type: $resource_type)"
        
        # Map resource type to config category
        local category=""
        case "$resource_type" in
            "database"|"cache"|"storage"|"vectordb"|"timeseries"|"security")
                category="storage"
                ;;
            "ai")
                category="ai"
                ;;
            "automation")
                category="automation"
                ;;
            "agent")
                category="agents"
                ;;
            "execution")
                category="execution"
                ;;
            "search")
                category="search"
                ;;
            *)
                log_warning "Unknown resource type: $resource_type for $resource"
                continue
                ;;
        esac
        
        # SAFE OPERATION: Only modify the "enabled" field, preserve all other config
        local resource_exists=$(echo "$updated_config" | jq -r ".resources.$category.$resource // false" 2>/dev/null)
        
        if [[ "$resource_exists" != "false" && "$resource_exists" != "null" ]]; then
            # Resource exists - only enable it (preserve all other settings)
            updated_config=$(echo "$updated_config" | jq ".resources.$category.$resource.enabled = true")
            log_info "✓ Enabled existing resource: $resource"
        else
            # Resource doesn't exist - add minimal configuration
            local minimal_resource='{"enabled": true, "type": "'$resource'"}'
            updated_config=$(echo "$updated_config" | jq ".resources.$category.$resource = $minimal_resource")
            log_info "✓ Added new resource: $resource"
        fi
    done
    
    # ATOMIC WRITE: Write to temporary file first, then move
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would safely update service config: $SERVICE_CONFIG"
        [[ "$VERBOSE" == true ]] && echo "$updated_config" | jq '.resources'
    else
        local temp_file="${SERVICE_CONFIG}.tmp.$$"
        
        # Ensure directory exists
        mkdir -p "$(dirname "$SERVICE_CONFIG")"
        
        # Write to temporary file
        if echo "$updated_config" | jq '.' > "$temp_file"; then
            # Atomic move
            mv "$temp_file" "$SERVICE_CONFIG"
            log_success "✓ Safely updated service configuration"
        else
            rm -f "$temp_file"
            log_error "Failed to write updated configuration"
            return 1
        fi
        
        # Verify the update worked
        if jq empty "$SERVICE_CONFIG" 2>/dev/null; then
            log_success "Configuration validation passed"
        else
            log_error "Configuration file is corrupted!"
            # Restore from backup if available
            if [[ -n "$backup_file" && -f "$backup_file" ]]; then
                log_info "Restoring from backup: $backup_file"
                cp "$backup_file" "$SERVICE_CONFIG"
            fi
            return 1
        fi
    fi
}

# Call injection engine to initialize resources
inject_resources() {
    log_step "Injecting resource initialization data..."
    
    local injection_engine="${INJECTION_DIR}/engine.sh"
    
    if [[ ! -f "$injection_engine" ]]; then
        log_warning "Injection engine not found: $injection_engine"
        log_warning "Skipping resource injection"
        return 0
    fi
    
    # Get list of resources to inject
    local resources=$(echo "$SERVICE_JSON" | jq -r '.spec.scenarios.initialization.resources // {} | keys[]' 2>/dev/null)
    
    if [[ -z "$resources" ]]; then
        log_info "No resources to inject"
        return 0
    fi
    
    # Inject each resource
    for resource in $resources; do
        log_info "Injecting data for: $resource"
        
        # Find injection script for resource
        local inject_script=$(find "${PROJECT_ROOT}/scripts/resources" -name "inject.sh" -path "*/${resource}/*" 2>/dev/null | head -1)
        
        if [[ -z "$inject_script" ]]; then
            log_warning "  No injection handler for: $resource"
            continue
        fi
        
        if [[ "$DRY_RUN" == true ]]; then
            log_info "  [DRY RUN] Would run: $inject_script --service-json $SERVICE_JSON_PATH"
        else
            # Run injection script
            if [[ "$VERBOSE" == true ]]; then
                "$inject_script" --service-json "$SERVICE_JSON_PATH" --verbose
            else
                "$inject_script" --service-json "$SERVICE_JSON_PATH"
            fi
            
            if [[ $? -eq 0 ]]; then
                log_success "  ✓ Injected data for: $resource"
            else
                log_error "  ✗ Failed to inject data for: $resource"
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
    local scenario_name=$(echo "$SERVICE_JSON" | jq -r '.metadata.displayName // .metadata.name')
    local scenario_id=$(echo "$SERVICE_JSON" | jq -r '.metadata.name')
    local version=$(echo "$SERVICE_JSON" | jq -r '.metadata.version')
    
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
    local endpoints=$(echo "$SERVICE_JSON" | jq -r '.spec.serve.endpoints[]? | "  \(.name): \(.protocol)://localhost:\(.port)\(.path)"' 2>/dev/null)
    
    if [[ -n "$endpoints" ]]; then
        log_info "Application Endpoints:"
        echo "$endpoints"
        echo ""
    fi
    
    # Show resource-specific URLs
    local resources=$(echo "$SERVICE_JSON" | jq -r '.spec.dependencies.resources[] | select(.optional == false) | .name' 2>/dev/null)
    
    log_info "Resource Access:"
    for resource in $resources; do
        case "$resource" in
            n8n)
                echo "  📡 n8n Editor: http://localhost:5678"
                ;;
            windmill)
                echo "  🌪️ Windmill: http://localhost:8000"
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
    validate_scenario
    log_info "After validate_scenario"
    validate_structure
    log_info "After validate_structure"
    validate_resources
    log_info "After validate_resources"
    log_success "Validation completed"
    
    echo ""
    
    # Phase 2: Configuration
    log_phase "Phase 2: Configuration"
    generate_resources_config
    log_success "Configuration completed"
    
    echo ""
    
    # Phase 3: Resource Injection
    log_phase "Phase 3: Resource Injection"
    inject_resources
    log_success "Resource injection completed"
    
    echo ""
    
    # Phase 4: Deployment
    log_phase "Phase 4: Deployment"
    deploy_application
    start_monitoring
    log_success "Deployment completed"
    
    echo ""
    
    # Phase 5: Testing
    log_phase "Phase 5: Testing"
    run_tests
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
