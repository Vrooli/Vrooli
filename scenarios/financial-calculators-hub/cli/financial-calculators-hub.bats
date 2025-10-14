#!/usr/bin/env bats

# Financial Calculators Hub CLI Tests

setup() {
    export CLI_PATH="./cli/financial-calculators-hub"
    export API_PORT="${API_PORT:-19500}"
}

@test "CLI help command shows usage" {
    run $CLI_PATH --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Financial Calculators Hub"* ]] || [[ "$output" == *"Usage:"* ]]
}

@test "CLI version command shows version" {
    run $CLI_PATH version
    [ "$status" -eq 0 ]
    [[ "$output" == *"version"* ]] || [[ "$output" == *"1."* ]]
}

@test "CLI status command shows service status" {
    run $CLI_PATH status
    [ "$status" -eq 0 ]
    [[ "$output" == *"status"* ]] || [[ "$output" == *"API"* ]] || [[ "$output" == *"healthy"* ]]
}

@test "CLI FIRE command requires parameters" {
    run $CLI_PATH fire
    [ "$status" -ne 0 ]
    [[ "$output" == *"age"* ]] || [[ "$output" == *"required"* ]] || [[ "$output" == *"Usage:"* ]]
}

@test "CLI FIRE command with valid input works" {
    run $CLI_PATH fire --age 30 --savings 100000 --income 100000 --expenses 50000
    [ "$status" -eq 0 ]
    [[ "$output" == *"Retirement"* ]] || [[ "$output" == *"Age"* ]] || [[ "$output" == *"Years"* ]]
}

@test "CLI compound interest command requires parameters" {
    run $CLI_PATH compound
    [ "$status" -ne 0 ]
    [[ "$output" == *"principal"* ]] || [[ "$output" == *"required"* ]] || [[ "$output" == *"Usage:"* ]]
}

@test "CLI compound interest command with valid input works" {
    run $CLI_PATH compound --principal 10000 --rate 7 --years 10
    [ "$status" -eq 0 ]
    [[ "$output" == *"Final"* ]] || [[ "$output" == *"Amount"* ]] || [[ "$output" == *"Interest"* ]]
}

@test "CLI mortgage command requires parameters" {
    run $CLI_PATH mortgage
    [ "$status" -ne 0 ]
    [[ "$output" == *"amount"* ]] || [[ "$output" == *"required"* ]] || [[ "$output" == *"Usage:"* ]]
}

@test "CLI mortgage command with valid input works" {
    run $CLI_PATH mortgage --amount 300000 --rate 4.5 --years 30
    [ "$status" -eq 0 ]
    [[ "$output" == *"Payment"* ]] || [[ "$output" == *"Monthly"* ]] || [[ "$output" == *"Interest"* ]]
}

@test "CLI inflation command requires parameters" {
    run $CLI_PATH inflation
    [ "$status" -ne 0 ]
    [[ "$output" == *"amount"* ]] || [[ "$output" == *"required"* ]] || [[ "$output" == *"Usage:"* ]]
}

@test "CLI inflation command with valid input works" {
    run $CLI_PATH inflation --amount 100000 --years 20 --rate 3
    [ "$status" -eq 0 ]
    [[ "$output" == *"future"* ]] || [[ "$output" == *"value"* ]] || [[ "$output" == *"purchasing"* ]]
}

@test "CLI with invalid command shows error" {
    run $CLI_PATH invalid-command
    [ "$status" -ne 0 ]
    [[ "$output" == *"Unknown command"* ]] || [[ "$output" == *"Usage:"* ]] || [[ "$output" == *"help"* ]]
}

@test "CLI JSON output format flag works" {
    run $CLI_PATH fire --age 30 --savings 100000 --income 100000 --expenses 50000 --json
    [ "$status" -eq 0 ]
    [[ "$output" == *"{"* ]] || [[ "$output" == *"}"* ]]
}
