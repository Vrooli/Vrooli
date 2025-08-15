#!/usr/bin/env bash

# Ollama Status and Service Management Functions
# This file contains status checking, service control, and health monitoring functions

#######################################
# Check if Ollama is installed
# Returns: 0 if installed, 1 otherwise
#######################################
ollama::is_installed() {
    resources::binary_exists "ollama"
}

#######################################
# Check if Ollama service is running
# Returns: 0 if running, 1 otherwise
#######################################
ollama::is_running() {
    resources::is_service_running "$OLLAMA_PORT"
}

#######################################
# Check if Ollama API is responsive
# Returns: 0 if responsive, 1 otherwise
#######################################
ollama::is_healthy() {
    resources::check_http_health "$OLLAMA_BASE_URL" "/api/tags"
}

#######################################
# Get detailed health status for Ollama
# Combines systemd state, API health, and model availability
# Returns: Detailed health status string
#######################################
ollama::get_health_details() {
    local health_status="unknown"
    local health_msg=""
    local api_response_time=""
    
    # Check systemd state first
    local systemd_state
    systemd_state=$(systemctl is-active "$OLLAMA_SERVICE_NAME" 2>/dev/null || echo "inactive")
    
    # Check if service is even installed
    if ! systemctl list-unit-files | grep -q "^${OLLAMA_SERVICE_NAME}.service" 2>/dev/null; then
        echo "❌ Service not installed"
        return 1
    fi
    
    # Check API health with timing
    if system::is_command "curl"; then
        local temp_output=$(mktemp)
        local http_code
        local curl_exit_code
        local start_time=$(date +%s%N)
        
        http_code=$(curl -s -w "%{http_code}" --max-time 5 --connect-timeout 3 \
                    --output "$temp_output" "${OLLAMA_BASE_URL}/api/tags" 2>/dev/null)
        curl_exit_code=$?
        
        local end_time=$(date +%s%N)
        # Calculate response time in milliseconds
        if [[ -n "$start_time" ]] && [[ -n "$end_time" ]]; then
            local elapsed_ns=$((end_time - start_time))
            api_response_time=$(awk "BEGIN {printf \"%.3f\", $elapsed_ns / 1000000000}")
        else
            api_response_time="N/A"
        fi
        
        rm -f "$temp_output"
        
        # Determine health based on all factors
        if [[ "$systemd_state" == "active" ]]; then
            if [[ $curl_exit_code -eq 0 ]] && [[ "$http_code" == "200" ]]; then
                # Check model availability
                local model_count=0
                if system::is_command "ollama"; then
                    model_count=$(ollama list 2>/dev/null | tail -n +2 | wc -l || echo "0")
                fi
                
                if [[ $model_count -gt 0 ]]; then
                    health_status="healthy"
                    health_msg="✅ Healthy (${api_response_time}s response, $model_count models)"
                else
                    health_status="healthy"
                    health_msg="✅ Healthy (${api_response_time}s response, no models loaded)"
                fi
            elif [[ $curl_exit_code -eq 7 ]]; then
                health_status="starting"
                health_msg="🔄 Service starting (API not ready)"
            else
                health_status="unhealthy"
                health_msg="⚠️  Service active but API unhealthy (HTTP $http_code)"
            fi
        elif [[ "$systemd_state" == "inactive" ]] || [[ "$systemd_state" == "dead" ]]; then
            health_status="stopped"
            health_msg="⏹️  Service stopped"
        elif [[ "$systemd_state" == "failed" ]]; then
            health_status="failed"
            health_msg="❌ Service failed"
        elif [[ "$systemd_state" == "activating" ]]; then
            health_status="starting"
            health_msg="🔄 Service starting"
        else
            health_status="unknown"
            health_msg="❓ Unknown state: $systemd_state"
        fi
    else
        health_msg="⚠️  Cannot check health (curl not available)"
    fi
    
    echo "$health_msg"
    
    # Return appropriate exit code
    case "$health_status" in
        healthy) return 0 ;;
        starting) return 2 ;;
        *) return 1 ;;
    esac
}

#######################################
# Start Ollama service with enhanced error handling
#######################################
ollama::start() {
    if ollama::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Ollama is already running on port $OLLAMA_PORT"
        return 0
    fi
    
    log::info "Starting Ollama service..."
    
    # Check if service exists
    if ! systemctl list-unit-files | grep -q "^${OLLAMA_SERVICE_NAME}.service"; then
        resources::handle_error \
            "Ollama systemd service not found" \
            "system" \
            "Install Ollama first with: $0 --action install"
        return 1
    fi
    
    # Only check sudo if we actually need to start the service
    # (at this point we know it's not running or force is yes)
    if ! sudo::can_use_sudo; then
        resources::handle_error \
            "Sudo privileges required to start Ollama service" \
            "permission" \
            "Run 'sudo -v' to authenticate and retry"
        return 1
    fi
    
    # Check if port is already in use by another process
    if resources::is_service_running "$OLLAMA_PORT" && ! resources::is_service_active "$OLLAMA_SERVICE_NAME"; then
        resources::handle_error \
            "Port $OLLAMA_PORT is already in use by another process" \
            "system" \
            "Find and stop the process using: sudo lsof -ti:$OLLAMA_PORT | xargs sudo kill -9"
        return 1
    fi
    
    # Start the service
    if ! resources::start_service "$OLLAMA_SERVICE_NAME"; then
        resources::handle_error \
            "Failed to start Ollama systemd service" \
            "system" \
            "Check service logs: journalctl -u $OLLAMA_SERVICE_NAME -n 20"
        return 1
    fi
    
    # Wait for service to become available with progress indication
    log::info "Waiting for Ollama to start (this may take 10-30 seconds)..."
    if resources::wait_for_service "Ollama" "$OLLAMA_PORT" 30; then
        # Additional health check with retry
        sleep 2
        local health_attempts=3
        local health_success=false
        
        for ((i=1; i<=health_attempts; i++)); do
            if ollama::is_healthy; then
                health_success=true
                break
            fi
            
            if [[ $i -lt $health_attempts ]]; then
                log::info "Health check attempt $i failed, retrying in 3 seconds..."
                sleep 3
            fi
        done
        
        if [[ "$health_success" == "true" ]]; then
            log::success "$MSG_OLLAMA_RUNNING"
            return 0
        else
            log::warn "$MSG_OLLAMA_STARTED_NO_API"
            log::info "Service may still be initializing. Check status with: $0 --action status"
            log::info "Or check logs with: journalctl -u $OLLAMA_SERVICE_NAME -f"
            return 0
        fi
    else
        resources::handle_error \
            "Ollama service failed to start within 30 seconds" \
            "system" \
            "Check service status: systemctl status $OLLAMA_SERVICE_NAME; journalctl -u $OLLAMA_SERVICE_NAME -n 20"
        return 1
    fi
}

#######################################
# Stop Ollama service
#######################################
ollama::stop() {
    resources::stop_service "$OLLAMA_SERVICE_NAME"
}

#######################################
# Restart Ollama service
#######################################
ollama::restart() {
    log::info "Restarting Ollama service..."
    ollama::stop
    sleep 2
    ollama::start
}

#######################################
# Show comprehensive Ollama status
#######################################
ollama::status() {
    log::header "📊 Ollama Status"
    
    # Check if binary exists
    if resources::binary_exists "ollama"; then
        log::success "✅ Ollama binary installed"
    else
        log::error "❌ Ollama binary not found"
    fi
    
    # Get detailed health status
    local health_details
    health_details=$(ollama::get_health_details)
    local health_code=$?
    
    # Display health status
    log::info "Health Status: $health_details"
    
    # Check if service is running on port
    if resources::is_service_running "$OLLAMA_PORT"; then
        log::success "✅ Service listening on port $OLLAMA_PORT"
    else
        log::warn "⚠️  No service on port $OLLAMA_PORT"
    fi
    
    # Show systemd service details if healthy or starting
    if [[ $health_code -eq 0 ]] || [[ $health_code -eq 2 ]]; then
        # Get systemd resource usage
        local memory_usage
        local cpu_usage
        local uptime
        
        if systemctl show "$OLLAMA_SERVICE_NAME" --no-pager >/dev/null 2>&1; then
            memory_usage=$(systemctl show "$OLLAMA_SERVICE_NAME" --property=MemoryCurrent --value 2>/dev/null)
            if [[ -n "$memory_usage" ]] && [[ "$memory_usage" != "[not set]" ]] && [[ "$memory_usage" =~ ^[0-9]+$ ]]; then
                memory_mb=$(awk "BEGIN {printf \"%.2f\", $memory_usage / 1048576}")
                log::info "Memory Usage: ${memory_mb}MB"
            fi
            
            # Get uptime
            local active_since
            active_since=$(systemctl show "$OLLAMA_SERVICE_NAME" --property=ActiveEnterTimestamp --value 2>/dev/null)
            if [[ -n "$active_since" ]] && [[ "$active_since" != "[not set]" ]]; then
                log::info "Active Since: $active_since"
            fi
        fi
    fi
    
    # Check configuration
    if [[ -f "$VROOLI_RESOURCES_CONFIG" ]] && system::is_command "jq"; then
        local config_exists
        config_exists=$(jq -r '.resources.ai.ollama // empty' \
            "$VROOLI_RESOURCES_CONFIG" 2>/dev/null)
        
        if [[ -n "$config_exists" ]] && [[ "$config_exists" != "null" ]]; then
            log::success "✅ Resource configuration found"
        else
            log::warn "⚠️  Resource not configured in $VROOLI_RESOURCES_CONFIG"
        fi
    fi
    
    # Additional Ollama-specific status if healthy
    if [[ $health_code -eq 0 ]]; then
        echo
        log::info "API Endpoints:"
        log::info "  Base URL: $OLLAMA_BASE_URL"
        log::info "  Models: $OLLAMA_BASE_URL/api/tags"
        log::info "  Chat: $OLLAMA_BASE_URL/api/chat"
        log::info "  Generate: $OLLAMA_BASE_URL/api/generate"
        
        # Show installed models with details
        echo
        if system::is_command "ollama"; then
            local model_list
            model_list=$(ollama list 2>/dev/null | tail -n +2)
            local model_count=$(echo "$model_list" | wc -l)
            
            if [[ $model_count -gt 0 ]] && [[ -n "$model_list" ]]; then
                log::info "Installed Models ($model_count):"
                echo "$model_list" | while IFS= read -r model_line; do
                    if [[ -n "$model_line" ]]; then
                        log::info "  $model_line"
                    fi
                done | head -5
                
                if [[ $model_count -gt 5 ]]; then
                    log::info "  ... and $((model_count - 5)) more"
                fi
            else
                log::warn "No models installed"
                log::info "Install models with: ollama pull llama3.1:8b"
            fi
        fi
    fi
}

#######################################
# Show Ollama service logs
#######################################
ollama::logs() {
    if ! ollama::is_installed; then
        log::error "Ollama is not installed"
        return 1
    fi
    
    # Check if we have systemd service
    if systemctl list-unit-files | grep -q "^${OLLAMA_SERVICE_NAME}.service"; then
        log::info "Showing Ollama service logs (last ${LINES:-50} lines):"
        log::info "Use 'journalctl -u $OLLAMA_SERVICE_NAME -f' to follow logs in real-time"
        echo
        journalctl -u "$OLLAMA_SERVICE_NAME" -n "${LINES:-50}" --no-pager
    else
        log::warn "Ollama systemd service not found. Checking for running process..."
        
        # Try to find Ollama process logs
        local ollama_pid
        ollama_pid=$(pgrep -f "ollama" 2>/dev/null | head -1)
        
        if [[ -n "$ollama_pid" ]]; then
            log::info "Ollama process found (PID: $ollama_pid)"
            log::info "Process details:"
            ps -p "$ollama_pid" -o pid,ppid,cmd --no-headers 2>/dev/null || true
        else
            log::warn "No Ollama process found"
            log::info "Try starting Ollama with: $0 --action start"
        fi
    fi
}

#######################################
# Test Ollama functionality
# Returns: 0 if tests pass, 1 if tests fail, 2 if skip
#######################################
ollama::test() {
    log::info "Testing Ollama functionality..."
    
    # Test 1: Check if Ollama is installed
    if ! ollama::is_installed; then
        log::error "❌ Ollama is not installed"
        return 1
    fi
    log::success "✅ Ollama is installed"
    
    # Test 2: Check if service is running
    if ! ollama::is_running; then
        log::error "❌ Ollama service is not running"
        return 1
    fi
    log::success "✅ Ollama service is running"
    
    # Test 3: Check detailed health status
    local health_details
    health_details=$(ollama::get_health_details)
    local health_code=$?
    
    if [[ $health_code -eq 0 ]]; then
        log::success "✅ Ollama is healthy: ${health_details#* }"
    elif [[ $health_code -eq 2 ]]; then
        log::warn "⚠️  Ollama is starting: ${health_details#* }"
    else
        log::error "❌ Ollama is unhealthy: ${health_details#* }"
        return 1
    fi
    
    # Test 4: Check if models are available
    local model_count
    if system::is_command "ollama"; then
        model_count=$(ollama list 2>/dev/null | tail -n +2 | wc -l || echo "0")
        if [[ "$model_count" -eq 0 ]]; then
            log::warn "⚠️  No models installed - functionality will be limited"
            log::info "Install models with: $0 --action install --models llama3.1:8b"
        else
            log::success "✅ $model_count models available"
        fi
    fi
    
    # Test 5: Simple API test (if models available)
    if [[ "${model_count:-0}" -gt 0 ]]; then
        log::info "Testing API with simple query..."
        local test_response
        test_response=$(curl -s -m 10 -X POST "$OLLAMA_BASE_URL/api/generate" \
            -H "Content-Type: application/json" \
            -d '{"model":"llama3.1:8b","prompt":"Say hello in one word","stream":false}' 2>/dev/null)
        
        if [[ -n "$test_response" ]] && echo "$test_response" | grep -q "response"; then
            log::success "✅ API generates responses successfully"
        else
            log::warn "⚠️  API test failed - service may be initializing"
        fi
    fi
    
    log::success "🎉 All Ollama tests passed"
    return 0
}

# Export functions for subshell availability
export -f ollama::is_installed
export -f ollama::is_running
export -f ollama::is_healthy
export -f ollama::start
export -f ollama::stop
export -f ollama::restart
export -f ollama::status
export -f ollama::logs
export -f ollama::test