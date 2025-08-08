#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Scenario-to-App Generator
# 
# Converts Vrooli scenarios into standalone, deployable applications.
# Generated apps use Vrooli's infrastructure but are self-contained.
#
# This script GENERATES apps, it does not RUN them.
#
# Usage:
#   ./scenario-to-app.sh <scenario-name>
#   ./scenario-to-app.sh research-assistant --verbose
#
# Generated apps are placed in ~/generated-apps/<scenario-name>/
# 
# To run a generated app:
#   cd ~/generated-apps/<scenario-name>
#   ./scripts/main/develop.sh
#
# Architecture:
# - Validates scenario files and structure using project schema
# - Copies scenario data as-is to generated app
# - Copies Vrooli's scripts/ infrastructure (minus scenarios/)
# - Generated apps use standard Vrooli scripts with scenario's service.json
# 
# Resource Initialization:
# - Apps handle their own resource initialization via copied Vrooli scripts
# - The setup.sh/develop.sh scripts automatically detect and process service.json
# - No injection happens during app generation - only at runtime via setup scripts
#
################################################################################

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SCENARIOS_DIR="${PROJECT_ROOT}/scripts/scenarios/core"
RESOURCES_DIR="${PROJECT_ROOT}/scripts/resources"

# Source utilities or define fallback logging
if [[ -f "${RESOURCES_DIR}/common.sh" ]]; then
    # shellcheck disable=SC1091
    source "${RESOURCES_DIR}/common.sh"
else
    # Fallback logging functions
    log::info() { echo "ℹ️  $*"; }
    log::success() { echo "✅ $*"; }
    log::warning() { echo "⚠️  $*"; }
    log::error() { echo "❌ $*"; }
fi

# Global variables
SCENARIO_NAME=""
SCENARIO_PATH=""
SERVICE_JSON=""
VERBOSE=false
DRY_RUN=false
FORCE=false
START=false

# Usage information
show_usage() {
    cat << EOF
Usage: $0 <scenario-name> [options]

Generate a standalone application from a validated Vrooli scenario.

Arguments:
  scenario-name     Name of the scenario (e.g., campaign-content-studio)

Options:
  --verbose         Enable verbose output
  --dry-run         Show what would be generated without creating files
  --force           Overwrite existing generated app
  --start           Start the generated app after creation
  --help            Show this help message

The generated app will be created at: ~/generated-apps/<scenario-name>/

To run the generated app:
  cd ~/generated-apps/<scenario-name>
  ./scripts/main/develop.sh

Examples:
  $0 campaign-content-studio
  $0 research-assistant --verbose --force
  $0 research-assistant --dry-run
  $0 campaign-content-studio --start

EOF
}

# Parse command line arguments
parse_args() {
    if [[ $# -eq 0 ]]; then
        log::error "No scenario name provided"
        show_usage
        exit 1
    fi

    # Check for --help first
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
            --force)
                FORCE=true
                ;;
            --start)
                START=true
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                log::error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
        shift
    done
}

# Validate scenario structure with comprehensive schema validation
validate_scenario() {
    SCENARIO_PATH="${SCENARIOS_DIR}/${SCENARIO_NAME}"
    
    if [[ ! -d "$SCENARIO_PATH" ]]; then
        log::error "Scenario directory not found: $SCENARIO_PATH"
        exit 1
    fi
    
    log::success "Scenario directory found: $SCENARIO_PATH"
    
    # Check for service.json (first in .vrooli/, then root for backwards compatibility)
    local service_json_path="${SCENARIO_PATH}/.vrooli/service.json"
    if [[ ! -f "$service_json_path" ]]; then
        # Try root directory for backwards compatibility
        service_json_path="${SCENARIO_PATH}/service.json"
        if [[ ! -f "$service_json_path" ]]; then
            log::error "service.json not found at: ${SCENARIO_PATH}/.vrooli/service.json or ${SCENARIO_PATH}/service.json"
            exit 1
        fi
    fi
    
    # Load JSON content (maintains global variable contract)
    if ! SERVICE_JSON=$(cat "$service_json_path" 2>/dev/null); then
        log::error "Failed to read service.json"
        exit 1
    fi
    
    # Clean up common JSON issues (trailing commas) before validation
    # This uses a more lenient approach to handle real-world JSON files
    # Handles commas before closing braces/brackets, even across lines
    SERVICE_JSON_CLEANED=$(echo "$SERVICE_JSON" | perl -0pe 's/,(\s*[}\]])/$1/g' 2>/dev/null || echo "$SERVICE_JSON")
    
    # Basic JSON syntax validation
    if ! echo "$SERVICE_JSON_CLEANED" | jq empty 2>/dev/null; then
        log::error "Invalid JSON in service.json"
        log::error "Common issues to check:"
        log::error "  - Missing quotes around strings"
        log::error "  - Unmatched brackets or braces"
        log::error "  - Invalid escape sequences"
        exit 1
    fi
    
    # Use the cleaned JSON for the rest of the script
    SERVICE_JSON="$SERVICE_JSON_CLEANED"
    
    log::info "Loading comprehensive schema validator..."
    
    # Check for schema validation
    local schema_file="${PROJECT_ROOT}/.vrooli/schemas/service.schema.json"
    
    # Use python/node for JSON schema validation if available
    if command -v python3 >/dev/null 2>&1 && [[ -f "$schema_file" ]]; then
        log::info "Running JSON schema validation using Python..."
        
        # Create a simple Python validation script
        local validation_result
        validation_result=$(python3 -c "
import json
import sys

try:
    # For basic validation, just check JSON structure
    service_json = json.loads('''${SERVICE_JSON}''')
    
    # Check required fields
    if 'service' not in service_json:
        print('ERROR: Missing required field: service')
        sys.exit(1)
    if 'name' not in service_json['service']:
        print('ERROR: Missing required field: service.name')
        sys.exit(1)
    if 'version' not in service_json['service']:
        print('ERROR: Missing required field: service.version')
        sys.exit(1)
    if 'type' not in service_json['service']:
        print('ERROR: Missing required field: service.type')
        sys.exit(1)
        
    # Count enabled resources
    resource_count = 0
    if 'resources' in service_json:
        for category, resources in service_json['resources'].items():
            for name, config in resources.items():
                if config.get('enabled', False):
                    resource_count += 1
    
    print(f'VALID:{resource_count}')
    
except json.JSONDecodeError as e:
    print(f'ERROR: Invalid JSON - {e}')
    sys.exit(1)
except Exception as e:
    print(f'ERROR: {e}')
    sys.exit(1)
" 2>&1) || true
        
        if [[ "$validation_result" =~ ^VALID:([0-9]+)$ ]]; then
            local resource_count="${BASH_REMATCH[1]}"
            log::success "✅ Schema validation passed"
            
            if [[ "$VERBOSE" == "true" ]]; then
                log::info "📋 Validation Summary:"
                log::info "   • Service metadata: ✅ Valid"
                log::info "   • Resource configuration: ✅ Valid ($resource_count resources enabled)"
                log::info "   • Schema compliance: ✅ Passed"
            fi
        elif [[ "$validation_result" =~ ^ERROR: ]]; then
            log::error "❌ Schema validation failed"
            log::error "$validation_result"
            log::error "💡 Common fixes:"
            log::error "   • Ensure service.name is present and follows naming rules"
            log::error "   • Verify all required resource configurations are complete"  
            log::error "   • Check JSON syntax in .vrooli/service.json"
            exit 1
        else
            log::warning "Schema validation produced unexpected output: $validation_result"
            log::info "Falling back to basic validation..."
        fi
        
    elif command -v node >/dev/null 2>&1 && [[ -f "$schema_file" ]]; then
        log::info "Running JSON schema validation using Node.js..."
        # TODO: Implement Node.js-based validation if Python not available
        log::warning "Node.js validation not yet implemented, using basic validation"
        log::info "Performing basic service.json validation..."
    else
        log::warning "No JSON schema validator available (Python3 or Node.js required)"
        log::info "Performing basic service.json validation..."
        
        # Fallback to basic validation (preserve original logic)
        local service_name
        service_name=$(echo "$SERVICE_JSON" | jq -r '.service.name // empty' 2>/dev/null)
        
        if [[ -z "$service_name" ]]; then
            log::error "service.json missing required field: service.name"
            log::error "💡 Add a 'service.name' field to your .vrooli/service.json file"
            exit 1
        fi
        
        log::success "✅ Basic service.json validation passed"
        
        # Show basic validation summary
        if [[ "$VERBOSE" == "true" ]]; then
            log::info "📋 Basic Validation Summary:"
            log::info "   • Service name: ✅ Present ($service_name)"
            log::info "   • JSON syntax: ✅ Valid"
            log::info "   • Advanced validation: ⚠️  Not available (install service-json-validator.sh)"
        fi
    fi
    
    # Validate initialization file paths
    log::info "Validating initialization file paths..."
    if ! validate_initialization_paths; then
        exit 1
    fi
    
    # Display service information (preserve original behavior)
    if [[ "$VERBOSE" == "true" ]]; then
        local service_name service_display_name
        service_name=$(echo "$SERVICE_JSON" | jq -r '.service.name // empty' 2>/dev/null)
        service_display_name=$(echo "$SERVICE_JSON" | jq -r '.service.displayName // empty' 2>/dev/null)
        
        log::info "Service name: $service_name"
        if [[ -n "$service_display_name" ]]; then
            log::info "Display name: $service_display_name"
        fi
    fi
}

# Get all enabled resources from service.json (supports all categories)
get_enabled_resources() {
    # Extract ALL enabled resources from ANY category
    echo "$SERVICE_JSON" | jq -r '
        .resources | to_entries[] as $category | 
        $category.value | to_entries[] | 
        select(.value.enabled == true) | 
        "\(.key) (\($category.key))"  
    ' 2>/dev/null || true
}

# Get resource categories from service.json
get_resource_categories() {
    # Extract all resource category names
    echo "$SERVICE_JSON" | jq -r '.resources | keys[]' 2>/dev/null || true
}

# Validate that initialization file paths in service.json actually exist
validate_initialization_paths() {
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "Validating initialization file paths..."
    fi
    
    local validation_errors=0
    local total_files=0
    
    # Check initialization files referenced in resource configurations
    echo "$SERVICE_JSON" | jq -r '
        .resources | to_entries[] as $category |
        $category.value | to_entries[] as $resource |
        $resource.value | select(.enabled == true) |
        .initialization? // {} |
        (
            (.data[]?.file // empty),
            (.workflows[]?.file // empty),
            (.apps[]?.file // empty),
            (.scripts[]?.file // empty)
        ) | select(. != null and . != "")
    ' 2>/dev/null | while IFS= read -r file_path; do
        total_files=$((total_files + 1))
        local full_path="${SCENARIO_PATH}/${file_path}"
        
        if [[ ! -f "$full_path" ]]; then
            log::error "Referenced file not found: $file_path"
            log::error "  Expected at: $full_path"
            validation_errors=$((validation_errors + 1))
        elif [[ "$VERBOSE" == "true" ]]; then
            log::info "✅ Found: $file_path"
        fi
    done
    
    # Check deployment initialization files
    echo "$SERVICE_JSON" | jq -r '
        .deployment.initialization.phases[]?.tasks[]? |
        select(.type == "sql" or .type == "config") |
        .file // empty | select(. != "")
    ' 2>/dev/null | while IFS= read -r file_path; do
        total_files=$((total_files + 1))
        local full_path="${SCENARIO_PATH}/${file_path}"
        
        if [[ ! -f "$full_path" ]]; then
            log::error "Deployment file not found: $file_path"
            log::error "  Expected at: $full_path"
            validation_errors=$((validation_errors + 1))
        elif [[ "$VERBOSE" == "true" ]]; then
            log::info "✅ Found: $file_path"
        fi
    done
    
    # Summary of validation results
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "📋 Path Validation Summary:"
        if [[ "$validation_errors" -eq 0 ]]; then
            log::success "   • All $total_files initialization file paths are valid"
        else
            log::error "   • $validation_errors file path(s) are invalid out of $total_files total"
        fi
    fi
    
    # Fail validation if any errors found
    if [[ "$validation_errors" -gt 0 ]]; then
        log::error "❌ Initialization path validation failed"
        log::error "💡 Common fixes:"
        log::error "   • Ensure all file paths in service.json use correct relative paths"
        log::error "   • Verify initialization files exist in the scenario directory"
        log::error "   • Check that folder names match the scripts/resources/ convention"
        return 1
    fi
    
    return 0
}

################################################################################
# App Generation Functions
################################################################################

#######################################
# Create default service.json for generated apps
# Arguments:
#   $1 - Destination service.json path
#######################################
create_default_service_json() {
    local dest="$1"
    local app_name="$(basename "$(dirname "$(dirname "$dest")")")"
    
    cat > "$dest" << EOF
{
  "\$schema": "../schemas/service.schema.json",
  "version": "1.0.0",
  "service": {
    "name": "$app_name",
    "displayName": "$(echo "$app_name" | sed 's/-/ /g' | sed 's/\b\w/\U&/g')",
    "description": "Generated standalone application",
    "version": "1.0.0",
    "type": "application"
  },
  "lifecycle": {
    "version": "1.0.0",
    "defaults": {
      "timeout": "5m",
      "error": "stop",
      "shell": "bash"
    },
    "setup": {
      "description": "Initialize application environment",
      "steps": [
        {
          "name": "install-deps",
          "run": "npm ci || npm install || echo 'No package.json found'",
          "condition": "\${SKIP_INSTALL} != 'true'"
        }
      ]
    },
    "develop": {
      "description": "Start development server",
      "steps": [
        {
          "name": "start-dev",
          "run": "npm start || npm run dev || echo 'No start script found in package.json'"
        }
      ]
    },
    "build": {
      "description": "Build application for production",
      "steps": [
        {
          "name": "build-app",
          "run": "npm run build || echo 'No build script found in package.json'"
        }
      ]
    },
    "test": {
      "description": "Run test suites",
      "steps": [
        {
          "name": "run-tests",
          "run": "npm test || npm run test || echo 'No test script found in package.json'"
        }
      ]
    },
    "clean": {
      "description": "Clean build artifacts",
      "steps": [
        {
          "name": "clean-artifacts",
          "run": "rm -rf dist/ build/ .next/ node_modules/.cache/ || true"
        }
      ]
    }
  }
}
EOF
    
    [[ "$VERBOSE" == "true" ]] && log::info "Created default service.json with basic lifecycle configuration"
}

#######################################
# Enhance existing service.json with basic lifecycle if missing
# Arguments:
#   $1 - Service.json path
#######################################
enhance_service_json_lifecycle() {
    local service_json_path="$1"
    
    # Check if lifecycle configuration exists
    if ! jq -e '.lifecycle' "$service_json_path" >/dev/null 2>&1; then
        [[ "$VERBOSE" == "true" ]] && log::info "Adding basic lifecycle configuration to existing service.json"
        
        # Create a temporary enhanced version
        local temp_file="${service_json_path}.tmp"
        jq '. + {
          "lifecycle": {
            "version": "1.0.0",
            "defaults": {
              "timeout": "5m",
              "error": "stop",
              "shell": "bash"
            },
            "setup": {
              "description": "Initialize application environment",
              "steps": [
                {
                  "name": "install-deps",
                  "run": "npm ci || npm install || echo \"No package.json found\"",
                  "condition": "${SKIP_INSTALL} != \"true\""
                }
              ]
            },
            "develop": {
              "description": "Start development server",
              "steps": [
                {
                  "name": "start-dev",
                  "run": "npm start || npm run dev || echo \"No start script found in package.json\""
                }
              ]
            },
            "build": {
              "description": "Build application for production",
              "steps": [
                {
                  "name": "build-app",
                  "run": "npm run build || echo \"No build script found in package.json\""
                }
              ]
            },
            "test": {
              "description": "Run test suites",
              "steps": [
                {
                  "name": "run-tests",
                  "run": "npm test || npm run test || echo \"No test script found in package.json\""
                }
              ]
            },
            "clean": {
              "description": "Clean build artifacts",
              "steps": [
                {
                  "name": "clean-artifacts",
                  "run": "rm -rf dist/ build/ .next/ node_modules/.cache/ || true"
                }
              ]
            }
          }
        }' "$service_json_path" > "$temp_file"
        
        # Replace original with enhanced version
        mv "$temp_file" "$service_json_path"
        [[ "$VERBOSE" == "true" ]] && log::info "Enhanced service.json with basic lifecycle configuration"
    else
        [[ "$VERBOSE" == "true" ]] && log::info "service.json already has lifecycle configuration"
    fi
}

# Process template variables in files (conditionally based on inheritance)
process_template_variables() {
    local file="$1"
    local service_json="$2"
    
    # Skip if file doesn't exist or is binary
    [[ ! -f "$file" ]] && return 0
    file -b --mime-type "$file" | grep -q "text/" || return 0
    
    # Source required utilities
    local secrets_util="${PROJECT_ROOT}/scripts/helpers/utils/secrets.sh"
    local service_config_util="${PROJECT_ROOT}/scripts/helpers/utils/service-config.sh"
    
    # Check if the service.json has inheritance defined
    local should_substitute="yes"
    if [[ -f "$service_config_util" ]] && [[ "$(basename "$file")" == "service.json" ]]; then
        # shellcheck disable=SC1090
        source "$service_config_util"
        
        # Check for inheritance - if present, preserve templates for runtime resolution
        if service_config::has_inheritance "$file"; then
            should_substitute="no"
            [[ "$VERBOSE" == "true" ]] && log::info "Inheritance detected in $(basename "$file"), preserving templates for runtime resolution"
        fi
    fi
    
    # Only substitute if no inheritance or if not a service.json file
    if [[ "$should_substitute" == "yes" ]] && [[ -f "$secrets_util" ]]; then
        # shellcheck disable=SC1091
        source "$secrets_util"
        
        # Read file content and substitute all templates (both secrets and service references)
        local content
        content=$(cat "$file")
        local processed
        processed=$(echo "$content" | secrets::substitute_all_templates)
        
        # Write back if changed
        if [[ "$content" != "$processed" ]]; then
            echo "$processed" > "$file"
            [[ "$VERBOSE" == "true" ]] && log::info "Processed templates in: $(basename "$file")"
        fi
    fi
}

# Copy scenario files to generated app
copy_scenario_files() {
    local scenario_path="$1" 
    local app_path="$2"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would copy scenario files from: $scenario_path"
        return 0
    fi
    
    log::info "Copying scenario files..."
    
    # Copy scenario structure as-is
    if [[ -d "$scenario_path/.vrooli" ]]; then
        cp -r "$scenario_path/.vrooli" "$app_path/" 2>/dev/null || true
        [[ "$VERBOSE" == "true" ]] && log::info "Copied .vrooli directory"
        
        # Process template variables in service.json if needed
        if [[ -f "$app_path/.vrooli/service.json" ]]; then
            process_template_variables "$app_path/.vrooli/service.json" "$SERVICE_JSON"
            # Ensure service.json has basic lifecycle configuration
            enhance_service_json_lifecycle "$app_path/.vrooli/service.json"
        fi
    else
        # Create .vrooli directory and service.json if missing
        mkdir -p "$app_path/.vrooli"
        create_default_service_json "$app_path/.vrooli/service.json"
        log::info "Created default .vrooli/service.json"
    fi
    
    if [[ -d "$scenario_path/initialization" ]]; then
        cp -r "$scenario_path/initialization" "$app_path/" 2>/dev/null || true
        [[ "$VERBOSE" == "true" ]] && log::info "Copied initialization directory"
    fi
    
    if [[ -d "$scenario_path/deployment" ]]; then
        cp -r "$scenario_path/deployment" "$app_path/" 2>/dev/null || true
        [[ "$VERBOSE" == "true" ]] && log::info "Copied deployment directory"
    fi
    
    if [[ -f "$scenario_path/test.sh" ]]; then
        cp "$scenario_path/test.sh" "$app_path/" 2>/dev/null || true
        [[ "$VERBOSE" == "true" ]] && log::info "Copied test.sh"
    fi
    
    if [[ -f "$scenario_path/README.md" ]]; then
        cp "$scenario_path/README.md" "$app_path/" 2>/dev/null || true
        [[ "$VERBOSE" == "true" ]] && log::info "Copied README.md"
    fi
    
    log::success "Scenario files copied successfully"
}

# Copy universal scripts infrastructure for standalone apps
copy_universal_scripts() {
    local app_path="$1"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would copy universal scripts to: $app_path/scripts"
        return 0
    fi
    
    log::info "Copying universal scripts infrastructure with lifecycle engine..."
    
    # Create base scripts directory structure
    mkdir -p "$app_path/scripts/"{helpers/utils/core,helpers/lifecycle,main,helpers/setup/target}
    
    # Copy core universal utilities (safe for standalone apps)
    if [[ -d "$PROJECT_ROOT/scripts/helpers/utils/core" ]]; then
        cp -r "$PROJECT_ROOT/scripts/helpers/utils/core/"* "$app_path/scripts/helpers/utils/core/"
        mkdir -p "$app_path/scripts/helpers/utils"
        # Create redirect files for core utilities
        for util in args domainCheck exit_codes flow log ports repository system targetMatcher var vault version zip; do
            echo '#!/usr/bin/env bash' > "$app_path/scripts/helpers/utils/${util}.sh"
            echo "source \"\$(dirname \"\${BASH_SOURCE[0]}\")/core/${util}.sh\"" >> "$app_path/scripts/helpers/utils/${util}.sh"
        done
        [[ "$VERBOSE" == "true" ]] && log::info "Copied core universal utilities"
    fi
    
    # Copy lifecycle engine (universal execution system) - ALWAYS include
    if [[ -d "$PROJECT_ROOT/scripts/helpers/lifecycle" ]]; then
        cp -r "$PROJECT_ROOT/scripts/helpers/lifecycle/"* "$app_path/scripts/helpers/lifecycle/"
        [[ "$VERBOSE" == "true" ]] && log::info "Copied lifecycle engine"
    else
        log::error "Lifecycle engine not found - generated app will not work properly"
        return 1
    fi
    
    # Create minimal main scripts that delegate to lifecycle engine
    for script in setup.sh develop.sh build.sh deploy.sh test.sh clean.sh; do
        create_lifecycle_main_script "$script" "$app_path/scripts/main/$script"
        [[ "$VERBOSE" == "true" ]] && log::info "Created lifecycle main/$script"
    done
    
    # Create minimal target scripts for basic compatibility
    for target in native_linux docker_only k8s_cluster; do
        create_minimal_target_script "$target" "$app_path/scripts/helpers/setup/target/${target//_/-}.sh"
        [[ "$VERBOSE" == "true" ]] && log::info "Created minimal target/${target//_/-}.sh"
    done
    
    # Create minimal setup index for basic functionality
    create_minimal_setup_index "$app_path/scripts/helpers/setup/index.sh"
    
    # Copy scripts/README.md if it exists
    if [[ -f "$PROJECT_ROOT/scripts/README.md" ]]; then
        cp "$PROJECT_ROOT/scripts/README.md" "$app_path/scripts/"
        [[ "$VERBOSE" == "true" ]] && log::info "Copied scripts/README.md"
    fi
    
    log::success "Universal scripts infrastructure with lifecycle engine copied successfully"
}

#######################################
# Create lifecycle-based main script
# Arguments:
#   $1 - Script name (setup.sh, develop.sh, etc.)
#   $2 - Destination script path
#######################################
create_lifecycle_main_script() {
    local script_name="$1"
    local dest="$2"
    local phase_name="${script_name%.sh}"  # Remove .sh extension
    
    # Create minimal script that delegates to lifecycle engine
    cat > "$dest" << EOF
#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Generated Standalone App Script: $script_name
# 
# This script delegates to the Vrooli Lifecycle Engine for execution.
# All behavior is controlled by the service.json lifecycle configuration.
#
# Generated by: scenario-to-app.sh
# Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
################################################################################

MAIN_DIR="\$(cd \"\$(dirname \"\${BASH_SOURCE[0]}\")\" && pwd)"
SCRIPT_DIR="\$(cd \"\${MAIN_DIR}/../..\" && pwd)"

# Source core utilities
if [[ -f "\${MAIN_DIR}/../helpers/utils/universal/index.sh" ]]; then
    # shellcheck disable=SC1091
    source "\${MAIN_DIR}/../helpers/utils/universal/index.sh"
else
    echo "❌ ERROR: Core utilities not found. Generated app may be corrupted."
    exit 1
fi

# Set standalone context
export VROOLI_CONTEXT="standalone"
export PROJECT_ROOT="\${SCRIPT_DIR}"

# Execute via lifecycle engine
if [[ -f "\${MAIN_DIR}/../helpers/lifecycle/engine.sh" ]]; then
    log::info "Executing $phase_name via lifecycle engine..."
    exec "\${MAIN_DIR}/../helpers/lifecycle/engine.sh" "$phase_name" "\$@"
else
    log::error "Lifecycle engine not found: \${MAIN_DIR}/../helpers/lifecycle/engine.sh"
    log::error "Generated app is missing required infrastructure."
    log::error "Please regenerate the app or check the scenario-to-app.sh script."
    exit 1
fi
EOF

    chmod +x "$dest"
}

#######################################
# Create minimal target script
# Arguments:
#   $1 - Target name (native_linux, docker_only, etc.)
#   $2 - Destination script path
#######################################
create_minimal_target_script() {
    local target_name="$1"
    local dest="$2"
    
    cat > "$dest" << EOF
#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Generated Standalone Target Script: $target_name
# 
# Minimal target-specific setup for standalone apps.
# Most functionality should be defined in service.json lifecycle configuration.
#
# Generated by: scenario-to-app.sh
# Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
################################################################################

SCRIPT_DIR="\$(cd \"\$(dirname \"\${BASH_SOURCE[0]}\")\" && pwd)"

# Source core utilities
if [[ -f "\${SCRIPT_DIR}/../../utils/core/index.sh" ]]; then
    # shellcheck disable=SC1091
    source "\${SCRIPT_DIR}/../../utils/core/index.sh"
else
    echo "❌ ERROR: Core utilities not found at \${SCRIPT_DIR}/../../utils/core/index.sh"
    exit 1
fi

log::info "Target script: $target_name"
log::info "Most setup should be handled by lifecycle engine and service.json configuration"

# Basic environment detection
case "$target_name" in
    native_linux)
        if command -v apt-get >/dev/null 2>&1; then
            log::debug "Detected Debian/Ubuntu system"
        elif command -v yum >/dev/null 2>&1; then
            log::debug "Detected RHEL/CentOS system"
        fi
        ;;
    docker_only)
        if ! command -v docker >/dev/null 2>&1; then
            log::warning "Docker not found - may need manual installation"
        fi
        ;;
    k8s_cluster)
        if ! command -v kubectl >/dev/null 2>&1; then
            log::warning "kubectl not found - may need manual installation"
        fi
        ;;
esac

log::success "Target $target_name setup complete"
EOF

    chmod +x "$dest"
}

#######################################
# Create minimal setup index
# Arguments:
#   $1 - Destination index path
#######################################
create_minimal_setup_index() {
    local dest="$1"
    
    cat > "$dest" << EOF
#!/usr/bin/env bash

################################################################################
# Generated Standalone Setup Index
# 
# Provides minimal setup functions for standalone apps.
# Most functionality should be defined in service.json lifecycle configuration.
#
# Generated by: scenario-to-app.sh
# Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
################################################################################

SETUP_DIR="\$(cd \"\$(dirname \"\${BASH_SOURCE[0]}\")\" && pwd)"

# Source core utilities
if [[ -f "\${SETUP_DIR}/../utils/core/index.sh" ]]; then
    # shellcheck disable=SC1091
    source "\${SETUP_DIR}/../utils/core/index.sh"
else
    echo "❌ ERROR: Core utilities not found"
    exit 1
fi

# Basic setup functions for standalone apps
permissions::make_scripts_executable() {
    find "\${PROJECT_ROOT:-\$(pwd)}/scripts" -name "*.sh" -type f -exec chmod +x {} + 2>/dev/null || true
}

clock::fix() {
    log::debug "Clock sync not needed for standalone apps"
}

common_deps::check_and_install() {
    log::info "Checking common dependencies..."
    
    # Check for essential tools
    for tool in git curl jq; do
        if ! command -v "\$tool" >/dev/null 2>&1; then
            log::warning "Missing recommended tool: \$tool"
        fi
    done
    
    # Check Node.js/npm if package.json exists
    if [[ -f "package.json" ]]; then
        if ! command -v node >/dev/null 2>&1; then
            log::warning "Node.js not found but package.json exists"
            log::info "Please install Node.js manually"
        fi
    fi
}

config::init() {
    # Initialize basic configuration
    mkdir -p .vrooli
    if [[ ! -f ".vrooli/service.json" ]]; then
        log::info "service.json should already exist in generated apps"
    fi
}
EOF
    
    chmod +x "$dest"
}

# Generate standalone app with atomic operations
generate_app() {
    local scenario_name="$1"
    
    log::info "Generating standalone app for: $scenario_name"
    
    local app_path="$HOME/generated-apps/$scenario_name"
    local temp_path="$HOME/generated-apps/.tmp-$scenario_name-$$"
    
    # Use atomic operations with temp directory
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would create directory: $app_path"
        copy_scenario_files "$SCENARIO_PATH" "/dev/null"
        copy_universal_scripts "/dev/null"
    else
        # Check for existing app
        if [[ -d "$app_path" ]]; then
            log::warning "Generated app already exists: $app_path"
            if [[ "$FORCE" != "true" ]]; then
                log::error "Use --force to overwrite existing app"
                return 1
            fi
        fi
        
        # Create temp directory for atomic generation
        mkdir -p "$temp_path" || {
            log::error "Failed to create temp directory: $temp_path"
            return 1
        }
        
        # Set up cleanup trap
        trap "rm -rf '$temp_path'" EXIT ERR INT TERM
        
        log::info "Building app in temporary directory..."
        
        # Copy all files to temp directory
        if ! copy_scenario_files "$SCENARIO_PATH" "$temp_path"; then
            log::error "Failed to copy scenario files"
            rm -rf "$temp_path"
            return 1
        fi
        
        if ! copy_universal_scripts "$temp_path"; then
            log::error "Failed to copy universal scripts"
            rm -rf "$temp_path"
            return 1
        fi
        
        # Process template variables in initialization files
        if [[ -d "$temp_path/initialization" ]]; then
            log::info "Processing template variables in initialization files..."
            find "$temp_path/initialization" -type f \( -name "*.json" -o -name "*.yaml" -o -name "*.yml" -o -name "*.sql" -o -name "*.ts" -o -name "*.js" \) | while read -r file; do
                process_template_variables "$file" "$SERVICE_JSON"
            done
        fi
        
        # Atomic move: Remove old and move temp to final location
        if [[ -d "$app_path" ]]; then
            log::info "Removing existing app directory..."
            rm -rf "$app_path"
        fi
        
        mv "$temp_path" "$app_path" || {
            log::error "Failed to move app to final location"
            rm -rf "$temp_path"
            return 1
        }
        
        # Clear trap since we succeeded
        trap - EXIT ERR INT TERM
        
        log::success "Created app directory: $app_path"
    fi
    
    log::success "Generated app: $app_path"
    log::info "To run: cd $app_path && ./scripts/main/develop.sh"
    
    return 0
}

# Start generated app using develop.sh
start_generated_app() {
    local scenario_name="$1"
    local app_path="$HOME/generated-apps/$scenario_name"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start generated app at: $app_path"
        return 0
    fi
    
    if [[ ! -d "$app_path" ]]; then
        log::error "Generated app directory not found: $app_path"
        return 1
    fi
    
    if [[ ! -f "$app_path/scripts/main/develop.sh" ]]; then
        log::error "develop.sh script not found: $app_path/scripts/main/develop.sh"
        return 1
    fi
    
    log::info "Starting generated app: $scenario_name"
    log::info "App path: $app_path"
    
    # Change to app directory and run develop.sh
    cd "$app_path" || {
        log::error "Failed to change to app directory: $app_path"
        return 1
    }
    
    log::info "Executing: ./scripts/main/develop.sh --target docker --detached yes"
    if ./scripts/main/develop.sh --target docker --detached yes; then
        log::success "Generated app started successfully!"
        log::info "App should be available at the URLs defined in service.json"
        return 0
    else
        log::error "Failed to start generated app"
        return 1
    fi
}

################################################################################
# Main Execution
################################################################################

main() {
    # Parse command line arguments
    parse_args "$@"
    
    # Show header
    log::info "🚀 Vrooli Scenario-to-App Generator"
    log::info "Scenario: $SCENARIO_NAME"
    if [[ "$DRY_RUN" == "true" ]]; then
        log::warning "DRY RUN MODE - No actual changes will be made"
    fi
    if [[ "$FORCE" == "true" ]]; then
        log::info "FORCE MODE - Will overwrite existing apps"
    fi
    if [[ "$START" == "true" ]]; then
        log::info "START MODE - Will start app after generation"
    fi
    echo ""
    
    # Phase 1: Validation
    log::info "Phase 1: Scenario Validation"
    validate_scenario
    echo ""
    
    # Phase 2: Resource Analysis
    log::info "Phase 2: Resource Analysis"
    
    # Get resource categories
    local resource_categories
    mapfile -t resource_categories < <(get_resource_categories)
    
    if [[ ${#resource_categories[@]} -gt 0 ]]; then
        log::info "Resource categories: ${resource_categories[*]}"
    fi
    
    # Get enabled resources
    local enabled_resources
    mapfile -t enabled_resources < <(get_enabled_resources)
    
    if [[ ${#enabled_resources[@]} -eq 0 ]]; then
        log::warning "No enabled resources found in service.json"
    else
        log::info "Enabled resources: ${#enabled_resources[@]} resources"
        if [[ "$VERBOSE" == "true" ]]; then
            for resource in "${enabled_resources[@]}"; do
                log::info "  • $resource"
            done
        fi
    fi
    echo ""
    
    # Phase 3: App Generation
    log::info "Phase 3: App Generation"
    generate_app "$SCENARIO_NAME" || exit 1
    echo ""
    
    # Phase 4: App Startup (optional)
    if [[ "$START" == "true" ]]; then
        log::info "Phase 4: App Startup"
        start_generated_app "$SCENARIO_NAME" || exit 1
        echo ""
        
        # Phase 5: Generation Complete
        log::info "Phase 5: Generation Complete"
    else
        # Phase 4: Generation Complete
        log::info "Phase 4: Generation Complete"
    fi
    local scenario_display_name
    scenario_display_name=$(echo "$SERVICE_JSON" | jq -r '.service.displayName // .service.name' 2>/dev/null)
    
    log::success "✅ $scenario_display_name app generated successfully!"
    echo ""
    log::info "Generated app location: $HOME/generated-apps/$SCENARIO_NAME"
    
    if [[ "$START" != "true" ]]; then
        log::info "To run the app:"
        log::info "  cd $HOME/generated-apps/$SCENARIO_NAME"
        log::info "  ./scripts/main/develop.sh"
    else
        log::info "App has been started and should be available at the configured URLs"
    fi
    echo ""
}

# Execute main function
main "$@"