#!/usr/bin/env bash
# CLI testing utilities
set -euo pipefail

# Source core utilities
SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHELL_DIR/core.sh"

# Test CLI binary existence and basic functionality
testing::cli::test_integration() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    
    echo "🖥️  Testing CLI integration for $scenario_name..."
    
    local cli_binary="cli/$scenario_name"
    
    if [ ! -f "$cli_binary" ]; then
        echo "ℹ️  No CLI binary found at $cli_binary"
        return 0
    fi
    
    if [ ! -x "$cli_binary" ]; then
        echo "❌ CLI binary is not executable"
        return 1
    fi
    
    local cli_test_count=0
    local cli_error_count=0
    
    # Test help command
    echo "  Testing CLI help..."
    if "$cli_binary" --help >/dev/null 2>&1 || "$cli_binary" help >/dev/null 2>&1; then
        echo "    ✅ CLI help command works"
        ((cli_test_count++))
    else
        echo "    ❌ CLI help command failed"
        ((cli_error_count++))
    fi
    
    # Test version command
    echo "  Testing CLI version..."
    if "$cli_binary" --version >/dev/null 2>&1 || "$cli_binary" version >/dev/null 2>&1; then
        echo "    ✅ CLI version command works"
        ((cli_test_count++))
    else
        echo "    ℹ️  CLI version command not available"
    fi
    
    # Run BATS tests if they exist
    local bats_file="cli/${scenario_name}.bats"
    if [ -f "$bats_file" ] && command -v bats >/dev/null 2>&1; then
        echo "  Running CLI BATS tests..."
        if bats "$bats_file" >/dev/null 2>&1; then
            echo "    ✅ CLI BATS tests passed"
            ((cli_test_count++))
        else
            echo "    ❌ CLI BATS tests failed"
            ((cli_error_count++))
        fi
    else
        echo "    ℹ️  No BATS tests found or BATS not available"
    fi
    
    echo "📊 CLI integration summary: $cli_test_count tests passed, $cli_error_count failed"
    
    if [ $cli_error_count -eq 0 ]; then
        return 0
    else
        return 1
    fi
}

# Test specific CLI command
testing::cli::test_command() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    local command="${2:-help}"
    local expected_exit_code="${3:-0}"
    
    local cli_binary="cli/$scenario_name"
    
    if [ ! -f "$cli_binary" ] || [ ! -x "$cli_binary" ]; then
        echo "❌ CLI binary not available or not executable"
        return 1
    fi
    
    echo "Testing CLI command: $command"
    
    set +e  # Don't exit on command failure
    "$cli_binary" $command >/dev/null 2>&1
    local exit_code=$?
    set -e
    
    if [ $exit_code -eq $expected_exit_code ]; then
        echo "✅ Command exited with expected code: $expected_exit_code"
        return 0
    else
        echo "❌ Command exited with $exit_code, expected $expected_exit_code"
        return 1
    fi
}

# Run CLI with input and check output
testing::cli::test_with_input() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    local command="$2"
    local input="$3"
    local expected_output_pattern="$4"
    
    local cli_binary="cli/$scenario_name"
    
    if [ ! -f "$cli_binary" ] || [ ! -x "$cli_binary" ]; then
        echo "❌ CLI binary not available or not executable"
        return 1
    fi
    
    echo "Testing CLI with input..."
    
    local output
    output=$(echo "$input" | "$cli_binary" $command 2>&1)
    
    if echo "$output" | grep -q "$expected_output_pattern"; then
        echo "✅ Output matches expected pattern"
        return 0
    else
        echo "❌ Output does not match expected pattern"
        echo "Expected pattern: $expected_output_pattern"
        echo "Actual output: $output"
        return 1
    fi
}

# Export functions
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    export -f testing::cli::test_integration
    export -f testing::cli::test_command
    export -f testing::cli::test_with_input
fi