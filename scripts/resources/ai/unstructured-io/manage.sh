#!/usr/bin/env bash
set -euo pipefail

# Unstructured.io AI Resource Management
# This script orchestrates Unstructured.io installation, configuration, and document processing

export DESCRIPTION="Install and manage Unstructured.io document processing service"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

# Handle Ctrl+C gracefully
trap 'echo ""; log::info "Unstructured.io operation interrupted by user. Exiting..."; exit 130' INT TERM

# Source common resources using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"

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
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/cache-simple.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/validate.sh"

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
                if ! unstructured_io::batch_process "${files[@]}" \
                    --strategy "$STRATEGY" \
                    --output "$OUTPUT"; then
                    log::error "Batch processing failed"
                    exit 1
                fi
            else
                # Process single file
                if ! unstructured_io::process_document "$FILE_INPUT" "$STRATEGY" "$OUTPUT" "$LANGUAGES"; then
                    log::error "Document processing failed"
                    exit 1
                fi
            fi
            ;;
        info)
            unstructured_io::info
            ;;
        logs)
            unstructured_io::logs "$FOLLOW"
            ;;
        test)
            unstructured_io::test_api
            ;;
        extract-tables)
            if [[ -z "$FILE_INPUT" ]]; then
                log::error "No file provided for table extraction"
                log::info "Use: $0 --action extract-tables --file <path>"
                exit 1
            fi
            unstructured_io::extract_tables "$FILE_INPUT"
            ;;
        extract-metadata)
            if [[ -z "$FILE_INPUT" ]]; then
                log::error "No file provided for metadata extraction"
                log::info "Use: $0 --action extract-metadata --file <path>"
                exit 1
            fi
            unstructured_io::extract_metadata "$FILE_INPUT"
            ;;
        process-directory)
            if [[ -z "$DIRECTORY" ]]; then
                log::error "No directory provided for processing"
                log::info "Use: $0 --action process-directory --directory <path>"
                exit 1
            fi
            unstructured_io::process_directory "$DIRECTORY" "$STRATEGY" "$OUTPUT" "$RECURSIVE"
            ;;
        create-report)
            if [[ -z "$FILE_INPUT" ]]; then
                log::error "No file provided for report generation"
                log::info "Use: $0 --action create-report --file <path>"
                exit 1
            fi
            unstructured_io::create_report "$FILE_INPUT" "$REPORT_FILE"
            ;;
        cache-stats)
            unstructured_io::cache_stats
            ;;
        clear-cache)
            if [[ -n "$FILE_INPUT" ]]; then
                # Clear cache for specific file
                unstructured_io::clear_cache "$FILE_INPUT"
                echo "✅ Cleared cache for: $FILE_INPUT"
            else
                # Clear all cache
                unstructured_io::clear_all_cache
            fi
            ;;
        validate-installation)
            unstructured_io::validate_installation
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