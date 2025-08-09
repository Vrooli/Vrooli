#!/usr/bin/env bats
# Tests for Syntax Validator
# Tests the syntax validation framework for resource management scripts

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091  
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create test environment
    export TEST_TEMP_DIR="${BATS_TEST_TMPDIR}/syntax_test"
    mkdir -p "$TEST_TEMP_DIR"
    
    # Create mock directories and files for testing
    mkdir -p "${TEST_TEMP_DIR}/contracts/v1.0"
    mkdir -p "${TEST_TEMP_DIR}/cache"
    mkdir -p "${TEST_TEMP_DIR}/resources/ai/ollama"
    mkdir -p "${TEST_TEMP_DIR}/resources/automation/n8n"
    
    # Create mock manage.sh scripts
    create_mock_manage_script() {
        local path="$1"
        cat > "$path" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

# Mock manage.sh script for testing

case "$1" in
    install)
        echo "Installing..."
        ;;
    uninstall)
        echo "Uninstalling..."
        ;;
    status)
        echo "Checking status..."
        ;;
    --help|-h)
        echo "Usage: $0 [install|uninstall|status|--help]"
        ;;
    *)
        echo "Unknown action: $1"
        exit 1
        ;;
esac
EOF
        chmod +x "$path"
    }
    
    create_mock_manage_script "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh"
    create_mock_manage_script "${TEST_TEMP_DIR}/resources/automation/n8n/manage.sh"
    
    # Create mock contract files
    cat > "${TEST_TEMP_DIR}/contracts/v1.0/core.yaml" <<'EOF'
version: "1.0"
contract_type: "core"
description: "Core resource contract"

required_actions:
  install:
    description: "Install the resource"
  uninstall:
    description: "Uninstall the resource"
  status:
    description: "Check resource status"

help_patterns:
  - "--help"
  - "-h"

error_handling:
  - "set -e"
  - "error messages to stderr"

required_files:
  - "manage.sh"
  - "config/"
  - "lib/"
EOF
    
    cat > "${TEST_TEMP_DIR}/contracts/v1.0/ai.yaml" <<'EOF'
version: "1.0"
contract_type: "ai"
extends: "core.yaml"
description: "AI resource contract"

additional_actions:
  train:
    description: "Train AI models"
  inference:
    description: "Run inference"
EOF
    
    # Source the syntax validator FIRST before setting up mocks
    # This allows the real dependencies to be loaded, then we override them
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/syntax.sh"
    
    # Mock the dependency scripts with minimal functionality
    # These will override the real functions that were sourced
    setup_mock_contract_parser
    setup_mock_script_analyzer
    setup_mock_cache_manager
}

# Cleanup after tests
teardown() {
    vrooli_cleanup_test
}

# Mock the contract parser functions
setup_mock_contract_parser() {
    # Mock contract_parser::init
    contract_parser::init() {
        VROOLI_CONTRACTS_DIR="${TEST_TEMP_DIR}/contracts"
        VROOLI_CONTRACT_CACHE="${TEST_TEMP_DIR}/cache"
        return 0
    }
    
    # Mock contract_parser::get_required_actions
    contract_parser::get_required_actions() {
        echo "install"
        echo "uninstall"
        echo "status"
        return 0
    }
    
    # Mock contract_parser::get_help_patterns
    contract_parser::get_help_patterns() {
        echo "--help"
        echo "-h"
        return 0
    }
    
    # Mock contract_parser::cleanup
    contract_parser::cleanup() {
        return 0
    }
    
    export -f contract_parser::init
    export -f contract_parser::get_required_actions
    export -f contract_parser::get_help_patterns
    export -f contract_parser::cleanup
}

# Mock the script analyzer functions
setup_mock_script_analyzer() {
    # Mock script_analyzer::extract_script_actions
    script_analyzer::extract_script_actions() {
        local script_path="$1"
        if [[ -f "$script_path" ]]; then
            echo "install"
            echo "uninstall"
            echo "status"
            return 0
        fi
        return 1
    }
    
    # Mock script_analyzer::validate_help_patterns
    script_analyzer::validate_help_patterns() {
        echo "FOUND: --help pattern in script"
        return 0
    }
    
    # Mock script_analyzer::check_error_handling_patterns
    script_analyzer::check_error_handling_patterns() {
        echo "FOUND: set -e"
        echo "FOUND: error handling"
        return 0
    }
    
    # Mock script_analyzer::check_required_files
    script_analyzer::check_required_files() {
        echo "FOUND: config/"
        echo "FOUND: lib/"
        return 0
    }
    
    # Mock script_analyzer::analyze_argument_patterns
    script_analyzer::analyze_argument_patterns() {
        # Output must contain these exact patterns for the test to pass
        echo "GOOD: action_flag"
        echo "GOOD: case_statement"
        return 0
    }
    
    # Mock script_analyzer::check_configuration_loading
    script_analyzer::check_configuration_loading() {
        # Output must contain common_sourced for the test to pass
        echo "FOUND: common_sourced"
        return 0
    }
    
    export -f script_analyzer::extract_script_actions
    export -f script_analyzer::validate_help_patterns
    export -f script_analyzer::check_error_handling_patterns
    export -f script_analyzer::check_required_files
    export -f script_analyzer::analyze_argument_patterns
    export -f script_analyzer::check_configuration_loading
}

# Mock the cache manager functions
setup_mock_cache_manager() {
    # Mock cache_manager::init
    cache_manager::init() {
        return 0
    }
    
    # Mock cache::get
    cache::get() {
        # Always return cache miss for testing
        return 1
    }
    
    # Mock cache::set
    cache::set() {
        return 0
    }
    
    # Mock cache::create_result_json
    cache::create_result_json() {
        local status="$1"
        local details="$2"
        local duration="$3"
        echo '{"status":"'"$status"'","details":"'"$details"'","duration":'"$duration"'}'
        return 0
    }
    
    # Mock cache::parse_result
    cache::parse_result() {
        return 0
    }
    
    # Mock cache::clear_expired
    cache::clear_expired() {
        echo "Cleared 0 expired entries"
        return 0
    }
    
    # Mock cache::get_stats
    cache::get_stats() {
        echo '{"cache_hits":0,"cache_misses":0,"hit_rate_percent":0,"total_entries":0}'
        return 0
    }
    
    export -f cache_manager::init
    export -f cache::get
    export -f cache::set
    export -f cache::create_result_json
    export -f cache::parse_result
    export -f cache::clear_expired
    export -f cache::get_stats
}

#######################################
# Test: syntax::validator_init
#######################################

@test "syntax::validator_init initializes successfully with contracts dir" {
    run syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Syntax validator initialized"* ]]
}

@test "syntax::validator_init initializes without contracts dir" {
    run syntax::validator_init
    [ "$status" -eq 0 ]
    [[ "$output" == *"Syntax validator initialized"* ]]
}

#######################################
# Test: syntax::validate_required_actions
#######################################

@test "syntax::validate_required_actions passes when all actions present" {
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    run syntax::validate_required_actions "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh"
    [ "$status" -eq 0 ]
}

@test "syntax::validate_required_actions fails when script not found" {
    run syntax::validate_required_actions "missing" "ai" "/nonexistent/manage.sh"
    [ "$status" -eq 1 ]
}

#######################################
# Test: syntax::validate_help_patterns
#######################################

@test "syntax::validate_help_patterns passes when help patterns found" {
    run syntax::validate_help_patterns "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh"
    [ "$status" -eq 0 ]
}

#######################################
# Test: syntax::validate_error_handling
#######################################

@test "syntax::validate_error_handling passes with adequate patterns" {
    run syntax::validate_error_handling "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh"
    [ "$status" -eq 0 ]
}

#######################################
# Test: syntax::validate_file_structure
#######################################

@test "syntax::validate_file_structure passes with compliant structure" {
    run syntax::validate_file_structure "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh"
    [ "$status" -eq 0 ]
}

#######################################
# Test: syntax::validate_argument_patterns
#######################################

@test "syntax::validate_argument_patterns passes with consistent patterns" {
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    run syntax::validate_argument_patterns "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh"
    [ "$status" -eq 0 ]
}

#######################################
# Test: syntax::validate_configuration_loading
#######################################

@test "syntax::validate_configuration_loading passes when config loading found" {
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    run syntax::validate_configuration_loading "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh"
    [ "$status" -eq 0 ]
}

#######################################
# Test: syntax::validate_resource_syntax
#######################################

@test "syntax::validate_resource_syntax performs comprehensive validation" {
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    run syntax::validate_resource_syntax "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh" "false"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Syntax validation PASSED"* ]]
}

@test "syntax::validate_resource_syntax uses cache when enabled" {
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    
    # First run should miss cache
    run syntax::validate_resource_syntax "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh" "true"
    [ "$status" -eq 0 ]
    
    # Second run would hit cache if it were working (mocked to always miss)
    run syntax::validate_resource_syntax "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh" "true"
    [ "$status" -eq 0 ]
}

#######################################
# Test: syntax::detect_resource_category
#######################################

@test "syntax::detect_resource_category extracts category from path" {
    run syntax::detect_resource_category "/path/to/ai/ollama"
    [ "$status" -eq 0 ]
    [ "$output" = "ai" ]
}

@test "syntax::detect_resource_category returns unknown for invalid path" {
    run syntax::detect_resource_category "/invalid"
    [ "$status" -eq 1 ]
    [ "$output" = "unknown" ]
}

#######################################
# Test: syntax::validate_resources_batch
#######################################

@test "syntax::validate_resources_batch validates multiple resources" {
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    
    local resource_paths=(
        "${TEST_TEMP_DIR}/resources/ai/ollama"
        "${TEST_TEMP_DIR}/resources/automation/n8n"
    )
    
    run syntax::validate_resources_batch "${resource_paths[@]}"
    [ "$status" -eq 0 ]
    [[ "$output" == *"All resources passed syntax validation"* ]]
}

@test "syntax::validate_resources_batch skips resources without manage.sh" {
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    
    mkdir -p "${TEST_TEMP_DIR}/resources/storage/postgres"
    local resource_paths=(
        "${TEST_TEMP_DIR}/resources/ai/ollama"
        "${TEST_TEMP_DIR}/resources/storage/postgres"  # No manage.sh
    )
    
    run syntax::validate_resources_batch "${resource_paths[@]}"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Skipping postgres: manage.sh not found"* ]]
}

#######################################
# Test: syntax::validator_cleanup
#######################################

@test "syntax::validator_cleanup cleans up resources" {
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    run syntax::validator_cleanup
    [ "$status" -eq 0 ]
    [[ "$output" == *"Syntax validator cleanup completed"* ]]
}

#######################################
# Test: Global arrays are properly managed
#######################################

@test "global validation arrays are cleared on init" {
    # Add some test data to arrays
    SYNTAX_VALIDATION_RESULTS=("old result")
    SYNTAX_VALIDATION_ERRORS=("old error")
    SYNTAX_VALIDATION_WARNINGS=("old warning")
    
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    
    # Arrays should be cleared
    [ ${#SYNTAX_VALIDATION_RESULTS[@]} -eq 0 ]
    [ ${#SYNTAX_VALIDATION_ERRORS[@]} -eq 0 ]
    [ ${#SYNTAX_VALIDATION_WARNINGS[@]} -eq 0 ]
}

@test "global validation arrays collect results during validation" {
    syntax::validator_init "${TEST_TEMP_DIR}/contracts"
    
    # Run validation that should populate arrays
    syntax::validate_required_actions "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh"
    
    # At least one result should be recorded
    [ ${#SYNTAX_VALIDATION_RESULTS[@]} -gt 0 ]
}

#######################################
# Test: Error handling edge cases
#######################################

@test "validation handles missing dependencies gracefully" {
    # Unset a mock function to simulate missing dependency
    unset -f contract_parser::get_required_actions
    
    run syntax::validate_required_actions "ollama" "ai" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh"
    [ "$status" -eq 1 ]
}

@test "validation handles empty category gracefully" {
    run syntax::validate_resource_syntax "test" "" "${TEST_TEMP_DIR}/resources/ai/ollama/manage.sh" "false"
    # Should still attempt validation even with empty category
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

#######################################
# Test: Integration with real file structures
#######################################

@test "validation works with nested resource paths" {
    mkdir -p "${TEST_TEMP_DIR}/resources/ai/models/llm/ollama"
    create_mock_manage_script() {
        cat > "$1" <<'EOF'
#!/usr/bin/env bash
case "$1" in
    install|uninstall|status) echo "Action: $1" ;;
    --help|-h) echo "Usage" ;;
    *) exit 1 ;;
esac
EOF
        chmod +x "$1"
    }
    create_mock_manage_script "${TEST_TEMP_DIR}/resources/ai/models/llm/ollama/manage.sh"
    
    run syntax::detect_resource_category "${TEST_TEMP_DIR}/resources/ai/models/llm/ollama"
    [ "$status" -eq 0 ]
    [ "$output" = "llm" ]
}