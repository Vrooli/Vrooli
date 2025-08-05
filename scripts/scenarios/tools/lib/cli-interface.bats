#!/usr/bin/env bats

# Test suite for CLI Interface Module
# Tests command-line argument parsing, logging functions, and usage display

setup() {
    # Get the directory containing this test file - resolve correctly for bats
    if [[ -n "${BATS_TEST_DIRNAME:-}" ]]; then
        TEST_DIR="${BATS_TEST_DIRNAME}"
    else
        TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    fi
    
    # Source the module we're testing
    source "${TEST_DIR}/cli-interface.sh"
    
    # Create temporary directory for test output
    TEST_OUTPUT_DIR="$(mktemp -d)"
}

teardown() {
    # Clean up temporary directory
    [[ -n "$TEST_OUTPUT_DIR" && -d "$TEST_OUTPUT_DIR" ]] && rm -rf "$TEST_OUTPUT_DIR"
}

################################################################################
# Logging Functions Tests
################################################################################

@test "log_info outputs correctly formatted message" {
    run log_info "test message"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "[INFO] test message" ]]
}

@test "log_success outputs correctly formatted message" {
    run log_success "test success"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "[SUCCESS] test success" ]]
}

@test "log_warning outputs correctly formatted message" {
    run log_warning "test warning"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "[WARNING] test warning" ]]
}

@test "log_error outputs to stderr with correct format" {
    run log_error "test error"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "[ERROR] test error" ]]
}

@test "log_step outputs correctly formatted message" {
    run log_step "test step"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "[STEP] test step" ]]
}

@test "log_phase outputs correctly formatted message" {
    run log_phase "test phase"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "=== test phase ===" ]]
}

@test "log_banner outputs with proper spacing" {
    run log_banner "test banner"
    [[ "$status" -eq 0 ]]
    # Check that output contains the banner text with spacing
    [[ "$output" =~ "=== test banner ===" ]]
    # Check that there are multiple lines (indicating spacing)
    local line_count=$(echo "$output" | wc -l)
    [[ "$line_count" -gt 1 ]]
}

################################################################################
# Usage Display Tests
################################################################################

@test "show_usage displays help message" {
    run show_usage
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Scenario-to-App Conversion Script" ]]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Options:" ]]
    [[ "$output" =~ "Examples:" ]]
}

@test "show_usage includes schema reference" {
    run show_usage
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "SCHEMA REFERENCE:" ]]
    [[ "$output" =~ ".vrooli/schemas/service.schema.json" ]]
}

@test "show_usage includes all deployment modes" {
    run show_usage
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "local, docker, k8s" ]]
}

@test "show_usage includes all validation modes" {
    run show_usage
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "none, basic, full" ]]
}

################################################################################
# Argument Parsing Tests - Success Cases
################################################################################

@test "parse_args handles scenario name only" {
    parse_args "test-scenario"
    [[ "$SCENARIO_NAME" == "test-scenario" ]]
    [[ "$DEPLOYMENT_MODE" == "local" ]]  # default
    [[ "$VALIDATION_MODE" == "full" ]]   # default
    [[ "$DRY_RUN" == false ]]           # default
    [[ "$VERBOSE" == false ]]           # default
}

@test "parse_args handles all valid options" {
    parse_args "test-scenario" --mode "docker" --validate "basic" --dry-run --verbose
    [[ "$SCENARIO_NAME" == "test-scenario" ]]
    [[ "$DEPLOYMENT_MODE" == "docker" ]]
    [[ "$VALIDATION_MODE" == "basic" ]]
    [[ "$DRY_RUN" == true ]]
    [[ "$VERBOSE" == true ]]
}

@test "parse_args handles options in different order" {
    parse_args --verbose --mode "k8s" "my-scenario" --validate "none" --dry-run
    [[ "$SCENARIO_NAME" == "my-scenario" ]]
    [[ "$DEPLOYMENT_MODE" == "k8s" ]]
    [[ "$VALIDATION_MODE" == "none" ]]
    [[ "$DRY_RUN" == true ]]
    [[ "$VERBOSE" == true ]]
}

@test "parse_args accepts valid scenario names with hyphens and underscores" {
    parse_args "test-scenario_123"
    [[ "$SCENARIO_NAME" == "test-scenario_123" ]]
}

@test "parse_args accepts scenario names with numbers" {
    parse_args "scenario123"
    [[ "$SCENARIO_NAME" == "scenario123" ]]
}

################################################################################
# Argument Parsing Tests - Help Cases
################################################################################

@test "parse_args shows help and exits successfully with --help" {
    run parse_args --help
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Scenario-to-App Conversion Script" ]]
}

@test "parse_args shows help with scenario and --help" {
    run parse_args "test-scenario" --help
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Usage:" ]]
}

################################################################################
# Argument Parsing Tests - Error Cases
################################################################################

@test "parse_args fails with no arguments" {
    run parse_args
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "No scenario specified" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "parse_args fails with multiple scenario names" {
    run parse_args "scenario1" "scenario2"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Multiple scenario names provided" ]]
}

@test "parse_args fails with unknown option" {
    run parse_args "test-scenario" --invalid-option
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Unknown option: --invalid-option" ]]
}

@test "parse_args fails with invalid deployment mode" {
    run parse_args "test-scenario" --mode "invalid"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Invalid deployment mode: invalid" ]]
    [[ "$output" =~ "Valid modes are: local, docker, k8s" ]]
}

@test "parse_args fails with invalid validation mode" {
    run parse_args "test-scenario" --validate "invalid"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Invalid validation mode: invalid" ]]
    [[ "$output" =~ "Valid modes are: none, basic, full" ]]
}

@test "parse_args fails with mode option but no value" {
    run parse_args "test-scenario" --mode
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Option --mode requires a value" ]]
}

@test "parse_args fails with validate option but no value" {
    run parse_args "test-scenario" --validate
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Option --validate requires a value" ]]
}

@test "parse_args fails with invalid scenario name characters" {
    run parse_args "test@scenario"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Invalid scenario name: test@scenario" ]]
    [[ "$output" =~ "must contain only letters, numbers, hyphens, and underscores" ]]
}

@test "parse_args fails with empty scenario name" {
    run parse_args ""
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Invalid scenario name" ]]
}

################################################################################
# Validation Helper Tests
################################################################################

@test "validate_parsed_args succeeds with all variables set" {
    DEPLOYMENT_MODE="local"
    VALIDATION_MODE="full"
    SCENARIO_NAME="test-scenario"
    
    run validate_parsed_args
    [[ "$status" -eq 0 ]]
}

@test "validate_parsed_args fails with missing DEPLOYMENT_MODE" {
    DEPLOYMENT_MODE=""
    VALIDATION_MODE="full"
    SCENARIO_NAME="test-scenario"
    
    run validate_parsed_args
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Missing required variables" ]]
    [[ "$output" =~ "DEPLOYMENT_MODE" ]]
}

@test "validate_parsed_args fails with missing VALIDATION_MODE" {
    DEPLOYMENT_MODE="local"
    VALIDATION_MODE=""
    SCENARIO_NAME="test-scenario"
    
    run validate_parsed_args
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Missing required variables" ]]
    [[ "$output" =~ "VALIDATION_MODE" ]]
}

@test "validate_parsed_args fails with missing SCENARIO_NAME" {
    DEPLOYMENT_MODE="local"
    VALIDATION_MODE="full"
    SCENARIO_NAME=""
    
    run validate_parsed_args
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Missing required variables" ]]
    [[ "$output" =~ "SCENARIO_NAME" ]]
}

@test "validate_parsed_args fails with multiple missing variables" {
    DEPLOYMENT_MODE=""
    VALIDATION_MODE=""
    SCENARIO_NAME="test-scenario"
    
    run validate_parsed_args
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Missing required variables" ]]
    [[ "$output" =~ "DEPLOYMENT_MODE" ]]
    [[ "$output" =~ "VALIDATION_MODE" ]]
}

################################################################################
# Display Configuration Tests
################################################################################

@test "display_parsed_config shows nothing when VERBOSE is false" {
    VERBOSE=false
    SCENARIO_NAME="test-scenario"
    DEPLOYMENT_MODE="local"
    VALIDATION_MODE="full"
    DRY_RUN=false
    
    run display_parsed_config
    [[ "$status" -eq 0 ]]
    [[ "$output" == "" ]]
}

@test "display_parsed_config shows configuration when VERBOSE is true" {
    VERBOSE=true
    SCENARIO_NAME="test-scenario"
    DEPLOYMENT_MODE="docker"
    VALIDATION_MODE="basic"
    DRY_RUN=true
    
    run display_parsed_config
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Parsed configuration:" ]]
    [[ "$output" =~ "Scenario: test-scenario" ]]
    [[ "$output" =~ "Deployment mode: docker" ]]
    [[ "$output" =~ "Validation mode: basic" ]]
    [[ "$output" =~ "Dry run: true" ]]
    [[ "$output" =~ "Verbose: true" ]]
}

################################################################################
# Integration Tests - Real Usage Patterns
################################################################################

@test "parse_args integration: typical usage" {
    parse_args "multi-modal-ai-assistant" --mode "docker" --verbose
    [[ "$SCENARIO_NAME" == "multi-modal-ai-assistant" ]]
    [[ "$DEPLOYMENT_MODE" == "docker" ]]
    [[ "$VALIDATION_MODE" == "full" ]]  # default
    [[ "$DRY_RUN" == false ]]          # default
    [[ "$VERBOSE" == true ]]
}

@test "parse_args integration: dry run testing" {
    parse_args "test-scenario" --dry-run --validate "basic"
    [[ "$SCENARIO_NAME" == "test-scenario" ]]
    [[ "$DEPLOYMENT_MODE" == "local" ]]  # default
    [[ "$VALIDATION_MODE" == "basic" ]]
    [[ "$DRY_RUN" == true ]]
    [[ "$VERBOSE" == false ]]           # default
}

@test "parse_args integration: k8s deployment" {
    parse_args "production-app" --mode "k8s" --validate "full" --verbose
    [[ "$SCENARIO_NAME" == "production-app" ]]
    [[ "$DEPLOYMENT_MODE" == "k8s" ]]
    [[ "$VALIDATION_MODE" == "full" ]]
    [[ "$DRY_RUN" == false ]]           # default
    [[ "$VERBOSE" == true ]]
}