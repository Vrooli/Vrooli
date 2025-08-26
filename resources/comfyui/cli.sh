#!/usr/bin/env bash
################################################################################
# ComfyUI Resource CLI - v2.0 Universal Contract Compliant
# 
# AI image generation and manipulation workflow platform
#
# Usage:
#   resource-comfyui <command> [options]
#   resource-comfyui <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    COMFYUI_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${COMFYUI_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
COMFYUI_CLI_DIR="${APP_ROOT}/resources/comfyui"

source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
source "${COMFYUI_CLI_DIR}/config/defaults.sh"

for lib in common docker install status gpu models workflows; do
    lib_file="${COMFYUI_CLI_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

cli::init "comfyui" "ComfyUI AI image generation and manipulation platform" "v2"

CLI_COMMAND_HANDLERS["manage::install"]="install::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="comfyui::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="comfyui::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="comfyui::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="comfyui::status"
CLI_COMMAND_HANDLERS["test::integration"]="comfyui::test::integration"

CLI_COMMAND_HANDLERS["content::add"]="workflows::import_workflow"
CLI_COMMAND_HANDLERS["content::list"]="workflows::list_workflows"
CLI_COMMAND_HANDLERS["content::get"]="comfyui::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="comfyui::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="workflows::execute_workflow"

cli::register_subcommand "content" "models" "List available AI models" "models::list_models"
cli::register_subcommand "content" "download" "Download default AI models" "models::download_default_models" "modifies-system"
cli::register_command "status" "Show detailed resource status" "comfyui::status"
cli::register_command "logs" "Show ComfyUI logs" "comfyui::docker::logs"
cli::register_command "credentials" "Show ComfyUI credentials for integration" "comfyui::credentials"
cli::register_command "gpu-info" "Show GPU information for AI workloads" "gpu::get_gpu_info"

comfyui::test::integration() {
    if ! command -v curl &> /dev/null; then log::error "curl required"; return 1; fi
    local base_url="${COMFYUI_BASE_URL:-http://localhost:8188}"
    log::info "Testing ComfyUI API connectivity at $base_url..."
    if curl -s --max-time 10 "$base_url/system_stats" >/dev/null 2>&1; then
        log::success "✅ ComfyUI API is responsive"; return 0
    else
        log::warning "⚠️ ComfyUI API not accessible (may be stopped)"; return 1
    fi
}

comfyui::content::get() {
    local name="${1:-}"; [[ -z "$name" ]] && { log::error "Workflow name required"; return 1; }
    local file="${COMFYUI_DATA_DIR:-${HOME}/.comfyui}/workflows/$name"
    [[ -f "$file" ]] && cat "$file" || { log::error "Workflow not found: $name"; return 1; }
}

comfyui::content::remove() {
    local name="${1:-}"; [[ -z "$name" ]] && { log::error "Workflow name required"; return 1; }
    local file="${COMFYUI_DATA_DIR:-${HOME}/.comfyui}/workflows/$name"
    [[ -f "$file" ]] && { rm -f "$file"; log::success "Workflow removed: $name"; } || { log::error "Workflow not found: $name"; return 1; }
}

comfyui::credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    credentials::parse_args "$@" || return $?
    local status=$(credentials::get_resource_status "${COMFYUI_CONTAINER_NAME:-comfyui}")
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local conn=$(jq -n --arg host "localhost" --argjson port "${COMFYUI_DIRECT_PORT:-8188}" --arg path "/api" --argjson ssl false '{host: $host, port: $port, path: $path, ssl: $ssl}')
        local meta=$(jq -n --arg desc "ComfyUI AI image generation" --arg web "${COMFYUI_BASE_URL:-http://localhost:8188}" '{description: $desc, web_ui_url: $web, auth_enabled: false}')
        local connection=$(credentials::build_connection "main" "ComfyUI API" "httpRequest" "$conn" "{}" "$meta")
        connections_array="[$connection]"
    fi
    local response=$(credentials::build_response "comfyui" "$status" "$connections_array")
    credentials::format_output "$response"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi