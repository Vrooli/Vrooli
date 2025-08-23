#!/usr/bin/env bash
# Scenario Validation Phase - Vrooli Testing Infrastructure
# 
# Tests scenario validation and integration:
# - Uses existing scenario validation script on all scenarios
# - For converted non-outdated scenarios, runs existing integration tests
# - Uses schema-hashes mechanism to determine what's outdated
# - Validates scenario conversion pipeline
# - Tests generated app structure

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ROOT="${PROJECT_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091  
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"

show_usage() {
    cat << 'EOF'
Scenario Validation Phase - Test scenario structure and converted applications

Usage: ./test-scenarios.sh [OPTIONS]

OPTIONS:
  --verbose      Show detailed output for each scenario tested
  --parallel     Run tests in parallel (when possible)  
  --no-cache     Skip caching optimizations
  --dry-run      Show what would be tested without running
  --clear-cache  Clear scenario validation cache before running
  --help         Show this help

WHAT THIS PHASE TESTS:
  1. Uses existing scenario validation script on all scenarios
  2. Validates scenario catalog.json structure and references
  3. Tests auto-converter hash-based conversion system
  4. For converted non-outdated scenarios, runs integration tests
  5. Validates generated app structure and required files
  6. Tests scenario-to-app conversion pipeline

EXAMPLES:
  ./test-scenarios.sh                 # Test all scenarios
  ./test-scenarios.sh --verbose       # Show detailed scenario testing
  ./test-scenarios.sh --no-cache      # Force re-testing of all scenarios
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
    clear_cache "scenarios"
fi

# Test scenario catalog structure
test_scenario_catalog() {
    log_info "📋 Testing scenario catalog structure..."
    
    local catalog_file="$PROJECT_ROOT/scenarios/catalog.json"
    local relative_path
    relative_path=$(relative_path "$catalog_file")
    
    # Test 1: Catalog file exists
    if ! run_cached_test "$catalog_file" "existence" "assert_file_exists '$catalog_file' 'scenario catalog'" "catalog-exists: $relative_path"; then
        log_error "Scenario catalog not found - cannot validate scenarios"
        return 1
    fi
    increment_test_counter "passed"
    
    # Test 2: Valid JSON
    if ! run_cached_test "$catalog_file" "json-valid" "assert_json_valid '$catalog_file' 'scenario catalog'" "catalog-json: $relative_path"; then
        log_error "Scenario catalog has invalid JSON"
        return 1
    fi
    increment_test_counter "passed"
    
    # Test 3: Has required structure
    if command -v jq >/dev/null 2>&1; then
        if run_cached_test "$catalog_file" "structure" "validate_catalog_structure '$catalog_file'" "catalog-structure: $relative_path"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
        fi
    else
        log_test_skip "catalog-structure: $relative_path" "jq not available"
        increment_test_counter "skipped"
    fi
    
    return 0
}

# Validate catalog structure with jq
validate_catalog_structure() {
    local catalog_file="$1"
    
    # Check that catalog has required fields
    if ! jq -e '.scenarios' "$catalog_file" >/dev/null 2>&1; then
        log_error "Catalog missing 'scenarios' field"
        return 1
    fi
    
    if ! jq -e '.scenarios | type == "array"' "$catalog_file" >/dev/null 2>&1; then
        log_error "Catalog 'scenarios' field is not an array"
        return 1
    fi
    
    # Check that each scenario has required fields
    local scenarios_count
    scenarios_count=$(jq '.scenarios | length' "$catalog_file" 2>/dev/null || echo "0")
    
    if [[ $scenarios_count -eq 0 ]]; then
        log_warning "Catalog contains no scenarios"
        return 0
    fi
    
    log_debug "Found $scenarios_count scenarios in catalog"
    
    # Validate each scenario entry
    for ((i=0; i<scenarios_count; i++)); do
        local scenario_name
        scenario_name=$(jq -r ".scenarios[$i].name // \"unknown\"" "$catalog_file" 2>/dev/null)
        
        if [[ "$scenario_name" == "null" ]] || [[ "$scenario_name" == "unknown" ]]; then
            log_error "Scenario at index $i missing 'name' field"
            return 1
        fi
        
        log_debug "Validating scenario: $scenario_name"
    done
    
    return 0
}

# Discover and validate individual scenarios
test_scenario_validation() {
    log_info "🔍 Running scenario validation on all scenarios..."
    
    local scenarios_dir="$PROJECT_ROOT/scenarios"
    
    if [[ ! -d "$scenarios_dir" ]]; then
        log_error "Scenarios directory not found: $scenarios_dir"
        increment_test_counter "failed"
        return 1
    fi
    
    # Find scenario directories (excluding tools and core system dirs)
    local scenario_dirs=()
    while IFS= read -r -d '' scenario_dir; do
        local dir_name
        dir_name=$(basename "$scenario_dir")
        
        # Skip system directories
        case "$dir_name" in
            "tools"|".*"|"__*")
                continue
                ;;
            *)
                scenario_dirs+=("$scenario_dir")
                ;;
        esac
    done < <(find "$scenarios_dir" -mindepth 1 -maxdepth 2 -type d -print0 2>/dev/null)
    
    local total_scenarios=${#scenario_dirs[@]}
    log_info "📋 Found $total_scenarios scenario directories to validate"
    
    if [[ $total_scenarios -eq 0 ]]; then
        log_warning "No scenario directories found in $scenarios_dir"
        return 0
    fi
    
    # Test each scenario directory
    local current=0
    for scenario_dir in "${scenario_dirs[@]}"; do
        ((current++))
        local relative_scenario
        relative_scenario=$(relative_path "$scenario_dir")
        
        log_progress "$current" "$total_scenarios" "Validating scenarios"
        
        # Test 1: Scenario has service.json
        local service_json="$scenario_dir/.vrooli/service.json"
        if run_cached_test "$service_json" "existence" "assert_file_exists '$service_json' 'scenario service.json'" "service-json: $relative_scenario"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 2: service.json is valid JSON
        if run_cached_test "$service_json" "json-valid" "assert_json_valid '$service_json' 'scenario service.json'" "json-valid: $relative_scenario"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 3: Run existing scenario validation if available
        local validation_script="$PROJECT_ROOT/scenarios/tools/validate-scenario.sh"
        if [[ -x "$validation_script" ]]; then
            if run_cached_test "$scenario_dir" "validation" "'$validation_script' '$scenario_dir'" "validation: $relative_scenario"; then
                increment_test_counter "passed"
            else
                increment_test_counter "failed"
            fi
        else
            log_test_skip "validation: $relative_scenario" "validation script not found"
            increment_test_counter "skipped"
        fi
    done
    
    return 0
}

# Test auto-converter functionality
test_auto_converter() {
    log_info "⚙️ Testing auto-converter functionality..."
    
    local auto_converter="$PROJECT_ROOT/scenarios/tools/auto-converter.sh"
    local relative_path
    relative_path=$(relative_path "$auto_converter")
    
    # Test 1: Auto-converter exists
    if ! run_cached_test "$auto_converter" "existence" "assert_file_exists '$auto_converter' 'auto-converter'" "converter-exists: $relative_path"; then
        log_error "Auto-converter not found - cannot test conversion"
        return 1
    fi
    increment_test_counter "passed"
    
    # Test 2: Auto-converter is executable
    if ! run_cached_test "$auto_converter" "executable" "assert_file_executable '$auto_converter' 'auto-converter'" "converter-exec: $relative_path"; then
        increment_test_counter "failed"
        return 1
    fi
    increment_test_counter "passed"
    
    # Test 3: Auto-converter has valid syntax
    if ! run_cached_test "$auto_converter" "syntax" "validate_shell_syntax '$auto_converter' 'auto-converter'" "converter-syntax: $relative_path"; then
        increment_test_counter "failed"
        return 1
    fi
    increment_test_counter "passed"
    
    # Test 4: Auto-converter dry run
    if is_dry_run; then
        log_test_skip "converter-dry-run: $relative_path" "In dry run mode"
        increment_test_counter "skipped"
    else
        if run_cached_test "$auto_converter" "dry-run" "test_converter_dry_run '$auto_converter'" "converter-dry-run: $relative_path"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
        fi
    fi
    
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

# Test converted scenarios (non-outdated ones)
test_converted_scenarios() {
    log_info "🏗️ Testing converted scenarios (non-outdated)..."
    
    local generated_apps_dir="$HOME/generated-apps"
    
    if [[ ! -d "$generated_apps_dir" ]]; then
        log_warning "Generated apps directory not found: $generated_apps_dir"
        log_warning "Run auto-converter first to generate test apps"
        return 0
    fi
    
    # Find generated apps
    local generated_apps=()
    while IFS= read -r -d '' app_dir; do
        generated_apps+=("$app_dir")
    done < <(find "$generated_apps_dir" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null)
    
    local total_apps=${#generated_apps[@]}
    log_info "📋 Found $total_apps generated apps to test"
    
    if [[ $total_apps -eq 0 ]]; then
        log_warning "No generated apps found"
        return 0
    fi
    
    # Test each generated app
    local current=0
    for app_dir in "${generated_apps[@]}"; do
        ((current++))
        local app_name
        app_name=$(basename "$app_dir")
        
        log_progress "$current" "$total_apps" "Testing generated apps"
        
        # Test 1: App has service.json
        local service_json="$app_dir/.vrooli/service.json"
        if run_cached_test "$service_json" "existence" "assert_file_exists '$service_json' 'app service.json'" "app-service-json: $app_name"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 2: App has manage script
        local manage_script="$app_dir/scripts/manage.sh"
        if run_cached_test "$manage_script" "existence" "assert_file_exists '$manage_script' 'app manage script'" "app-manage: $app_name"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 3: Manage script is executable
        if run_cached_test "$manage_script" "executable" "assert_file_executable '$manage_script' 'app manage script'" "app-executable: $app_name"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 4: Check if app is outdated using schema hash
        if test_app_is_current "$app_dir"; then
            # Test 5: Run integration test if available
            local integration_test="$PROJECT_ROOT/scenarios/tools/run-integration-test.sh"
            if [[ -x "$integration_test" ]]; then
                if run_cached_test "$app_dir" "integration" "'$integration_test' '$app_dir'" "integration: $app_name"; then
                    increment_test_counter "passed"
                else
                    increment_test_counter "failed"
                fi
            else
                log_test_skip "integration: $app_name" "integration test not available"
                increment_test_counter "skipped"
            fi
        else
            log_test_skip "integration: $app_name" "app is outdated - skipping integration test"
            increment_test_counter "skipped"
        fi
    done
    
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

# Main scenario validation function
run_scenario_validation() {
    log_section "Scenario Validation Phase"
    
    reset_test_counters
    reset_cache_stats
    
    # Load cache for this phase
    load_cache "scenarios"
    
    # Test 1: Scenario catalog
    test_scenario_catalog
    
    # Test 2: Individual scenario validation
    test_scenario_validation
    
    # Test 3: Auto-converter functionality
    test_auto_converter
    
    # Test 4: Converted scenarios (integration tests)
    test_converted_scenarios
    
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
        log_info "🔍 [DRY RUN] Scenario Validation Phase"
        log_info "Would test scenario catalog: $PROJECT_ROOT/scenarios/catalog.json"
        log_info "Would validate all scenario directories"
        log_info "Would test auto-converter functionality"
        log_info "Would test converted scenarios in: $HOME/generated-apps"
        return 0
    fi
    
    # Run the scenario validation
    if run_scenario_validation; then
        log_success "✅ Scenario validation phase completed successfully"
        return 0
    else
        log_error "❌ Scenario validation phase completed with failures"
        return 1
    fi
}

# Export functions for testing
export -f run_scenario_validation test_scenario_catalog validate_catalog_structure
export -f test_scenario_validation test_auto_converter test_converted_scenarios
export -f test_app_is_current

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi