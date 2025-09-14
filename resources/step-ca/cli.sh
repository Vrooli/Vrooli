#!/bin/bash
# Step-CA CLI - v2.0 Contract Compliant
# Private certificate authority with ACME protocol support

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="step-ca"

# Determine resource directory (handle both direct and symlinked execution)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve to actual location
    RESOURCE_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
    RESOURCE_DIR="$SCRIPT_DIR"
fi

# Source required libraries
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Main CLI entry point
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Information Commands
        help)
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        credentials)
            show_credentials "$@"
            ;;
        status)
            show_status "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
            
        # Management Commands
        manage)
            handle_manage "$@"
            ;;
            
        # Testing Commands
        test)
            handle_test "$@"
            ;;
            
        # Content Commands
        content)
            handle_content "$@"
            ;;
            
        # Legacy support
        install|uninstall|start|stop|restart)
            echo "ℹ️  Please use 'resource-$RESOURCE_NAME manage $command' instead"
            handle_manage "$command" "$@"
            ;;
            
        *)
            echo "❌ Unknown command: $command"
            echo "Run 'resource-$RESOURCE_NAME help' for usage information"
            exit 1
            ;;
    esac
}

# Show help message
show_help() {
    cat <<EOF
🔐 Step-CA Resource Management

📋 USAGE:
    resource-$RESOURCE_NAME <command> [subcommand] [options]

📖 DESCRIPTION:
    Private certificate authority with ACME protocol support

🎯 COMMAND GROUPS:
    content              📄 Certificate management
    manage               ⚙️  Resource lifecycle management
    test                 🧪 Testing and validation

    💡 Use 'resource-$RESOURCE_NAME <group> --help' for subcommands

ℹ️  INFORMATION COMMANDS:
    credentials          Show CA connection details
    help                 Show this help message
    logs                 Show Step-CA logs
    status               Show detailed Step-CA status
    info                 Show resource configuration

⚙️  OPTIONS:
    --dry-run            Show what would be done without making changes
    --help, -h           Show help message

💡 EXAMPLES:
    # Resource lifecycle
    resource-$RESOURCE_NAME manage install
    resource-$RESOURCE_NAME manage start
    resource-$RESOURCE_NAME manage stop

    # Certificate management
    resource-$RESOURCE_NAME content add --cn service.local
    resource-$RESOURCE_NAME content list

    # Testing
    resource-$RESOURCE_NAME test smoke
    resource-$RESOURCE_NAME test all

📚 For more help on a specific command:
    resource-$RESOURCE_NAME <command> --help
EOF
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
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
        --help|-h|"")
            show_manage_help
            ;;
        *)
            echo "❌ Unknown manage subcommand: $subcommand"
            show_manage_help
            exit 1
            ;;
    esac
}

# Handle test subcommands
handle_test() {
    local subcommand="${1:-}"
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
        --help|-h|"")
            show_test_help
            ;;
        *)
            echo "❌ Unknown test subcommand: $subcommand"
            show_test_help
            exit 1
            ;;
    esac
}

# Handle content subcommands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add)
            content_add "$@"
            ;;
        list)
            content_list "$@"
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
        --help|-h|"")
            show_content_help
            ;;
        *)
            echo "❌ Unknown content subcommand: $subcommand"
            show_content_help
            exit 1
            ;;
    esac
}

# Show manage help
show_manage_help() {
    cat <<EOF
⚙️  Manage Commands - Resource lifecycle management

USAGE:
    resource-$RESOURCE_NAME manage <subcommand> [options]

SUBCOMMANDS:
    install              Install Step-CA and dependencies
    uninstall            Remove Step-CA completely
    start                Start Step-CA service
    stop                 Stop Step-CA service gracefully
    restart              Restart Step-CA service

OPTIONS:
    --force              Skip confirmation prompts
    --wait               Wait for service to be ready
    --timeout <seconds>  Operation timeout (default: 60)

EXAMPLES:
    resource-$RESOURCE_NAME manage install
    resource-$RESOURCE_NAME manage start --wait
    resource-$RESOURCE_NAME manage stop --force
EOF
}

# Show test help
show_test_help() {
    cat <<EOF
🧪 Test Commands - Validation and testing

USAGE:
    resource-$RESOURCE_NAME test <subcommand> [options]

SUBCOMMANDS:
    smoke                Quick health validation (<30s)
    integration          Full functionality tests (<120s)
    unit                 Library function tests (<60s)
    all                  Run all test suites

OPTIONS:
    --verbose            Show detailed test output
    --json               Output results as JSON

EXAMPLES:
    resource-$RESOURCE_NAME test smoke
    resource-$RESOURCE_NAME test all --verbose
EOF
}

# Show content help
show_content_help() {
    cat <<EOF
📄 Content Commands - Certificate management

USAGE:
    resource-$RESOURCE_NAME content <subcommand> [options]

SUBCOMMANDS:
    add                  Issue a new certificate
    list                 List issued certificates
    get                  Get certificate details
    remove               Revoke a certificate
    execute              Run CA operations

CERTIFICATE OPTIONS:
    --cn <name>          Common name for certificate
    --san <names>        Subject alternative names (comma-separated)
    --duration <time>    Certificate validity duration (e.g., 30d, 720h)
    --type <type>        Certificate type (x509, ssh)

CA OPERATIONS (content execute):
    add-provisioner      Add authentication method (OIDC, JWK, AWS, etc)
    list-provisioners    List configured provisioners
    remove-provisioner   Remove a provisioner
    set-policy           Configure certificate lifetime policies
    get-policy           Display current certificate policies

PROVISIONER OPTIONS:
    --type <type>        Provisioner type (JWK, OIDC, ACME, AWS, GCP, Azure)
    --name <name>        Provisioner name
    --client-id <id>     OIDC client ID
    --client-secret <s>  OIDC client secret
    --issuer <url>       OIDC issuer URL
    --domain <domain>    Allowed domain for OIDC

POLICY OPTIONS:
    --default-duration   Default certificate lifetime (e.g., 24h, 30d)
    --max-duration       Maximum certificate lifetime (e.g., 720h, 90d)
    --min-duration       Minimum certificate lifetime (e.g., 5m, 1h)
    --allow-renewal-after-expiry  Allow renewal after expiry (true/false)

EXAMPLES:
    # Issue certificate
    resource-$RESOURCE_NAME content add --cn service.local --duration 30d
    
    # Add OIDC provisioner
    resource-$RESOURCE_NAME content execute add-provisioner \\
        --type OIDC --name keycloak \\
        --client-id step-ca --issuer https://auth.example.com
    
    # Set certificate policies
    resource-$RESOURCE_NAME content execute set-policy \\
        --default-duration 24h --max-duration 90d
    
    # List provisioners
    resource-$RESOURCE_NAME content execute list-provisioners
EOF
}

# Run main function
main "$@"