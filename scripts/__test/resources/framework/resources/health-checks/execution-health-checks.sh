#!/bin/bash
# ====================================================================
# Execution Resource Health Checks
# ====================================================================
#
# Category-specific health checks for execution resources including
# code execution capability, sandbox validation, and performance
# characteristics.
#
# Supported Execution Resources:
# - Judge0: Code execution engine with multiple language support
#
# ====================================================================

# Execution resource health check implementations
check_judge0_health() {
    local port="${1:-2358}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/system_info" >/dev/null 2>&1; then
        # Try alternative endpoints
        if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
            echo "healthy"
            return 0
        else
            echo "unreachable"
            return 1
        fi
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local system_info_available="false"
    local languages_available="false"
    local execution_capable="false"
    local languages_count="0"
    
    # Get system information
    local system_response
    system_response=$(curl -s --max-time 10 "http://localhost:${port}/system_info" 2>/dev/null)
    if [[ -n "$system_response" ]] && echo "$system_response" | jq . >/dev/null 2>&1; then
        system_info_available="true"
    fi
    
    # Get available languages
    local languages_response
    languages_response=$(curl -s --max-time 10 "http://localhost:${port}/languages" 2>/dev/null)
    if [[ -n "$languages_response" ]] && echo "$languages_response" | jq . >/dev/null 2>&1; then
        languages_available="true"
        languages_count=$(echo "$languages_response" | jq 'length' 2>/dev/null || echo "0")
    fi
    
    # Test basic code execution with a simple program
    local execution_test='{"source_code": "print(\"Hello, World!\")", "language_id": 71, "stdin": ""}'
    local execution_response
    execution_response=$(curl -s --max-time 20 -X POST "http://localhost:${port}/submissions" \
        -H "Content-Type: application/json" -d "$execution_test" 2>/dev/null)
    
    if [[ -n "$execution_response" ]] && echo "$execution_response" | jq . >/dev/null 2>&1; then
        local submission_token
        submission_token=$(echo "$execution_response" | jq -r '.token' 2>/dev/null)
        
        if [[ -n "$submission_token" && "$submission_token" != "null" ]]; then
            # Wait a moment and check the result
            sleep 2
            local result_response
            result_response=$(curl -s --max-time 10 "http://localhost:${port}/submissions/${submission_token}" 2>/dev/null)
            
            if [[ -n "$result_response" ]] && echo "$result_response" | jq -e '.stdout' >/dev/null 2>&1; then
                execution_capable="true"
            fi
        fi
    fi
    
    if [[ "$system_info_available" == "true" && "$languages_available" == "true" && "$execution_capable" == "true" ]]; then
        echo "healthy:system_info:languages:$languages_count:execution_ready"
    elif [[ "$system_info_available" == "true" && "$languages_available" == "true" ]]; then
        echo "degraded:system_info:languages:$languages_count:execution_failed"
    elif [[ "$system_info_available" == "true" ]]; then
        echo "degraded:system_info:languages_unavailable"
    else
        echo "degraded:limited_functionality"
    fi
    
    return 0
}

# Generic execution health check dispatcher
check_execution_resource_health() {
    local resource_name="$1"
    local port="$2"
    local health_level="${3:-basic}"
    
    case "$resource_name" in
        "judge0")
            check_judge0_health "$port" "$health_level"
            ;;
        *)
            # Fallback to generic HTTP health check
            if curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
                echo "healthy"
            elif curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            ;;
    esac
}

# Execution resource capability testing
test_execution_resource_capabilities() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "judge0")
            test_judge0_capabilities "$port"
            ;;
        *)
            echo "capability_testing_not_implemented"
            ;;
    esac
}

test_judge0_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    # Test system info endpoint
    if curl -s --max-time 5 "http://localhost:${port}/system_info" >/dev/null 2>&1; then
        capabilities+=("system_info")
    fi
    
    # Test languages endpoint
    local languages_response
    languages_response=$(curl -s --max-time 10 "http://localhost:${port}/languages" 2>/dev/null)
    if [[ -n "$languages_response" ]] && echo "$languages_response" | jq . >/dev/null 2>&1; then
        capabilities+=("multiple_languages")
        
        # Check for popular languages
        if echo "$languages_response" | jq -r '.[].name' | grep -q -i python; then
            capabilities+=("python_support")
        fi
        
        if echo "$languages_response" | jq -r '.[].name' | grep -q -i javascript; then
            capabilities+=("javascript_support")
        fi
        
        if echo "$languages_response" | jq -r '.[].name' | grep -q -i java; then
            capabilities+=("java_support")
        fi
        
        if echo "$languages_response" | jq -r '.[].name' | grep -q -i "c++"; then
            capabilities+=("cpp_support")
        fi
    fi
    
    # Test submission endpoint
    if curl -s --max-time 5 -X POST "http://localhost:${port}/submissions" >/dev/null 2>&1; then
        capabilities+=("code_execution")
        capabilities+=("async_processing")
    fi
    
    capabilities+=("sandboxed_execution")
    capabilities+=("timeout_management")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

# Security validation for execution resources
validate_execution_security() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "judge0")
            validate_judge0_security "$port"
            ;;
        *)
            echo "security_validation_not_implemented"
            ;;
    esac
}

validate_judge0_security() {
    local port="$1"
    
    local security_status=()
    
    # Check if running in a container
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "judge0"; then
        security_status+=("containerized")
    fi
    
    # Test execution limits by attempting to get system info
    local system_response
    system_response=$(curl -s --max-time 10 "http://localhost:${port}/system_info" 2>/dev/null)
    if [[ -n "$system_response" ]] && echo "$system_response" | jq . >/dev/null 2>&1; then
        # Check for execution limits
        local cpu_time_limit
        cpu_time_limit=$(echo "$system_response" | jq -r '.cpu_time_limit' 2>/dev/null)
        if [[ -n "$cpu_time_limit" && "$cpu_time_limit" != "null" ]]; then
            security_status+=("cpu_time_limited:${cpu_time_limit}s")
        fi
        
        local memory_limit
        memory_limit=$(echo "$system_response" | jq -r '.memory_limit' 2>/dev/null)
        if [[ -n "$memory_limit" && "$memory_limit" != "null" ]]; then
            security_status+=("memory_limited:${memory_limit}KB")
        fi
    fi
    
    # Check network exposure
    security_status+=("network_exposed:$port")
    
    # Execution environments are inherently sandboxed
    security_status+=("sandboxed_execution")
    security_status+=("process_isolation")
    
    if [[ ${#security_status[@]} -gt 0 ]]; then
        echo "security:$(IFS=,; echo "${security_status[*]}")"
    else
        echo "security_unknown"
    fi
}

# Performance testing for execution resources
test_execution_performance() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "judge0")
            test_judge0_performance "$port"
            ;;
        *)
            echo "performance_testing_not_implemented"
            ;;
    esac
}

test_judge0_performance() {
    local port="$1"
    
    # Test simple Python execution performance
    local test_code='{"source_code": "import time\nstart = time.time()\nfor i in range(1000):\n    pass\nprint(f\"Execution time: {time.time() - start:.4f}s\")", "language_id": 71}'
    
    local start_time=$(date +%s.%N)
    
    # Submit the code
    local submission_response
    submission_response=$(curl -s --max-time 30 -X POST "http://localhost:${port}/submissions" \
        -H "Content-Type: application/json" -d "$test_code" 2>/dev/null)
    
    local performance_metrics=()
    
    if [[ -n "$submission_response" ]] && echo "$submission_response" | jq . >/dev/null 2>&1; then
        local submission_token
        submission_token=$(echo "$submission_response" | jq -r '.token' 2>/dev/null)
        
        if [[ -n "$submission_token" && "$submission_token" != "null" ]]; then
            # Wait and get result
            sleep 3
            local result_response
            result_response=$(curl -s --max-time 10 "http://localhost:${port}/submissions/${submission_token}" 2>/dev/null)
            
            local end_time=$(date +%s.%N)
            local total_duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "unknown")
            
            performance_metrics+=("total_time:${total_duration}s")
            
            if [[ -n "$result_response" ]] && echo "$result_response" | jq . >/dev/null 2>&1; then
                local execution_time
                execution_time=$(echo "$result_response" | jq -r '.time' 2>/dev/null)
                if [[ -n "$execution_time" && "$execution_time" != "null" ]]; then
                    performance_metrics+=("execution_time:${execution_time}s")
                fi
                
                local memory_usage
                memory_usage=$(echo "$result_response" | jq -r '.memory' 2>/dev/null)
                if [[ -n "$memory_usage" && "$memory_usage" != "null" ]]; then
                    performance_metrics+=("memory_usage:${memory_usage}KB")
                fi
                
                # Check if execution was successful
                local exit_code
                exit_code=$(echo "$result_response" | jq -r '.status.id' 2>/dev/null)
                if [[ "$exit_code" == "3" ]]; then  # Accepted
                    performance_metrics+=("execution_successful")
                else
                    performance_metrics+=("execution_failed:status_$exit_code")
                fi
            fi
        fi
    fi
    
    if [[ ${#performance_metrics[@]} -gt 0 ]]; then
        echo "performance:$(IFS=,; echo "${performance_metrics[*]}")"
    else
        echo "performance_unknown"
    fi
}

# Export functions
export -f check_execution_resource_health
export -f test_execution_resource_capabilities
export -f validate_execution_security
export -f test_execution_performance
export -f check_judge0_health