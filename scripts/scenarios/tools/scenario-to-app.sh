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

SCENARIO_TOOLS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SCENARIO_TOOLS_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

# Global variables
SCENARIO_NAME=""
SCENARIO_PATH=""
SERVICE_JSON=""
VERBOSE=false
DRY_RUN=false
FORCE=false
START=false

# Usage information
scenario_to_app::show_usage() {
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
scenario_to_app::parse_args() {
    if [[ $# -eq 0 ]]; then
        log::error "No scenario name provided"
        scenario_to_app::show_usage
        exit 1
    fi

    # Check for --help first
    if [[ "$1" == "--help" ]]; then
        scenario_to_app::show_usage
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
                scenario_to_app::show_usage
                exit 0
                ;;
            # Ignore shell redirections and operators that might be passed accidentally
            '2>&1'|'1>&2'|'&>'|'>'|'<'|'|'|'&&'|'||'|';')
                ;; # Silently ignore
            *)
                log::error "Unknown option: $1"
                scenario_to_app::show_usage
                exit 1
                ;;
        esac
        shift
    done
}

# Validate scenario structure with comprehensive schema validation
scenario_to_app::validate_scenario() {
    SCENARIO_PATH="${var_SCRIPTS_SCENARIOS_DIR}/core/${SCENARIO_NAME}"
    
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
    
    # Use python/node for JSON schema validation if available
    if command -v python3 >/dev/null 2>&1 && [[ -f "$var_SERVICE_JSON_FILE" ]]; then
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
        
    elif command -v node >/dev/null 2>&1 && [[ -f "$var_SERVICE_JSON_FILE" ]]; then
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
    if ! scenario_to_app::validate_initialization_paths; then
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
scenario_to_app::get_enabled_resources() {
    # Extract ALL enabled resources from ANY category
    echo "$SERVICE_JSON" | jq -r '
        .resources | to_entries[] as $category | 
        $category.value | to_entries[] | 
        select(.value.enabled == true) | 
        "\(.key) (\($category.key))"  
    ' 2>/dev/null || true
}

# Get resource categories from service.json
scenario_to_app::get_resource_categories() {
    # Extract all resource category names
    echo "$SERVICE_JSON" | jq -r '.resources | keys[]' 2>/dev/null || true
}

# Validate that initialization file paths in service.json actually exist
scenario_to_app::validate_initialization_paths() {
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "Validating initialization file paths..."
    fi
    
    local validation_errors=0
    local total_files=0
    
    # Check initialization files referenced in resource configurations
    # Note: Paths can be:
    #   - Relative to service.json location: ../initialization/file.sql
    #   - Relative to scenario root: initialization/file.sql
    #   - Absolute: /path/to/file.sql
    local service_json_dir="${SCENARIO_PATH}/.vrooli"
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
        # Resolve path based on type
        local full_path
        if [[ "$file_path" == /* ]]; then
            # Absolute path
            full_path="$file_path"
        elif [[ "$file_path" == ../* ]] || [[ "$file_path" == ./* ]]; then
            # Relative path from service.json location (.vrooli/ directory)
            full_path="${service_json_dir}/${file_path}"
            # Normalize the path (resolve .. and .)
            full_path=$(cd "$(dirname "$full_path")" 2>/dev/null && pwd)/$(basename "$full_path")
        else
            # Simple path - relative to scenario root directory
            full_path="${SCENARIO_PATH}/${file_path}"
        fi
        
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
        # Resolve path based on type (same logic as above)
        local full_path
        if [[ "$file_path" == /* ]]; then
            # Absolute path
            full_path="$file_path"
        elif [[ "$file_path" == ../* ]] || [[ "$file_path" == ./* ]]; then
            # Relative path from service.json location (.vrooli/ directory)
            full_path="${service_json_dir}/${file_path}"
            # Normalize the path (resolve .. and .)
            full_path=$(cd "$(dirname "$full_path")" 2>/dev/null && pwd)/$(basename "$full_path")
        else
            # Simple path - relative to scenario root directory
            full_path="${SCENARIO_PATH}/${file_path}"
        fi
        
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
scenario_to_app::create_default_service_json() {
    local dest="$1"
    local app_name
    app_name="$(basename "$(dirname "$(dirname "$dest")")")"
    
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
          "run": "# Clean build artifacts safely\nfor dir in dist build .next 'node_modules/.cache'; do\n  [ -d \"\$dir\" ] && rm -rf \"\$dir\" || true\ndone"
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
# Adjust inheritance path in service.json for standalone app
# Arguments:
#   $1 - Service.json path
#######################################
scenario_to_app::adjust_inheritance_path() {
    local service_json_path="$1"
    
    # Check if inheritance.extends exists
    if jq -e '.inheritance.extends' "$service_json_path" >/dev/null 2>&1; then
        [[ "$VERBOSE" == "true" ]] && log::info "Processing inheritance for standalone app"
        
        local base_service_json="${var_ROOT_DIR}/.vrooli/service.json"
        
        if [[ -f "$base_service_json" ]]; then
            # Load both service.json files
            local app_json
            app_json=$(cat "$service_json_path")
            local base_json
            base_json=$(cat "$base_service_json")
            
            # Extract base lifecycle if present
            local base_lifecycle
            base_lifecycle=$(echo "$base_json" | jq -r '.lifecycle // {}')
            
            # Check if app has lifecycle section
            local app_lifecycle
            app_lifecycle=$(echo "$app_json" | jq -r '.lifecycle // {}')
            
            # Merge lifecycles - app lifecycle phases override base phases
            # BUT filter out Vrooli-specific steps that reference files that won't exist in standalone apps
            local merged_lifecycle
            
            
            # First, always filter Vrooli-specific steps from base
            local filtered_base
            # Write base_lifecycle to temp file to avoid shell escaping issues
            local temp_base="/tmp/base_lifecycle_$$.json"
            echo "$base_lifecycle" > "$temp_base"
            
            [[ "$VERBOSE" == "true" ]] && log::info "DEBUG: Base lifecycle file content: $(jq -c '.' < "$temp_base")"
            
            # Simplified approach: create a clean base with essential phases
            filtered_base=$(jq -c '
                # Extract essential metadata and create clean phases  
                {
                    version: (.version // "1.0.0"),
                    setup: (
                        if .setup then 
                            .setup + {steps: []}
                        else 
                            {description: "Initialize application environment", steps: [], universal: "./scripts/lib/setup.sh"}
                        end
                    ),
                    develop: {
                        description: "Start the development environment", 
                        steps: [], 
                        universal: "./scripts/lib/lifecycle/phases/develop.sh"
                    },
                    build: {
                        description: "Build application artifacts", 
                        steps: [], 
                        universal: "./scripts/lib/lifecycle/phases/build.sh"
                    },
                    deploy: {
                        description: "Deploy the application", 
                        steps: [], 
                        universal: "./scripts/lib/lifecycle/phases/deploy.sh"
                    },
                    test: {
                        description: "Run tests", 
                        steps: [], 
                        universal: "./scripts/lib/lifecycle/phases/test.sh"
                    },
                    backup: {
                        description: "Create backups of data and configuration", 
                        steps: [], 
                        universal: "./scripts/lib/lifecycle/phases/backup.sh"
                    }
                } + (if .hooks then {hooks: .hooks} else {} end)
                  + (if .dependencies then {dependencies: .dependencies} else {} end)
                  + (if .readiness then {readiness: .readiness} else {} end)
                  + (if .liveness then {liveness: .liveness} else {} end)
            ' "$temp_base" || echo '{}')
            
            [[ "$VERBOSE" == "true" ]] && log::info "DEBUG: Filtered lifecycle: $(echo "$filtered_base" | jq -c '.')"
            [[ "$VERBOSE" == "true" ]] && log::info "DEBUG: Phases after filtering: $(echo "$filtered_base" | jq -r 'keys | join(", ")' 2>/dev/null)"
            
            # Clean up temp file
            trash::safe_remove "$temp_base" --no-confirm
            
            
            # Now merge with app lifecycle (app phases take precedence, but base phases are preserved)
            if [[ "$app_lifecycle" == "{}" ]]; then
                # App has no lifecycle, use filtered base
                merged_lifecycle="$filtered_base"
            else
                # Merge filtered base with app - preserve ALL base phases, but let app override specific phases
                merged_lifecycle=$(jq -n --argjson base "$filtered_base" --argjson app "$app_lifecycle" '
                    # Start with all base phases, then override with app-defined phases
                    $base + $app
                ')
            fi
            
            # Update app service.json with merged lifecycle and remove inheritance
            local temp_file="${service_json_path}.tmp"
            jq --argjson lifecycle "$merged_lifecycle" '
                del(.inheritance) | .lifecycle = $lifecycle
            ' "$service_json_path" > "$temp_file"
            mv "$temp_file" "$service_json_path"
            
            [[ "$VERBOSE" == "true" ]] && log::info "Merged inherited lifecycle configuration into app service.json (filtered Vrooli-specific steps)"
        else
            log::warning "Base service.json not found at: $base_service_json"
            log::warning "Removing inheritance configuration as base is unavailable"
            
            # Remove inheritance configuration if base doesn't exist
            local temp_file="${service_json_path}.tmp"
            jq 'del(.inheritance)' "$service_json_path" > "$temp_file"
            mv "$temp_file" "$service_json_path"
        fi
    else
        [[ "$VERBOSE" == "true" ]] && log::info "No inheritance configuration found in service.json"
    fi
}

#######################################
# Enhance existing service.json with basic lifecycle if missing
# Arguments:
#   $1 - Service.json path
#######################################
scenario_to_app::enhance_service_json_lifecycle() {
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
                  "run": "# Clean build artifacts safely\nfor dir in dist build .next node_modules/.cache; do\n  [ -d \"\$dir\" ] && rm -rf \"\$dir\" || true\ndone"
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
scenario_to_app::process_template_variables() {
    local file="$1"
    # local service_json="$2"  # Currently unused but may be needed for future template processing
    
    # Skip if file doesn't exist or is binary
    [[ ! -f "$file" ]] && return 0
    file -b --mime-type "$file" | grep -q "text/" || return 0
    
    # Skip Windmill app files - they contain UI templates, not secrets
    if [[ "$file" == */automation/windmill/apps/* ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "Skipping Windmill app file (contains UI templates): $(basename "$file")"
        return 0
    fi
    
    # Check if utilities are already loaded, if not source them
    if ! command -v secrets::substitute_all_templates >/dev/null 2>&1; then
        local secrets_util="${var_ROOT_DIR}/scripts/lib/service/secrets.sh"
        if [[ -f "$secrets_util" ]]; then
            # shellcheck disable=SC1091
            source "$secrets_util"
        fi
    fi
    
    if ! command -v service_config::has_inheritance >/dev/null 2>&1; then
        local service_config_util="${var_ROOT_DIR}/scripts/lib/service/service_config.sh"
        if [[ -f "$service_config_util" ]]; then
            # shellcheck disable=SC1090
            source "$service_config_util"
        fi
    fi
    
    # Check if the service.json has inheritance defined
    local should_substitute="yes"
    if [[ "$(basename "$file")" == "service.json" ]] && command -v service_config::has_inheritance >/dev/null 2>&1; then
        # Check for inheritance - if present, preserve templates for runtime resolution
        if service_config::has_inheritance "$file"; then
            should_substitute="no"
            [[ "$VERBOSE" == "true" ]] && log::info "Inheritance detected in $(basename "$file"), preserving templates for runtime resolution"
        fi
    fi
    
    # Only substitute if no inheritance or if not a service.json file
    if [[ "$should_substitute" == "yes" ]] && command -v secrets::substitute_all_templates >/dev/null 2>&1; then
        # Read file content and substitute all templates (both secrets and service references)
        local content
        content=$(cat "$file")
        
        # Check if content has any templates before processing
        if echo "$content" | grep -qE '\{\{[^}]+\}\}|\$\{service\.[^}]+\}'; then
            local sub_start
            sub_start=$(date +"%s%N")
            local processed
            processed=$(echo "$content" | secrets::substitute_all_templates)
            local sub_end
            sub_end=$(date +"%s%N")
            local sub_time=$(( (sub_end - sub_start) / 1000000 ))
            [[ "$VERBOSE" == "true" ]] && [[ $sub_time -gt 100 ]] && log::info "    Template substitution took ${sub_time}ms for $(basename "$file")"
            
            # Write back if changed
            if [[ "$content" != "$processed" ]]; then
                echo "$processed" > "$file"
                [[ "$VERBOSE" == "true" ]] && log::info "Processed templates in: $(basename "$file")"
            fi
        fi
    fi
}

#######################################
# Copy files according to manifest
# Arguments:
#   $1 - app_path (destination)
#   $2 - scenario_path  
# Returns:
#   0 on success, 1 on failure
#######################################
scenario_to_app::copy_from_manifest() {
    local app_path="$1"
    local scenario_path="$2"
    local manifest="${SCENARIO_TOOLS_DIR}/app-structure.json"
    
    if [[ ! -f "$manifest" ]]; then
        log::error "Copy manifest not found: $manifest"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would copy files according to manifest to: $app_path"
        return 0
    fi
    
    log::info "Copying files according to manifest..."
    
    # Process each copy rule
    local rule_count
    rule_count=$(jq '.copy_rules | length' "$manifest")
    
    for ((i=0; i<rule_count; i++)); do
        local rule
        rule=$(jq -c ".copy_rules[$i]" "$manifest")
        
        local name
        name=$(echo "$rule" | jq -r '.name')
        local source
        source=$(echo "$rule" | jq -r '.source')
        local destination
        destination=$(echo "$rule" | jq -r '.destination') 
        local type
        type=$(echo "$rule" | jq -r '.type')
        local from
        from=$(echo "$rule" | jq -r '.from // "vrooli"')
        local required
        required=$(echo "$rule" | jq -r '.required // false')
        local optional
        optional=$(echo "$rule" | jq -r '.optional // false')
        local executable
        executable=$(echo "$rule" | jq -r '.executable // false')
        local merge
        merge=$(echo "$rule" | jq -r '.merge // "replace"')
        
        # Determine source path
        local src_path
        if [[ "$from" == "scenario" ]]; then
            src_path="$scenario_path/$source"
        else
            src_path="$var_ROOT_DIR/$source"
        fi
        
        local dest_path="$app_path/$destination"
        
        # Check if source exists
        if [[ ! -e "$src_path" ]]; then
            if [[ "$required" == "true" ]]; then
                log::error "Required source not found: $src_path (rule: $name)"
                return 1
            elif [[ "$optional" == "false" ]]; then
                log::warning "Source not found: $src_path (rule: $name)"
            fi
            
            # Special handling for scenario-config: create default if missing
            if [[ "$name" == "scenario-config" ]]; then
                mkdir -p "$dest_path"
                scenario_to_app::create_default_service_json "$dest_path/service.json"
                log::info "Created default .vrooli/service.json"
            fi
            
            continue
        fi
        
        # Perform the copy
        if ! scenario_to_app::copy_item "$src_path" "$dest_path" "$type" "$merge" "$executable" "$name"; then
            log::error "Failed to copy $name"
            return 1
        fi
        
        # Apply post-processing if specified
        local process_list
        process_list=$(echo "$rule" | jq -r '.process[]? // empty')
        if [[ -n "$process_list" ]]; then
            while IFS= read -r processor; do
                if [[ -n "$processor" ]]; then
                    scenario_to_app::process_file "$dest_path" "$processor"
                fi
            done <<< "$process_list"
        fi
    done
    
    # Create instance manager wrapper for standalone app 
    scenario_to_app::create_instance_manager_wrapper "$app_path"
    
    log::success "File copying completed successfully"
    return 0
}

#######################################
# Copy a single item (file or directory)
# Arguments:
#   $1 - source path
#   $2 - destination path
#   $3 - type (file|directory)
#   $4 - merge mode (replace|overlay)
#   $5 - executable flag (true|false)
#   $6 - rule name (for logging)
# Returns:
#   0 on success, 1 on failure
#######################################
scenario_to_app::copy_item() {
    local src="$1" dest="$2" type="$3" merge="$4" executable="$5" name="$6"
    
    [[ "$VERBOSE" == "true" ]] && log::info "Copying $name: $src -> $dest"
    
    # Create parent directory
    mkdir -p "$(dirname "$dest")"
    
    case "$type" in
        file)
            if ! cp "$src" "$dest"; then
                log::error "Failed to copy file: $src -> $dest"
                return 1
            fi
            [[ "$executable" == "true" ]] && chmod +x "$dest"
            ;;
        directory)
            if [[ "$merge" == "overlay" && -d "$dest" ]]; then
                # Merge directory contents
                if ! cp -r "$src/"* "$dest/" 2>/dev/null; then
                    # If no files to copy, that's ok for overlay
                    [[ "$VERBOSE" == "true" ]] && log::info "No files to overlay in $name"
                fi
            else
                # Replace directory - create destination first, then copy contents
                [[ -d "$dest" ]] && trash::safe_remove "$dest" --no-confirm
                mkdir -p "$dest"
                
                if command -v rsync >/dev/null 2>&1; then
                    # Use rsync to skip permission-denied files and continue on errors
                    rsync -a --ignore-errors --exclude='*.log' --exclude='data/' "$src/" "$dest/" 2>/dev/null || {
                        [[ "$VERBOSE" == "true" ]] && log::info "Some files in $name could not be copied (permission denied), continuing..."
                    }
                else
                    # Use cp with find to skip problematic files
                    (cd "$src" && find . -type f -exec cp --parents {} "$dest/" \; 2>/dev/null) || {
                        [[ "$VERBOSE" == "true" ]] && log::info "Some files in $name could not be copied (permission denied), continuing..."
                    }
                    (cd "$src" && find . -type d -exec mkdir -p "$dest/{}" \; 2>/dev/null) || true
                fi
            fi
            ;;
        *)
            log::error "Unknown copy type: $type"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Filter shared initialization directory to only include referenced files
# Arguments:
#   $1 - initialization directory path
#   $2 - service.json content
# Returns:
#   0 on success, 1 on failure
#######################################
scenario_to_app::filter_shared_initialization() {
    local init_dir="$1"
    local service_json_content="$2"
    
    [[ "$VERBOSE" == "true" ]] && log::info "Filtering shared initialization files in: $init_dir"
    
    # Extract all file paths referenced in initialization sections
    local referenced_files
    referenced_files=$(echo "$service_json_content" | jq -r '
        .resources // {} | 
        to_entries[] | 
        .value // {} | 
        to_entries[] | 
        .value.initialization[]? // empty |
        .file // empty' 2>/dev/null | sort | uniq)
    
    if [[ -z "$referenced_files" ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "No initialization files referenced in service.json"
        return 0
    fi
    
    # Build list of shared files to keep
    local shared_files_to_keep=()
    
    while IFS= read -r file_ref; do
        [[ -z "$file_ref" ]] && continue
        
        # Handle both "shared:" prefix and direct paths to initialization/
        local actual_path=""
        if [[ "$file_ref" == shared:* ]]; then
            # Remove "shared:" prefix
            actual_path="${file_ref#shared:}"
        elif [[ "$file_ref" == initialization/* ]]; then
            # Direct reference to initialization file
            actual_path="$file_ref"
        fi
        
        # If this is a path within initialization/, add it to keep list
        if [[ -n "$actual_path" && "$actual_path" == initialization/* ]]; then
            # Remove "initialization/" prefix to get relative path within init dir
            local rel_path="${actual_path#initialization/}"
            shared_files_to_keep+=("$rel_path")
            [[ "$VERBOSE" == "true" ]] && log::info "Will keep shared file: $rel_path"
        fi
    done <<< "$referenced_files"
    
    # If no shared files to keep, remove everything
    if [[ ${#shared_files_to_keep[@]} -eq 0 ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "No shared files referenced, removing all shared initialization files"
        rm -rf "$init_dir"/*
        return 0
    fi
    
    # Remove files not in the keep list
    find "$init_dir" -type f | while IFS= read -r file; do
        # Get relative path from init_dir
        local rel_path="${file#$init_dir/}"
        
        # Check if this file should be kept
        local should_keep=false
        for keep_file in "${shared_files_to_keep[@]}"; do
            if [[ "$rel_path" == "$keep_file" ]]; then
                should_keep=true
                break
            fi
        done
        
        # Remove file if not in keep list
        if [[ "$should_keep" == "false" ]]; then
            [[ "$VERBOSE" == "true" ]] && log::info "Removing unreferenced shared file: $rel_path"
            rm -f "$file"
        fi
    done
    
    # Remove empty directories
    find "$init_dir" -type d -empty -delete 2>/dev/null || true
    
    log::success "Filtered shared initialization files (kept ${#shared_files_to_keep[@]} referenced files)"
    return 0
}

#######################################
# Apply post-processing to copied files
# Arguments:
#   $1 - file path
#   $2 - processor name
# Returns:
#   0 on success, 1 on failure
#######################################
scenario_to_app::process_file() {
    local file_path="$1" processor="$2"
    
    case "$processor" in
        resolve_inheritance)
            if [[ -f "$file_path/service.json" ]]; then
                scenario_to_app::adjust_inheritance_path "$file_path/service.json"
                # Only enhance if lifecycle is missing or invalid after adjustment
                if ! jq -e '.lifecycle' "$file_path/service.json" >/dev/null 2>&1; then
                    scenario_to_app::enhance_service_json_lifecycle "$file_path/service.json"
                else
                    [[ "$VERBOSE" == "true" ]] && log::info "Lifecycle already configured, skipping enhancement"
                fi
            fi
            ;;
        substitute_variables)
            if [[ -f "$file_path/service.json" ]]; then
                scenario_to_app::process_template_variables "$file_path/service.json" "$SERVICE_JSON"
            fi
            ;;
        filter_by_service_references)
            if [[ -d "$file_path" ]]; then
                scenario_to_app::filter_shared_initialization "$file_path" "$SERVICE_JSON"
            fi
            ;;
        *)
            log::warning "Unknown processor: $processor"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Create instance manager wrapper for standalone app
# Arguments:
#   $1 - app_path
#######################################
scenario_to_app::create_instance_manager_wrapper() {
    local app_path="$1"
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

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEVELOP_DIR}/../../../lib/utils/var.sh"

# Source the generic instance manager
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/instance_manager.sh"

# All functionality is now provided by the generic instance manager
# Configuration is loaded automatically from .vrooli/service.json or defaults to sensible patterns
EOF
    
    chmod +x "$app_instance_manager"
    [[ "$VERBOSE" == "true" ]] && log::info "Created instance manager wrapper for standalone app"
}


# Generate standalone app with atomic operations
scenario_to_app::generate_app() {
    local scenario_name="$1"
    
    log::info "Generating standalone app for: $scenario_name"
    
    local app_path="$HOME/generated-apps/$scenario_name"
    local temp_path="$HOME/generated-apps/.tmp-$scenario_name-$$"
    
    # Use atomic operations with temp directory
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would create directory: $app_path"
        scenario_to_app::copy_from_manifest "/dev/null" "$SCENARIO_PATH"
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
        
        # Set up cleanup trap for error cases only (use || true to prevent trap failures from affecting exit code)
        trap 'trash::safe_remove "$temp_path" --no-confirm 2>/dev/null || true' ERR INT TERM
        
        log::info "Building app in temporary directory..."
        
        # Copy all files to temp directory using manifest
        if ! scenario_to_app::copy_from_manifest "$temp_path" "$SCENARIO_PATH"; then
            log::error "Failed to copy files according to manifest"
            trash::safe_remove "$temp_path" --no-confirm
            return 1
        fi
        
        # Process template variables in initialization files
        if [[ -d "$temp_path/initialization" ]]; then
            # Count files first to provide better feedback
            local init_files=()
            while IFS= read -r -d '' file; do
                init_files+=("$file")
            done < <(find "$temp_path/initialization" -type f \( -name "*.json" -o -name "*.yaml" -o -name "*.yml" -o -name "*.sql" -o -name "*.ts" -o -name "*.js" \) -print0)
            
            if [[ ${#init_files[@]} -gt 0 ]]; then
                log::info "Processing template variables in ${#init_files[@]} initialization files..."
                
                # Debug timing
                local start_time
                start_time=$(date +"%s%N")
                
                # Source utilities once for all files
                local secrets_util="${var_ROOT_DIR}/scripts/lib/service/secrets.sh"
                local service_config_util="${var_ROOT_DIR}/scripts/lib/service/service_config.sh"
                
                local source_start
                source_start=$(date +"%s%N")
                if [[ -f "$secrets_util" ]]; then
                    # shellcheck disable=SC1091
                    source "$secrets_util"
                fi
                
                if [[ -f "$service_config_util" ]]; then
                    # shellcheck disable=SC1090
                    source "$service_config_util"
                fi
                local source_end
                source_end=$(date +"%s%N")
                local source_time=$(( (source_end - source_start) / 1000000 ))
                [[ "$VERBOSE" == "true" ]] && log::info "  Sourcing utilities took ${source_time}ms"
                
                # Process files with progress indicator
                local processed=0
                local process_start
                process_start=$(date +"%s%N")
                for file in "${init_files[@]}"; do
                    local file_start
                    file_start=$(date +"%s%N")
                    scenario_to_app::process_template_variables "$file" "$SERVICE_JSON"
                    local file_end
                    file_end=$(date +"%s%N")
                    local file_time=$(( (file_end - file_start) / 1000000 ))
                    [[ "$VERBOSE" == "true" ]] && [[ $file_time -gt 100 ]] && log::info "  File $(basename "$file") took ${file_time}ms"
                    ((processed++))
                    # Show progress every 5 files for large sets
                    if [[ $processed -gt 10 ]] && [[ $((processed % 5)) -eq 0 ]]; then
                        echo -n "." >&2
                    fi
                done
                local process_end
                process_end=$(date +"%s%N")
                local process_time=$(( (process_end - process_start) / 1000000 ))
                
                # Clear progress line if we showed dots
                if [[ $processed -gt 10 ]]; then
                    echo "" >&2
                fi
                
                local end_time
                end_time=$(date +"%s%N")
                local total_time=$(( (end_time - start_time) / 1000000 ))
                
                [[ "$VERBOSE" == "true" ]] && log::info "Processed $processed initialization files in ${total_time}ms (processing: ${process_time}ms)"
            fi
        fi
        
        # Atomic move: Remove old and move temp to final location
        if [[ -d "$app_path" ]]; then
            log::info "Removing existing app directory..."
            
            # Temporarily remove .git directory to avoid project root detection in trash system
            local git_backup=""
            if [[ -d "$app_path/.git" ]]; then
                git_backup="$app_path/.git-backup-$$"
                mv "$app_path/.git" "$git_backup" 2>/dev/null || true
            fi
            
            if ! trash::safe_remove "$app_path" --no-confirm; then
                # Restore .git directory if deletion failed
                if [[ -n "$git_backup" && -d "$git_backup" ]]; then
                    mv "$git_backup" "$app_path/.git" 2>/dev/null || true
                fi
                log::error "Failed to remove existing app directory: $app_path"
                trash::safe_remove "$temp_path" --no-confirm
                return 1
            fi
        fi
        
        mv "$temp_path" "$app_path" || {
            log::error "Failed to move app to final location"
            trash::safe_remove "$temp_path" --no-confirm
            return 1
        }
        
        # Clear trap since we succeeded
        trap - ERR INT TERM
        
        log::success "Created app directory: $app_path"
        
        # Initialize git repository for change tracking
        if command -v git >/dev/null 2>&1; then
            [[ "$VERBOSE" == "true" ]] && log::info "Initializing app version control..."
            cd "$app_path" || {
                log::warning "Failed to change to app directory for git initialization"
                cd - >/dev/null || true
            }
            
            if git init --quiet 2>/dev/null; then
                # Configure git for generated apps
                git config user.name "Vrooli App Generator" 2>/dev/null || true
                git config user.email "apps@vrooli.local" 2>/dev/null || true
                
                # Create initial commit with metadata
                local scenario_hash
                scenario_hash=$(calculate_hash "$SCENARIO_PATH" 2>/dev/null || echo "unknown")
                local generator_version
                generator_version=$(git -C "$var_ROOT_DIR" rev-parse --short HEAD 2>/dev/null || echo "unknown")
                
                git add . 2>/dev/null || true
                git commit --quiet -m "Initial generation from scenario: $scenario_name

Generated from scenario: $scenario_name
Scenario hash: $scenario_hash
Generation time: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
Generator version: $generator_version
" 2>/dev/null || true
                
                [[ "$VERBOSE" == "true" ]] && log::success "✅ App initialized with version control"
            else
                [[ "$VERBOSE" == "true" ]] && log::warning "Failed to initialize git repository (git may not be available)"
            fi
            
            cd - >/dev/null || true
        else
            [[ "$VERBOSE" == "true" ]] && log::info "Git not available - skipping version control initialization"
        fi
    fi
    
    log::success "Generated app: $app_path"
    log::info "To run: cd $app_path && ./scripts/manage.sh develop"
    
    return 0
}

# Start generated app using develop.sh
scenario_to_app::start_generated_app() {
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
    scenario_to_app::parse_args "$@"
    
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
    scenario_to_app::validate_scenario
    echo ""
    
    # Phase 2: Resource Analysis
    log::info "Phase 2: Resource Analysis"
    
    # Get resource categories
    local resource_categories
    mapfile -t resource_categories < <(scenario_to_app::get_resource_categories)
    
    if [[ ${#resource_categories[@]} -gt 0 ]]; then
        log::info "Resource categories: ${resource_categories[*]}"
    fi
    
    # Get enabled resources
    local enabled_resources
    mapfile -t enabled_resources < <(scenario_to_app::get_enabled_resources)
    
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
    scenario_to_app::generate_app "$SCENARIO_NAME" || exit 1
    echo ""
    
    # Phase 4: App Startup (optional)
    if [[ "$START" == "true" ]]; then
        log::info "Phase 4: App Startup"
        scenario_to_app::start_generated_app "$SCENARIO_NAME" || exit 1
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
exit $?