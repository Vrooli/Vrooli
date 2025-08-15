#!/usr/bin/env bash
################################################################################
# agent-s2 Resource CLI
# 
# Lightweight CLI interface for agent-s2 using the CLI Command Framework
#
# Usage:
#   resource-agent-s2 <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    AGENT_S2_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    AGENT_S2_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
AGENT_S2_CLI_DIR="$(cd "$(dirname "$AGENT_S2_CLI_SCRIPT")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$AGENT_S2_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$AGENT_S2_CLI_DIR"
export AGENT_S2_SCRIPT_DIR="$AGENT_S2_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/lib/cli-command-framework.sh"

# Source agent-s2 configuration
# shellcheck disable=SC1091
source "${AGENT_S2_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${AGENT_S2_CLI_DIR}/config/messages.sh" 2>/dev/null || true
agents2::export_config 2>/dev/null || true
agents2::export_messages 2>/dev/null || true

# Source agent-s2 libraries
for lib in common docker status api install modes; do
    lib_file="${AGENT_S2_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "agent-s2" "Agent-S2 browser automation and AI control management"

# Register additional agent-s2-specific commands
cli::register_command "inject" "Inject automation/config into agent-s2" "resource_cli::inject" "modifies-system"
cli::register_command "screenshot" "Take screenshot" "resource_cli::screenshot"
cli::register_command "execute" "Execute automation task" "resource_cli::execute"
cli::register_command "switch-mode" "Switch between sandbox/host mode" "resource_cli::switch_mode" "modifies-system"
cli::register_command "show-mode" "Show current operation mode" "resource_cli::show_mode"
cli::register_command "health" "Show detailed health information" "resource_cli::health"
cli::register_command "credentials" "Show n8n credentials for agent-s2" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall agent-s2 (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject automation or configuration into agent-s2
resource_cli::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-agent-s2 inject <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Use existing injection script
    local inject_script="${AGENT_S2_CLI_DIR}/inject.sh"
    if [[ -f "$inject_script" ]]; then
        "$inject_script" "$file"
    else
        log::error "agent-s2 injection script not available"
        return 1
    fi
}

# Validate agent-s2 configuration
resource_cli::validate() {
    if command -v agents2::is_healthy &>/dev/null; then
        agents2::is_healthy
    else
        # Basic validation
        log::header "Validating agent-s2"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "${AGENTS2_CONTAINER_NAME:-agent-s2}" || {
            log::error "agent-s2 container not running"
            return 1
        }
        log::success "agent-s2 is running"
    fi
}

# Show agent-s2 status
resource_cli::status() {
    if command -v agents2::show_status &>/dev/null; then
        agents2::show_status
    else
        # Basic status
        log::header "agent-s2 Status"
        local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start agent-s2
resource_cli::start() {
    if command -v agents2::docker_start &>/dev/null; then
        agents2::docker_start
    elif command -v agents2::start_in_mode &>/dev/null; then
        agents2::start_in_mode "${MODE:-sandbox}"
    else
        local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
        docker start "$container_name" || log::error "Failed to start agent-s2"
    fi
}

# Stop agent-s2
resource_cli::stop() {
    if command -v agents2::docker_stop &>/dev/null; then
        agents2::docker_stop
    else
        local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
        docker stop "$container_name" || log::error "Failed to stop agent-s2"
    fi
}

# Install agent-s2
resource_cli::install() {
    FORCE="${FORCE:-false}"
    export FORCE
    
    if command -v agents2::install_service &>/dev/null; then
        agents2::install_service
    else
        log::error "agents2::install_service not available"
        return 1
    fi
}

# Uninstall agent-s2
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    YES="${FORCE}"  # Set YES to FORCE for uninstall prompts
    export FORCE YES
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove agent-s2 and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v agents2::uninstall_service &>/dev/null; then
        agents2::uninstall_service
    else
        local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
        docker stop "$container_name" 2>/dev/null || true
        docker rm "$container_name" 2>/dev/null || true
        log::success "agent-s2 uninstalled"
    fi
}

# Show credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "agent-s2"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${AGENTS2_CONTAINER_NAME:-agent-s2}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Build connection object for agent-s2 API
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "${AGENTS2_HOST:-localhost}" \
            --argjson port "${AGENTS2_PORT:-8080}" \
            '{
                host: $host,
                port: $port
            }')
        
        # Build auth object and determine type based on API key availability
        local auth_obj="{}"
        local auth_type="httpRequest"
        if [[ -n "${AGENTS2_API_KEY:-}" ]]; then
            auth_type="httpHeaderAuth"
            auth_obj=$(jq -n \
                --arg header_name "Authorization" \
                --arg header_value "Bearer ${AGENTS2_API_KEY}" \
                '{
                    header_name: $header_name,
                    header_value: $header_value
                }')
        fi
        
        # Build metadata object
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Agent-S2 browser automation and AI control" \
            --arg base_url "${AGENTS2_BASE_URL:-http://localhost:8080}" \
            '{
                description: $description,
                base_url: $base_url
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "Agent-S2 Automation API" \
            "$auth_type" \
            "$connection_obj" \
            "$auth_obj" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    local response
    response=$(credentials::build_response "agent-s2" "$status" "$connections_array")
    credentials::format_output "$response"
}

# Take screenshot
resource_cli::screenshot() {
    local output_file="${1:-screenshot.png}"
    
    if command -v agents2::api_request &>/dev/null; then
        local response
        response=$(agents2::api_request "GET" "/screenshot")
        if [[ $? -eq 0 ]]; then
            echo "$response" | base64 -d > "$output_file"
            log::success "Screenshot saved to: $output_file"
        else
            log::error "Failed to take screenshot"
            return 1
        fi
    else
        log::error "Screenshot functionality not available"
        return 1
    fi
}

# Execute automation task
resource_cli::execute() {
    local task="${1:-}"
    
    if [[ -z "$task" ]]; then
        log::error "Task required for execution"
        echo "Usage: resource-agent-s2 execute '<task description>'"
        echo ""
        echo "Examples:"
        echo "  resource-agent-s2 execute \"Click the login button\""
        echo "  resource-agent-s2 execute \"Fill out the contact form with test data\""
        echo "  resource-agent-s2 execute \"Navigate to example.com and take a screenshot\""
        return 1
    fi
    
    if command -v agents2::api_request &>/dev/null; then
        local payload
        payload=$(jq -n --arg task "$task" '{task: $task}')
        agents2::api_request "POST" "/execute" "$payload"
    else
        log::error "Execute functionality not available"
        return 1
    fi
}

# Switch operation mode
resource_cli::switch_mode() {
    local mode="${1:-}"
    
    if [[ -z "$mode" ]]; then
        log::error "Mode required (sandbox|host)"
        echo "Usage: resource-agent-s2 switch-mode <sandbox|host>"
        echo ""
        echo "Modes:"
        echo "  sandbox - Safe mode with limited system access"
        echo "  host    - Full system access mode (use with caution)"
        return 1
    fi
    
    if command -v agents2::switch_mode &>/dev/null; then
        agents2::switch_mode "$mode"
    elif command -v agents2::api_request &>/dev/null; then
        local payload
        payload=$(jq -n --arg mode "$mode" '{mode: $mode}')
        agents2::api_request "POST" "/modes/switch" "$payload"
    else
        log::error "Mode switching not available"
        return 1
    fi
}

# Show current mode
resource_cli::show_mode() {
    if command -v agents2::api_request &>/dev/null; then
        agents2::api_request "GET" "/modes/current"
    else
        log::error "Mode query not available"
        return 1
    fi
}

# Show health information
resource_cli::health() {
    if command -v agents2::api_request &>/dev/null; then
        agents2::api_request "GET" "/health"
    else
        log::error "Health check not available"
        return 1
    fi
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi