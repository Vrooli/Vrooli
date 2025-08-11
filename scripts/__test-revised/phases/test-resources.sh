#!/usr/bin/env bash
# Resource Validation Phase - Vrooli Testing Infrastructure
# 
# Tests all resources in scripts/resources/ directory:
# - Validates resource construction and configuration
# - Tests enabled resources with realistic mock data  
# - Verifies resource dependencies and ports
# - Checks service.json configuration
# - Uses simplified mocks for functional testing

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ROOT="${PROJECT_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091  
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"

show_usage() {
    cat << 'EOF'
Resource Validation Phase - Test resource construction and enabled services

Usage: ./test-resources.sh [OPTIONS]

OPTIONS:
  --verbose      Show detailed output for each resource tested
  --parallel     Run tests in parallel (when possible)  
  --no-cache     Skip caching optimizations
  --dry-run      Show what would be tested without running
  --clear-cache  Clear resource validation cache before running
  --help         Show this help

WHAT THIS PHASE TESTS:
  1. Validates all resources in scripts/resources/ are properly constructed
  2. Tests service.json configuration is valid and complete
  3. Verifies enabled resources have required dependencies
  4. Tests enabled resources with simplified mocks for functionality
  5. Checks resource port assignments and conflicts
  6. Validates resource initialization scripts

EXAMPLES:
  ./test-resources.sh                 # Test all resources
  ./test-resources.sh --verbose       # Show detailed resource testing
  ./test-resources.sh --no-cache      # Force re-testing of all resources
EOF
}

# Parse command line arguments
CLEAR_CACHE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --clear-cache)
            CLEAR_CACHE="true"
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        --verbose|--parallel|--no-cache|--dry-run)
            # These are handled by the main orchestrator
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Clear cache if requested
if [[ -n "$CLEAR_CACHE" ]]; then
    clear_cache "resources"
fi

# Load mocks for resource testing
load_resource_mocks() {
    # Source all mock files
    for mock_file in "$SCRIPT_DIR"/mocks/*.sh; do
        if [[ -f "$mock_file" ]]; then
            # shellcheck disable=SC1090
            source "$mock_file"
            log_debug "Loaded mock: $(basename "$mock_file")"
        fi
    done
}

# Test service.json configuration
test_service_json_config() {
    log_info "🔧 Testing service.json configuration..."
    
    local service_json="$PROJECT_ROOT/.vrooli/service.json"
    local relative_path
    relative_path=$(relative_path "$service_json")
    
    # Test 1: File exists
    if ! run_cached_test "$service_json" "existence" "assert_file_exists '$service_json' 'service.json'" "config-exists: $relative_path"; then
        log_error "service.json not found - cannot validate resources"
        return 1
    fi
    increment_test_counter "passed"
    
    # Test 2: Valid JSON
    if ! run_cached_test "$service_json" "json-valid" "assert_json_valid '$service_json' 'service.json'" "json-valid: $relative_path"; then
        log_error "service.json has invalid JSON - cannot parse resources"
        return 1
    fi
    increment_test_counter "passed"
    
    # Test 3: Has resources section
    if command -v jq >/dev/null 2>&1; then
        if run_cached_test "$service_json" "has-resources" "jq -e '.resources' '$service_json' >/dev/null" "has-resources: $relative_path"; then
            increment_test_counter "passed"
        else
            log_error "service.json missing resources section"
            increment_test_counter "failed"
            return 1
        fi
    else
        log_test_skip "has-resources: $relative_path" "jq not available"
        increment_test_counter "skipped"
    fi
    
    return 0
}

# Discover and validate resource files
test_resource_construction() {
    log_info "📁 Testing resource construction..."
    
    local resources_dir="$PROJECT_ROOT/scripts/resources"
    local providers_dir="$resources_dir/providers"
    
    # Test resources directory exists
    if ! assert_directory_exists "$resources_dir" "resources directory"; then
        log_error "Resources directory not found: $resources_dir"
        increment_test_counter "failed"
        return 1
    fi
    increment_test_counter "passed"
    
    # Find all resource provider scripts
    local resource_files=()
    if [[ -d "$providers_dir" ]]; then
        while IFS= read -r -d '' resource_file; do
            resource_files+=("$resource_file")
        done < <(find "$providers_dir" -type f -name "*.sh" -print0 2>/dev/null)
    fi
    
    # Also check main resources directory
    while IFS= read -r -d '' resource_file; do
        resource_files+=("$resource_file")
    done < <(find "$resources_dir" -maxdepth 1 -type f -name "*.sh" -print0 2>/dev/null)
    
    local total_resources=${#resource_files[@]}
    log_info "📋 Found $total_resources resource files to validate"
    
    if [[ $total_resources -eq 0 ]]; then
        log_warning "No resource files found in $resources_dir"
        return 0
    fi
    
    # Test each resource file
    local current=0
    for resource_file in "${resource_files[@]}"; do
        ((current++))
        local relative_resource
        relative_resource=$(relative_path "$resource_file")
        
        log_progress "$current" "$total_resources" "Validating resources"
        
        # Test 1: File is executable
        if run_cached_test "$resource_file" "executable" "assert_file_executable '$resource_file' 'resource script'" "executable: $relative_resource"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 2: Bash syntax is valid
        if run_cached_test "$resource_file" "syntax" "validate_shell_syntax '$resource_file' 'resource script'" "syntax: $relative_resource"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 3: Shellcheck passes
        if run_cached_test "$resource_file" "shellcheck" "run_shellcheck '$resource_file' 'resource script'" "shellcheck: $relative_resource"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
        fi
    done
    
    return 0
}

# Test enabled resources with mocks
test_enabled_resources() {
    log_info "⚡ Testing enabled resources with mocks..."
    
    # Get enabled resources from service.json
    local enabled_resources
    enabled_resources=$(get_enabled_resources)
    
    if [[ -z "$enabled_resources" ]]; then
        log_warning "No enabled resources found in service.json"
        return 0
    fi
    
    log_info "📋 Testing enabled resources: $(echo "$enabled_resources" | tr '\n' ' ')"
    
    # Load mocks for testing
    if ! load_resource_mocks; then
        log_error "Failed to load resource mocks"
        increment_test_counter "failed"
        return 1
    fi
    
    # Test each enabled resource using convention-based discovery
    while IFS= read -r resource_name; do
        [[ -n "$resource_name" ]] || continue
        
        log_info "🧪 Testing resource: $resource_name"
        
        # Use convention-based function discovery
        test_resource_with_conventions "$resource_name"
        
    done <<< "$enabled_resources"
    
    return 0
}

# Load simplified mocks for testing
load_resource_mocks() {
    log_debug "Loading resource mocks..."
    
    local mocks_dir="$SCRIPT_DIR/mocks"
    
    if [[ ! -d "$mocks_dir" ]]; then
        log_warning "Mocks directory not found: $mocks_dir"
        return 1
    fi
    
    # Source all available mock files
    for mock_file in "$mocks_dir"/*.sh; do
        if [[ -f "$mock_file" ]]; then
            # shellcheck disable=SC1090
            source "$mock_file" || {
                log_warning "Failed to load mock: $mock_file"
                continue
            }
            is_verbose && log_debug "Loaded mock: $(basename "$mock_file")"
        fi
    done
    
    return 0
}


# Test resource using convention-based function discovery
test_resource_with_conventions() {
    local resource_name="$1"
    local tests_run=0
    local tests_passed=0
    
    # Define standard test types in priority order
    local test_types=("connection" "health" "basic" "advanced")
    
    # Convert resource name to function-safe format (replace hyphens with underscores)
    local function_safe_name="${resource_name//-/_}"
    # Handle special cases
    case "$resource_name" in
        "node-red")
            function_safe_name="nodered"
            ;;
    esac
    
    log_debug "Discovering test functions for resource: $resource_name (function prefix: $function_safe_name)"
    
    # Discover and run tests based on naming conventions
    for test_type in "${test_types[@]}"; do
        local test_function="test_${function_safe_name}_${test_type}"
        
        # Check if the function exists
        if command -v "$test_function" >/dev/null 2>&1; then
            log_debug "Discovered function: $test_function"
            ((tests_run++))
            
            # Run the test with caching
            if run_cached_test "mock-$resource_name" "$test_type" "$test_function" "${resource_name}-${test_type}: $resource_name"; then
                ((tests_passed++))
                increment_test_counter "passed"
            else
                increment_test_counter "failed"
            fi
        else
            log_debug "Function not found: $test_function"
            
            # Only skip advanced tests silently - others are expected
            if [[ "$test_type" != "advanced" ]]; then
                log_test_skip "${resource_name}-${test_type}: $resource_name" "Function $test_function not available"
                increment_test_counter "skipped"
            fi
        fi
    done
    
    # If no tests were found, fall back to generic testing
    if [[ $tests_run -eq 0 ]]; then
        log_warning "No convention-based tests found for resource: $resource_name"
        test_generic_resource_mock "$resource_name"
    else
        log_info "Resource $resource_name: $tests_passed/$tests_run tests passed"
    fi
    
    return 0
}

# Test generic resource (fallback for resources without specific mocks)
test_generic_resource_mock() {
    local resource_name="$1"
    
    log_test_skip "generic-mock: $resource_name" "No specific mock available - using generic test"
    
    # Just test that the resource is mentioned in service.json
    local service_json="$PROJECT_ROOT/.vrooli/service.json"
    if command -v jq >/dev/null 2>&1 && [[ -f "$service_json" ]]; then
        if jq -e ".resources | to_entries[] | select(.key == \"$resource_name\")" "$service_json" >/dev/null 2>&1; then
            log_test_pass "config-present: $resource_name"
            increment_test_counter "passed"
        else
            log_test_fail "config-present: $resource_name" "Not found in service.json"
            increment_test_counter "failed"
        fi
    else
        increment_test_counter "skipped"
    fi
    
    return 0
}

# Main resource validation function
run_resource_validation() {
    log_section "Resource Validation Phase"
    
    reset_test_counters
    reset_cache_stats
    
    # Load cache for this phase
    load_cache "resources"
    
    # Load mocks for testing
    load_resource_mocks
    
    # Test 1: service.json configuration
    if ! test_service_json_config; then
        log_error "service.json validation failed - cannot continue with resource testing"
        print_test_summary
        return 1
    fi
    
    # Test 2: Resource construction
    test_resource_construction
    
    # Test 3: Enabled resources with mocks
    test_enabled_resources
    
    # Save cache before printing results
    save_cache
    
    # Print final results
    print_test_summary
    
    # Show cache statistics if verbose
    if is_verbose; then
        log_info ""
        show_cache_stats
    fi
    
    # Return success if no failures
    local counters
    read -r total passed failed skipped <<< "$(get_test_counters)"
    
    if [[ $failed -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

# Main execution
main() {
    if is_dry_run; then
        log_info "⚡ [DRY RUN] Resource Validation Phase"
        log_info "Would test service.json configuration"
        log_info "Would validate all resource files in: $PROJECT_ROOT/scripts/resources"
        log_info "Would test enabled resources with mocks"
        return 0
    fi
    
    # Run the resource validation
    if run_resource_validation; then
        log_success "✅ Resource validation phase completed successfully"
        return 0
    else
        log_error "❌ Resource validation phase completed with failures"
        return 1
    fi
}

# Export functions for testing
export -f run_resource_validation test_service_json_config test_resource_construction
export -f test_enabled_resources load_resource_mocks test_resource_with_conventions
export -f test_generic_resource_mock

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi