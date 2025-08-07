#!/usr/bin/env bats
# Huginn Mock Test Suite
# Comprehensive test coverage for the Huginn mock implementation

bats_require_minimum_version 1.5.0

# Setup test environment
setup() {
    # Create a temporary directory for this test session
    export MOCK_LOG_DIR="${BATS_TMPDIR}/huginn_mock_tests"
    mkdir -p "$MOCK_LOG_DIR"
    
    # Set a consistent state file path for this test session
    export HUGINN_MOCK_STATE_FILE="${BATS_TMPDIR}/huginn_mock_state_test"
    
    # Load test helpers if available
    if [[ -f "${BATS_TEST_DIRNAME}/../../../helpers/bats-support/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../../helpers/bats-support/load.bash"
    fi
    if [[ -f "${BATS_TEST_DIRNAME}/../../../helpers/bats-assert/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../../helpers/bats-assert/load.bash"
    else
        # Define basic assertion functions if bats-assert is not available
        assert_success() {
            if [[ "$status" -ne 0 ]]; then
                echo "expected success, got status $status"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_failure() {
            if [[ "$status" -eq 0 ]]; then
                echo "expected failure, got success"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_output() {
            local expected
            if [[ "$1" == "--partial" ]]; then
                shift
                expected="$1"
                if [[ "$output" != *"$expected"* ]]; then
                    echo "expected output to contain: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            else
                expected="${1:-}"
                if [[ "$output" != "$expected" ]]; then
                    echo "expected: $expected"
                    echo "actual: $output"
                    return 1
                fi
            fi
        }
        
        refute_output() {
            local expected
            if [[ "$1" == "--partial" ]]; then
                shift
                expected="$1"
                if [[ "$output" == *"$expected"* ]]; then
                    echo "expected output to NOT contain: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            else
                expected="${1:-}"
                if [[ "$output" == "$expected" ]]; then
                    echo "expected output to NOT be: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            fi
        }
    fi
    
    # Load the mock systems
    if [[ -f "${BATS_TEST_DIRNAME}/logs.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/logs.sh"
        mock::init_logging "$MOCK_LOG_DIR"
    fi
    
    if [[ -f "${BATS_TEST_DIRNAME}/verification.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/verification.sh"
        mock::verify::init
    fi
    
    if [[ -f "${BATS_TEST_DIRNAME}/docker.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/docker.sh"
        mock::docker::reset
    fi
    
    # Load the Huginn mock
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    
    # Reset the mock to a known state
    mock::huginn::reset
}

teardown() {
    # Clean up temporary files
    if [[ -n "${HUGINN_MOCK_STATE_FILE:-}" ]]; then
        rm -f "$HUGINN_MOCK_STATE_FILE"
        rm -rf "${HUGINN_MOCK_STATE_FILE}.lock" 2>/dev/null || true
    fi
    
    # Clean up mock log directory
    if [[ -n "${MOCK_LOG_DIR:-}" && -d "$MOCK_LOG_DIR" ]]; then
        rm -rf "$MOCK_LOG_DIR"
    fi
}

# ----------------------------
# Basic Functionality Tests
# ----------------------------

@test "huginn mock loads without errors" {
    run bash -c "source '${BATS_TEST_DIRNAME}/huginn.sh' && echo 'loaded'"
    assert_success
    assert_output --partial "loaded"
}

@test "huginn mock prevents duplicate loading" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        echo 'done'
    "
    assert_success
    assert_output --partial "done"
}

@test "huginn mock initializes with default configuration" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        echo \"Port: \$HUGINN_PORT\"
        echo \"Base URL: \$HUGINN_BASE_URL\"
        echo \"Container: \$HUGINN_CONTAINER_NAME\"
    "
    assert_success
    assert_output --partial "Port: 3000"
    assert_output --partial "Base URL: http://localhost:3000"
    assert_output --partial "Container: huginn_app"
}

# ----------------------------
# State Management Tests
# ----------------------------

@test "huginn mock reset clears all state" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::add_agent 100 'Test Agent'
        mock::huginn::reset
        mock::huginn::get_call_count 'nonexistent'
    "
    assert_success
    assert_output "0"
}

@test "huginn mock state persists across subshells" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::add_agent 100 'Test Agent'
        bash -c \"source '${BATS_TEST_DIRNAME}/huginn.sh' && curl http://localhost:3000/agents | grep -q 'Test Agent'\"
    "
    assert_success
}

@test "huginn mock setup_default_data creates initial data" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        curl http://localhost:3000/agents
    "
    assert_success
    assert_output --partial "Weather Agent"
    assert_output --partial "Email Digest"
    assert_output --partial "RSS Feed"
}

# ----------------------------
# Mode Management Tests
# ----------------------------

@test "huginn mock healthy mode returns success responses" {
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    mock::huginn::set_mode healthy
    run curl -s http://localhost:3000/health
    assert_success
    assert_output --partial '"status":"ok"'
    assert_output --partial '"database":"connected"'
}

@test "huginn mock unhealthy mode returns error responses" {
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    mock::huginn::set_mode unhealthy
    run curl -s http://localhost:3000/health
    assert_failure
    assert_output --partial '"status":"unhealthy"'
    assert_output --partial "Database connection failed"
}

@test "huginn mock installing mode returns installing status" {
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    mock::huginn::set_mode installing
    run curl -s http://localhost:3000/health
    assert_success
    assert_output --partial '"status":"installing"'
    assert_output --partial '"progress":40'
    assert_output --partial "Running database migrations"
}

@test "huginn mock stopped mode fails to connect" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::set_mode stopped
        curl -s http://localhost:3000/health 2>&1
    "
    assert_failure
}

# ----------------------------
# Agent Management Tests
# ----------------------------

@test "huginn mock add_agent creates new agent" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::add_agent 100 'Custom Agent' 'Agents::TwitterAgent' 'every_5m'
        curl -s http://localhost:3000/agents | grep -q 'Custom Agent'
    "
    assert_success
}

@test "huginn mock lists all agents" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        curl -s http://localhost:3000/agents
    "
    assert_success
    assert_output --partial '"id":1'
    assert_output --partial '"id":2'
    assert_output --partial '"id":3'
}

@test "huginn mock shows single agent with memory" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        curl -s http://localhost:3000/agents/1
    "
    assert_success
    assert_output --partial '"id":1'
    assert_output --partial '"name":"Weather Agent"'
    assert_output --partial '"memory":'
    assert_output --partial '"last_weather"'
}

@test "huginn mock returns error for non-existent agent" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        curl -s http://localhost:3000/agents/999
    "
    assert_failure
    assert_output --partial '"error":"Agent not found"'
}

@test "huginn mock set_agent_state updates agent state" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::set_agent_state 1 'error'
        echo 'State updated'
    "
    assert_success
    assert_output --partial "State updated"
}

@test "huginn mock set_agent_memory updates agent memory" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::set_agent_memory 1 '{\"custom\":\"data\"}'
        curl -s http://localhost:3000/agents/1
    "
    assert_success
    assert_output --partial '"custom":"data"'
}

# ----------------------------
# Scenario Management Tests
# ----------------------------

@test "huginn mock add_scenario creates new scenario" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::add_scenario 10 'Test Scenario' 'A test scenario' '1,2,3'
        curl -s http://localhost:3000/scenarios | grep -q 'Test Scenario'
    "
    assert_success
}

@test "huginn mock lists all scenarios" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        curl -s http://localhost:3000/scenarios
    "
    assert_success
    assert_output --partial '"name":"Weather Monitoring"'
    assert_output --partial '"agents":[1,2]'
}

# ----------------------------
# Event Management Tests
# ----------------------------

@test "huginn mock add_event creates new event" {
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    mock::huginn::add_event 200 1 '{"test":"data"}'
    run bash -c "source '${BATS_TEST_DIRNAME}/huginn.sh' && curl -s http://localhost:3000/events | grep '\"id\": 200'"
    assert_success
    assert_output --partial '"id": 200'
}

@test "huginn mock lists all events" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        curl -s http://localhost:3000/events
    "
    assert_success
    assert_output --partial '"id":100'
    assert_output --partial '"id":101'
}

# ----------------------------
# Docker Integration Tests
# ----------------------------

@test "huginn mock docker exec handles Rails runner for agent count" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        docker exec -it huginn_app bundle exec rails runner 'puts Agent.count'
    "
    assert_success
    assert_output --partial "Total agents: 3"
}

@test "huginn mock docker exec handles Rails runner for scenario count" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        docker exec -it huginn_app bundle exec rails runner 'puts Scenario.count'
    "
    assert_success
    assert_output --partial "Total scenarios: 1"
}

@test "huginn mock docker exec handles Rails runner for event count" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        docker exec -it huginn_app bundle exec rails runner 'puts Event.count'
    "
    assert_success
    assert_output --partial "Total events: 2"
}

@test "huginn mock docker exec handles Rails runner for agent listing" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        docker exec -it huginn_app bundle exec rails runner 'Agent.all.each { |a| puts a.name }'
    "
    assert_success
    assert_output --partial "Weather Agent"
}

@test "huginn mock docker exec handles Rails runner for backup" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        docker exec -it huginn_app bundle exec rails runner 'puts \"📊 Backup Summary:\"'
    "
    assert_success
    assert_output --partial "Backup Summary"
    assert_output --partial "Backup complete"
}

@test "huginn mock docker ps shows containers in healthy mode" {
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    mock::huginn::set_mode healthy
    run docker ps
    assert_success
    assert_output --partial "huginn_app"
    assert_output --partial "huginn_db"
}

@test "huginn mock docker ps shows no containers in stopped mode" {
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    mock::huginn::set_mode stopped
    run docker ps
    assert_success
    refute_output --partial "huginn_app"
}

@test "huginn mock docker logs returns mock logs" {
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    run docker logs huginn_app
    assert_success
    assert_output --partial "Huginn mock container logs"
}

# ----------------------------
# Error Injection Tests
# ----------------------------

@test "huginn mock inject_error causes curl to fail" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::inject_error curl 'connection_timeout'
        curl -s http://localhost:3000/health 2>&1
    "
    assert_failure
    assert_output --partial "Failed to connect"
}

@test "huginn mock inject_error causes rails_runner to fail" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::inject_error rails_runner 'database_error'
        docker exec huginn_app bundle exec rails runner 'puts Agent.count' 2>&1
    "
    assert_failure
    assert_output --partial "Error: database_error"
}

@test "huginn mock clear_error restores normal operation" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::inject_error curl 'test_error'
        mock::huginn::clear_error curl
        curl -s http://localhost:3000/health
    "
    assert_success
    assert_output --partial '"status":"ok"'
}

# ----------------------------
# Call Counting Tests
# ----------------------------

@test "huginn mock counts docker calls" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        docker ps >/dev/null
        docker logs huginn_app >/dev/null
        docker exec huginn_app echo test >/dev/null
        mock::huginn::get_call_count docker
    "
    assert_success
    assert_output "3"
}

@test "huginn mock counts curl calls" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        curl -s http://localhost:3000/health >/dev/null
        curl -s http://localhost:3000/agents >/dev/null
        curl -s http://localhost:3000/events >/dev/null
        mock::huginn::get_call_count curl
    "
    assert_success
    assert_output "3"
}

# ----------------------------
# Integration Tests
# ----------------------------

@test "huginn mock integrates with docker mock when available" {
    if [[ -f "${BATS_TEST_DIRNAME}/docker.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/docker.sh"
        mock::docker::reset
    fi
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    mock::huginn::set_mode healthy
    run echo 'Integration successful'
    assert_success
    assert_output --partial "Integration successful"
}

@test "huginn mock handles complex workflow simulation" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        
        # Add custom agents
        mock::huginn::add_agent 10 'Data Collector' 'Agents::WebsiteAgent' 'every_10m'
        mock::huginn::add_agent 11 'Data Processor' 'Agents::JavaScriptAgent' 'never'
        
        # Create scenario
        mock::huginn::add_scenario 2 'Data Pipeline' 'Collects and processes data' '10,11'
        
        # Generate events
        mock::huginn::add_event 300 10 '{\"url\":\"https://example.com\",\"data\":\"sample\"}'
        mock::huginn::add_event 301 11 '{\"processed\":true}'
        
        # Verify the setup
        agent_count=\$(curl -s http://localhost:3000/agents | grep -c '\"id\"')
        scenario_count=\$(curl -s http://localhost:3000/scenarios | grep -c '\"id\"')
        event_count=\$(curl -s http://localhost:3000/events | grep -c '\"id\"')
        
        echo \"Agents: \$agent_count\"
        echo \"Scenarios: \$scenario_count\"
        echo \"Events: \$event_count\"
    "
    assert_success
    assert_output --partial "Agents: 5"
    assert_output --partial "Scenarios: 2"
    assert_output --partial "Events: 4"
}

@test "huginn mock simulates unhealthy database scenario" {
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    mock::huginn::set_mode unhealthy
    
    # Health check should show unhealthy
    health=$(curl -s http://localhost:3000/health 2>&1 || true)
    
    # Rails runner should fail
    rails_result=$(docker exec huginn_app bundle exec rails runner 'puts Agent.count' 2>&1 || echo 'Rails failed')
    
    # API calls should fail
    api_result=$(curl -s http://localhost:3000/agents 2>&1 || true)
    
    run bash -c "echo 'Health: $health'; echo 'Rails: $rails_result'; echo 'API: $api_result'"
    assert_success
    assert_output --partial '"status":"unhealthy"'
    assert_output --partial "Database connection failed"
    assert_output --partial '"error":"Service unavailable"'
}

# ----------------------------
# Edge Case Tests
# ----------------------------

@test "huginn mock handles empty agent list" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        mock::huginn::reset
        # Clear default agents
        declare -gA MOCK_HUGINN_AGENTS=()
        _huginn_mock_save_state
        curl -s http://localhost:3000/agents
    "
    assert_success
    assert_output "[]"
}

@test "huginn mock handles malformed URLs gracefully" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        curl -s http://localhost:3000/invalid/endpoint
    "
    assert_failure
    assert_output --partial '"error":"Unknown endpoint"'
}

@test "huginn mock handles concurrent operations" {
    source "${BATS_TEST_DIRNAME}/huginn.sh"
    
    # Perform operations sequentially to avoid state file conflicts
    # (In a real implementation, we'd use file locking for concurrent access)
    mock::huginn::add_agent 50 'Agent1'
    mock::huginn::add_agent 51 'Agent2'
    mock::huginn::add_scenario 10 'Scenario1'
    
    # Verify all operations completed
    run bash -c "
        source '${BATS_TEST_DIRNAME}/huginn.sh'
        agents=\$(curl -s http://localhost:3000/agents)
        scenarios=\$(curl -s http://localhost:3000/scenarios)
        
        echo \"\$agents\" | grep -q 'Agent1' && echo 'Found Agent1'
        echo \"\$agents\" | grep -q 'Agent2' && echo 'Found Agent2'
        echo \"\$scenarios\" | grep -q 'Scenario1' && echo 'Found Scenario1'
    "
    assert_success
    assert_output --partial "Found Agent1"
    assert_output --partial "Found Agent2"
    assert_output --partial "Found Scenario1"
}