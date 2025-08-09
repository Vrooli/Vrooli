#!/usr/bin/env bash
# Node-RED User Messages and Help Text
# All user-facing messages, prompts, and documentation

#######################################
# Display usage information
#######################################
node_red::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  # Install with custom image"
    echo "  $0 --action install --build-image yes"
    echo
    echo "  # Import flows from file"
    echo "  $0 --action flow-import --flow-file ./my-flows.json"
    echo
    echo "  # Execute test flow"
    echo "  $0 --action flow-execute --endpoint /test/exec --data '{\"command\":\"ls\"}'"
    echo
    echo "  # Export all flows"
    echo "  $0 --action flow-export --output ./backup-flows.json"
    echo
}

#######################################
# Success messages
#######################################
node_red::show_success_message() {
    log::success "Node-RED installed successfully!"
    log::info "Access Node-RED at: http://localhost:$RESOURCE_PORT"
    echo
    echo "To check status: $0 --action status"
    echo "To view logs: $0 --action logs"
    echo "To test flows: $0 --action test"
}

node_red::show_installation_complete() {
    log::success "Node-RED installed successfully!"
    log::info "Access Node-RED at: http://localhost:$RESOURCE_PORT"
    
    # Update resource configuration
    update_resource_config
    
    # Import test flows if they exist
    if [[ -d "$SCRIPT_DIR/flows" ]] && ls "$SCRIPT_DIR/flows"/*.json >/dev/null 2>&1; then
        log::info "Importing example flows..."
        for flow in "$SCRIPT_DIR/flows"/*.json; do
            import_flow "$flow" || log::warning "Failed to import $(basename "$flow")"
        done
    fi
}

#######################################
# Status display messages
#######################################
node_red::show_status_header() {
    echo "Node-RED Status:"
    echo "================"
}

node_red::show_not_installed() {
    echo "Status: Not installed"
}

node_red::show_running_status() {
    echo "Status: Running"
    echo "URL: http://localhost:$RESOURCE_PORT"
}

node_red::show_stopped_status() {
    echo "Status: Stopped"
}

node_red::show_container_info() {
    local container_info="$1"
    echo
    echo "Container Information:"
    echo "- Name: $CONTAINER_NAME"
    echo "- Image: $(echo "$container_info" | jq -r '.[0].Config.Image')"
    echo "- Created: $(echo "$container_info" | jq -r '.[0].Created' | cut -d'T' -f1)"
    echo "- Uptime: $(docker ps -f name="$CONTAINER_NAME" --format "table {{.Status}}" | tail -n 1)"
}

node_red::show_resource_usage() {
    echo
    echo "Resource Usage:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" "$CONTAINER_NAME"
}

node_red::show_health_status() {
    echo
    local health
    health=$(docker inspect --format='{{.State.Health.Status}}' "$CONTAINER_NAME" 2>/dev/null || echo "unknown")
    echo "Health: $health"
}

node_red::show_flow_info() {
    echo
    echo "Flow Information:"
    local flow_count=$(docker exec "$CONTAINER_NAME" ls /data/*.json 2>/dev/null | wc -l || echo "0")
    echo "- Flow files: $flow_count"
}

#######################################
# Test result messages
#######################################
node_red::show_test_header() {
    log::info "Running Node-RED validation test suite..."
}

node_red::show_test_result() {
    local test_name="$1"
    local result="$2"
    local message="${3:-}"
    
    echo -n "Testing $test_name... "
    if [[ "$result" == "passed" ]]; then
        echo "✓ PASSED"
    else
        echo "✗ FAILED${message:+ ($message)}"
    fi
}

node_red::show_test_summary() {
    local tests_passed="$1"
    local tests_failed="$2"
    local total=$((tests_passed + tests_failed))
    
    echo
    echo "Test Summary:"
    echo "============="
    echo "Passed: $tests_passed"
    echo "Failed: $tests_failed"
    echo "Total: $total"
    
    if [[ $tests_failed -eq 0 ]]; then
        log::success "All tests passed!"
        return 0
    else
        log::error "Some tests failed"
        return 1
    fi
}

#######################################
# Flow management messages
#######################################
node_red::show_flow_export_success() {
    local output_file="$1"
    local flow_count="$2"
    local node_count="$3"
    
    log::success "Flows exported successfully to: $output_file"
    echo "Exported $flow_count flows with $node_count total nodes"
}

node_red::show_flow_execution_success() {
    local response="$1"
    
    log::success "Flow executed successfully"
    if [[ -n "$response" ]]; then
        echo "Response:"
        echo "$response" | jq . 2>/dev/null || echo "$response"
    fi
}

#######################################
# Error messages
#######################################
node_red::show_docker_build_error() {
    log::error "Dockerfile not found at $SCRIPT_DIR/docker/Dockerfile"
}

node_red::show_build_failed_error() {
    log::error "Failed to build custom image"
}

node_red::show_startup_timeout_error() {
    local max_attempts="$1"
    log::error "Node-RED failed to start after $max_attempts attempts"
}

node_red::show_not_running_error() {
    log::error "Node-RED is not running"
}

node_red::show_already_installed_warning() {
    log::warning "Node-RED is already installed"
}

node_red::show_already_running_warning() {
    log::warning "Node-RED is already running"
}

node_red::show_not_installed_error() {
    log::error "Node-RED is not installed"
}

node_red::show_installation_failed_error() {
    log::error "Node-RED installation failed"
}

node_red::show_flow_fetch_error() {
    log::error "Failed to fetch flows"
}

node_red::show_flow_export_error() {
    log::error "Failed to export flows"
}

node_red::show_flow_import_error() {
    log::error "Failed to import flows"
}

node_red::show_flow_execution_error() {
    log::error "Flow execution failed"
}

#######################################
# Prompts and confirmations
#######################################
node_red::prompt_reinstall() {
    if [[ "$YES" == "yes" ]]; then
        log::info "Auto-confirming reinstallation due to --yes flag"
        return 0
    else
        read -p "Do you want to reinstall? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            return 0
        else
            log::info "Installation cancelled"
            return 1
        fi
    fi
}

node_red::prompt_uninstall() {
    if [[ "$FORCE" != "yes" ]]; then
        log::warning "This will remove Node-RED and all its data!"
        if [[ "$YES" == "yes" ]]; then
            log::info "Auto-confirming uninstall due to --yes flag"
            return 0
        else
            read -p "Are you sure? (y/N) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                return 0
            else
                log::info "Uninstall cancelled"
                return 1
            fi
        fi
    fi
    return 0
}

#######################################
# Informational messages
#######################################
node_red::show_waiting_message() {
    log::info "Waiting for Node-RED to be ready..."
}

node_red::show_building_image() {
    log::info "Building custom Node-RED image..."
}

node_red::show_installing() {
    log::info "Installing Node-RED..."
}

node_red::show_uninstalling() {
    log::info "Uninstalling Node-RED..."
}

node_red::show_starting() {
    log::info "Starting Node-RED..."
}

node_red::show_stopping() {
    log::info "Stopping Node-RED..."
}

node_red::show_restarting() {
    log::info "Restarting Node-RED..."
}

node_red::show_creating_network() {
    log::info "Creating Docker network: $NETWORK_NAME"
}

node_red::show_creating_volume() {
    log::info "Creating data volume: $VOLUME_NAME"
}

node_red::show_creating_settings() {
    log::info "Creating default settings.js"
}

node_red::show_starting_container() {
    log::info "Starting Node-RED container..."
}

node_red::show_importing_flows() {
    log::info "Importing example flows..."
}

node_red::show_exporting_flows() {
    local output_file="$1"
    log::info "Exporting flows to: $output_file"
}

node_red::show_importing_flow() {
    local flow_file="$1"
    log::info "Importing flows from: $flow_file"
}

node_red::show_executing_flow() {
    local url="$1"
    log::info "Executing flow at: $url"
}

#######################################
# Host validation messages
#######################################
node_red::show_host_validation_header() {
    log::info "Validating host command execution..."
}

node_red::show_docker_validation_header() {
    log::info "Validating Docker socket access..."
}

node_red::show_host_access_summary() {
    local passed="$1"
    local failed="$2"
    echo
    echo "Host Access Summary: $passed passed, $failed failed"
}

node_red::show_docker_access_warning() {
    log::warning "Docker socket access not available"
    log::info "This is expected if Docker socket was not mounted"
}

#######################################
# Metrics messages
#######################################
node_red::show_metrics_header() {
    log::info "Node-RED Resource Metrics"
    echo "========================="
}

node_red::show_flow_metrics() {
    echo
    echo "Flow Metrics:"
    local flow_count=$(docker exec "$CONTAINER_NAME" find /data -name "*.json" -type f 2>/dev/null | wc -l || echo "0")
    echo "- Flow files: $flow_count"
    
    # Disk usage
    local disk_usage=$(docker exec "$CONTAINER_NAME" du -sh /data 2>/dev/null | cut -f1 || echo "unknown")
    echo "- Data directory size: $disk_usage"
}

node_red::show_nodejs_metrics() {
    echo
    echo "Node.js Process:"
    docker exec "$CONTAINER_NAME" ps aux | grep node-red | grep -v grep || echo "Process information not available"
}

#######################################
# Interrupt handling
#######################################
node_red::show_interrupt_message() {
    echo ""
    log::info "Node-RED installation interrupted by user. Exiting..."
}