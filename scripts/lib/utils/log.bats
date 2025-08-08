#!/usr/bin/env bats
# Tests for log.sh - Color logging functions

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../__test/helpers/bats-assert/load"

# Load the logging library
source "${BATS_TEST_DIRNAME}/log.sh"

setup() {
    vrooli_setup_unit_test
    # Store original TERM value
    ORIGINAL_TERM="${TERM:-}"
}

teardown() {
    vrooli_cleanup_test
    # Restore TERM
    export TERM="$ORIGINAL_TERM"
}

# Test color code mapping
@test "log::get_color_code returns correct codes for all colors" {
    run log::get_color_code "RED"
    assert_success
    assert_output "1"

    run log::get_color_code "GREEN"
    assert_success
    assert_output "2"

    run log::get_color_code "YELLOW"
    assert_success
    assert_output "3"

    run log::get_color_code "BLUE"
    assert_success
    assert_output "4"

    run log::get_color_code "MAGENTA"
    assert_success
    assert_output "5"

    run log::get_color_code "CYAN"
    assert_success
    assert_output "6"

    run log::get_color_code "WHITE"
    assert_success
    assert_output "7"
}

@test "log::get_color_code returns 0 for unknown colors" {
    run log::get_color_code "UNKNOWN"
    assert_success
    assert_output "0"

    run log::get_color_code ""
    assert_success
    assert_output "0"

    run log::get_color_code "purple"
    assert_success
    assert_output "0"
}

# Test basic logging functions
@test "log::header outputs message with HEADER prefix" {
    run log::header "Test Header"
    assert_success
    assert_output_contains "[HEADER]  Test Header"
}

@test "log::subheader outputs message with SECTION prefix" {
    run log::subheader "Test Section"
    assert_success
    assert_output_contains "[SECTION] Test Section"
}

@test "log::info outputs message with INFO prefix" {
    run log::info "Test Info"
    assert_success
    assert_output_contains "[INFO]    Test Info"
}

@test "log::success outputs message with SUCCESS prefix" {
    run log::success "Test Success"
    assert_success
    assert_output_contains "[SUCCESS] Test Success"
}

@test "log::error outputs message with ERROR prefix" {
    run log::error "Test Error"
    assert_success
    assert_output_contains "[ERROR]   Test Error"
}

@test "log::warning outputs message with WARNING prefix" {
    run log::warning "Test Warning"
    assert_success
    assert_output_contains "[WARNING] Test Warning"
}

@test "log::prompt outputs message with PROMPT prefix" {
    run log::prompt "Test Prompt"
    assert_success
    assert_output_contains "[PROMPT]  Test Prompt"
}

# Test logging functions with multiple arguments
@test "log::header handles multiple arguments" {
    run log::header "Test" "Multiple" "Args"
    assert_success
    assert_output_contains "[HEADER]  Test Multiple Args"
}

@test "log::info handles multiple arguments" {
    run log::info "File" "not" "found"
    assert_success
    assert_output_contains "[INFO]    File not found"
}

@test "log::error handles multiple arguments with special characters" {
    run log::error "Failed to process file:" "/path/to/file.txt"
    assert_success
    assert_output_contains "[ERROR]   Failed to process file: /path/to/file.txt"
}

# Test logging functions with empty arguments
@test "log::info handles empty arguments" {
    run log::info ""
    assert_success
    assert_output_contains "[INFO]    "
}

@test "log::error handles no arguments" {
    run log::error
    assert_success
    assert_output_contains "[ERROR]   "
}

# Test alias functions
@test "log::warn is alias for log::warning" {
    run log::warn "Test Warning Alias"
    assert_success
    assert_output_contains "[WARNING] Test Warning Alias"
}

@test "log::warn and log::warning produce identical output" {
    local warn_output warning_output
    warn_output=$(log::warn "test message" 2>&1)
    warning_output=$(log::warning "test message" 2>&1)
    
    [[ "$warn_output" == "$warning_output" ]]
}

# Test debug function
@test "log::debug shows message when DEBUG=true" {
    export DEBUG="true"
    
    run log::debug "Debug message"
    assert_success
    assert_output_contains "[DEBUG]   Debug message"
}

@test "log::debug hides message when DEBUG is false" {
    export DEBUG="false"
    
    run log::debug "Debug message"
    assert_success
    assert_output ""
}

@test "log::debug hides message when DEBUG is unset" {
    unset DEBUG
    
    run log::debug "Debug message"
    assert_success
    assert_output ""
}

@test "log::debug hides message when DEBUG is empty string" {
    export DEBUG=""
    
    run log::debug "Debug message"
    assert_success
    assert_output ""
}

@test "log::debug only shows when DEBUG equals exactly 'true'" {
    export DEBUG="TRUE"
    
    run log::debug "Debug message"
    assert_success
    # This should NOT show the message as the code checks for exact "true"
    assert_output ""
}

# Test special characters and escaping
@test "logging functions handle special characters" {
    run log::info "Message with \$variables and 'quotes' and \"double quotes\""
    assert_success
    assert_output_contains "[INFO]    Message with \$variables and 'quotes' and \"double quotes\""
}

@test "logging functions handle backslashes" {
    run log::error "Path: \\path\\to\\file"
    assert_success
    assert_output_contains "[ERROR]   Path: \\path\\to\\file"
}

# Test TERM environment variable handling
@test "functions work with TERM unset" {
    unset TERM
    
    run log::info "Test message"
    assert_success
    assert_output_contains "[INFO]    Test message"
}

@test "functions work with TERM=dumb" {
    export TERM="dumb"
    
    run log::success "Test message"
    assert_success
    assert_output_contains "[SUCCESS] Test message"
}

@test "functions work with custom TERM value" {
    export TERM="xterm-256color"
    
    run log::header "Test message"
    assert_success
    assert_output_contains "[HEADER]  Test message"
}

# Test error handling
@test "functions handle very long messages" {
    local long_message
    long_message=$(printf "A%.0s" {1..100})  # 100 character message
    
    run log::info "$long_message"
    assert_success
    assert_output_contains "[INFO]"
}

@test "functions handle unicode characters" {
    run log::success "Success: Complete"
    assert_success
    assert_output_contains "[SUCCESS] Success: Complete"
}

# Test basic functionality integration
@test "all log levels produce correct output format" {
    # Test that each function produces output in the expected format
    run log::header "Header Test"
    assert_success
    assert_output_contains "[HEADER]  Header Test"
    
    run log::info "Info Test"
    assert_success
    assert_output_contains "[INFO]    Info Test"
    
    run log::success "Success Test"
    assert_success
    assert_output_contains "[SUCCESS] Success Test"
    
    run log::warning "Warning Test"
    assert_success
    assert_output_contains "[WARNING] Warning Test"
    
    run log::error "Error Test"
    assert_success
    assert_output_contains "[ERROR]   Error Test"
}

# Test that functions don't crash
@test "logging functions are robust" {
    # Ensure functions complete without errors even with edge cases
    run log::info
    assert_success
    
    run log::error ""
    assert_success
    
    run log::warning "test with \n newlines \t tabs"
    assert_success
    assert_output_contains "[WARNING]"
}

# Simple performance test
@test "logging functions execute without hanging" {
    # Simple test to ensure functions don't hang
    run log::info "Message 1"
    assert_success
    
    run log::info "Message 2"
    assert_success
    
    run log::info "Message 3"
    assert_success
}