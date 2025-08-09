#!/usr/bin/env bats
# Comprehensive Tests for Argument Parsing Utilities
# Tests all functionality of scripts/lib/utils/args.sh

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    vrooli_setup_unit_test
    
    # Source the args utility
    source "${BATS_TEST_DIRNAME}/args.sh"
    
    # Clear any existing ARGS array and reinitialize it properly
    unset ARGS 2>/dev/null || true
    declare -gA ARGS
}

teardown() {
    vrooli_cleanup_test
    
    # Clean up ARGS array
    unset ARGS
}

#######################################
# Test args::parse function
#######################################

@test "args::parse handles long options with equals format" {
    # Arrange
    local test_args=("--environment=production" "--version=1.2.3" "--enable-feature=true")
    
    # Act - Don't use run since args::parse modifies global ARGS
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[environment] ]] || { echo "ARGS[environment] not set"; return 1; }
    [[ -v ARGS[version] ]] || { echo "ARGS[version] not set"; return 1; }
    [[ -v ARGS[enable-feature] ]] || { echo "ARGS[enable-feature] not set"; return 1; }
    
    assert_equal "${ARGS[environment]}" "production"
    assert_equal "${ARGS[version]}" "1.2.3" 
    assert_equal "${ARGS[enable-feature]}" "true"
}

@test "args::parse handles long options with space format" {
    # Arrange
    local test_args=("--environment" "staging" "--port" "3000" "--debug" "false")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[environment] ]] || { echo "ARGS[environment] not set"; return 1; }
    [[ -v ARGS[port] ]] || { echo "ARGS[port] not set"; return 1; }
    [[ -v ARGS[debug] ]] || { echo "ARGS[debug] not set"; return 1; }
    
    assert_equal "${ARGS[environment]}" "staging"
    assert_equal "${ARGS[port]}" "3000"
    assert_equal "${ARGS[debug]}" "false"
}

@test "args::parse handles long options as boolean flags" {
    # Arrange
    local test_args=("--verbose" "--dry-run" "--force")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[verbose] ]] || { echo "ARGS[verbose] not set"; return 1; }
    [[ -v ARGS[dry-run] ]] || { echo "ARGS[dry-run] not set"; return 1; }
    [[ -v ARGS[force] ]] || { echo "ARGS[force] not set"; return 1; }
    
    assert_equal "${ARGS[verbose]}" "true"
    assert_equal "${ARGS[dry-run]}" "true"
    assert_equal "${ARGS[force]}" "true"
}

@test "args::parse handles short options with values" {
    # Arrange
    local test_args=("-e" "development" "-p" "8080" "-v" "2.1.0")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[e] ]] || { echo "ARGS[e] not set"; return 1; }
    [[ -v ARGS[p] ]] || { echo "ARGS[p] not set"; return 1; }
    [[ -v ARGS[v] ]] || { echo "ARGS[v] not set"; return 1; }
    
    assert_equal "${ARGS[e]}" "development"
    assert_equal "${ARGS[p]}" "8080"
    assert_equal "${ARGS[v]}" "2.1.0"
}

@test "args::parse handles short options as boolean flags" {
    # Arrange
    local test_args=("-v" "-d" "-f")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[v] ]] || { echo "ARGS[v] not set"; return 1; }
    [[ -v ARGS[d] ]] || { echo "ARGS[d] not set"; return 1; }
    [[ -v ARGS[f] ]] || { echo "ARGS[f] not set"; return 1; }
    
    assert_equal "${ARGS[v]}" "true"
    assert_equal "${ARGS[d]}" "true"
    assert_equal "${ARGS[f]}" "true"
}

@test "args::parse handles mixed argument formats" {
    # Arrange
    local test_args=("--environment=production" "-p" "8080" "--verbose" "-f" "--version" "1.0.0")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[environment] ]] || { echo "ARGS[environment] not set"; return 1; }
    [[ -v ARGS[p] ]] || { echo "ARGS[p] not set"; return 1; }
    [[ -v ARGS[verbose] ]] || { echo "ARGS[verbose] not set"; return 1; }
    [[ -v ARGS[f] ]] || { echo "ARGS[f] not set"; return 1; }
    [[ -v ARGS[version] ]] || { echo "ARGS[version] not set"; return 1; }
    
    assert_equal "${ARGS[environment]}" "production"
    assert_equal "${ARGS[p]}" "8080"
    assert_equal "${ARGS[verbose]}" "true"
    assert_equal "${ARGS[f]}" "true"
    assert_equal "${ARGS[version]}" "1.0.0"
}

@test "args::parse ignores positional arguments" {
    # Arrange
    local test_args=("--environment=test" "positional1" "--port" "3000" "positional2")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[environment] ]] || { echo "ARGS[environment] not set"; return 1; }
    [[ -v ARGS[port] ]] || { echo "ARGS[port] not set"; return 1; }
    
    assert_equal "${ARGS[environment]}" "test"
    assert_equal "${ARGS[port]}" "3000"
    # Positional arguments should not be in ARGS
    [[ ! -v ARGS[positional1] ]]
    [[ ! -v ARGS[positional2] ]]
}

@test "args::parse handles empty arguments" {
    # Arrange
    local test_args=()
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    # ARGS should be empty
    assert_equal "$(echo "${!ARGS[@]}")" ""
}

@test "args::parse handles arguments with spaces in values" {
    # Arrange
    local test_args=("--message=Hello World" "--path" "/home/user/My Documents")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[message] ]] || { echo "ARGS[message] not set"; return 1; }
    [[ -v ARGS[path] ]] || { echo "ARGS[path] not set"; return 1; }
    
    assert_equal "${ARGS[message]}" "Hello World"
    assert_equal "${ARGS[path]}" "/home/user/My Documents"
}

@test "args::parse handles arguments with special characters" {
    # Arrange
    local test_args=("--regex=^[a-zA-Z]+$" "--password" "p@ssw0rd!" "--url=https://example.com/path?param=value")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[regex] ]] || { echo "ARGS[regex] not set"; return 1; }
    [[ -v ARGS[password] ]] || { echo "ARGS[password] not set"; return 1; }
    [[ -v ARGS[url] ]] || { echo "ARGS[url] not set"; return 1; }
    
    assert_equal "${ARGS[regex]}" "^[a-zA-Z]+$"
    assert_equal "${ARGS[password]}" "p@ssw0rd!"
    assert_equal "${ARGS[url]}" "https://example.com/path?param=value"
}

@test "args::parse handles consecutive flag options" {
    # Arrange
    local test_args=("--flag1" "--flag2" "--flag3")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[flag1] ]] || { echo "ARGS[flag1] not set"; return 1; }
    [[ -v ARGS[flag2] ]] || { echo "ARGS[flag2] not set"; return 1; }
    [[ -v ARGS[flag3] ]] || { echo "ARGS[flag3] not set"; return 1; }
    
    assert_equal "${ARGS[flag1]}" "true"
    assert_equal "${ARGS[flag2]}" "true"
    assert_equal "${ARGS[flag3]}" "true"
}

@test "args::parse overwrites duplicate keys with latest value" {
    # Arrange
    local test_args=("--env=dev" "--env=staging" "--env=production")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[env] ]] || { echo "ARGS[env] not set"; return 1; }
    
    assert_equal "${ARGS[env]}" "production"
}

#######################################
# Test args::get function
#######################################

@test "args::get returns existing argument value" {
    # Arrange
    ARGS["environment"]="production"
    ARGS["port"]="8080"
    
    # Act
    run args::get "environment"
    local env_result="$output"
    
    run args::get "port"
    local port_result="$output"
    
    # Assert
    assert_equal "$env_result" "production"
    assert_equal "$port_result" "8080"
}

@test "args::get returns default value when argument does not exist" {
    # Arrange - ARGS is empty
    
    # Act
    run args::get "nonexistent" "default_value"
    local result="$output"
    
    # Assert
    assert_equal "$result" "default_value"
}

@test "args::get returns empty string when argument does not exist and no default provided" {
    # Arrange - ARGS is empty
    
    # Act
    run args::get "nonexistent"
    local result="$output"
    
    # Assert
    assert_equal "$result" ""
}

@test "args::get handles arguments with empty values" {
    # Arrange
    ARGS["empty_arg"]=""
    
    # Act
    run args::get "empty_arg" "default"
    local result="$output"
    
    # Assert
    assert_equal "$result" ""
}

@test "args::get handles arguments with whitespace values" {
    # Arrange
    ARGS["spaces"]="   "
    ARGS["tabs"]=$'\t\t\t'
    ARGS["mixed_whitespace"]=$' \t \n \t '
    
    # Act & Assert - Test spaces
    run args::get "spaces"
    assert_equal "$output" "   "
    
    # Test tabs
    run args::get "tabs"
    assert_equal "$output" $'\t\t\t'
    
    # Test mixed whitespace (capture directly to avoid potential output trimming)
    local mixed_result
    mixed_result=$(args::get "mixed_whitespace")
    assert_equal "$mixed_result" $' \t \n \t '
}

#######################################
# Test args::has function
#######################################

@test "args::has returns success when argument exists" {
    # Arrange
    ARGS["existing"]="value"
    ARGS["flag"]="true"
    ARGS["empty"]=""
    
    # Act & Assert
    run args::has "existing"
    assert_success
    
    run args::has "flag"
    assert_success
    
    run args::has "empty"
    assert_success
}

@test "args::has returns failure when argument does not exist" {
    # Arrange - ARGS has some values but not the tested one
    ARGS["existing"]="value"
    
    # Act & Assert
    run args::has "nonexistent"
    assert_failure
}

@test "args::has works with various key formats" {
    # Arrange
    ARGS["simple"]="value1"
    ARGS["with-dashes"]="value2"
    ARGS["with_underscores"]="value3"
    ARGS["123numeric"]="value4"
    
    # Act & Assert
    run args::has "simple"
    assert_success
    
    run args::has "with-dashes"
    assert_success
    
    run args::has "with_underscores"
    assert_success
    
    run args::has "123numeric"
    assert_success
}

#######################################
# Test args::keys function
#######################################

@test "args::keys returns all argument keys" {
    # Arrange
    ARGS["env"]="production"
    ARGS["port"]="8080"
    ARGS["debug"]="true"
    
    # Act
    run args::keys
    local result="$output"
    
    # Assert
    assert_success
    # Keys should contain all three keys (order may vary)
    assert_output --partial "env"
    assert_output --partial "port"
    assert_output --partial "debug"
}

@test "args::keys returns empty string when no arguments" {
    # Arrange - ARGS is empty (cleared in setup)
    
    # Act
    run args::keys
    
    # Assert
    assert_success
    assert_output ""
}

@test "args::keys returns keys in consistent format" {
    # Arrange
    ARGS["key1"]="value1"
    ARGS["key-with-dashes"]="value2"
    ARGS["key_with_underscores"]="value3"
    
    # Act
    run args::keys
    local keys="$output"
    
    # Assert
    assert_success
    # All three keys should be present
    echo "$keys" | grep -q "key1"
    echo "$keys" | grep -q "key-with-dashes"
    echo "$keys" | grep -q "key_with_underscores"
}

#######################################
# Test integration scenarios
#######################################

@test "integration: complete argument parsing workflow" {
    # Arrange
    local test_args=("--environment=production" "--port" "8080" "--verbose" "-d" "--config=/etc/app.conf")
    
    # Act - Parse arguments
    args::parse "${test_args[@]}"
    
    # Test args::has functionality
    run args::has "environment"
    assert_success
    
    run args::has "nonexistent"
    assert_failure
    
    # Test args::get functionality
    run args::get "environment"
    assert_output "production"
    
    run args::get "port"
    assert_output "8080"
    
    run args::get "nonexistent" "default"
    assert_output "default"
    
    # Test boolean flags
    run args::get "verbose"
    assert_output "true"
    
    run args::get "d"
    assert_output "true"
    
    # Test args::keys functionality
    run args::keys
    local keys="$output"
    echo "$keys" | grep -q "environment"
    echo "$keys" | grep -q "port"
    echo "$keys" | grep -q "verbose"
    echo "$keys" | grep -q "d"
    echo "$keys" | grep -q "config"
}

@test "integration: real-world deployment scenario" {
    # Arrange - Simulate real deployment command
    local deployment_args=(
        "--environment=production"
        "--version=2.1.0"
        "--replicas=3"
        "--memory=2Gi"
        "--cpu=1000m"
        "--enable-monitoring"
        "--dry-run"
        "--config-file=/path/to/config.yaml"
        "--namespace=vrooli-prod"
        "--wait-timeout=600"
    )
    
    # Act
    args::parse "${deployment_args[@]}"
    
    # Assert - Verify all arguments are parsed correctly
    assert_equal "${ARGS[environment]}" "production"
    assert_equal "${ARGS[version]}" "2.1.0"
    assert_equal "${ARGS[replicas]}" "3"
    assert_equal "${ARGS[memory]}" "2Gi"
    assert_equal "${ARGS[cpu]}" "1000m"
    assert_equal "${ARGS[enable-monitoring]}" "true"
    assert_equal "${ARGS[dry-run]}" "true"
    assert_equal "${ARGS[config-file]}" "/path/to/config.yaml"
    assert_equal "${ARGS[namespace]}" "vrooli-prod"
    assert_equal "${ARGS[wait-timeout]}" "600"
    
    # Verify keys count
    run args::keys
    local key_count=$(echo "$output" | wc -w)
    assert_equal "$key_count" "10"
}

@test "integration: argument validation and defaults" {
    # Arrange
    local test_args=("--port=8080" "--debug")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Test with defaults for missing values
    run args::get "port"
    assert_output "8080"
    
    run args::get "host" "localhost"
    assert_output "localhost"
    
    run args::get "timeout" "30"
    assert_output "30"
    
    run args::get "debug"
    assert_output "true"
    
    run args::get "verbose" "false"
    assert_output "false"
}

#######################################
# Test edge cases and error conditions
#######################################

@test "edge case: arguments with equals in values" {
    # Arrange
    local test_args=("--query=SELECT * FROM users WHERE id=123" "--url=http://localhost:8080/api/v1/data?param=value")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[query] ]] || { echo "ARGS[query] not set"; return 1; }
    [[ -v ARGS[url] ]] || { echo "ARGS[url] not set"; return 1; }
    
    assert_equal "${ARGS[query]}" "SELECT * FROM users WHERE id=123"
    assert_equal "${ARGS[url]}" "http://localhost:8080/api/v1/data?param=value"
}

@test "edge case: empty values in different formats" {
    # Arrange
    local test_args=("--empty=" "--also-empty" "")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[empty] ]] || { echo "ARGS[empty] not set"; return 1; }
    [[ -v ARGS[also-empty] ]] || { echo "ARGS[also-empty] not set"; return 1; }
    
    assert_equal "${ARGS[empty]}" ""
    assert_equal "${ARGS[also-empty]}" ""
}

@test "edge case: numeric and boolean-like values" {
    # Arrange
    local test_args=("--number=42" "--float=3.14" "--bool-true=true" "--bool-false=false" "--zero=0")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[number] ]] || { echo "ARGS[number] not set"; return 1; }
    [[ -v ARGS[float] ]] || { echo "ARGS[float] not set"; return 1; }
    [[ -v ARGS[bool-true] ]] || { echo "ARGS[bool-true] not set"; return 1; }
    [[ -v ARGS[bool-false] ]] || { echo "ARGS[bool-false] not set"; return 1; }
    [[ -v ARGS[zero] ]] || { echo "ARGS[zero] not set"; return 1; }
    
    assert_equal "${ARGS[number]}" "42"
    assert_equal "${ARGS[float]}" "3.14"
    assert_equal "${ARGS[bool-true]}" "true"
    assert_equal "${ARGS[bool-false]}" "false"
    assert_equal "${ARGS[zero]}" "0"
}

@test "edge case: very long argument names and values" {
    # Arrange
    local long_name="very-long-argument-name-with-many-dashes-and-characters"
    local long_value="this-is-a-very-long-value-that-might-be-used-in-real-scenarios-like-file-paths-or-urls-or-other-lengthy-strings"
    local test_args=("--$long_name=$long_value")
    
    # Act
    args::parse "${test_args[@]}"
    
    # Assert
    [[ -v ARGS[$long_name] ]] || { echo "ARGS[$long_name] not set"; return 1; }
    
    assert_equal "${ARGS[$long_name]}" "$long_value"
    
    run args::has "$long_name"
    assert_success
    
    run args::get "$long_name"
    assert_output "$long_value"
}

#######################################
# Test function exports and availability
#######################################

@test "functions are properly exported and available" {
    # Test that all main functions are available
    declare -f args::parse >/dev/null || { echo "args::parse function not available"; return 1; }
    declare -f args::get >/dev/null || { echo "args::get function not available"; return 1; }
    declare -f args::has >/dev/null || { echo "args::has function not available"; return 1; }
    declare -f args::keys >/dev/null || { echo "args::keys function not available"; return 1; }
}

@test "ARGS array is accessible after sourcing" {
    # Arrange - Verify ARGS is declared as an associative array
    declare -p ARGS >/dev/null 2>&1 || { echo "ARGS not declared"; return 1; }
    
    # Test that we can set and access values
    ARGS["test"]="value"
    
    # Assert
    assert_equal "${ARGS[test]}" "value"
    
    # Test that the array behaves like an associative array
    [[ -v ARGS[test] ]] || { echo "ARGS[test] not accessible"; return 1; }
}