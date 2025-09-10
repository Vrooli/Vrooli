#!/usr/bin/env bash
# Matrix Synapse Resource - CLI Interface
# v2.0 Contract Compliant Implementation

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$SCRIPT_DIR"

# Load libraries
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Resource metadata
readonly RESOURCE_NAME="matrix-synapse"
readonly RESOURCE_VERSION="2.0.0"

# Help function
show_help() {
    cat <<EOF
Matrix Synapse Resource - Federated Communication Server

USAGE:
    resource-${RESOURCE_NAME} <command> [options]

COMMANDS:
    help                    Show this help message
    info                    Show resource information
    manage <subcommand>     Manage resource lifecycle
    test <subcommand>       Run tests
    content <subcommand>    Manage Matrix content
    status                  Show current status
    logs                    View resource logs

MANAGE SUBCOMMANDS:
    install                 Install Matrix Synapse
    start                   Start the service
    stop                    Stop the service
    restart                 Restart the service
    uninstall              Remove Matrix Synapse

TEST SUBCOMMANDS:
    smoke                   Quick health check (<30s)
    integration            Full functionality test (<120s)
    unit                    Library function tests (<60s)
    all                     Run all tests

CONTENT SUBCOMMANDS:
    add-user <username> [password]    Create a new user
    list-users                         List all users
    create-room <name>                 Create a new room
    send-message <room> <message>     Send a message to a room
    add-bot <name>                     Create a bot user

EXAMPLES:
    # Install and start Matrix Synapse
    resource-${RESOURCE_NAME} manage install
    resource-${RESOURCE_NAME} manage start

    # Create a user
    resource-${RESOURCE_NAME} content add-user alice

    # Check status
    resource-${RESOURCE_NAME} status

    # View logs
    resource-${RESOURCE_NAME} logs --tail 50

CONFIGURATION:
    Required environment variables:
    - MATRIX_SYNAPSE_SERVER_NAME: Your server domain
    - SYNAPSE_DB_PASSWORD: PostgreSQL password
    - SYNAPSE_REGISTRATION_SECRET: User registration secret

    Optional:
    - MATRIX_SYNAPSE_PORT: API port (default: 8008)
    - MATRIX_SYNAPSE_FEDERATION_ENABLED: Enable federation (default: false)

EOF
    exit 0
}

# Info function
show_info() {
    local json_output="${1:-false}"
    
    if [[ "$json_output" == "true" ]] || [[ "$json_output" == "--json" ]]; then
        cat "${RESOURCE_DIR}/config/runtime.json"
    else
        echo "Resource: ${RESOURCE_NAME}"
        echo "Version: ${RESOURCE_VERSION}"
        echo "Status: $(get_status)"
        echo "Port: ${MATRIX_SYNAPSE_PORT}"
        echo "Server Name: ${MATRIX_SYNAPSE_SERVER_NAME}"
        echo "Data Directory: ${MATRIX_SYNAPSE_DATA_DIR}"
        echo ""
        echo "Dependencies:"
        echo "  - postgres (required)"
        echo "  - redis (optional)"
        echo ""
        echo "For JSON output, use: resource-${RESOURCE_NAME} info --json"
    fi
}

# Manage subcommands
manage_resource() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            log_info "Installing Matrix Synapse..."
            ensure_directories
            
            # Check dependencies
            if ! command -v python3 &>/dev/null; then
                log_error "Python 3 is required but not installed"
                exit 1
            fi
            
            # Install Synapse via pip
            if [[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "pip" ]]; then
                log_info "Installing via pip..."
                python3 -m pip install --user matrix-synapse
            else
                log_info "Installing via Docker..."
                docker pull matrixdotorg/synapse:latest
            fi
            
            # Generate configuration
            generate_config
            
            # Initialize database
            init_database
            
            # Generate signing key
            if [[ ! -f "${MATRIX_SYNAPSE_CONFIG_DIR}/signing.key" ]]; then
                python3 -m synapse.app.homeserver \
                    --generate-keys \
                    -c "${MATRIX_SYNAPSE_CONFIG_DIR}/homeserver.yaml" 2>/dev/null || true
            fi
            
            log_info "Matrix Synapse installed successfully"
            ;;
            
        start)
            if is_running; then
                log_info "Matrix Synapse is already running"
                exit 0
            fi
            
            log_info "Starting Matrix Synapse..."
            ensure_directories
            
            # Start Synapse
            if [[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "pip" ]]; then
                python3 -m synapse.app.homeserver \
                    -c "${MATRIX_SYNAPSE_CONFIG_DIR}/homeserver.yaml" \
                    --daemonize \
                    --pid-file="${MATRIX_SYNAPSE_PID_FILE}"
            else
                docker run -d \
                    --name matrix-synapse \
                    -v "${MATRIX_SYNAPSE_CONFIG_DIR}:/data" \
                    -p "${MATRIX_SYNAPSE_PORT}:8008" \
                    matrixdotorg/synapse:latest
            fi
            
            # Wait for startup
            if wait_for_ready; then
                log_info "Matrix Synapse started successfully"
                
                # Create admin user if first start
                if ! grep -q "admin_created" "${MATRIX_SYNAPSE_DATA_DIR}/.initialized" 2>/dev/null; then
                    create_user "admin" "admin123" true &>/dev/null || true
                    echo "admin_created" >> "${MATRIX_SYNAPSE_DATA_DIR}/.initialized"
                    log_info "Admin user created (username: admin, password: admin123)"
                fi
            else
                log_error "Failed to start Matrix Synapse"
                exit 1
            fi
            ;;
            
        stop)
            if ! is_running; then
                log_info "Matrix Synapse is not running"
                exit 0
            fi
            
            log_info "Stopping Matrix Synapse..."
            
            if [[ -f "${MATRIX_SYNAPSE_PID_FILE}" ]]; then
                local pid
                pid=$(cat "${MATRIX_SYNAPSE_PID_FILE}")
                kill "$pid" 2>/dev/null || true
                
                # Wait for shutdown
                local timeout=30
                while [[ $timeout -gt 0 ]] && kill -0 "$pid" 2>/dev/null; do
                    sleep 1
                    ((timeout--))
                done
                
                rm -f "${MATRIX_SYNAPSE_PID_FILE}"
            fi
            
            if [[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "docker" ]]; then
                docker stop matrix-synapse 2>/dev/null || true
                docker rm matrix-synapse 2>/dev/null || true
            fi
            
            log_info "Matrix Synapse stopped"
            ;;
            
        restart)
            "$0" manage stop
            sleep 2
            "$0" manage start
            ;;
            
        uninstall)
            "$0" manage stop
            
            log_info "Uninstalling Matrix Synapse..."
            
            if [[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "pip" ]]; then
                python3 -m pip uninstall -y matrix-synapse
            else
                docker rmi matrixdotorg/synapse:latest 2>/dev/null || true
            fi
            
            # Optional: Remove data
            read -p "Remove all data? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                rm -rf "${MATRIX_SYNAPSE_DATA_DIR}"
                log_info "Data removed"
            fi
            
            log_info "Matrix Synapse uninstalled"
            ;;
            
        *)
            echo "Unknown manage subcommand: $subcommand"
            echo "Use 'resource-${RESOURCE_NAME} help' for usage"
            exit 1
            ;;
    esac
}

# Test subcommands
test_resource() {
    local subcommand="${1:-all}"
    
    case "$subcommand" in
        smoke)
            test_smoke
            ;;
        integration)
            test_integration
            ;;
        unit)
            test_unit
            ;;
        all)
            test_all
            ;;
        *)
            echo "Unknown test subcommand: $subcommand"
            exit 1
            ;;
    esac
}

# Content management
content_resource() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add-user|add)
            local username="${1:-}"
            local password="${2:-}"
            if [[ -z "$username" ]]; then
                echo "Usage: resource-${RESOURCE_NAME} content add-user <username> [password]"
                exit 1
            fi
            create_user "$username" "$password" false
            ;;
            
        add-bot)
            local botname="${1:-}"
            if [[ -z "$botname" ]]; then
                echo "Usage: resource-${RESOURCE_NAME} content add-bot <name>"
                exit 1
            fi
            create_user "$botname" "$(openssl rand -base64 32)" false
            ;;
            
        list-users|list)
            curl -sf "http://localhost:${MATRIX_SYNAPSE_PORT}/_synapse/admin/v2/users" 2>/dev/null | jq -r '.users[].name' 2>/dev/null || echo "Failed to list users"
            ;;
            
        create-room)
            local room_name="${1:-}"
            if [[ -z "$room_name" ]]; then
                echo "Usage: resource-${RESOURCE_NAME} content create-room <name>"
                exit 1
            fi
            # This would require an access token - simplified for scaffold
            echo "Room creation requires authentication - use the API directly"
            ;;
            
        send-message)
            local room="${1:-}"
            local message="${2:-}"
            if [[ -z "$room" ]] || [[ -z "$message" ]]; then
                echo "Usage: resource-${RESOURCE_NAME} content send-message <room> <message>"
                exit 1
            fi
            # This would require an access token - simplified for scaffold
            echo "Message sending requires authentication - use the API directly"
            ;;
            
        *)
            echo "Unknown content subcommand: $subcommand"
            echo "Available: add-user, list-users, create-room, send-message, add-bot"
            exit 1
            ;;
    esac
}

# Status command
show_status() {
    local status
    status=$(get_status)
    
    echo "Matrix Synapse Status: $status"
    
    if [[ "$status" == "running" ]] || [[ "$status" == "healthy" ]]; then
        echo "  PID: $(cat "${MATRIX_SYNAPSE_PID_FILE}" 2>/dev/null || echo "unknown")"
        echo "  Port: ${MATRIX_SYNAPSE_PORT}"
        echo "  Server: ${MATRIX_SYNAPSE_SERVER_NAME}"
        echo "  Health Check: $(check_health 2 && echo "passing" || echo "failing")"
    fi
}

# Logs command
show_logs() {
    local tail_lines="${1:-50}"
    
    if [[ "$1" == "--tail" ]]; then
        tail_lines="${2:-50}"
    fi
    
    if [[ -f "${MATRIX_SYNAPSE_LOG_FILE}" ]]; then
        tail -n "$tail_lines" "${MATRIX_SYNAPSE_LOG_FILE}"
    else
        echo "No log file found at ${MATRIX_SYNAPSE_LOG_FILE}"
    fi
}

# Main command handler
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help|--help|-h)
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        manage)
            manage_resource "$@"
            ;;
        test)
            test_resource "$@"
            ;;
        content)
            content_resource "$@"
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$@"
            ;;
        *)
            echo "Unknown command: $command"
            echo "Use 'resource-${RESOURCE_NAME} help' for usage"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"