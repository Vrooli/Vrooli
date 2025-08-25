#!/usr/bin/env bash
set -euo pipefail

# ComfyUI Resource Management Script
# This is the main entry point that sources all modular components

export DESCRIPTION="Manages ComfyUI - AI image generation workflow platform"

# Get script directory (unique variable name)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
COMFYUI_SCRIPT_DIR="${APP_ROOT}/resources/comfyui"
COMFYUI_LIB_DIR="${COMFYUI_SCRIPT_DIR}/lib"

# Source var.sh first to get all path variables
# shellcheck disable=SC1091
source "${COMFYUI_SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source all modules in correct order using var_ variables
# 1. Common utilities first (includes config)
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/common.sh"

# 2. Core functionality modules
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/gpu.sh"
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/docker.sh"
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/models.sh"
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/workflows.sh"
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/status.sh"
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/install.sh"

# 3. User messages (optional)
# shellcheck disable=SC1091
source "${COMFYUI_SCRIPT_DIR}/config/messages.sh"

#######################################
# Main function - route actions to appropriate handlers
#######################################

manage::main() {
    common::parse_arguments "$@"
    
    # ACTION is declared and set in common.sh which is sourced above
    # shellcheck disable=SC2153
    case "$ACTION" in
        install)
            install::install
            ;;
        uninstall)
            install::uninstall
            ;;
        start)
            comfyui::docker::start
            ;;
        stop)
            comfyui::docker::stop
            ;;
        restart)
            comfyui::docker::restart
            ;;
        status)
            status::status
            ;;
        logs)
            comfyui::docker::logs
            ;;
        info)
            status::info
            ;;
        download-models)
            models::download_models
            ;;
        execute-workflow)
            workflows::execute_workflow
            ;;
        import-workflow)
            workflows::import_workflow
            ;;
        list-models)
            models::list_models
            ;;
        list-workflows)
            workflows::list_workflows
            ;;
        gpu-info)
            gpu::get_gpu_info
            ;;
        validate-nvidia)
            gpu::validate_nvidia_requirements
            ;;
        check-ready)
            common::check_ready
            ;;
        cleanup-help)
            common::cleanup_help
            ;;
        help)
            # Show help using the args system
            common::parse_arguments --help
            ;;
        extended-help)
            messages::show_extended_help
            ;;
        quickstart)
            messages::show_quickstart
            ;;
        model-sources)
            messages::show_model_sources
            ;;
        workflow-examples)
            messages::show_workflow_examples
            ;;
        api-examples)
            messages::show_api_examples
            ;;
        *)
            log::error "Unknown action: $ACTION"
            # Show help using the args system
            common::parse_arguments --help
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    manage::main "$@"
fi