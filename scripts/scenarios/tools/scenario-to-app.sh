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

# Logging functions will be provided by cli-interface.sh module
# (imported below after SCRIPT_DIR is defined)

# Script location and imports
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SCENARIOS_DIR="${PROJECT_ROOT}/scripts/scenarios/core"

# Import service JSON processor module (REQUIRED)
if [[ -f "${SCRIPT_DIR}/lib/service-json-processor.sh" ]]; then
    source "${SCRIPT_DIR}/lib/service-json-processor.sh"
else
    echo "[ERROR] lib/service-json-processor.sh not found. This module is required." >&2
    exit 1
fi

# Import service JSON validator module (REQUIRED)
if [[ -f "${SCRIPT_DIR}/lib/service-json-validator.sh" ]]; then
    source "${SCRIPT_DIR}/lib/service-json-validator.sh"
else
    echo "[ERROR] lib/service-json-validator.sh not found. This module is required." >&2
    exit 1
fi

# Import file system operations module (REQUIRED)
if [[ -f "${SCRIPT_DIR}/lib/fs-operations.sh" ]]; then
    source "${SCRIPT_DIR}/lib/fs-operations.sh"
else
    echo "[ERROR] lib/fs-operations.sh not found. This module is required." >&2
    exit 1
fi

# Import CLI interface module (REQUIRED)
if [[ -f "${SCRIPT_DIR}/lib/cli-interface.sh" ]]; then
    source "${SCRIPT_DIR}/lib/cli-interface.sh"
else
    echo "[ERROR] lib/cli-interface.sh not found. This module is required." >&2
    exit 1
fi

# Import helper functions (override defaults if available)
if [[ -f "${PROJECT_ROOT}/scripts/helpers/utils/log.sh" ]]; then
    source "${PROJECT_ROOT}/scripts/helpers/utils/log.sh"
fi

if [[ -f "${PROJECT_ROOT}/scripts/helpers/utils/system.sh" ]]; then
    source "${PROJECT_ROOT}/scripts/helpers/utils/system.sh"
fi


# Global variables (will be set by cli-interface.sh parse_args function)
SCENARIO_PATH=""
SERVICE_JSON=""
SERVICE_JSON_PATH=""

# Configure validator for scenario-to-app usage
export VALIDATOR_VERBOSE="false"  # Will be updated in parse_args if --verbose is set
export VALIDATOR_STRICT="false"   # Use non-strict mode for scenario conversion

# Rollback tracking
declare -a ROLLBACK_ACTIONS=()

################################################################################
# Helper Functions
################################################################################

# Usage display and argument parsing functions are provided by cli-interface.sh module

# Validate scenario exists and has required files
validate_scenario() {
    # Initialize fs-operations module paths
    if ! resolve_project_paths; then
        log_error "Failed to resolve project paths"
        exit 1
    fi
    
    # Use fs-operations to resolve scenario path
    if ! SCENARIO_PATH=$(resolve_scenario_path "$SCENARIO_NAME"); then
        exit 1  # Error already logged by resolve_scenario_path
    fi
    
    # Use fs-operations to validate scenario structure
    if ! validate_scenario_structure "$SCENARIO_PATH"; then
        exit 1  # Error already logged by validate_scenario_structure
    fi
    
    # Set service.json path for global use
    SERVICE_JSON_PATH="${SCENARIO_PATH}/service.json"
    
    # Load service.json using fs-operations
    log_info "Loading service.json from $SERVICE_JSON_PATH"
    if ! SERVICE_JSON=$(load_service_json "$SERVICE_JSON_PATH"); then
        exit 1  # Error already logged by load_service_json
    fi
    log_info "Loaded service.json successfully"
}

# Extract resources from service.json based on condition
# NOTE: This function now uses the service-json-processor module
extract_resources_by_condition() {
    local condition="$1"
    local error_message="$2"
    
    local resources
    if ! resources=$(sjp_get_resources_by_condition "$SERVICE_JSON" "$condition" 2>/dev/null); then
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
    # NOTE: Using service-json-processor module for reliable extraction
    local scenario_name
    local scenario_version
    if ! scenario_name=$(sjp_get_service_info "$SERVICE_JSON" "name" 2>/dev/null); then
        log_error "Invalid service.json: missing service.name (see .vrooli/schemas/service.schema.json)"
        return 1
    fi
    
    scenario_version=$(sjp_get_service_info "$SERVICE_JSON" "version" 2>/dev/null || echo "unknown")
    
    [[ "$VERBOSE" == true ]] && log_info "Scenario: $scenario_name v$scenario_version"
    
    # Check for required initialization files using service-json-processor
    # NOTE: Extract all referenced files and validate they exist
    [[ "$VERBOSE" == true ]] && log_info "Checking initialization files..."
    
    local all_files
    if ! all_files=$(sjp_get_all_referenced_files "$SERVICE_JSON" 2>/dev/null); then
        log_warning "Failed to extract referenced files from service.json"
        return 0  # Non-fatal, continue validation
    fi
    
    if [[ -n "$all_files" ]]; then
        while IFS= read -r file; do
            [[ -z "$file" ]] && continue
            local full_path="${SCENARIO_PATH}/$file"
            if [[ -f "$full_path" ]]; then
                [[ "$VERBOSE" == true ]] && log_success "  ✓ Found: $file"
            else
                log_error "  ✗ Missing: $file"
                return 1
            fi
        done <<< "$all_files"
    else
        [[ "$VERBOSE" == true ]] && log_info "No initialization files referenced"
    fi
    
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
    
    log_step "Validating injection readiness (validation-only mode)..."
    
    # Use the new service-json-validator module for comprehensive validation
    # This replaces all the buggy JQ queries and duplicated validation logic
    if validate_deployment_readiness "$scenario_path" "$SERVICE_JSON"; then
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
    
    # Create deployment directory structure using fs-operations
    if ! create_app_directory_structure "$target_dir"; then
        log_error "Failed to create app directory structure"
        return 1
    fi
    
    # Extract required resources from service.json
    local required_resources
    required_resources=$(echo "$SERVICE_JSON" | jq -r '
        .resources | to_entries[] | 
        select(.value | type == "object") | 
        .value | to_entries[] | 
        select(.value.enabled == true) | 
        .key
    ' 2>/dev/null | sort -u | tr '\n' ' ')
    
    log_info "Required resources: ${required_resources:-none}"
    
    # Create minimal runtime package instead of copying everything
    create_minimal_runtime "$target_dir" "$required_resources"
    
    # Copy scenario files to target location
    log_info "Copying scenario files..."
    if [[ -d "$scenario_path/.vrooli" ]]; then
        cp -r "$scenario_path/.vrooli" "$target_dir/"
    fi
    if [[ -d "$scenario_path/initialization" ]]; then
        cp -r "$scenario_path/initialization" "$target_dir/"
    fi
    if [[ -d "$scenario_path/deployment" ]]; then
        cp -r "$scenario_path/deployment" "$target_dir/"
    fi
    
    # Fix schema reference to be relative
    if [[ -f "$target_dir/.vrooli/service.json" ]]; then
        local temp_file="$target_dir/.vrooli/service.json.tmp"
        jq '."$schema" = "./schemas/service.schema.json"' "$target_dir/.vrooli/service.json" > "$temp_file"
        mv "$temp_file" "$target_dir/.vrooli/service.json"
    fi
    
    # Copy service.json to config directory for compatibility
    mkdir -p "${target_dir}/config"
    if [[ -f "$target_dir/.vrooli/service.json" ]]; then
        cp "$target_dir/.vrooli/service.json" "${target_dir}/config/service.json"
    fi
    
    # Generate runtime injection manifest
    log_info "Generating runtime injection manifest..."
    generate_injection_manifest "$scenario_path" "${target_dir}/manifests/injection.json"
    
    # Generate secrets template if it doesn't exist
    if [[ ! -f "$target_dir/.vrooli/secrets.json" ]]; then
        log_info "Generating secrets template..."
        cat > "$target_dir/.vrooli/secrets.json" << 'SECRETS_TEMPLATE'
{
  "DB_PASSWORD": "changeme",
  "DB_USER": "postgres",
  "DB_NAME": "app",
  "MINIO_USER": "minioadmin",
  "MINIO_PASSWORD": "changeme",
  "N8N_DB_NAME": "n8n",
  "WINDMILL_DB_NAME": "windmill",
  "SEARXNG_SECRET": "ultrasecretkey",
  "_comment": "Update these values before running the application"
}
SECRETS_TEMPLATE
    fi
    
    # Create .env file for the app
    local app_name=$(echo "$SERVICE_JSON" | jq -r '.service.name // "app"')
    local app_version=$(echo "$SERVICE_JSON" | jq -r '.service.version // "1.0.0"')
    
    cat > "$target_dir/.env" << ENV_FILE
# Application Environment Variables
APP_NAME=$app_name
APP_VERSION=$app_version

# Database Configuration
DB_PASSWORD=changeme
DB_USER=postgres
DB_NAME=$app_name

# Resource-specific Configuration
MINIO_USER=minioadmin
MINIO_PASSWORD=changeme
N8N_DB_NAME=n8n
WINDMILL_DB_NAME=windmill
SEARXNG_SECRET=ultrasecretkey
ENV_FILE
    
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
    
    # Make scripts executable using fs-operations
    make_executable "${target_dir}/bin" "*"
    make_executable "${target_dir}/scripts" "*"
    
    # Generate README for the app
    local scenario_display_name=$(echo "$SERVICE_JSON" | jq -r '.service.displayName // .service.name')
    cat > "$target_dir/README.md" << README
# $scenario_display_name

This is a minimal standalone application generated from a Vrooli scenario.

## Quick Start

1. **Configure environment** (optional):
   Edit \`.env\` to set passwords and configuration

2. **Start the application**:
   \`\`\`bash
   ./bin/start.sh
   \`\`\`

3. **Stop the application**:
   \`\`\`bash
   ./bin/stop.sh
   \`\`\`

## Structure

- \`runtime/\` - Minimal Vrooli runtime (injection engine + utilities)
- \`.vrooli/\` - Application configuration
- \`initialization/\` - Initialization data (SQL, workflows, configs)
- \`docker-compose.yml\` - Resource containers
- \`bin/start.sh\` - Production startup script
- \`.env\` - Environment configuration

## Resources

This app uses the following resources:
${required_resources:-None}

## Generated From

- Scenario: $scenario_path
- Date: $(date)
- Method: Minimal runtime extraction

## Size Comparison

- This app: ~10-20MB
- Full Vrooli copy: ~1GB+

The minimal runtime contains only the essential components needed to run the scenario.
README
    
    log_success "✅ Scenario app packaged successfully"
    
    # Display size information
    local app_size
    app_size=$(du -sh "$target_dir" 2>/dev/null | cut -f1)
    log_info "📊 App size: ${app_size:-unknown} (minimal runtime)"
    
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
    local app_version=$(echo "$SERVICE_JSON" | jq -r '.service.version // "1.0.0"')
    
    cat > "$script_path" << 'EOF'
#!/usr/bin/env bash
# Auto-generated self-contained startup script
# Generated by scenario-to-app.sh (minimal runtime version)

set -euo pipefail

APP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
RUNTIME_DIR="$APP_DIR/runtime"

# Source utilities if available
if [[ -f "$RUNTIME_DIR/utils/log.sh" ]]; then
    source "$RUNTIME_DIR/utils/log.sh"
else
    # Fallback logging functions
    log_info() { echo "[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $*"; }
    log_error() { echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2; }
    log_success() { echo "[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $*"; }
fi

# Function to wait for service health
wait_for_health() {
    local service="$1"
    local port="$2"
    local max_attempts="${3:-30}"
    local attempt=0
    
    log_info "Waiting for $service on port $port..."
    while (( attempt < max_attempts )); do
        if nc -z localhost "$port" 2>/dev/null; then
            log_success "✅ $service is ready"
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    log_error "❌ $service failed to start after $((max_attempts * 2)) seconds"
    return 1
}

main() {
EOF
    
    # Add scenario-specific metadata
    cat >> "$script_path" << EOF
    # Load app configuration
    if [[ ! -f "\$APP_DIR/.vrooli/service.json" ]]; then
        log_error "Missing .vrooli/service.json"
        exit 1
    fi
    
    APP_NAME="${scenario_name}"
    APP_VERSION="${app_version}"
    
    log_info "🚀 Starting ${scenario_display_name} v\$APP_VERSION"
    
    # Load environment variables if .env exists
    if [[ -f "\$APP_DIR/.env" ]]; then
        log_info "Loading environment variables..."
        set -a
        source "\$APP_DIR/.env"
        set +a
    fi
    
    # Start resources using docker-compose
    if [[ -f "\$APP_DIR/docker-compose.yml" ]]; then
        log_info "Starting application resources..."
        cd "\$APP_DIR"
        docker-compose up -d
        
        # Wait for critical services based on what's in service.json
        REQUIRED_RESOURCES=\$(jq -r '
            .resources | to_entries[] | 
            select(.value | type == "object") | 
            .value | to_entries[] | 
            select(.value.enabled == true and .value.required == true) | 
            .key
        ' "\$APP_DIR/.vrooli/service.json" 2>/dev/null | tr '\n' ' ')
        
        for resource in \$REQUIRED_RESOURCES; do
            case "\$resource" in
                postgres|postgresql)
                    wait_for_health "PostgreSQL" 5432 || exit 1
                    ;;
                redis)
                    wait_for_health "Redis" 6379 || exit 1
                    ;;
                n8n)
                    wait_for_health "n8n" 5678 60 || exit 1
                    ;;
                windmill)
                    wait_for_health "Windmill" 8000 60 || exit 1
                    ;;
                ollama)
                    wait_for_health "Ollama" 11434 || exit 1
                    ;;
                whisper)
                    wait_for_health "Whisper" 9000 || exit 1
                    ;;
                minio)
                    wait_for_health "MinIO" 9000 || exit 1
                    ;;
                qdrant)
                    wait_for_health "Qdrant" 6333 || exit 1
                    ;;
                vault)
                    wait_for_health "Vault" 8200 || exit 1
                    ;;
                searxng)
                    wait_for_health "SearXNG" 8080 || exit 1
                    ;;
                comfyui)
                    wait_for_health "ComfyUI" 8188 || exit 1
                    ;;
                judge0)
                    wait_for_health "Judge0" 2358 || exit 1
                    ;;
            esac
        done
    fi
    
    # Run initialization using injection engine if available
    if [[ -f "\$RUNTIME_DIR/injection/engine.sh" ]]; then
        log_info "Running initialization..."
        
        # Set required environment variables for injection engine
        export PROJECT_ROOT="\$APP_DIR"
        export RESOURCES_DIR="\$RUNTIME_DIR/resources"
        
        # Check if already initialized
        INIT_MARKER="\$APP_DIR/.initialized"
        if [[ -f "\$INIT_MARKER" ]]; then
            LAST_VERSION=\$(cat "\$INIT_MARKER" 2>/dev/null || echo "")
            if [[ "\$LAST_VERSION" == "\$APP_VERSION" ]]; then
                log_info "Already initialized for version \$APP_VERSION, skipping..."
            else
                log_info "Version changed from \$LAST_VERSION to \$APP_VERSION, re-initializing..."
                rm -f "\$INIT_MARKER"
            fi
        fi
        
        if [[ ! -f "\$INIT_MARKER" ]]; then
            if "\$RUNTIME_DIR/injection/engine.sh" \\
                --action inject \\
                --scenario-dir "\$APP_DIR" \\
                --dry-run no; then
                echo "\$APP_VERSION" > "\$INIT_MARKER"
                log_success "✅ Initialization completed"
            else
                log_error "❌ Initialization failed"
                docker-compose down
                exit 1
            fi
        fi
    fi
    
    # Run custom startup script if provided
    if [[ -f "\$APP_DIR/deployment/startup.sh" ]]; then
        log_info "Running custom startup script..."
        exec "\$APP_DIR/deployment/startup.sh"
    else
        log_success "✅ ${scenario_display_name} is ready!"
        log_info ""
        log_info "Resources are running. Access points:"
        
        # Display access URLs based on running services
        cd "\$APP_DIR"
        [[ -n "\$(docker-compose ps -q postgres 2>/dev/null)" ]] && log_info "  PostgreSQL: localhost:5432"
        [[ -n "\$(docker-compose ps -q redis 2>/dev/null)" ]] && log_info "  Redis: localhost:6379"
        [[ -n "\$(docker-compose ps -q n8n 2>/dev/null)" ]] && log_info "  n8n: http://localhost:5678"
        [[ -n "\$(docker-compose ps -q windmill 2>/dev/null)" ]] && log_info "  Windmill: http://localhost:8000"
        [[ -n "\$(docker-compose ps -q ollama 2>/dev/null)" ]] && log_info "  Ollama: http://localhost:11434"
        [[ -n "\$(docker-compose ps -q minio 2>/dev/null)" ]] && log_info "  MinIO: http://localhost:9001"
        [[ -n "\$(docker-compose ps -q whisper 2>/dev/null)" ]] && log_info "  Whisper: http://localhost:9000"
        [[ -n "\$(docker-compose ps -q qdrant 2>/dev/null)" ]] && log_info "  Qdrant: http://localhost:6333"
        [[ -n "\$(docker-compose ps -q vault 2>/dev/null)" ]] && log_info "  Vault: http://localhost:8200"
        [[ -n "\$(docker-compose ps -q searxng 2>/dev/null)" ]] && log_info "  SearXNG: http://localhost:8080"
        [[ -n "\$(docker-compose ps -q comfyui 2>/dev/null)" ]] && log_info "  ComfyUI: http://localhost:8188"
        [[ -n "\$(docker-compose ps -q judge0 2>/dev/null)" ]] && log_info "  Judge0: http://localhost:2358"
        
        log_info ""
        log_info "Press Ctrl+C to stop all services"
        
        # Handle shutdown gracefully
        shutdown() {
            log_info "Shutting down services..."
            docker-compose down
            exit 0
        }
        
        trap shutdown SIGINT SIGTERM
        
        # Keep running
        while true; do
            sleep 1
        done
    fi
}

main "\$@"
EOF
    
    chmod +x "$script_path"
    
    # Also create a stop script for convenience
    local stop_script="${target_dir}/bin/stop.sh"
    cat > "$stop_script" << 'STOP_SCRIPT'
#!/usr/bin/env bash
set -euo pipefail

APP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$APP_DIR"

echo "Stopping all services..."
docker-compose down
echo "✅ All services stopped"
STOP_SCRIPT
    
    chmod +x "$stop_script"
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
    
    # Generate proper docker-compose.yml using the new function
    generate_docker_compose "$target_dir"
}

#######################################
# Create minimal runtime package (ported from minimal version)
# Arguments:
#   $1 - output directory
#   $2 - required resources (space-separated)
#######################################
create_minimal_runtime() {
    local output_dir="$1"
    local required_resources="$2"
    
    log_info "Creating minimal runtime package..."
    
    # Create runtime directory structure
    mkdir -p "$output_dir/runtime/injection"
    mkdir -p "$output_dir/runtime/utils"
    mkdir -p "$output_dir/runtime/resources"
    mkdir -p "$output_dir/runtime/startup"
    
    # Copy injection engine
    log_debug "Copying injection engine..."
    if [[ -d "$PROJECT_ROOT/scripts/scenarios/injection" ]]; then
        cp -r "$PROJECT_ROOT/scripts/scenarios/injection/"* "$output_dir/runtime/injection/" 2>/dev/null || true
    fi
    
    # Copy essential utilities
    log_debug "Copying essential utilities..."
    for util in log.sh flow.sh ports.sh system.sh docker.sh repository.sh args.sh; do
        if [[ -f "$PROJECT_ROOT/scripts/helpers/utils/$util" ]]; then
            cp "$PROJECT_ROOT/scripts/helpers/utils/$util" "$output_dir/runtime/utils/"
        fi
    done
    
    # Copy resource common utilities
    if [[ -f "$PROJECT_ROOT/scripts/resources/common.sh" ]]; then
        cp "$PROJECT_ROOT/scripts/resources/common.sh" "$output_dir/runtime/resources/"
    fi
    if [[ -f "$PROJECT_ROOT/scripts/resources/port-registry.sh" ]]; then
        cp "$PROJECT_ROOT/scripts/resources/port-registry.sh" "$output_dir/runtime/resources/"
    fi
    
    # Copy only required resource adapters
    if [[ -n "$required_resources" ]]; then
        log_debug "Copying required resource adapters: $required_resources"
        for resource in $required_resources; do
            local resource_dir=""
            # Handle different resource names and locations
            case "$resource" in
                postgres|postgresql)
                    resource_dir="$PROJECT_ROOT/scripts/resources/storage/postgres"
                    ;;
                n8n)
                    resource_dir="$PROJECT_ROOT/scripts/resources/automation/n8n"
                    ;;
                windmill)
                    resource_dir="$PROJECT_ROOT/scripts/resources/automation/windmill"
                    ;;
                ollama)
                    resource_dir="$PROJECT_ROOT/scripts/resources/ai/ollama"
                    ;;
                whisper)
                    resource_dir="$PROJECT_ROOT/scripts/resources/ai/whisper"
                    ;;
                redis)
                    resource_dir="$PROJECT_ROOT/scripts/resources/storage/redis"
                    ;;
                minio)
                    resource_dir="$PROJECT_ROOT/scripts/resources/storage/minio"
                    ;;
                qdrant)
                    resource_dir="$PROJECT_ROOT/scripts/resources/storage/qdrant"
                    ;;
                comfyui)
                    resource_dir="$PROJECT_ROOT/scripts/resources/automation/comfyui"
                    ;;
                judge0)
                    resource_dir="$PROJECT_ROOT/scripts/resources/execution/judge0"
                    ;;
                vault)
                    resource_dir="$PROJECT_ROOT/scripts/resources/storage/vault"
                    ;;
                searxng)
                    resource_dir="$PROJECT_ROOT/scripts/resources/search/searxng"
                    ;;
                *)
                    # Try to find it with find command
                    resource_dir=$(find "$PROJECT_ROOT/scripts/resources" -type d -name "$resource" 2>&1 | grep -v "Permission denied" | head -1)
                    ;;
            esac
            
            if [[ -n "$resource_dir" && -d "$resource_dir" ]]; then
                local parent_dir=$(basename "$(dirname "$resource_dir")")
                mkdir -p "$output_dir/runtime/resources/$parent_dir"
                # Copy only essential files, exclude data directories
                rsync -av --exclude="instances/" --exclude="data/" --exclude="*.log" \
                    "$resource_dir/" "$output_dir/runtime/resources/$parent_dir/$resource/" 2>/dev/null || \
                    cp -r "$resource_dir" "$output_dir/runtime/resources/$parent_dir/" 2>/dev/null || true
                log_debug "  ✓ Copied resource adapter: $resource"
            else
                log_warn "  ⚠ Resource adapter not found: $resource (will use Docker only)"
            fi
        done
    fi
    
    log_success "✅ Minimal runtime package created"
}

#######################################
# Generate production-ready docker-compose.yml
# Arguments:
#   $1 - output directory
#######################################
generate_docker_compose() {
    local output_dir="$1"
    
    log_info "Generating docker-compose.yml for required resources..."
    
    # Extract required resources from SERVICE_JSON
    local required_resources
    required_resources=$(echo "$SERVICE_JSON" | jq -r '
        .resources | to_entries[] | 
        select(.value | type == "object") | 
        .value | to_entries[] | 
        select(.value.enabled == true) | 
        .key
    ' 2>/dev/null | sort -u | tr '\n' ' ')
    
    # Start with basic compose structure
    cat > "$output_dir/docker-compose.yml" << 'COMPOSE_HEADER'
version: '3.8'

networks:
  app-network:
    driver: bridge

services:
COMPOSE_HEADER
    
    # Track which volumes we need
    local volumes_needed=()
    
    # Add each required resource to compose file
    for resource in $required_resources; do
        case "$resource" in
            postgres|postgresql)
                cat >> "$output_dir/docker-compose.yml" << 'POSTGRES_SERVICE'
  postgres:
    image: postgres:16-alpine
    container_name: app-postgres
    environment:
      POSTGRES_PASSWORD: ${DB_PASSWORD:-postgres}
      POSTGRES_USER: ${DB_USER:-postgres}
      POSTGRES_DB: ${DB_NAME:-app}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./initialization/database:/docker-entrypoint-initdb.d:ro
    networks:
      - app-network
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

POSTGRES_SERVICE
                volumes_needed+=("postgres_data")
                ;;
                
            redis)
                cat >> "$output_dir/docker-compose.yml" << 'REDIS_SERVICE'
  redis:
    image: redis:7-alpine
    container_name: app-redis
    networks:
      - app-network
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
    restart: unless-stopped

REDIS_SERVICE
                ;;
                
            n8n)
                cat >> "$output_dir/docker-compose.yml" << 'N8N_SERVICE'
  n8n:
    image: n8nio/n8n:latest
    container_name: app-n8n
    environment:
      - N8N_BASIC_AUTH_ACTIVE=false
      - N8N_PORT=5678
      - DB_TYPE=postgresdb
      - DB_POSTGRESDB_HOST=postgres
      - DB_POSTGRESDB_PORT=5432
      - DB_POSTGRESDB_DATABASE=${N8N_DB_NAME:-n8n}
      - DB_POSTGRESDB_USER=${DB_USER:-postgres}
      - DB_POSTGRESDB_PASSWORD=${DB_PASSWORD:-postgres}
    networks:
      - app-network
    ports:
      - "5678:5678"
    volumes:
      - n8n_data:/home/node/.n8n
      - ./initialization/workflows/n8n:/workflows:ro
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

N8N_SERVICE
                volumes_needed+=("n8n_data")
                ;;
                
            windmill)
                cat >> "$output_dir/docker-compose.yml" << 'WINDMILL_SERVICE'
  windmill:
    image: ghcr.io/windmill-labs/windmill:latest
    container_name: app-windmill
    networks:
      - app-network
    ports:
      - "8000:8000"
    environment:
      - DATABASE_URL=postgres://${DB_USER:-postgres}:${DB_PASSWORD:-postgres}@postgres/${WINDMILL_DB_NAME:-windmill}
      - INIT_SCRIPT=/initialization/windmill/init.sh
    volumes:
      - ./initialization/ui:/initialization/windmill:ro
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

WINDMILL_SERVICE
                ;;
                
            ollama)
                cat >> "$output_dir/docker-compose.yml" << 'OLLAMA_SERVICE'
  ollama:
    image: ollama/ollama:latest
    container_name: app-ollama
    networks:
      - app-network
    ports:
      - "11434:11434"
    volumes:
      - ollama_models:/root/.ollama
    environment:
      - OLLAMA_HOST=0.0.0.0
    restart: unless-stopped

OLLAMA_SERVICE
                volumes_needed+=("ollama_models")
                ;;
                
            whisper)
                cat >> "$output_dir/docker-compose.yml" << 'WHISPER_SERVICE'
  whisper:
    image: onerahmet/openai-whisper-asr-webservice:latest
    container_name: app-whisper
    networks:
      - app-network
    ports:
      - "9000:9000"
    environment:
      - ASR_MODEL=base
    restart: unless-stopped

WHISPER_SERVICE
                ;;
                
            minio)
                cat >> "$output_dir/docker-compose.yml" << 'MINIO_SERVICE'
  minio:
    image: minio/minio:latest
    container_name: app-minio
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=${MINIO_USER:-minioadmin}
      - MINIO_ROOT_PASSWORD=${MINIO_PASSWORD:-minioadmin}
    networks:
      - app-network
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3
    restart: unless-stopped

MINIO_SERVICE
                volumes_needed+=("minio_data")
                ;;
                
            qdrant)
                cat >> "$output_dir/docker-compose.yml" << 'QDRANT_SERVICE'
  qdrant:
    image: qdrant/qdrant:latest
    container_name: app-qdrant
    networks:
      - app-network
    ports:
      - "6333:6333"
    volumes:
      - qdrant_data:/qdrant/storage
    restart: unless-stopped

QDRANT_SERVICE
                volumes_needed+=("qdrant_data")
                ;;
                
            comfyui)
                cat >> "$output_dir/docker-compose.yml" << 'COMFYUI_SERVICE'
  comfyui:
    image: ghcr.io/comfyanonymous/comfyui:latest
    container_name: app-comfyui
    networks:
      - app-network
    ports:
      - "8188:8188"
    volumes:
      - comfyui_data:/workspace
    environment:
      - COMMANDLINE_ARGS=--listen 0.0.0.0
    restart: unless-stopped

COMFYUI_SERVICE
                volumes_needed+=("comfyui_data")
                ;;
                
            judge0)
                cat >> "$output_dir/docker-compose.yml" << 'JUDGE0_SERVICE'
  judge0:
    image: judge0/judge0:latest
    container_name: app-judge0
    networks:
      - app-network
    ports:
      - "2358:2358"
    environment:
      - RAILS_ENV=production
      - REDIS_HOST=redis
    depends_on:
      - redis
    restart: unless-stopped

JUDGE0_SERVICE
                ;;
                
            vault)
                cat >> "$output_dir/docker-compose.yml" << 'VAULT_SERVICE'
  vault:
    image: hashicorp/vault:latest
    container_name: app-vault
    networks:
      - app-network
    ports:
      - "8200:8200"
    environment:
      - VAULT_DEV_ROOT_TOKEN_ID=root
      - VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200
    cap_add:
      - IPC_LOCK
    restart: unless-stopped

VAULT_SERVICE
                ;;
                
            searxng)
                cat >> "$output_dir/docker-compose.yml" << 'SEARXNG_SERVICE'
  searxng:
    image: searxng/searxng:latest
    container_name: app-searxng
    networks:
      - app-network
    ports:
      - "8080:8080"
    volumes:
      - searxng_data:/etc/searxng
    environment:
      - SEARXNG_SECRET_KEY=${SEARXNG_SECRET:-ultrasecretkey}
    restart: unless-stopped

SEARXNG_SERVICE
                volumes_needed+=("searxng_data")
                ;;
                
            *)
                log_warn "No docker-compose template for resource: $resource"
                ;;
        esac
    done
    
    # Add volumes section if any services were added
    if [[ ${#volumes_needed[@]} -gt 0 ]]; then
        echo "" >> "$output_dir/docker-compose.yml"
        echo "volumes:" >> "$output_dir/docker-compose.yml"
        for volume in "${volumes_needed[@]}"; do
            echo "  $volume:" >> "$output_dir/docker-compose.yml"
        done
    fi
    
    log_success "✅ Docker compose file generated"
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

# Note: create_safe_backup function now provided by fs-operations.sh module

# Pre-flight safety checks before any modifications
preflight_safety_check() {
    local service_config="$1"
    
    # Use fs-operations run_preflight_checks function
    local config_dir
    config_dir=$(dirname "$service_config")
    
    if ! run_preflight_checks "$config_dir"; then
        return 1
    fi
    
    # Additional validation for existing config if it exists
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
    if ! parse_args "$@"; then
        exit $?
    fi
    
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
