#!/usr/bin/env bash
# Godot Engine Resource CLI - v2.0 Contract Compliant

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Source configuration and libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Resource metadata
readonly RESOURCE_NAME="godot"
readonly RESOURCE_VERSION="2.0.0"
readonly RESOURCE_DESCRIPTION="Godot Engine game development platform with AI assistance"

# Command handlers
help_command() {
    cat << EOF
🎮 GODOT ENGINE Resource Management

📋 USAGE:
    resource-godot <command> [subcommand] [options]

📖 DESCRIPTION:
    ${RESOURCE_DESCRIPTION}

🎯 COMMAND GROUPS:
    manage               ⚙️  Resource lifecycle management
    test                 🧪 Testing and validation
    content              📄 Content management (projects, templates)

    💡 Use 'resource-godot <group> --help' for subcommands

ℹ️  INFORMATION COMMANDS:
    help                 Show this help message
    info                 Show resource runtime configuration
    status               Show detailed Godot status
    logs                 View Godot service logs
    credentials          Display API and LSP connection details

🔧 OTHER COMMANDS:
    projects             Manage Godot projects (list/create/delete)
    templates            Manage project templates (list/install)
    lsp                  GDScript Language Server info

⚙️  OPTIONS:
    --dry-run            Show what would be done without making changes
    --json               Output in JSON format where applicable
    --help, -h           Show help message

💡 EXAMPLES:
    # Resource lifecycle (v2.0 style)
    resource-godot manage install
    resource-godot manage start --wait
    resource-godot manage stop

    # Testing and validation
    resource-godot test smoke
    resource-godot test all

    # Content management
    resource-godot content list
    resource-godot content add --type template --name 2d-platformer

    # Project management
    resource-godot projects list
    resource-godot projects create --name MyGame

📚 For more help on a specific command:
    resource-godot <command> --help

🌐 API Endpoints:
    Health:     http://localhost:${GODOT_API_PORT}/health
    Projects:   http://localhost:${GODOT_API_PORT}/api/projects
    LSP:        localhost:${GODOT_LSP_PORT} (GDScript Language Server)

EOF
}

info_command() {
    if [[ "${1:-}" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "🎮 Godot Engine Resource Information"
        echo "=================================="
        jq -r 'to_entries | .[] | "\(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

status_command() {
    godot::status "$@"
}

logs_command() {
    godot::logs "$@"
}

credentials_command() {
    cat << EOF
🔑 Godot Engine Connection Details
==================================
API URL:        http://localhost:${GODOT_API_PORT}
Health Check:   http://localhost:${GODOT_API_PORT}/health
LSP Port:       ${GODOT_LSP_PORT}
Projects Dir:   ${GODOT_PROJECTS_DIR}
Exports Dir:    ${GODOT_EXPORTS_DIR}

📝 Example Usage:
curl http://localhost:${GODOT_API_PORT}/api/projects
EOF
}

# Manage command group
manage_command() {
    local subcommand="${1:-}"
    shift || true

    case "$subcommand" in
        install)
            godot::install "$@"
            ;;
        uninstall)
            godot::uninstall "$@"
            ;;
        start)
            godot::start "$@"
            ;;
        stop)
            godot::stop "$@"
            ;;
        restart)
            godot::restart "$@"
            ;;
        --help|-h|"")
            cat << EOF
⚙️ GODOT Lifecycle Management

USAGE:
    resource-godot manage <subcommand> [options]

SUBCOMMANDS:
    install              Install Godot Engine and dependencies
    uninstall            Remove Godot Engine completely
    start                Start Godot service
    stop                 Stop Godot service
    restart              Restart Godot service

OPTIONS:
    --force              Skip confirmation prompts
    --wait               Wait for service to be ready (start/restart)
    --timeout <seconds>  Operation timeout
    --keep-data          Preserve project data (uninstall)

EXAMPLES:
    resource-godot manage install
    resource-godot manage start --wait
    resource-godot manage stop --force
EOF
            ;;
        *)
            echo "Unknown manage subcommand: $subcommand" >&2
            echo "Run 'resource-godot manage --help' for usage" >&2
            exit 1
            ;;
    esac
}

# Test command group
test_command() {
    local subcommand="${1:-}"
    shift || true

    case "$subcommand" in
        all)
            godot::test::all "$@"
            ;;
        smoke)
            godot::test::smoke "$@"
            ;;
        integration)
            godot::test::integration "$@"
            ;;
        unit)
            godot::test::unit "$@"
            ;;
        --help|-h|"")
            cat << EOF
🧪 GODOT Testing Commands

USAGE:
    resource-godot test <subcommand> [options]

SUBCOMMANDS:
    all                  Run all test suites
    smoke                Quick health validation (<30s)
    integration          Test Godot functionality
    unit                 Test library functions

OPTIONS:
    --verbose            Show detailed test output
    --timeout <seconds>  Test timeout

EXAMPLES:
    resource-godot test smoke
    resource-godot test all --verbose
EOF
            ;;
        *)
            echo "Unknown test subcommand: $subcommand" >&2
            echo "Run 'resource-godot test --help' for usage" >&2
            exit 1
            ;;
    esac
}

# Content command group
content_command() {
    local subcommand="${1:-}"
    shift || true

    case "$subcommand" in
        add)
            godot::content::add "$@"
            ;;
        list)
            godot::content::list "$@"
            ;;
        get)
            godot::content::get "$@"
            ;;
        remove)
            godot::content::remove "$@"
            ;;
        execute)
            godot::content::execute "$@"
            ;;
        --help|-h|"")
            cat << EOF
📄 GODOT Content Management

USAGE:
    resource-godot content <subcommand> [options]

SUBCOMMANDS:
    list                 List projects and templates
    add                  Add new template or asset
    get                  Download specific content
    remove               Delete content
    execute              Run Godot commands

OPTIONS:
    --type <type>        Content type (project/template/asset)
    --name <name>        Content identifier
    --file <path>        Input/output file path
    --format <format>    Output format (text/json)

EXAMPLES:
    resource-godot content list --type template
    resource-godot content add --type template --name rpg
    resource-godot content execute --command "build MyGame"
EOF
            ;;
        *)
            echo "Unknown content subcommand: $subcommand" >&2
            echo "Run 'resource-godot content --help' for usage" >&2
            exit 1
            ;;
    esac
}

# Project management commands
projects_command() {
    local subcommand="${1:-list}"
    shift || true

    case "$subcommand" in
        list)
            godot::projects::list "$@"
            ;;
        create)
            godot::projects::create "$@"
            ;;
        delete)
            godot::projects::delete "$@"
            ;;
        --help|-h)
            cat << EOF
🎮 GODOT Project Management

USAGE:
    resource-godot projects <subcommand> [options]

SUBCOMMANDS:
    list                 List all projects
    create               Create new project
    delete               Delete project

OPTIONS:
    --name <name>        Project name
    --template <type>    Project template
    --json               Output as JSON

EXAMPLES:
    resource-godot projects list
    resource-godot projects create --name MyGame --template 2d-platformer
EOF
            ;;
        *)
            echo "Unknown projects subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# Template management  
templates_command() {
    local subcommand="${1:-list}"
    shift || true

    case "$subcommand" in
        list)
            godot::templates::list "$@"
            ;;
        install)
            godot::templates::install "$@"
            ;;
        *)
            echo "Unknown templates subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# LSP information
lsp_command() {
    cat << EOF
🔧 GDScript Language Server Protocol

Status:  $(godot::lsp::status)
Port:    ${GODOT_LSP_PORT}
URL:     localhost:${GODOT_LSP_PORT}

Configure your editor to connect to the LSP server:
- VSCode: Install godot-tools extension
- Emacs: Use eglot or lsp-mode
- Sublime: Configure godot-lsp client

EOF
}

# Main command router
main() {
    local command="${1:-}"
    shift || true

    case "$command" in
        help|--help|-h|"")
            help_command
            ;;
        info)
            info_command "$@"
            ;;
        status)
            status_command "$@"
            ;;
        logs)
            logs_command "$@"
            ;;
        credentials)
            credentials_command
            ;;
        manage)
            manage_command "$@"
            ;;
        test)
            test_command "$@"
            ;;
        content)
            content_command "$@"
            ;;
        projects)
            projects_command "$@"
            ;;
        templates)
            templates_command "$@"
            ;;
        lsp)
            lsp_command "$@"
            ;;
        *)
            echo "Unknown command: $command" >&2
            echo "Run 'resource-godot --help' for usage" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"