#!/usr/bin/env bash
################################################################################
# Integration Test for New Codex Architecture
# 
# Tests all major components working together:
# - Orchestrator
# - Execution contexts  
# - Models system
# - APIs
# - Capabilities
# - Tools
# - Workspace management
# - Interface integration
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
CODEX_ROOT="${APP_ROOT}/resources/codex"
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || {
    # Fallback logging functions if not available
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::error() { echo "[ERROR] $*"; }
    log::warn() { echo "[WARN] $*"; }
    log::debug() { echo "[DEBUG] $*"; }
    log::header() { echo "=== $* ==="; }
}

# Test configuration
TEST_WORKSPACE=""
CLEANUP_ON_EXIT=true
VERBOSE=false

################################################################################
# Test Utilities
################################################################################

# Track test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

#######################################
# Run a test with automatic result tracking
# Arguments:
#   $1 - Test name
#   $2 - Test function
#######################################
run_test() {
    local test_name="$1"
    local test_function="$2"
    
    ((TESTS_RUN++))
    
    log::info "Running test: $test_name"
    
    if [[ "$VERBOSE" == "true" ]]; then
        if $test_function; then
            log::success "✓ $test_name"
            ((TESTS_PASSED++))
            return 0
        else
            log::error "✗ $test_name"
            ((TESTS_FAILED++))
            FAILED_TESTS+=("$test_name")
            return 1
        fi
    else
        # Capture output for non-verbose mode
        local output exit_code
        output=$($test_function 2>&1)
        exit_code=$?
        
        if [[ $exit_code -eq 0 ]]; then
            log::success "✓ $test_name"
            ((TESTS_PASSED++))
            return 0
        else
            log::error "✗ $test_name"
            if [[ -n "$output" ]]; then
                echo "  Error output: $output"
            fi
            ((TESTS_FAILED++))
            FAILED_TESTS+=("$test_name")
            return 1
        fi
    fi
}

#######################################
# Print test summary
#######################################
print_test_summary() {
    echo ""
    log::header "Integration Test Results"
    echo "Tests run: $TESTS_RUN"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo ""
        log::error "Failed tests:"
        for failed_test in "${FAILED_TESTS[@]}"; do
            echo "  - $failed_test"
        done
        return 1
    else
        log::success "All tests passed!"
        return 0
    fi
}

################################################################################
# Component Tests
################################################################################

#######################################
# Test orchestrator loading and basic functionality
#######################################
test_orchestrator_loading() {
    # Source orchestrator
    source "${CODEX_ROOT}/lib/orchestrator.sh" 2>/dev/null || return 1
    
    # Check if main function exists
    if ! type -t orchestrator::execute &>/dev/null; then
        return 1
    fi
    
    return 0
}

#######################################
# Test execution contexts loading
#######################################
test_execution_contexts() {
    # Test each execution context can be loaded
    local contexts=("external-cli" "internal-sandbox" "text-only")
    
    for context in "${contexts[@]}"; do
        source "${CODEX_ROOT}/lib/execution-contexts/${context}.sh" 2>/dev/null || return 1
        
        # Check for main execute function
        local func_name="${context//-/_}_context::execute"
        if ! type -t "$func_name" &>/dev/null; then
            echo "Missing function: $func_name"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Test models system loading
#######################################
test_models_system() {
    # Test model selector
    source "${CODEX_ROOT}/lib/models/model-selector.sh" 2>/dev/null || return 1
    
    if ! type -t model_selector::get_best_model &>/dev/null; then
        return 1
    fi
    
    # Test model registries
    local categories=("coding" "general" "reasoning")
    
    for category in "${categories[@]}"; do
        source "${CODEX_ROOT}/lib/models/${category}/registry.sh" 2>/dev/null || return 1
    done
    
    return 0
}

#######################################
# Test APIs system loading
#######################################
test_apis_system() {
    # Test each API module
    local apis=("http-client" "completions" "responses")
    
    for api in "${apis[@]}"; do
        source "${CODEX_ROOT}/lib/apis/${api}.sh" 2>/dev/null || return 1
    done
    
    # Check key functions exist
    if ! type -t http_client::request &>/dev/null; then
        return 1
    fi
    
    if ! type -t completions_api::call &>/dev/null; then
        return 1
    fi
    
    if ! type -t responses_api::call &>/dev/null; then
        return 1
    fi
    
    return 0
}

#######################################
# Test capabilities system loading
#######################################
test_capabilities_system() {
    # Test each capability module
    local capabilities=("text-generation" "function-calling" "reasoning")
    
    for capability in "${capabilities[@]}"; do
        source "${CODEX_ROOT}/lib/capabilities/${capability}.sh" 2>/dev/null || return 1
        
        # Check main function exists
        local func_name="${capability//-/_}::execute"
        if ! type -t "$func_name" &>/dev/null; then
            echo "Missing function: $func_name"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Test tools system loading and basic functionality
#######################################
test_tools_system() {
    # Test tools registry
    source "${CODEX_ROOT}/lib/tools/registry.sh" 2>/dev/null || return 1
    
    if ! type -t tool_registry::list_tools &>/dev/null; then
        return 1
    fi
    
    # Initialize built-in tools
    if ! tool_registry::initialize_builtin_tools; then
        return 1
    fi
    
    # Test that basic tools are available
    local basic_tools=("write_file" "read_file" "list_files" "execute_command" "analyze_code")
    
    for tool in "${basic_tools[@]}"; do
        local tool_info
        tool_info=$(tool_registry::get_tool "$tool")
        
        if [[ $(echo "$tool_info" | jq 'keys | length') -eq 0 ]]; then
            echo "Tool not found: $tool"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Test workspace system loading and basic functionality
#######################################
test_workspace_system() {
    # Test workspace manager
    source "${CODEX_ROOT}/lib/workspace/manager.sh" 2>/dev/null || return 1
    source "${CODEX_ROOT}/lib/workspace/security.sh" 2>/dev/null || return 1
    source "${CODEX_ROOT}/lib/workspace/isolation.sh" 2>/dev/null || return 1
    
    # Check key functions exist
    if ! type -t workspace_manager::create &>/dev/null; then
        return 1
    fi
    
    if ! type -t workspace_security::apply_policy &>/dev/null; then
        return 1
    fi
    
    if ! type -t workspace_isolation::create_environment &>/dev/null; then
        return 1
    fi
    
    return 0
}

#######################################
# Test interface integration
#######################################
test_interface_integration() {
    # Test content.sh
    source "${CODEX_ROOT}/lib/content.sh" 2>/dev/null || return 1
    
    if ! type -t codex::content::execute &>/dev/null; then
        return 1
    fi
    
    if ! type -t codex::content::analyze &>/dev/null; then
        return 1
    fi
    
    if ! type -t codex::content::create_workspace &>/dev/null; then
        return 1
    fi
    
    # Test status.sh
    source "${CODEX_ROOT}/lib/status.sh" 2>/dev/null || return 1
    
    if ! type -t codex::status::collect_data &>/dev/null; then
        return 1
    fi
    
    return 0
}

################################################################################
# Functional Tests
################################################################################

#######################################
# Test workspace creation and management
#######################################
test_workspace_functionality() {
    # Create a test workspace
    local result
    result=$(workspace_manager::create "test-integration-$(date +%s)" "moderate" '{"description": "Integration test workspace", "auto_cleanup": true}')
    
    if [[ $(echo "$result" | jq -r '.success // false') != "true" ]]; then
        echo "Failed to create workspace"
        return 1
    fi
    
    TEST_WORKSPACE=$(echo "$result" | jq -r '.workspace_id')
    
    # Test workspace info retrieval
    local info
    info=$(workspace_manager::get_info "$TEST_WORKSPACE")
    
    if [[ $(echo "$info" | jq -r '.success // false') != "true" ]]; then
        echo "Failed to get workspace info"
        return 1
    fi
    
    # Test workspace listing
    local workspaces
    workspaces=$(workspace_manager::list "active")
    
    if [[ $(echo "$workspaces" | jq "map(select(.workspace_id == \"$TEST_WORKSPACE\")) | length") -eq 0 ]]; then
        echo "Workspace not found in active list"
        return 1
    fi
    
    return 0
}

#######################################
# Test tool execution functionality  
#######################################
test_tool_functionality() {
    if [[ -z "$TEST_WORKSPACE" ]]; then
        echo "No test workspace available"
        return 1
    fi
    
    # Test file operations
    local write_args read_args list_args
    write_args=$(jq -n --arg path "test.txt" --arg content "Hello, integration test!" '{path: $path, content: $content}')
    
    # Test write_file tool
    local write_result
    write_result=$(tool_registry::execute_tool "write_file" "$write_args" "sandbox")
    
    if [[ $(echo "$write_result" | jq -r '.success // false') != "true" ]]; then
        echo "write_file tool failed: $(echo "$write_result" | jq -r '.error // "unknown"')"
        return 1
    fi
    
    # Test read_file tool
    read_args=$(jq -n --arg path "test.txt" '{path: $path}')
    local read_result
    read_result=$(tool_registry::execute_tool "read_file" "$read_args" "sandbox")
    
    if [[ $(echo "$read_result" | jq -r '.success // false') != "true" ]]; then
        echo "read_file tool failed: $(echo "$read_result" | jq -r '.error // "unknown"')"
        return 1
    fi
    
    # Verify content matches
    local content
    content=$(echo "$read_result" | jq -r '.content')
    if [[ "$content" != "Hello, integration test!" ]]; then
        echo "Content mismatch: expected 'Hello, integration test!', got '$content'"
        return 1
    fi
    
    # Test list_files tool
    list_args=$(jq -n '{path: "."}')
    local list_result
    list_result=$(tool_registry::execute_tool "list_files" "$list_args" "sandbox")
    
    if [[ $(echo "$list_result" | jq -r '.success // false') != "true" ]]; then
        echo "list_files tool failed: $(echo "$list_result" | jq -r '.error // "unknown"')"
        return 1
    fi
    
    return 0
}

#######################################
# Test code analysis functionality
#######################################
test_code_analysis() {
    # Test analyze_code tool with sample code
    local sample_code='def hello_world():
    print("Hello, world!")
    return "success"

if __name__ == "__main__":
    result = hello_world()
    print(result)'
    
    local analyze_args
    analyze_args=$(jq -n \
        --arg code "$sample_code" \
        --arg language "python" \
        --arg type "all" \
        '{code: $code, language: $language, analysis_type: $type}')
    
    local analyze_result
    analyze_result=$(tool_registry::execute_tool "analyze_code" "$analyze_args" "sandbox")
    
    if [[ $(echo "$analyze_result" | jq -r '.success // false') != "true" ]]; then
        echo "analyze_code tool failed: $(echo "$analyze_result" | jq -r '.error // "unknown"')"
        return 1
    fi
    
    # Check that we got comprehensive analysis results
    local analysis_type
    analysis_type=$(echo "$analyze_result" | jq -r '.analysis_type')
    if [[ "$analysis_type" != "comprehensive" ]]; then
        echo "Expected comprehensive analysis, got: $analysis_type"
        return 1
    fi
    
    return 0
}

#######################################
# Test model selection functionality
#######################################
test_model_selection() {
    # Test model selection for different capabilities
    local capabilities=("text-generation" "function-calling" "reasoning")
    
    for capability in "${capabilities[@]}"; do
        local model_config
        model_config=$(model_selector::get_best_model "$capability" "sandbox" "Test request")
        
        if [[ $(echo "$model_config" | jq 'keys | length') -eq 0 ]]; then
            echo "No model selected for capability: $capability"
            return 1
        fi
        
        # Check required fields are present
        local model_name api_endpoint
        model_name=$(echo "$model_config" | jq -r '.model_name // ""')
        api_endpoint=$(echo "$model_config" | jq -r '.api_endpoint // ""')
        
        if [[ -z "$model_name" || -z "$api_endpoint" ]]; then
            echo "Invalid model config for $capability: missing model_name or api_endpoint"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Test system status reporting
#######################################
test_status_reporting() {
    # Test new status collection
    local status_data
    status_data=$(codex::status::collect_data --fast)
    
    # Convert to associative array format for testing
    local -A status_map
    local key value
    while IFS= read -r line; do
        if [[ -z "$key" ]]; then
            key="$line"
        else
            value="$line"
            status_map["$key"]="$value"
            key=""
        fi
    done <<< "$status_data"
    
    # Check that new architecture components are reported
    local required_keys=("orchestrator_available" "tools_available" "workspace_available" "models_available")
    
    for req_key in "${required_keys[@]}"; do
        if [[ -z "${status_map[$req_key]}" ]]; then
            echo "Missing status key: $req_key"
            return 1
        fi
    done
    
    return 0
}

################################################################################
# Cleanup
################################################################################

cleanup() {
    if [[ "$CLEANUP_ON_EXIT" == "true" && -n "$TEST_WORKSPACE" ]]; then
        log::debug "Cleaning up test workspace: $TEST_WORKSPACE"
        workspace_manager::delete "$TEST_WORKSPACE" "true" >/dev/null 2>&1
    fi
}

trap cleanup EXIT

################################################################################
# Main Test Runner
################################################################################

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --no-cleanup)
                CLEANUP_ON_EXIT=false
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [--verbose] [--no-cleanup]"
                echo "  --verbose    : Show detailed test output"
                echo "  --no-cleanup : Don't cleanup test workspace on exit"
                exit 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    log::header "Codex Architecture Integration Test"
    echo "Testing new orchestrator-based architecture..."
    echo ""
    
    # Component loading tests
    log::info "=== Component Loading Tests ==="
    run_test "Orchestrator loading" "test_orchestrator_loading"
    run_test "Execution contexts loading" "test_execution_contexts"
    run_test "Models system loading" "test_models_system"
    run_test "APIs system loading" "test_apis_system"
    run_test "Capabilities system loading" "test_capabilities_system"
    run_test "Tools system loading" "test_tools_system"
    run_test "Workspace system loading" "test_workspace_system"
    run_test "Interface integration" "test_interface_integration"
    
    echo ""
    log::info "=== Functional Tests ==="
    run_test "Workspace functionality" "test_workspace_functionality"
    run_test "Tool functionality" "test_tool_functionality"
    run_test "Code analysis" "test_code_analysis"
    run_test "Model selection" "test_model_selection"
    run_test "Status reporting" "test_status_reporting"
    
    # Print final results
    echo ""
    print_test_summary
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi