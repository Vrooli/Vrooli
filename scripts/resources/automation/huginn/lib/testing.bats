#!/usr/bin/env bats
# Tests for Huginn lib/testing.sh

load ../test_fixtures/test_helper

setup() {
    setup_test_environment
    mock_docker "success"
    mock_curl "success"
    source_huginn_scripts
    source "$HUGINN_TEST_DIR/lib/testing.sh"
}

teardown() {
    teardown_test_environment
}

@test "testing.sh: run_tests executes all tests" {
    # Mock all test functions to track execution
    local tests_executed=()
    
    huginn::test_container_status() {
        tests_executed+=("container")
        return 0
    }
    
    huginn::test_web_interface() {
        tests_executed+=("web")
        return 0
    }
    
    huginn::test_database_connection() {
        tests_executed+=("database")
        return 0
    }
    
    huginn::test_rails_runner() {
        tests_executed+=("rails")
        return 0
    }
    
    huginn::test_docker_access() {
        tests_executed+=("docker")
        return 0
    }
    
    huginn::test_agent_creation() {
        tests_executed+=("agent")
        return 0
    }
    
    huginn::test_event_flow() {
        tests_executed+=("event")
        return 0
    }
    
    huginn::test_performance() {
        tests_executed+=("performance")
        return 0
    }
    
    export -f huginn::test_container_status huginn::test_web_interface
    export -f huginn::test_database_connection huginn::test_rails_runner
    export -f huginn::test_docker_access huginn::test_agent_creation
    export -f huginn::test_event_flow huginn::test_performance
    
    run huginn::run_tests
    assert_success
    assert_output_contains "Test Results"
    assert_output_contains "All tests passed"
    
    # Verify all tests were executed
    [[ ${#tests_executed[@]} -eq 8 ]]
}

@test "testing.sh: run_tests reports failures correctly" {
    # Make some tests fail
    huginn::test_container_status() { return 0; }
    huginn::test_web_interface() { return 1; }  # This fails
    huginn::test_database_connection() { return 0; }
    huginn::test_rails_runner() { return 1; }  # This fails
    huginn::test_docker_access() { return 0; }
    huginn::test_agent_creation() { return 0; }
    huginn::test_event_flow() { return 0; }
    huginn::test_performance() { return 0; }
    
    export -f huginn::test_container_status huginn::test_web_interface
    export -f huginn::test_database_connection huginn::test_rails_runner
    export -f huginn::test_docker_access huginn::test_agent_creation
    export -f huginn::test_event_flow huginn::test_performance
    
    run huginn::run_tests
    assert_failure
    assert_output_contains "Failed: 2"
    assert_output_contains "Some tests failed"
}

@test "testing.sh: test_container_status checks both containers" {
    mock_docker "success"
    
    run huginn::test_container_status
    assert_success
}

@test "testing.sh: test_container_status fails when not running" {
    mock_docker "not_running"
    
    run huginn::test_container_status
    assert_failure
}

@test "testing.sh: test_web_interface checks HTTP endpoint" {
    mock_curl "success"
    
    # Mock curl to return 200 status
    curl() {
        case "$*" in
            *"-w"*"%{http_code}"*)
                echo "200"
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f curl
    
    run huginn::test_web_interface
    assert_success
}

@test "testing.sh: test_database_connection uses check_database" {
    # Mock successful database check
    huginn::check_database() {
        return 0
    }
    export -f huginn::check_database
    
    run huginn::test_database_connection
    assert_success
}

@test "testing.sh: test_rails_runner validates Rails functionality" {
    # Mock Rails runner
    huginn::rails_runner() {
        echo "RAILS_OK"
        return 0
    }
    export -f huginn::rails_runner
    
    run huginn::test_rails_runner
    assert_success
}

@test "testing.sh: test_docker_access checks socket access" {
    # Mock docker exec to succeed
    docker() {
        case "$*" in
            *"exec"*"test -S /var/run/docker.sock"*)
                return 0
                ;;
            *)
                command docker "$@" 2>/dev/null || return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::test_docker_access
    assert_success
}

@test "testing.sh: test_agent_creation validates agent creation" {
    # Mock Rails runner for agent creation
    huginn::rails_runner() {
        echo "CREATE_OK"
        return 0
    }
    export -f huginn::rails_runner
    
    run huginn::test_agent_creation
    assert_success
}

@test "testing.sh: test_event_flow checks event system" {
    # Mock Rails runner for event flow
    huginn::rails_runner() {
        echo "FLOW_OK"
        return 0
    }
    export -f huginn::rails_runner
    
    run huginn::test_event_flow
    assert_success
}

@test "testing.sh: test_performance measures response time" {
    # Mock curl with fast response
    curl() {
        sleep 0.1  # Simulate quick response
        return 0
    }
    export -f curl
    
    # Mock date to control timing
    local call_count=0
    date() {
        if [[ "$1" == "+%s" ]]; then
            if [[ $call_count -eq 0 ]]; then
                call_count=$((call_count + 1))
                echo "1000"
            else
                echo "1001"  # 1 second later
            fi
        fi
    }
    export -f date
    
    run huginn::test_performance
    assert_success
}

@test "testing.sh: show_test_summary displays results" {
    run huginn::show_test_summary 7 1
    assert_success
    assert_output_contains "Total Tests: 8"
    assert_output_contains "Passed: 7"
    assert_output_contains "Failed: 1"
    assert_output_contains "Some tests failed"
}

@test "testing.sh: benchmark runs performance tests" {
    # Mock Rails runner for benchmark
    huginn::rails_runner() {
        case "$*" in
            *"Agent execution"*)
                echo "Agent execution: 12.34ms"
                ;;
            *"Database queries"*)
                echo "Database queries: 5.67ms"
                ;;
            *"Event creation"*)
                echo "Event creation (10 events): 45.89ms"
                ;;
            *)
                echo "mock output"
                ;;
        esac
        return 0
    }
    export -f huginn::rails_runner
    
    # Mock curl for web response test
    curl() {
        case "$*" in
            *"%{time_total}"*)
                echo "0.123"
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f curl
    
    run huginn::benchmark
    assert_success
    assert_output_contains "Running performance benchmark"
    assert_output_contains "Agent execution"
    assert_output_contains "Database queries"
    assert_output_contains "Event creation"
    assert_output_contains "Web response"
}

@test "testing.sh: load_test performs basic load testing" {
    # Override sleep and wait to speed up test
    wait() { return 0; }
    export -f wait
    
    # Mock curl for load test
    curl() {
        echo "success"
        return 0
    }
    export -f curl
    
    # Mock date for timing
    local call_count=0
    date() {
        if [[ "$1" == "+%s" ]]; then
            if [[ $call_count -eq 0 ]]; then
                call_count=$((call_count + 1))
                echo "1000"
            else
                echo "1005"  # 5 seconds later
            fi
        fi
    }
    export -f date
    
    run huginn::load_test 10 2
    assert_success
    assert_output_contains "Running load test"
    assert_output_contains "Requests: 10"
    assert_output_contains "Concurrency: 2"
    assert_output_contains "Load Test Results"
}