#!/usr/bin/env bats
# Resource Interface Validation Tests
# Tests that all resources follow standardized interface patterns

# Load test infrastructure
load "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load interface validation library
source "${BATS_TEST_DIRNAME}/../tests/lib/interface-validation.sh"

# Test configuration
RESOURCES_DIR="${BATS_TEST_DIRNAME}/.."

setup() {
    # Basic test setup
    export RESOURCES_DIR
}

#######################################
# Helper functions
#######################################

# Get all resource directories
get_all_resources() {
    find "$RESOURCES_DIR" -mindepth 2 -maxdepth 2 -type d | \
        grep -E "(ai|automation|storage|execution|agents|search)/" | \
        sort
}

# Get resources by category
get_resources_by_category() {
    local category="$1"
    find "$RESOURCES_DIR/$category" -mindepth 1 -maxdepth 1 -type d | sort
}

#######################################
# Individual Interface Component Tests
#######################################

@test "all resources have executable manage.sh" {
    local failed_resources=()
    local passed_count=0
    
    while IFS= read -r resource_path; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        if validate_manage_script "$resource_path"; then
            ((passed_count++))
        else
            failed_resources+=("$resource_name")
        fi
    done < <(get_all_resources)
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "Resources missing or with non-executable manage.sh:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count resources have valid manage.sh scripts"
}

@test "all resources have required config files" {
    local failed_resources=()
    local passed_count=0
    
    while IFS= read -r resource_path; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        if validate_config_structure "$resource_path" >/dev/null 2>&1; then
            ((passed_count++))
        else
            failed_resources+=("$resource_name")
        fi
    done < <(get_all_resources)
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "Resources missing required config files:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count resources have valid config structure"
}

@test "all resources support --help flag" {
    local failed_resources=()
    local passed_count=0
    
    while IFS= read -r resource_path; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        if validate_function_help "$resource_path" >/dev/null 2>&1; then
            ((passed_count++))
        else
            failed_resources+=("$resource_name")
        fi
    done < <(get_all_resources)
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "Resources with poor or missing --help support:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count resources have proper --help support"
}

@test "all resources have adequate error handling" {
    local failed_resources=()
    local passed_count=0
    
    while IFS= read -r resource_path; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        if validate_error_handling "$resource_path" >/dev/null 2>&1; then
            ((passed_count++))
        else
            failed_resources+=("$resource_name")
        fi
    done < <(get_all_resources)
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "Resources with inadequate error handling:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count resources have adequate error handling"
}

#######################################
# Category-Specific Function Tests
#######################################

@test "AI resources have required functions" {
    local ai_resources
    mapfile -t ai_resources < <(get_resources_by_category "ai")
    
    local failed_resources=()
    local passed_count=0
    
    for resource_path in "${ai_resources[@]}"; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        if validate_required_functions "$resource_path" "ai" >/dev/null 2>&1; then
            ((passed_count++))
        else
            failed_resources+=("$resource_name")
        fi
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "AI resources missing required functions:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count AI resources have required functions"
}

@test "Automation resources have required functions" {
    local automation_resources
    mapfile -t automation_resources < <(get_resources_by_category "automation")
    
    [[ ${#automation_resources[@]} -eq 0 ]] && skip "No automation resources found"
    
    local failed_resources=()
    local passed_count=0
    
    for resource_path in "${automation_resources[@]}"; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        if validate_required_functions "$resource_path" "automation" >/dev/null 2>&1; then
            ((passed_count++))
        else
            failed_resources+=("$resource_name")
        fi
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "Automation resources missing required functions:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count automation resources have required functions"
}

@test "Storage resources have required functions" {
    local storage_resources
    mapfile -t storage_resources < <(get_resources_by_category "storage")
    
    [[ ${#storage_resources[@]} -eq 0 ]] && skip "No storage resources found"
    
    local failed_resources=()
    local passed_count=0
    
    for resource_path in "${storage_resources[@]}"; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        if validate_required_functions "$resource_path" "storage" >/dev/null 2>&1; then
            ((passed_count++))
        else
            failed_resources+=("$resource_name")
        fi
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "Storage resources missing required functions:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count storage resources have required functions"
}

@test "Agent resources have required functions" {
    local agent_resources
    mapfile -t agent_resources < <(get_resources_by_category "agents")
    
    [[ ${#agent_resources[@]} -eq 0 ]] && skip "No agent resources found"
    
    local failed_resources=()
    local passed_count=0
    
    for resource_path in "${agent_resources[@]}"; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        if validate_required_functions "$resource_path" "agents" >/dev/null 2>&1; then
            ((passed_count++))
        else
            failed_resources+=("$resource_name")
        fi
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "Agent resources missing required functions:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count agent resources have required functions"
}

@test "Execution resources have required functions" {
    local execution_resources
    mapfile -t execution_resources < <(get_resources_by_category "execution")
    
    [[ ${#execution_resources[@]} -eq 0 ]] && skip "No execution resources found"
    
    local failed_resources=()
    local passed_count=0
    
    for resource_path in "${execution_resources[@]}"; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        if validate_required_functions "$resource_path" "execution" >/dev/null 2>&1; then
            ((passed_count++))
        else
            failed_resources+=("$resource_name")
        fi
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "Execution resources missing required functions:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count execution resources have required functions"
}

#######################################
# Port Registry Integration Tests
#######################################

@test "service resources are registered in port registry" {
    local failed_resources=()
    local passed_count=0
    local skipped_count=0
    
    while IFS= read -r resource_path; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        local result
        if result=$(validate_port_registry "$resource_path" 2>&1); then
            local exit_code=$?
            case $exit_code in
                0) ((passed_count++)) ;;
                1) failed_resources+=("$resource_name") ;;
                2) ((skipped_count++)) ;;
            esac
        else
            failed_resources+=("$resource_name")
        fi
    done < <(get_all_resources)
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo "Resources not registered in port registry:"
        printf "  - %s\n" "${failed_resources[@]}"
    fi
    
    echo "✅ $passed_count resources properly registered"
    echo "ℹ️  $skipped_count resources skipped (port registry not found)"
    
    # Don't fail the test if port registry is missing
    [[ $skipped_count -gt 0 && ${#failed_resources[@]} -eq 0 ]]
}

#######################################
# Comprehensive Interface Tests
#######################################

@test "comprehensive interface validation for sample resources" {
    # Test a representative sample of resources rather than all
    # to keep test execution time reasonable
    local sample_resources=(
        "$RESOURCES_DIR/ai/ollama"
        "$RESOURCES_DIR/automation/n8n"
        "$RESOURCES_DIR/storage/minio"
    )
    
    local failed_resources=()
    local passed_count=0
    
    for resource_path in "${sample_resources[@]}"; do
        if [[ ! -d "$resource_path" ]]; then
            continue  # Skip if resource doesn't exist
        fi
        
        local resource_name
        resource_name=$(basename "$resource_path")
        
        echo "Testing comprehensive interface for: $resource_name"
        
        if validate_resource_interface "$resource_path" >/dev/null 2>&1; then
            ((passed_count++))
            echo "✅ $resource_name passed comprehensive validation"
        else
            failed_resources+=("$resource_name")
            echo "❌ $resource_name failed comprehensive validation"
        fi
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        echo
        echo "Resources that failed comprehensive interface validation:"
        printf "  - %s\n" "${failed_resources[@]}"
        false
    fi
    
    echo
    echo "✅ $passed_count resources passed comprehensive interface validation"
}

#######################################
# Interface Evolution Tests
#######################################

@test "resources follow consistent naming patterns" {
    local inconsistent_resources=()
    local passed_count=0
    
    while IFS= read -r resource_path; do
        local resource_name
        resource_name=$(basename "$resource_path")
        
        # Check for consistent naming patterns
        local manage_script="$resource_path/manage.sh"
        local config_dir="$resource_path/config"
        local lib_dir="$resource_path/lib"
        
        local issues=()
        
        # Validate directory structure exists
        [[ ! -d "$config_dir" ]] && issues+=("missing config/ directory")
        [[ ! -d "$lib_dir" ]] && issues+=("missing lib/ directory")
        
        # Validate naming consistency in lib directory
        if [[ -d "$lib_dir" ]]; then
            local expected_files=("common.sh" "install.sh" "status.sh")
            for expected_file in "${expected_files[@]}"; do
                [[ ! -f "$lib_dir/$expected_file" ]] && issues+=("missing lib/$expected_file")
            done
        fi
        
        if [[ ${#issues[@]} -gt 0 ]]; then
            inconsistent_resources+=("$resource_name: ${issues[*]}")
        else
            ((passed_count++))
        fi
    done < <(get_all_resources)
    
    if [[ ${#inconsistent_resources[@]} -gt 0 ]]; then
        echo "Resources with naming/structure inconsistencies:"
        printf "  - %s\n" "${inconsistent_resources[@]}"
        false
    fi
    
    echo "✅ $passed_count resources follow consistent naming patterns"
}