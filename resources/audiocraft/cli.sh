#!/usr/bin/env bash
################################################################################
# AudioCraft Resource CLI
# Meta's comprehensive audio generation framework
################################################################################
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="audiocraft"

# Load common utilities
if [[ -f "${SCRIPT_DIR}/../../scripts/resources/common.sh" ]]; then
    source "${SCRIPT_DIR}/../../scripts/resources/common.sh"
fi

# Load configuration
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    source "${SCRIPT_DIR}/config/defaults.sh"
fi

# Load core library
if [[ -f "${SCRIPT_DIR}/lib/core.sh" ]]; then
    source "${SCRIPT_DIR}/lib/core.sh"
fi

################################################################################
# Help
################################################################################
show_help() {
    cat << EOF
🎵 AUDIOCRAFT Resource Management

📋 USAGE:
    resource-audiocraft <command> [subcommand] [options]

📖 DESCRIPTION:
    Meta's audio generation suite with MusicGen, AudioGen, and EnCodec

🎯 COMMAND GROUPS:
    manage               ⚙️  Resource lifecycle management
    test                 🧪 Testing and validation
    content              📄 Content management

ℹ️  INFORMATION COMMANDS:
    help                 Show this help message
    info                 Show resource information
    status               Show detailed service status
    logs                 Show service logs
    credentials          Show API credentials

⚙️  OPTIONS:
    --dry-run            Show what would be done without making changes
    --help, -h           Show help message

💡 EXAMPLES:
    # Lifecycle management
    resource-audiocraft manage install
    resource-audiocraft manage start
    resource-audiocraft manage stop

    # Testing
    resource-audiocraft test smoke
    resource-audiocraft test all

    # Content operations
    resource-audiocraft content list
    resource-audiocraft content add --file audio.wav

📚 For more help on a specific command:
    resource-audiocraft <command> --help

EOF
}

################################################################################
# Info Command
################################################################################
show_info() {
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "{}"
    fi
}

################################################################################
# Main Command Router
################################################################################
main() {
    local cmd="${1:-help}"
    shift || true

    case "$cmd" in
        # Help & Information
        help|--help|-h)
            show_help
            ;;
        info)
            show_info
            ;;
            
        # Management Commands
        manage)
            if [[ -z "${1:-}" ]]; then
                audiocraft::manage::help
            else
                local subcmd="$1"
                shift
                case "$subcmd" in
                    install)
                        audiocraft::manage::install "$@"
                        ;;
                    uninstall)
                        audiocraft::manage::uninstall "$@"
                        ;;
                    start|develop)
                        audiocraft::manage::start "$@"
                        ;;
                    stop)
                        audiocraft::manage::stop "$@"
                        ;;
                    restart)
                        audiocraft::manage::restart "$@"
                        ;;
                    *)
                        echo "Unknown manage subcommand: $subcmd"
                        audiocraft::manage::help
                        exit 1
                        ;;
                esac
            fi
            ;;
            
        # Test Commands
        test)
            if [[ -z "${1:-}" ]]; then
                audiocraft::test::help
            else
                local subcmd="$1"
                shift
                case "$subcmd" in
                    all)
                        audiocraft::test::all "$@"
                        ;;
                    smoke)
                        audiocraft::test::smoke "$@"
                        ;;
                    integration)
                        audiocraft::test::integration "$@"
                        ;;
                    unit)
                        audiocraft::test::unit "$@"
                        ;;
                    *)
                        echo "Unknown test subcommand: $subcmd"
                        audiocraft::test::help
                        exit 1
                        ;;
                esac
            fi
            ;;
            
        # Content Commands
        content)
            if [[ -z "${1:-}" ]]; then
                audiocraft::content::help
            else
                local subcmd="$1"
                shift
                case "$subcmd" in
                    list)
                        audiocraft::content::list "$@"
                        ;;
                    add)
                        audiocraft::content::add "$@"
                        ;;
                    get)
                        audiocraft::content::get "$@"
                        ;;
                    remove)
                        audiocraft::content::remove "$@"
                        ;;
                    execute)
                        audiocraft::content::execute "$@"
                        ;;
                    *)
                        echo "Unknown content subcommand: $subcmd"
                        audiocraft::content::help
                        exit 1
                        ;;
                esac
            fi
            ;;
            
        # Status & Monitoring
        status)
            audiocraft::status "$@"
            ;;
        logs)
            audiocraft::logs "$@"
            ;;
        credentials)
            audiocraft::credentials "$@"
            ;;
            
        # Shortcuts
        start|develop)
            audiocraft::manage::start "$@"
            ;;
        stop)
            audiocraft::manage::stop "$@"
            ;;
            
        *)
            echo "Unknown command: $cmd"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"