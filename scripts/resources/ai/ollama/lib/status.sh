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
    
    # Validate prerequisites
    if ! resources::can_sudo; then
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
    resources::print_status "ollama" "$OLLAMA_PORT" "$OLLAMA_SERVICE_NAME"
    
    # Additional Ollama-specific status
    if ollama::is_healthy; then
        echo
        log::info "API Endpoints:"
        log::info "  Base URL: $OLLAMA_BASE_URL"
        log::info "  Models: $OLLAMA_BASE_URL/api/tags"
        log::info "  Chat: $OLLAMA_BASE_URL/api/chat"
        log::info "  Generate: $OLLAMA_BASE_URL/api/generate"
        
        # Show installed models
        echo
        local model_count
        if system::is_command "ollama"; then
            model_count=$(ollama list 2>/dev/null | tail -n +2 | wc -l || echo "0")
            log::info "Installed models: $model_count"
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
    
    # Test 3: Check API health
    if ! ollama::is_healthy; then
        log::error "❌ Ollama API is not responding"
        return 1
    fi
    log::success "✅ Ollama API is healthy"
    
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