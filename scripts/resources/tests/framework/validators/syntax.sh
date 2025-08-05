#!/usr/bin/env bash
# Syntax Validator - Core Layer 1 Validation Logic
# Part of Layer 1 Syntax Validation System

set -euo pipefail

# Get script directory for sourcing dependencies
SYNTAX_VALIDATOR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRAMEWORK_DIR="$(cd "$SYNTAX_VALIDATOR_DIR/.." && pwd)"

# Source required components
source "$FRAMEWORK_DIR/parsers/contract-parser.sh"
source "$FRAMEWORK_DIR/parsers/script-analyzer.sh"
source "$FRAMEWORK_DIR/cache/cache-manager.sh"

# Global validation results
declare -g SYNTAX_VALIDATION_RESULTS=()
declare -g SYNTAX_VALIDATION_ERRORS=()
declare -g SYNTAX_VALIDATION_WARNINGS=()

#######################################
# Initialize syntax validator
# Arguments:
#   $1 - contracts directory (optional)
# Returns:
#   0 on success, 1 on error
#######################################
syntax_validator_init() {
    local contracts_dir="${1:-}"
    
    # Initialize contract parser
    if ! contract_parser_init "$contracts_dir"; then
        echo "Failed to initialize contract parser" >&2
        return 1
    fi
    
    # Initialize cache manager
    if ! cache_manager_init; then
        echo "Warning: Failed to initialize cache manager - proceeding without caching" >&2
    fi
    
    # Clear previous results
    SYNTAX_VALIDATION_RESULTS=()
    SYNTAX_VALIDATION_ERRORS=()
    SYNTAX_VALIDATION_WARNINGS=()
    
    echo "Syntax validator initialized"
    return 0
}

#######################################
# Validate required actions are implemented
# Arguments:
#   $1 - resource name
#   $2 - resource category
#   $3 - manage.sh script path
# Returns:
#   0 if all required actions present, 1 if missing actions
#######################################
validate_required_actions() {
    local resource_name="$1"
    local category="$2"
    local script_path="$3"
    
    echo "Validating required actions for $resource_name ($category)..."
    
    # Get required actions from contract
    local required_actions
    if ! required_actions=$(get_required_actions "$category" 2>/dev/null); then
        SYNTAX_VALIDATION_ERRORS+=("Failed to get required actions for $category")
        return 1
    fi
    
    # Get implemented actions from script
    local implemented_actions
    if ! implemented_actions=$(extract_script_actions "$script_path"); then
        SYNTAX_VALIDATION_ERRORS+=("Failed to extract actions from $script_path")
        return 1
    fi
    
    # Check each required action
    local missing_actions=()
    local found_actions=()
    
    while IFS= read -r action; do
        [[ -z "$action" ]] && continue
        
        if echo "$implemented_actions" | grep -q "^$action$"; then
            found_actions+=("$action")
        else
            missing_actions+=("$action")
        fi
    done <<< "$required_actions"
    
    # Report results
    if [[ ${#missing_actions[@]} -eq 0 ]]; then
        SYNTAX_VALIDATION_RESULTS+=("✅ $resource_name: All required actions implemented (${found_actions[*]})")
        return 0
    else
        SYNTAX_VALIDATION_ERRORS+=("❌ $resource_name: Missing required actions: ${missing_actions[*]}")
        return 1
    fi
}

#######################################
# Validate help patterns are implemented
# Arguments:
#   $1 - resource name
#   $2 - resource category
#   $3 - manage.sh script path
# Returns:
#   0 if help patterns present, 1 if missing
#######################################
validate_syntax_help_patterns() {
    local resource_name="$1"
    local category="$2"
    local script_path="$3"
    
    echo "Validating help patterns for $resource_name..."
    
    # Get required help patterns from contract
    local required_patterns
    if ! required_patterns=$(get_help_patterns "$category"); then
        SYNTAX_VALIDATION_ERRORS+=("Failed to get help patterns for $category")
        return 1
    fi
    
    # Validate help patterns in script using script analyzer
    local validation_output
    if validation_output=$(validate_help_patterns "$script_path" 2>&1); then
        # Parse the output to check if --help is found
        if echo "$validation_output" | grep -q "FOUND:.*--help"; then
            SYNTAX_VALIDATION_RESULTS+=("✅ $resource_name: Help patterns implemented")
            return 0
        else
            SYNTAX_VALIDATION_ERRORS+=("❌ $resource_name: Required --help pattern not implemented")
            return 1
        fi
    else
        SYNTAX_VALIDATION_ERRORS+=("❌ $resource_name: Help pattern validation failed")
        return 1
    fi
}

#######################################
# Validate error handling patterns
# Arguments:
#   $1 - resource name
#   $2 - resource category  
#   $3 - manage.sh script path
# Returns:
#   0 if adequate error handling, 1 if insufficient
#######################################
validate_error_handling() {
    local resource_name="$1"
    local category="$2"
    local script_path="$3"
    
    echo "Validating error handling for $resource_name..."
    
    # Check error handling patterns in script
    local validation_output
    if validation_output=$(check_error_handling_patterns "$script_path" 2>&1); then
        # Count found patterns
        local found_count
        found_count=$(echo "$validation_output" | grep -c "FOUND:" || echo "0")
        
        if [[ $found_count -ge 2 ]]; then
            SYNTAX_VALIDATION_RESULTS+=("✅ $resource_name: Adequate error handling patterns ($found_count found)")
            return 0
        else
            SYNTAX_VALIDATION_WARNINGS+=("⚠️  $resource_name: Limited error handling patterns ($found_count found)")
            return 1
        fi
    else
        SYNTAX_VALIDATION_ERRORS+=("❌ $resource_name: Error handling validation failed")
        return 1
    fi
}

#######################################
# Validate file structure compliance
# Arguments:
#   $1 - resource name
#   $2 - resource category
#   $3 - manage.sh script path
# Returns:
#   0 if structure compliant, 1 if missing files
#######################################
validate_file_structure() {
    local resource_name="$1"
    local category="$2"
    local script_path="$3"
    
    echo "Validating file structure for $resource_name..."
    
    # Check required files structure
    local validation_output
    if validation_output=$(check_required_files "$script_path" 2>&1); then
        # Check if config and lib directories exist
        if echo "$validation_output" | grep -q "FOUND:.*config/" && echo "$validation_output" | grep -q "FOUND:.*lib/"; then
            SYNTAX_VALIDATION_RESULTS+=("✅ $resource_name: File structure compliant")
            return 0
        else
            SYNTAX_VALIDATION_WARNINGS+=("⚠️  $resource_name: File structure issues detected")
            # Don't fail validation for file structure as it's not critical for basic functionality
            return 0
        fi
    else
        SYNTAX_VALIDATION_WARNINGS+=("⚠️  $resource_name: File structure validation failed")
        return 0
    fi
}

#######################################
# Validate argument parsing patterns
# Arguments:
#   $1 - resource name
#   $2 - resource category
#   $3 - manage.sh script path
# Returns:
#   0 if patterns consistent, 1 if issues
#######################################
validate_argument_patterns() {
    local resource_name="$1"
    local category="$2"
    local script_path="$3"
    
    echo "Validating argument patterns for $resource_name..."
    
    # Check argument parsing patterns
    local validation_output
    if validation_output=$(analyze_argument_patterns "$script_path" 2>&1); then
        # Check for basic required patterns
        if echo "$validation_output" | grep -q "GOOD:.*action_flag" && echo "$validation_output" | grep -q "GOOD:.*case_statement"; then
            SYNTAX_VALIDATION_RESULTS+=("✅ $resource_name: Argument patterns consistent")
            return 0
        else
            SYNTAX_VALIDATION_WARNINGS+=("⚠️  $resource_name: Argument parsing patterns need improvement")
            return 1
        fi
    else
        SYNTAX_VALIDATION_WARNINGS+=("⚠️  $resource_name: Argument pattern analysis failed")
        return 1
    fi
}

#######################################
# Validate configuration loading
# Arguments:
#   $1 - resource name
#   $2 - resource category
#   $3 - manage.sh script path
# Returns:
#   0 if configuration loading proper, 1 if issues
#######################################
validate_configuration_loading() {
    local resource_name="$1"
    local category="$2"
    local script_path="$3"
    
    echo "Validating configuration loading for $resource_name..."
    
    # Check configuration loading patterns
    local validation_output
    if validation_output=$(check_configuration_loading "$script_path" 2>&1); then
        # Check if common.sh is sourced (minimum requirement)
        if echo "$validation_output" | grep -q "FOUND:.*common_sourced"; then
            SYNTAX_VALIDATION_RESULTS+=("✅ $resource_name: Configuration loading implemented")
            return 0
        else
            SYNTAX_VALIDATION_WARNINGS+=("⚠️  $resource_name: Configuration loading needs improvement")
            return 1
        fi
    else
        SYNTAX_VALIDATION_WARNINGS+=("⚠️  $resource_name: Configuration loading validation failed")
        return 1
    fi
}

#######################################
# Comprehensive syntax validation for a resource
# Arguments:
#   $1 - resource name
#   $2 - resource category
#   $3 - manage.sh script path
# Returns:
#   0 if validation passes, 1 if critical failures
#######################################
validate_resource_syntax() {
    local resource_name="$1"
    local category="$2"
    local script_path="$3"
    local use_cache="${4:-true}"  # Optional: disable cache for testing
    
    local start_time
    start_time=$(date +%s%3N)  # Milliseconds for performance tracking
    
    # Check cache first (if enabled)
    if [[ "$use_cache" == "true" ]]; then
        local cached_result
        if cached_result=$(cache_get "$resource_name" "$script_path" 2>/dev/null); then
            echo "=== Syntax Validation: $resource_name ($category) [CACHED] ==="
            echo "Script: $script_path"
            echo
            
            # Parse and return cached result
            if cache_parse_result "$cached_result"; then
                echo "✅ $resource_name: Syntax validation PASSED (from cache)"
                return 0
            else
                echo "❌ $resource_name: Syntax validation FAILED (from cache)"
                return 1
            fi
        fi
    fi
    
    echo "=== Syntax Validation: $resource_name ($category) ==="
    echo "Script: $script_path"
    echo
    
    # Clear per-resource results
    local resource_results=()
    local resource_errors=()
    local resource_warnings=()
    
    # Critical validations (must pass) - Initially lenient to avoid false positives
    local critical_validations=(
        "validate_required_actions"
    )
    
    # Important validations (warnings if fail)
    local important_validations=(
        "validate_syntax_help_patterns"
        "validate_error_handling"
        "validate_file_structure"
        "validate_argument_patterns"
        "validate_configuration_loading"
    )
    
    local critical_passed=0
    local critical_failed=0
    local important_passed=0
    local important_failed=0
    
    # Capture validation output for caching
    local validation_output=""
    
    # Run critical validations
    for validation in "${critical_validations[@]}"; do
        local validation_result
        if validation_result=$($validation "$resource_name" "$category" "$script_path" 2>&1); then
            ((critical_passed++))
            validation_output+="PASS: $validation\n"
        else
            ((critical_failed++))
            validation_output+="FAIL: $validation - $validation_result\n"
        fi
    done
    
    # Run important validations
    for validation in "${important_validations[@]}"; do
        local validation_result
        if validation_result=$($validation "$resource_name" "$category" "$script_path" 2>&1); then
            ((important_passed++))
            validation_output+="PASS: $validation\n"
        else
            ((important_failed++))
            validation_output+="WARN: $validation - $validation_result\n"
        fi
    done
    
    echo
    echo "=== Validation Summary for $resource_name ==="
    echo "Critical validations: $critical_passed passed, $critical_failed failed"
    echo "Important validations: $important_passed passed, $important_failed failed"
    echo
    
    # Calculate duration
    local end_time
    end_time=$(date +%s%3N)
    local duration_ms=$((end_time - start_time))
    
    # Determine result and cache it
    local status
    local result_code
    if [[ $critical_failed -eq 0 ]]; then
        status="passed"
        result_code=0
        echo "✅ $resource_name: Syntax validation PASSED"
    else
        status="failed"
        result_code=1
        echo "❌ $resource_name: Syntax validation FAILED (critical issues)"
    fi
    
    # Cache the result (if caching enabled)
    if [[ "$use_cache" == "true" ]]; then
        local cache_details
        cache_details="Critical: $critical_passed/$((critical_passed + critical_failed)) passed, Important: $important_passed/$((important_passed + important_failed)) passed"
        local result_json
        result_json=$(cache_create_result_json "$status" "$cache_details" "$duration_ms")
        cache_set "$resource_name" "$script_path" "$result_json" 2>/dev/null || echo "Warning: Failed to cache validation result" >&2
    fi
    
    return $result_code
}

#######################################
# Auto-detect resource category from path
# Arguments:
#   $1 - resource path
# Returns:
#   0 on success, 1 on error
# Outputs:
#   Resource category
#######################################
detect_resource_category() {
    local resource_path="$1"
    
    # Extract category from path (e.g., /path/to/ai/ollama -> ai)
    if [[ "$resource_path" =~ /([^/]+)/[^/]+/?$ ]]; then
        echo "${BASH_REMATCH[1]}"
        return 0
    else
        echo "unknown"
        return 1
    fi
}

#######################################
# Validate multiple resources in batch
# Arguments:
#   $@ - list of resource paths (directories containing manage.sh)
# Returns:
#   0 if all pass, 1 if any fail
#######################################
validate_resources_batch() {
    local resource_paths=("$@")
    
    echo "=== Batch Syntax Validation ==="
    echo "Validating ${#resource_paths[@]} resources..."
    echo
    
    local total_passed=0
    local total_failed=0
    local start_time
    start_time=$(date +%s)
    
    for resource_path in "${resource_paths[@]}"; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        local category
        category=$(detect_resource_category "$resource_path")
        
        local script_path="$resource_path/manage.sh"
        
        if [[ ! -f "$script_path" ]]; then
            echo "⚠️  Skipping $resource_name: manage.sh not found"
            continue
        fi
        
        echo "Validating $resource_name..."
        
        if validate_resource_syntax "$resource_name" "$category" "$script_path"; then
            ((total_passed++))
        else
            ((total_failed++))
        fi
        
        echo "----------------------------------------"
    done
    
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo
    echo "=== Batch Validation Summary ==="
    echo "Total resources: $((total_passed + total_failed))"
    echo "Passed: $total_passed"
    echo "Failed: $total_failed"
    echo "Duration: ${duration}s"
    echo "Average: $((duration * 1000 / (total_passed + total_failed)))ms per resource"
    
    # Display cache statistics
    local cache_stats
    if cache_stats=$(cache_get_stats 2>/dev/null); then
        echo
        echo "=== Cache Performance ==="
        local hit_rate
        hit_rate=$(echo "$cache_stats" | grep -o '"hit_rate_percent":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ')
        local cache_hits
        cache_hits=$(echo "$cache_stats" | grep -o '"cache_hits":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ')
        local cache_misses
        cache_misses=$(echo "$cache_stats" | grep -o '"cache_misses":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ')
        local total_entries
        total_entries=$(echo "$cache_stats" | grep -o '"total_entries":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ')
        
        echo "Cache hits: $cache_hits"
        echo "Cache misses: $cache_misses"
        echo "Hit rate: ${hit_rate}%"
        echo "Total cached entries: $total_entries"
    fi
    echo
    
    # Display collected results
    if [[ ${#SYNTAX_VALIDATION_RESULTS[@]} -gt 0 ]]; then
        echo "Successful validations:"
        printf '%s\n' "${SYNTAX_VALIDATION_RESULTS[@]}"
        echo
    fi
    
    if [[ ${#SYNTAX_VALIDATION_WARNINGS[@]} -gt 0 ]]; then
        echo "Warnings:"
        printf '%s\n' "${SYNTAX_VALIDATION_WARNINGS[@]}"
        echo
    fi
    
    if [[ ${#SYNTAX_VALIDATION_ERRORS[@]} -gt 0 ]]; then
        echo "Errors:"
        printf '%s\n' "${SYNTAX_VALIDATION_ERRORS[@]}"
        echo
    fi
    
    if [[ $total_failed -eq 0 ]]; then
        echo "🎉 All resources passed syntax validation!"
        return 0
    else
        echo "⚠️  $total_failed resource(s) failed syntax validation"
        return 1
    fi
}

#######################################
# Cleanup syntax validator resources
# Arguments: None
# Returns: 0
#######################################
syntax_validator_cleanup() {
    contract_parser_cleanup
    
    # Clear expired cache entries
    local cleared_message
    if cleared_message=$(cache_clear_expired 2>/dev/null); then
        echo "Cache maintenance: $cleared_message"
    fi
    
    # Clear results
    SYNTAX_VALIDATION_RESULTS=()
    SYNTAX_VALIDATION_ERRORS=()
    SYNTAX_VALIDATION_WARNINGS=()
    
    echo "Syntax validator cleanup completed"
    return 0
}

# Export functions for use in other scripts
export -f syntax_validator_init
export -f validate_required_actions
export -f validate_syntax_help_patterns
export -f validate_error_handling
export -f validate_file_structure
export -f validate_argument_patterns
export -f validate_configuration_loading
export -f validate_resource_syntax
export -f detect_resource_category
export -f validate_resources_batch
export -f syntax_validator_cleanup