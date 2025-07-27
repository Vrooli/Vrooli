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

@test "windmill list-apps shows available examples" {
    run "${BATS_TEST_DIRNAME}/manage.sh" --action list-apps
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Available Windmill App Examples" ]]
}

@test "windmill examples directory structure exists" {
    [ -d "${BATS_TEST_DIRNAME}/examples/apps" ]
    [ -d "${BATS_TEST_DIRNAME}/examples/scripts" ]
    [ -d "${BATS_TEST_DIRNAME}/examples/flows" ]
    [ -f "${BATS_TEST_DIRNAME}/examples/apps/admin-dashboard.json" ]
    [ -f "${BATS_TEST_DIRNAME}/examples/apps/data-entry-form.json" ]
    [ -f "${BATS_TEST_DIRNAME}/examples/apps/monitoring-dashboard.json" ]
}

@test "windmill app examples have valid JSON structure" {
    if command -v jq >/dev/null 2>&1; then
        run jq '.' "${BATS_TEST_DIRNAME}/examples/apps/admin-dashboard.json"
        [ "$status" -eq 0 ]
        
        run jq '.name' "${BATS_TEST_DIRNAME}/examples/apps/admin-dashboard.json"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "User Management Dashboard" ]]
    else
        skip "jq not available"
    fi
}

@test "windmill prepare-app creates output files" {
    local temp_dir="/tmp/windmill-test-$$"
    mkdir -p "$temp_dir"
    
    run "${BATS_TEST_DIRNAME}/manage.sh" --action prepare-app --app-name admin-dashboard --output-dir "$temp_dir"
    [ "$status" -eq 0 ]
    
    [ -f "$temp_dir/admin-dashboard.json" ]
    [ -f "$temp_dir/admin-dashboard-instructions.md" ]
    
    rm -rf "$temp_dir"
}

@test "windmill manage script supports app management actions" {
    run "${BATS_TEST_DIRNAME}/manage.sh" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "list-apps" ]]
    [[ "$output" =~ "prepare-app" ]]
    [[ "$output" =~ "deploy-app" ]]
    [[ "$output" =~ "check-app-api" ]]
}