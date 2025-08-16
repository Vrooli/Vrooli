#!/usr/bin/env bash
set -euo pipefail

# QuestDB Time-Series Database Management Script
# This script handles installation, configuration, and management of QuestDB

# shellcheck disable=SC2034
DESCRIPTION="Install and manage QuestDB time-series database for high-performance analytics"

QUESTDB_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${QUESTDB_SCRIPT_DIR}/../.."

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../lib/utils/args-cli.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/lib/docker-utils.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

# Source configuration
# shellcheck disable=SC1091
source "${QUESTDB_SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${QUESTDB_SCRIPT_DIR}/config/messages.sh"

# Export configuration
questdb::export_config
questdb::messages::init

# Source all library modules
# shellcheck disable=SC1091
source "${QUESTDB_SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${QUESTDB_SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${QUESTDB_SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${QUESTDB_SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${QUESTDB_SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${QUESTDB_SCRIPT_DIR}/lib/tables.sh"

#######################################
# Parse command line arguments
#######################################
questdb::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    # Define actions
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform (install, uninstall, start, stop, restart, status, logs, monitor, backup, restore, test, shell, query, create-table, drop-table, list-tables, export, import, reset)" \
        --type "value" \
        --default "status"

    # Additional options
    args::register \
        --name "query" \
        --flag "q" \
        --desc "SQL query to execute" \
        --type "value" \
        --default ""
    
    args::register \
        --name "table" \
        --flag "t" \
        --desc "Table name for operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "schema" \
        --flag "s" \
        --desc "Schema file for table creation" \
        --type "value" \
        --default ""
    
    args::register \
        --name "limit" \
        --flag "l" \
        --desc "Limit query results" \
        --type "value" \
        --default "100"
    
    args::register \
        --name "tail" \
        --flag "f" \
        --desc "Follow logs in real-time" \
        --type "value" \
        --default "false"
    
    # Parse arguments
    if ! args::parse "$@"; then
        args::usage "$@"
        return 1
    fi
    
    # Show help if requested
    if args::is_help_set; then
        args::usage "$@"
        return 0
    fi
    
    # Get action
    local action
    action=$(args::get_option_value action)
    
    if [[ -z "$action" ]]; then
        echo_error "No action specified"
        args::usage "$@"
        return 1
    fi
    
    # Route to appropriate handler
    case "$action" in
        install)
            questdb::install::run
            ;;
        uninstall)
            questdb::uninstall
            ;;
        start)
            questdb::docker::start
            ;;
        stop)
            questdb::docker::stop
            ;;
        restart)
            questdb::docker::restart
            ;;
        status)
            questdb::status::check
            ;;
        logs)
            local tail_logs
            tail_logs=$(args::is_option_set tail && echo "true" || echo "false")
            questdb::docker::logs "$tail_logs"
            ;;
        info)
            questdb::info
            ;;
        test)
            questdb::test
            ;;
        query)
            local query
            query=$(args::get_option_value query)
            if [[ -z "$query" ]]; then
                echo_error "No query specified. Use --query 'SQL query'"
                return 1
            fi
            questdb::api::query "$query"
            ;;
        tables)
            local table schema
            table=$(args::get_option_value table)
            schema=$(args::get_option_value schema)
            
            if [[ -n "$table" && -n "$schema" ]]; then
                questdb::tables::create_from_file "$table" "$schema"
            else
                questdb::tables::list
            fi
            ;;
        api)
            echo_info "Use --query for SQL queries or see API documentation"
            echo_info "${QUESTDB_INFO_MESSAGES["api_docs"]}"
            ;;
        console)
            questdb::open_console
            ;;
        *)
            echo_error "Unknown action: $action"
            args::usage "$@"
            return 1
            ;;
    esac
}

#######################################
# Uninstall QuestDB
#######################################
questdb::uninstall() {
    echo_header "Uninstalling QuestDB"
    
    if ! args::prompt_yes_no "Are you sure you want to uninstall QuestDB? This will remove all data." "n"; then
        echo_info "Uninstall cancelled"
        return 0
    fi
    
    # Stop container if running
    questdb::docker::stop || true
    
    # Remove container
    if docker ps -a --format '{{.Names}}' | grep -q "^${QUESTDB_CONTAINER_NAME}$"; then
        echo_info "Removing QuestDB container..."
        docker rm -f "${QUESTDB_CONTAINER_NAME}" || true
    fi
    
    # Remove network
    if docker network ls --format '{{.Name}}' | grep -q "^${QUESTDB_NETWORK_NAME}$"; then
        echo_info "Removing Docker network..."
        docker network rm "${QUESTDB_NETWORK_NAME}" || true
    fi
    
    # Remove data directories
    if args::prompt_yes_no "Remove all QuestDB data directories?" "n"; then
        echo_info "Removing data directories..."
        trash::safe_remove "${QUESTDB_DATA_DIR}" --no-confirm 2>/dev/null || true
        trash::safe_remove "${QUESTDB_CONFIG_DIR}" --no-confirm 2>/dev/null || true
        trash::safe_remove "${QUESTDB_LOG_DIR}" --no-confirm 2>/dev/null || true
        trash::safe_remove "${HOME}/.questdb" --no-confirm 2>/dev/null || true
    fi
    
    echo_success "QuestDB uninstalled successfully"
}

#######################################
# Open QuestDB web console
#######################################
questdb::open_console() {
    if ! questdb::docker::is_running; then
        echo_error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    echo_info "${QUESTDB_INFO_MESSAGES["web_console"]}"
    
    # Try to open in browser
    if command -v xdg-open &> /dev/null; then
        xdg-open "${QUESTDB_BASE_URL}"
    elif command -v open &> /dev/null; then
        open "${QUESTDB_BASE_URL}"
    else
        echo_info "Please open your browser and navigate to: ${QUESTDB_BASE_URL}"
    fi
}

# Run the script
questdb::parse_arguments "$@"