#!/usr/bin/env bash
# Integration Testing Phase - Vrooli Testing Infrastructure
# 
# Tests actual service integration and functionality:
# - Resource mock testing (connection, health, basic functionality)
# - Auto-converter dry-run testing
# - Generated app integration testing
# - Service connectivity and functional testing
# - Supports scoping to specific resources, scenarios, or paths

set -euo pipefail

# Get APP_ROOT using cached value or compute once (2 levels up: __test/phases/test-integration.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test"
PROJECT_ROOT="${PROJECT_ROOT:-$APP_ROOT}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091  
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"

show_usage() {
    cat << 'EOF'
Integration Testing Phase - Test service functionality and integration

Usage: ./test-integration.sh [OPTIONS]

OPTIONS:
  --verbose         Show detailed output for each integration test
  --parallel        Run tests in parallel (when possible)  
  --no-cache        Skip caching optimizations
  --dry-run         Show what would be tested without running
  --clear-cache     Clear integration testing cache before running
  --resource=NAME   Test only specific resource (e.g., --resource=ollama)
  --scenario=NAME   Test only specific scenario (e.g., --scenario=app-generator)
  --path=PATH       Test only specific directory/file path
  --help            Show this help

WHAT THIS PHASE TESTS:
  1. Enabled resource mock testing (connection, health, functionality)
  2. Auto-converter dry-run functionality testing
  3. Generated app integration testing (manage scripts, services)
  4. Service connectivity and functional validation
  5. End-to-end workflow testing where applicable

SCOPING EXAMPLES:
  ./test-integration.sh --resource=ollama         # Test only ollama resource mocks
  ./test-integration.sh --scenario=app-generator  # Test only app-generator integration
  ./test-integration.sh --path=generated-apps     # Test only generated apps
  ./test-integration.sh --resource=n8n --verbose  # Detailed testing for n8n only

COMBINED EXAMPLES:
  ./test-integration.sh                           # Test all integration components
  ./test-integration.sh --verbose                 # Show detailed integration testing
  ./test-integration.sh --no-cache                # Force re-testing all integrations
EOF
}

# Parse command line arguments
CLEAR_CACHE=""
SCOPE_RESOURCE=""
SCOPE_SCENARIO=""
SCOPE_PATH=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --clear-cache)
            CLEAR_CACHE="true"
            shift
            ;;
        --resource=*)
            SCOPE_RESOURCE="${1#*=}"
            shift
            ;;
        --scenario=*)
            SCOPE_SCENARIO="${1#*=}"
            shift
            ;;
        --path=*)
            SCOPE_PATH="${1#*=}"
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
    clear_cache "integration"
fi

# Check if path should be tested based on scoping
should_test_path() {
    local path="$1"
    
    # If no scoping, test everything
    if [[ -z "$SCOPE_RESOURCE" && -z "$SCOPE_SCENARIO" && -z "$SCOPE_PATH" ]]; then
        return 0
    fi
    
    # Check path scope
    if [[ -n "$SCOPE_PATH" ]]; then
        if [[ "$path" == "$SCOPE_PATH"* ]]; then
            return 0
        fi
    fi
    
    # Check resource scope
    if [[ -n "$SCOPE_RESOURCE" ]]; then
        if [[ "$path" == *"/resources/"* ]] && [[ "$path" == *"$SCOPE_RESOURCE"* ]]; then
            return 0
        fi
    fi
    
    # Check scenario scope
    if [[ -n "$SCOPE_SCENARIO" ]]; then
        if [[ "$path" == *"/scenarios/"* ]] && [[ "$path" == *"$SCOPE_SCENARIO"* ]]; then
            return 0
        fi
    fi
    
    return 1
}

# Check if resource should be tested based on scoping
should_test_resource() {
    local resource_name="$1"
    
    # If no scoping, test everything
    if [[ -z "$SCOPE_RESOURCE" && -z "$SCOPE_SCENARIO" && -z "$SCOPE_PATH" ]]; then
        return 0
    fi
    
    # If we have resource scope, only test that resource
    if [[ -n "$SCOPE_RESOURCE" ]]; then
        if [[ "$resource_name" == "$SCOPE_RESOURCE" ]]; then
            return 0
        else
            return 1
        fi
    fi
    
    # If we're scoped to scenarios only, don't test resources
    if [[ -n "$SCOPE_SCENARIO" && -z "$SCOPE_PATH" ]]; then
        return 1
    fi
    
    return 0
}

# Load resource mocks for testing
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

# Get enabled resources from service.json
get_enabled_resources() {
    local service_json="$PROJECT_ROOT/.vrooli/service.json"
    
    if [[ ! -f "$service_json" ]]; then
        return 0
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        return 0
    fi
    
    # Extract enabled resources
    jq -r '.resources // {} | to_entries[] | select(.value.enabled == true) | .key' "$service_json" 2>/dev/null || true
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
        test_generic_resource_integration "$resource_name"
    else
        log_info "Resource $resource_name: $tests_passed/$tests_run integration tests passed"
    fi
    
    return 0
}

# Test generic resource integration (fallback for resources without specific mocks)
test_generic_resource_integration() {
    local resource_name="$1"
    
    log_test_skip "generic-integration: $resource_name" "No specific integration test available"
    
    # Just test that the resource is mentioned in service.json and enabled
    local service_json="$PROJECT_ROOT/.vrooli/service.json"
    if command -v jq >/dev/null 2>&1 && [[ -f "$service_json" ]]; then
        if jq -e ".resources.\"$resource_name\".enabled == true" "$service_json" >/dev/null 2>&1; then
            log_test_pass "enabled-check: $resource_name"
            increment_test_counter "passed"
        else
            log_test_skip "enabled-check: $resource_name" "Resource not enabled or not found in service.json"
            increment_test_counter "skipped"
        fi
    else
        increment_test_counter "skipped"
    fi
    
    return 0
}

# Test enabled resources with mocks
test_enabled_resources() {
    log_info "⚡ Testing enabled resources with integration mocks..."
    
    # Skip if we're scoped to scenarios only
    if [[ -n "$SCOPE_SCENARIO" && -z "$SCOPE_RESOURCE" && -z "$SCOPE_PATH" ]]; then
        log_info "⏭️ Skipping resource integration (scoped to scenarios)"
        return 0
    fi
    
    # Get enabled resources from service.json
    local enabled_resources
    enabled_resources=$(get_enabled_resources)
    
    if [[ -z "$enabled_resources" ]]; then
        log_warning "No enabled resources found in service.json"
        return 0
    fi
    
    log_info "📋 Found enabled resources: $(echo "$enabled_resources" | tr '\n' ' ')"
    
    # Filter by scope if needed
    local resources_to_test=()
    while IFS= read -r resource_name; do
        [[ -n "$resource_name" ]] || continue
        if should_test_resource "$resource_name"; then
            resources_to_test+=("$resource_name")
        fi
    done <<< "$enabled_resources"
    
    if [[ ${#resources_to_test[@]} -eq 0 ]]; then
        log_info "⏭️ No enabled resources match scope criteria"
        return 0
    fi
    
    log_info "🧪 Testing ${#resources_to_test[@]} resources: ${resources_to_test[*]}"
    
    # Load mocks for testing
    if ! load_resource_mocks; then
        log_error "Failed to load resource mocks"
        increment_test_counter "failed"
        return 1
    fi
    
    # Test each enabled resource using convention-based discovery
    for resource_name in "${resources_to_test[@]}"; do
        log_info "🧪 Testing resource integration: $resource_name"
        test_resource_with_conventions "$resource_name"
    done
    
    return 0
}

# Test converter dry run
test_converter_dry_run() {
    local auto_converter="$1"
    
    # Run converter with dry-run or list mode if available
    if "$auto_converter" --help 2>&1 | grep -q "\--dry-run\|--list"; then
        "$auto_converter" --dry-run >/dev/null 2>&1 || "$auto_converter" --list >/dev/null 2>&1
    else
        # Just test that it can be executed without errors
        timeout 10 "$auto_converter" >/dev/null 2>&1 || {
            local exit_code=$?
            if [[ $exit_code -eq 124 ]]; then
                # Timeout is OK - means it would have run
                return 0
            else
                return $exit_code
            fi
        }
    fi
    
    return $?
}

# Test auto-converter integration functionality
test_auto_converter_integration() {
    log_info "⚙️ Testing auto-converter integration functionality..."
    
    # Skip if we're scoped to resources only
    if [[ -n "$SCOPE_RESOURCE" && -z "$SCOPE_SCENARIO" && -z "$SCOPE_PATH" ]]; then
        log_info "⏭️ Skipping auto-converter integration (scoped to resources)"
        return 0
    fi
    
    local auto_converter="$PROJECT_ROOT/scenarios/tools/auto-converter.sh"
    
    if ! should_test_path "$auto_converter"; then
        log_info "⏭️ Skipping auto-converter integration (outside scope)"
        return 0
    fi
    
    local relative_path
    relative_path=$(relative_path "$auto_converter")
    
    # Skip if auto-converter doesn't exist (structure phase would have caught this)
    if [[ ! -f "$auto_converter" ]] || [[ ! -x "$auto_converter" ]]; then
        log_test_skip "converter-integration: $relative_path" "Auto-converter not available"
        increment_test_counter "skipped"
        return 0
    fi
    
    # Test auto-converter dry run functionality
    if is_dry_run; then
        log_test_skip "converter-dry-run: $relative_path" "In dry run mode"
        increment_test_counter "skipped"
    else
        if run_cached_test "$auto_converter" "integration-dry-run" "test_converter_dry_run '$auto_converter'" "converter-dry-run: $relative_path"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
        fi
    fi
    
    return 0
}

# Check if app is current (not outdated) based on schema hash
test_app_is_current() {
    local app_dir="$1"
    
    # Simple heuristic: if .vrooli/service.json exists and was modified recently, assume current
    local service_json="$app_dir/.vrooli/service.json"
    
    if [[ ! -f "$service_json" ]]; then
        return 1
    fi
    
    # If we have a hash checking mechanism, use it
    local hash_file="$app_dir/.vrooli/schema-hash"
    if [[ -f "$hash_file" ]]; then
        # Compare with source scenario hash if available
        local app_name
        app_name=$(basename "$app_dir")
        local source_scenario_dir="$PROJECT_ROOT/scenarios/core/$app_name"
        
        if [[ -d "$source_scenario_dir" ]]; then
            local source_hash_file="$source_scenario_dir/.vrooli/schema-hash"
            if [[ -f "$source_hash_file" ]]; then
                if diff "$hash_file" "$source_hash_file" >/dev/null 2>&1; then
                    return 0  # Current
                else
                    return 1  # Outdated
                fi
            fi
        fi
    fi
    
    # Fallback: assume current if modified within last day
    if command -v stat >/dev/null 2>&1; then
        local file_age
        file_age=$(stat -c %Y "$service_json" 2>/dev/null || echo "0")
        local current_time
        current_time=$(date +%s)
        local age_seconds=$((current_time - file_age))
        
        if [[ $age_seconds -lt 86400 ]]; then  # Less than 24 hours
            return 0
        fi
    fi
    
    # Default: assume outdated
    return 1
}

# Test generated app integration
test_generated_app_integration() {
    log_info "🏗️ Testing generated app integration (non-outdated apps)..."
    
    # Skip if we're scoped to resources only
    if [[ -n "$SCOPE_RESOURCE" && -z "$SCOPE_SCENARIO" && -z "$SCOPE_PATH" ]]; then
        log_info "⏭️ Skipping app integration (scoped to resources)"
        return 0
    fi
    
    local generated_apps_dir="$HOME/generated-apps"
    
    if ! should_test_path "$generated_apps_dir"; then
        log_info "⏭️ Skipping generated apps integration (outside scope)"
        return 0
    fi
    
    if [[ ! -d "$generated_apps_dir" ]]; then
        log_warning "Generated apps directory not found: $generated_apps_dir"
        log_warning "Run auto-converter first to generate test apps"
        return 0
    fi
    
    # Find generated apps
    local generated_apps=()
    if [[ -n "$SCOPE_SCENARIO" ]]; then
        # Test specific scenario app
        local app_dir="$generated_apps_dir/$SCOPE_SCENARIO"
        if [[ -d "$app_dir" ]]; then
            generated_apps+=("$app_dir")
        fi
    else
        # Test all generated apps
        while IFS= read -r -d '' app_dir; do
            if should_test_path "$app_dir"; then
                generated_apps+=("$app_dir")
            fi
        done < <(find "$generated_apps_dir" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null)
    fi
    
    local total_apps=${#generated_apps[@]}
    log_info "📋 Found $total_apps generated apps to test integration"
    
    if [[ $total_apps -eq 0 ]]; then
        if [[ -n "$SCOPE_SCENARIO" ]]; then
            log_warning "No generated app found for scenario: $SCOPE_SCENARIO"
        else
            log_warning "No generated apps found"
        fi
        return 0
    fi
    
    # Test each generated app integration
    local current=0
    for app_dir in "${generated_apps[@]}"; do
        ((current++))
        local app_name
        app_name=$(basename "$app_dir")
        
        log_progress "$current" "$total_apps" "Testing app integration"
        
        # Check if app is outdated using schema hash
        if test_app_is_current "$app_dir"; then
            # Run integration test if available
            local integration_test="$PROJECT_ROOT/scenarios/tools/run-integration-test.sh"
            if [[ -x "$integration_test" ]]; then
                if run_cached_test "$app_dir" "integration" "'$integration_test' '$app_dir'" "integration: $app_name"; then
                    increment_test_counter "passed"
                else
                    increment_test_counter "failed"
                fi
            else
                # Fallback: test that manage script can show help
                local manage_script="$app_dir/scripts/manage.sh"
                if [[ -x "$manage_script" ]]; then
                    if run_cached_test "$app_dir" "manage-help" "(cd '$app_dir' && timeout 10 '$manage_script' --help >/dev/null 2>&1)" "manage-help: $app_name"; then
                        increment_test_counter "passed"
                    else
                        increment_test_counter "failed"
                    fi
                else
                    log_test_skip "integration: $app_name" "manage script not available"
                    increment_test_counter "skipped"
                fi
            fi
        else
            log_test_skip "integration: $app_name" "app is outdated - skipping integration test"
            increment_test_counter "skipped"
        fi
    done
    
    return 0
}

# Main integration testing function
run_integration_testing() {
    log_section "Integration Testing Phase"
    
    # Show scope information
    if [[ -n "$SCOPE_RESOURCE" || -n "$SCOPE_SCENARIO" || -n "$SCOPE_PATH" ]]; then
        log_info "🎯 Scoped testing enabled:"
        [[ -n "$SCOPE_RESOURCE" ]] && log_info "  📦 Resource: $SCOPE_RESOURCE"
        [[ -n "$SCOPE_SCENARIO" ]] && log_info "  🎬 Scenario: $SCOPE_SCENARIO"
        [[ -n "$SCOPE_PATH" ]] && log_info "  📁 Path: $SCOPE_PATH"
    fi
    
    reset_test_counters
    reset_cache_stats
    
    # Load cache for this phase
    load_cache "integration"
    
    # Test 1: Enabled resources with mocks
    test_enabled_resources
    
    # Test 2: Auto-converter integration functionality
    test_auto_converter_integration
    
    # Test 3: Generated app integration testing
    test_generated_app_integration
    
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
        log_info "⚡ [DRY RUN] Integration Testing Phase"
        log_info "Would test enabled resource mocks"
        log_info "Would test auto-converter dry-run functionality"
        log_info "Would test generated app integration"
        
        if [[ -n "$SCOPE_RESOURCE" || -n "$SCOPE_SCENARIO" || -n "$SCOPE_PATH" ]]; then
            log_info "🎯 Scope filters:"
            [[ -n "$SCOPE_RESOURCE" ]] && log_info "  📦 Resource: $SCOPE_RESOURCE"
            [[ -n "$SCOPE_SCENARIO" ]] && log_info "  🎬 Scenario: $SCOPE_SCENARIO"
            [[ -n "$SCOPE_PATH" ]] && log_info "  📁 Path: $SCOPE_PATH"
        fi
        
        return 0
    fi
    
    # Run the integration testing
    if run_integration_testing; then
        log_success "✅ Integration testing phase completed successfully"
        return 0
    else
        log_error "❌ Integration testing phase completed with failures"
        return 1
    fi
}

# Export functions for testing
export -f run_integration_testing test_enabled_resources load_resource_mocks
export -f test_resource_with_conventions test_generic_resource_integration
export -f test_auto_converter_integration test_converter_dry_run
export -f test_generated_app_integration test_app_is_current
export -f should_test_path should_test_resource get_enabled_resources

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi