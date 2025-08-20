#!/usr/bin/env bats
# SimPy integration tests

# Source resource files  
setup() {
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-support/load"
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-assert/load"
    
    export SIMPY_TEST_DIR="$(cd "$(dirname "${BATS_TEST_FILENAME}")" && pwd)"
    export SIMPY_LIB_DIR="${SIMPY_TEST_DIR}/../lib"
    
    # Source core module
    source "${SIMPY_LIB_DIR}/core.sh"
    source "${SIMPY_LIB_DIR}/status.sh"
}

@test "simpy: service is installed" {
    run simpy::is_installed
    assert_success
}

@test "simpy: service is running" {
    run simpy::is_running
    assert_success
}

@test "simpy: health endpoint responds" {
    run simpy::test_connection
    assert_success
}

@test "simpy: can get version" {
    run simpy::get_version
    assert_success
    assert_output --regexp "^[0-9]+\.[0-9]+\.[0-9]+$"
}

@test "simpy: health endpoint returns valid JSON" {
    run curl -s http://localhost:9510/health
    assert_success
    
    # Parse JSON to verify structure
    echo "$output" | jq -e '.status' >/dev/null
    echo "$output" | jq -e '.version' >/dev/null
    echo "$output" | jq -e '.timestamp' >/dev/null
}

@test "simpy: examples endpoint works" {
    run curl -s http://localhost:9510/examples
    assert_success
    
    # Should return JSON array
    echo "$output" | jq -e '.' >/dev/null
}

@test "simpy: can run basic simulation" {
    # Create a simple test simulation
    cat > /tmp/test_simpy.py << 'SIMTEST'
import simpy
import json

def car(env):
    while True:
        print(f'Start parking at {env.now}')
        parking_duration = 5
        yield env.timeout(parking_duration)
        print(f'Start driving at {env.now}')
        trip_duration = 2
        yield env.timeout(trip_duration)

env = simpy.Environment()
env.process(car(env))
env.run(until=15)
print(json.dumps({"success": True, "final_time": env.now}))
SIMTEST

    run python3 /tmp/test_simpy.py
    assert_success
    
    # Check output contains success
    echo "$output" | grep -q '"success": true'
    
    # Clean up
    rm -f /tmp/test_simpy.py
}

@test "simpy: examples directory exists and has files" {
    run simpy::list_examples
    assert_success
    
    # Should have at least one example
    [[ -n "$output" ]]
}

@test "simpy: status returns healthy" {
    run simpy::status --format json
    assert_success
    
    # Parse JSON status
    echo "$output" | jq -e '.health == "healthy"' >/dev/null
}
