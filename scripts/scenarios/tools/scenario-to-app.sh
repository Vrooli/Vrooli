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

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCENARIO_TOOLS_DIR="${APP_ROOT}/scripts/scenarios/tools"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

# Global variables
SCENARIO_NAMES=()
VERBOSE=false
DRY_RUN=false
FORCE=false
START=false

# Shared state for batch processing
SHARED_BASE_SERVICE_JSON=""
SHARED_UTILS_LOADED=false

# Usage information
scenario_to_app::show_usage() {
    cat << EOF
Usage: $0 <scenario-name> [scenario-name2 ...] [options]

Generate standalone applications from validated Vrooli scenarios.
Supports both single scenario and batch processing.

Arguments:
  scenario-name     One or more scenario names (e.g., campaign-content-studio)

Options:
  --verbose         Enable verbose output
  --dry-run         Show what would be generated without creating files
  --force           Overwrite existing generated apps
  --start           Start the generated apps after creation
  --help            Show this help message

The generated apps will be created at: ~/generated-apps/<scenario-name>/

To run a generated app:
  cd ~/generated-apps/<scenario-name>
  ./scripts/manage.sh develop

Examples:
  $0 campaign-content-studio
  $0 research-assistant --verbose --force
  $0 research-assistant simple-test make-it-vegan --dry-run
  $0 campaign-content-studio --start

Batch Processing:
  When multiple scenarios are provided, they are processed efficiently
  with shared initialization to reduce overhead.

EOF
}

# Parse command line arguments
scenario_to_app::parse_args() {
    if [[ $# -eq 0 ]]; then
        log::error "No scenario names provided"
        scenario_to_app::show_usage
        exit 1
    fi

    # Check for --help first
    if [[ "$1" == "--help" ]]; then
        scenario_to_app::show_usage
        exit 0
    fi

    local scenarios=()
    
    # Parse arguments - collect scenarios and options
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
            -*)
                log::error "Unknown option: $1"
                scenario_to_app::show_usage
                exit 1
                ;;
            *)
                # This is a scenario name - validate it
                local name="$1"
                
                # Validate scenario name format
                if [[ -z "$name" ]]; then
                    log::error "Empty scenario name provided"
                    exit 1
                elif [[ "$name" =~ ^[0-9]+$ ]]; then
                    log::error "Invalid scenario name: '$name' (pure numbers are not valid scenario names)"
                    exit 1
                elif [[ ! "$name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
                    log::error "Invalid scenario name: '$name' (only alphanumeric, hyphens, and underscores allowed)"
                    exit 1
                fi
                
                scenarios+=("$name")
                ;;
        esac
        shift
    done
    
    # Validate we have at least one scenario
    if [[ ${#scenarios[@]} -eq 0 ]]; then
        log::error "No scenario names provided"
        scenario_to_app::show_usage
        exit 1
    fi
    
    # Store scenarios globally
    SCENARIO_NAMES=("${scenarios[@]}")
    
    # Log what we're processing
    if [[ ${#SCENARIO_NAMES[@]} -eq 1 ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "Processing single scenario: ${SCENARIO_NAMES[0]}" || true
    else
        [[ "$VERBOSE" == "true" ]] && log::info "Processing ${#SCENARIO_NAMES[@]} scenarios in batch mode: ${SCENARIO_NAMES[*]}" || true
    fi
}

#######################################
# Shared initialization for batch processing
# Performs expensive one-time operations to avoid repeating them per scenario
#######################################
scenario_to_app::shared_initialization() {
    if [[ "$SHARED_UTILS_LOADED" == "true" ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "Shared utilities already loaded, skipping initialization" || true
        return 0
    fi
    
    local init_start
    init_start=$(date +"%s%N")
    
    [[ "$VERBOSE" == "true" ]] && log::info "Initializing shared resources for batch processing..." || true
    
    # Pre-load utilities (avoids sourcing per scenario)
    local secrets_util="${var_ROOT_DIR}/scripts/lib/service/secrets.sh"
    local service_config_util="${var_ROOT_DIR}/scripts/lib/service/service_config.sh"
    
    if [[ -f "$secrets_util" ]]; then
        # shellcheck disable=SC1091
        source "$secrets_util"
        [[ "$VERBOSE" == "true" ]] && log::info "  ✅ Loaded secrets utilities" || true
    fi
    
    if [[ -f "$service_config_util" ]]; then
        # shellcheck disable=SC1090
        source "$service_config_util"
        [[ "$VERBOSE" == "true" ]] && log::info "  ✅ Loaded service config utilities" || true
    fi
    
    # Pre-load port registry (avoids repeated loading)
    if command -v secrets::source_port_registry >/dev/null 2>&1; then
        secrets::source_port_registry
        [[ "$VERBOSE" == "true" ]] && log::info "  ✅ Loaded port registry" || true
    fi
    
    # Pre-load base service.json for batch processing
    local base_service_json="${var_ROOT_DIR}/.vrooli/service.json"
    if [[ -f "$base_service_json" ]]; then
        SHARED_BASE_SERVICE_JSON=$(cat "$base_service_json")
        [[ "$VERBOSE" == "true" ]] && log::info "  ✅ Loaded base service.json" || true
    fi
    
    # Warm up Python for faster JSON validation
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "import json; import sys" 2>/dev/null || true
        [[ "$VERBOSE" == "true" ]] && log::info "  ✅ Warmed up Python interpreter" || true
    fi
    
    SHARED_UTILS_LOADED=true
    
    local init_end
    init_end=$(date +"%s%N")
    local init_time=$(( (init_end - init_start) / 1000000 ))
    
    [[ "$VERBOSE" == "true" ]] && log::info "Shared initialization completed in ${init_time}ms" || true
}

# Validate scenario structure with comprehensive schema validation
scenario_to_app::validate_scenario() {
    local scenario_name="$1"
    local -n scenario_data_ref="$2"  # nameref for returning data
    
    local scenario_path="${var_ROOT_DIR}/scenarios/${scenario_name}"
    
    if [[ ! -d "$scenario_path" ]]; then
        log::error "Scenario directory not found: $scenario_path"
        return 1
    fi
    
    [[ "$VERBOSE" == "true" ]] && log::success "Scenario directory found: $scenario_path" || true
    
    # Check for service.json (first in .vrooli/, then root for backwards compatibility)
    local service_json_path="${scenario_path}/.vrooli/service.json"
    if [[ ! -f "$service_json_path" ]]; then
        # Try root directory for backwards compatibility
        service_json_path="${scenario_path}/service.json"
        if [[ ! -f "$service_json_path" ]]; then
            log::error "service.json not found at: ${scenario_path}/.vrooli/service.json or ${scenario_path}/service.json"
            return 1
        fi
    fi
    
    # Load JSON content
    local service_json
    if ! service_json=$(cat "$service_json_path" 2>/dev/null); then
        log::error "Failed to read service.json"
        return 1
    fi
    
    # Clean up common JSON issues (trailing commas) before validation
    # This uses a more lenient approach to handle real-world JSON files
    # Handles commas before closing braces/brackets, even across lines
    local service_json_cleaned
    service_json_cleaned=$(echo "$service_json" | perl -0pe 's/,(\s*[}\]])/$1/g' 2>/dev/null || echo "$service_json")
    
    # Basic JSON syntax validation
    if ! echo "$service_json_cleaned" | jq empty 2>/dev/null; then
        log::error "Invalid JSON in service.json for scenario: $scenario_name"
        log::error "Common issues to check:"
        log::error "  - Missing quotes around strings"
        log::error "  - Unmatched brackets or braces"
        log::error "  - Invalid escape sequences"
        return 1
    fi
    
    # Use the cleaned JSON for validation
    service_json="$service_json_cleaned"
    
    log::info "Loading robust schema validator..."
    
    # Enhanced validation with timeout and proper error handling
    local validation_success=false
    local validation_error=""
    local resource_count=0
    local service_name=""
    
    # Try Python validation first with robust error handling
    if command -v python3 >/dev/null 2>&1 && [[ -f "$var_SERVICE_JSON_FILE" ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "Attempting Python-based schema validation..." || true
        
        # Write JSON to temp file with proper cleanup trap
        local temp_json_file="/tmp/service_json_validation_${scenario_name}_$$.json"
        
        # Set up cleanup trap that works even if process is killed
        trap "rm -f '$temp_json_file' 2>/dev/null || true" EXIT ERR INT TERM
        
        # Write JSON to temp file with error handling
        if ! echo "$service_json" > "$temp_json_file" 2>/dev/null; then
            log::warning "Failed to write temp validation file, falling back to basic validation"
        else
            # Run Python validation with timeout and proper error handling
            local validation_result=""
            local python_exit_code=0
            
            # Use timeout command to prevent hanging (10 second timeout)
            if command -v timeout >/dev/null 2>&1; then
                validation_result=$(timeout 10 python3 -c "
import json
import sys
import signal

def timeout_handler(signum, frame):
    print('ERROR: Validation timeout - malformed JSON may be causing infinite processing')
    sys.exit(124)

signal.signal(signal.SIGALRM, timeout_handler)
signal.alarm(8)  # Internal 8s timeout as backup

try:
    # Read JSON from temp file to avoid shell escaping issues
    with open('$temp_json_file', 'r') as f:
        content = f.read().strip()
        if not content:
            print('ERROR: Empty service.json file')
            sys.exit(1)
        service_json = json.loads(content)
    
    signal.alarm(0)  # Cancel timeout
    
    # Check required fields with fallbacks for different schema formats
    if 'service' in service_json:
        # New schema format
        service_obj = service_json['service']
        if 'name' not in service_obj:
            print('ERROR: Missing required field: service.name')
            sys.exit(1)
    elif 'name' in service_json:
        # Legacy schema format (direct name field)
        service_obj = service_json
    else:
        print('ERROR: Missing required field: service.name or name')
        sys.exit(1)
        
    # Count enabled resources (handle multiple schema formats)
    resource_count = 0
    if 'resources' in service_json:
        resources = service_json['resources']
        for key, value in resources.items():
            if isinstance(value, dict):
                # Check if it's a flat structure (direct resource config)
                if 'enabled' in value:
                    if value.get('enabled', False):
                        resource_count += 1
                else:
                    # Nested structure: check sub-resources
                    for name, config in value.items():
                        if isinstance(config, dict) and config.get('enabled', False):
                            resource_count += 1
                        elif isinstance(config, dict) and config.get('required', False):
                            resource_count += 1
            elif isinstance(value, bool) and value:
                # Simple boolean: resources.n8n: true
                resource_count += 1
    
    # Extract service name for summary
    service_name = ''
    if 'service' in service_json and 'name' in service_json['service']:
        service_name = service_json['service']['name']
    elif 'name' in service_json:
        service_name = service_json['name']
    
    print(f'VALID:{resource_count}:{service_name}')
    
except json.JSONDecodeError as e:
    print(f'ERROR: Invalid JSON syntax - {str(e)[:100]}...' if len(str(e)) > 100 else f'ERROR: Invalid JSON syntax - {e}')
    sys.exit(1)
except FileNotFoundError:
    print('ERROR: Temporary validation file not found')
    sys.exit(1)
except Exception as e:
    print(f'ERROR: Validation error - {str(e)[:100]}...' if len(str(e)) > 100 else f'ERROR: Validation error - {e}')
    sys.exit(1)
" 2>&1) 
                python_exit_code=$?
            else
                # Fallback without timeout command
                validation_result=$(python3 -c "
import json
import sys

try:
    with open('$temp_json_file', 'r') as f:
        service_json = json.load(f)
    print('VALID:0')
except json.JSONDecodeError as e:
    print(f'ERROR: Invalid JSON - {e}')
    sys.exit(1)
except Exception as e:
    print(f'ERROR: {e}')
    sys.exit(1)
" 2>&1)
                python_exit_code=$?
            fi
            
            # Clean up temp file immediately
            rm -f "$temp_json_file" 2>/dev/null || true
            trap - EXIT ERR INT TERM  # Remove trap
            
            # Process validation results
            if [[ $python_exit_code -eq 124 ]]; then
                log::warning "⚠️  Python validation timed out (likely malformed JSON)"
                validation_error="Python validation timeout"
            elif [[ "$validation_result" =~ ^VALID:([0-9]+):(.*)$ ]]; then
                resource_count="${BASH_REMATCH[1]}"
                service_name="${BASH_REMATCH[2]}"
                validation_success=true
                [[ "$VERBOSE" == "true" ]] && log::success "✅ Python schema validation passed"
            elif [[ "$validation_result" =~ ^ERROR: ]]; then
                validation_error="$validation_result"
                log::warning "⚠️  Python validation failed: ${validation_result#ERROR: }"
            else
                validation_error="Python validation produced unexpected output: $validation_result"
                log::warning "⚠️  $validation_error"
            fi
        fi
    fi
    
    # Fallback to basic validation if Python validation failed or unavailable
    if [[ "$validation_success" != "true" ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "Using fallback basic validation..."
        
        # Enhanced basic validation with better error handling
        local basic_validation_failed=false
        
        # Try to extract service name with error handling
        if service_name=$(echo "$service_json" | jq -r '.service.name // .name // empty' 2>/dev/null) && [[ -n "$service_name" ]]; then
            validation_success=true
            [[ "$VERBOSE" == "true" ]] && log::success "✅ Basic validation passed"
        elif service_name=$(echo "$service_json" | jq -r '.name // empty' 2>/dev/null) && [[ -n "$service_name" ]]; then
            validation_success=true
            [[ "$VERBOSE" == "true" ]] && log::success "✅ Basic validation passed (legacy format)"
        else
            basic_validation_failed=true
            validation_error="Missing required field: service.name or name"
        fi
        
        # If basic validation also failed, provide helpful guidance but don't fail completely in batch mode
        if [[ "$basic_validation_failed" == "true" ]]; then
            log::error "❌ JSON validation failed completely for scenario: $scenario_name"
            log::error "Error: $validation_error"
            log::error "💡 Common fixes:"
            log::error "   • Ensure JSON syntax is valid (use 'jq empty < service.json' to test)"
            log::error "   • Add a 'service.name' field to your .vrooli/service.json file"
            log::error "   • Remove any non-JSON content from the file"
            log::error "   • Check for trailing commas or unmatched brackets"
            
            # In batch processing mode, warn but continue (don't exit)
            # Individual processing will still return failure
            if [[ "${#SCENARIO_NAMES[@]}" -gt 1 ]]; then
                log::warning "Continuing batch processing - this scenario will be skipped"
                return 1
            else
                return 1
            fi
        fi
    fi
    
    # Show validation summary
    if [[ "$validation_success" == "true" ]]; then
        if [[ "$VERBOSE" == "true" ]]; then
            log::info "📋 Validation Summary:"
            if [[ -n "$service_name" ]]; then
                log::info "   • Service name: ✅ Present ($service_name)"
            fi
            log::info "   • JSON syntax: ✅ Valid"
            if [[ $resource_count -gt 0 ]]; then
                log::info "   • Resource configuration: ✅ Valid ($resource_count resources enabled)"
            fi
            log::info "   • Schema compliance: ✅ Passed"
        fi
    else
        log::warning "⚠️  Validation completed with warnings"
        [[ "$VERBOSE" == "true" ]] && [[ -n "$validation_error" ]] && log::info "   Details: $validation_error"
    fi
    
    # Validate initialization file paths
    [[ "$VERBOSE" == "true" ]] && log::info "Validating initialization file paths for $scenario_name..."
    if ! scenario_to_app::validate_initialization_paths "$scenario_path" "$service_json"; then
        return 1
    fi
    
    # Display service information
    if [[ "$VERBOSE" == "true" ]]; then
        local service_name service_display_name
        service_name=$(echo "$service_json" | jq -r '.service.name // empty' 2>/dev/null)
        service_display_name=$(echo "$service_json" | jq -r '.service.displayName // empty' 2>/dev/null)
        
        log::info "Service name: $service_name"
        if [[ -n "$service_display_name" ]]; then
            log::info "Display name: $service_display_name"
        fi
    fi
    
    # Return scenario data via nameref
    scenario_data_ref[name]="$scenario_name"
    scenario_data_ref[path]="$scenario_path"
    scenario_data_ref[service_json]="$service_json"
    scenario_data_ref[service_json_path]="$service_json_path"
    
    return 0
}

# Get all enabled resources from service.json (supports both flat and nested structures)
scenario_to_app::get_enabled_resources() {
    local service_json="$1"
    # Handle both flat (resources.n8n.enabled) and nested (resources.automation.n8n.enabled) structures
    echo "$service_json" | jq -r '
        .resources | to_entries[] as $item |
        if ($item.value | type) == "object" then
            if $item.value.enabled then
                # Flat structure: resources.n8n.enabled = true
                "\($item.key)"
            elif ($item.value | to_entries | length) > 0 then
                # Nested structure: check for sub-resources
                $item.value | to_entries[] |
                select(.value.enabled == true) |
                "\(.key) (\($item.key))"
            else
                empty
            end
        elif ($item.value | type) == "boolean" and $item.value then
            # Simple boolean: resources.n8n = true
            "\($item.key)"
        else
            empty
        end
    ' 2>/dev/null || true
}

# Helper function to safely normalize paths without infinite loops
scenario_to_app::normalize_path() {
    local path="$1"
    
    # First try realpath if available
    if command -v realpath >/dev/null 2>&1; then
        realpath -m "$path" 2>/dev/null || echo "$path"
        return
    fi
    
    # Fallback: manual normalization with safety checks
    local dir="${path%/*}"
    local file="${path##*/}"
    
    # Safety check: limit path resolution attempts
    local attempts=0
    local max_attempts=10
    
    while [[ $attempts -lt $max_attempts ]]; do
        ((attempts++))
        
        # Check if directory exists or can be resolved
        if [[ -d "$dir" ]]; then
            # Directory exists, get absolute path
            local abs_dir
            abs_dir=$(cd "$dir" 2>/dev/null && pwd)
            if [[ -n "$abs_dir" ]]; then
                echo "${abs_dir}/${file}"
                return
            fi
        fi
        
        # If directory doesn't exist, try to resolve parent
        if [[ "$dir" == *".."* ]]; then
            # Simplify the path by resolving one level
            dir=$(echo "$dir" | sed 's|/[^/]*/\.\./|/|g')
        else
            # Can't resolve further, return original
            echo "$path"
            return
        fi
    done
    
    # Max attempts reached, return original path
    echo "$path"
}

# Get resource categories from service.json
scenario_to_app::get_resource_categories() {
    local service_json="$1"
    # For flat structure, resource names ARE the categories
    # For nested structure, extract category names
    echo "$service_json" | jq -r '
        .resources | to_entries[] as $item |
        if ($item.value | type) == "object" and ($item.value.enabled | type) != "boolean" then
            # This is a category (nested structure)
            $item.key
        else
            # This is a direct resource (flat structure) - skip it
            empty
        end
    ' 2>/dev/null || true
}

# Validate that initialization file paths in service.json actually exist
scenario_to_app::validate_initialization_paths() {
    local scenario_path="$1"
    local service_json="$2"
    
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
    local service_json_dir="${scenario_path}/.vrooli"
    
    # Use process substitution instead of pipe to avoid subshell issues
    while IFS= read -r file_path; do
        [[ -z "$file_path" ]] && continue
        
        total_files=$((total_files + 1))
        
        # Safety check: limit iterations
        if [[ $total_files -gt 1000 ]]; then
            log::error "Too many initialization files (>1000), possible configuration error"
            return 1
        fi
        
        # Resolve path based on type
        local full_path
        if [[ "$file_path" == /* ]]; then
            # Absolute path
            full_path="$file_path"
        elif [[ "$file_path" == ../* ]] || [[ "$file_path" == ./* ]]; then
            # Relative path from service.json location (.vrooli/ directory)
            full_path="${service_json_dir}/${file_path}"
            # Safely normalize the path
            full_path=$(scenario_to_app::normalize_path "$full_path")
        else
            # Simple path - relative to scenario root directory
            full_path="${scenario_path}/${file_path}"
        fi
        
        if [[ ! -f "$full_path" ]]; then
            log::error "Referenced file not found: $file_path"
            log::error "  Expected at: $full_path"
            validation_errors=$((validation_errors + 1))
        elif [[ "$VERBOSE" == "true" ]]; then
            log::info "✅ Found: $file_path"
        fi
    done < <(echo "$service_json" | jq -r '
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
    ' 2>/dev/null)
    
    # Check deployment initialization files
    # Use process substitution to avoid subshell variable scope issues
    while IFS= read -r file_path; do
        [[ -z "$file_path" ]] && continue
        
        total_files=$((total_files + 1))
        
        # Safety check: limit iterations
        if [[ $total_files -gt 1000 ]]; then
            log::error "Too many initialization files (>1000), possible configuration error"
            return 1
        fi
        
        # Resolve path based on type (same logic as above)
        local full_path
        if [[ "$file_path" == /* ]]; then
            # Absolute path
            full_path="$file_path"
        elif [[ "$file_path" == ../* ]] || [[ "$file_path" == ./* ]]; then
            # Relative path from service.json location (.vrooli/ directory)
            full_path="${service_json_dir}/${file_path}"
            # Safely normalize the path
            full_path=$(scenario_to_app::normalize_path "$full_path")
        else
            # Simple path - relative to scenario root directory
            full_path="${scenario_path}/${file_path}"
        fi
        
        if [[ ! -f "$full_path" ]]; then
            log::error "Deployment file not found: $file_path"
            log::error "  Expected at: $full_path"
            validation_errors=$((validation_errors + 1))
        elif [[ "$VERBOSE" == "true" ]]; then
            log::info "✅ Found: $file_path"
        fi
    done < <(echo "$service_json" | jq -r '
        .deployment.initialization.phases[]?.tasks[]? |
        select(.type == "sql" or .type == "config") |
        .file // empty | select(. != "")
    ' 2>/dev/null)
    
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
    app_name="$(basename "${dest%/*%/*}")"
    
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

# Process template variables in files
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
    
    # Process template variables (no inheritance checks needed)
    if command -v secrets::substitute_safe_templates >/dev/null 2>&1; then
        # Read file content and substitute safe templates (only Vrooli secrets and service references)
        local content
        content=$(cat "$file")
        
        # Check if content has any Vrooli templates before processing (skip n8n patterns)
        if echo "$content" | grep -qE '\{\{[A-Z][A-Z0-9_]*\}\}|\$\{service\.[^}]+\}'; then
            local sub_start
            sub_start=$(date +"%s%N")
            local processed
            processed=$(echo "$content" | secrets::substitute_safe_templates)
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
# Adjust APP_ROOT depth for scenario-to-app conversion
# Converts 3-level scenario paths (../../..) to 1-level app paths (/..)
# Arguments:
#   $1 - file path
# Returns:
#   0 on success, 1 on failure
#######################################
scenario_to_app::adjust_app_root_depth() {
    local file="$1"
    
    # Skip if file doesn't exist
    [[ ! -f "$file" ]] && return 0
    
    # Only process shell script files (.sh, .bash, .bats)
    if [[ ! "$file" =~ \.(sh|bash|bats)$ ]]; then
        # Also check shebang for extensionless scripts
        if ! head -1 "$file" 2>/dev/null | grep -qE "^#!/.*/(bash|sh)$"; then
            return 0
        fi
    fi
    
    # Read file content
    local content
    content=$(cat "$file")
    local original_content="$content"
    local modified=false
    
    # Extract scenario name if present in paths
    local scenario_name=""
    if echo "$content" | grep -q "/scenarios/[^/]*/"; then
        scenario_name=$(echo "$content" | grep -o "/scenarios/[^/]*/" | head -1 | sed 's|/scenarios/||g' | sed 's|/||g')
    fi
    
    # Create a temporary file for processing
    local temp_file=$(mktemp)
    
    # Process the file line by line
    while IFS= read -r line; do
        local processed_line="$line"
        
        # Check if this line is setting APP_ROOT
        if echo "$line" | grep -qE "^[[:space:]]*APP_ROOT="; then
            
            # Pattern 1: ${BASH_SOURCE[0]%/*} followed by relative path
            if echo "$line" | grep -q '\${BASH_SOURCE\[0\]%/\*}'; then
                # Count the ../ segments
                local depth=$(echo "$line" | grep -o '\.\./\.\.' | wc -l)
                depth=$((depth * 2))  # Each ../.. is 2 levels
                local single_dots=$(echo "$line" | sed 's|\.\./\.\.||g' | grep -o '\.\.' | wc -l)
                depth=$((depth + single_dots))
                
                if [[ $depth -ge 2 ]]; then
                    # Reduce by 2 levels
                    local new_depth=$((depth - 2))
                    if [[ $new_depth -eq 0 ]]; then
                        # Remove all the /.. parts
                        processed_line=$(echo "$line" | sed 's|\(\${BASH_SOURCE\[0\]%/\*}\)/\.*|\1|')
                    else
                        # Build new path
                        local new_path=""
                        for ((i=0; i<new_depth; i++)); do
                            new_path="${new_path}/.."
                        done
                        processed_line=$(echo "$line" | sed "s|\\(\${BASH_SOURCE\\[0\\]%/\\*}\\)/[\\./]*|\\1${new_path}|")
                    fi
                fi
            fi
            
            # Pattern 2: builtin cd or just cd with paths
            if echo "$line" | grep -qE "(builtin )?cd "; then
                # Count depth in the cd path
                local depth=0
                if echo "$line" | grep -q 'cd.*\.\.'; then
                    # Count ../.. patterns
                    local double_dots=$(echo "$line" | sed -n 's/.*cd[^"]*"\([^"]*\)".*/\1/p' | grep -o '\.\./\.\.' | wc -l)
                    depth=$((double_dots * 2))
                    # Count remaining single ..
                    local temp_path=$(echo "$line" | sed -n 's/.*cd[^"]*"\([^"]*\)".*/\1/p' | sed 's|\.\./\.\.||g')
                    local single_dots=$(echo "$temp_path" | grep -o '\.\.' | wc -l)
                    depth=$((depth + single_dots))
                    
                    if [[ $depth -ge 2 ]]; then
                        local new_depth=$((depth - 2))
                        if [[ $new_depth -eq 0 ]]; then
                            # Replace with current dir
                            processed_line=$(echo "$line" | sed 's|\(cd[^"]*"\)[^"]*\("\)|\1.\2|')
                        else
                            # Build new path
                            local new_path=""
                            for ((i=0; i<new_depth; i++)); do
                                [[ $i -gt 0 ]] && new_path="${new_path}/"
                                new_path="${new_path}.."
                            done
                            processed_line=$(echo "$line" | sed "s|\\(cd[^\"]*\"\\)[^\"]*\\(\"\\)|\\1${new_path}\\2|")
                        fi
                    elif [[ $depth -eq 1 ]]; then
                        # Single .. becomes current dir
                        processed_line=$(echo "$line" | sed 's|\(cd[^"]*"\)[^"]*\("\)|\1.\2|')
                    fi
                fi
            fi
            
            # Pattern 3: Simple APP_ROOT="../.." style assignments  
            if echo "$line" | grep -qE "APP_ROOT=[\"']?\\.\\./"; then
                local depth=$(echo "$line" | sed -n 's/.*APP_ROOT[^.]*\(\.\.[/.]*\).*/\1/' | grep -o '\.\.' | wc -l)
                if [[ $depth -ge 2 ]]; then
                    local new_depth=$((depth - 2))
                    if [[ $new_depth -eq 0 ]]; then
                        processed_line=$(echo "$line" | sed 's|\(APP_ROOT[^.]*\)[^"'\'' ]*|\1.|')
                    else
                        local new_path=""
                        for ((i=0; i<new_depth; i++)); do
                            [[ $i -gt 0 ]] && new_path="${new_path}/"
                            new_path="${new_path}.."
                        done
                        processed_line=$(echo "$line" | sed "s|\\(APP_ROOT[^.]*\\)[^\"' ]*|\\1${new_path}|")
                    fi
                elif [[ $depth -eq 1 ]]; then
                    processed_line=$(echo "$line" | sed 's|\(APP_ROOT[^.]*\)[^"'\'' ]*|\1.|')
                fi
            fi
        fi
        
        # Pattern 4: Remove /scenarios/<scenario-name>/ from paths
        if [[ -n "$scenario_name" ]]; then
            # Remove /scenarios/<name>/ (with trailing slash)
            processed_line=$(echo "$processed_line" | sed "s|/scenarios/${scenario_name}/|/|g")
            # Remove /scenarios/<name> at end of line or before quote
            processed_line=$(echo "$processed_line" | sed "s|/scenarios/${scenario_name}\\(\$\\|[\"']\\)|\\1|g")
        fi
        
        echo "$processed_line" >> "$temp_file"
    done <<< "$content"
    
    # Read the processed content
    local new_content=$(cat "$temp_file")
    rm -f "$temp_file"
    
    # Check if content was modified
    if [[ "$new_content" != "$original_content" ]]; then
        modified=true
        echo "$new_content" > "$file"
        [[ "$VERBOSE" == "true" ]] && log::info "Adjusted APP_ROOT paths in: $(basename "$file")"
    fi
    
    return 0
}

#######################################
# Copy files according to manifest
# Arguments:
#   $1 - app_path (destination)
#   $2 - scenario_path  
#   $3 - service_json (for processing)
# Returns:
#   0 on success, 1 on failure
#######################################
scenario_to_app::copy_from_manifest() {
    local app_path="$1"
    local scenario_path="$2"
    local service_json="$3"
    local filter="${4:-}"  # Optional filter: "vrooli" or "scenario"
    local manifest="${SCENARIO_TOOLS_DIR}/app-structure.json"
    
    if [[ ! -f "$manifest" ]]; then
        log::error "Copy manifest not found: $manifest"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would copy files according to manifest to: $app_path"
        return 0
    fi
    
    # Adjust message based on filter
    local copy_msg="Copying files according to manifest"
    [[ -n "$filter" ]] && copy_msg="Copying $filter files"
    log::info "${copy_msg}..."
    
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
        
        # Apply filter if specified
        if [[ -n "$filter" ]] && [[ "$from" != "$filter" ]]; then
            [[ "$VERBOSE" == "true" ]] && log::info "Skipping $name (from=$from, filter=$filter)"
            continue
        fi
                
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
                    scenario_to_app::process_file "$dest_path" "$processor" "$service_json"
                fi
            done <<< "$process_list"
        fi
    done
    
    # Create instance manager wrapper for standalone app 
    scenario_to_app::create_instance_manager_wrapper "$app_path"
    
    # Phase 2: Bulk copy remaining scenario files (if enabled and not filtered)
    if [[ -z "$filter" ]] || [[ "$filter" == "scenario" ]]; then
        scenario_to_app::bulk_copy_remaining_files "$app_path" "$scenario_path" "$manifest"
    fi
    
    log::success "File copying completed successfully"
    return 0
}

#######################################
# Bulk copy remaining scenario files not handled by manifest
# Arguments:
#   $1 - app_path (destination)
#   $2 - scenario_path (source)
#   $3 - manifest_path (for configuration)
# Returns:
#   0 on success, 1 on failure
#######################################
scenario_to_app::bulk_copy_remaining_files() {
    local app_path="$1"
    local scenario_path="$2" 
    local manifest_path="$3"
    
    # Check if bulk copy is enabled in manifest
    local bulk_enabled
    bulk_enabled=$(jq -r '.bulk_copy.enabled // false' "$manifest_path" 2>/dev/null)
    
    if [[ "$bulk_enabled" != "true" ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "Bulk copy disabled in manifest, skipping"
        return 0
    fi
    
    [[ "$VERBOSE" == "true" ]] && log::info "Starting bulk copy of remaining scenario files..."
    
    # Build set of already copied paths from manifest rules
    declare -A copied_paths
    scenario_to_app::build_copied_paths_set copied_paths "$scenario_path" "$manifest_path"
    
    # Get exclusion patterns from manifest
    local exclude_patterns
    mapfile -t exclude_patterns < <(jq -r '.bulk_copy.exclude_patterns[]? // empty' "$manifest_path" 2>/dev/null)
    
    # Find and copy remaining files
    local files_copied=0
    local files_skipped=0
    
    # Use find to get all files, then filter
    while IFS= read -r -d '' file; do
        local rel_path="${file#$scenario_path/}"
        
        # Skip if already copied by manifest
        if [[ "${copied_paths[$rel_path]:-}" == "1" ]]; then
            ((files_skipped++))
            [[ "$VERBOSE" == "true" ]] && log::info "  Skipping (already copied): $rel_path"
            continue
        fi
        
        # Check exclusion patterns
        local excluded=false
        for pattern in "${exclude_patterns[@]}"; do
            if [[ -n "$pattern" ]]; then
                # Convert glob pattern to match against relative path
                if [[ "$rel_path" == $pattern ]] || [[ "$rel_path" == */$pattern ]] || [[ "$rel_path" == $pattern/* ]]; then
                    excluded=true
                    break
                fi
            fi
        done
        
        if [[ "$excluded" == "true" ]]; then
            ((files_skipped++))
            [[ "$VERBOSE" == "true" ]] && log::info "  Skipping (excluded): $rel_path"
            continue
        fi
        
        # Copy the file
        local dest_file="$app_path/$rel_path"
        if scenario_to_app::copy_bulk_file "$file" "$dest_file" "$rel_path"; then
            ((files_copied++))
        else
            log::warning "Failed to copy: $rel_path"
        fi
        
    done < <(find "$scenario_path" -type f -print0 2>/dev/null)
    
    if [[ $files_copied -gt 0 ]]; then
        log::success "Bulk copied $files_copied additional file(s) from scenario (skipped $files_skipped)"
    else
        [[ "$VERBOSE" == "true" ]] && log::info "No additional files to bulk copy (skipped $files_skipped already handled)"
    fi
    
    return 0
}

#######################################
# Build set of paths already copied by manifest rules
# Arguments:
#   $1 - nameref to associative array to populate
#   $2 - scenario_path
#   $3 - manifest_path
#######################################
scenario_to_app::build_copied_paths_set() {
    local -n paths_ref=$1
    local scenario_path="$2"
    local manifest_path="$3"
    
    # Get all copy rules from manifest
    local rule_count
    rule_count=$(jq '.copy_rules | length' "$manifest_path")
    
    for ((i=0; i<rule_count; i++)); do
        local rule
        rule=$(jq -c ".copy_rules[$i]" "$manifest_path")
        
        local from source type
        from=$(echo "$rule" | jq -r '.from // "vrooli"')
        source=$(echo "$rule" | jq -r '.source')
        type=$(echo "$rule" | jq -r '.type')
        
        # Only track scenario files (skip vrooli files)
        if [[ "$from" == "scenario" ]]; then
            if [[ "$type" == "file" ]]; then
                # Single file
                paths_ref["$source"]=1
            elif [[ "$type" == "directory" ]]; then
                # Directory - mark all files within it
                if [[ -d "$scenario_path/$source" ]]; then
                    while IFS= read -r -d '' file; do
                        local rel_path="${file#$scenario_path/}"
                        paths_ref["$rel_path"]=1
                    done < <(find "$scenario_path/$source" -type f -print0 2>/dev/null)
                fi
            fi
        fi
    done
    
    [[ "$VERBOSE" == "true" ]] && log::info "Tracking ${#paths_ref[@]} paths already copied by manifest"
}

#######################################
# Copy a single file during bulk copy
# Arguments:
#   $1 - source file path
#   $2 - destination file path  
#   $3 - relative path (for logging)
# Returns:
#   0 on success, 1 on failure
#######################################
scenario_to_app::copy_bulk_file() {
    local src="$1"
    local dest="$2"
    local rel_path="$3"
    
    # Create parent directory
    mkdir -p "${dest%/*}" || {
        log::error "Failed to create directory for: $rel_path"
        return 1
    }
    
    # Copy file preserving permissions and timestamps
    if cp -p "$src" "$dest" 2>/dev/null; then
        [[ "$VERBOSE" == "true" ]] && log::info "  Copied: $rel_path"
        return 0
    else
        log::warning "Failed to copy file: $rel_path"
        return 1
    fi
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
    mkdir -p "${dest%/*}"
    
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
                    # Use tar for efficient bulk copying (avoids spawning thousands of processes)
                    (cd "$src" && tar cf - --exclude='*.log' --exclude='data/' . 2>/dev/null | (cd "$dest" && tar xf - 2>/dev/null)) || {
                        [[ "$VERBOSE" == "true" ]] && log::info "Some files in $name could not be copied, continuing..."
                    }
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
    # Use process substitution to avoid subshell issues
    while IFS= read -r file; do
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
    done < <(find "$init_dir" -type f 2>/dev/null)
    
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
#   $3 - service_json (for processing)
# Returns:
#   0 on success, 1 on failure
#######################################
scenario_to_app::process_file() {
    local file_path="$1" processor="$2" service_json="$3"
    
    case "$processor" in
        substitute_variables)
            if [[ -f "$file_path/service.json" ]]; then
                # Note: The service.json here is the app's service.json, not the original scenario one
                # So we pass the loaded service.json from that file, not the original scenario service.json
                local app_service_json
                app_service_json=$(cat "$file_path/service.json" 2>/dev/null || echo '{}')
                scenario_to_app::process_template_variables "$file_path/service.json" "$app_service_json"
            fi
            ;;
        filter_by_service_references)
            if [[ -d "$file_path" ]]; then
                # For this processor, we use the original scenario service.json
                # to understand which shared files should be kept
                scenario_to_app::filter_shared_initialization "$file_path" "$service_json"
            fi
            ;;
        adjust_app_root_depth)
            # Adjust APP_ROOT depth for scenario-to-app conversion
            # Convert 3-level scenario paths to 1-level app paths
            if [[ -d "$file_path" ]]; then
                # Process all shell scripts in the directory
                # Use process substitution to avoid subshell issues
                while IFS= read -r shell_file; do
                    scenario_to_app::adjust_app_root_depth "$shell_file"
                done < <(find "$file_path" -type f \( -name "*.sh" -o -name "*.bash" \) 2>/dev/null)
            elif [[ -f "$file_path" ]]; then
                # Process single file
                scenario_to_app::adjust_app_root_depth "$file_path"
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
    mkdir -p "${app_instance_manager%/*}"
    
    # Create a wrapper that properly integrates with the standalone app structure
    cat > "$app_instance_manager" << 'EOF'
#!/usr/bin/env bash
# Standalone App Instance Manager - Generic Integration Wrapper
# 
# This wrapper sources the generic instance manager for standalone apps.
# The generic instance manager reads configuration from service.json automatically.
set -euo pipefail

APP_ROOT="\${APP_ROOT:-\$(builtin cd \"\${BASH_SOURCE[0]%/*}/../../..\" && builtin pwd)}"
APP_LIFECYCLE_DEVELOP_DIR="\${APP_ROOT}/scripts/scenarios/tools"

# shellcheck disable=SC1091
source "\${APP_ROOT}/scripts/lib/utils/var.sh"

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
    local scenario_path="$2"
    local service_json="$3"
    
    [[ "$VERBOSE" == "true" ]] && log::info "Generating standalone app for: $scenario_name"
    
    local app_path="$HOME/generated-apps/$scenario_name"
    local temp_path="$HOME/generated-apps/.tmp-$scenario_name-$$"
    
    # Use atomic operations with temp directory
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would create directory: $app_path"
        scenario_to_app::copy_from_manifest "/dev/null" "$scenario_path" "$service_json"
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
        
        # Copy all files according to manifest
        if ! scenario_to_app::copy_from_manifest "$temp_path" "$scenario_path" "$service_json"; then
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
                    scenario_to_app::process_template_variables "$file" "$service_json"
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
                # Simple hash calculation fallback if function not available
                if command -v calculate_hash >/dev/null 2>&1; then
                    scenario_hash=$(calculate_hash "$scenario_path" 2>/dev/null || echo "unknown")
                else
                    # Fallback: use find and md5sum/sha256sum to generate a hash
                    if command -v sha256sum >/dev/null 2>&1; then
                        scenario_hash=$(find "$scenario_path" -type f -exec sha256sum {} \; 2>/dev/null | sha256sum | cut -d' ' -f1 | head -c 12)
                    elif command -v md5sum >/dev/null 2>&1; then
                        scenario_hash=$(find "$scenario_path" -type f -exec md5sum {} \; 2>/dev/null | md5sum | cut -d' ' -f1 | head -c 12)
                    else
                        scenario_hash="unknown"
                    fi
                fi
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
# Single Scenario Processing
################################################################################

# Process a single scenario with isolated state
scenario_to_app::process_single_scenario() {
    local scenario_name="$1"
    
    # Declare associative array for scenario data
    declare -A scenario_data
    
    # Phase 1: Validation
    [[ "$VERBOSE" == "true" ]] && log::info "[${scenario_name}] Phase 1: Scenario Validation"
    if ! scenario_to_app::validate_scenario "$scenario_name" scenario_data; then
        log::error "[${scenario_name}] Scenario validation failed"
        return 1
    fi
    
    # Phase 2: Resource Analysis
    [[ "$VERBOSE" == "true" ]] && log::info "[${scenario_name}] Phase 2: Resource Analysis"
    
    # Get resource categories
    local resource_categories
    mapfile -t resource_categories < <(scenario_to_app::get_resource_categories "${scenario_data[service_json]}")
    
    if [[ ${#resource_categories[@]} -gt 0 ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "[${scenario_name}] Resource categories: ${resource_categories[*]}"
    fi
    
    # Get enabled resources
    local enabled_resources
    mapfile -t enabled_resources < <(scenario_to_app::get_enabled_resources "${scenario_data[service_json]}")
    
    if [[ ${#enabled_resources[@]} -eq 0 ]]; then
        [[ "$VERBOSE" == "true" ]] && log::warning "[${scenario_name}] No enabled resources found in service.json"
    else
        [[ "$VERBOSE" == "true" ]] && log::info "[${scenario_name}] Enabled resources: ${#enabled_resources[@]} resources"
        if [[ "$VERBOSE" == "true" ]]; then
            for resource in "${enabled_resources[@]}"; do
                log::info "[${scenario_name}]   • $resource"
            done
        fi
    fi
    
    # Phase 3: App Generation
    [[ "$VERBOSE" == "true" ]] && log::info "[${scenario_name}] Phase 3: App Generation"
    if ! scenario_to_app::generate_app "$scenario_name" "${scenario_data[path]}" "${scenario_data[service_json]}"; then
        log::error "[${scenario_name}] App generation failed"
        return 1
    fi
    
    # Phase 4: App Startup (optional)
    if [[ "$START" == "true" ]]; then
        [[ "$VERBOSE" == "true" ]] && log::info "[${scenario_name}] Phase 4: App Startup"
        if ! scenario_to_app::start_generated_app "$scenario_name"; then
            log::error "[${scenario_name}] App startup failed"
            return 1
        fi
    fi
    
    return 0
}

# Batch processing for multiple scenarios
scenario_to_app::process_batch() {
    local scenarios=("$@")
    local total=${#scenarios[@]}
    local failed=0
    local succeeded=0
    
    log::info "Processing ${total} scenario(s) in batch mode..."
    
    # Shared initialization (done once for all scenarios)
    scenario_to_app::shared_initialization
    
    # Process each scenario
    for i in "${!scenarios[@]}"; do
        local scenario_name="${scenarios[i]}"
        local progress="$((i+1))/$total"
        
        if [[ $total -gt 1 ]]; then
            log::info "[$progress] Processing: $scenario_name"
        else
            log::info "Processing: $scenario_name"
        fi
        
        # Process single scenario with isolated state
        if scenario_to_app::process_single_scenario "$scenario_name"; then
            ((succeeded++))
            if [[ $total -gt 1 ]]; then
                log::success "[$progress] ✅ $scenario_name completed"
            else
                log::success "✅ $scenario_name completed"
            fi
        else
            ((failed++))
            if [[ $total -gt 1 ]]; then
                log::error "[$progress] ❌ $scenario_name failed"
                # Continue with other scenarios unless critical failure
                if [[ "$DRY_RUN" != "true" ]]; then
                    log::info "Continuing with remaining scenarios..."
                fi
            else
                log::error "❌ $scenario_name failed"
            fi
        fi
    done
    
    # Batch summary
    if [[ $total -gt 1 ]]; then
        echo ""
        log::info "Batch processing complete:"
        log::info "  ✅ Succeeded: $succeeded"
        log::info "  ❌ Failed: $failed"
        log::info "  📊 Success rate: $((succeeded * 100 / total))%"
    fi
    
    # Return proper exit code: 0 for success, 1 for any failures
    if [[ $failed -gt 0 ]]; then
        return 1
    else
        return 0
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
    if [[ ${#SCENARIO_NAMES[@]} -eq 1 ]]; then
        log::info "Scenario: ${SCENARIO_NAMES[0]}"
    else
        log::info "Scenarios: ${SCENARIO_NAMES[*]} (${#SCENARIO_NAMES[@]} total)"
    fi
    if [[ "$DRY_RUN" == "true" ]]; then
        log::warning "DRY RUN MODE - No actual changes will be made"
    fi
    if [[ "$FORCE" == "true" ]]; then
        log::info "FORCE MODE - Will overwrite existing apps"
    fi
    if [[ "$START" == "true" ]]; then
        log::info "START MODE - Will start apps after generation"
    fi
    echo ""
    
    # Process scenarios using batch processing
    local exit_code=0
    if ! scenario_to_app::process_batch "${SCENARIO_NAMES[@]}"; then
        exit_code=$?
    fi
    
    # Final summary
    if [[ ${#SCENARIO_NAMES[@]} -eq 1 ]]; then
        local scenario_name="${SCENARIO_NAMES[0]}"
        if [[ $exit_code -eq 0 ]]; then
            log::success "✅ $scenario_name app generated successfully!"
            echo ""
            log::info "Generated app location: $HOME/generated-apps/$scenario_name"
            
            if [[ "$START" != "true" ]]; then
                log::info "To run the app:"
                log::info "  cd $HOME/generated-apps/$scenario_name"
                log::info "  ./scripts/manage.sh develop"
            else
                log::info "App has been started and should be available at the configured URLs"
            fi
        fi
    else
        if [[ $exit_code -eq 0 ]]; then
            log::success "✅ All ${#SCENARIO_NAMES[@]} scenarios processed successfully!"
        else
            log::error "❌ Some scenarios failed during batch processing"
        fi
        echo ""
        log::info "Generated apps location: $HOME/generated-apps/"
        
        if [[ "$START" != "true" ]]; then
            log::info "To run any app:"
            log::info "  cd $HOME/generated-apps/<scenario-name>"
            log::info "  ./scripts/manage.sh develop"
        else
            log::info "Started apps should be available at their configured URLs"
        fi
    fi
    echo ""
    
    return $exit_code
}

# Execute main function
main "$@"
exit $?