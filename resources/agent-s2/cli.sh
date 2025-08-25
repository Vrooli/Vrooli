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

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    AGENT_S2_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    AGENT_S2_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
AGENT_S2_CLI_DIR="${APP_ROOT}/resources/agent-s2"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

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
cli::register_command "inject" "Inject automation/config into agent-s2" "agents2_inject" "modifies-system"
cli::register_command "screenshot" "Take screenshot" "agents2_screenshot"
cli::register_command "execute" "Execute automation task" "agents2_execute"
cli::register_command "switch-mode" "Switch between sandbox/host mode" "agents2_switch_mode" "modifies-system"
cli::register_command "show-mode" "Show current operation mode" "agents2_show_mode"
cli::register_command "health" "Show detailed health information" "agents2_health"
cli::register_command "credentials" "Show n8n credentials for agent-s2" "agents2_credentials"
cli::register_command "uninstall" "Uninstall agent-s2 (requires --force)" "agents2_uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject automation or configuration into agent-s2
agents2_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-agent-s2 inject <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${var_ROOT_DIR}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    "${AGENT_S2_CLI_DIR}/inject.sh" "$file"
}

# Validate agent-s2 configuration
agents2_uninstall() {
    FORCE="${FORCE:-false}"
    YES="${FORCE}"  # Set YES to FORCE for uninstall prompts
    export FORCE YES
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove agent-s2 and all its data. Use --force to confirm."
        return 1
    fi
    
    agents2::uninstall_service
}

# Show credentials for n8n integration
agents2_credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
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
agents2_screenshot() {
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
agents2_execute() {
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
agents2_switch_mode() {
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
agents2_show_mode() {
    if command -v agents2::api_request &>/dev/null; then
        agents2::api_request "GET" "/modes/current"
    else
        log::error "Mode query not available"
        return 1
    fi
}

# Show health information
agents2_health() {
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
