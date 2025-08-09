#!/usr/bin/env bats
# Tests for config.sh - System Configuration Utilities

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    vrooli_setup_unit_test
    
    # Source the config utility
    source "${BATS_TEST_DIRNAME}/config.sh"
}

teardown() {
    vrooli_cleanup_test
}

#######################################
# Test config::load function
#######################################

@test "config::load returns success when loading valid config file" {
    # Arrange - Create a test config file
    local test_config="${VROOLI_TEST_TMPDIR:-${BATS_TEST_TMPDIR}}/test.config"
    cat > "$test_config" << 'EOF'
# Test configuration
TEST_VAR1="value1"
TEST_VAR2="value2"
EOF
    
    # Act
    run config::load "$test_config"
    
    # Assert
    assert_success
    assert_equal "$TEST_VAR1" "value1"
    assert_equal "$TEST_VAR2" "value2"
}

@test "config::load returns failure when file does not exist" {
    # Arrange
    local nonexistent_file="/path/that/does/not/exist.config"
    
    # Act
    run config::load "$nonexistent_file"
    
    # Assert
    assert_failure
}

@test "config::load returns failure when no file path provided" {
    # Act
    run config::load
    
    # Assert
    assert_failure
}

@test "config::load returns failure when empty file path provided" {
    # Act  
    run config::load ""
    
    # Assert
    assert_failure
}

@test "config::load handles config files with complex values" {
    # Arrange - Create config with various value types
    local test_config="${VROOLI_TEST_TMPDIR:-${BATS_TEST_TMPDIR}}/complex.config"
    cat > "$test_config" << 'EOF'
# Complex configuration test
SIMPLE_STRING="hello world"
PATH_VALUE="/home/user/documents"
NUMBER_VALUE="42"
BOOLEAN_VALUE="true"
URL_VALUE="https://example.com/api/v1"
MULTILINE_VALUE="line1
line2
line3"
EOF
    
    # Act
    run config::load "$test_config"
    
    # Assert
    assert_success
    assert_equal "$SIMPLE_STRING" "hello world"
    assert_equal "$PATH_VALUE" "/home/user/documents"
    assert_equal "$NUMBER_VALUE" "42"
    assert_equal "$BOOLEAN_VALUE" "true"
    assert_equal "$URL_VALUE" "https://example.com/api/v1"
}

@test "config::load handles empty config files" {
    # Arrange - Create empty config file
    local empty_config="${VROOLI_TEST_TMPDIR:-${BATS_TEST_TMPDIR}}/empty.config"
    touch "$empty_config"
    
    # Act
    run config::load "$empty_config"
    
    # Assert
    assert_success
}

@test "config::load handles config files with comments and blank lines" {
    # Arrange - Create config with comments and formatting
    local test_config="${VROOLI_TEST_TMPDIR:-${BATS_TEST_TMPDIR}}/formatted.config"
    cat > "$test_config" << 'EOF'
# This is a comment
# Another comment

# Configuration section 1
SECTION1_VAR="value1"

# Configuration section 2
SECTION2_VAR="value2"

# End of config
EOF
    
    # Act
    run config::load "$test_config"
    
    # Assert
    assert_success
    assert_equal "$SECTION1_VAR" "value1"
    assert_equal "$SECTION2_VAR" "value2"
}