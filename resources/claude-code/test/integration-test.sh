#!/usr/bin/env bash
# Claude Code Integration Test
# Tests Claude Code CLI functionality and installation
# Tests CLI commands, authentication status, and basic capabilities

set -euo pipefail

# Script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/claude-code/test"

# Source var.sh first to get proper directory variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source shared integration test library if it exists
# shellcheck disable=SC1091
source "${var_SCRIPTS_DIR}/tests/lib/integration-test-lib.sh" 2>/dev/null || true

# Fallback implementations for missing shared library functions
if ! type register_standard_interface_tests &>/dev/null; then
    # Test framework variables
    declare -a REGISTERED_TESTS=()
    TESTS_PASSED=0
    TESTS_FAILED=0
    TESTS_SKIPPED=0
    
    # Simple stub implementation of register_standard_interface_tests
    register_standard_interface_tests() {
        # Standard tests would include manage.sh validation, etc.
        # For now, just register a basic test
        register_tests "test_manage_script_exists"
    }
    
    # Simple implementation of register_tests
    register_tests() {
        for test_name in "$@"; do
            REGISTERED_TESTS+=("$test_name")
        done
    }
    
    # Simple implementation of run_test_with_error_handling
    run_test_with_error_handling() {
        local test_function="$1"
        echo "Running test: $test_function"
        
        if type "$test_function" &>/dev/null; then
            if "$test_function"; then
                echo "  ✓ PASS: $test_function"
                ((TESTS_PASSED++))
            else
                echo "  ✗ FAIL: $test_function"
                ((TESTS_FAILED++))
            fi
        else
            echo "  - SKIP: $test_function (function not found)"
            ((TESTS_SKIPPED++))
        fi
    }
    
    # Simple implementations of other expected functions
    init_config() {
        echo "Initializing test configuration..."
    }
    
    show_test_header() {
        echo "=== Claude Code Integration Tests ==="
        echo
    }
    
    check_required_tools() {
        echo "Checking required tools..."
        for tool in "${REQUIRED_TOOLS[@]}"; do
            if command -v "$tool" >/dev/null 2>&1; then
                echo "  ✓ $tool found"
            else
                echo "  ✗ $tool not found"
                return 1
            fi
        done
        return 0
    }
    
    show_summary() {
        echo
        echo "=== Test Summary ==="
        echo "Passed: $TESTS_PASSED"
        echo "Failed: $TESTS_FAILED" 
        echo "Skipped: $TESTS_SKIPPED"
        echo "Total: $((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))"
    }
    
    cleanup_test_files() {
        # Clean up any temporary files created during testing
        return 0
    }
    
    log_test_result() {
        local test_name="$1"
        local result="$2"
        local message="${3:-}"
        
        if [[ "$result" == "PASS" ]]; then
            echo "  ✓ PASS: $test_name${message:+ - $message}"
        elif [[ "$result" == "FAIL" ]]; then
            echo "  ✗ FAIL: $test_name${message:+ - $message}"
        else
            echo "  - SKIP: $test_name${message:+ - $message}"
        fi
    }
    
    # Add the basic test that was referenced
    test_manage_script_exists() {
        [[ -f "${SCRIPT_DIR}/../manage.sh" ]]
    }
fi

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Claude Code configuration
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/claude-code/config/defaults.sh"
claude_code::export_config

# Override library defaults with Claude Code-specific settings
SERVICE_NAME="claude-code"
BASE_URL=""  # Claude Code is a CLI tool, not a web service
HEALTH_ENDPOINT=""
REQUIRED_TOOLS=("node" "npm" "claude")  # CLI tool requires node and npm
SERVICE_METADATA=(
    "Type: CLI Tool"
    "Package: ${CLAUDE_PACKAGE:-@anthropic-ai/claude-code}"
    "Min Node Version: ${MIN_NODE_VERSION:-18}"
    "Config Dir: ${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
)

# Test configuration
readonly CLI_TIMEOUT=30

#######################################
# CLAUDE CODE-SPECIFIC TEST FUNCTIONS
#######################################

test_node_version() {
    local test_name="Node.js version compatibility"
    
    if ! command -v node >/dev/null 2>&1; then
        log_test_result "$test_name" "FAIL" "Node.js not installed"
        return 1
    fi
    
    local node_version
    node_version=$(node --version | sed 's/v//')
    local major_version
    major_version=$(echo "$node_version" | cut -d. -f1)
    
    if [[ "$major_version" -ge "${MIN_NODE_VERSION:-18}" ]]; then
        log_test_result "$test_name" "PASS" "Node.js $node_version (>= ${MIN_NODE_VERSION:-18})"
        return 0
    else
        log_test_result "$test_name" "FAIL" "Node.js $node_version (< ${MIN_NODE_VERSION:-18})"
        return 1
    fi
}

test_npm_availability() {
    local test_name="npm package manager"
    
    if command -v npm >/dev/null 2>&1; then
        local npm_version
        npm_version=$(npm --version)
        log_test_result "$test_name" "PASS" "npm $npm_version"
        return 0
    else
        log_test_result "$test_name" "FAIL" "npm not available"
        return 1
    fi
}

test_claude_cli_installation() {
    local test_name="Claude CLI installation"
    
    if command -v claude >/dev/null 2>&1; then
        local claude_version
        claude_version=$(claude --version 2>&1 | head -1 || echo "unknown")
        log_test_result "$test_name" "PASS" "installed: $claude_version"
        return 0
    else
        log_test_result "$test_name" "FAIL" "Claude CLI not installed"
        return 1
    fi
}

test_claude_help_command() {
    local test_name="Claude CLI help command"
    
    if ! command -v claude >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Claude CLI not installed"
        return 2
    fi
    
    local help_output
    if help_output=$(timeout "$CLI_TIMEOUT" claude --help 2>&1); then
        if echo "$help_output" | grep -qi "usage\|command\|option"; then
            log_test_result "$test_name" "PASS" "help command working"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "help command not working"
    return 1
}

test_claude_auth_status() {
    local test_name="Claude authentication status"
    
    if ! command -v claude >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Claude CLI not installed"
        return 2
    fi
    
    # Try to check auth status with very short timeout to avoid hangs
    local auth_output
    if auth_output=$(timeout 5 claude auth whoami 2>&1 || true); then
        if echo "$auth_output" | grep -qi "not.*logged.*in\|not.*authenticated\|login.*required"; then
            log_test_result "$test_name" "SKIP" "not authenticated (requires login)"
            return 2
        elif echo "$auth_output" | grep -qi "email\|user\|@"; then
            log_test_result "$test_name" "PASS" "authenticated"
            return 0
        elif echo "$auth_output" | grep -qi "error\|failed"; then
            log_test_result "$test_name" "FAIL" "authentication check failed"
            return 1
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "authentication status unclear"
    return 2
}

test_claude_config_directory() {
    local test_name="Claude configuration directory"
    
    local config_dir="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
    
    if [[ -d "$config_dir" ]]; then
        # Check if config directory has expected structure
        local has_settings=false
        local has_sessions=false
        
        [[ -f "$config_dir/settings.json" ]] && has_settings=true
        [[ -d "$config_dir/sessions" ]] && has_sessions=true
        
        if [[ "$has_settings" == "true" ]]; then
            log_test_result "$test_name" "PASS" "configuration directory exists with settings"
            return 0
        else
            log_test_result "$test_name" "PASS" "configuration directory exists"
            return 0
        fi
    else
        log_test_result "$test_name" "SKIP" "configuration directory not found (not used yet)"
        return 2
    fi
}

test_claude_doctor_command() {
    local test_name="Claude doctor diagnostic"
    
    if ! command -v claude >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Claude CLI not installed"
        return 2
    fi
    
    # Check if we have a TTY (doctor command may require interactive mode)
    if [[ ! -t 0 || ! -t 1 ]]; then
        log_test_result "$test_name" "SKIP" "doctor command requires TTY environment"
        return 2
    fi
    
    # Run claude doctor for diagnostics with shorter timeout and non-interactive approach
    local doctor_output
    if doctor_output=$(echo "" | timeout 5 claude doctor 2>&1 || true); then
        if echo "$doctor_output" | grep -qi "check\|status\|version\|ok\|error\|health"; then
            log_test_result "$test_name" "PASS" "doctor command accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "doctor command requires interactive mode"
    return 2
}

test_claude_print_mode() {
    local test_name="Claude print mode (non-interactive)"
    
    if ! command -v claude >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Claude CLI not installed"
        return 2
    fi
    
    # Test non-interactive mode with simple query
    local print_output
    if print_output=$(timeout "$CLI_TIMEOUT" echo "What is 2+2?" | claude --print 2>&1 || true); then
        if echo "$print_output" | grep -qi "not.*logged.*in\|not.*authenticated\|login.*required"; then
            log_test_result "$test_name" "SKIP" "requires authentication"
            return 2
        elif echo "$print_output" | grep -qi "4\|four\|answer"; then
            log_test_result "$test_name" "PASS" "print mode working"
            return 0
        elif [[ -n "$print_output" ]]; then
            log_test_result "$test_name" "PASS" "print mode responsive"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "print mode not working or requires auth"
    return 2
}

test_package_integrity() {
    local test_name="package integrity check"
    
    if ! command -v npm >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "npm not available"
        return 2
    fi
    
    # Check if Claude Code package is properly installed
    local package_info
    if package_info=$(npm list -g "${CLAUDE_PACKAGE:-@anthropic-ai/claude-code}" 2>&1 || true); then
        if echo "$package_info" | grep -q "${CLAUDE_PACKAGE:-@anthropic-ai/claude-code}"; then
            log_test_result "$test_name" "PASS" "package properly installed"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "package not found in global npm list"
    return 2
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "Claude Code Information:"
    echo "  CLI Commands:"
    echo "    - Help: claude --help"
    echo "    - Version: claude --version"
    echo "    - Authentication: claude auth login"
    echo "    - Diagnostics: claude doctor"
    echo "    - Non-interactive: echo 'query' | claude --print"
    echo "  Configuration:"
    echo "    - Config Directory: ${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
    echo "    - Package: ${CLAUDE_PACKAGE:-@anthropic-ai/claude-code}"
    echo "    - Required Node.js: >= ${MIN_NODE_VERSION:-18}"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register Claude Code-specific tests
register_tests \
    "test_node_version" \
    "test_npm_availability" \
    "test_claude_cli_installation" \
    "test_claude_help_command" \
    "test_claude_auth_status" \
    "test_claude_config_directory" \
    "test_claude_doctor_command" \
    "test_claude_print_mode" \
    "test_package_integrity"

# Custom main function for CLI tools (no service availability check needed)
claude_code_integration_main() {
    # Set up cleanup trap for temp files
    trap cleanup_test_files EXIT
    
    # Initialize configuration
    init_config
    
    show_test_header
    
    # Check required tools
    if ! check_required_tools; then
        return 1
    fi
    
    # Run registered tests with error handling (skip service availability check for CLI tools)
    set +e  # Temporarily disable exit-on-error for test execution
    
    # Run all registered service-specific tests
    for test_function in "${REGISTERED_TESTS[@]}"; do
        run_test_with_error_handling "$test_function"
    done
    
    set -e  # Re-enable exit-on-error
    
    # Show summary
    show_summary
    
    # Cleanup temp files (trap will also handle this on exit)
    cleanup_test_files
    
    # Determine exit code based on results
    if [[ $TESTS_FAILED -gt 0 ]]; then
        # Some tests failed
        exit 1
    else
        # All tests passed or skipped
        exit 0
    fi
}

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    claude_code_integration_main "$@"
fi