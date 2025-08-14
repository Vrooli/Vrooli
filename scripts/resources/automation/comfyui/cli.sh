#!/usr/bin/env bash
################################################################################
# ComfyUI Resource CLI
# 
# Lightweight CLI interface for ComfyUI that delegates to existing lib functions.
#
# Usage:
#   resource-comfyui <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
COMFYUI_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$COMFYUI_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$COMFYUI_CLI_DIR"
export COMFYUI_SCRIPT_DIR="$COMFYUI_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source ComfyUI configuration
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/config/defaults.sh" 2>/dev/null || true

# Source ComfyUI libraries
for lib in common gpu docker models workflows status install; do
    lib_file="${COMFYUI_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "comfyui"

################################################################################
# Delegate to existing ComfyUI functions
################################################################################

# Inject workflow into ComfyUI
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-comfyui inject <workflow.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would inject workflow: $file"
        return 0
    fi
    
    # Use existing workflow import function
    export WORKFLOW_PATH="$file"
    if command -v workflows::import_workflow &>/dev/null; then
        workflows::import_workflow
    else
        log::error "ComfyUI workflow import function not available"
        return 1
    fi
}

# Validate ComfyUI configuration
resource_cli::validate() {
    if command -v common::check_ready &>/dev/null; then
        common::check_ready
    else
        # Basic validation
        log::header "Validating ComfyUI"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "comfyui" || {
            log::error "ComfyUI container not running"
            return 1
        }
        log::success "ComfyUI is running"
    fi
}

# Show ComfyUI status
resource_cli::status() {
    if command -v status::status &>/dev/null; then
        status::status
    else
        # Basic status
        log::header "ComfyUI Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "comfyui"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=comfyui" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start ComfyUI
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start ComfyUI"
        return 0
    fi
    
    if command -v docker::start &>/dev/null; then
        docker::start
    else
        docker start comfyui || log::error "Failed to start ComfyUI"
    fi
}

# Stop ComfyUI
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop ComfyUI"
        return 0
    fi
    
    if command -v docker::stop &>/dev/null; then
        docker::stop
    else
        docker stop comfyui || log::error "Failed to stop ComfyUI"
    fi
}

# Install ComfyUI
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install ComfyUI"
        return 0
    fi
    
    if command -v install::install &>/dev/null; then
        install::install
    else
        log::error "ComfyUI install function not available"
        return 1
    fi
}

# Uninstall ComfyUI
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove ComfyUI and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall ComfyUI"
        return 0
    fi
    
    if command -v install::uninstall &>/dev/null; then
        install::uninstall
    else
        docker stop comfyui 2>/dev/null || true
        docker rm comfyui 2>/dev/null || true
        log::success "ComfyUI uninstalled"
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "comfyui"; return 0; }
        return 1
    fi
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "$COMFYUI_CONTAINER_NAME")
    
    # Build connections array - ComfyUI runs without auth by default
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # ComfyUI HTTP API connection (no authentication by default)
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "$COMFYUI_DIRECT_PORT" \
            --arg path "/api" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "ComfyUI AI image generation and manipulation" \
            --arg web_ui "$COMFYUI_BASE_URL" \
            --argjson jupyter_port "$COMFYUI_JUPYTER_PORT" \
            --argjson models_dir "$COMFYUI_MODELS_DIR" \
            '{
                description: $description,
                web_ui_url: $web_ui,
                jupyter_port: $jupyter_port,
                models_directory: $models_dir,
                auth_enabled: false
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "ComfyUI API" \
            "httpRequest" \
            "$connection_obj" \
            "{}" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "comfyui" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

################################################################################
# ComfyUI-specific commands
################################################################################

# Execute workflow
comfyui_execute_workflow() {
    local workflow="${1:-}"
    
    if [[ -z "$workflow" ]]; then
        log::error "Workflow file required"
        echo "Usage: resource-comfyui execute-workflow <workflow.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$workflow" == shared:* ]]; then
        workflow="${VROOLI_ROOT}/${workflow#shared:}"
    fi
    
    if [[ ! -f "$workflow" ]]; then
        log::error "Workflow file not found: $workflow"
        return 1
    fi
    
    export WORKFLOW_PATH="$workflow"
    if command -v workflows::execute_workflow &>/dev/null; then
        workflows::execute_workflow
    else
        log::error "Workflow execution not available"
        return 1
    fi
}

# List workflows
comfyui_list_workflows() {
    if command -v workflows::list_workflows &>/dev/null; then
        workflows::list_workflows
    else
        log::error "Workflow listing not available"
        return 1
    fi
}

# Download models
comfyui_download_models() {
    if command -v models::download_default_models &>/dev/null; then
        models::download_default_models
    elif command -v models::download_models &>/dev/null; then
        models::download_models
    else
        log::error "Model download not available"
        return 1
    fi
}

# List models
comfyui_list_models() {
    if command -v models::list_models &>/dev/null; then
        models::list_models
    else
        log::error "Model listing not available"
        return 1
    fi
}

# GPU info
comfyui_gpu_info() {
    if command -v gpu::get_gpu_info &>/dev/null; then
        gpu::get_gpu_info
    else
        log::error "GPU info not available"
        return 1
    fi
}

# Validate NVIDIA requirements
comfyui_validate_nvidia() {
    if command -v gpu::validate_nvidia_requirements &>/dev/null; then
        gpu::validate_nvidia_requirements
    else
        log::error "NVIDIA validation not available"
        return 1
    fi
}

# Show logs
comfyui_logs() {
    if command -v docker::logs &>/dev/null; then
        docker::logs
    else
        docker logs comfyui 2>/dev/null || log::error "Failed to show logs"
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🎨 ComfyUI Resource CLI

USAGE:
    resource-comfyui <command> [options]

CORE COMMANDS:
    inject <file>         Inject workflow into ComfyUI
    validate             Validate ComfyUI configuration
    status               Show ComfyUI status
    start                Start ComfyUI container
    stop                 Stop ComfyUI container
    install              Install ComfyUI
    uninstall            Uninstall ComfyUI (requires --force)
    credentials          Get connection credentials for n8n integration
    
COMFYUI COMMANDS:
    execute-workflow <file>  Execute a workflow
    list-workflows          List available workflows
    download-models         Download default models
    list-models            List available models
    gpu-info               Show GPU information
    validate-nvidia        Validate NVIDIA requirements
    logs                   Show ComfyUI logs

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-comfyui status
    resource-comfyui execute-workflow examples/basic_text_to_image.json
    resource-comfyui download-models
    resource-comfyui gpu-info
    resource-comfyui inject shared:initialization/automation/comfyui/workflow.json

For more information: https://docs.vrooli.com/resources/comfyui
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
            ;;
            
        # ComfyUI-specific commands
        execute-workflow|exec-workflow)
            comfyui_execute_workflow "$@"
            ;;
        list-workflows)
            comfyui_list_workflows "$@"
            ;;
        download-models)
            comfyui_download_models "$@"
            ;;
        list-models)
            comfyui_list_models "$@"
            ;;
        gpu-info)
            comfyui_gpu_info "$@"
            ;;
        validate-nvidia)
            comfyui_validate_nvidia "$@"
            ;;
        logs)
            comfyui_logs "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi