#!/usr/bin/env bash
set -euo pipefail

# Validate Scenario Template
# This tool validates the structure and integrity of scenario templates

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCENARIOS_DIR="${SCRIPT_DIR}/.."
RESOURCES_DIR="${SCENARIOS_DIR}/.."
INJECTION_DIR="${RESOURCES_DIR}/injection"

# Source common utilities
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"

# Validation results
ERRORS=()
WARNINGS=()
VERBOSE=false

#######################################
# Display usage information
#######################################
usage() {
    cat << EOF
Validate Scenario Template

USAGE:
    $0 TEMPLATE_PATH [OPTIONS]

DESCRIPTION:
    Validates a scenario template for:
    - JSON structure and syntax
    - Required fields and schema compliance
    - Resource availability
    - Asset file existence
    - Variable naming conventions
    - Dependency resolution

OPTIONS:
    --strict              Treat warnings as errors
    --verbose, -v         Show detailed validation output
    --schema PATH         Custom schema file (default: injection/config/schema.json)
    --check-resources     Verify resources are installed
    --help, -h           Show this help message

EXAMPLES:
    # Validate a template file
    $0 templates/ecommerce/scenario.json
    
    # Validate with strict mode
    $0 my-scenario.json --strict
    
    # Check resource availability
    $0 scenario.json --check-resources

EXIT CODES:
    0 - Validation successful
    1 - Validation failed with errors
    2 - Validation passed with warnings (strict mode only)

EOF
}

#######################################
# Add error message
# Arguments:
#   $1 - error message
#######################################
add_error() {
    ERRORS+=("❌ $1")
}

#######################################
# Add warning message
# Arguments:
#   $1 - warning message
#######################################
add_warning() {
    WARNINGS+=("⚠️  $1")
}

#######################################
# Validate JSON structure
# Arguments:
#   $1 - file path
# Returns:
#   0 if valid JSON, 1 otherwise
#######################################
validate_json() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        add_error "File not found: $file"
        return 1
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "Validating JSON structure..."
    fi
    
    if ! jq . "$file" >/dev/null 2>&1; then
        add_error "Invalid JSON syntax in $file"
        return 1
    fi
    
    return 0
}

#######################################
# Validate against schema
# Arguments:
#   $1 - scenario file
#   $2 - schema file
# Returns:
#   0 if valid, 1 otherwise
#######################################
validate_schema() {
    local scenario_file="$1"
    local schema_file="$2"
    
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "Validating against schema..."
    fi
    
    # Use the injection schema validator if available
    local validator="${INJECTION_DIR}/schema-validator.sh"
    
    if [[ -x "$validator" ]]; then
        if ! "$validator" "$scenario_file" 2>/dev/null; then
            add_error "Schema validation failed"
            return 1
        fi
    else
        add_warning "Schema validator not found, skipping schema validation"
    fi
    
    return 0
}

#######################################
# Validate required fields
# Arguments:
#   $1 - scenario content (JSON)
# Returns:
#   0 if all required fields present, 1 otherwise
#######################################
validate_required_fields() {
    local content="$1"
    
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "Checking required fields..."
    fi
    
    # Check for required top-level fields
    local required_fields=("name" "description" "version" "resources")
    
    for field in "${required_fields[@]}"; do
        if ! echo "$content" | jq -e ".$field" >/dev/null 2>&1; then
            add_error "Missing required field: $field"
        fi
    done
    
    # Check name format
    local name
    name=$(echo "$content" | jq -r '.name // ""')
    if [[ ! "$name" =~ ^[a-z0-9-]+$ ]]; then
        add_warning "Name should contain only lowercase letters, numbers, and hyphens: $name"
    fi
    
    # Check version format
    local version
    version=$(echo "$content" | jq -r '.version // ""')
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        add_warning "Version should follow semantic versioning (x.y.z): $version"
    fi
}

#######################################
# Validate resource configurations
# Arguments:
#   $1 - scenario content (JSON)
# Returns:
#   0 if valid, 1 otherwise
#######################################
validate_resources() {
    local content="$1"
    
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "Validating resource configurations..."
    fi
    
    # Get list of resources
    local resources
    resources=$(echo "$content" | jq -r '.resources | keys[]' 2>/dev/null)
    
    if [[ -z "$resources" ]]; then
        add_error "No resources defined"
        return 1
    fi
    
    # Known valid resources
    local valid_resources=(
        "postgres" "minio" "n8n" "node-red" "windmill"
        "vault" "questdb" "qdrant" "ollama" "comfyui"
        "judge0" "searxng" "browserless" "huginn"
    )
    
    for resource in $resources; do
        local found=false
        for valid in "${valid_resources[@]}"; do
            if [[ "$resource" == "$valid" ]]; then
                found=true
                break
            fi
        done
        
        if [[ "$found" == "false" ]]; then
            add_warning "Unknown resource type: $resource"
        fi
        
        # Check for inject configuration
        if ! echo "$content" | jq -e ".resources.$resource.inject" >/dev/null 2>&1; then
            add_warning "Resource '$resource' has no inject configuration"
        fi
    done
}

#######################################
# Validate asset files
# Arguments:
#   $1 - scenario file path
#   $2 - scenario content (JSON)
# Returns:
#   0 if all assets exist, 1 otherwise
#######################################
validate_assets() {
    local scenario_file="$1"
    local content="$2"
    
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "Validating asset files..."
    fi
    
    local scenario_dir
    scenario_dir=$(dirname "$scenario_file")
    
    # Find all file references in the scenario
    local file_refs
    file_refs=$(echo "$content" | jq -r '.. | select(type == "string" and (startswith("assets/") or endswith(".sql") or endswith(".json")))' 2>/dev/null | sort -u)
    
    for file_ref in $file_refs; do
        # Skip if it's a variable
        if [[ "$file_ref" =~ \$ ]]; then
            continue
        fi
        
        # Check if file exists
        local full_path="$scenario_dir/$file_ref"
        
        if [[ ! -f "$full_path" ]]; then
            # Only warn for asset files, not all string values
            if [[ "$file_ref" =~ ^assets/ ]]; then
                add_warning "Asset file not found: $file_ref"
            fi
        fi
    done
}

#######################################
# Validate variables
# Arguments:
#   $1 - scenario content (JSON)
# Returns:
#   0 if valid, 1 otherwise
#######################################
validate_variables() {
    local content="$1"
    
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "Checking variable usage..."
    fi
    
    # Find all variable references
    local variables
    variables=$(echo "$content" | grep -oE '\$\{[A-Z_][A-Z0-9_]*\}' | sed 's/\${//g' | sed 's/}//g' | sort -u)
    
    if [[ -n "$variables" ]]; then
        # Check if environment variables are documented
        local env_required
        env_required=$(echo "$content" | jq -r '.environment.required[]?' 2>/dev/null)
        local env_optional
        env_optional=$(echo "$content" | jq -r '.environment.optional[]?' 2>/dev/null)
        
        for var in $variables; do
            local documented=false
            
            # Check if variable is in required or optional lists
            if echo "$env_required" | grep -q "^$var$"; then
                documented=true
            elif echo "$env_optional" | grep -q "^$var$"; then
                documented=true
            fi
            
            if [[ "$documented" == "false" ]]; then
                add_warning "Variable \${$var} is used but not documented in environment section"
            fi
        done
    fi
}

#######################################
# Validate dependencies
# Arguments:
#   $1 - scenario content (JSON)
# Returns:
#   0 if valid, 1 otherwise
#######################################
validate_dependencies() {
    local content="$1"
    
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "Checking dependencies..."
    fi
    
    # Check for circular dependencies
    local deps
    deps=$(echo "$content" | jq -r '.dependencies[]?' 2>/dev/null)
    
    # TODO: Implement circular dependency detection
    # For now, just check that dependencies are valid scenario names
    for dep in $deps; do
        if [[ ! "$dep" =~ ^[a-z0-9-]+$ ]]; then
            add_warning "Invalid dependency name format: $dep"
        fi
    done
}

#######################################
# Check if resources are available
# Arguments:
#   $1 - scenario content (JSON)
# Returns:
#   0 if all available, 1 otherwise
#######################################
check_resource_availability() {
    local content="$1"
    
    log::info "Checking resource availability..."
    
    # Get list of resources
    local resources
    resources=$(echo "$content" | jq -r '.resources | keys[]' 2>/dev/null)
    
    for resource in $resources; do
        # Check if resource management script exists
        local resource_script="${RESOURCES_DIR}/${resource}/manage.sh"
        
        # Map resource names to correct paths
        case "$resource" in
            postgres|minio|vault|qdrant|questdb)
                resource_script="${RESOURCES_DIR}/storage/${resource}/manage.sh"
                ;;
            n8n|node-red|windmill|comfyui|huginn)
                resource_script="${RESOURCES_DIR}/automation/${resource}/manage.sh"
                ;;
            ollama)
                resource_script="${RESOURCES_DIR}/ai/${resource}/manage.sh"
                ;;
            searxng)
                resource_script="${RESOURCES_DIR}/search/${resource}/manage.sh"
                ;;
        esac
        
        if [[ ! -f "$resource_script" ]]; then
            add_warning "Resource management script not found for: $resource"
        else
            # Check if resource is running (optional)
            if command -v docker >/dev/null 2>&1; then
                if docker ps --format '{{.Names}}' | grep -q "$resource"; then
                    log::success "✓ Resource $resource is running"
                else
                    log::info "○ Resource $resource is not running"
                fi
            fi
        fi
    done
}

#######################################
# Display validation results
# Returns:
#   0 if passed, 1 if failed
#######################################
display_results() {
    local strict="$1"
    
    echo ""
    log::header "Validation Results"
    
    # Display errors
    if [[ ${#ERRORS[@]} -gt 0 ]]; then
        log::error "Errors found: ${#ERRORS[@]}"
        for error in "${ERRORS[@]}"; do
            echo "  $error"
        done
        echo ""
    fi
    
    # Display warnings
    if [[ ${#WARNINGS[@]} -gt 0 ]]; then
        log::warn "Warnings found: ${#WARNINGS[@]}"
        for warning in "${WARNINGS[@]}"; do
            echo "  $warning"
        done
        echo ""
    fi
    
    # Summary
    if [[ ${#ERRORS[@]} -eq 0 ]]; then
        if [[ ${#WARNINGS[@]} -eq 0 ]]; then
            log::success "✅ Validation passed with no issues"
            return 0
        else
            if [[ "$strict" == "true" ]]; then
                log::error "❌ Validation failed in strict mode (warnings treated as errors)"
                return 2
            else
                log::warn "⚠️  Validation passed with warnings"
                return 0
            fi
        fi
    else
        log::error "❌ Validation failed with errors"
        return 1
    fi
}

#######################################
# Main validation function
# Arguments:
#   $1 - scenario file path
#   $2 - strict mode (true/false)
#   $3 - check resources (true/false)
#   $4 - schema file path
#######################################
validate_template() {
    local scenario_file="$1"
    local strict="${2:-false}"
    local check_resources="${3:-false}"
    local schema_file="${4:-${INJECTION_DIR}/config/schema.json}"
    
    log::header "Validating Scenario Template"
    log::info "File: $scenario_file"
    
    # Reset results
    ERRORS=()
    WARNINGS=()
    
    # Validate JSON structure
    if ! validate_json "$scenario_file"; then
        display_results "$strict"
        return 1
    fi
    
    # Read scenario content
    local content
    content=$(cat "$scenario_file")
    
    # Run validations
    validate_required_fields "$content"
    validate_resources "$content"
    validate_assets "$scenario_file" "$content"
    validate_variables "$content"
    validate_dependencies "$content"
    
    # Validate against schema if available
    if [[ -f "$schema_file" ]]; then
        validate_schema "$scenario_file" "$schema_file"
    else
        add_warning "Schema file not found, skipping schema validation"
    fi
    
    # Check resource availability if requested
    if [[ "$check_resources" == "true" ]]; then
        check_resource_availability "$content"
    fi
    
    # Display results
    display_results "$strict"
}

#######################################
# Parse command line arguments
#######################################
main() {
    local scenario_file=""
    local strict=false
    local check_resources=false
    local schema_file=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --strict)
                strict=true
                shift
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --schema)
                schema_file="$2"
                shift 2
                ;;
            --check-resources)
                check_resources=true
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            -*)
                log::error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                scenario_file="$1"
                shift
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$scenario_file" ]]; then
        log::error "Scenario file path is required"
        usage
        exit 1
    fi
    
    # Run validation
    validate_template "$scenario_file" "$strict" "$check_resources" "$schema_file"
    exit $?
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi