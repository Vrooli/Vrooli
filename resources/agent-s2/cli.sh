#!/usr/bin/env bash
################################################################################
# Agent-S2 Resource CLI - v2.0 Universal Contract Compliant
# 
# Autonomous computer interaction service with GUI automation and AI control
#
# Usage:
#   resource-agent-s2 <command> [options]
#   resource-agent-s2 <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    AGENT_S2_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${AGENT_S2_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
AGENT_S2_CLI_DIR="${APP_ROOT}/resources/agent-s2"

# Source standard variables and utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
source "${AGENT_S2_CLI_DIR}/config/defaults.sh"

# Source agent-s2 libraries
for lib in common docker install status api usage modes stealth; do
    lib_file="${AGENT_S2_CLI_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

# Initialize CLI framework in v2.0 mode
cli::init "agent-s2" "Agent-S2 browser automation and AI control management" "v2"

# ==============================================================================
# REQUIRED HANDLERS - Direct mapping for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="agents2::install_service"
CLI_COMMAND_HANDLERS["manage::uninstall"]="agents2::uninstall_service"
CLI_COMMAND_HANDLERS["manage::start"]="agents2::docker_start"
CLI_COMMAND_HANDLERS["manage::stop"]="agents2::docker_stop"
CLI_COMMAND_HANDLERS["manage::restart"]="agents2::docker_restart"
CLI_COMMAND_HANDLERS["test::smoke"]="agents2::is_healthy"

# Content handlers - Map execute to AI task execution
CLI_COMMAND_HANDLERS["content::execute"]="agents2::execute_ai_task"

# Custom content subcommands for Agent-S2 operations
cli::register_subcommand "content" "screenshot" "Take a screenshot" "agents2::test_screenshot"
cli::register_subcommand "content" "automation" "Run automation examples" "agents2::test_automation_sequence"
cli::register_subcommand "content" "inject" "Inject automation/config" "agents2_inject" "modifies-system"
cli::register_subcommand "content" "usage" "Show usage examples" "agents2::run_usage_example"

# Custom test subcommands
cli::register_subcommand "test" "automation" "Test automation capabilities" "agents2::test_automation_sequence"
cli::register_subcommand "test" "api" "Test API functionality" "agents2::api_test_menu"
cli::register_subcommand "test" "stealth" "Test stealth mode" "agents2::test_stealth"

# Mode management
cli::register_subcommand "manage" "switch-mode" "Switch operating mode" "agents2::switch_mode" "modifies-system"

# Session management - stub implementations
cli::register_subcommand "manage" "reset-session" "Reset session data" "agents2_reset_session" "modifies-system"
cli::register_subcommand "manage" "export-session" "Export session data" "agents2_export_session"
cli::register_subcommand "manage" "import-session" "Import session data" "agents2_import_session" "modifies-system"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "agents2::show_status"
cli::register_command "logs" "Show Agent-S2 logs" "agents2::docker_logs"

# ==============================================================================
# OPTIONAL RESOURCE-SPECIFIC COMMANDS  
# ==============================================================================
cli::register_command "credentials" "Show integration credentials" "agents2_credentials"
cli::register_command "show-mode" "Show current operation mode" "agents2_show_mode"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject automation or configuration  
agents2_inject() {
    local file="${1:-}"
    [[ -z "$file" ]] && { log::error "File path required"; echo "Usage: resource-agent-s2 content inject <file.json>"; return 1; }
    [[ "$file" == shared:* ]] && file="${var_ROOT_DIR}/${file#shared:}"
    [[ ! -f "$file" ]] && { log::error "File not found: $file"; return 1; }
    "${AGENT_S2_CLI_DIR}/inject.sh" "$file"
}

# Show current mode
agents2_show_mode() {
    if command -v agents2::get_current_mode &>/dev/null; then
        local mode=$(agents2::get_current_mode)
        echo "Current mode: $mode"
        agents2::show_mode_info 2>/dev/null || echo "Mode: $mode"
    else
        echo "Mode information not available"
        return 1
    fi
}

# Session management stubs (original functions not found in libraries)
agents2_reset_session() {
    log::warning "Session reset functionality not yet migrated to v2.0"
    log::info "This feature was part of the original manage.sh but functions were not found in library files"
    return 1
}

agents2_export_session() {
    log::warning "Session export functionality not yet migrated to v2.0"
    log::info "This feature was part of the original manage.sh but functions were not found in library files"
    return 1
}

agents2_import_session() {
    log::warning "Session import functionality not yet migrated to v2.0"
    log::info "This feature was part of the original manage.sh but functions were not found in library files"
    return 1
}

# Show credentials for integration
agents2_credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    ! credentials::parse_args "$@" && [[ $? -eq 2 ]] && { credentials::show_help "agent-s2"; return 0; }
    local status=$(credentials::get_resource_status "${AGENTS2_CONTAINER_NAME:-agent-s2}")
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connection_obj=$(jq -n --arg host "${AGENTS2_HOST:-localhost}" --argjson port "${AGENTS2_PORT:-8080}" '{host: $host, port: $port}')
        local auth_obj="{}" auth_type="httpRequest"
        if [[ -n "${AGENTS2_API_KEY:-}" ]]; then
            auth_type="httpHeaderAuth"
            auth_obj=$(jq -n --arg header_name "Authorization" --arg header_value "Bearer ${AGENTS2_API_KEY}" '{header_name: $header_name, header_value: $header_value}')
        fi
        local metadata_obj=$(jq -n --arg description "Agent-S2 browser automation and AI control" --arg base_url "${AGENTS2_BASE_URL:-http://localhost:8080}" '{description: $description, base_url: $base_url}')
        local connection=$(credentials::build_connection "main" "Agent-S2 Automation API" "$auth_type" "$connection_obj" "$auth_obj" "$metadata_obj")
        connections_array="[$connection]"
    fi
    local response=$(credentials::build_response "agent-s2" "$status" "$connections_array")
    credentials::format_output "$response"
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi