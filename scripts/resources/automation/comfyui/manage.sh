#!/usr/bin/env bash
set -euo pipefail

# ComfyUI Resource Management Script
# This is the main entry point that sources all modular components

DESCRIPTION="Manages ComfyUI - AI image generation workflow platform"

# Get script directory
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source all modules in correct order
# 1. Common utilities first (includes config)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"

# 2. Core functionality modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/gpu.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/models.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/workflows.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"

# 3. User messages (optional)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

#######################################
# Main function - route actions to appropriate handlers
#######################################
comfyui::main() {
    comfyui::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            comfyui::install
            ;;
        "uninstall")
            comfyui::uninstall
            ;;
        "start")
            comfyui::start
            ;;
        "stop")
            comfyui::stop
            ;;
        "restart")
            comfyui::restart
            ;;
        "status")
            comfyui::status
            ;;
        "logs")
            comfyui::logs
            ;;
        "info")
            comfyui::info
            ;;
        "download-models")
            comfyui::download_models
            ;;
        "execute-workflow")
            comfyui::execute_workflow
            ;;
        "import-workflow")
            comfyui::import_workflow
            ;;
        "list-models")
            comfyui::list_models
            ;;
        "list-workflows")
            comfyui::list_workflows
            ;;
        "gpu-info")
            comfyui::get_gpu_info
            ;;
        "validate-nvidia")
            comfyui::validate_nvidia_requirements
            ;;
        "check-ready")
            comfyui::check_ready
            ;;
        "cleanup-help")
            comfyui::cleanup_help
            ;;
        "help")
            # Show help using the args system
            comfyui::parse_arguments --help
            ;;
        "extended-help")
            comfyui::show_extended_help
            ;;
        "quickstart")
            comfyui::show_quickstart
            ;;
        "model-sources")
            comfyui::show_model_sources
            ;;
        "workflow-examples")
            comfyui::show_workflow_examples
            ;;
        "api-examples")
            comfyui::show_api_examples
            ;;
        *)
            log::error "Unknown action: $ACTION"
            # Show help using the args system
            comfyui::parse_arguments --help
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    comfyui::main "$@"
fi