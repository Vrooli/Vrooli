#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Define location for this test file
APP_LIFECYCLE_DEPLOY_TOOLS_DIR="$BATS_TEST_DIRNAME"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_TOOLS_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/logs.sh"

# Path to the script under test
SCRIPT_PATH="${APP_LIFECYCLE_DEPLOY_TOOLS_DIR}/cost-calculator.sh"

setup() {
    # Initialize mocks
    mock::logs::reset
}

teardown() {
    # Clean up mocks
    mock::logs::cleanup
}

@test "cost-calculator functions exist when sourced" {
    run bash -c "source '$SCRIPT_PATH' && declare -f cost_calculator::calculate"
    [ "$status" -eq 0 ]
    [[ "$output" =~ cost_calculator::calculate ]]
    
    run bash -c "source '$SCRIPT_PATH' && declare -f cost_calculator::show_configurations"
    [ "$status" -eq 0 ]
    [[ "$output" =~ cost_calculator::show_configurations ]]
}

@test "cost_calculator::calculate computes correct cost for s-1vcpu-2gb" {
    run bash -c "
        source '$SCRIPT_PATH'
        cost_calculator::calculate 's-1vcpu-2gb' 3 false false 2>&1
    "
    
    [ "$status" -eq 36 ]  # Exit code is the total cost
    [[ "$output" =~ "Node Cost: \$36" ]]
    [[ "$output" =~ "TOTAL: \$36/month" ]]
}

@test "cost_calculator::calculate includes database costs when enabled" {
    run bash -c "
        source '$SCRIPT_PATH'
        cost_calculator::calculate 's-2vcpu-4gb' 3 true false 2>&1
    "
    
    [ "$status" -eq 102 ]  # 72 (nodes) + 30 (databases)
    [[ "$output" =~ "Database Cost: \$30" ]]
    [[ "$output" =~ "TOTAL: \$102/month" ]]
}

@test "cost_calculator::calculate includes load balancer when enabled" {
    run bash -c "
        source '$SCRIPT_PATH'
        cost_calculator::calculate 's-2vcpu-4gb' 3 false true 2>&1
    "
    
    [ "$status" -eq 84 ]  # 72 (nodes) + 12 (LB)
    [[ "$output" =~ "Load Balancer Cost: \$12" ]]
    [[ "$output" =~ "TOTAL: \$84/month" ]]
}

@test "cost_calculator::calculate handles all services enabled" {
    run bash -c "
        source '$SCRIPT_PATH'
        cost_calculator::calculate 's-4vcpu-8gb' 3 true true 2>&1
    "
    
    [ "$status" -eq 186 ]  # 144 (nodes) + 30 (db) + 12 (LB)
    [[ "$output" =~ "Node Cost: \$144" ]]
    [[ "$output" =~ "Database Cost: \$30" ]]
    [[ "$output" =~ "Load Balancer Cost: \$12" ]]
    [[ "$output" =~ "TOTAL: \$186/month" ]]
}

@test "cost_calculator::calculate handles unknown node size" {
    run bash -c "
        source '$SCRIPT_PATH'
        cost_calculator::calculate 'unknown-size' 3 false false 2>&1
    "
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown node size" ]]
}

@test "cost_calculator::show_configurations displays all example configurations" {
    run bash -c "
        source '$SCRIPT_PATH'
        cost_calculator::show_configurations 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Minimal Setup" ]]
    [[ "$output" =~ "Recommended Setup" ]]
    [[ "$output" =~ "High Performance Setup" ]]
    [[ "$output" =~ "Budget Option" ]]
    [[ "$output" =~ "Annual Cost Comparison" ]]
}

@test "cost_calculator::main with 'show' argument displays configurations" {
    run bash -c "
        source '$SCRIPT_PATH'
        cost_calculator::main show 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "DigitalOcean Kubernetes Cost Calculator" ]]
}

@test "cost_calculator::main with 'help' shows usage" {
    run bash -c "
        source '$SCRIPT_PATH'
        cost_calculator::main help 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "show" ]]
    [[ "$output" =~ "interactive" ]]
}

@test "cost_calculator::main with unknown option shows error" {
    run bash -c "
        source '$SCRIPT_PATH'
        cost_calculator::main unknown 2>&1
    "
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown option" ]]
}

@test "cost-calculator can be run directly from command line" {
    run bash "$SCRIPT_PATH" help
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}