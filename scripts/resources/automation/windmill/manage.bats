#!/usr/bin/env bats
# Windmill Management Tests

setup() {
    # Load test helpers
    load "${BATS_TEST_DIRNAME}/test-fixtures/test_helper.bash"
    common_test_setup
}

teardown() {
    common_test_teardown
}

@test "windmill manage script has correct permissions" {
    [ -x "${BATS_TEST_DIRNAME}/manage.sh" ]
}

@test "windmill status returns successfully when running" {
    # Check if Windmill is actually running first
    if curl -s http://localhost:5681/api/version >/dev/null 2>&1; then
        run "${BATS_TEST_DIRNAME}/manage.sh" -a status
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Status: Running and Healthy" ]]
    else
        skip "Windmill not running"
    fi
}

@test "windmill api endpoint responds" {
    if curl -s http://localhost:5681/api/version >/dev/null 2>&1; then
        run curl -s http://localhost:5681/api/version
        [ "$status" -eq 0 ]
        [[ "$output" =~ "CE v" ]]
    else
        skip "Windmill not running"
    fi
}

@test "windmill web interface is accessible" {
    if curl -s http://localhost:5681/api/version >/dev/null 2>&1; then
        run curl -s -I http://localhost:5681/
        [ "$status" -eq 0 ]
        [[ "$output" =~ "HTTP/1.1 200 OK" ]]
        [[ "$output" =~ "text/html" ]]
    else
        skip "Windmill not running"
    fi
}

@test "windmill port 5681 is listening" {
    if command -v nc >/dev/null 2>&1; then
        run nc -z localhost 5681
        [ "$status" -eq 0 ]
    else
        skip "netcat not available"
    fi
}

@test "windmill containers are running" {
    if docker ps --format "{{.Names}}" | grep -q "windmill-vrooli"; then
        # Check that at least server and db are running
        run bash -c "docker ps | grep windmill-vrooli-server"
        [ "$status" -eq 0 ]
        
        run bash -c "docker ps | grep windmill-vrooli-db"  
        [ "$status" -eq 0 ]
    else
        skip "Windmill containers not running"
    fi
}

@test "windmill help shows usage information" {
    run "${BATS_TEST_DIRNAME}/manage.sh" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "install" ]]
    [[ "$output" =~ "status" ]]
}