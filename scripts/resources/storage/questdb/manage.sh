#!/usr/bin/env bash
set -euo pipefail

# QuestDB Time-Series Database Management Script
# This script handles installation, configuration, and management of QuestDB

DESCRIPTION="Install and manage QuestDB time-series database for high-performance analytics"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Source configuration
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration
questdb::export_config
questdb::messages::init

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/tables.sh"

#######################################
# Parse command line arguments
#######################################
questdb::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    # Define actions
    args::define_option_with_value action "" "action" "
        Action to perform:
        - install:     Install QuestDB
        - uninstall:   Remove QuestDB
        - start:       Start QuestDB container
        - stop:        Stop QuestDB container
        - restart:     Restart QuestDB container
        - status:      Check QuestDB status
        - logs:        View QuestDB logs
        - query:       Execute SQL query
        - tables:      List or create tables
        - api:         Make API request
        - console:     Open web console
    " false

    # Additional options
    args::define_option_with_value query "" "query,q" "SQL query to execute" false
    args::define_option_with_value table "" "table,t" "Table name for operations" false
    args::define_option_with_value schema "" "schema,s" "Schema file for table creation" false
    args::define_option_with_value limit "100" "limit,l" "Limit query results" false
    args::define_option tail "tail,f" "Follow logs in real-time" false
    
    # Parse arguments
    if ! args::parse "$@"; then
        args::usage
        return 1
    fi
    
    # Show help if requested
    if args::is_help_set; then
        args::usage
        return 0
    fi
    
    # Get action
    local action
    action=$(args::get_option_value action)
    
    if [[ -z "$action" ]]; then
        echo_error "No action specified"
        args::usage
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
            args::usage
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
        rm -rf "${QUESTDB_DATA_DIR}" "${QUESTDB_CONFIG_DIR}" "${QUESTDB_LOG_DIR}"
        rm -rf "${HOME}/.questdb"
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