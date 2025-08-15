#!/usr/bin/env bash
################################################################################
# Huginn Resource CLI
# 
# Lightweight CLI interface for Huginn using the CLI Command Framework
#
# Usage:
#   resource-huginn <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
HUGINN_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$HUGINN_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$HUGINN_CLI_DIR"
export HUGINN_SCRIPT_DIR="$HUGINN_CLI_DIR"  # For compatibility with existing libs

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

# Source huginn configuration
# shellcheck disable=SC1091
source "${HUGINN_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
huginn::export_config 2>/dev/null || true

# Source huginn libraries
for lib in common docker status api install testing; do
    lib_file="${HUGINN_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "huginn" "Huginn agent-based workflow automation platform"

# Override help to provide Huginn-specific examples
cli::register_command "help" "Show this help message with Huginn examples" "resource_cli::show_help"

# Register additional Huginn-specific commands
cli::register_command "inject" "Inject agents/scenarios into Huginn" "resource_cli::inject" "modifies-system"
cli::register_command "list-agents" "List all agents" "resource_cli::list_agents"
cli::register_command "show-agent" "Show specific agent details" "resource_cli::show_agent"
cli::register_command "run-agent" "Run specific agent" "resource_cli::run_agent" "modifies-system"
cli::register_command "list-scenarios" "List all scenarios" "resource_cli::list_scenarios"
cli::register_command "show-events" "Show recent events" "resource_cli::show_events"
cli::register_command "credentials" "Show n8n credentials for Huginn" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall Huginn (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject agents or scenarios into huginn
resource_cli::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-huginn inject <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-huginn inject agents.json"
        echo "  resource-huginn inject shared:initialization/automation/huginn/agents.json"
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
    
    # Use existing injection function
    if command -v huginn::inject_file &>/dev/null; then
        huginn::inject_file "$file"
    elif command -v huginn::import_scenario_file &>/dev/null; then
        huginn::import_scenario_file "$file"
    else
        log::error "huginn injection functions not available"
        return 1
    fi
}

# Validate huginn configuration
resource_cli::validate() {
    if command -v huginn::health_check &>/dev/null; then
        huginn::health_check
    elif command -v huginn::is_healthy &>/dev/null; then
        if huginn::is_healthy; then
            log::success "Huginn is healthy"
        else
            log::error "Huginn health check failed"
            return 1
        fi
    else
        # Basic validation
        log::header "Validating Huginn"
        if huginn::is_running; then
            log::success "Huginn is running"
        else
            log::error "Huginn is not running"
            return 1
        fi
    fi
}

# Show huginn status
resource_cli::status() {
    if command -v huginn::show_status &>/dev/null; then
        huginn::show_status
    elif command -v huginn::show_basic_status &>/dev/null; then
        huginn::show_basic_status
    else
        # Basic status
        log::header "Huginn Status"
        if huginn::is_running; then
            echo "Container: ✅ Running"
            docker ps --filter "name=huginn" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start huginn
resource_cli::start() {
    if command -v huginn::start &>/dev/null; then
        huginn::start
    else
        log::error "huginn::start not available"
        return 1
    fi
}

# Stop huginn
resource_cli::stop() {
    if command -v huginn::stop &>/dev/null; then
        huginn::stop
    else
        log::error "huginn::stop not available"
        return 1
    fi
}

# Install huginn
resource_cli::install() {
    if command -v huginn::install &>/dev/null; then
        huginn::install
    else
        log::error "huginn::install not available"
        return 1
    fi
}

# Uninstall huginn
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove huginn and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v huginn::uninstall &>/dev/null; then
        huginn::uninstall
    else
        log::error "huginn::uninstall not available"
        return 1
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "huginn"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${CONTAINER_NAME:-huginn}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Huginn web interface - typically uses basic auth
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${HUGINN_PORT:-3000}" \
            --arg path "/api" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local auth_obj
        auth_obj=$(jq -n \
            --arg username "${DEFAULT_ADMIN_USERNAME:-admin}" \
            --arg password "changeme" \
            '{
                username: $username,
                password: $password
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Huginn agent-based workflow automation platform" \
            --arg web_ui "${HUGINN_BASE_URL:-http://localhost:3000}" \
            --arg setup_note "Default password should be changed after first login" \
            '{
                description: $description,
                web_ui_url: $web_ui,
                setup_note: $setup_note
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "Huginn API" \
            "httpBasicAuth" \
            "$connection_obj" \
            "$auth_obj" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    local response
    response=$(credentials::build_response "huginn" "$status" "$connections_array")
    credentials::format_output "$response"
}

# List agents
resource_cli::list_agents() {
    local format="${1:-table}"
    
    if command -v huginn::list_agents &>/dev/null; then
        huginn::list_agents "$format"
    else
        log::error "Agent listing not available"
        return 1
    fi
}

# Show specific agent
resource_cli::show_agent() {
    local agent_id="${1:-}"
    
    if [[ -z "$agent_id" ]]; then
        log::error "Agent ID required"
        echo "Usage: resource-huginn show-agent <agent-id>"
        echo ""
        echo "Examples:"
        echo "  resource-huginn show-agent 1"
        echo "  resource-huginn show-agent 42"
        return 1
    fi
    
    if command -v huginn::show_agent &>/dev/null; then
        huginn::show_agent "$agent_id"
    else
        log::error "Agent display not available"
        return 1
    fi
}

# Run specific agent
resource_cli::run_agent() {
    local agent_id="${1:-}"
    
    if [[ -z "$agent_id" ]]; then
        log::error "Agent ID required"
        echo "Usage: resource-huginn run-agent <agent-id>"
        echo ""
        echo "Examples:"
        echo "  resource-huginn run-agent 1"
        echo "  resource-huginn run-agent 42"
        return 1
    fi
    
    if command -v huginn::run_agent &>/dev/null; then
        huginn::run_agent "$agent_id"
    else
        log::error "Agent execution not available"
        return 1
    fi
}

# List scenarios
resource_cli::list_scenarios() {
    if command -v huginn::list_scenarios &>/dev/null; then
        huginn::list_scenarios
    else
        log::error "Scenario listing not available"
        return 1
    fi
}

# Show recent events
resource_cli::show_events() {
    local count="${1:-10}"
    
    if command -v huginn::show_recent_events &>/dev/null; then
        huginn::show_recent_events "$count"
    else
        log::error "Event display not available"
        return 1
    fi
}

# Custom help function with Huginn-specific examples
resource_cli::show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Huginn-specific examples
    echo ""
    echo "🤖 Huginn Agent-Based Automation Examples:"
    echo ""
    echo "Agent Management:"
    echo "  resource-huginn list-agents                       # List all agents"
    echo "  resource-huginn show-agent 1                      # Show agent details"
    echo "  resource-huginn run-agent 1                       # Execute specific agent"
    echo "  resource-huginn list-scenarios                    # List all scenarios"
    echo ""
    echo "Data & Configuration:"
    echo "  resource-huginn inject agents.json                # Import agents"
    echo "  resource-huginn inject shared:init/huginn/agents.json  # Import shared config"
    echo "  resource-huginn show-events 20                    # Show recent events"
    echo ""
    echo "Monitoring:"
    echo "  resource-huginn status                            # Check service status"
    echo "  resource-huginn credentials                       # Get API credentials"
    echo ""
    echo "Automation Features:"
    echo "  • Agent-based workflow automation"
    echo "  • Web scraping and data monitoring"
    echo "  • API integrations and webhooks"
    echo "  • Event-driven automation chains"
    echo ""
    echo "Default Port: 3000"
    echo "Web UI: http://localhost:3000"
    echo "Default Credentials: admin / changeme (should be changed)"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi