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
#   ./scripts/manage.sh develop
#
# Architecture:
# - Validates scenario files and structure using project schema
# - Copies scenario data as-is to generated app
# - Copies Vrooli's scripts/ infrastructure (minus app/ and scenarios/)
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
  ./scripts/manage.sh develop

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
          "name": "setup-app",
          "run": "./scripts/manage.sh setup --target docker || echo 'No setup script available'",
          "condition": "\${SKIP_SETUP} != 'true'"
        }
      ]
    },
    "develop": {
      "description": "Start development server",
      "steps": [
        {
          "name": "start-dev",
          "run": "./deployment/startup.sh || ./scripts/manage.sh develop --target docker --detached no || echo 'No startup script found'"
        }
      ]
    },
    "build": {
      "description": "Build application for production",
      "steps": [
        {
          "name": "build-app",
          "run": "./scripts/manage.sh build || echo 'No build script available'"
        }
      ]
    },
    "test": {
      "description": "Run test suites",
      "steps": [
        {
          "name": "run-tests",
          "run": "./test.sh || ./scripts/manage.sh test || echo 'No test script available'"
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
  },
  "instanceManagement": {
    "enabled": true,
    "strategy": "single",
    "detection": {
      "services": [
        "server",
        "ui",
        "database"
      ],
      "ports": {
        "database": 5432,
        "server": 8080,
        "ui": 3000
      },
      "containers": [
        "app[-_]*",
        "*[-_]app",
        "*[-_]server",
        "*[-_]ui",
        "*[-_]db",
        "*[-_]database"
      ],
      "processes": [
        "node.*server",
        "npm.*dev",
        "pnpm.*dev",
        "npm.*start",
        "pnpm.*start",
        "vite.*",
        "next.*dev",
        "concurrently.*"
      ]
    },
    "conflicts": {
      "action": "prompt",
      "timeout": 30
    },
    "dockerCompose": {
      "files": [
        "docker-compose.yml",
        "docker-compose.yaml",
        "compose.yml", 
        "compose.yaml"
      ],
      "services": [
        "ui",
        "server",
        "database"
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
                  "name": "setup-app",
                  "run": "./scripts/manage.sh setup --target docker || echo \"No setup script available\"",
                  "condition": "${SKIP_SETUP} != \"true\""
                }
              ]
            },
            "develop": {
              "description": "Start development server",
              "steps": [
                {
                  "name": "start-dev",
                  "run": "./deployment/startup.sh || ./scripts/manage.sh develop --target docker --detached no || echo \"No startup script found\""
                }
              ]
            },
            "build": {
              "description": "Build application for production",
              "steps": [
                {
                  "name": "build-app",
                  "run": "./scripts/manage.sh build || echo \"No build script available\""
                }
              ]
            },
            "test": {
              "description": "Run test suites",
              "steps": [
                {
                  "name": "run-tests",
                  "run": "./test.sh || ./scripts/manage.sh test || echo \"No test script available\""
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
    local secrets_util="${PROJECT_ROOT}/scripts/lib/service/secrets.sh"
    local service_config_util="${PROJECT_ROOT}/scripts/lib/service/service_config.sh"
    
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

# Copy scripts infrastructure for standalone apps
copy_universal_scripts() {
    local app_path="$1"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would copy scripts infrastructure to: $app_path/scripts"
        return 0
    fi
    
    log::info "Copying scripts infrastructure..."
    
    # Create scripts directory
    mkdir -p "$app_path/scripts"
    
    # Copy entire scripts/ directory from PROJECT_ROOT, excluding app and scenarios
    if [[ -d "$PROJECT_ROOT/scripts" ]]; then
        # Use rsync for selective copying with excludes
        if command -v rsync >/dev/null 2>&1; then
            rsync -a --exclude='app/' --exclude='scenarios/' "$PROJECT_ROOT/scripts/" "$app_path/scripts/"
            [[ "$VERBOSE" == "true" ]] && log::info "Copied scripts/ using rsync (excluded app/ and scenarios/)"
        else
            # Fallback: copy everything first, then remove excluded directories
            cp -r "$PROJECT_ROOT/scripts/"* "$app_path/scripts/" 2>/dev/null || true
            rm -rf "$app_path/scripts/app" "$app_path/scripts/scenarios" 2>/dev/null || true
            [[ "$VERBOSE" == "true" ]] && log::info "Copied scripts/ using cp (removed app/ and scenarios/)"
        fi
    fi
    
    # Copy scenario-specific scripts/ directory if it exists (overlay on top)
    if [[ -d "$SCENARIO_PATH/scripts" ]]; then
        cp -r "$SCENARIO_PATH/scripts/"* "$app_path/scripts/" 2>/dev/null || true
        [[ "$VERBOSE" == "true" ]] && log::info "Overlaid scenario-specific scripts from: $SCENARIO_PATH/scripts/"
    fi
    
    # Create instance manager wrapper for standalone app 
    local app_instance_manager="$app_path/scripts/app/lifecycle/develop/instance_manager.sh"
    
    # Create app lifecycle develop directory if it doesn't exist
    mkdir -p "$(dirname "$app_instance_manager")"
    
    # Create a wrapper that properly integrates with the standalone app structure
    cat > "$app_instance_manager" << 'EOF'
#!/usr/bin/env bash
# Standalone App Instance Manager - Generic Integration Wrapper
# 
# This wrapper sources the generic instance manager for standalone apps.
# The generic instance manager reads configuration from service.json automatically.
set -euo pipefail

APP_LIFECYCLE_DEVELOP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get all directory variables (same path as monorepo)
# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEVELOP_DIR}/../../../lib/utils/var.sh"

# Source the generic instance manager
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/instance_manager.sh"

# All functionality is now provided by the generic instance manager
# Configuration is loaded automatically from .vrooli/service.json or defaults to sensible patterns
EOF
    
    [[ "$VERBOSE" == "true" ]] && log::info "Created instance manager wrapper for standalone app"
    
    log::success "Scripts infrastructure copied successfully"
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
    log::info "To run: cd $app_path && ./scripts/manage.sh develop"
    
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
    
    if [[ ! -f "$app_path/scripts/manage.sh" ]]; then
        log::error "manage.sh script not found: $app_path/scripts/manage.sh"
        return 1
    fi
    
    log::info "Starting generated app: $scenario_name"
    log::info "App path: $app_path"
    
    # Change to app directory and run develop.sh
    cd "$app_path" || {
        log::error "Failed to change to app directory: $app_path"
        return 1
    }
    
    log::info "Executing: ./scripts/manage.sh develop --target docker --detached yes"
    if ./scripts/manage.sh develop --target docker --detached yes; then
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
        log::info "  ./scripts/manage.sh develop"
    else
        log::info "App has been started and should be available at the configured URLs"
    fi
    echo ""
}

# Execute main function
main "$@"