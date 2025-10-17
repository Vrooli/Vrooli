#!/usr/bin/env bash
################################################################################
# Twilio Resource CLI - v2.0 Universal Contract Compliant
# 
# Cloud communications platform for SMS, voice, and video
#
# Usage:
#   resource-twilio <command> [options]
#   resource-twilio <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    TWILIO_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${TWILIO_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
TWILIO_CLI_DIR="${APP_ROOT}/resources/twilio"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${TWILIO_CLI_DIR}/config/defaults.sh"

# Source Twilio libraries
for lib in common core config inject install lifecycle logs numbers sms history templates audit analytics voice whatsapp start status stop test; do
    lib_file="${TWILIO_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "twilio" "Twilio cloud communications platform management" "v2"

# Override default handlers to point directly to twilio implementations
CLI_COMMAND_HANDLERS["manage::install"]="twilio::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="twilio::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="twilio::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="twilio::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="twilio::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="twilio::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="twilio::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="twilio::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="twilio::test::all"

# Override content handlers for Twilio-specific messaging functionality
CLI_COMMAND_HANDLERS["content::add"]="twilio::inject"
CLI_COMMAND_HANDLERS["content::list"]="twilio::list_injected" 
CLI_COMMAND_HANDLERS["content::get"]="twilio::list_numbers"
CLI_COMMAND_HANDLERS["content::remove"]="twilio::content::remove_placeholder"
CLI_COMMAND_HANDLERS["content::execute"]="twilio::send_sms"

# Add Twilio-specific content subcommands not in the standard framework
cli::register_subcommand "content" "send-sms" "Send SMS message" "twilio::send_sms"
cli::register_subcommand "content" "send-bulk" "Send bulk SMS to multiple recipients" "twilio::send_bulk_sms"
cli::register_subcommand "content" "send-from-file" "Send SMS from CSV file" "twilio::send_sms_from_file"
cli::register_subcommand "content" "numbers" "List phone numbers" "twilio::list_numbers"

# Add history management commands
cli::register_subcommand "content" "history" "List message history" "twilio::history::list"
cli::register_subcommand "content" "history-stats" "Show message statistics" "twilio::history::stats"
cli::register_subcommand "content" "history-export" "Export message history to CSV" "twilio::history::export_csv"
cli::register_subcommand "content" "history-update" "Update delivery statuses from API" "twilio::history::update_pending_statuses"
cli::register_subcommand "content" "history-clear" "Clear message history (requires --force)" "twilio::history::clear"

# Add template management commands
cli::register_subcommand "content" "template-create" "Create a message template" "twilio::templates::create"
cli::register_subcommand "content" "template-list" "List all templates" "twilio::templates::list"
cli::register_subcommand "content" "template-get" "Get a specific template" "twilio::templates::get"
cli::register_subcommand "content" "template-update" "Update a template" "twilio::templates::update"
cli::register_subcommand "content" "template-delete" "Delete a template" "twilio::templates::delete"
cli::register_subcommand "content" "template-send" "Send SMS using a template" "twilio::templates::send"
cli::register_subcommand "content" "template-send-bulk" "Send bulk SMS using template and CSV" "twilio::templates::send_bulk"

# Add audit commands
cli::register_subcommand "content" "audit-search" "Search audit logs" "twilio::audit::search"
cli::register_subcommand "content" "audit-stats" "Show audit statistics" "twilio::audit::stats"
cli::register_subcommand "content" "audit-archive" "Archive old audit logs" "twilio::audit::archive"

# Add analytics dashboard commands
cli::register_subcommand "content" "dashboard" "Show analytics dashboard" "twilio::analytics::dashboard"
cli::register_subcommand "content" "analytics-export" "Export analytics report" "twilio::analytics::export_report"
cli::register_subcommand "content" "analytics-costs" "Show daily cost breakdown" "twilio::analytics::daily_costs"
cli::register_subcommand "content" "analytics-performance" "Show performance metrics" "twilio::analytics::performance"

# Add voice call commands
cli::register_subcommand "content" "make-call" "Make a voice call" "make_call"
cli::register_subcommand "content" "make-conference" "Create a conference call" "make_conference"
cli::register_subcommand "content" "voice-history" "List voice call history" "get_voice_history"
cli::register_subcommand "content" "voice-stats" "Show voice call statistics" "get_voice_stats"
cli::register_subcommand "content" "voice-status" "Update call status from API" "update_call_status"
cli::register_subcommand "content" "voice-export" "Export voice history to CSV" "export_voice_history"

# Add WhatsApp commands
cli::register_subcommand "content" "send-whatsapp" "Send WhatsApp message" "send_whatsapp"
cli::register_subcommand "content" "send-whatsapp-template" "Send WhatsApp template message" "send_whatsapp_template"
cli::register_subcommand "content" "send-whatsapp-bulk" "Send bulk WhatsApp messages" "send_whatsapp_bulk"
cli::register_subcommand "content" "whatsapp-history" "List WhatsApp message history" "get_whatsapp_history"
cli::register_subcommand "content" "whatsapp-stats" "Show WhatsApp statistics" "get_whatsapp_stats"
cli::register_subcommand "content" "whatsapp-status" "Update WhatsApp message status" "update_whatsapp_status"
cli::register_subcommand "content" "whatsapp-export" "Export WhatsApp history to CSV" "export_whatsapp_history"
cli::register_subcommand "content" "whatsapp-setup" "WhatsApp sandbox setup guide" "setup_whatsapp_sandbox"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "twilio::status::new"
cli::register_command "logs" "Show Twilio logs" "twilio::docker::logs"
cli::register_command "config" "View/update Twilio configuration" "twilio::config"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi