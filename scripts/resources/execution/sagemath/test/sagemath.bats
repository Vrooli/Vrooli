#!/usr/bin/env bats

# SageMath integration tests

setup() {
    export SAGEMATH_TEST_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
    export SAGEMATH_CLI="$SAGEMATH_TEST_DIR/../cli.sh"
}

@test "SageMath CLI exists and is executable" {
    [ -f "$SAGEMATH_CLI" ]
    [ -x "$SAGEMATH_CLI" ]
}

@test "SageMath help command works" {
    run "$SAGEMATH_CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SageMath Resource CLI" ]]
}

@test "SageMath status command works" {
    run "$SAGEMATH_CLI" status
    [ "$status" -eq 0 ]
}

@test "SageMath status JSON format works" {
    run "$SAGEMATH_CLI" status --format json
    [ "$status" -eq 0 ]
    # Verify it's valid JSON
    echo "$output" | jq . >/dev/null
}

@test "SageMath install creates container" {
    skip "Installation test - requires Docker"
    run "$SAGEMATH_CLI" install
    [ "$status" -eq 0 ]
}

@test "SageMath calculate basic arithmetic" {
    skip "Requires running container"
    run "$SAGEMATH_CLI" calculate "2 + 2"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "4" ]]
}

@test "SageMath inject script file" {
    skip "Requires running container"
    echo "print('test')" > /tmp/test.sage
    run "$SAGEMATH_CLI" inject /tmp/test.sage
    [ "$status" -eq 0 ]
    rm -f /tmp/test.sage
}