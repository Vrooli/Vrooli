#!/usr/bin/env bash
set -euo pipefail

# Unstructured.io AI Resource Management
# This script orchestrates Unstructured.io installation, configuration, and document processing

DESCRIPTION="Install and manage Unstructured.io document processing service"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Handle Ctrl+C gracefully
trap 'echo ""; log::info "Unstructured.io operation interrupted by user. Exiting..."; exit 130' INT TERM

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Source configuration modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration
unstructured_io::export_config

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/process.sh"

#######################################
# Main orchestration function
# This function routes commands to the appropriate module functions
#######################################
unstructured_io::main() {
    # Parse command line arguments
    unstructured_io::parse_arguments "$@"
    
    # Route to appropriate action
    case "$ACTION" in
        install)
            unstructured_io::install "$FORCE"
            ;;
        uninstall)
            unstructured_io::uninstall
            ;;
        start)
            unstructured_io::start
            ;;
        stop)
            unstructured_io::stop
            ;;
        restart)
            unstructured_io::restart
            ;;
        status)
            unstructured_io::status
            ;;
        process)
            if [[ -z "$FILE_INPUT" ]]; then
                log::error "No file provided for processing"
                log::info "Use: $0 --action process --file <path>"
                exit 1
            fi
            
            if [[ "$BATCH" = "yes" ]]; then
                # Process multiple files (comma-separated)
                IFS=',' read -ra files <<< "$FILE_INPUT"
                unstructured_io::batch_process "${files[@]}" \
                    --strategy "$STRATEGY" \
                    --output "$OUTPUT"
            else
                # Process single file
                unstructured_io::process_document "$FILE_INPUT" "$STRATEGY" "$OUTPUT" "$LANGUAGES"
            fi
            ;;
        info)
            unstructured_io::info
            ;;
        logs)
            unstructured_io::logs "$FOLLOW"
            ;;
        *)
            log::error "Unknown action: $ACTION"
            unstructured_io::usage
            exit 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    unstructured_io::main "$@"
fi