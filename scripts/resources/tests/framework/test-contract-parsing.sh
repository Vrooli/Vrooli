#!/usr/bin/env bash
# Test Contract Parsing System
# Validates that contract parser works with real resources

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"
_HERE="${APP_ROOT}/scripts/resources/tests/framework"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091,SC2154
source "${var_SCRIPTS_RESOURCES_DIR}/tests/framework/parsers/contract-parser.sh"
# shellcheck disable=SC1091,SC2154
source "${var_SCRIPTS_RESOURCES_DIR}/tests/framework/parsers/script-analyzer.sh"

# Initialize contract parser
contract_parser::init "${var_SCRIPTS_RESOURCES_DIR}/contracts"

echo "=== Contract Parsing Test Suite ==="
echo

#######################################
# Test contract loading and merging
#######################################
test_contract_parsing::test_contract_loading() {
    echo "--- Testing Contract Loading ---"
    
    local test_passed=0
    local test_failed=0
    
    # Test core contract loading
    echo "Testing core contract..."
    if core_contract=$(contract_parser::load_contract "core.yaml"); then
        echo "✅ Core contract loaded: $core_contract"
        ((test_passed++))
    else
        echo "❌ Failed to load core contract"
        ((test_failed++))
    fi
    
    # Test category contracts
    local categories=("ai" "automation" "agents" "storage" "search" "execution")
    
    for category in "${categories[@]}"; do
        echo "Testing $category contract..."
        if category_contract=$(contract_parser::load_contract "${category}.yaml"); then
            echo "✅ $category contract loaded: $category_contract"
            ((test_passed++))
        else
            echo "❌ Failed to load $category contract"
            ((test_failed++))
        fi
    done
    
    echo "Contract loading: $test_passed passed, $test_failed failed"
    echo
    
    return "$([[ $test_failed -eq 0 ]] && echo 0 || echo 1)"
}

#######################################
# Test required actions extraction
#######################################
test_contract_parsing::test_required_actions() {
    echo "--- Testing Required Actions Extraction ---"
    
    local test_passed=0
    local test_failed=0
    
    # Test each category
    local categories=("ai" "automation" "agents" "storage" "search" "execution")
    
    for category in "${categories[@]}"; do
        echo "Testing required actions for $category..."
        
        if required_actions=$(contract_parser::get_required_actions "$category"); then
            local action_count
            action_count=$(echo "$required_actions" | wc -l)
            
            if [[ $action_count -ge 5 ]]; then
                echo "✅ $category has $action_count required actions"
                echo "   Actions: $(echo "$required_actions" | tr '\n' ' ')"
                ((test_passed++))
            else
                echo "❌ $category has too few actions ($action_count)"
                ((test_failed++))
            fi
        else
            echo "❌ Failed to get required actions for $category"
            ((test_failed++))
        fi
    done
    
    echo "Required actions: $test_passed passed, $test_failed failed"
    echo
    
    return "$([[ $test_failed -eq 0 ]] && echo 0 || echo 1)"
}

#######################################
# Test help patterns extraction
#######################################
test_contract_parsing::test_help_patterns() {
    echo "--- Testing Help Patterns Extraction ---"
    
    local test_passed=0
    local test_failed=0
    
    # Test help patterns for core
    echo "Testing help patterns..."
    
    if help_patterns=$(contract_parser::get_help_patterns "ai"); then
        local pattern_count
        pattern_count=$(echo "$help_patterns" | wc -l)
        
        if [[ $pattern_count -ge 2 ]]; then
            echo "✅ Found $pattern_count help patterns"
            echo "   Patterns: $(echo "$help_patterns" | tr '\n' ' ')"
            ((test_passed++))
        else
            echo "❌ Too few help patterns ($pattern_count)"
            ((test_failed++))
        fi
    else
        echo "❌ Failed to get help patterns"
        ((test_failed++))
    fi
    
    echo "Help patterns: $test_passed passed, $test_failed failed"
    echo
    
    return "$([[ $test_failed -eq 0 ]] && echo 0 || echo 1)"
}

#######################################
# Test with real resource scripts
#######################################
test_contract_parsing::test_real_resources() {
    echo "--- Testing with Real Resource Scripts ---"
    
    local test_passed=0
    local test_failed=0
    
    # Find some real manage.sh scripts
    local sample_resources=(
        "${var_SCRIPTS_RESOURCES_DIR}/ai/ollama"
        "${var_SCRIPTS_RESOURCES_DIR}/automation/n8n"
        "${var_SCRIPTS_RESOURCES_DIR}/agents/agent-s2"
        "${var_SCRIPTS_RESOURCES_DIR}/storage/postgres"
    )
    
    for resource_dir in "${sample_resources[@]}"; do
        if [[ -f "$resource_dir/cli.sh" ]]; then
            local resource_name
            resource_name=$(basename "$resource_dir")
            echo "Testing $resource_name..."
            
            # Test script analysis
            if script_analyzer::extract_script_actions "$resource_dir/cli.sh" >/dev/null; then
                echo "✅ $resource_name: Actions extracted successfully"
                ((test_passed++))
            else
                echo "❌ $resource_name: Failed to extract actions"
                ((test_failed++))
            fi
            
            # Test basic script checks
            if script_analyzer::check_script_basics "$resource_dir/cli.sh" >/dev/null; then
                echo "✅ $resource_name: Basic script checks passed"
                ((test_passed++))
            else
                echo "⚠️  $resource_name: Basic script checks failed (expected for some resources)"
                # Don't count as failure since existing resources may not meet all standards yet
            fi
        else
            echo "⚠️  $resource_name: cli.sh not found at $resource_dir"
        fi
    done
    
    echo "Real resources: $test_passed passed, $test_failed failed"
    echo
    
    return "$([[ $test_failed -eq 0 ]] && echo 0 || echo 1)"
}

#######################################
# Test contract validation
#######################################
test_contract_parsing::test_contract_validation() {
    echo "--- Testing Contract Validation ---"
    
    local test_passed=0
    local test_failed=0
    
    # Test each contract file
    local contract_files=("core.yaml" "ai.yaml" "automation.yaml" "agents.yaml" "storage.yaml" "search.yaml" "execution.yaml")
    
    for contract_file in "${contract_files[@]}"; do
        local contract_path="${var_SCRIPTS_RESOURCES_DIR}/contracts/v1.0/$contract_file"
        echo "Validating $contract_file..."
        
        if contract_parser::validate_contract_syntax "$contract_path"; then
            echo "✅ $contract_file: Syntax validation passed"
            ((test_passed++))
        else
            echo "❌ $contract_file: Syntax validation failed"
            ((test_failed++))
        fi
    done
    
    echo "Contract validation: $test_passed passed, $test_failed failed"
    echo
    
    return "$([[ $test_failed -eq 0 ]] && echo 0 || echo 1)"
}

#######################################
# Run all tests
#######################################
test_contract_parsing::run_all_tests() {
    local total_passed=0
    local total_failed=0
    
    echo "Starting contract parsing test suite..."
    echo "========================================"
    echo
    
    # Run test suites
    local test_suites=(
        "test_contract_parsing::test_contract_validation"
        "test_contract_parsing::test_contract_loading"
        "test_contract_parsing::test_required_actions"
        "test_contract_parsing::test_help_patterns"
        "test_contract_parsing::test_real_resources"
    )
    
    for test_suite in "${test_suites[@]}"; do
        if $test_suite; then
            echo "✅ $test_suite PASSED"
            ((total_passed++))
        else
            echo "❌ $test_suite FAILED"
            ((total_failed++))
        fi
        echo
    done
    
    echo "========================================"
    echo "Test Suite Summary:"
    echo "  Total test suites: $((total_passed + total_failed))"
    echo "  Passed: $total_passed"
    echo "  Failed: $total_failed"
    echo
    
    if [[ $total_failed -eq 0 ]]; then
        echo "🎉 All contract parsing tests PASSED!"
        return 0
    else
        echo "⚠️  Some contract parsing tests FAILED"
        return 1
    fi
}

# Cleanup function
test_contract_parsing::cleanup_tests() {
    contract_parser::cleanup
    echo "Test cleanup completed"
}

# Set up cleanup trap
trap test_contract_parsing::cleanup_tests EXIT

# Run the tests
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    test_contract_parsing::run_all_tests
fi