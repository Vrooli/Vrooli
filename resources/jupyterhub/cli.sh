#!/bin/bash
# JupyterHub Resource CLI - v2.0 Universal Contract Compliant

set -euo pipefail

# Determine the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"

# Source configuration
source "${SCRIPT_DIR}/config/defaults.sh"

# Source library functions
source "${LIB_DIR}/core.sh"
source "${LIB_DIR}/test.sh"

# Main command dispatcher
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Information commands
        help)
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        status)
            show_status "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        credentials)
            show_credentials "$@"
            ;;
            
        # Management commands
        manage)
            handle_manage "$@"
            ;;
            
        # Testing commands
        test)
            handle_test "$@"
            ;;
            
        # Content management
        content)
            handle_content "$@"
            ;;
            
        *)
            echo "❌ Unknown command: $command" >&2
            echo "Run 'resource-jupyterhub help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Show comprehensive help
show_help() {
    cat << EOF
🚀 JUPYTERHUB Resource Management

📋 USAGE:
    resource-jupyterhub <command> [subcommand] [options]

📖 DESCRIPTION:
    Multi-user Jupyter notebook server for collaborative computing

🎯 COMMAND GROUPS:
    manage               ⚙️  Resource lifecycle management
    test                 🧪 Testing and validation
    content              📄 User and notebook management

    💡 Use 'resource-jupyterhub <group> --help' for subcommands

ℹ️  INFORMATION COMMANDS:
    help                 Show this help message
    info                 Show resource configuration
    status               Show detailed status
    logs                 Show JupyterHub logs
    credentials          Show access credentials

⚙️  OPTIONS:
    --dry-run            Preview actions without executing
    --json               Output in JSON format
    --verbose            Show detailed output

💡 EXAMPLES:
    # Lifecycle management
    resource-jupyterhub manage install
    resource-jupyterhub manage start --wait
    resource-jupyterhub manage stop
    
    # User management
    resource-jupyterhub content list --type users
    resource-jupyterhub content add --type user --name alice
    resource-jupyterhub content spawn --user alice
    
    # Testing
    resource-jupyterhub test smoke
    resource-jupyterhub test all

📚 For detailed documentation:
    cat ${SCRIPT_DIR}/README.md
EOF
    exit 0
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        install)
            manage_install "$@"
            ;;
        uninstall)
            manage_uninstall "$@"
            ;;
        start)
            manage_start "$@"
            ;;
        stop)
            manage_stop "$@"
            ;;
        restart)
            manage_restart "$@"
            ;;
        help|--help)
            show_manage_help
            ;;
        *)
            echo "❌ Unknown manage subcommand: $subcommand" >&2
            show_manage_help
            exit 1
            ;;
    esac
}

# Show manage help
show_manage_help() {
    cat << EOF
⚙️  MANAGE - Resource Lifecycle Management

USAGE:
    resource-jupyterhub manage <subcommand> [options]

SUBCOMMANDS:
    install              Install JupyterHub and dependencies
    uninstall            Remove JupyterHub completely
    start                Start JupyterHub service
    stop                 Stop JupyterHub service
    restart              Restart JupyterHub service

OPTIONS:
    --force              Skip confirmation prompts
    --wait               Wait for service to be ready
    --timeout <seconds>  Operation timeout (default: 120)
    --keep-data          Preserve user data on uninstall

EXAMPLES:
    resource-jupyterhub manage install
    resource-jupyterhub manage start --wait
    resource-jupyterhub manage restart --force
EOF
}

# Handle test subcommands
handle_test() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        smoke)
            test_smoke "$@"
            ;;
        integration)
            test_integration "$@"
            ;;
        unit)
            test_unit "$@"
            ;;
        all)
            test_all "$@"
            ;;
        help|--help)
            show_test_help
            ;;
        *)
            echo "❌ Unknown test subcommand: $subcommand" >&2
            show_test_help
            exit 1
            ;;
    esac
}

# Show test help
show_test_help() {
    cat << EOF
🧪 TEST - Validation and Testing

USAGE:
    resource-jupyterhub test <subcommand> [options]

SUBCOMMANDS:
    smoke                Quick health validation (<30s)
    integration          End-to-end functionality tests
    unit                 Library function tests
    all                  Run all test suites

OPTIONS:
    --verbose            Show detailed test output
    --timeout <seconds>  Test timeout (default: 300)

EXAMPLES:
    resource-jupyterhub test smoke
    resource-jupyterhub test integration --verbose
    resource-jupyterhub test all
EOF
}

# Handle content subcommands
handle_content() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        list)
            content_list "$@"
            ;;
        add)
            content_add "$@"
            ;;
        get)
            content_get "$@"
            ;;
        remove)
            content_remove "$@"
            ;;
        execute)
            content_execute "$@"
            ;;
        spawn)
            content_spawn "$@"
            ;;
        help|--help)
            show_content_help
            ;;
        *)
            echo "❌ Unknown content subcommand: $subcommand" >&2
            show_content_help
            exit 1
            ;;
    esac
}

# Show content help
show_content_help() {
    cat << EOF
📄 CONTENT - User and Notebook Management

USAGE:
    resource-jupyterhub content <subcommand> [options]

SUBCOMMANDS:
    list                 List users, notebooks, or extensions
    add                  Add user, notebook, or extension
    get                  Get specific item details
    remove               Remove user, notebook, or extension
    execute              Execute administrative command
    spawn                Start user's notebook server

OPTIONS:
    --type <type>        Item type (users|notebooks|extensions|profiles)
    --user <username>    Target user
    --name <name>        Item name
    --json               Output in JSON format

EXAMPLES:
    resource-jupyterhub content list --type users
    resource-jupyterhub content add --type user --name alice
    resource-jupyterhub content spawn --user alice
    resource-jupyterhub content list --type extensions
EOF
}

# Run main function
main "$@"