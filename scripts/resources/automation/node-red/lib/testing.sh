#!/usr/bin/env bash
# Node-RED Testing and Validation Functions
# Functions for testing Node-RED installation and functionality

# Source var.sh first to get standard directory variables  
LIB_TESTING_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${LIB_TESTING_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

#######################################
# Run complete Node-RED test suite
#######################################
node_red::run_tests() {
    node_red::show_test_header
    
    local tests_passed=0
    local tests_failed=0
    
    # Test 1: Container status
    if node_red::test_container_status; then
        node_red::show_test_result "container status" "passed"
        ((tests_passed++))
    else
        node_red::show_test_result "container status" "failed"
        ((tests_failed++))
    fi
    
    # Test 2: HTTP endpoint
    if node_red::test_http_endpoint; then
        node_red::show_test_result "HTTP endpoint" "passed"
        ((tests_passed++))
    else
        node_red::show_test_result "HTTP endpoint" "failed"
        ((tests_failed++))
    fi
    
    # Test 3: Admin API
    if node_red::test_admin_api; then
        node_red::show_test_result "admin API" "passed"
        ((tests_passed++))
    else
        node_red::show_test_result "admin API" "failed"
        ((tests_failed++))
    fi
    
    # Test 4: Docker access
    if node_red::test_docker_access; then
        node_red::show_test_result "Docker access" "passed"
        ((tests_passed++))
    else
        node_red::show_test_result "Docker access" "failed" "Docker access not available"
        ((tests_failed++))
    fi
    
    # Test 5: Workspace access
    if node_red::test_workspace_access; then
        node_red::show_test_result "workspace access" "passed"
        ((tests_passed++))
    else
        node_red::show_test_result "workspace access" "failed"
        ((tests_failed++))
    fi
    
    # Test 6: Host command execution
    if node_red::test_host_commands; then
        node_red::show_test_result "host command execution" "passed"
        ((tests_passed++))
    else
        node_red::show_test_result "host command execution" "failed"
        ((tests_failed++))
    fi
    
    # Test 7: Flow persistence
    if node_red::test_flow_persistence; then
        node_red::show_test_result "flow persistence" "passed"
        ((tests_passed++))
    else
        node_red::show_test_result "flow persistence" "failed" "No flows file found"
        ((tests_failed++))
    fi
    
    # Show summary
    node_red::show_test_summary "$tests_passed" "$tests_failed"
}

#######################################
# Individual test functions
#######################################

# Test if container is running
node_red::test_container_status() {
    node_red::is_running
}

# Test if HTTP endpoint is accessible
node_red::test_http_endpoint() {
    curl -s -f --max-time 10 "http://localhost:$RESOURCE_PORT" >/dev/null 2>&1
}

# Test if admin API is accessible
node_red::test_admin_api() {
    curl -s --max-time 10 "http://localhost:$RESOURCE_PORT/flows" >/dev/null 2>&1
}

# Test Docker socket access
node_red::test_docker_access() {
    if ! node_red::is_running; then
        return 1
    fi
    
    docker exec "$CONTAINER_NAME" docker ps >/dev/null 2>&1
}

# Test workspace access
node_red::test_workspace_access() {
    if ! node_red::is_running; then
        return 1
    fi
    
    docker exec "$CONTAINER_NAME" ls /workspace >/dev/null 2>&1
}

# Test host command execution
node_red::test_host_commands() {
    if ! node_red::is_running; then
        return 1
    fi
    
    docker exec "$CONTAINER_NAME" /bin/sh -c "which ls" >/dev/null 2>&1
}

# Test flow persistence
node_red::test_flow_persistence() {
    if ! node_red::is_running; then
        return 1
    fi
    
    docker exec "$CONTAINER_NAME" ls /data/flows.json >/dev/null 2>&1
}

#######################################
# Validate host command access
#######################################
node_red::validate_host_access() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    node_red::show_host_validation_header
    
    # Test basic commands
    local commands=("ls" "pwd" "date" "echo" "cat")
    local passed=0
    local failed=0
    
    for cmd in "${commands[@]}"; do
        echo -n "Testing '$cmd'... "
        if docker exec "$CONTAINER_NAME" "$cmd" --version >/dev/null 2>&1 || 
           docker exec "$CONTAINER_NAME" "$cmd" >/dev/null 2>&1; then
            echo "✓"
            ((passed++))
        else
            echo "✗"
            ((failed++))
        fi
    done
    
    # Test workspace write access
    echo -n "Testing workspace write access... "
    if docker exec "$CONTAINER_NAME" touch /workspace/node-red-test-file 2>/dev/null &&
       docker exec "$CONTAINER_NAME" rm /workspace/node-red-test-file 2>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗"
        ((failed++))
    fi
    
    node_red::show_host_access_summary "$passed" "$failed"
    
    [[ $failed -eq 0 ]]
}

#######################################
# Validate Docker socket access
#######################################
node_red::validate_docker_access() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    node_red::show_docker_validation_header
    
    # Test Docker commands
    echo -n "Testing 'docker ps'... "
    if docker exec "$CONTAINER_NAME" docker ps >/dev/null 2>&1; then
        echo "✓"
        
        # Test specific container operations
        echo -n "Testing container inspection... "
        if docker exec "$CONTAINER_NAME" docker inspect "$CONTAINER_NAME" >/dev/null 2>&1; then
            echo "✓"
        else
            echo "✗"
        fi
        
        return 0
    else
        echo "✗"
        node_red::show_docker_access_warning
        return 1
    fi
}

#######################################
# Benchmark Node-RED performance
#######################################
node_red::benchmark() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Running Node-RED performance benchmark..."
    
    echo "Node-RED Performance Benchmark"
    echo "=============================="
    echo
    
    # Test HTTP response time
    echo "HTTP Response Time:"
    local total_time=0
    local requests=5
    
    for i in $(seq 1 $requests); do
        local response_time
        response_time=$(curl -s -w "%{time_total}" -o /dev/null "http://localhost:$RESOURCE_PORT" 2>/dev/null || echo "0")
        echo "- Request $i: ${response_time}s"
        total_time=$(awk "BEGIN {print $total_time + $response_time}")
    done
    
    local avg_time
    avg_time=$(awk "BEGIN {print $total_time / $requests}")
    echo "- Average: ${avg_time}s"
    echo
    
    # Test admin API response time
    echo "Admin API Response Time:"
    local api_time
    api_time=$(curl -s -w "%{time_total}" -o /dev/null "http://localhost:$RESOURCE_PORT/flows" 2>/dev/null || echo "0")
    echo "- API Response: ${api_time}s"
    echo
    
    # Memory usage
    echo "Resource Usage:"
    local memory_usage
    memory_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "$CONTAINER_NAME" 2>/dev/null || echo "unknown")
    echo "- Memory: $memory_usage"
    
    local cpu_usage
    cpu_usage=$(docker stats --no-stream --format "{{.CPUPerc}}" "$CONTAINER_NAME" 2>/dev/null || echo "unknown")
    echo "- CPU: $cpu_usage"
    echo
    
    # Flow count and complexity
    echo "Flow Information:"
    local flow_count
    flow_count=$(docker exec "$CONTAINER_NAME" find /data -name "*.json" -type f 2>/dev/null | wc -l || echo "0")
    echo "- Flow files: $flow_count"
    
    # Data directory size
    local data_size
    data_size=$(docker exec "$CONTAINER_NAME" du -sh /data 2>/dev/null | cut -f1 || echo "unknown")
    echo "- Data size: $data_size"
    
    echo
    log::success "Benchmark completed"
}

#######################################
# Test flow execution capabilities
#######################################
node_red::test_flow_execution() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Testing flow execution capabilities..."
    
    local tests_passed=0
    local tests_failed=0
    
    # Test basic HTTP endpoint (if test flows exist)
    echo -n "Testing basic HTTP endpoint... "
    if curl -s -f --max-time 10 "http://localhost:$RESOURCE_PORT/test/hello" >/dev/null 2>&1; then
        echo "✓ PASSED"
        ((tests_passed++))
    else
        echo "✗ FAILED (endpoint not available)"
        ((tests_failed++))
    fi
    
    # Test command execution endpoint
    echo -n "Testing command execution... "
    local test_response
    test_response=$(curl -s --max-time 10 \
        -X POST -H "Content-Type: application/json" \
        -d '{"command": "echo test"}' \
        "http://localhost:$RESOURCE_PORT/test/exec" 2>/dev/null)
    
    if [[ -n "$test_response" ]] && echo "$test_response" | grep -q "test"; then
        echo "✓ PASSED"
        ((tests_passed++))
    else
        echo "✗ FAILED (command execution not working)"
        ((tests_failed++))
    fi
    
    # Test Docker endpoint (if available)
    echo -n "Testing Docker integration... "
    if curl -s -f --max-time 10 "http://localhost:$RESOURCE_PORT/test/docker" >/dev/null 2>&1; then
        echo "✓ PASSED"
        ((tests_passed++))
    else
        echo "✗ FAILED (Docker integration not available)"
        ((tests_failed++))
    fi
    
    echo
    echo "Flow Execution Test Summary:"
    echo "- Passed: $tests_passed"
    echo "- Failed: $tests_failed"
    
    [[ $tests_failed -eq 0 ]]
}

#######################################
# Load test Node-RED
#######################################
node_red::load_test() {
    local concurrent_requests="${1:-10}"
    local total_requests="${2:-100}"
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    if ! command -v ab >/dev/null 2>&1; then
        log::error "Apache Bench (ab) is required for load testing"
        log::info "Install with: sudo apt-get install apache2-utils"
        return 1
    fi
    
    log::info "Running load test (concurrent: $concurrent_requests, total: $total_requests)..."
    
    echo "Node-RED Load Test"
    echo "=================="
    echo "Concurrent requests: $concurrent_requests"
    echo "Total requests: $total_requests"
    echo "Target: http://localhost:$RESOURCE_PORT/"
    echo
    
    # Run the load test
    ab -c "$concurrent_requests" -n "$total_requests" "http://localhost:$RESOURCE_PORT/" 2>/dev/null | \
    grep -E "(Requests per second|Time per request|Transfer rate|Connection Times)" || {
        log::error "Load test failed"
        return 1
    }
    
    echo
    log::success "Load test completed"
}

#######################################
# Test Node-RED with sample flows
#######################################
node_red::test_with_sample_flows() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Testing Node-RED with sample flows..."
    
    # Check if sample flows exist
    local sample_flows_dir="$SCRIPT_DIR/flows"
    if [[ ! -d "$sample_flows_dir" ]] || [[ -z "$(ls -A "$sample_flows_dir" 2>/dev/null)" ]]; then
        log::warning "No sample flows found in $sample_flows_dir"
        return 1
    fi
    
    local tests_passed=0
    local tests_failed=0
    
    # Import and test each sample flow
    for flow_file in "$sample_flows_dir"/*.json; do
        if [[ -f "$flow_file" ]]; then
            local flow_name
            flow_name=$(basename "$flow_file" .json)
            
            echo -n "Testing flow: $flow_name... "
            
            # Import the flow
            if node_red::import_flow_file "$flow_file" >/dev/null 2>&1; then
                # Wait a moment for deployment
                sleep 2
                
                # Try to execute if it's a test flow
                local test_passed=true
                case "$flow_name" in
                    *test*|*basic*)
                        if ! curl -s -f --max-time 10 "http://localhost:$RESOURCE_PORT/test/hello" >/dev/null 2>&1; then
                            test_passed=false
                        fi
                        ;;
                    *exec*)
                        local exec_response
                        exec_response=$(curl -s --max-time 10 \
                            -X POST -H "Content-Type: application/json" \
                            -d '{"command": "pwd"}' \
                            "http://localhost:$RESOURCE_PORT/test/exec" 2>/dev/null)
                        if [[ -z "$exec_response" ]]; then
                            test_passed=false
                        fi
                        ;;
                esac
                
                if [[ "$test_passed" == "true" ]]; then
                    echo "✓ PASSED"
                    ((tests_passed++))
                else
                    echo "✗ FAILED (execution test failed)"
                    ((tests_failed++))
                fi
            else
                echo "✗ FAILED (import failed)"
                ((tests_failed++))
            fi
        fi
    done
    
    echo
    echo "Sample Flow Test Summary:"
    echo "- Passed: $tests_passed"
    echo "- Failed: $tests_failed"
    
    [[ $tests_failed -eq 0 ]]
}

#######################################
# Stress test Node-RED memory usage
#######################################
node_red::stress_test() {
    local duration="${1:-60}"  # seconds
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Running stress test for ${duration} seconds..."
    
    echo "Node-RED Stress Test"
    echo "==================="
    echo "Duration: ${duration}s"
    echo "Starting memory monitoring..."
    echo
    
    # Start memory monitoring in background
    local temp_file=$(mktemp)
    trap "rm -f $temp_file" EXIT
    
    # Monitor memory usage
    (
        local start_time=$(date +%s)
        while [[ $(($(date +%s) - start_time)) -lt $duration ]]; do
            local timestamp=$(date +%H:%M:%S)
            local memory_usage
            memory_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "$CONTAINER_NAME" 2>/dev/null || echo "unknown")
            echo "$timestamp: $memory_usage" >> "$temp_file"
            sleep 5
        done
    ) &
    local monitor_pid=$!
    
    # Generate load
    local request_count=0
    local start_time=$(date +%s)
    
    while [[ $(($(date +%s) - start_time)) -lt $duration ]]; do
        # Make requests to various endpoints
        curl -s "http://localhost:$RESOURCE_PORT" >/dev/null 2>&1 &
        curl -s "http://localhost:$RESOURCE_PORT/flows" >/dev/null 2>&1 &
        
        # Try test endpoints if available
        curl -s "http://localhost:$RESOURCE_PORT/test/hello" >/dev/null 2>&1 &
        
        ((request_count++))
        
        # Throttle requests
        sleep 0.1
    done
    
    # Wait for monitoring to complete
    wait $monitor_pid 2>/dev/null
    
    echo "Stress test completed"
    echo "Requests made: $request_count"
    echo
    echo "Memory usage over time:"
    cat "$temp_file"
    
    # Check final status
    echo
    if node_red::is_healthy; then
        log::success "Node-RED survived stress test"
        return 0
    else
        log::error "Node-RED failed stress test"
        return 1
    fi
}

#######################################
# Verify Node-RED installation integrity
#######################################
node_red::verify_installation() {
    log::info "Verifying Node-RED installation integrity..."
    
    local issues=0
    
    echo "Installation Verification"
    echo "========================"
    echo
    
    # Check container
    echo -n "Container exists: "
    if node_red::is_installed; then
        echo "✓"
    else
        echo "✗"
        ((issues++))
    fi
    
    # Check running
    echo -n "Container running: "
    if node_red::is_running; then
        echo "✓"
    else
        echo "✗"
        ((issues++))
    fi
    
    # Check image
    echo -n "Image available: "
    if docker image inspect "$IMAGE_NAME" >/dev/null 2>&1 || docker image inspect "$OFFICIAL_IMAGE" >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        ((issues++))
    fi
    
    # Check network
    echo -n "Network exists: "
    if docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗" 
        ((issues++))
    fi
    
    # Check volume
    echo -n "Volume exists: "
    if docker volume inspect "$VOLUME_NAME" >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        ((issues++))
    fi
    
    # Check settings file
    echo -n "Settings file: "
    if [[ -f "$SCRIPT_DIR/settings.js" ]]; then
        echo "✓"
    else
        echo "✗"
        ((issues++))
    fi
    
    # Check flows directory
    echo -n "Flows directory: "
    if [[ -d "$SCRIPT_DIR/flows" ]]; then
        echo "✓"
    else
        echo "✗"
        ((issues++))
    fi
    
    # Check configuration
    echo -n "Resource config: "
    local config_file
    config_file="$(secrets::get_project_config_file)"
    if [[ -f "$config_file" ]] && jq -e '.services.automation."node-red"' "$config_file" >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        ((issues++))
    fi
    
    echo
    if [[ $issues -eq 0 ]]; then
        log::success "Installation verification passed"
        return 0
    else
        log::error "Found $issues issues with installation"
        return 1
    fi
}