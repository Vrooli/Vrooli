#!/usr/bin/env bash
# farmOS Resource CLI Interface - v2.0 Universal Contract Compliant
# Provides command-line access to farmOS agricultural management platform

set -euo pipefail

# Get application root directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    FARMOS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${FARMOS_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
RESOURCE_DIR="${APP_ROOT}/resources/farmos"
RESOURCE_NAME="farmos"

# Source required framework files
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${APP_ROOT}/scripts/lib/resources/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${APP_ROOT}/scripts/lib/resources/common.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source resource configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Initialize CLI framework in v2.0 mode
cli::init "farmos" "farmOS agricultural management platform" "v2"

# Override default handlers for management commands
CLI_COMMAND_HANDLERS["manage::install"]="farmos::manage::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="farmos::manage::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="farmos::manage::start"
CLI_COMMAND_HANDLERS["manage::stop"]="farmos::manage::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="farmos::manage::restart"

# Override test handlers
CLI_COMMAND_HANDLERS["test::all"]="farmos::test::all"
CLI_COMMAND_HANDLERS["test::smoke"]="farmos::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="farmos::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="farmos::test::unit"

# Override content handlers
CLI_COMMAND_HANDLERS["content::add"]="farmos::content::add"
CLI_COMMAND_HANDLERS["content::list"]="farmos::content::list"
CLI_COMMAND_HANDLERS["content::get"]="farmos::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="farmos::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="farmos::content::execute"

# Register standard commands
cli::register_command "help" "Show this help message" "farmos::help"
cli::register_command "info" "Display resource information" "farmos::info"
cli::register_command "status" "Show farmOS status" "farmos::status"
cli::register_command "logs" "View farmOS logs" "farmos::logs"
cli::register_command "credentials" "Display API credentials" "farmos::credentials"

# Farm-specific operations group
cli::register_command_group "farm" "Farm-specific operations"
cli::register_subcommand "farm" "create-field" "Create a new field" "farmos::farm::create_field"
cli::register_subcommand "farm" "log-activity" "Log farm activity" "farmos::farm::log_activity"
cli::register_subcommand "farm" "export" "Export farm records" "farmos::farm::export"
cli::register_subcommand "farm" "import" "Import farm data" "farmos::farm::import"
cli::register_subcommand "farm" "seed-demo" "Seed with demo data" "farmos::farm::seed_demo"

# IoT integration commands group
cli::register_command_group "iot" "IoT sensor integration"
cli::register_subcommand "iot" "connect" "Connect to MQTT broker" "farmos::iot::connect"
cli::register_subcommand "iot" "ingest" "Ingest sensor data" "farmos::iot::ingest"
cli::register_subcommand "iot" "sync" "Sync sensor data to farmOS" "farmos::iot::sync"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi