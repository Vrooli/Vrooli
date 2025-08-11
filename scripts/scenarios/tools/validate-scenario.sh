#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Scenario Static Validator
# 
# Comprehensive static analysis for Vrooli scenarios.
# Validates structure, schema compliance, and file paths without generation.
#
# Usage:
#   ./validate-scenario.sh <scenario-name> [options]
#   ./validate-scenario.sh agent-metareasoning-manager --verbose
#
# This validator checks:
# - service.json location (.vrooli/ directory)
# - JSON syntax (no trailing commas)
# - Schema compliance with .vrooli/schemas/service.schema.json
# - File path validation (scenario + framework paths)
# - Resource configuration validity
#
################################################################################

TOOLS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${TOOLS_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Global variables (namespaced for sourcing)
VALIDATE_SCENARIO_NAME=""
VALIDATE_SCENARIO_PATH=""
VALIDATE_SERVICE_JSON=""
VALIDATE_VERBOSE=false
VALIDATE_ERRORS=0
VALIDATE_WARNINGS=0

# Color codes (not readonly since script can be sourced)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Usage information
show_usage() {
    cat << EOF
Usage: $0 <scenario-name> [options]

Perform comprehensive static validation of a Vrooli scenario.

Arguments:
  scenario-name     Name of the scenario to validate

Options:
  --verbose         Show detailed validation output
  --help            Show this help message

Validation Checks:
  ✓ service.json location and syntax
  ✓ JSON schema compliance
  ✓ File path resolution
  ✓ Resource configuration
  ✓ Lifecycle command validity

Examples:
  $0 agent-metareasoning-manager
  $0 research-assistant --verbose

Exit Codes:
  0 - All validations passed
  1 - Critical errors found
  2 - Warnings only (non-critical issues)
EOF
}

# Parse arguments
parse_args() {
    if [[ $# -eq 0 ]]; then
        echo -e "${RED}Error: No scenario name provided${NC}"
        show_usage
        exit 1
    fi

    VALIDATE_SCENARIO_NAME="$1"
    shift

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose)
                VALIDATE_VERBOSE=true
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                show_usage
                exit 1
                ;;
        esac
        shift
    done
}

# Print functions
print_header() {
    echo
    echo "════════════════════════════════════════════════════════"
    echo "    Scenario Static Validator"
    echo "════════════════════════════════════════════════════════"
    echo
}

print_check() {
    echo -e "${BLUE}[CHECK]${NC} $1"
}

print_success() {
    echo -e "${GREEN}  ✓${NC} $1"
}

print_error() {
    echo -e "${RED}  ✗${NC} $1" >&2
    VALIDATE_ERRORS=$((VALIDATE_ERRORS + 1))
}

print_warning() {
    echo -e "${YELLOW}  ⚠${NC} $1"
    VALIDATE_WARNINGS=$((VALIDATE_WARNINGS + 1))
}

print_info() {
    if [[ "$VALIDATE_VERBOSE" == "true" ]]; then
        echo -e "    ℹ $1"
    fi
}

# Validate scenario exists
validate_scenario_exists() {
    print_check "Checking scenario directory..."
    
    VALIDATE_SCENARIO_PATH="${var_SCRIPTS_SCENARIOS_DIR}/core/${VALIDATE_SCENARIO_NAME}"
    
    if [[ ! -d "$VALIDATE_SCENARIO_PATH" ]]; then
        print_error "Scenario directory not found: $VALIDATE_SCENARIO_PATH"
        return 1
    fi
    
    print_success "Scenario directory found"
    print_info "Path: $VALIDATE_SCENARIO_PATH"
    return 0
}

# Validate service.json location and syntax
validate_service_json() {
    print_check "Validating service.json..."
    
    local service_json_path="${VALIDATE_SCENARIO_PATH}/.vrooli/service.json"
    
    # Check location
    if [[ ! -f "$service_json_path" ]]; then
        # Check legacy location
        if [[ -f "${VALIDATE_SCENARIO_PATH}/service.json" ]]; then
            print_warning "service.json in legacy location (should be in .vrooli/)"
            service_json_path="${VALIDATE_SCENARIO_PATH}/service.json"
        else
            print_error "service.json not found in .vrooli/ directory"
            return 1
        fi
    else
        print_success "service.json in correct location (.vrooli/)"
    fi
    
    # Read file
    if ! VALIDATE_SERVICE_JSON=$(cat "$service_json_path" 2>/dev/null); then
        print_error "Failed to read service.json"
        return 1
    fi
    
    # Check for trailing commas (common issue)
    if echo "$VALIDATE_SERVICE_JSON" | grep -E ',\s*[}\]]' > /dev/null 2>&1; then
        print_warning "Trailing commas detected (will be auto-fixed)"
        VALIDATE_SERVICE_JSON=$(echo "$VALIDATE_SERVICE_JSON" | perl -0pe 's/,(\s*[}\]])/$1/g')
    fi
    
    # Validate JSON syntax
    if ! echo "$VALIDATE_SERVICE_JSON" | timeout 5s jq empty 2>/dev/null; then
        print_error "Invalid JSON syntax in service.json"
        print_info "Use 'jq .' to debug JSON issues"
        return 1
    fi
    
    print_success "JSON syntax is valid"
    return 0
}

# Validate against schema
validate_schema_compliance() {
    print_check "Validating schema compliance..."
    
    local schema_path="${var_ROOT_DIR}/.vrooli/schemas/service.schema.json"
    
    if [[ ! -f "$schema_path" ]]; then
        print_warning "Schema file not found: $schema_path"
        print_info "Skipping schema validation"
        return 0
    fi
    
    # Try Python validation first
    if command -v python3 >/dev/null 2>&1; then
        local validation_script="/tmp/validate_schema_$$.py"
        cat > "$validation_script" << 'EOF'
import json
import sys
import os

def validate_basic_schema(service_json):
    errors = []
    warnings = []
    scenario_path = os.environ.get('VALIDATE_SCENARIO_PATH')
    
    # Check required top-level fields
    if 'version' not in service_json:
        errors.append("Missing required field: version")
    if 'service' not in service_json:
        errors.append("Missing required field: service")
    else:
        # Check required service fields
        service = service_json['service']
        if 'name' not in service:
            errors.append("Missing required field: service.name")
        elif not service['name'].replace('-', '').replace('_', '').isalnum():
            warnings.append(f"service.name '{service['name']}' should follow pattern: ^[a-z0-9][a-z0-9-]*[a-z0-9]$")
        
        if 'version' not in service:
            errors.append("Missing required field: service.version")
    
    # Check ports configuration
    if 'ports' in service_json:
        for port_name, port_config in service_json['ports'].items():
            if 'env_var' not in port_config:
                errors.append(f"Port '{port_name}' missing required field: env_var")
            if 'range' not in port_config and 'fixed' not in port_config:
                errors.append(f"Port '{port_name}' must have either 'range' or 'fixed'")
    
    # Check resources
    if 'resources' in service_json:
        resource_count = 0
        for category, resources in service_json['resources'].items():
            for name, config in resources.items():
                if config.get('enabled', False):
                    resource_count += 1
                    if config.get('required', False):
                        # Check for initialization in config or actual files
                        if 'initialization' not in config and category in ['automation', 'storage']:
                            # Check if initialization directory exists with files
                            if scenario_path:
                                init_dir = os.path.join(scenario_path, 'initialization', category, name)
                                if os.path.exists(init_dir) and os.listdir(init_dir):
                                    warnings.append(f"Resource '{name}' has initialization files but no 'initialization' field in service.json")
                                else:
                                    warnings.append(f"Resource '{name}' is required but has no initialization")
                            else:
                                warnings.append(f"Resource '{name}' is required but has no initialization field")
        
        if resource_count == 0:
            warnings.append("No resources are enabled")
    
    return errors, warnings

try:
    service_json = json.loads(sys.stdin.read())
    errors, warnings = validate_basic_schema(service_json)
    
    for error in errors:
        print(f"ERROR:{error}")
    for warning in warnings:
        print(f"WARNING:{warning}")
    
    if not errors:
        print("VALID")
    
    sys.exit(0 if not errors else 1)
    
except json.JSONDecodeError as e:
    print(f"ERROR:Invalid JSON - {e}")
    sys.exit(1)
except Exception as e:
    print(f"ERROR:Validation failed - {e}")
    sys.exit(1)
EOF
        
        local validation_output
        validation_output=$(timeout 10s bash -c 'echo "$VALIDATE_SERVICE_JSON" | VALIDATE_SCENARIO_PATH="$VALIDATE_SCENARIO_PATH" python3 "$validation_script"' 2>&1 || echo "TIMEOUT_OR_ERROR")
        rm -f "$validation_script"
        
        local has_errors=false
        if [[ "$validation_output" == "TIMEOUT_OR_ERROR" ]]; then
            print_warning "Schema validation timed out or failed"
            print_info "Install python3 and ensure service.json is well-formed"
        else
            while IFS= read -r line; do
                if [[ "$line" =~ ^ERROR:(.*)$ ]]; then
                    print_error "${BASH_REMATCH[1]}"
                    has_errors=true
                elif [[ "$line" =~ ^WARNING:(.*)$ ]]; then
                    print_warning "${BASH_REMATCH[1]}"
                elif [[ "$line" == "VALID" ]]; then
                    print_success "Schema validation passed"
                fi
            done <<< "$validation_output"
        fi
        
        if [[ "$has_errors" == "true" ]]; then
            return 1
        fi
    else
        print_warning "Python3 not available for schema validation"
        print_info "Install python3 for comprehensive schema checking"
    fi
    
    return 0
}

# Validate file paths
validate_file_paths() {
    print_check "Validating file paths..."
    
    local invalid_paths=0
    local checked_paths=0
    
    # Function to check if a path is valid
    check_path() {
        local path="$1"
        local context="$2"
        
        checked_paths=$((checked_paths + 1))
        
        # Skip if path contains variables
        if [[ "$path" =~ \$\{.*\} ]] || [[ "$path" =~ \$[A-Z_] ]]; then
            print_info "Skipping variable path: $path"
            return 0
        fi
        
        # Check in scenario directory
        if [[ -e "${VALIDATE_SCENARIO_PATH}/${path}" ]]; then
            print_info "✓ Found in scenario: $path"
            return 0
        fi
        
        # Check in Vrooli root (files that get copied)
        if [[ -e "${var_ROOT_DIR}/${path}" ]]; then
            print_info "✓ Found in framework: $path"
            return 0
        fi
        
        # Check known framework patterns
        case "$path" in
            scripts/lib/*.sh|scripts/manage.sh|scripts/main/*.sh)
                # Only validate if it's a framework pattern - let missing files be warnings
                if [[ -e "${var_ROOT_DIR}/${path}" ]]; then
                    print_info "✓ Found framework file: $path"
                    return 0
                else
                    print_warning "Framework file not found: $path"
                    invalid_paths=$((invalid_paths + 1))
                    return 0  # Warn but don't fail validation
                fi
                ;;
        esac
        
        print_warning "Path not found ($context): $path"
        invalid_paths=$((invalid_paths + 1))
        return 1
    }
    
    # Extract and check lifecycle command paths
    if command -v jq >/dev/null 2>&1; then
        # Check lifecycle commands
        local lifecycle_paths
        lifecycle_paths=$(echo "$VALIDATE_SERVICE_JSON" | timeout 5s jq -r '.lifecycle // {} | to_entries | .[] | select(.value | type == "object") | .value.steps[]? | .run // empty' 2>/dev/null || true)
        
        if [[ -n "$lifecycle_paths" ]]; then
            while IFS= read -r cmd; do
                # Extract the script/file path from the command
                local script_path
                script_path=$(echo "$cmd" | awk '{print $1}' | sed 's/^bash\s*//')
                
                if [[ -n "$script_path" ]] && [[ ! "$script_path" =~ ^(echo|cd|ln|chmod|curl|sleep|pkill) ]]; then
                    check_path "$script_path" "lifecycle command"
                fi
            done <<< "$lifecycle_paths"
        fi
        
        # Check initialization file paths
        local init_files
        init_files=$(echo "$VALIDATE_SERVICE_JSON" | timeout 5s jq -r '.resources // {} | to_entries | .[] | .value | to_entries | .[] | .value.initialization?.workflows[]?.file // empty, .value.initialization?.apps[]?.file // empty, .value.initialization?.scripts[]?.file // empty' 2>/dev/null || true)
        
        if [[ -n "$init_files" ]]; then
            while IFS= read -r file; do
                [[ -n "$file" ]] && check_path "$file" "initialization file"
            done <<< "$init_files"
        fi
    else
        print_warning "jq not available for path extraction"
        print_info "Install jq for comprehensive path validation"
    fi
    
    if [[ $checked_paths -eq 0 ]]; then
        print_info "No file paths to validate"
    elif [[ $invalid_paths -eq 0 ]]; then
        print_success "All $checked_paths file paths validated"
    else
        print_warning "$invalid_paths of $checked_paths paths could not be validated"
        print_info "These paths may be valid in the generated app"
    fi
    
    return 0
}

# Validate resources
validate_resources() {
    print_check "Validating resource configuration..."
    
    local enabled_resources=0
    local required_resources=0
    
    if command -v jq >/dev/null 2>&1; then
        enabled_resources=$(echo "$VALIDATE_SERVICE_JSON" | timeout 5s jq '[.resources // {} | to_entries | .[] | .value | to_entries | .[] | select(.value.enabled == true)] | length' 2>/dev/null || echo "0")
        
        required_resources=$(echo "$VALIDATE_SERVICE_JSON" | timeout 5s jq '[.resources // {} | to_entries | .[] | .value | to_entries | .[] | select(.value.required == true)] | length' 2>/dev/null || echo "0")
        
        if [[ $enabled_resources -eq 0 ]]; then
            print_warning "No resources are enabled"
        else
            print_success "$enabled_resources resources enabled ($required_resources required)"
        fi
        
        # List resources if verbose
        if [[ "$VALIDATE_VERBOSE" == "true" ]] && [[ $enabled_resources -gt 0 ]]; then
            local resource_list
            resource_list=$(echo "$VALIDATE_SERVICE_JSON" | timeout 5s jq -r '
                .resources // {} | 
                to_entries | 
                .[] | 
                .value | 
                to_entries | 
                .[] | 
                select(.value.enabled == true) | 
                "\(.key) (\(if .value.required then "required" else "optional" end))"
            ' 2>/dev/null || true)
            
            if [[ -n "$resource_list" ]]; then
                while IFS= read -r resource; do
                    [[ -n "$resource" ]] && print_info "• $resource"
                done <<< "$resource_list"
            fi
        fi
    else
        print_warning "jq not available for resource analysis"
    fi
    
    return 0
}

#######################################
# Validate resource initialization data using manage.sh validators
#######################################
validate_resource_initializations() {
    print_check "Validating resource initialization data..."
    
    local total=0
    local passed=0
    local failed=0
    local skipped=0
    
    # Process each resource category
    for category in ai automation agents storage execution; do
        # Get resources in this category
        local resources
        resources=$(echo "$VALIDATE_SERVICE_JSON" | jq -r "
            .resources.$category // {} | 
            to_entries[] | 
            select(.value.enabled == true) | 
            .key
        " 2>/dev/null || true)
        
        [[ -z "$resources" ]] && continue
        
        while IFS= read -r resource_type; do
            [[ -z "$resource_type" ]] && continue
            
            # Get initialization array
            local init_items
            init_items=$(echo "$VALIDATE_SERVICE_JSON" | jq -c "
                .resources.$category.\"$resource_type\".initialization // []
            " 2>/dev/null)
            
            # Skip if no initialization
            [[ "$init_items" == "[]" ]] && continue
            
            # Check if manage.sh exists
            local manage_script="${var_SCRIPTS_RESOURCES_DIR}/${category}/${resource_type}/manage.sh"
            if [[ ! -x "$manage_script" ]]; then
                print_info "No validator available for $resource_type"
                continue
            fi
            
            # Validate each item
            echo "$init_items" | jq -c '.[]' | while IFS= read -r item; do
                ((total++))
                
                local item_type item_file item_enabled item_description
                item_type=$(echo "$item" | jq -r '.type')
                item_file=$(echo "$item" | jq -r '.file')
                item_enabled=$(echo "$item" | jq -r '.enabled // true')
                item_description=$(echo "$item" | jq -r '.description // ""')
                
                # Skip disabled items
                if [[ "$item_enabled" == "false" ]]; then
                    print_info "Skipping disabled: $resource_type/$item_type"
                    ((skipped++))
                    continue
                fi
                
                # Build full file path
                local file_path="${VALIDATE_SCENARIO_PATH}/${item_file}"
                
                # Check file exists
                if [[ ! -f "$file_path" ]]; then
                    print_error "$resource_type: File not found - $item_file"
                    ((failed++))
                    continue
                fi
                
                # Run validation using new interface
                local validation_result=0
                "$manage_script" \
                    --action validate-injection \
                    --validation-type "$item_type" \
                    --validation-file "$file_path" \
                    --yes yes >/dev/null 2>&1 || validation_result=$?
                
                if [[ $validation_result -eq 0 ]]; then
                    local display_name="$(basename "$item_file")"
                    [[ -n "$item_description" ]] && display_name="$display_name ($item_description)"
                    print_success "$resource_type: $item_type validated ($display_name)"
                    ((passed++))
                else
                    local display_name="$(basename "$item_file")"
                    print_warning "$resource_type: $item_type validation failed ($display_name)"
                    ((failed++))
                fi
            done
        done <<< "$resources"
    done
    
    # Summary
    if [[ $total -eq 0 ]]; then
        print_info "No resource initialization data to validate"
    else
        echo
        print_info "Resource Initialization Summary:"
        print_info "  Total items: $total"
        [[ $passed -gt 0 ]] && print_success "  Passed: $passed"
        [[ $failed -gt 0 ]] && print_warning "  Failed: $failed"
        [[ $skipped -gt 0 ]] && print_info "  Skipped: $skipped"
    fi
    
    # Don't fail overall validation for resource init issues (warnings only)
    return 0
}

# Generate summary
generate_summary() {
    echo
    echo "════════════════════════════════════════════════════════"
    echo "    Validation Summary"
    echo "════════════════════════════════════════════════════════"
    
    local exit_code=0
    
    if [[ $VALIDATE_ERRORS -gt 0 ]]; then
        echo -e "${RED}✗ Validation failed with $VALIDATE_ERRORS error(s)${NC}"
        exit_code=1
    elif [[ $VALIDATE_WARNINGS -gt 0 ]]; then
        echo -e "${YELLOW}⚠ Validation passed with $VALIDATE_WARNINGS warning(s)${NC}"
        exit_code=2
    else
        echo -e "${GREEN}✓ All validations passed${NC}"
        exit_code=0
    fi
    
    echo
    echo "Scenario: $VALIDATE_SCENARIO_NAME"
    echo "Errors:   $VALIDATE_ERRORS"
    echo "Warnings: $VALIDATE_WARNINGS"
    echo "════════════════════════════════════════════════════════"
    echo
    
    return $exit_code
}

# Main validation function (renamed for sourcing)
validate_scenario_main() {
    # Reset counters
    VALIDATE_ERRORS=0
    VALIDATE_WARNINGS=0
    
    parse_args "$@"
    print_header
    
    echo "Validating scenario: $VALIDATE_SCENARIO_NAME"
    echo
    
    # Run validations
    validate_scenario_exists || return 1
    validate_service_json || return 1
    validate_schema_compliance
    validate_file_paths
    validate_resources
    validate_resource_initializations
    
    # Generate summary and return exit code
    generate_summary
    return $?
}

# Only execute main if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    validate_scenario_main "$@"
    exit $?
fi