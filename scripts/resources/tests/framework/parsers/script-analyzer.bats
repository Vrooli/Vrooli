#!/usr/bin/env bats
# Script Analyzer Tests - Comprehensive test suite for script analysis functions

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/tests/framework/parsers"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../../__test/fixtures/setup.bash"

# =============================================================================
# Test Setup and Teardown
# =============================================================================

setup() {
    # Setup test environment
    vrooli_setup_unit_test
    
    # Load the script analyzer
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/script-analyzer.sh"
    
    # Create temporary test environment
    export TEST_SCRIPTS_DIR="${BATS_TEST_TMPDIR}/test_scripts"
    
    mkdir -p "$TEST_SCRIPTS_DIR/config"
    mkdir -p "$TEST_SCRIPTS_DIR/lib"
    
    # Create test script fixtures
    create_test_scripts
}

teardown() {
    # Clean up test environment
    vrooli_cleanup_test
    trash::safe_remove "$TEST_SCRIPTS_DIR" --test-cleanup
}

# =============================================================================
# Test Script Fixtures
# =============================================================================

create_test_scripts() {
    # Create a well-formed manage.sh script
    cat > "$TEST_SCRIPTS_DIR/manage.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

# Source required files
source "config/defaults.sh"
source "config/messages.sh"
source "lib/common.sh"

# Cleanup function
cleanup() {
    echo "Cleaning up..."
}

trap cleanup EXIT

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --yes)
            YES="yes"
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        --version)
            echo "1.0.0"
            exit 0
            ;;
        *)
            echo "ERROR: Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# Validate arguments
if [[ -z "${ACTION:-}" ]]; then
    echo "ERROR: Action required" >&2
    exit 1
fi

# Main action switch
case "$ACTION" in
    "install")
        echo "Installing..."
        exit 0
        ;;
    "start")
        echo "Starting..."
        exit 0
        ;;
    "stop")
        echo "Stopping..."
        exit 0
        ;;
    *)
        echo "ERROR: Unknown action: $ACTION" >&2
        exit 1
        ;;
esac
EOF
    chmod +x "$TEST_SCRIPTS_DIR/manage.sh"
    
    # Create required config files
    cat > "$TEST_SCRIPTS_DIR/config/defaults.sh" << 'EOF'
#!/usr/bin/env bash
# Default configuration
DEFAULT_PORT=8080
EOF
    
    cat > "$TEST_SCRIPTS_DIR/config/messages.sh" << 'EOF'
#!/usr/bin/env bash
# Message strings
MSG_SUCCESS="Operation successful"
EOF
    
    # Create common library
    cat > "$TEST_SCRIPTS_DIR/lib/common.sh" << 'EOF'
#!/usr/bin/env bash
# Common functions
show_help() {
    echo "Usage: manage.sh --action <action> [--yes]"
}

log_error() {
    echo "ERROR: $1" >&2
}
EOF
    
    # Create a minimal script (missing many patterns)
    cat > "$TEST_SCRIPTS_DIR/minimal.sh" << 'EOF'
#!/bin/bash
echo "Hello"
EOF
    chmod +x "$TEST_SCRIPTS_DIR/minimal.sh"
    
    # Create an invalid script (no shebang)
    cat > "$TEST_SCRIPTS_DIR/invalid.sh" << 'EOF'
echo "No shebang"
EOF
    
    # Create a script with no error handling
    cat > "$TEST_SCRIPTS_DIR/no_error_handling.sh" << 'EOF'
#!/usr/bin/env bash
# No error handling
echo "Processing..."
EOF
    chmod +x "$TEST_SCRIPTS_DIR/no_error_handling.sh"
}

# =============================================================================
# Basic Function Tests
# =============================================================================

@test "script_analyzer::extract_script_actions: extracts actions from manage.sh" {
    run script_analyzer::extract_script_actions "$TEST_SCRIPTS_DIR/manage.sh"
    
    assert_success
    assert_line "install"
    assert_line "start"
    assert_line "stop"
}

@test "script_analyzer::extract_script_actions: handles missing script" {
    run script_analyzer::extract_script_actions "$TEST_SCRIPTS_DIR/nonexistent.sh"
    
    assert_failure
    assert_output --partial "Script not found"
}

@test "script_analyzer::extract_script_actions: handles script with no actions" {
    run script_analyzer::extract_script_actions "$TEST_SCRIPTS_DIR/minimal.sh"
    
    assert_success
    # Should have no output (no actions found)
    [[ -z "$output" ]]
}

# =============================================================================
# Error Handling Pattern Tests
# =============================================================================

@test "script_analyzer::check_error_handling_patterns: detects good patterns" {
    run script_analyzer::check_error_handling_patterns "$TEST_SCRIPTS_DIR/manage.sh"
    
    assert_success
    assert_output --partial "FOUND: strict_mode"
    assert_output --partial "FOUND: cleanup_trap"
    assert_output --partial "FOUND: error_logging"
    assert_output --partial "FOUND: exit_codes"
}

@test "script_analyzer::check_error_handling_patterns: detects missing patterns" {
    run script_analyzer::check_error_handling_patterns "$TEST_SCRIPTS_DIR/no_error_handling.sh"
    
    assert_failure
    assert_output --partial "MISSING: strict_mode"
    assert_output --partial "MISSING: cleanup_trap"
    assert_output --partial "MISSING: error_logging"
    assert_output --partial "MISSING: exit_codes"
}

@test "script_analyzer::check_error_handling_patterns: handles missing script" {
    run script_analyzer::check_error_handling_patterns "$TEST_SCRIPTS_DIR/nonexistent.sh"
    
    assert_failure
    assert_output --partial "MISSING: Script not found"
}

# =============================================================================
# Help Pattern Tests
# =============================================================================

@test "script_analyzer::validate_help_patterns: detects help patterns" {
    run script_analyzer::validate_help_patterns "$TEST_SCRIPTS_DIR/manage.sh"
    
    assert_success
    assert_output --partial "FOUND: --help"
    assert_output --partial "FOUND: -h"
    assert_output --partial "FOUND: --version"
}

@test "script_analyzer::validate_help_patterns: detects missing help patterns" {
    run script_analyzer::validate_help_patterns "$TEST_SCRIPTS_DIR/minimal.sh"
    
    assert_failure
    assert_output --partial "MISSING: --help"
    assert_output --partial "MISSING: -h"
    assert_output --partial "MISSING: --version"
}

# =============================================================================
# Required Files Tests
# =============================================================================

@test "script_analyzer::check_required_files: validates file structure" {
    run script_analyzer::check_required_files "$TEST_SCRIPTS_DIR/manage.sh"
    
    assert_success
    assert_output --partial "FOUND: config/"
    assert_output --partial "FOUND: lib/"
    assert_output --partial "FOUND: config/defaults.sh"
    assert_output --partial "FOUND: config/messages.sh"
    assert_output --partial "FOUND: lib/common.sh"
}

@test "script_analyzer::check_required_files: detects missing files" {
    # Create a script in a directory without required files
    mkdir -p "$TEST_SCRIPTS_DIR/incomplete"
    cp "$TEST_SCRIPTS_DIR/manage.sh" "$TEST_SCRIPTS_DIR/incomplete/manage.sh"
    
    run script_analyzer::check_required_files "$TEST_SCRIPTS_DIR/incomplete/manage.sh"
    
    assert_failure
    assert_output --partial "MISSING: config/"
    assert_output --partial "MISSING: lib/"
}

# =============================================================================
# Argument Pattern Tests
# =============================================================================

@test "script_analyzer::analyze_argument_patterns: detects good patterns" {
    run script_analyzer::analyze_argument_patterns "$TEST_SCRIPTS_DIR/manage.sh"
    
    assert_success
    assert_output --partial "GOOD: action_flag"
    assert_output --partial "GOOD: yes_flag"
    assert_output --partial "GOOD: case_statement"
    assert_output --partial "GOOD: argument_validation"
}

@test "script_analyzer::analyze_argument_patterns: detects missing patterns" {
    run script_analyzer::analyze_argument_patterns "$TEST_SCRIPTS_DIR/minimal.sh"
    
    assert_failure
    assert_output --partial "ISSUE: missing_action_flag"
    assert_output --partial "ISSUE: missing_yes_flag"
    assert_output --partial "ISSUE: missing_case_statement"
    assert_output --partial "ISSUE: missing_argument_validation"
}

# =============================================================================
# Configuration Loading Tests
# =============================================================================

@test "script_analyzer::check_configuration_loading: detects proper sourcing" {
    run script_analyzer::check_configuration_loading "$TEST_SCRIPTS_DIR/manage.sh"
    
    assert_success
    assert_output --partial "FOUND: defaults_sourced"
    assert_output --partial "FOUND: messages_sourced"
    assert_output --partial "FOUND: common_sourced"
    assert_output --partial "FOUND: defaults_exists"
}

@test "script_analyzer::check_configuration_loading: detects missing sourcing" {
    run script_analyzer::check_configuration_loading "$TEST_SCRIPTS_DIR/minimal.sh"
    
    assert_failure
    assert_output --partial "MISSING: defaults_sourced"
    assert_output --partial "MISSING: messages_sourced"
    assert_output --partial "MISSING: common_sourced"
}

# =============================================================================
# Function Definition Tests
# =============================================================================

@test "script_analyzer::extract_function_definitions: extracts functions" {
    run script_analyzer::extract_function_definitions "$TEST_SCRIPTS_DIR/lib/common.sh"
    
    assert_success
    assert_line "show_help"
    assert_line "log_error"
}

@test "script_analyzer::extract_function_definitions: handles no functions" {
    run script_analyzer::extract_function_definitions "$TEST_SCRIPTS_DIR/minimal.sh"
    
    assert_success
    # Should have no output (no functions found)
    [[ -z "$output" ]]
}

# =============================================================================
# Script Basics Tests
# =============================================================================

@test "script_analyzer::check_script_basics: validates good script" {
    run script_analyzer::check_script_basics "$TEST_SCRIPTS_DIR/manage.sh"
    
    assert_success
    assert_output --partial "GOOD: bash_shebang"
    assert_output --partial "GOOD: executable"
    assert_output --partial "GOOD: not_empty"
}

@test "script_analyzer::check_script_basics: detects missing shebang" {
    run script_analyzer::check_script_basics "$TEST_SCRIPTS_DIR/invalid.sh"
    
    assert_failure
    assert_output --partial "ISSUE: missing_bash_shebang"
    assert_output --partial "ISSUE: not_executable"
}

@test "script_analyzer::check_script_basics: detects non-executable script" {
    chmod -x "$TEST_SCRIPTS_DIR/invalid.sh"
    
    run script_analyzer::check_script_basics "$TEST_SCRIPTS_DIR/invalid.sh"
    
    assert_failure
    assert_output --partial "ISSUE: not_executable"
}

# =============================================================================
# Comprehensive Analysis Tests
# =============================================================================

@test "script_analyzer::analyze_script_comprehensive: full analysis of good script" {
    run script_analyzer::analyze_script_comprehensive "$TEST_SCRIPTS_DIR/manage.sh"
    
    assert_success
    assert_output --partial "Script analysis PASSED"
    assert_output --partial "Checks passed:"
    assert_output --partial "Checks failed:"
}

@test "script_analyzer::analyze_script_comprehensive: full analysis of bad script" {
    run script_analyzer::analyze_script_comprehensive "$TEST_SCRIPTS_DIR/minimal.sh"
    
    assert_failure
    assert_output --partial "Script analysis FAILED"
    assert_output --partial "Checks passed:"
    assert_output --partial "Checks failed:"
}

# =============================================================================
# Script Metrics Tests
# =============================================================================

@test "script_analyzer::get_script_metrics: generates metrics" {
    run script_analyzer::get_script_metrics "$TEST_SCRIPTS_DIR/manage.sh"
    
    assert_success
    assert_output --partial "Script Metrics"
    assert_output --partial "Lines of code:"
    assert_output --partial "Functions defined:"
    assert_output --partial "Actions implemented:"
    assert_output --partial "File size:"
}

@test "script_analyzer::get_script_metrics: handles missing script" {
    run script_analyzer::get_script_metrics "$TEST_SCRIPTS_DIR/nonexistent.sh"
    
    assert_failure
    assert_output --partial "Script not found"
}

# =============================================================================
# Edge Case Tests
# =============================================================================

@test "edge_case: handles empty script" {
    touch "$TEST_SCRIPTS_DIR/empty.sh"
    chmod +x "$TEST_SCRIPTS_DIR/empty.sh"
    
    run script_analyzer::check_script_basics "$TEST_SCRIPTS_DIR/empty.sh"
    
    assert_failure
    assert_output --partial "ISSUE: empty_file"
}

@test "edge_case: handles very large script" {
    # Create a large script
    {
        echo "#!/usr/bin/env bash"
        echo "set -euo pipefail"
        for i in {1..1000}; do
            echo "echo 'Line $i'"
        done
    } > "$TEST_SCRIPTS_DIR/large.sh"
    chmod +x "$TEST_SCRIPTS_DIR/large.sh"
    
    run script_analyzer::get_script_metrics "$TEST_SCRIPTS_DIR/large.sh"
    
    assert_success
    assert_output --partial "Lines of code: 1002"
}

@test "edge_case: handles script with special characters in name" {
    local special_script="$TEST_SCRIPTS_DIR/script-with-special_chars.sh"
    cp "$TEST_SCRIPTS_DIR/manage.sh" "$special_script"
    
    run script_analyzer::check_script_basics "$special_script"
    
    assert_success
}

# =============================================================================
# Integration Tests
# =============================================================================

@test "integration: analyzes real manage.sh script" {
    # Test with an actual cli.sh script if it exists
    local real_manage="${BATS_TEST_DIRNAME}/../../../ai/ollama/cli.sh"
    
    if [[ -f "$real_manage" ]]; then
        run script_analyzer::extract_script_actions "$real_manage"
        
        assert_success
        # Should find some standard actions
        assert_line "install"
        assert_line "start"
        assert_line "stop"
        assert_line "status"
    else
        skip "Real cli.sh script not found"
    fi
}

@test "integration: validates real script structure" {
    local real_manage="${BATS_TEST_DIRNAME}/../../../ai/ollama/cli.sh"
    
    if [[ -f "$real_manage" ]]; then
        run script_analyzer::check_required_files "$real_manage"
        
        assert_success
        assert_output --partial "FOUND: config/"
        assert_output --partial "FOUND: lib/"
    else
        skip "Real cli.sh script not found"
    fi
}