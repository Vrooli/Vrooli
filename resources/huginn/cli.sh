#!/usr/bin/env bash
################################################################################
# Huginn Resource CLI - v2.0 Universal Contract Compliant
# 
# Agent-based workflow automation platform with web scraping and data monitoring
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    HUGINN_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${HUGINN_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
HUGINN_CLI_DIR="${APP_ROOT}/resources/huginn"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${HUGINN_CLI_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${HUGINN_CLI_DIR}/config/messages.sh"

# Source Huginn libraries
for lib in common docker install status api inject test testing; do
    lib_file="${HUGINN_CLI_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

cli::init "huginn" "Huginn agent-based workflow automation platform" "v2"
CLI_COMMAND_HANDLERS["manage::install"]="huginn::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="huginn::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="huginn::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="huginn::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="huginn::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="huginn::test_smoke"
CLI_COMMAND_HANDLERS["test::integration"]="huginn::test_integration"
CLI_COMMAND_HANDLERS["test::unit"]="huginn::test_unit"
CLI_COMMAND_HANDLERS["test::all"]="huginn::test_all"

CLI_COMMAND_HANDLERS["content::add"]="huginn::import_scenario_file"
CLI_COMMAND_HANDLERS["content::list"]="huginn::list_agents"
CLI_COMMAND_HANDLERS["content::get"]="huginn::show_agent"
CLI_COMMAND_HANDLERS["content::remove"]="huginn_content_remove"
CLI_COMMAND_HANDLERS["content::execute"]="huginn::run_agent"

cli::register_command "status" "Show detailed resource status" "huginn::status"
cli::register_command "logs" "Show Huginn logs" "huginn::view_logs"
cli::register_command "credentials" "Show Huginn credentials for integration" "huginn_credentials"

cli::register_subcommand "content" "agents" "List all agents" "huginn::list_agents"
cli::register_subcommand "content" "scenarios" "List all scenarios" "huginn::list_scenarios"
cli::register_subcommand "content" "events" "Show recent events" "huginn::show_recent_events"

huginn_content_remove() {
    log::error "Agent/scenario removal not yet implemented"
    log::info "Use the Huginn web interface to remove agents or scenarios"
    return 1
}

huginn_credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "huginn"; return 0; }
        return 1
    fi
    local status=$(credentials::get_resource_status "${CONTAINER_NAME:-huginn}")
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connection_obj=$(jq -n --arg host "localhost" --argjson port "${HUGINN_PORT:-3000}" --arg path "/api" --argjson ssl false '{host: $host, port: $port, path: $path, ssl: $ssl}')
        local auth_obj=$(jq -n --arg username "${DEFAULT_ADMIN_USERNAME:-admin}" --arg password "changeme" '{username: $username, password: $password}')
        local metadata_obj=$(jq -n --arg description "Huginn agent-based workflow automation platform" --arg web_ui "${HUGINN_BASE_URL:-http://localhost:3000}" --arg setup_note "Default password should be changed after first login" '{description: $description, web_ui_url: $web_ui, setup_note: $setup_note}')
        local connection=$(credentials::build_connection "main" "Huginn API" "httpBasicAuth" "$connection_obj" "$auth_obj" "$metadata_obj")
        connections_array="[$connection]"
    fi
    local response=$(credentials::build_response "huginn" "$status" "$connections_array")
    credentials::format_output "$response"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi