#!/usr/bin/env bash
################################################################################
# SQLite Resource CLI - v2.0 Universal Contract Compliant
# 
# Lightweight, serverless SQL database engine for local data persistence
#
# Usage:
#   resource-sqlite <command> [options]
#   resource-sqlite <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    SQLITE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${SQLITE_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
SQLITE_CLI_DIR="${APP_ROOT}/resources/sqlite"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${SQLITE_CLI_DIR}/config/defaults.sh"

# Source SQLite libraries
for lib in core test replication webui; do
    lib_file="${SQLITE_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "sqlite" "Lightweight serverless SQL database engine" "v2"

# Register status and logs as flat commands (required by v2.0 contract)
cli::register_command "status" "Show resource status" "cli::_handle_status"
cli::register_command "logs" "Show resource logs" "cli::_handle_logs"

# Register replication group with its subcommands
cli::register_command_group "replicate" "Database replication management"
cli::register_subcommand "replicate" "add" "Add a replica target" "handle_replicate_add"
cli::register_subcommand "replicate" "remove" "Remove a replica" "handle_replicate_remove"
cli::register_subcommand "replicate" "list" "List configured replicas" "handle_replicate_list"
cli::register_subcommand "replicate" "sync" "Sync database to replicas" "handle_replicate_sync"
cli::register_subcommand "replicate" "verify" "Verify replica consistency" "handle_replicate_verify"
cli::register_subcommand "replicate" "toggle" "Enable/disable a replica" "handle_replicate_toggle"
cli::register_subcommand "replicate" "monitor" "Start replication monitor" "handle_replicate_monitor"

# Override default handlers to point directly to SQLite implementations
CLI_COMMAND_HANDLERS["manage::install"]="sqlite::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="sqlite::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="sqlite::start"
CLI_COMMAND_HANDLERS["manage::stop"]="sqlite::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="sqlite::restart"

CLI_COMMAND_HANDLERS["test::smoke"]="sqlite::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="sqlite::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="sqlite::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="sqlite::test::all"

CLI_COMMAND_HANDLERS["content::create"]="sqlite::content::create"
CLI_COMMAND_HANDLERS["content::execute"]="sqlite::content::execute"
CLI_COMMAND_HANDLERS["content::list"]="sqlite::content::list"
CLI_COMMAND_HANDLERS["content::get"]="sqlite::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="sqlite::content::remove"
CLI_COMMAND_HANDLERS["content::backup"]="sqlite::content::backup"
CLI_COMMAND_HANDLERS["content::restore"]="sqlite::content::restore"
CLI_COMMAND_HANDLERS["content::encrypt"]="sqlite::content::encrypt"
CLI_COMMAND_HANDLERS["content::decrypt"]="sqlite::content::decrypt"
CLI_COMMAND_HANDLERS["content::batch"]="sqlite::content::batch"
CLI_COMMAND_HANDLERS["content::import_csv"]="sqlite::content::import_csv"
CLI_COMMAND_HANDLERS["content::export_csv"]="sqlite::content::export_csv"

CLI_COMMAND_HANDLERS["status"]="sqlite::status"
CLI_COMMAND_HANDLERS["logs"]="sqlite::logs"
CLI_COMMAND_HANDLERS["info"]="sqlite::info"

# Register migration commands as a group
cli::register_command_group "migrate" "Database migration management"
cli::register_subcommand "migrate" "init" "Initialize migration tracking" "sqlite::migrate::init"
cli::register_subcommand "migrate" "create" "Create new migration file" "sqlite::migrate::create"
cli::register_subcommand "migrate" "up" "Apply pending migrations" "sqlite::migrate::up"
cli::register_subcommand "migrate" "status" "Show migration status" "sqlite::migrate::status"

# Register query builder commands as a group
cli::register_command_group "query" "Query builder helpers"
cli::register_subcommand "query" "select" "Build and execute SELECT query" "sqlite::query::select"
cli::register_subcommand "query" "insert" "Insert data with automatic escaping" "sqlite::query::insert"
cli::register_subcommand "query" "update" "Update data with conditions" "sqlite::query::update"

# Register stats commands as a group
cli::register_command_group "stats" "Performance monitoring and analysis"
cli::register_subcommand "stats" "enable" "Enable query statistics" "sqlite::stats::enable"
cli::register_subcommand "stats" "show" "Show query statistics" "sqlite::stats::show"
cli::register_subcommand "stats" "analyze" "Analyze database for optimization" "sqlite::stats::analyze"

# Register webui commands as a group
cli::register_command_group "webui" "Web interface for batch operations"
cli::register_subcommand "webui" "start" "Start web UI server" "sqlite::webui::start"
cli::register_subcommand "webui" "stop" "Stop web UI server" "sqlite::webui::stop"
cli::register_subcommand "webui" "restart" "Restart web UI server" "sqlite::webui::restart"
cli::register_subcommand "webui" "status" "Show web UI status" "sqlite::webui::status"

# Replication command handlers
handle_replicate_add() {
    local db_name=""
    local target=""
    local interval=60
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --database|-d)
                db_name="$2"
                shift 2
                ;;
            --target|-t)
                target="$2"
                shift 2
                ;;
            --interval|-i)
                interval="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$db_name" ]] || [[ -z "$target" ]]; then
        log::error "Usage: replicate add --database <name> --target <path> [--interval <seconds>]"
        return 1
    fi
    
    sqlite::replication::add_replica "$db_name" "$target" "$interval"
}

handle_replicate_remove() {
    local db_name=""
    local target=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --database|-d)
                db_name="$2"
                shift 2
                ;;
            --target|-t)
                target="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$db_name" ]] || [[ -z "$target" ]]; then
        log::error "Usage: replicate remove --database <name> --target <path>"
        return 1
    fi
    
    sqlite::replication::remove_replica "$db_name" "$target"
}

handle_replicate_list() {
    sqlite::replication::list
}

handle_replicate_sync() {
    local db_name=""
    local force=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --database|-d)
                db_name="$2"
                shift 2
                ;;
            --force|-f)
                force=true
                shift
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$db_name" ]]; then
        log::error "Usage: replicate sync --database <name> [--force]"
        return 1
    fi
    
    sqlite::replication::sync "$db_name" "$force"
}

handle_replicate_verify() {
    local db_name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --database|-d)
                db_name="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$db_name" ]]; then
        log::error "Usage: replicate verify --database <name>"
        return 1
    fi
    
    sqlite::replication::verify "$db_name"
}

handle_replicate_toggle() {
    local db_name=""
    local target=""
    local enabled="true"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --database|-d)
                db_name="$2"
                shift 2
                ;;
            --target|-t)
                target="$2"
                shift 2
                ;;
            --enable)
                enabled="true"
                shift
                ;;
            --disable)
                enabled="false"
                shift
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$db_name" ]] || [[ -z "$target" ]]; then
        log::error "Usage: replicate toggle --database <name> --target <path> [--enable|--disable]"
        return 1
    fi
    
    sqlite::replication::toggle "$db_name" "$target" "$enabled"
}

handle_replicate_monitor() {
    local interval=60
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --interval|-i)
                interval="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    sqlite::replication::monitor "$interval"
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi