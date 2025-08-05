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
# 🏗️ ARCHITECTURE: Why Validation-Only?
#
# The Problem with Deploy-Time Injection:
#   - Tight Coupling: Deployment success depends on external resource state
#   - Timing Complexity: Resources must be running during deployment  
#   - State Management: Deployment tool becomes responsible for resource lifecycle
#   - Rollback Complexity: Must understand and reverse resource-specific operations
#   - Environment Sensitivity: Different resource states across dev/staging/prod
#
# The Validation-Only Solution:
#   - Loose Coupling: Apps are self-contained and manage their own resources
#   - Deployment Simplicity: Only validates and packages, no external dependencies
#   - Runtime Responsibility: Apps handle resource startup and injection when ready
#   - Atomic Operations: Resource injection happens atomically during app startup
#   - Environment Agnostic: Apps work consistently across all deployment targets
#
# 🔄 NEW DEPLOYMENT FLOW:
#
# Deploy-Time (this script):
#   1. Validate service.json against schema
#   2. Verify all referenced files exist and are valid
#   3. Package scenario into deployable app structure  
#   4. Generate runtime injection manifests
#   5. Create deployment artifacts (scripts/containers/k8s)
#
# Runtime (Generated App Startup):
#   1. Inherit Vrooli's resource management capabilities
#   2. Start required resources using existing infrastructure
#   3. Wait for resource health using service.json specifications
#   4. Inject initialization data using validated manifests
#   5. Start application services
#   6. Begin monitoring and health checks
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
    cat << 'EOF'
Scenario-to-App Conversion Script

Converts Vrooli scenarios into deployable applications using the unified
service.json format and resource injection engine.

SCHEMA REFERENCE:
  All service.json files follow the official schema:
  .vrooli/schemas/service.schema.json
  
  Key paths used in this script:
  - .service.name           (scenario identifier)
  - .service.displayName    (human-readable name)
  - .resources.{category}.{resource}  (resource definitions)
  - .resources.{category}.{resource}.required  (not .optional!)
  - .resources.{category}.{resource}.initialization  (setup data)

Usage:
  ./scenario-to-app.sh <scenario-name> [options]

Options:
  --mode        Deployment mode (local, docker, k8s) [default: local]
  --validate    Validation mode (none, basic, full) [default: full]
  --dry-run     Show what would be done without executing
  --verbose     Enable verbose logging
  --help        Show this help message
EOF
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

# Extract resources from service.json based on condition
extract_resources_by_condition() {
    local condition="$1"
    local error_message="$2"
    
    local resources
    if ! resources=$(echo "$SERVICE_JSON" | jq -r "
      .resources | 
      to_entries[] | 
      .value | 
      to_entries[] | 
      select(.value.$condition) | 
      .key
    "); then
        log_error "$error_message"
        return 1
    fi
    
    echo "$resources"
    return 0
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
    if ! init_resources=$(extract_resources_by_condition "initialization != null" "Failed to extract initialization resources from service.json"); then
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
    if ! required_resources=$(extract_resources_by_condition "required == true" "Failed to extract required resources from service.json"); then
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
        
    done
    
    log_success "Resource validation completed"
}

################################################################################
# Enhanced Validation for Runtime Injection Architecture
################################################################################

# Comprehensive validation for injection readiness (no actual injection)
# This replaces resource connectivity checks with file/schema validation
validate_injection_readiness() {
    local scenario_path="$1"
    local service_config="${scenario_path}/service.json"
    
    log_step "Validating injection readiness (validation-only mode)..."
    
    # 1. Validate service.json schema compliance
    log_info "Checking service.json schema compliance..."
    local schema_path="${PROJECT_ROOT}/.vrooli/schemas/service.schema.json"
    
    if [[ -f "$schema_path" ]]; then
        # Use ajv-cli if available, otherwise basic jq validation
        if command -v ajv &>/dev/null; then
            if ! ajv validate -s "$schema_path" -d "$service_config" 2>/dev/null; then
                log_error "Service config does not match schema"
                return 1
            fi
        else
            # Fallback to basic structure validation
            if ! validate_service_config "$service_config" "full"; then
                return 1
            fi
        fi
        log_success "✓ Schema validation passed"
    else
        log_warning "Schema file not found, using basic validation"
        if ! validate_service_config "$service_config" "full"; then
            return 1
        fi
    fi
    
    # 2. Validate all referenced files exist
    log_info "Checking referenced files..."
    local missing_files=()
    
    # Check database files
    local db_files
    db_files=$(echo "$SERVICE_JSON" | jq -r '
        .resources.storage | 
        to_entries[] | 
        select(.value.initialization.database // false) | 
        .value.initialization.database[]? // empty
    ' 2>/dev/null || true)
    
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        local full_path="${scenario_path}/${file}"
        if [[ ! -f "$full_path" ]]; then
            missing_files+=("$file")
            log_error "  ✗ Missing database file: $file"
        else
            [[ "$VERBOSE" == true ]] && log_success "  ✓ Found: $file"
        fi
    done <<< "$db_files"
    
    # Check workflow files
    local workflow_files
    workflow_files=$(echo "$SERVICE_JSON" | jq -r '
        .resources.automation | 
        to_entries[] | 
        select(.value.initialization.workflows // false) | 
        .value.initialization.workflows[]? // empty
    ' 2>/dev/null || true)
    
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        local full_path="${scenario_path}/${file}"
        if [[ ! -f "$full_path" ]]; then
            missing_files+=("$file")
            log_error "  ✗ Missing workflow file: $file"
        else
            [[ "$VERBOSE" == true ]] && log_success "  ✓ Found: $file"
        fi
    done <<< "$workflow_files"
    
    # Check script files
    local script_files
    script_files=$(echo "$SERVICE_JSON" | jq -r '
        .resources.automation | 
        to_entries[] | 
        select(.value.initialization.scripts // false) | 
        .value.initialization.scripts[].file // empty
    ' 2>/dev/null || true)
    
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        local full_path="${scenario_path}/${file}"
        if [[ ! -f "$full_path" ]]; then
            missing_files+=("$file")
            log_error "  ✗ Missing script file: $file"
        else
            [[ "$VERBOSE" == true ]] && log_success "  ✓ Found: $file"
        fi
    done <<< "$script_files"
    
    # 3. Validate JSON/YAML files are syntactically valid
    log_info "Validating file syntax..."
    local invalid_files=()
    
    # Validate JSON files
    for json_file in "${scenario_path}"/**/*.json; do
        [[ -f "$json_file" ]] || continue
        if ! jq empty "$json_file" 2>/dev/null; then
            invalid_files+=("${json_file#$scenario_path/}")
            log_error "  ✗ Invalid JSON: ${json_file#$scenario_path/}"
        fi
    done
    
    # Validate YAML files if yq is available
    if command -v yq &>/dev/null; then
        for yaml_file in "${scenario_path}"/**/*.{yml,yaml}; do
            [[ -f "$yaml_file" ]] || continue
            if ! yq eval '.' "$yaml_file" >/dev/null 2>&1; then
                invalid_files+=("${yaml_file#$scenario_path/}")
                log_error "  ✗ Invalid YAML: ${yaml_file#$scenario_path/}"
            fi
        done
    fi
    
    # 4. Validate resource configurations match expected patterns
    log_info "Validating resource configurations..."
    
    # Extract all resources
    local all_resources
    all_resources=$(echo "$SERVICE_JSON" | jq -r '
        .resources | 
        to_entries[] | 
        .value | 
        to_entries[] | 
        .key
    ' 2>/dev/null || true)
    
    # 5. Check required inject.sh scripts exist (for runtime use)
    log_info "Checking injection handlers (for runtime use)..."
    local resources_without_handlers=()
    
    while IFS= read -r resource; do
        [[ -z "$resource" ]] && continue
        
        # Check if resource has an inject.sh script
        local inject_script
        inject_script=$(find "${PROJECT_ROOT}/scripts/resources" \
            -name "inject.sh" \
            -path "*/${resource}/*" \
            2>/dev/null | head -1)
        
        if [[ -z "$inject_script" ]]; then
            # Check if resource has initialization data
            local has_init_data
            has_init_data=$(echo "$SERVICE_JSON" | jq -r "
                .resources | 
                to_entries[] | 
                .value.\"$resource\".initialization // false
            " 2>/dev/null || echo "false")
            
            if [[ "$has_init_data" != "false" ]]; then
                resources_without_handlers+=("$resource")
                log_warning "  ⚠ Resource '$resource' has initialization data but no injection handler"
            fi
        else
            [[ "$VERBOSE" == true ]] && log_success "  ✓ Handler found: $resource"
        fi
    done <<< "$all_resources"
    
    # 6. Check for conflicting resource requirements
    log_info "Checking for resource conflicts..."
    
    # Check port conflicts
    local ports=()
    local port_conflicts=()
    
    local resource_ports
    resource_ports=$(echo "$SERVICE_JSON" | jq -r '
        .resources | 
        to_entries[] | 
        .value | 
        to_entries[] | 
        select(.value.port // false) | 
        "\(.key):\(.value.port)"
    ' 2>/dev/null || true)
    
    while IFS=: read -r resource port; do
        [[ -z "$port" ]] && continue
        if [[ " ${ports[@]} " =~ " ${port} " ]]; then
            port_conflicts+=("Port $port used by multiple resources")
        else
            ports+=("$port")
        fi
    done <<< "$resource_ports"
    
    # Report validation results
    local validation_passed=true
    
    if [[ ${#missing_files[@]} -gt 0 ]]; then
        log_error "Missing files: ${#missing_files[@]}"
        validation_passed=false
    fi
    
    if [[ ${#invalid_files[@]} -gt 0 ]]; then
        log_error "Invalid files: ${#invalid_files[@]}"
        validation_passed=false
    fi
    
    if [[ ${#resources_without_handlers[@]} -gt 0 ]]; then
        log_warning "Resources without handlers: ${#resources_without_handlers[@]}"
        # This is a warning, not a failure
    fi
    
    if [[ ${#port_conflicts[@]} -gt 0 ]]; then
        log_error "Port conflicts detected: ${port_conflicts[*]}"
        validation_passed=false
    fi
    
    if [[ "$validation_passed" == true ]]; then
        log_success "✅ Injection readiness validation passed"
        log_info "Note: This is validation-only mode. Actual resource injection will happen at runtime."
        return 0
    else
        log_error "❌ Injection readiness validation failed"
        return 1
    fi
}

################################################################################
# Packaging Functions for Self-Contained Apps
################################################################################

# Package scenario into deployable app structure
package_scenario_app() {
    local scenario_path="$1"
    local target_dir="$2"
    local deployment_mode="${3:-local}"  # local, docker, k8s
    
    log_step "Packaging scenario app for $deployment_mode deployment..."
    
    # Create deployment directory structure
    mkdir -p "${target_dir}"/{bin,config,data,scripts,manifests}
    
    # Copy scenario files to target location
    log_info "Copying scenario files..."
    cp -r "${scenario_path}"/* "${target_dir}/data/" 2>/dev/null || true
    
    # Copy service.json to config directory
    cp "${scenario_path}/service.json" "${target_dir}/config/"
    
    # Generate runtime injection manifest
    log_info "Generating runtime injection manifest..."
    generate_injection_manifest "$scenario_path" "${target_dir}/manifests/injection.json"
    
    # Generate app-specific startup script
    log_info "Generating startup script..."
    generate_startup_script "$deployment_mode" "$target_dir"
    
    # Set up monitoring configurations from service.json
    if [[ $(echo "$SERVICE_JSON" | jq -r '.monitoring // false') != "false" ]]; then
        log_info "Setting up monitoring configuration..."
        echo "$SERVICE_JSON" | jq '.monitoring' > "${target_dir}/config/monitoring.json"
    fi
    
    # Generate health check scripts
    log_info "Generating health check scripts..."
    generate_health_check_scripts "$target_dir"
    
    # Generate cleanup procedures
    log_info "Generating cleanup scripts..."
    generate_cleanup_scripts "$target_dir"
    
    # Create environment-specific configurations
    case "$deployment_mode" in
        local)
            create_local_config "$target_dir"
            ;;
        docker)
            create_docker_config "$target_dir"
            ;;
        k8s)
            create_k8s_config "$target_dir"
            ;;
    esac
    
    # Make scripts executable
    chmod +x "${target_dir}"/bin/* 2>/dev/null || true
    chmod +x "${target_dir}"/scripts/* 2>/dev/null || true
    
    log_success "✅ Scenario app packaged successfully"
    return 0
}

# Generate injection manifest for runtime use
generate_injection_manifest() {
    local scenario_path="$1"
    local output_file="$2"
    
    # Extract resource initialization data
    local manifest
    manifest=$(echo "$SERVICE_JSON" | jq '{
        resources: [
            .resources | 
            to_entries[] | 
            .value | 
            to_entries[] | 
            select(.value.initialization // false) | 
            {
                category: (.key | split(".")[0]),
                name: .key,
                type: .value.type,
                initialization: .value.initialization,
                inject_script: null
            }
        ],
        scenario: {
            name: .service.name,
            version: .service.version,
            path: "'$scenario_path'"
        }
    }')
    
    # Add inject script paths
    local updated_manifest="$manifest"
    for resource in $(echo "$manifest" | jq -r '.resources[].name'); do
        local inject_script
        inject_script=$(find "${PROJECT_ROOT}/scripts/resources" \
            -name "inject.sh" \
            -path "*/${resource}/*" \
            2>/dev/null | head -1 || echo "")
        
        if [[ -n "$inject_script" ]]; then
            # Store relative path for portability
            inject_script="${inject_script#$PROJECT_ROOT/}"
            updated_manifest=$(echo "$updated_manifest" | jq \
                --arg resource "$resource" \
                --arg script "$inject_script" \
                '(.resources[] | select(.name == $resource) | .inject_script) = $script')
        fi
    done
    
    echo "$updated_manifest" | jq '.' > "$output_file"
}

# Generate startup script for the app
generate_startup_script() {
    local deployment_mode="$1"
    local target_dir="$2"
    local script_path="${target_dir}/bin/start.sh"
    
    # Get scenario metadata
    local scenario_name=$(echo "$SERVICE_JSON" | jq -r '.service.name')
    local scenario_display_name=$(echo "$SERVICE_JSON" | jq -r '.service.displayName // .service.name')
    
    cat > "$script_path" << 'EOF'
#!/usr/bin/env bash
# Auto-generated startup script for scenario app
# Generated by scenario-to-app.sh

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
APP_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)

# Source Vrooli resource management functions
# These will be provided by the Vrooli runtime environment
if [[ -f "/opt/vrooli/scripts/resources/lib/resource-functions.sh" ]]; then
    source "/opt/vrooli/scripts/resources/lib/resource-functions.sh"
elif [[ -f "${VROOLI_RUNTIME_DIR}/scripts/resources/lib/resource-functions.sh" ]]; then
    source "${VROOLI_RUNTIME_DIR}/scripts/resources/lib/resource-functions.sh"
else
    echo "ERROR: Vrooli resource functions not found"
    echo "This app requires Vrooli runtime environment"
    exit 1
fi

# Load injection manifest
INJECTION_MANIFEST="${APP_ROOT}/manifests/injection.json"

log_info() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $*"
}

log_success() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $*"
}

log_error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

main() {
EOF
    
    # Add scenario-specific metadata
    cat >> "$script_path" << EOF
    log_info "Starting ${scenario_display_name} (${deployment_mode} mode)"
    
    # 1. Start required resources (inherited from Vrooli)
    log_info "Starting required resources..."
    if ! start_required_resources; then
        log_error "Failed to start required resources"
        exit 1
    fi
    
    # 2. Wait for resource health using service.json specifications
    log_info "Waiting for resources to be healthy..."
    if ! wait_for_resource_health; then
        log_error "Resources failed health checks"
        exit 1
    fi
    
    # 3. Inject initialization data using validated manifests
    log_info "Injecting initialization data..."
    if [[ -f "\$INJECTION_MANIFEST" ]]; then
        if ! inject_initialization_data "\$INJECTION_MANIFEST"; then
            log_error "Failed to inject initialization data"
            exit 1
        fi
    else
        log_info "No initialization data to inject"
    fi
    
    # 4. Start application services
    log_info "Starting application services..."
    if ! start_application_services; then
        log_error "Failed to start application services"
        exit 1
    fi
    
    # 5. Begin health monitoring
    log_info "Starting health monitoring..."
    start_health_monitoring &
    
    log_success "${scenario_display_name} started successfully"
    
    # Keep the script running
    wait
}

# Trap signals for cleanup
trap 'cleanup_and_exit' EXIT INT TERM

cleanup_and_exit() {
    log_info "Shutting down ${scenario_display_name}..."
    stop_health_monitoring
    stop_application_services
    stop_required_resources
    log_info "Shutdown complete"
}

main "\$@"
EOF
    
    chmod +x "$script_path"
}

# Generate health check scripts
generate_health_check_scripts() {
    local target_dir="$1"
    local health_script="${target_dir}/scripts/health-check.sh"
    
    cat > "$health_script" << 'EOF'
#!/usr/bin/env bash
# Health check script for scenario app

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
APP_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
CONFIG_FILE="${APP_ROOT}/config/service.json"

# Extract health check configurations
HEALTH_CHECKS=$(jq -r '.resources | 
    to_entries[] | 
    .value | 
    to_entries[] | 
    select(.value.healthCheck // false) | 
    {
        resource: .key,
        check: .value.healthCheck
    }' "$CONFIG_FILE" 2>/dev/null || echo '[]')

# Run health checks
all_healthy=true

echo "$HEALTH_CHECKS" | jq -c '.[]' | while read -r check; do
    resource=$(echo "$check" | jq -r '.resource')
    check_type=$(echo "$check" | jq -r '.check.type // "tcp"')
    
    case "$check_type" in
        tcp)
            port=$(echo "$check" | jq -r '.check.port // 0')
            if [[ "$port" -ne 0 ]]; then
                if nc -z localhost "$port" 2>/dev/null; then
                    echo "✓ $resource: healthy (port $port)"
                else
                    echo "✗ $resource: unhealthy (port $port unreachable)"
                    all_healthy=false
                fi
            fi
            ;;
        http)
            endpoint=$(echo "$check" | jq -r '.check.endpoint // "/"')
            port=$(echo "$check" | jq -r '.check.port // 80')
            if curl -sf "http://localhost:${port}${endpoint}" >/dev/null; then
                echo "✓ $resource: healthy (HTTP endpoint)"
            else
                echo "✗ $resource: unhealthy (HTTP endpoint failed)"
                all_healthy=false
            fi
            ;;
    esac
done

if [[ "$all_healthy" == true ]]; then
    exit 0
else
    exit 1
fi
EOF
    
    chmod +x "$health_script"
}

# Generate cleanup scripts
generate_cleanup_scripts() {
    local target_dir="$1"
    local cleanup_script="${target_dir}/scripts/cleanup.sh"
    
    cat > "$cleanup_script" << 'EOF'
#!/usr/bin/env bash
# Cleanup script for scenario app

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
APP_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)

log_info() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $*"
}

# Stop all services
log_info "Stopping application services..."
pkill -f "start.sh" 2>/dev/null || true

# Clean up temporary files
log_info "Cleaning up temporary files..."
rm -rf "${APP_ROOT}/tmp/*" 2>/dev/null || true

# Clean up logs older than 7 days
log_info "Cleaning up old logs..."
find "${APP_ROOT}/logs" -type f -mtime +7 -delete 2>/dev/null || true

log_info "Cleanup complete"
EOF
    
    chmod +x "$cleanup_script"
}

# Create local deployment configuration
create_local_config() {
    local target_dir="$1"
    
    cat > "${target_dir}/config/deployment.json" << EOF
{
    "mode": "local",
    "runtime": {
        "vrooli_dir": "${PROJECT_ROOT}",
        "resource_scripts": "${PROJECT_ROOT}/scripts/resources"
    },
    "environment": {
        "NODE_ENV": "development",
        "LOG_LEVEL": "debug"
    }
}
EOF
}

# Create Docker deployment configuration  
create_docker_config() {
    local target_dir="$1"
    
    # Generate Dockerfile
    cat > "${target_dir}/Dockerfile" << 'EOF'
FROM node:18-alpine

# Install dependencies
RUN apk add --no-cache bash curl jq

# Copy app files
WORKDIR /app
COPY . .

# Make scripts executable
RUN chmod +x /app/bin/* /app/scripts/*

# Expose common ports (customize based on service.json)
EXPOSE 3000 8080

# Start the app
CMD ["/app/bin/start.sh"]
EOF
    
    # Generate docker-compose.yml
    local compose_file="${target_dir}/docker-compose.yml"
    echo "version: '3.8'" > "$compose_file"
    echo "services:" >> "$compose_file"
    echo "  app:" >> "$compose_file"
    echo "    build: ." >> "$compose_file"
    echo "    environment:" >> "$compose_file"
    echo "      - NODE_ENV=production" >> "$compose_file"
    echo "    networks:" >> "$compose_file"
    echo "      - vrooli-network" >> "$compose_file"
    echo "" >> "$compose_file"
    echo "networks:" >> "$compose_file"
    echo "  vrooli-network:" >> "$compose_file"
    echo "    external: true" >> "$compose_file"
}

# Create Kubernetes deployment configuration
create_k8s_config() {
    local target_dir="$1"
    local k8s_dir="${target_dir}/k8s"
    mkdir -p "$k8s_dir"
    
    # Extract metadata
    local app_name=$(echo "$SERVICE_JSON" | jq -r '.service.name')
    local app_version=$(echo "$SERVICE_JSON" | jq -r '.service.version')
    
    # Generate ConfigMap for configuration
    cat > "${k8s_dir}/configmap.yaml" << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: ${app_name}-config
data:
  service.json: |
$(cat "${target_dir}/config/service.json" | sed 's/^/    /')
EOF
    
    # Generate Deployment
    cat > "${k8s_dir}/deployment.yaml" << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${app_name}
  labels:
    app: ${app_name}
    version: ${app_version}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${app_name}
  template:
    metadata:
      labels:
        app: ${app_name}
        version: ${app_version}
    spec:
      containers:
      - name: ${app_name}
        image: ${app_name}:${app_version}
        command: ["/app/bin/start.sh"]
        volumeMounts:
        - name: config
          mountPath: /app/config
        env:
        - name: NODE_ENV
          value: "production"
      volumes:
      - name: config
        configMap:
          name: ${app_name}-config
EOF
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
    
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
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
    local config_dir
    config_dir=$(dirname "$service_config")
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

################################################################################
# Main Execution
################################################################################

main() {
    # Parse arguments
    parse_args "$@"
    
    # Show banner
    log_banner "Vrooli Scenario-to-App Converter (Validation-Only Mode)"
    log_info "Converting scenario: $SCENARIO_NAME"
    log_info "Deployment mode: $DEPLOYMENT_MODE"
    log_info "Architecture: Validation-only with runtime injection"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warning "DRY RUN MODE - No actual changes will be made"
    fi
    
    echo ""
    
    # Phase 1: Comprehensive Validation
    log_phase "Phase 1: Injection Readiness Validation"
    
    # Basic scenario validation
    if ! validate_scenario; then
        log_error "Scenario validation failed"
        exit 1
    fi
    
    # Enhanced validation for injection readiness
    if ! validate_injection_readiness "$SCENARIO_PATH"; then
        log_error "Injection readiness validation failed"
        exit 1
    fi
    
    log_success "✅ All validations passed"
    
    echo ""
    
    # Phase 2: Package Scenario App
    log_phase "Phase 2: Packaging Self-Contained App"
    
    # Determine output directory
    local output_dir="${OUTPUT_DIR:-${PROJECT_ROOT}/generated-apps/${SCENARIO_NAME}}"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would package app to: $output_dir"
        log_info "[DRY RUN] Deployment mode: $DEPLOYMENT_MODE"
    else
        # Create output directory
        mkdir -p "$output_dir"
        
        # Package the scenario
        if ! package_scenario_app "$SCENARIO_PATH" "$output_dir" "$DEPLOYMENT_MODE"; then
            log_error "Failed to package scenario app"
            exit 1
        fi
        
        log_success "✅ App packaged successfully"
    fi
    
    echo ""
    
    # Phase 3: Generate Deployment Artifacts
    log_phase "Phase 3: Deployment Artifacts"
    
    case "$DEPLOYMENT_MODE" in
        local)
            log_info "Generated local deployment scripts"
            if [[ "$DRY_RUN" != true ]]; then
                log_info "Start script: ${output_dir}/bin/start.sh"
                log_info "Health check: ${output_dir}/scripts/health-check.sh"
            fi
            ;;
        docker)
            log_info "Generated Docker deployment files"
            if [[ "$DRY_RUN" != true ]]; then
                log_info "Dockerfile: ${output_dir}/Dockerfile"
                log_info "Compose file: ${output_dir}/docker-compose.yml"
                log_info ""
                log_info "To build and run:"
                log_info "  cd ${output_dir}"
                log_info "  docker-compose up --build"
            fi
            ;;
        k8s)
            log_info "Generated Kubernetes manifests"
            if [[ "$DRY_RUN" != true ]]; then
                log_info "ConfigMap: ${output_dir}/k8s/configmap.yaml"
                log_info "Deployment: ${output_dir}/k8s/deployment.yaml"
                log_info ""
                log_info "To deploy:"
                log_info "  kubectl apply -f ${output_dir}/k8s/"
            fi
            ;;
    esac
    
    echo ""
    
    # Summary
    log_phase "Summary"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_success "✅ Dry run completed successfully!"
        log_info "The app would be packaged with:"
        log_info "  - Runtime injection manifest"
        log_info "  - Self-contained startup script"
        log_info "  - Health monitoring"
        log_info "  - Deployment configurations"
        log_info ""
        log_info "Run without --dry-run to generate the actual app"
    else
        log_success "✅ Scenario app generated successfully!"
        log_info ""
        log_info "📦 App location: $output_dir"
        log_info ""
        log_info "🚀 Next steps:"
        log_info "  1. Review generated files in $output_dir"
        log_info "  2. Deploy using the $DEPLOYMENT_MODE instructions above"
        log_info "  3. The app will handle resource startup and injection at runtime"
        log_info ""
        log_info "Note: This is a self-contained app that inherits Vrooli's"
        log_info "      resource management capabilities. No external resources"
        log_info "      need to be running during deployment."
    fi
}

# Execute main function
main "$@"
