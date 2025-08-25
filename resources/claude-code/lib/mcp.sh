#!/usr/bin/env bash
set -euo pipefail

# MCP (Model Context Protocol) helper functions for Claude Code integration
# This file provides utilities for registering Vrooli as an MCP server with Claude Code

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLAUDE_CODE_LIB_DIR="${APP_ROOT}/resources/claude-code/lib"

# Source required libraries
source "${CLAUDE_CODE_LIB_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"

# MCP configuration constants are defined in config/defaults.sh
# Common utilities are sourced by manage.sh

#######################################
# Detect if Vrooli server is running and get connection info
# Returns: JSON object with server info or empty if not running
#######################################
mcp::detect_vrooli_server() {
    local base_url="http://localhost:${VROOLI_DEFAULT_PORT}"
    local health_url="${base_url}${VROOLI_HEALTH_ENDPOINT}"
    
    # Try to detect Vrooli server on default port
    if system::is_command "curl"; then
        if curl -f -s --max-time 5 "$health_url" >/dev/null 2>&1; then
            echo "{\"baseUrl\":\"$base_url\",\"mcpEndpoint\":\"${base_url}${VROOLI_MCP_ENDPOINT}\",\"port\":\"${VROOLI_DEFAULT_PORT}\"}"
            return 0
        fi
    fi
    
    # Try alternative ports if default fails
    for port in 3001 8080 8000; do
        local alt_url="http://localhost:${port}"
        local alt_health="${alt_url}${VROOLI_HEALTH_ENDPOINT}"
        
        if system::is_command "curl"; then
            if curl -f -s --max-time 3 "$alt_health" >/dev/null 2>&1; then
                log::info "Found Vrooli server on port $port"
                echo "{\"baseUrl\":\"$alt_url\",\"mcpEndpoint\":\"${alt_url}${VROOLI_MCP_ENDPOINT}\",\"port\":\"$port\"}"
                return 0
            fi
        fi
    done
    
    # No Vrooli server found
    echo ""
    return 1
}

#######################################
# Get or generate API key for MCP access
# Arguments:
#   $1 - Optional: specific API key to use
# Returns: API key string
#######################################
mcp::get_api_key() {
    local api_key="${1:-}"
    
    # If API key provided, use it
    if [[ -n "$api_key" ]]; then
        echo "$api_key"
        return 0
    fi
    
    # Check environment variables
    if [[ -n "${VROOLI_API_KEY:-}" ]]; then
        echo "$VROOLI_API_KEY"
        return 0
    fi
    
    if [[ -n "${CLAUDE_MCP_API_KEY:-}" ]]; then
        echo "$CLAUDE_MCP_API_KEY"
        return 0
    fi
    
    # For now, suggest manual configuration
    log::warn "No API key found. Please set VROOLI_API_KEY environment variable."
    log::info "You can generate an API key from your Vrooli dashboard."
    echo ""
    return 1
}

#######################################
# Check if Claude Code CLI is installed and accessible
# Returns: 0 if available, 1 otherwise
#######################################
mcp::check_claude_code_available() {
    if system::is_command "claude"; then
        # Verify it's actually Claude Code CLI
        if claude --version >/dev/null 2>&1; then
            return 0
        fi
    fi
    return 1
}

#######################################
# Determine appropriate configuration scope for MCP registration
# Returns: scope name (local, project, user)
#######################################
mcp::determine_scope() {
    local scope="${1:-auto}"
    
    if [[ "$scope" != "auto" ]]; then
        echo "$scope"
        return 0
    fi
    
    # Auto-detect best scope
    if [[ -f "./.claude.json" ]]; then
        echo "local"    # Project has Claude config
    elif [[ -n "${TEAM_MODE:-}" && "$TEAM_MODE" == "true" ]]; then
        echo "project"  # Team development
    else
        echo "user"     # Personal/global
    fi
}

#######################################
# Register Vrooli server with Claude Code
# Arguments:
#   $1 - Server URL (e.g., http://localhost:3000)
#   $2 - API key
#   $3 - Scope (local, project, user)
# Returns: 0 on success, 1 on failure
#######################################
mcp::register_server() {
    local server_url="$1"
    local api_key="$2"
    local scope="$3"
    
    if ! mcp::check_claude_code_available; then
        log::error "Claude Code CLI not available"
        return 1
    fi
    
    local mcp_endpoint="${server_url}${VROOLI_MCP_ENDPOINT}"
    
    log::info "Registering Vrooli MCP server..."
    log::info "  URL: $mcp_endpoint"
    log::info "  Scope: $scope"
    
    # Build Claude MCP command
    local cmd="claude mcp add $VROOLI_MCP_SERVER_NAME"
    cmd="$cmd --type sse"
    cmd="$cmd --url \"$mcp_endpoint\""
    cmd="$cmd --scope \"$scope\""
    
    # Add authentication if API key provided
    if [[ -n "$api_key" ]]; then
        cmd="$cmd --headers \"Authorization: Bearer $api_key\""
    fi
    
    # Execute registration
    if eval "$cmd" 2>/dev/null; then
        log::success "✓ Vrooli MCP server registered successfully"
        return 0
    else
        log::error "Failed to register Vrooli MCP server"
        return 1
    fi
}

#######################################
# Unregister Vrooli server from Claude Code
# Arguments:
#   $1 - Optional: scope (local, project, user)
# Returns: 0 on success, 1 on failure
#######################################
mcp::unregister_server() {
    local scope="${1:-auto}"
    
    if ! mcp::check_claude_code_available; then
        log::error "Claude Code CLI not available"
        return 1
    fi
    
    scope=$(mcp::determine_scope "$scope")
    
    log::info "Unregistering Vrooli MCP server (scope: $scope)..."
    
    if claude mcp remove "$VROOLI_MCP_SERVER_NAME" --scope "$scope" 2>/dev/null; then
        log::success "✓ Vrooli MCP server unregistered successfully"
        return 0
    else
        log::warn "Failed to unregister Vrooli MCP server (may not have been registered)"
        return 1
    fi
}

#######################################
# Check if Vrooli is registered as an MCP server with Claude Code
# Arguments:
#   $1 - Optional: scope to check (local, project, user, or "all")
# Returns: JSON object with registration status
#######################################
mcp::get_registration_status() {
    local check_scope="${1:-all}"
    local status="{\"registered\":false,\"scopes\":[]}"
    
    if ! mcp::check_claude_code_available; then
        echo "{\"error\":\"Claude Code CLI not available\",\"registered\":false,\"scopes\":[]}"
        return 1
    fi
    
    local registered_scopes=()
    local scopes_to_check=()
    
    if [[ "$check_scope" == "all" ]]; then
        scopes_to_check=("local" "project" "user")
    else
        scopes_to_check=("$check_scope")
    fi
    
    # Check each scope
    for scope in "${scopes_to_check[@]}"; do
        if claude mcp list --scope "$scope" 2>/dev/null | grep -q "$VROOLI_MCP_SERVER_NAME"; then
            registered_scopes+=("$scope")
        fi
    done
    
    # Build JSON response
    if [[ ${#registered_scopes[@]} -gt 0 ]]; then
        local scopes_json
        scopes_json=$(printf '"%s",' "${registered_scopes[@]}" | sed 's/,$//')
        status="{\"registered\":true,\"scopes\":[$scopes_json]}"
    fi
    
    echo "$status"
    return 0
}

#######################################
# Validate MCP connection to Vrooli server
# Arguments:
#   $1 - Optional: server URL to test
# Returns: 0 if connection successful, 1 otherwise
#######################################
mcp::validate_connection() {
    local server_url="${1:-}"
    
    if [[ -z "$server_url" ]]; then
        local server_info
        server_info=$(mcp::detect_vrooli_server)
        if [[ -z "$server_info" ]]; then
            log::error "No Vrooli server detected"
            return 1
        fi
        
        # Extract URL from JSON (simple approach)
        server_url=$(echo "$server_info" | grep -o '"mcpEndpoint":"[^"]*"' | cut -d'"' -f4)
    fi
    
    log::info "Validating MCP connection to: $server_url"
    
    # Test basic connectivity
    if system::is_command "curl"; then
        if curl -f -s --max-time 10 "$server_url" >/dev/null 2>&1; then
            log::success "✓ MCP endpoint accessible"
            return 0
        else
            log::error "✗ MCP endpoint not accessible"
            return 1
        fi
    else
        log::warn "curl not available, skipping connection validation"
        return 0
    fi
}

#######################################
# Get comprehensive MCP status information
# Returns: JSON object with detailed status
#######################################
mcp::get_status() {
    local claude_available
    local vrooli_detected
    local registration_status
    local server_info
    
    # Check Claude Code availability
    if mcp::check_claude_code_available; then
        local claude_version
        claude_version=$(claude --version 2>/dev/null | head -1 || echo "unknown")
        claude_available="{\"available\":true,\"version\":\"$claude_version\"}"
    else
        claude_available="{\"available\":false}"
    fi
    
    # Check Vrooli server
    server_info=$(mcp::detect_vrooli_server)
    if [[ -n "$server_info" ]]; then
        vrooli_detected="{\"detected\":true,\"serverInfo\":$server_info}"
    else
        vrooli_detected="{\"detected\":false}"
    fi
    
    # Get registration status
    registration_status=$(mcp::get_registration_status)
    
    # Combine into comprehensive status
    echo "{\"claudeCode\":$claude_available,\"vrooliServer\":$vrooli_detected,\"registration\":$registration_status}"
}

#######################################
# Create MCP configuration file based on template and scope
# Arguments:
#   $1 - Scope (local, project, user)
#   $2 - Server URL
#   $3 - API key (optional)
# Returns: 0 on success, 1 on failure
#######################################
mcp::create_config() {
    local scope="$1"
    local server_url="$2"
    local api_key="${3:-}"
    
    local template_file="${CLAUDE_CODE_LIB_DIR}/templates/mcp-${scope}.json"
    local config_file
    
    # Determine config file location based on scope
    case "$scope" in
        "local")
            config_file="./.claude.json"
            ;;
        "project")
            config_file="./.mcp.json"
            ;;
        "user")
            config_file="$HOME/.claude.json"
            ;;
        *)
            log::error "Invalid scope: $scope"
            return 1
            ;;
    esac
    
    # Check if template exists
    if [[ ! -f "$template_file" ]]; then
        log::error "Template file not found: $template_file"
        return 1
    fi
    
    log::info "Creating MCP configuration file: $config_file"
    
    # Read template and substitute variables
    local config_content
    config_content=$(cat "$template_file")
    
    # Replace URL placeholder
    config_content=$(echo "$config_content" | sed "s|http://localhost:3000|$server_url|g")
    
    # Handle API key
    if [[ -n "$api_key" ]]; then
        # Replace environment variable with actual key (for local testing)
        config_content=$(echo "$config_content" | sed "s/\${VROOLI_API_KEY}/$api_key/g")
    else
        # Keep environment variable placeholder
        log::info "API key not provided, using environment variable placeholder"
    fi
    
    # Create config directory if it doesn't exist
    local config_dir
    config_dir=${config_file%/*
    if [[ "$config_dir" != "." && ! -d "$config_dir" ]]; then
        mkdir -p "$config_dir"
    fi
    
    # Write configuration file
    echo "$config_content" > "$config_file"
    
    if [[ -f "$config_file" ]]; then
        log::success "✓ MCP configuration created: $config_file"
        
        # Set appropriate permissions
        chmod 600 "$config_file"
        
        return 0
    else
        log::error "Failed to create configuration file"
        return 1
    fi
}

#######################################
# Auto-register Vrooli with Claude Code if both are available
# Arguments:
#   $1 - Optional: scope (local, project, user, auto)
#   $2 - Optional: API key
# Returns: 0 on success, 1 on failure
#######################################
mcp::auto_register() {
    local scope="${1:-auto}"
    local api_key="${2:-}"
    
    log::info "Attempting auto-registration of Vrooli MCP server..."
    
    # Check prerequisites
    if ! mcp::check_claude_code_available; then
        log::error "Claude Code CLI not available for auto-registration"
        return 1
    fi
    
    # Detect Vrooli server
    local server_info
    server_info=$(mcp::detect_vrooli_server)
    if [[ -z "$server_info" ]]; then
        log::error "Vrooli server not detected for auto-registration"
        return 1
    fi
    
    # Extract server URL
    local server_url
    server_url=$(echo "$server_info" | grep -o '"baseUrl":"[^"]*"' | cut -d'"' -f4)
    
    # Get API key
    if [[ -z "$api_key" ]]; then
        api_key=$(mcp::get_api_key)
        if [[ -z "$api_key" ]]; then
            log::warn "No API key available, registering without authentication"
        fi
    fi
    
    # Determine scope
    scope=$(mcp::determine_scope "$scope")
    
    # Check if already registered
    local registration_status
    registration_status=$(mcp::get_registration_status "$scope")
    if echo "$registration_status" | grep -q '"registered":true'; then
        log::info "Vrooli MCP server already registered in scope: $scope"
        return 0
    fi
    
    # Create configuration file
    if mcp::create_config "$scope" "$server_url" "$api_key"; then
        log::info "Configuration file created successfully"
    else
        log::error "Failed to create configuration file"
        return 1
    fi
    
    # Perform registration using Claude Code CLI
    mcp::register_server "$server_url" "$api_key" "$scope"
}

#######################################
# Claude Code specific MCP functions
# These are wrappers that integrate with the Claude Code management script
#######################################

#######################################
# Register Vrooli as MCP server with Claude Code
#######################################
claude_code::register_mcp() {
    log::header "🔗 Registering Vrooli MCP Server"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    # Detect Vrooli server
    local server_info
    server_info=$(mcp::detect_vrooli_server)
    if [[ -z "$server_info" ]]; then
        log::error "Vrooli server not detected. Please ensure Vrooli is running."
        log::info "Expected health endpoint: http://localhost:${VROOLI_DEFAULT_PORT}${VROOLI_HEALTH_ENDPOINT}"
        return 1
    fi
    
    # Extract server URL or use provided one
    local server_url="$MCP_SERVER_URL"
    if [[ -z "$server_url" ]]; then
        server_url=$(echo "$server_info" | grep -o '"baseUrl":"[^"]*"' | cut -d'"' -f4)
    fi
    
    # Get API key
    local api_key="$MCP_API_KEY"
    if [[ -z "$api_key" ]]; then
        api_key=$(mcp::get_api_key)
    fi
    
    # Determine scope
    local scope
    scope=$(mcp::determine_scope "$MCP_SCOPE")
    
    # Check if already registered
    local registration_status
    registration_status=$(mcp::get_registration_status "$scope")
    if echo "$registration_status" | grep -q '"registered":true'; then
        log::info "Vrooli MCP server already registered in scope: $scope"
        
        # Validate connection
        if mcp::validate_connection; then
            log::success "✓ MCP connection validated"
            return 0
        else
            log::warn "Registration exists but connection failed. Re-registering..."
        fi
    fi
    
    # Perform registration
    if mcp::register_server "$server_url" "$api_key" "$scope"; then
        # Validate the new registration
        if mcp::validate_connection; then
            log::success "✓ Vrooli MCP server registered and validated successfully"
            
            # Show next steps
            claude_code::mcp_next_steps
            
            return 0
        else
            log::error "Registration succeeded but connection validation failed"
            return 1
        fi
    else
        log::error "Failed to register Vrooli MCP server"
        return 1
    fi
}

#######################################
# Unregister Vrooli MCP server from Claude Code
#######################################
claude_code::unregister_mcp() {
    log::header "🗑️ Unregistering Vrooli MCP Server"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed"
        return 1
    fi
    
    # Determine scope
    local scope
    scope=$(mcp::determine_scope "$MCP_SCOPE")
    
    # Check current registration status
    local registration_status
    registration_status=$(mcp::get_registration_status "$scope")
    if ! echo "$registration_status" | grep -q '"registered":true'; then
        log::warn "Vrooli MCP server is not registered in scope: $scope"
        return 0
    fi
    
    # Confirm unregistration
    if ! confirm "Remove Vrooli MCP server registration from scope '$scope'?"; then
        log::info "Unregistration cancelled"
        return 0
    fi
    
    # Perform unregistration
    if mcp::unregister_server "$scope"; then
        log::success "✓ Vrooli MCP server unregistered successfully"
        return 0
    else
        log::error "Failed to unregister Vrooli MCP server"
        return 1
    fi
}

#######################################
# Check Vrooli MCP registration status
#######################################
claude_code::mcp_status() {
    log::header "📊 Vrooli MCP Status"
    
    # Initialize MCP_FORMAT with default value if not set
    MCP_FORMAT="${MCP_FORMAT:-text}"
    
    # Get comprehensive status
    local status
    status=$(mcp::get_status)
    
    if [[ "$MCP_FORMAT" == "json" ]]; then
        echo "$status"
        return 0
    fi
    
    # Parse status for human-readable output
    local claude_available
    claude_available=$(echo "$status" | grep -o '"available":[^,}]*' | cut -d':' -f2)
    
    local vrooli_detected
    vrooli_detected=$(echo "$status" | grep -o '"detected":[^,}]*' | cut -d':' -f2)
    
    local registration_status
    registration_status=$(echo "$status" | grep -o '"registered":[^,}]*' | cut -d':' -f2)
    
    # Display Claude Code status
    if [[ "$claude_available" == "true" ]]; then
        local claude_version
        claude_version=$(echo "$status" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
        log::success "✓ Claude Code CLI is available"
        log::info "  Version: $claude_version"
    else
        log::warn "✗ Claude Code CLI not available"
        log::info "  Run: $0 --action install"
    fi
    
    echo
    
    # Display Vrooli server status
    if [[ "$vrooli_detected" == "true" ]]; then
        log::success "✓ Vrooli server detected"
        local server_url
        server_url=$(echo "$status" | grep -o '"baseUrl":"[^"]*"' | cut -d'"' -f4)
        log::info "  Server URL: $server_url"
        log::info "  MCP Endpoint: ${server_url}${VROOLI_MCP_ENDPOINT}"
    else
        log::warn "✗ Vrooli server not detected"
        log::info "  Expected: http://localhost:${VROOLI_DEFAULT_PORT}"
    fi
    
    echo
    
    # Display registration status
    if [[ "$registration_status" == "true" ]]; then
        log::success "✓ Vrooli MCP server is registered"
        local scopes
        scopes=$(echo "$status" | grep -o '"scopes":\[[^]]*\]' | sed 's/"scopes":\[//' | sed 's/\]//' | tr ',' ' ')
        log::info "  Registered scopes: $scopes"
    else
        log::warn "✗ Vrooli MCP server not registered"
        log::info "  Run: $0 --action register-mcp"
    fi
    
    # Show next steps if everything is ready
    if [[ "$claude_available" == "true" && "$vrooli_detected" == "true" && "$registration_status" == "true" ]]; then
        echo
        log::header "🎯 Ready to Use"
        log::info "Start Claude Code and use @vrooli to access Vrooli tools"
    fi
}

#######################################
# Test MCP connection to Vrooli server
#######################################
claude_code::mcp_test() {
    log::header "🧪 Testing Vrooli MCP Connection"
    
    # Check prerequisites
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed"
        return 1
    fi
    
    # Detect Vrooli server
    local server_info
    server_info=$(mcp::detect_vrooli_server)
    if [[ -z "$server_info" ]]; then
        log::error "Vrooli server not detected"
        return 1
    fi
    
    local server_url
    server_url=$(echo "$server_info" | grep -o '"baseUrl":"[^"]*"' | cut -d'"' -f4)
    local mcp_endpoint="${server_url}${VROOLI_MCP_ENDPOINT}"
    
    log::info "Testing connection to: $mcp_endpoint"
    
    # Test basic connectivity
    if mcp::validate_connection "$mcp_endpoint"; then
        log::success "✓ Basic connectivity test passed"
    else
        log::error "✗ Basic connectivity test failed"
        return 1
    fi
    
    # Test health endpoint
    local health_endpoint="${server_url}${VROOLI_HEALTH_ENDPOINT}"
    log::info "Testing health endpoint: $health_endpoint"
    
    if system::is_command "curl"; then
        local health_response
        health_response=$(curl -f -s --max-time 10 "$health_endpoint" 2>/dev/null)
        if [[ -n "$health_response" ]]; then
            log::success "✓ Health endpoint accessible"
            
            # Check if response contains expected MCP info
            if echo "$health_response" | grep -q "activeConnections"; then
                log::success "✓ MCP health data found"
                local active_connections
                active_connections=$(echo "$health_response" | grep -o '"activeConnections":[0-9]*' | cut -d':' -f2)
                log::info "  Active MCP connections: $active_connections"
            else
                log::warn "⚠️  Health endpoint accessible but no MCP data found"
            fi
        else
            log::error "✗ Health endpoint not accessible"
            return 1
        fi
    else
        log::warn "curl not available, skipping health endpoint test"
    fi
    
    # Check registration status
    local registration_status
    registration_status=$(mcp::get_registration_status)
    if echo "$registration_status" | grep -q '"registered":true'; then
        log::success "✓ MCP server is registered with Claude Code"
    else
        log::warn "⚠️  MCP server not registered with Claude Code"
        log::info "  Run: $0 --action register-mcp"
    fi
    
    echo
    log::success "✓ MCP connection test completed"
}
