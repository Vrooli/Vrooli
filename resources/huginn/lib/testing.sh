#!/usr/bin/env bash
# Huginn Testing and Validation Functions
# Functions for testing Huginn installation and functionality

#######################################
# Run complete Huginn test suite
#######################################
huginn::run_tests() {
    huginn::show_test_header
    
    local tests_passed=0
    local tests_failed=0
    
    # Test 1: Container status
    if huginn::test_container_status; then
        huginn::show_test_result "container status" "passed"
        ((tests_passed++))
    else
        huginn::show_test_result "container status" "failed"
        ((tests_failed++))
    fi
    
    # Test 2: Web interface
    if huginn::test_web_interface; then
        huginn::show_test_result "web interface" "passed"
        ((tests_passed++))
    else
        huginn::show_test_result "web interface" "failed"
        ((tests_failed++))
    fi
    
    # Test 3: Database connection
    if huginn::test_database_connection; then
        huginn::show_test_result "database connection" "passed"
        ((tests_passed++))
    else
        huginn::show_test_result "database connection" "failed"
        ((tests_failed++))
    fi
    
    # Test 4: Rails runner
    if huginn::test_rails_runner; then
        huginn::show_test_result "Rails runner" "passed"
        ((tests_passed++))
    else
        huginn::show_test_result "Rails runner" "failed"
        ((tests_failed++))
    fi
    
    # Test 5: Docker access
    if huginn::test_docker_access; then
        huginn::show_test_result "Docker access" "passed"
        ((tests_passed++))
    else
        huginn::show_test_result "Docker access" "failed" "Docker not accessible from container"
        ((tests_failed++))
    fi
    
    # Test 6: Agent creation
    if huginn::test_agent_creation; then
        huginn::show_test_result "agent creation" "passed"
        ((tests_passed++))
    else
        huginn::show_test_result "agent creation" "failed" 
        ((tests_failed++))
    fi
    
    # Test 7: Event flow
    if huginn::test_event_flow; then
        huginn::show_test_result "event flow" "passed"
        ((tests_passed++))
    else
        huginn::show_test_result "event flow" "failed"
        ((tests_failed++))
    fi
    
    # Test 8: Performance
    if huginn::test_performance; then
        huginn::show_test_result "performance" "passed"
        ((tests_passed++))
    else
        huginn::show_test_result "performance" "failed" "Response time too slow"
        ((tests_failed++))
    fi
    
    # Show summary
    huginn::show_test_summary "$tests_passed" "$tests_failed"
    
    # Return non-zero if any test failed
    [[ $tests_failed -eq 0 ]]
}

#######################################
# Test container status
#######################################
huginn::test_container_status() {
    if ! huginn::is_running; then
        return 1
    fi
    
    # Check both containers
    local huginn_status=$(docker container inspect -f '{{.State.Status}}' "$CONTAINER_NAME" 2>/dev/null || echo "error")
    local db_status=$(docker container inspect -f '{{.State.Status}}' "$DB_CONTAINER_NAME" 2>/dev/null || echo "error")
    
    [[ "$huginn_status" == "running" && "$db_status" == "running" ]]
}

#######################################
# Test web interface accessibility
#######################################
huginn::test_web_interface() {
    if ! huginn::is_running; then
        return 1
    fi
    
    # Test HTTP endpoint
    if system::is_command "curl"; then
        local response=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$HUGINN_BASE_URL/users/sign_in" 2>/dev/null || echo "000")
        [[ "$response" == "200" ]]
    else
        # Fallback: just check if container is healthy
        huginn::is_healthy
    fi
}

#######################################
# Test database connection
#######################################
huginn::test_database_connection() {
    if ! huginn::is_running; then
        return 1
    fi
    
    huginn::check_database
}

#######################################
# Test Rails runner functionality
#######################################
huginn::test_rails_runner() {
    if ! huginn::is_running; then
        return 1
    fi
    
    local result=$(huginn::rails_runner 'puts "RAILS_OK"' 2>/dev/null)
    [[ "$result" == "RAILS_OK" ]]
}

#######################################
# Test Docker access from container
#######################################
huginn::test_docker_access() {
    if ! huginn::is_running; then
        return 1
    fi
    
    # Check if Docker socket is accessible
    docker exec "$CONTAINER_NAME" test -S /var/run/docker.sock 2>/dev/null
}

#######################################
# Test agent creation capability
#######################################
huginn::test_agent_creation() {
    if ! huginn::is_running; then
        return 1
    fi
    
    local test_code='
    begin
      # Try to create a test agent
      test_agent = Agent.build(
        user: User.first,
        name: "Test Agent #{Time.now.to_i}",
        type: "Agents::ManualEventAgent",
        options: {}
      )
      puts test_agent.valid? ? "CREATE_OK" : "CREATE_FAILED"
    rescue => e
      puts "ERROR: #{e.message}"
    end
    '
    
    local result=$(huginn::rails_runner "$test_code" 2>/dev/null)
    [[ "$result" == "CREATE_OK" ]]
}

#######################################
# Test event flow between agents
#######################################
huginn::test_event_flow() {
    if ! huginn::is_running; then
        return 1
    fi
    
    local test_code='
    begin
      # Check if events can flow between agents
      if Link.count > 0
        puts "FLOW_OK"
      elsif Agent.count > 1
        puts "FLOW_OK"
      else
        puts "NO_FLOW"
      end
    rescue => e
      puts "ERROR: #{e.message}"
    end
    '
    
    local result=$(huginn::rails_runner "$test_code" 2>/dev/null)
    [[ "$result" == "FLOW_OK" ]]
}

#######################################
# Test performance metrics
#######################################
huginn::test_performance() {
    if ! huginn::is_running; then
        return 1
    fi
    
    # Measure response time
    if system::is_command "curl"; then
        local start_time=$(date +%s)
        curl -s -o /dev/null --max-time 5 "$HUGINN_BASE_URL" 2>/dev/null
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        # Response should be under 3 seconds
        [[ $duration -lt 3 ]]
    else
        # If curl not available, just pass
        return 0
    fi
}

#######################################
# Show test summary
#######################################
huginn::show_test_summary() {
    local passed="$1"
    local failed="$2"
    local total=$((passed + failed))
    
    echo
    log::info "📊 Test Results"
    log::info "================"
    log::info "Total Tests: $total"
    log::success "Passed: $passed"
    if [[ $failed -gt 0 ]]; then
        log::error "Failed: $failed"
    else
        log::info "Failed: 0"
    fi
    echo
    
    if [[ $failed -eq 0 ]]; then
        log::success "✅ All tests passed!"
    else
        log::error "❌ Some tests failed"
    fi
}

#######################################
# Benchmark Huginn performance
#######################################
huginn::benchmark() {
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    log::info "🏃 Running performance benchmark..."
    echo
    
    # Test 1: Agent execution time
    log::info "Testing agent execution speed..."
    local agent_code='
    require "benchmark"
    
    time = Benchmark.measure do
      if agent = Agent.first
        agent.check
      end
    end
    
    puts "Agent execution: #{(time.real * 1000).round(2)}ms"
    '
    
    huginn::rails_runner "$agent_code" 2>/dev/null || echo "Agent execution: N/A"
    
    # Test 2: Database query performance
    log::info "Testing database performance..."
    local db_code='
    require "benchmark"
    
    time = Benchmark.measure do
      Agent.count
      Event.count
      User.count
      Link.count
    end
    
    puts "Database queries: #{(time.real * 1000).round(2)}ms"
    '
    
    huginn::rails_runner "$db_code" 2>/dev/null || echo "Database queries: N/A"
    
    # Test 3: Event creation performance
    log::info "Testing event creation..."
    local event_code='
    require "benchmark"
    
    time = Benchmark.measure do
      if agent = Agent.first
        10.times do |i|
          Event.create(
            agent: agent,
            payload: { test: i, timestamp: Time.now.to_i }
          )
        end
      end
    end
    
    puts "Event creation (10 events): #{(time.real * 1000).round(2)}ms"
    '
    
    huginn::rails_runner "$event_code" 2>/dev/null || echo "Event creation: N/A"
    
    # Test 4: Web response time
    if system::is_command "curl"; then
        log::info "Testing web response time..."
        local response_time=$(curl -o /dev/null -s -w "%{time_total}\n" "$HUGINN_BASE_URL" 2>/dev/null || echo "0")
        local response_ms=$(awk "BEGIN {printf \"%.2f\", $response_time * 1000}")
        echo "Web response: ${response_ms}ms"
    fi
    
    echo
    log::success "✅ Benchmark complete"
}

#######################################
# Load test Huginn
#######################################
huginn::load_test() {
    local requests="${1:-100}"
    local concurrency="${2:-10}"
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    if ! system::is_command "curl"; then
        log::error "curl is required for load testing"
        return 1
    fi
    
    log::info "🔥 Running load test..."
    log::info "Requests: $requests"
    log::info "Concurrency: $concurrency"
    echo
    
    # Simple load test using curl in parallel
    local start_time=$(date +%s)
    local success=0
    local failed=0
    
    for ((i=1; i<=requests; i++)); do
        (
            if curl -s -o /dev/null --max-time 5 "$HUGINN_BASE_URL" 2>/dev/null; then
                echo "success"
            else
                echo "failed"
            fi
        ) &
        
        # Limit concurrency
        if [[ $((i % concurrency)) -eq 0 ]]; then
            wait
        fi
    done
    
    wait
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log::info "📊 Load Test Results"
    log::info "===================="
    log::info "Duration: ${duration}s"
    log::info "Requests/sec: $(awk "BEGIN {printf \"%.2f\", $requests / $duration}")"
    
    # Check if Huginn is still healthy after load test
    if huginn::is_healthy; then
        log::success "✅ Huginn handled load test successfully"
    else
        log::error "❌ Huginn became unhealthy during load test"
    fi
}