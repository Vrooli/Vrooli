#!/usr/bin/env bash
################################################################################
# SearXNG Resource CLI - v2.0 Universal Contract Compliant
# 
# Privacy-respecting metasearch engine with comprehensive search capabilities
#
# Usage:
#   resource-searxng <command> [options]
#   resource-searxng <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    SEARXNG_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${SEARXNG_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
SEARXNG_CLI_DIR="${APP_ROOT}/resources/searxng"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/config/defaults.sh"

# Source SearXNG libraries
for lib in common docker install status config api; do
    lib_file="${SEARXNG_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "searxng" "SearXNG privacy-respecting search engine management" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="searxng::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="searxng::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="searxng::start_container"  
CLI_COMMAND_HANDLERS["manage::stop"]="searxng::stop_container"
CLI_COMMAND_HANDLERS["manage::restart"]="searxng::restart_container"
CLI_COMMAND_HANDLERS["test::smoke"]="searxng::is_healthy"

# Content handlers - SearXNG business functionality (search operations)
CLI_COMMAND_HANDLERS["content::add"]="searxng::batch_search_file"
CLI_COMMAND_HANDLERS["content::list"]="searxng::show_config"
CLI_COMMAND_HANDLERS["content::get"]="searxng::search"
CLI_COMMAND_HANDLERS["content::remove"]="searxng::reset_config"
CLI_COMMAND_HANDLERS["content::execute"]="searxng::execute_search"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "searxng::status"
cli::register_command "logs" "Show SearXNG logs" "searxng::get_logs"

# ==============================================================================
# OPTIONAL SEARXNG-SPECIFIC COMMANDS
# ==============================================================================
# Additional top-level commands for SearXNG-specific operations
cli::register_command "credentials" "Show n8n integration credentials" "searxng::show_credentials"

# Custom content subcommands for SearXNG-specific search operations  
cli::register_subcommand "content" "search" "Perform search query" "searxng::execute_search"
cli::register_subcommand "content" "lucky" "I'm Feeling Lucky search" "searxng::execute_lucky"
cli::register_subcommand "content" "headlines" "Get news headlines" "searxng::execute_headlines"
cli::register_subcommand "content" "batch" "Batch search from file" "searxng::execute_batch"
cli::register_subcommand "content" "interactive" "Interactive search mode" "searxng::interactive_search"
cli::register_subcommand "content" "benchmark" "Run performance benchmark" "searxng::benchmark"

# Custom content subcommands for configuration management
cli::register_subcommand "content" "config" "Show configuration" "searxng::show_config"
cli::register_subcommand "content" "backup" "Backup configuration" "searxng::backup"
cli::register_subcommand "content" "restore" "Restore from backup" "searxng::restore"

# Redis caching management
cli::register_subcommand "content" "enable-redis" "Enable Redis caching" "searxng::enable_redis"
cli::register_subcommand "content" "disable-redis" "Disable Redis caching" "searxng::disable_redis"
cli::register_subcommand "content" "redis-status" "Show Redis caching status" "searxng::redis_status"

# Custom test subcommands for SearXNG health/connectivity testing
cli::register_subcommand "test" "api" "Test API endpoints" "searxng::test_api"
cli::register_subcommand "test" "diagnose" "Run comprehensive diagnostics" "searxng::diagnose"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi