#!/usr/bin/env bash
# Nextcloud Resource CLI - v2.0 Contract Compliant
# Self-hosted file sync, share, and collaboration platform

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="nextcloud"

# Source configurations
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# ============================================================================
# Help & Information Commands
# ============================================================================

show_help() {
    cat << EOF
🔧 NEXTCLOUD Resource Management

📋 USAGE:
    resource-nextcloud <command> [subcommand] [options]

📖 DESCRIPTION:
    Self-hosted file sync, share, and collaboration platform

🎯 COMMAND GROUPS:
    manage               ⚙️  Resource lifecycle management
    content              📄 File and content management
    test                 🧪 Testing and validation

    💡 Use 'resource-nextcloud <group> --help' for subcommands

ℹ️  INFORMATION COMMANDS:
    help                 Show this help message
    info                 Show resource configuration
    status               Show detailed status
    logs                 Show container logs
    credentials          Show connection details

🔧 ADMIN COMMANDS:
    occ                  Execute Nextcloud OCC commands
    users                User management operations
    apps                 App management operations
    config               Configuration management

⚙️  OPTIONS:
    --dry-run            Show what would be done without making changes
    --json               Output in JSON format
    --help, -h           Show help message

💡 EXAMPLES:
    # Lifecycle management
    resource-nextcloud manage install
    resource-nextcloud manage start --wait
    resource-nextcloud manage stop

    # File operations
    resource-nextcloud content add --file document.pdf
    resource-nextcloud content list
    resource-nextcloud content get --name document.pdf

    # Testing
    resource-nextcloud test smoke
    resource-nextcloud test all

    # Administration
    resource-nextcloud occ maintenance:mode --on
    resource-nextcloud users add --username john

📚 For more help on a specific command:
    resource-nextcloud <command> --help
EOF
}

show_info() {
    if [[ "${1:-}" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "Nextcloud Resource Information"
        echo "=============================="
        echo "Port: ${NEXTCLOUD_PORT}"
        echo "Admin User: ${NEXTCLOUD_ADMIN_USER}"
        echo "Data Directory: ${NEXTCLOUD_DATA_DIR}"
        echo ""
        echo "Runtime Configuration:"
        jq -r 'to_entries[] | "  \(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# ============================================================================
# Management Commands
# ============================================================================

manage_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            install_nextcloud "$@"
            ;;
        uninstall)
            uninstall_nextcloud "$@"
            ;;
        start)
            start_nextcloud "$@"
            ;;
        stop)
            stop_nextcloud "$@"
            ;;
        restart)
            restart_nextcloud "$@"
            ;;
        --help|help)
            show_manage_help
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand" >&2
            show_manage_help
            exit 1
            ;;
    esac
}

show_manage_help() {
    cat << EOF
⚙️ Nextcloud Lifecycle Management

USAGE:
    resource-nextcloud manage <subcommand> [options]

SUBCOMMANDS:
    install              Install Nextcloud and dependencies
    uninstall            Remove Nextcloud completely
    start                Start Nextcloud services
    stop                 Stop Nextcloud services  
    restart              Restart Nextcloud services

OPTIONS:
    --force              Skip confirmation prompts
    --wait               Wait for service to be ready
    --timeout <seconds>  Timeout for operations (default: 60)
    --keep-data          Keep data when uninstalling

EXAMPLES:
    resource-nextcloud manage install
    resource-nextcloud manage start --wait
    resource-nextcloud manage stop
    resource-nextcloud manage uninstall --keep-data
EOF
}

# ============================================================================
# Content Commands
# ============================================================================

content_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add)
            add_content "$@"
            ;;
        list)
            list_content "$@"
            ;;
        get)
            get_content "$@"
            ;;
        remove)
            remove_content "$@"
            ;;
        execute)
            execute_content "$@"
            ;;
        --help|help)
            show_content_help
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand" >&2
            show_content_help
            exit 1
            ;;
    esac
}

show_content_help() {
    cat << EOF
📄 Nextcloud Content Management

USAGE:
    resource-nextcloud content <subcommand> [options]

SUBCOMMANDS:
    add                  Upload file to Nextcloud
    list                 List files and folders
    get                  Download file from Nextcloud
    remove               Delete file from Nextcloud
    execute              Execute operations (share, etc.)

OPTIONS:
    --file <path>        File path for upload
    --name <name>        File/folder name
    --filter <pattern>   Filter pattern for listing
    --output <path>      Output path for download
    --options <opts>     Options for execute command

EXAMPLES:
    resource-nextcloud content add --file document.pdf
    resource-nextcloud content list --filter "*.pdf"
    resource-nextcloud content get --name document.pdf --output ./download.pdf
    resource-nextcloud content remove --name old-file.txt
    resource-nextcloud content execute --name share --options "file=doc.pdf,user=bob"
EOF
}

# ============================================================================
# Test Commands
# ============================================================================

test_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        all)
            test_all "$@"
            ;;
        smoke)
            test_smoke "$@"
            ;;
        integration)
            test_integration "$@"
            ;;
        unit)
            test_unit "$@"
            ;;
        --help|help)
            show_test_help
            ;;
        *)
            echo "Error: Unknown test subcommand: $subcommand" >&2
            show_test_help
            exit 1
            ;;
    esac
}

show_test_help() {
    cat << EOF
🧪 Nextcloud Testing

USAGE:
    resource-nextcloud test <subcommand> [options]

SUBCOMMANDS:
    smoke                Quick health validation (<30s)
    integration          Full integration tests
    unit                 Unit tests for functions
    all                  Run all test suites

OPTIONS:
    --verbose            Show detailed test output
    --timeout <seconds>  Test timeout (default varies by test)

EXAMPLES:
    resource-nextcloud test smoke
    resource-nextcloud test integration --verbose
    resource-nextcloud test all
EOF
}

# ============================================================================
# Admin Commands
# ============================================================================

occ_command() {
    # Execute Nextcloud OCC commands
    local container_name="nextcloud_nextcloud_1"
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${container_name}$"; then
        echo "Error: Nextcloud container is not running" >&2
        echo "Start it with: resource-nextcloud manage start" >&2
        exit 1
    fi
    
    docker exec -u www-data "$container_name" php occ "$@"
}

users_command() {
    local operation="${1:-list}"
    shift || true
    
    case "$operation" in
        list)
            occ_command user:list
            ;;
        add)
            local username=""
            local password=""
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --username) username="$2"; shift 2 ;;
                    --password) password="$2"; shift 2 ;;
                    *) shift ;;
                esac
            done
            if [[ -z "$username" ]]; then
                echo "Error: --username required" >&2
                exit 1
            fi
            if [[ -z "$password" ]]; then
                password=$(openssl rand -base64 12)
                echo "Generated password: $password"
            fi
            occ_command user:add --password-from-env "$username" <<< "$password"
            ;;
        delete)
            local username="${1:-}"
            if [[ -z "$username" ]]; then
                echo "Error: Username required" >&2
                exit 1
            fi
            occ_command user:delete "$username"
            ;;
        *)
            echo "Usage: resource-nextcloud users [list|add|delete] [options]" >&2
            exit 1
            ;;
    esac
}

# ============================================================================
# Status & Monitoring
# ============================================================================

show_status() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        get_status_json
    else
        echo "Nextcloud Resource Status"
        echo "========================"
        
        # Check container status
        if docker ps --format "{{.Names}}" | grep -q "nextcloud_nextcloud_1"; then
            echo "Container: Running ✓"
            
            # Check health
            if timeout 5 curl -sf "http://localhost:${NEXTCLOUD_PORT}/status.php" &>/dev/null; then
                echo "Health: OK ✓"
                
                # Get Nextcloud status
                local status_json=$(curl -sf "http://localhost:${NEXTCLOUD_PORT}/status.php" 2>/dev/null || echo "{}")
                if [[ -n "$status_json" ]] && [[ "$status_json" != "{}" ]]; then
                    echo ""
                    echo "Nextcloud Status:"
                    echo "$status_json" | jq -r 'to_entries[] | "  \(.key): \(.value)"' 2>/dev/null || echo "  Unable to parse status"
                fi
            else
                echo "Health: Failed ✗"
            fi
            
            # Show resource usage
            echo ""
            echo "Resource Usage:"
            docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" nextcloud_nextcloud_1 2>/dev/null || echo "  Unable to get stats"
        else
            echo "Container: Not Running ✗"
            echo ""
            echo "Start with: resource-nextcloud manage start"
        fi
    fi
}

show_logs() {
    local lines="${1:-50}"
    local container_name="nextcloud_nextcloud_1"
    
    if docker ps -a --format "{{.Names}}" | grep -q "^${container_name}$"; then
        docker logs --tail "$lines" "$container_name"
    else
        echo "No Nextcloud container found" >&2
        exit 1
    fi
}

show_credentials() {
    cat << EOF
Nextcloud Connection Details
===========================
URL: http://localhost:${NEXTCLOUD_PORT}
Admin User: ${NEXTCLOUD_ADMIN_USER}
Admin Password: ${NEXTCLOUD_ADMIN_PASSWORD}

WebDAV URL: http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/
OCS API URL: http://localhost:${NEXTCLOUD_PORT}/ocs/v2.php/
Status URL: http://localhost:${NEXTCLOUD_PORT}/status.php

To change credentials, modify environment variables:
  export NEXTCLOUD_ADMIN_USER=newadmin
  export NEXTCLOUD_ADMIN_PASSWORD=newpassword
  
Then reinstall: resource-nextcloud manage uninstall && resource-nextcloud manage install
EOF
}

# ============================================================================
# Main Command Router
# ============================================================================

main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        help|--help|-h|"")
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        manage)
            manage_command "$@"
            ;;
        content)
            content_command "$@"
            ;;
        test)
            test_command "$@"
            ;;
        status)
            show_status "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        credentials)
            show_credentials
            ;;
        occ)
            occ_command "$@"
            ;;
        users)
            users_command "$@"
            ;;
        apps)
            occ_command app:list "$@"
            ;;
        config)
            occ_command config:list "$@"
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run 'resource-nextcloud help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"