#!/usr/bin/env bash

################################################################################
# Service JSON Validator
# 
# Validates service.json configurations and referenced files for Vrooli scenarios.
# Works with data extracted by service-json-processor.sh - does not parse JSON directly.
#
# This module focuses on business logic validation:
# - Schema compliance
# - File existence and syntax
# - Resource conflicts  
# - Deployment readiness
# - User-friendly error reporting
#
# Usage:
#   source service-json-validator.sh
#   validate_service_schema "$service_json" || exit 1
#   validate_file_references "$scenario_path" "$file_list" || exit 1
#
################################################################################

set -euo pipefail

# Import dependencies
VALIDATOR_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${VALIDATOR_SCRIPT_DIR}/../../../.." && pwd)"

# Source the processor for data extraction
if [[ -f "${VALIDATOR_SCRIPT_DIR}/service-json-processor.sh" ]]; then
    source "${VALIDATOR_SCRIPT_DIR}/service-json-processor.sh"
fi

# Default validation settings
VALIDATOR_VERBOSE="${VALIDATOR_VERBOSE:-false}"
VALIDATOR_STRICT="${VALIDATOR_STRICT:-true}"

################################################################################
# Logging Functions
################################################################################

validator_log_info() {
    if [[ "$VALIDATOR_VERBOSE" == "true" ]]; then
        echo "[VALIDATOR INFO] $1" >&2
    fi
}

validator_log_error() {
    echo "[VALIDATOR ERROR] $1" >&2
}

validator_log_warning() {
    echo "[VALIDATOR WARNING] $1" >&2
}

################################################################################
# Schema Validation Functions
################################################################################

# Validate service.json against schema requirements
# Args: $1=service_json_content
# Returns: 0 if valid, 1 if invalid
validate_service_schema() {
    local service_json="$1"
    
    validator_log_info "Validating service schema..."
    
    # Basic JSON validation
    if ! echo "$service_json" | jq empty 2>/dev/null; then
        validator_log_error "Service configuration is not valid JSON"
        return 1
    fi
    
    # Check required top-level sections
    local required_sections=("service" "resources")
    for section in "${required_sections[@]}"; do
        if [[ $(echo "$service_json" | jq -r ".$section // false") == "false" ]]; then
            validator_log_error "Missing required section: '$section'"
            validator_log_error "Add this section to your service.json file"
            return 1
        fi
    done
    
    # Validate service metadata
    local service_name
    service_name=$(echo "$service_json" | jq -r '.service.name // ""')
    if [[ -z "$service_name" ]]; then
        validator_log_error "Missing required field: service.name"
        validator_log_error "Add 'service.name' field to identify your scenario"
        return 1
    fi
    
    # Validate service name format (alphanumeric, hyphens, underscores only)
    if ! [[ "$service_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        validator_log_error "Invalid service name format: '$service_name'"
        validator_log_error "Service names must contain only letters, numbers, hyphens, and underscores"
        return 1
    fi
    
    # Check for resources section structure
    local resource_categories
    if ! resource_categories=$(echo "$service_json" | jq -r '.resources | keys[]' 2>/dev/null); then
        validator_log_error "Invalid resources section structure"
        validator_log_error "Resources must be organized by category (storage, ai, automation, etc.)"
        return 1
    fi
    
    if [[ -z "$resource_categories" ]]; then
        validator_log_warning "No resource categories found - service may need resources"
    else
        validator_log_info "Found resource categories: $(echo "$resource_categories" | tr '\n' ' ')"
    fi
    
    validator_log_info "✓ Service schema validation passed"
    return 0
}

# Use external schema validation if available
# Args: $1=service_json_path
# Returns: 0 if valid, 1 if invalid
validate_against_external_schema() {
    local service_json_path="$1"
    local schema_path="${PROJECT_ROOT}/.vrooli/schemas/service.schema.json"
    
    if [[ ! -f "$schema_path" ]]; then
        validator_log_info "Schema file not found, skipping external validation"
        return 0
    fi
    
    # Try ajv-cli first
    if command -v ajv &>/dev/null; then
        validator_log_info "Using ajv for schema validation..."
        if ajv validate -s "$schema_path" -d "$service_json_path" 2>/dev/null; then
            validator_log_info "✓ External schema validation passed"
            return 0
        else
            validator_log_error "Service configuration does not match official schema"
            validator_log_error "Schema location: $schema_path"
            return 1
        fi
    fi
    
    validator_log_info "ajv not available, skipping external schema validation"
    return 0
}

################################################################################
# File Reference Validation Functions
################################################################################

# Validate that all referenced files exist and are accessible
# Args: $1=scenario_path, $2=newline_or_space_separated_file_list
# Returns: 0 if all files exist, 1 if any missing
validate_file_references() {
    local scenario_path="$1"
    local file_list="$2"
    
    if [[ -z "$file_list" ]]; then
        validator_log_info "No files to validate"
        return 0
    fi
    
    validator_log_info "Validating file references..."
    
    local missing_files=()
    local checked_count=0
    
    # Handle both newline-separated (from processor) and space-separated input
    # If input contains newlines, use that; otherwise split on spaces carefully
    local files_array=()
    if [[ "$file_list" == *$'\n'* ]]; then
        # Input has newlines, split on newlines
        while IFS= read -r file; do
            [[ -n "$file" ]] && files_array+=("$file")
        done <<< "$file_list"
    else
        # Input is space-separated, but need to handle spaces in filenames
        # For now, assume no spaces in filenames when space-separated
        read -ra files_array <<< "$file_list"
    fi
    
    # Check each file
    for file in "${files_array[@]}"; do
        [[ -z "$file" ]] && continue
        
        local full_path="${scenario_path}/${file}"
        checked_count=$((checked_count + 1))
        
        if [[ -f "$full_path" ]]; then
            validator_log_info "  ✓ Found: $file"
        else
            missing_files+=("$file")
            validator_log_error "  ✗ Missing: $file"
        fi
    done
    
    if [[ ${#missing_files[@]} -gt 0 ]]; then
        validator_log_error "Missing ${#missing_files[@]} out of $checked_count referenced files:"
        for file in "${missing_files[@]}"; do
            validator_log_error "  - $file"
        done
        validator_log_error "Create these files or remove references from service.json"
        return 1
    fi
    
    validator_log_info "✓ All $checked_count file references validated"
    return 0
}

# Validate JSON/YAML syntax of referenced files
# Args: $1=scenario_path, $2=newline_or_space_separated_file_list
# Returns: 0 if all files are valid, 1 if any invalid
validate_file_syntax() {
    local scenario_path="$1"
    local file_list="$2"
    
    if [[ -z "$file_list" ]]; then
        validator_log_info "No files to validate syntax"
        return 0
    fi
    
    validator_log_info "Validating file syntax..."
    
    local invalid_files=()
    local checked_count=0
    
    # Handle both newline-separated and space-separated input
    local files_array=()
    if [[ "$file_list" == *$'\n'* ]]; then
        while IFS= read -r file; do
            [[ -n "$file" ]] && files_array+=("$file")
        done <<< "$file_list"
    else
        read -ra files_array <<< "$file_list"
    fi
    
    # Check each file
    for file in "${files_array[@]}"; do
        [[ -z "$file" ]] && continue
        
        local full_path="${scenario_path}/${file}"
        [[ ! -f "$full_path" ]] && continue  # Skip if file doesn't exist (caught by validate_file_references)
        
        checked_count=$((checked_count + 1))
        
        # Determine file type and validate accordingly
        case "$file" in
            *.json)
                if jq empty "$full_path" 2>/dev/null; then
                    validator_log_info "  ✓ Valid JSON: $file"
                else
                    invalid_files+=("$file (invalid JSON)")
                    validator_log_error "  ✗ Invalid JSON: $file"
                fi
                ;;
            *.yml|*.yaml)
                if command -v yq &>/dev/null; then
                    if yq eval '.' "$full_path" >/dev/null 2>&1; then
                        validator_log_info "  ✓ Valid YAML: $file"
                    else
                        invalid_files+=("$file (invalid YAML)")
                        validator_log_error "  ✗ Invalid YAML: $file"
                    fi
                else
                    validator_log_info "  ? YAML validation skipped (yq not available): $file"
                fi
                ;;
            *.sql)
                # Basic SQL validation - check for syntax errors
                if [[ -s "$full_path" ]]; then
                    validator_log_info "  ✓ SQL file present: $file"
                else
                    validator_log_warning "  ? Empty SQL file: $file"
                fi
                ;;
            *)
                validator_log_info "  ? Unknown file type, skipping validation: $file"
                ;;
        esac
    done
    
    if [[ ${#invalid_files[@]} -gt 0 ]]; then
        validator_log_error "Found ${#invalid_files[@]} files with syntax errors:"
        for file in "${invalid_files[@]}"; do
            validator_log_error "  - $file"
        done
        validator_log_error "Fix syntax errors in these files"
        return 1
    fi
    
    validator_log_info "✓ All $checked_count files have valid syntax"
    return 0
}

################################################################################
# Resource Validation Functions
################################################################################

# Validate resource configurations for conflicts and compatibility
# Args: $1=service_json_content
# Returns: 0 if no conflicts, 1 if conflicts found
validate_resource_conflicts() {
    local service_json="$1"
    
    validator_log_info "Checking for resource conflicts..."
    
    # Use the processor's conflict detection
    local conflicts
    if conflicts=$(sjp_check_resource_conflicts "$service_json" 2>/dev/null); then
        # Check if any conflicts were found
        local conflict_count
        conflict_count=$(echo "$conflicts" | jq 'length' 2>/dev/null || echo "0")
        
        if [[ "$conflict_count" -gt 0 ]]; then
            validator_log_error "Resource conflicts detected:"
            echo "$conflicts" | jq -r '.[] | "  - " + .description' 2>/dev/null || validator_log_error "  - Failed to parse conflict details"
            validator_log_error "Resolve conflicts before deployment"
            return 1
        fi
    fi
    
    # Additional compatibility checks
    local required_resources
    if required_resources=$(extract_required_resources "$service_json"); then
        # Check for known incompatible combinations
        if echo "$required_resources" | grep -q "postgres" && echo "$required_resources" | grep -q "mysql"; then
            validator_log_warning "Both PostgreSQL and MySQL are required - this may cause conflicts"
        fi
        
        # Check if AI resources have sufficient resource allocations
        if echo "$required_resources" | grep -qE "(ollama|whisper|comfyui)"; then
            validator_log_info "AI resources detected - ensure sufficient CPU/memory allocation"
        fi
    fi
    
    validator_log_info "✓ No resource conflicts found"
    return 0
}

# Check if resources have proper injection handlers available
# Args: $1=space_separated_required_resources
# Returns: 0 if all handlers available, 1 if any missing
validate_injection_handlers() {
    local required_resources="$1"
    
    if [[ -z "$required_resources" ]]; then
        validator_log_info "No required resources to check handlers for"
        return 0
    fi
    
    validator_log_info "Checking injection handlers for required resources..."
    
    local missing_handlers=()
    local checked_count=0
    
    while IFS= read -r resource; do
        [[ -z "$resource" ]] && continue
        
        checked_count=$((checked_count + 1))
        
        # Look for inject.sh script in resource directories
        local inject_script
        inject_script=$(find "${PROJECT_ROOT}/scripts/resources" \
            -name "inject.sh" \
            -path "*/${resource}/*" \
            2>/dev/null | head -1)
        
        if [[ -n "$inject_script" ]]; then
            validator_log_info "  ✓ Handler found: $resource"
        else
            missing_handlers+=("$resource")
            validator_log_warning "  ⚠ No injection handler: $resource"
        fi
    done <<< "$(echo "$required_resources" | tr ' ' '\n')"
    
    if [[ ${#missing_handlers[@]} -gt 0 ]]; then
        validator_log_warning "Missing injection handlers for ${#missing_handlers[@]} resources:"
        for resource in "${missing_handlers[@]}"; do
            validator_log_warning "  - $resource"
        done
        validator_log_warning "These resources will need manual setup at runtime"
        
        if [[ "$VALIDATOR_STRICT" == "true" ]]; then
            validator_log_error "Strict mode: All required resources must have injection handlers"
            return 1
        fi
    fi
    
    validator_log_info "✓ Checked injection handlers for $checked_count resources"
    return 0
}

################################################################################
# Comprehensive Validation Functions
################################################################################

# Complete validation for deployment readiness
# Args: $1=scenario_path, $2=service_json_content
# Returns: 0 if ready for deployment, 1 if not ready
validate_deployment_readiness() {
    local scenario_path="$1"
    local service_json="$2"
    
    validator_log_info "=== Comprehensive Deployment Readiness Check ==="
    
    local validation_errors=0
    
    # 1. Schema validation
    if ! validate_service_schema "$service_json"; then
        ((validation_errors++))
    fi
    
    # 2. External schema validation if available
    local service_json_path="${scenario_path}/service.json"
    if [[ -f "$service_json_path" ]]; then
        if ! validate_against_external_schema "$service_json_path"; then
            ((validation_errors++))
        fi
    fi
    
    # 3. Extract and validate file references
    local all_referenced_files
    if all_referenced_files=$(extract_all_referenced_files "$service_json"); then
        if ! validate_file_references "$scenario_path" "$all_referenced_files"; then
            ((validation_errors++))
        fi
        
        if ! validate_file_syntax "$scenario_path" "$all_referenced_files"; then
            ((validation_errors++))
        fi
    else
        validator_log_warning "Could not extract file references for validation"
    fi
    
    # 4. Resource conflict validation
    if ! validate_resource_conflicts "$service_json"; then
        ((validation_errors++))
    fi
    
    # 5. Injection handler validation
    local required_resources
    if required_resources=$(extract_required_resources "$service_json"); then
        if ! validate_injection_handlers "$required_resources"; then
            ((validation_errors++))
        fi
    fi
    
    # Summary
    if [[ $validation_errors -eq 0 ]]; then
        validator_log_info "✅ All deployment readiness checks passed"
        return 0
    else
        validator_log_error "❌ Deployment readiness validation failed with $validation_errors error(s)"
        return 1
    fi
}

# Quick validation for basic correctness (no file I/O)
# Args: $1=service_json_content
# Returns: 0 if basic structure is valid, 1 if invalid
validate_basic_structure() {
    local service_json="$1"
    
    validator_log_info "Performing basic structure validation..."
    
    # Only validate JSON structure and required fields
    if ! validate_service_schema "$service_json"; then
        return 1
    fi
    
    # Basic resource structure check
    if ! validate_resource_conflicts "$service_json"; then
        return 1
    fi
    
    validator_log_info "✓ Basic structure validation passed"
    return 0
}

################################################################################
# Utility Functions
################################################################################

# Extract all referenced files from service.json using the processor
# Args: $1=service_json_content
# Returns: space-separated list of all referenced files
extract_all_referenced_files() {
    local service_json="$1"
    
    # Use the processor's function to get all referenced files
    local all_files
    if all_files=$(sjp_get_all_referenced_files "$service_json" 2>/dev/null); then
        # Convert newline-separated to space-separated
        echo "$all_files" | tr '\n' ' '
    else
        echo ""
    fi
}

# Extract required resources from service.json using the processor
# Args: $1=service_json_content
# Returns: space-separated list of required resources
extract_required_resources() {
    local service_json="$1"
    
    # Use the processor's function to get required resources
    local required_resources
    if required_resources=$(sjp_get_resources_by_condition "$service_json" "required == true" 2>/dev/null); then
        # Convert newline-separated to space-separated
        echo "$required_resources" | tr '\n' ' '
    else
        echo ""
    fi
}

# Check if validator has all required dependencies
# Returns: 0 if all dependencies available, 1 if missing critical ones
check_validator_dependencies() {
    local missing_deps=()
    
    # Critical dependencies
    if ! command -v jq &>/dev/null; then
        missing_deps+=("jq")
    fi
    
    # Optional but recommended
    if ! command -v yq &>/dev/null; then
        validator_log_info "yq not available - YAML validation will be skipped"
    fi
    
    if ! command -v ajv &>/dev/null; then
        validator_log_info "ajv not available - external schema validation will be skipped"
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        validator_log_error "Missing critical dependencies: ${missing_deps[*]}"
        validator_log_error "Install missing tools before running validation"
        return 1
    fi
    
    return 0
}

################################################################################
# Main Entry Points
################################################################################

# Main validation function - comprehensive check
# Args: $1=scenario_path
# Returns: 0 if all validations pass, 1 if any fail
validate_scenario() {
    local scenario_path="$1"
    local service_json_path="${scenario_path}/service.json"
    
    # Check dependencies first
    if ! check_validator_dependencies; then
        return 1
    fi
    
    # Check if service.json exists
    if [[ ! -f "$service_json_path" ]]; then
        validator_log_error "service.json not found: $service_json_path"
        return 1
    fi
    
    # Load service.json
    local service_json
    if ! service_json=$(cat "$service_json_path"); then
        validator_log_error "Failed to read service.json: $service_json_path"
        return 1
    fi
    
    # Run comprehensive validation
    validate_deployment_readiness "$scenario_path" "$service_json"
}

# Export functions for use by other scripts
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    # Script is being sourced, export functions
    export -f validate_service_schema
    export -f validate_file_references
    export -f validate_file_syntax
    export -f validate_resource_conflicts
    export -f validate_injection_handlers
    export -f validate_deployment_readiness
    export -f validate_basic_structure
    export -f validate_scenario
fi