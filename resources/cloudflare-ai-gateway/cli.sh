#!/usr/bin/env bash
################################################################################
# Cloudflare AI Gateway Resource CLI - v2.0 Universal Contract Compliant
# 
# Resilient AI traffic proxy with caching, rate limiting, and analytics
#
# Usage:
#   resource-cloudflare-ai-gateway <command> [options]
#   resource-cloudflare-ai-gateway <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    CLOUDFLARE_AI_GATEWAY_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${CLOUDFLARE_AI_GATEWAY_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
CLOUDFLARE_AI_GATEWAY_CLI_DIR="${APP_ROOT}/resources/cloudflare-ai-gateway"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${CLOUDFLARE_AI_GATEWAY_CLI_DIR}/config/defaults.sh"

# Source resource libraries
for lib in gateway install config providers; do
    lib_file="${CLOUDFLARE_AI_GATEWAY_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "cloudflare-ai-gateway" "Cloudflare AI Gateway management" "v2"

# Override default handlers to point directly to cloudflare-ai-gateway implementations
CLI_COMMAND_HANDLERS["manage::install"]="install_cloudflare_ai_gateway"
CLI_COMMAND_HANDLERS["manage::uninstall"]="uninstall_cloudflare_ai_gateway"
CLI_COMMAND_HANDLERS["manage::start"]="gateway_start"
CLI_COMMAND_HANDLERS["manage::stop"]="gateway_stop"
CLI_COMMAND_HANDLERS["manage::restart"]="cloudflare_ai_gateway_restart"
CLI_COMMAND_HANDLERS["test::smoke"]="cloudflare_ai_gateway_test_smoke"
CLI_COMMAND_HANDLERS["content::add"]="handle_content_add"
CLI_COMMAND_HANDLERS["content::list"]="cloudflare_ai_gateway_content_list" 
CLI_COMMAND_HANDLERS["content::get"]="handle_content_get"
CLI_COMMAND_HANDLERS["content::remove"]="handle_content_remove"
CLI_COMMAND_HANDLERS["content::execute"]="handle_content_execute"

# Add cloudflare-ai-gateway-specific content subcommands
cli::register_subcommand "content" "configure" "Configure gateway settings" "gateway_configure"

# Additional information commands
cli::register_command "status" "Show detailed gateway status" "cloudflare_ai_gateway_status"
cli::register_command "logs" "Show gateway logs" "gateway_logs"
cli::register_command "info" "Display gateway configuration" "gateway_info"

# Gateway restart handler
cloudflare_ai_gateway_restart() { gateway_stop "$@"; sleep 2; gateway_start "$@"; }

# Simple smoke test function
cloudflare_ai_gateway_test_smoke() {
    initialize_data_dir
    echo "🧪 Running Cloudflare AI Gateway smoke test..."
    get_cloudflare_credentials >/dev/null 2>&1 || { echo "❌ FAIL: Credentials not configured"; return 1; }
    echo "✅ PASS: Credentials configured"
    [[ -d "${DATA_DIR}" ]] && echo "✅ PASS: Data directory exists" || { echo "❌ FAIL: Data directory missing"; return 1; }
    [[ -f "${CONFIG_FILE}" && -f "${STATE_FILE}" ]] && echo "✅ SUCCESS: Cloudflare AI Gateway smoke test passed" || { echo "❌ FAIL: Config files missing"; return 1; }
}

# Status function without format_output dependency
cloudflare_ai_gateway_status() {
    initialize_data_dir
    if ! get_cloudflare_credentials >/dev/null 2>&1; then
        echo "Status: Not Configured (missing Cloudflare credentials)"
    else
        echo "Status: $(jq -r '.status' "${STATE_FILE}" | sed 's/^./\U&/')"
    fi
}

# Content list function without format_output dependency
cloudflare_ai_gateway_content_list() {
    initialize_data_dir
    echo "Configurations:"
    if [[ -d "${DATA_DIR}/configs" ]]; then
        find "${DATA_DIR}/configs" -name "*.json" -exec basename {} .json \; 2>/dev/null | sed 's/^/  /' || echo "  No configurations found"
    else
        echo "  No configurations found"
    fi
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi