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

# Source agent management (load config and manager directly)
if [[ -f "${APP_ROOT}/resources/huginn/config/agents.conf" ]]; then
    source "${APP_ROOT}/resources/huginn/config/agents.conf"
    source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
fi
# shellcheck disable=SC1091
source "${HUGINN_CLI_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${HUGINN_CLI_DIR}/config/messages.sh"

# Source Huginn libraries
for lib in common docker install status api inject test testing agents vrooli-integration ollama-integration performance-metrics multi-tenant native-api; do
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

# Additional commands
CLI_COMMAND_HANDLERS["monitor"]="huginn::monitoring_dashboard"
CLI_COMMAND_HANDLERS["backup"]="huginn::backup"
CLI_COMMAND_HANDLERS["restore"]="huginn::restore"

cli::register_command "status" "Show detailed resource status" "huginn::status"
cli::register_command "logs" "Show Huginn logs" "huginn::view_logs"
cli::register_command "credentials" "Show Huginn credentials for integration" "huginn_credentials"
cli::register_command "monitor" "Show real-time monitoring dashboard" "huginn::monitoring_dashboard"
cli::register_command "backup" "Backup Huginn data and configuration" "huginn::backup"
cli::register_command "restore" "Restore Huginn from backup" "huginn::restore"
# Create wrapper for agents command that delegates to manager
huginn::agents::command() {
    if type -t agent_manager::load_config &>/dev/null; then
        "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" --config="huginn" "$@"
    else
        log::error "Agent management not available"
        return 1
    fi
}
export -f huginn::agents::command

cli::register_command "agents" "Manage running huginn agents" "huginn::agents::command"
cli::register_command "export" "Export scenarios or agents to JSON" "huginn::export_command"
cli::register_command "import" "Import scenarios from JSON file" "huginn::import_scenario_file"
cli::register_command "vrooli" "Vrooli integration management" "huginn::vrooli_command"
cli::register_command "ollama" "AI-powered event filtering with Ollama" "huginn::ollama_command"
cli::register_command "performance" "Performance metrics and analysis" "huginn::performance_command"
cli::register_command "tenant" "Multi-tenant workspace management" "huginn::tenant_command"
cli::register_command "api" "Native API endpoints" "huginn::api_command"

cli::register_subcommand "content" "agents" "List all agents" "huginn::list_agents"
cli::register_subcommand "content" "scenarios" "List all scenarios" "huginn::list_scenarios"
cli::register_subcommand "content" "events" "Show recent events" "huginn::show_recent_events"

cli::register_subcommand "export" "scenario" "Export scenario to JSON" "huginn::export_scenario"
cli::register_subcommand "export" "agents" "Export agents to JSON" "huginn::export_agents"

cli::register_subcommand "vrooli" "init" "Initialize Vrooli integration" "huginn::vrooli_init"
cli::register_subcommand "vrooli" "test" "Test Vrooli integration" "huginn::test_vrooli_integration"
cli::register_subcommand "vrooli" "setup-redis" "Setup Redis event listener" "huginn::setup_redis_listener"
cli::register_subcommand "vrooli" "setup-minio" "Setup MinIO storage agent" "huginn::setup_minio_storage"
cli::register_subcommand "vrooli" "publish-events" "Publish agent events to Redis" "huginn::publish_agent_events"

cli::register_subcommand "ollama" "test" "Test Ollama integration" "huginn::ollama_test"
cli::register_subcommand "ollama" "create-filter" "Create AI-powered filter agent" "huginn::ollama_create_filter"
cli::register_subcommand "ollama" "list-filters" "List AI filter agents" "huginn::ollama_list_filters"
cli::register_subcommand "ollama" "process" "Process events through AI filter" "huginn::ollama_process"
cli::register_subcommand "ollama" "analyze" "Analyze event with AI" "huginn::ollama_analyze"

cli::register_subcommand "performance" "dashboard" "Live performance dashboard" "huginn::performance_dashboard"
cli::register_subcommand "performance" "metrics" "Get current performance metrics" "huginn::performance_metrics"
cli::register_subcommand "performance" "analyze" "Analyze agent performance" "huginn::performance_analyze"
cli::register_subcommand "performance" "export" "Export performance data" "huginn::performance_export"
cli::register_subcommand "performance" "create-monitor" "Create performance monitoring agent" "huginn::performance_create_monitor"

cli::register_subcommand "tenant" "create" "Create new tenant workspace" "huginn::tenant_create"
cli::register_subcommand "tenant" "list" "List all tenants" "huginn::tenant_list"
cli::register_subcommand "tenant" "get" "Get tenant details" "huginn::tenant_get"
cli::register_subcommand "tenant" "delete" "Delete tenant workspace" "huginn::tenant_delete"
cli::register_subcommand "tenant" "update" "Update tenant settings" "huginn::tenant_update"
cli::register_subcommand "tenant" "export" "Export tenant data" "huginn::tenant_export"
cli::register_subcommand "tenant" "import" "Import tenant data" "huginn::tenant_import"
cli::register_subcommand "tenant" "isolate" "Ensure tenant isolation" "huginn::tenant_isolate"
cli::register_subcommand "tenant" "quota" "Check tenant quotas" "huginn::tenant_quota"
cli::register_subcommand "tenant" "stats" "Multi-tenant statistics" "huginn::tenant_stats"

cli::register_subcommand "api" "health" "Check API health" "huginn::api_health"
cli::register_subcommand "api" "status" "Get API status" "huginn::api_status"
cli::register_subcommand "api" "agents" "Agent operations" "huginn::api_agents"
cli::register_subcommand "api" "events" "Event operations" "huginn::api_events"
cli::register_subcommand "api" "scenarios" "Scenario operations" "huginn::api_scenarios"
cli::register_subcommand "api" "webhooks" "Webhook operations" "huginn::api_webhooks"
cli::register_subcommand "api" "users" "User operations" "huginn::api_users"

huginn_content_remove() {
    log::error "Agent/scenario removal not yet implemented"
    log::info "Use the Huginn web interface to remove agents or scenarios"
    return 1
}

huginn::export_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        scenario)
            huginn::export_scenario "$@"
            ;;
        agents)
            huginn::export_agents "$@"
            ;;
        *)
            log::error "Unknown export subcommand: $subcommand"
            log::info "Usage: huginn export [scenario|agents] [id] [output-file]"
            log::info "  scenario ID - Export specific scenario by ID"
            log::info "  scenario all - Export all scenarios"
            log::info "  agents ID,ID - Export specific agents by comma-separated IDs"
            log::info "  agents all - Export all agents"
            return 1
            ;;
    esac
}

huginn::vrooli_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        init)
            huginn::vrooli_init
            ;;
        test)
            huginn::test_vrooli_integration
            ;;
        setup-redis)
            huginn::setup_redis_listener
            ;;
        setup-minio)
            huginn::setup_minio_storage
            ;;
        publish-events)
            huginn::publish_agent_events
            ;;
        *)
            log::error "Unknown vrooli subcommand: $subcommand"
            log::info "Available subcommands:"
            log::info "  init           - Initialize Vrooli integration"
            log::info "  test           - Test integration connections"
            log::info "  setup-redis    - Setup Redis event listener agent"
            log::info "  setup-minio    - Setup MinIO storage agent"
            log::info "  publish-events - Publish recent events to Redis"
            return 1
            ;;
    esac
}

huginn::ollama_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        test)
            huginn::ollama_test
            ;;
        create-filter)
            huginn::ollama_create_filter "$@"
            ;;
        list-filters)
            huginn::ollama_list_filters
            ;;
        process)
            huginn::ollama_process "$@"
            ;;
        analyze)
            huginn::ollama_analyze "$@"
            ;;
        *)
            log::error "Unknown ollama subcommand: $subcommand"
            log::info "Available subcommands:"
            log::info "  test          - Test Ollama integration"
            log::info "  create-filter - Create AI-powered filter agent"
            log::info "  list-filters  - List AI filter agents"
            log::info "  process       - Process events through AI filter"
            log::info "  analyze       - Analyze event with AI"
            return 1
            ;;
    esac
}

# Wrapper functions for Ollama integration
huginn::ollama_test() {
    source "${HUGINN_CLI_DIR}/lib/ollama-integration.sh"
    test_ollama_integration
}

huginn::ollama_create_filter() {
    source "${HUGINN_CLI_DIR}/lib/ollama-integration.sh"
    create_ai_filter_agent "$@"
}

huginn::ollama_list_filters() {
    source "${HUGINN_CLI_DIR}/lib/ollama-integration.sh"
    list_ai_filter_agents
}

huginn::ollama_process() {
    source "${HUGINN_CLI_DIR}/lib/ollama-integration.sh"
    process_ai_filter "$@"
}

huginn::ollama_analyze() {
    source "${HUGINN_CLI_DIR}/lib/ollama-integration.sh"
    ollama_analyze_event "$@"
}

huginn::performance_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        dashboard)
            huginn::performance_dashboard
            ;;
        metrics)
            huginn::performance_metrics "$@"
            ;;
        analyze)
            huginn::performance_analyze "$@"
            ;;
        export)
            huginn::performance_export "$@"
            ;;
        create-monitor)
            huginn::performance_create_monitor "$@"
            ;;
        *)
            log::error "Unknown performance subcommand: $subcommand"
            log::info "Available subcommands:"
            log::info "  dashboard      - Live performance dashboard"
            log::info "  metrics        - Get current performance metrics"
            log::info "  analyze        - Analyze agent performance"
            log::info "  export         - Export performance data"
            log::info "  create-monitor - Create performance monitoring agent"
            return 1
            ;;
    esac
}

# Wrapper functions for Performance integration
huginn::performance_dashboard() {
    source "${HUGINN_CLI_DIR}/lib/performance-metrics-simple.sh"
    show_performance_summary
}

huginn::performance_metrics() {
    source "${HUGINN_CLI_DIR}/lib/performance-metrics-simple.sh"
    get_basic_metrics | jq .
}

huginn::performance_analyze() {
    log::info "Agent performance analysis not yet implemented"
    log::info "Use 'huginn performance metrics' for basic metrics"
}

huginn::performance_export() {
    source "${HUGINN_CLI_DIR}/lib/performance-metrics-simple.sh"
    export_metrics "$@"
}

huginn::performance_create_monitor() {
    log::info "Performance monitor agent creation not yet implemented"
    log::info "Use the web UI to create monitoring agents"
}

huginn::tenant_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        create)
            huginn::tenant_create "$@"
            ;;
        list)
            huginn::tenant_list
            ;;
        get)
            huginn::tenant_get "$@"
            ;;
        delete)
            huginn::tenant_delete "$@"
            ;;
        update)
            huginn::tenant_update "$@"
            ;;
        export)
            huginn::tenant_export "$@"
            ;;
        import)
            huginn::tenant_import "$@"
            ;;
        isolate)
            huginn::tenant_isolate "$@"
            ;;
        quota)
            huginn::tenant_quota "$@"
            ;;
        stats)
            huginn::tenant_stats
            ;;
        *)
            log::error "Unknown tenant subcommand: $subcommand"
            log::info "Available subcommands:"
            log::info "  create  - Create new tenant workspace"
            log::info "  list    - List all tenants"
            log::info "  get     - Get tenant details"
            log::info "  delete  - Delete tenant workspace"
            log::info "  update  - Update tenant settings"
            log::info "  export  - Export tenant data"
            log::info "  import  - Import tenant data"
            log::info "  isolate - Ensure tenant isolation"
            log::info "  quota   - Check tenant quotas"
            log::info "  stats   - Multi-tenant statistics"
            return 1
            ;;
    esac
}

# Wrapper functions for tenant management
huginn::tenant_create() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    create_tenant "$@"
}

huginn::tenant_list() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    list_tenants
}

huginn::tenant_get() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    get_tenant "$@"
}

huginn::tenant_delete() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    delete_tenant "$@"
}

huginn::tenant_update() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    update_tenant "$@"
}

huginn::tenant_export() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    export_tenant "$@"
}

huginn::tenant_import() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    import_tenant "$@"
}

huginn::tenant_isolate() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    isolate_tenant "$@"
}

huginn::tenant_quota() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    quota_status "$@"
}

huginn::tenant_stats() {
    source "${HUGINN_CLI_DIR}/lib/multi-tenant.sh"
    tenant_stats
}

huginn::api_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        health)
            huginn::api_health
            ;;
        status)
            huginn::api_status
            ;;
        agents)
            huginn::api_agents "$@"
            ;;
        events)
            huginn::api_events "$@"
            ;;
        scenarios)
            huginn::api_scenarios "$@"
            ;;
        webhooks)
            huginn::api_webhooks "$@"
            ;;
        users)
            huginn::api_users "$@"
            ;;
        *)
            log::error "Unknown api subcommand: $subcommand"
            log::info "Available subcommands:"
            log::info "  health    - Check API health"
            log::info "  status    - Get detailed status"
            log::info "  agents    - Agent operations (list, get, create, update, delete, run)"
            log::info "  events    - Event operations (list, create)"
            log::info "  scenarios - Scenario operations (list, get)"
            log::info "  webhooks  - Webhook operations (create)"
            log::info "  users     - User operations (list)"
            return 1
            ;;
    esac
}

# API wrapper functions
huginn::api_health() {
    source "${HUGINN_CLI_DIR}/lib/native-api.sh"
    api_health
}

huginn::api_status() {
    source "${HUGINN_CLI_DIR}/lib/native-api.sh"
    api_status
}

huginn::api_agents() {
    source "${HUGINN_CLI_DIR}/lib/native-api.sh"
    local operation="${1:-list}"
    shift || true
    
    case "$operation" in
        list)
            api_agents_list "$@"
            ;;
        get)
            api_agent_get "$@"
            ;;
        create)
            api_agent_create "$@"
            ;;
        update)
            api_agent_update "$@"
            ;;
        delete)
            api_agent_delete "$@"
            ;;
        run)
            api_agent_run "$@"
            ;;
        *)
            log::error "Unknown agent operation: $operation"
            log::info "Available operations: list, get, create, update, delete, run"
            return 1
            ;;
    esac
}

huginn::api_events() {
    source "${HUGINN_CLI_DIR}/lib/native-api.sh"
    local operation="${1:-list}"
    shift || true
    
    case "$operation" in
        list)
            api_events_list "$@"
            ;;
        create)
            api_event_create "$@"
            ;;
        *)
            log::error "Unknown event operation: $operation"
            log::info "Available operations: list, create"
            return 1
            ;;
    esac
}

huginn::api_scenarios() {
    source "${HUGINN_CLI_DIR}/lib/native-api.sh"
    local operation="${1:-list}"
    shift || true
    
    case "$operation" in
        list)
            api_scenarios_list
            ;;
        get)
            api_scenario_get "$@"
            ;;
        *)
            log::error "Unknown scenario operation: $operation"
            log::info "Available operations: list, get"
            return 1
            ;;
    esac
}

huginn::api_webhooks() {
    source "${HUGINN_CLI_DIR}/lib/native-api.sh"
    local operation="${1:-create}"
    shift || true
    
    case "$operation" in
        create)
            api_webhook_create "$@"
            ;;
        *)
            log::error "Unknown webhook operation: $operation"
            log::info "Available operations: create"
            return 1
            ;;
    esac
}

huginn::api_users() {
    source "${HUGINN_CLI_DIR}/lib/native-api.sh"
    api_users_list
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