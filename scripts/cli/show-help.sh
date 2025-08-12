#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Help System
# 
# Shows comprehensive help information for the Vrooli CLI.
#
################################################################################

set -euo pipefail

# CLI version
CLI_VERSION="1.0.0"
VROOLI_VERSION="2.0.0"

# Show main help
show_help() {
    cat << 'EOF'
                          ___ _ 
 _   _ _ __ ___   ___    / (_|_)
| | | | '__/ _ \ / _ \  / /| | |
| |_| | | | (_) | (_) |/ / | | |
 \__,_|_|  \___/ \___//_/  |_|_|
                                   
🚀 Vrooli CLI - AI Platform Management Tool

USAGE:
    vrooli <COMMAND> [OPTIONS]

LIFECYCLE COMMANDS:
    setup               Initialize and configure the development environment
    develop             Start development environment (services, apps, etc.)
    build               Build applications and artifacts
    test                Run test suites
    deploy              Deploy to target environment
    clean               Clean build artifacts and temporary files
    backup              Create backups of data and configuration
    restore             Restore from backups

APP MANAGEMENT:
    app list            List all generated apps with their status
    app status <name>   Show detailed status of a specific app  
    app protect <name>  Mark app as protected from auto-regeneration
    app diff <name>     Show changes from original generation
    app regenerate      Regenerate app from scenario
    app backup <name>   Create manual backup of app
    app restore <name>  Restore app from backup

SCENARIO OPERATIONS:
    scenario list       List all available scenarios
    scenario info       Show detailed information about a scenario
    scenario validate   Validate scenario configuration
    scenario convert    Convert scenario to standalone app
    scenario enable     Enable scenario in catalog
    scenario disable    Disable scenario in catalog

RESOURCE MANAGEMENT:
    resource list       List all available resources
    resource status     Show status of resources
    resource install    Install a specific resource
    resource enable     Enable resource in configuration
    resource disable    Disable resource in configuration

GLOBAL OPTIONS:
    --help, -h          Show this help message
    --version, -v       Show version information
    --verbose           Enable verbose output

EXAMPLES:
    vrooli setup --target docker           # Setup development environment
    vrooli develop --detached yes          # Start services in background
    vrooli app status research-assistant   # Check app customization status
    vrooli scenario convert my-scenario    # Generate app from scenario
    vrooli resource install ollama         # Install Ollama AI model

QUICK START:
    1. Run 'vrooli setup' to initialize your environment
    2. Run 'vrooli develop' to start the development server
    3. Run 'vrooli app list' to see generated applications

CONFIGURATION:
    • Main config: ~/.vrooli/config.json
    • Resources: ~/.vrooli/resources.local.json
    • Apps directory: ~/generated-apps/

For detailed documentation: https://docs.vrooli.com/cli
Report issues: https://github.com/Vrooli/Vrooli/issues

EOF

    echo "Version: Vrooli CLI v${CLI_VERSION} | Platform v${VROOLI_VERSION}"
}

# Show command-specific help
show_command_help() {
    local command="${1:-}"
    
    case "$command" in
        app)
            "${VROOLI_ROOT}/scripts/cli/app-commands.sh" --help
            ;;
        scenario)
            "${VROOLI_ROOT}/scripts/cli/scenario-commands.sh" --help
            ;;
        resource)
            "${VROOLI_ROOT}/scripts/cli/resource-commands.sh" --help
            ;;
        setup|develop|build|test|deploy|clean|backup|restore)
            "${VROOLI_ROOT}/scripts/manage.sh" "$command" --help 2>/dev/null || {
                echo "Help for '$command' command:"
                echo ""
                echo "This is a lifecycle command that manages the Vrooli environment."
                echo "Run 'vrooli $command --help' for detailed options."
            }
            ;;
        *)
            show_help
            ;;
    esac
}

# Main execution
if [[ $# -gt 0 ]]; then
    show_command_help "$1"
else
    show_help
fi