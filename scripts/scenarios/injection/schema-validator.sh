#!/usr/bin/env bash
set -euo pipefail

# Schema Validator for Scenarios Configuration
# This script validates scenarios.json files against the JSON schema

DESCRIPTION="Validates scenarios configuration files against the JSON schema"

# Source var.sh first to get standardized paths
SCRIPTS_SCENARIOS_INJECTION_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPTS_SCENARIOS_INJECTION_DIR}/../../lib/utils/var.sh"

# Source common utilities if available
if [[ -f "${var_SCRIPTS_RESOURCES_DIR}/common.sh" ]]; then
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
fi

# Source logging utilities
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Source argument helpers
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args.sh"

# Schema file paths - use official schemas
readonly SCHEMA_FILE="${var_VROOLI_CONFIG_DIR}/schemas/scenarios.schema.json"
readonly SERVICE_SCHEMA_FILE="${var_SCHEMAS_DIR}/service.schema.json"

#######################################
# Parse command line arguments
#######################################
validator::parse_arguments() {
    args::reset
    
    args::register_help
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "validate|init|check-schema" \
        --default "validate"
    
    args::register \
        --name "config-file" \
        --flag "f" \
        --desc "Path to scenarios configuration file" \
        --type "value" \
        --default "${HOME}/.vrooli/scenarios.json"
    
    args::register \
        --name "verbose" \
        --flag "v" \
        --desc "Verbose output" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    if args::is_asking_for_help "$@"; then
        validator::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export CONFIG_FILE=$(args::get "config-file")
    export VERBOSE=$(args::get "verbose")
}

#######################################
# Display usage information
#######################################
validator::usage() {
    cat << EOF
Scenarios Configuration Schema Validator

USAGE:
    $0 [OPTIONS]

DESCRIPTION:
    Validates scenarios.json configuration files against the JSON schema.
    Can also initialize new configuration files from defaults.

OPTIONS:
    -a, --action ACTION     Action to perform (default: validate)
                           Options: validate, init, check-schema
    -f, --config-file PATH  Path to scenarios.json file (default: ~/.vrooli/scenarios.json)
    -v, --verbose          Enable verbose output (yes/no, default: no)
    -h, --help             Show this help message

ACTIONS:
    validate      Validate a scenarios configuration file
    init          Initialize a new scenarios configuration file from defaults
    check-schema  Validate the schema file itself

EXAMPLES:
    # Validate default configuration
    $0 --action validate
    
    # Validate specific file
    $0 --action validate --config-file ./my-scenarios.json
    
    # Initialize new configuration
    $0 --action init --config-file ./new-scenarios.json
    
    # Check schema validity
    $0 --action check-schema

EOF
}

#######################################
# Check if required tools are available
#######################################
validator::check_dependencies() {
    if ! system::is_command "jq"; then
        log::error "jq command not available - required for JSON validation"
        log::info "Install jq: sudo apt-get install jq (Ubuntu/Debian) or brew install jq (macOS)"
        return 1
    fi
    
    # Check if ajv-cli is available for advanced schema validation
    if system::is_command "ajv"; then
        log::debug "ajv-cli available for advanced schema validation"
        export AJV_AVAILABLE=true
    else
        log::debug "ajv-cli not available, using basic jq validation"
        export AJV_AVAILABLE=false
    fi
    
    return 0
}

#######################################
# Validate JSON syntax using jq
# Arguments:
#   $1 - file path
# Returns:
#   0 if valid JSON, 1 otherwise
#######################################
validator::validate_json_syntax() {
    local file_path="$1"
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    if jq . "$file_path" >/dev/null 2>&1; then
        log::debug "JSON syntax is valid: $file_path"
        return 0
    else
        log::error "Invalid JSON syntax in: $file_path"
        return 1
    fi
}

#######################################
# Validate configuration structure using jq
# Arguments:
#   $1 - file path
# Returns:
#   0 if structure is valid, 1 otherwise
#######################################
validator::validate_structure() {
    local file_path="$1"
    
    log::info "Validating configuration structure..."
    
    # Check required root properties
    local has_scenarios
    has_scenarios=$(jq -e '.scenarios' "$file_path" >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_scenarios" != "true" ]]; then
        log::error "Missing required 'scenarios' property"
        return 1
    fi
    
    # Check that scenarios is an object
    local scenarios_type
    scenarios_type=$(jq -r '.scenarios | type' "$file_path")
    
    if [[ "$scenarios_type" != "object" ]]; then
        log::error "'scenarios' must be an object, got: $scenarios_type"
        return 1
    fi
    
    # Validate each scenario structure
    local scenario_names
    scenario_names=$(jq -r '.scenarios | keys[]' "$file_path")
    
    for scenario in $scenario_names; do
        log::debug "Validating scenario: $scenario"
        
        # Check required scenario properties
        local has_description has_version has_resources
        has_description=$(jq -e ".scenarios[\"$scenario\"].description" "$file_path" >/dev/null 2>&1 && echo "true" || echo "false")
        has_version=$(jq -e ".scenarios[\"$scenario\"].version" "$file_path" >/dev/null 2>&1 && echo "true" || echo "false")
        has_resources=$(jq -e ".scenarios[\"$scenario\"].resources" "$file_path" >/dev/null 2>&1 && echo "true" || echo "false")
        
        if [[ "$has_description" != "true" ]]; then
            log::error "Scenario '$scenario' missing required 'description' property"
            return 1
        fi
        
        if [[ "$has_version" != "true" ]]; then
            log::error "Scenario '$scenario' missing required 'version' property"
            return 1
        fi
        
        if [[ "$has_resources" != "true" ]]; then
            log::error "Scenario '$scenario' missing required 'resources' property"
            return 1
        fi
        
        # Validate version format (basic semantic versioning check)
        local version
        version=$(jq -r ".scenarios[\"$scenario\"].version" "$file_path")
        
        if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            log::error "Scenario '$scenario' has invalid version format: $version (expected: X.Y.Z)"
            return 1
        fi
        
        # Check for circular dependencies
        if ! validator::check_dependencies_cycle "$scenario" "$file_path"; then
            return 1
        fi
    done
    
    log::success "Configuration structure is valid"
    return 0
}

#######################################
# Check for circular dependencies in scenario
# Arguments:
#   $1 - scenario name
#   $2 - file path
#   $3 - visited scenarios (for recursion)
# Returns:
#   0 if no cycles, 1 if cycle detected
#######################################
validator::check_dependencies_cycle() {
    local scenario="$1"
    local file_path="$2"
    local visited="${3:-}"
    
    # Check if we've already visited this scenario
    if [[ "$visited" == *"$scenario"* ]]; then
        log::error "Circular dependency detected involving scenario: $scenario"
        return 1
    fi
    
    # Get dependencies for this scenario
    local dependencies
    dependencies=$(jq -r ".scenarios[\"$scenario\"].dependencies[]? // empty" "$file_path")
    
    if [[ -z "$dependencies" ]]; then
        return 0
    fi
    
    # Check each dependency recursively
    for dep in $dependencies; do
        # Check if dependency exists
        local dep_exists
        dep_exists=$(jq -e ".scenarios[\"$dep\"]" "$file_path" >/dev/null 2>&1 && echo "true" || echo "false")
        
        if [[ "$dep_exists" != "true" ]]; then
            log::error "Scenario '$scenario' depends on non-existent scenario: $dep"
            return 1
        fi
        
        # Recursively check for cycles
        if ! validator::check_dependencies_cycle "$dep" "$file_path" "$visited $scenario"; then
            return 1
        fi
    done
    
    return 0
}

#######################################
# Validate using ajv-cli if available
# Arguments:
#   $1 - file path
# Returns:
#   0 if valid, 1 otherwise
#######################################
validator::validate_with_ajv() {
    local file_path="$1"
    
    if [[ "$AJV_AVAILABLE" != "true" ]]; then
        log::debug "ajv-cli not available, skipping advanced schema validation"
        return 0
    fi
    
    log::info "Validating with ajv-cli..."
    
    if ajv validate -s "$SCHEMA_FILE" -d "$file_path" >/dev/null 2>&1; then
        log::success "Advanced schema validation passed"
        return 0
    else
        log::error "Advanced schema validation failed"
        if [[ "$VERBOSE" == "yes" ]]; then
            ajv validate -s "$SCHEMA_FILE" -d "$file_path"
        fi
        return 1
    fi
}

#######################################
# Validate scenarios configuration file
# Arguments:
#   $1 - file path
# Returns:
#   0 if valid, 1 otherwise
#######################################
validator::validate_config() {
    local file_path="$1"
    
    log::header "🔍 Validating Scenarios Configuration"
    log::info "File: $file_path"
    
    # Step 1: Check JSON syntax
    if ! validator::validate_json_syntax "$file_path"; then
        return 1
    fi
    
    # Step 2: Validate structure
    if ! validator::validate_structure "$file_path"; then
        return 1
    fi
    
    # Step 3: Advanced schema validation if available
    if ! validator::validate_with_ajv "$file_path"; then
        return 1
    fi
    
    log::success "✅ Configuration file is valid"
    return 0
}

#######################################
# Initialize new configuration file from defaults
# Arguments:
#   $1 - target file path
# Returns:
#   0 if successful, 1 otherwise
#######################################
validator::init_config() {
    local target_file="$1"
    
    log::header "🚀 Initializing Scenarios Configuration"
    log::info "Target: $target_file"
    
    # Check if file already exists
    if [[ -f "$target_file" ]]; then
        log::error "Configuration file already exists: $target_file"
        log::info "Use --force to overwrite or choose a different path"
        return 1
    fi
    
    # Create directory if needed
    local target_dir
    target_dir=$(dirname "$target_file")
    
    if [[ ! -d "$target_dir" ]]; then
        log::info "Creating directory: $target_dir"
        mkdir -p "$target_dir"
    fi
    
    # Copy defaults
    if [[ ! -f "$DEFAULTS_FILE" ]]; then
        log::error "Defaults file not found: $DEFAULTS_FILE"
        return 1
    fi
    
    cp "$DEFAULTS_FILE" "$target_file"
    
    # Validate the new file
    if validator::validate_config "$target_file"; then
        log::success "✅ Configuration file initialized successfully"
        log::info "Edit $target_file to customize your scenarios"
        return 0
    else
        log::error "Failed to validate newly created configuration file"
        rm -f "$target_file"
        return 1
    fi
}

#######################################
# Check schema file validity
# Returns:
#   0 if valid, 1 otherwise
#######################################
validator::check_schema() {
    log::header "🔍 Validating Schema File"
    log::info "Schema: $SCHEMA_FILE"
    
    if ! validator::validate_json_syntax "$SCHEMA_FILE"; then
        return 1
    fi
    
    # Additional schema-specific checks
    local has_schema_property
    has_schema_property=$(jq -e '."$schema"' "$SCHEMA_FILE" >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_schema_property" != "true" ]]; then
        log::warn "Schema file missing '\$schema' property"
    fi
    
    log::success "✅ Schema file is valid"
    return 0
}

#######################################
# Main execution function
#######################################
validator::main() {
    validator::parse_arguments "$@"
    
    if ! validator::check_dependencies; then
        exit 1
    fi
    
    case "$ACTION" in
        "validate")
            validator::validate_config "$CONFIG_FILE"
            ;;
        "init")
            validator::init_config "$CONFIG_FILE"
            ;;
        "check-schema")
            validator::check_schema
            ;;
        *)
            log::error "Unknown action: $ACTION"
            validator::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    validator::main "$@"
fi