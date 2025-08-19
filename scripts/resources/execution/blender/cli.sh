#!/bin/bash
# Blender Resource CLI

# Get the real script directory (following symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    BLENDER_CLI_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
    BLENDER_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# Source libraries
source "${BLENDER_CLI_DIR}/lib/core.sh"
source "${BLENDER_CLI_DIR}/lib/status.sh"
source "${BLENDER_CLI_DIR}/lib/inject.sh"

# Show help
show_help() {
    cat << EOF
Blender Resource CLI

Usage: $(basename "$0") <command> [options]

Commands:
  status [--format json]    Check Blender status
  start                     Start Blender service
  stop                      Stop Blender service
  install                   Install Blender
  inject <file>            Inject a Python script for Blender
  list                     List injected scripts
  run <script>             Run a specific script with Blender
  remove <script>          Remove an injected script
  help                     Show this help message

Options:
  --format json            Output in JSON format (for status command)
  --verbose               Show detailed information

Examples:
  $(basename "$0") status
  $(basename "$0") start
  $(basename "$0") inject my_model.py
  $(basename "$0") run generate_scene.py
  $(basename "$0") list

EOF
}

# Main command handler
main() {
    local command="${1:-help}"
    shift
    
    case "$command" in
        status)
            local verbose="false"
            local format="text"
            
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --format)
                        format="$2"
                        shift 2
                        ;;
                    --verbose)
                        verbose="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            blender::status "$verbose" "$format"
            ;;
            
        start)
            blender::start
            ;;
            
        stop)
            blender::stop
            ;;
            
        install)
            blender::install
            ;;
            
        inject)
            if [[ -z "$1" ]]; then
                echo "[ERROR] No file specified"
                echo "Usage: $(basename "$0") inject <file>"
                exit 1
            fi
            blender::inject "$1"
            ;;
            
        list)
            local format="text"
            if [[ "${1:-}" == "--format" && "${2:-}" == "json" ]]; then
                format="json"
            fi
            blender::list_injected "$format"
            ;;
            
        run)
            if [[ -z "$1" ]]; then
                echo "[ERROR] No script specified"
                echo "Usage: $(basename "$0") run <script>"
                exit 1
            fi
            blender::run_script "$1"
            ;;
            
        remove)
            if [[ -z "$1" ]]; then
                echo "[ERROR] No script specified"
                echo "Usage: $(basename "$0") remove <script>"
                exit 1
            fi
            blender::remove_injected "$1"
            ;;
            
        help|--help|-h)
            show_help
            ;;
            
        *)
            echo "[ERROR] Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"