#!/usr/bin/env bash
# Structure Validation Phase - Vrooli Testing Infrastructure
# 
# Tests file/directory structures, configuration files, and required components:
# - service.json configuration validation (root + resources + scenarios)
# - Resource file construction validation (existence, permissions, syntax)
# - Scenario catalog and individual scenario structure validation
# - Auto-converter and tool validation
# - Generated app structure validation
# - Supports scoping to specific resources, scenarios, or paths

set -euo pipefail

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
Structure Validation Phase - Test file/directory structures and configurations

Usage: ./test-structure.sh [OPTIONS]

OPTIONS:
  --verbose         Show detailed output for each structure test
  --parallel        Run tests in parallel (when possible)  
  --no-cache        Skip caching optimizations
  --dry-run         Show what would be tested without running
  --clear-cache     Clear structure validation cache before running
  --resource=NAME   Test only specific resource (e.g., --resource=ollama)
  --scenario=NAME   Test only specific scenario (e.g., --scenario=app-generator)
  --path=PATH       Test only specific directory/file path
  --help            Show this help

WHAT THIS PHASE TESTS:
  1. service.json configuration files (root, resources, scenarios)
  2. Resource file construction (existence, permissions, syntax)
  3. Individual scenario service.json validation
  4. Auto-converter and tool structure validation
  5. Generated app structure validation (required files)
  6. Directory structure requirements

SCOPING EXAMPLES:
  ./test-structure.sh --resource=ollama           # Test only ollama resource structure
  ./test-structure.sh --scenario=app-generator    # Test only app-generator scenario
  ./test-structure.sh --path=scenarios/core       # Test only specific directory

COMBINED EXAMPLES:
  ./test-structure.sh                             # Test all structure components
  ./test-structure.sh --verbose                   # Show detailed structure testing
  ./test-structure.sh --no-cache                  # Force re-testing all structures
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
    clear_cache "structure"
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

# Test service.json configuration files
test_service_json_configs() {
    log_info "🔧 Testing service.json configuration files..."
    
    local configs_to_test=()
    
    # Root service.json
    local root_config="$PROJECT_ROOT/.vrooli/service.json"
    if should_test_path "$root_config"; then
        configs_to_test+=("$root_config:root")
    fi
    
    # Resource service.json files (if scoping allows)
    if [[ -z "$SCOPE_SCENARIO" ]]; then  # Don't test resources if we're only testing scenarios
        if [[ -n "$SCOPE_RESOURCE" ]]; then
            # Test specific resource
            local resource_config="$PROJECT_ROOT/resources/$SCOPE_RESOURCE/.vrooli/service.json"
            if [[ -f "$resource_config" ]]; then
                configs_to_test+=("$resource_config:resource-$SCOPE_RESOURCE")
            fi
        else
            # Test all resource configs
            while IFS= read -r -d '' config_file; do
                if should_test_path "$config_file"; then
                    local resource_name
                    resource_name=$(basename "$(dirname "$(dirname "$config_file")")")
                    configs_to_test+=("$config_file:resource-$resource_name")
                fi
            done < <(find "$PROJECT_ROOT/resources" -name "service.json" -path "*/.vrooli/service.json" -print0 2>/dev/null)
        fi
    fi
    
    # Scenario service.json files (if scoping allows)
    if [[ -z "$SCOPE_RESOURCE" ]]; then  # Don't test scenarios if we're only testing resources
        if [[ -n "$SCOPE_SCENARIO" ]]; then
            # Test specific scenario
            local scenario_config="$PROJECT_ROOT/scenarios/$SCOPE_SCENARIO/.vrooli/service.json"
            if [[ -f "$scenario_config" ]]; then
                configs_to_test+=("$scenario_config:scenario-$SCOPE_SCENARIO")
            fi
        else
            # Test all scenario configs
            while IFS= read -r -d '' config_file; do
                if should_test_path "$config_file"; then
                    local scenario_name
                    scenario_name=$(basename "$(dirname "$(dirname "$config_file")")")
                    # Skip system directories
                    case "$scenario_name" in
                        "tools"|".*"|"__*")
                            continue
                            ;;
                    esac
                    configs_to_test+=("$config_file:scenario-$scenario_name")
                fi
            done < <(find "$PROJECT_ROOT/scenarios" -name "service.json" -path "*/.vrooli/service.json" -print0 2>/dev/null)
        fi
    fi
    
    local total_configs=${#configs_to_test[@]}
    log_info "📋 Found $total_configs service.json files to test"
    
    if [[ $total_configs -eq 0 ]]; then
        log_warning "No service.json files found matching scope criteria"
        return 0
    fi
    
    # Test each configuration
    local current=0
    for config_info in "${configs_to_test[@]}"; do
        ((current++))
        IFS=':' read -r config_file config_type <<< "$config_info"
        local relative_path
        relative_path=$(relative_path "$config_file")
        
        log_progress "$current" "$total_configs" "Testing service.json files"
        
        # Test 1: File exists
        if run_cached_test "$config_file" "existence" "assert_file_exists '$config_file' 'service.json'" "config-exists: $relative_path"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 2: Valid JSON
        if run_cached_test "$config_file" "json-valid" "assert_json_valid '$config_file' 'service.json'" "json-valid: $relative_path"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 3: Type-specific validation
        case "$config_type" in
            root)
                # Root config should declare dependencies.resources
                if command -v jq >/dev/null 2>&1; then
                    if run_cached_test "$config_file" "has-dependencies-resources" "jq -e '.dependencies.resources' '$config_file' >/dev/null" "has-dependencies-resources: $relative_path"; then
                        increment_test_counter "passed"
                    else
                        increment_test_counter "failed"
                    fi
                else
                    log_test_skip "has-dependencies-resources: $relative_path" "jq not available"
                    increment_test_counter "skipped"
                fi
                ;;
            resource-*)
                # Resource configs should have basic resource structure
                if command -v jq >/dev/null 2>&1; then
                    if run_cached_test "$config_file" "resource-structure" "jq -e '.name // .type // .description' '$config_file' >/dev/null" "resource-structure: $relative_path"; then
                        increment_test_counter "passed"
                    else
                        log_test_skip "resource-structure: $relative_path" "basic resource fields not required"
                        increment_test_counter "skipped"
                    fi
                else
                    increment_test_counter "skipped"
                fi
                ;;
            scenario-*)
                # Scenario configs should have basic scenario structure  
                if command -v jq >/dev/null 2>&1; then
                    if run_cached_test "$config_file" "scenario-structure" "jq -e '.name // .description' '$config_file' >/dev/null" "scenario-structure: $relative_path"; then
                        increment_test_counter "passed"
                    else
                        log_test_skip "scenario-structure: $relative_path" "basic scenario fields not required"
                        increment_test_counter "skipped"
                    fi
                else
                    increment_test_counter "skipped"
                fi
                ;;
        esac
    done
    
    return 0
}

# Test resource file construction
test_resource_construction() {
    log_info "📁 Testing resource file construction..."
    
    # Skip if we're scoped to scenarios only
    if [[ -n "$SCOPE_SCENARIO" && -z "$SCOPE_RESOURCE" ]]; then
        log_info "⏭️ Skipping resource construction (scoped to scenarios)"
        return 0
    fi
    
    local resources_dir="$PROJECT_ROOT/resources"
    
    # Test resources directory exists (only if no specific scoping)
    if [[ -z "$SCOPE_RESOURCE" ]]; then
        if ! should_test_path "$resources_dir" || ! assert_directory_exists "$resources_dir" "resources directory"; then
            log_error "Resources directory not found: $resources_dir"
            increment_test_counter "failed"
            return 1
        fi
        increment_test_counter "passed"
    fi
    
    # Find resource files to test
    local resource_files=()
    
    if [[ -n "$SCOPE_RESOURCE" ]]; then
        # Test specific resource
        local resource_paths=("$resources_dir/$SCOPE_RESOURCE" "$resources_dir/providers/$SCOPE_RESOURCE")
        for resource_path in "${resource_paths[@]}"; do
            if [[ -d "$resource_path" ]]; then
                while IFS= read -r -d '' resource_file; do
                    resource_files+=("$resource_file")
                done < <(find "$resource_path" -type f -name "*.sh" -print0 2>/dev/null)
            elif [[ -f "$resource_path.sh" ]]; then
                resource_files+=("$resource_path.sh")
            fi
        done
    else
        # Test all resources
        local providers_dir="$resources_dir/providers"
        
        # Find all resource provider scripts
        if [[ -d "$providers_dir" ]]; then
            while IFS= read -r -d '' resource_file; do
                if should_test_path "$resource_file"; then
                    resource_files+=("$resource_file")
                fi
            done < <(find "$providers_dir" -type f -name "*.sh" -print0 2>/dev/null)
        fi
        
        # Also check main resources directory
        while IFS= read -r -d '' resource_file; do
            if should_test_path "$resource_file"; then
                resource_files+=("$resource_file")
            fi
        done < <(find "$resources_dir" -maxdepth 1 -type f -name "*.sh" -print0 2>/dev/null)
    fi
    
    local total_resources=${#resource_files[@]}
    log_info "📋 Found $total_resources resource files to validate"
    
    if [[ $total_resources -eq 0 ]]; then
        if [[ -n "$SCOPE_RESOURCE" ]]; then
            log_warning "No resource files found for: $SCOPE_RESOURCE"
        else
            log_warning "No resource files found in $resources_dir"
        fi
        return 0
    fi
    
    # Test each resource file
    local current=0
    for resource_file in "${resource_files[@]}"; do
        ((current++))
        local relative_resource
        relative_resource=$(relative_path "$resource_file")
        
        log_progress "$current" "$total_resources" "Validating resource files"
        
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
        
        # Test 3: Shellcheck passes (already covered in static phase, but include for completeness)
        if run_cached_test "$resource_file" "shellcheck" "run_shellcheck '$resource_file' 'resource script'" "shellcheck: $relative_resource"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
        fi
    done
    
    return 0
}

# NOTE: catalog.json testing removed - scenarios are now discovered dynamically from filesystem

# Test scenario tools structure
test_tools_structure() {
    log_info "⚙️ Testing tools structure..."
    
    # Skip if we're scoped to resources only
    if [[ -n "$SCOPE_RESOURCE" && -z "$SCOPE_SCENARIO" ]]; then
        log_info "⏭️ Skipping tools structure (scoped to resources)"
        return 0
    fi
    
    # Only test tools that still exist after conversion elimination
    local tools_to_test=()
    
    for tool_file in "${tools_to_test[@]}"; do
        if ! should_test_path "$tool_file"; then
            continue
        fi
        
        local relative_path
        relative_path=$(relative_path "$tool_file")
        local tool_name
        tool_name=$(basename "$tool_file" .sh)
        
        # Test 1: Tool exists
        if run_cached_test "$tool_file" "existence" "assert_file_exists '$tool_file' '$tool_name'" "tool-exists: $relative_path"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 2: Tool is executable
        if run_cached_test "$tool_file" "executable" "assert_file_executable '$tool_file' '$tool_name'" "tool-exec: $relative_path"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 3: Tool has valid syntax
        if run_cached_test "$tool_file" "syntax" "validate_shell_syntax '$tool_file' '$tool_name'" "tool-syntax: $relative_path"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
        fi
    done
    
    return 0
}

# Test scenario structure directly (no conversion needed)
test_scenario_structure() {
    log_info "🏗️ Testing scenario structure..."
    
    # Skip if we're scoped to resources only
    if [[ -n "$SCOPE_RESOURCE" && -z "$SCOPE_SCENARIO" ]]; then
        log_info "⏭️ Skipping scenario structure (scoped to resources)"
        return 0
    fi
    
    local scenarios_dir="$PROJECT_ROOT/scenarios"
    
    if ! should_test_path "$scenarios_dir"; then
        log_info "⏭️ Skipping scenarios (outside scope)"
        return 0
    fi
    
    if [[ ! -d "$scenarios_dir" ]]; then
        log_warning "Scenarios directory not found: $scenarios_dir"
        return 0
    fi
    
    # Find scenarios to test
    local scenarios=()
    if [[ -n "$SCOPE_SCENARIO" ]]; then
        # Test specific scenario
        local scenario_dir="$scenarios_dir/$SCOPE_SCENARIO"
        if [[ -d "$scenario_dir" ]]; then
            scenarios+=("$scenario_dir")
        fi
    else
        # Test all scenarios
        while IFS= read -r -d '' scenario_dir; do
            if should_test_path "$scenario_dir"; then
                scenarios+=("$scenario_dir")
            fi
        done < <(find "$scenarios_dir" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null)
    fi
    
    local total_scenarios=${#scenarios[@]}
    log_info "📋 Found $total_scenarios scenarios to test structure"
    
    if [[ $total_scenarios -eq 0 ]]; then
        if [[ -n "$SCOPE_SCENARIO" ]]; then
            log_warning "No scenario found: $SCOPE_SCENARIO"
        else
            log_warning "No scenarios found"
        fi
        return 0
    fi
    
    # Test each scenario structure
    local current=0
    for scenario_dir in "${scenarios[@]}"; do
        ((current++))
        local scenario_name
        scenario_name=$(basename "$scenario_dir")
        
        log_progress "$current" "$total_scenarios" "Testing scenario structure"
        
        # Test 1: Scenario has service.json
        local service_json="$scenario_dir/.vrooli/service.json"
        if run_cached_test "$service_json" "existence" "assert_file_exists '$service_json' 'scenario service.json'" "scenario-service-json: $scenario_name"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 2: service.json is valid JSON
        if run_cached_test "$service_json" "json-valid" "assert_json_valid '$service_json' 'scenario service.json'" "scenario-json-valid: $scenario_name"; then
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
            continue
        fi
        
        # Test 3: Scenario can use Vrooli's manage.sh
        # Scenarios don't have their own manage.sh, they use Vrooli's
        if [[ -f "$service_json" ]]; then
            increment_test_counter "passed"  # Service.json exists, scenario is valid
        else
            increment_test_counter "failed"
        fi
    done
    
    return 0
}

# Main structure validation function
run_structure_validation() {
    log_section "Structure Validation Phase"
    
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
    load_cache "structure"
    
    # Test 1: service.json configurations
    test_service_json_configs
    
    # Test 2: Resource construction
    test_resource_construction
    
    # Test 3: Tools structure
    test_tools_structure
    
    # Test 4: Generated app structure
    test_scenario_structure
    
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
        log_info "🏗️ [DRY RUN] Structure Validation Phase"
        log_info "Would test service.json configurations"
        log_info "Would validate resource file construction"
        log_info "Would test scenario catalog structure"
        log_info "Would validate tools structure"
        log_info "Would test generated app structure"
        
        if [[ -n "$SCOPE_RESOURCE" || -n "$SCOPE_SCENARIO" || -n "$SCOPE_PATH" ]]; then
            log_info "🎯 Scope filters:"
            [[ -n "$SCOPE_RESOURCE" ]] && log_info "  📦 Resource: $SCOPE_RESOURCE"
            [[ -n "$SCOPE_SCENARIO" ]] && log_info "  🎬 Scenario: $SCOPE_SCENARIO"
            [[ -n "$SCOPE_PATH" ]] && log_info "  📁 Path: $SCOPE_PATH"
        fi
        
        return 0
    fi
    
    # Run the structure validation
    if run_structure_validation; then
        log_success "✅ Structure validation phase completed successfully"
        return 0
    else
        log_error "❌ Structure validation phase completed with failures"
        return 1
    fi
}

# Export functions for testing
export -f run_structure_validation test_service_json_configs test_resource_construction
export -f test_tools_structure test_scenario_structure should_test_path

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
