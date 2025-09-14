#!/usr/bin/env bash
################################################################################
# FFmpeg Resource CLI - v2.0 Universal Contract Compliant
# 
# Universal media processing framework for video and audio manipulation
#
# Usage:
#   resource-ffmpeg <command> [options]
#   resource-ffmpeg <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    FFMPEG_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${FFMPEG_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
FFMPEG_CLI_DIR="${APP_ROOT}/resources/ffmpeg"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${FFMPEG_CLI_DIR}/config/defaults.sh"

# Source FFmpeg libraries
for lib in core hardware install status inject start stop uninstall content test; do
    lib_file="${FFMPEG_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "ffmpeg" "FFmpeg universal media processing framework" "v2"

# ==============================================================================
# REQUIRED HANDLERS - Direct mapping to ffmpeg implementations
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="ffmpeg_install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="ffmpeg_uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="ffmpeg_start"  
CLI_COMMAND_HANDLERS["manage::stop"]="ffmpeg_stop"
CLI_COMMAND_HANDLERS["manage::restart"]="ffmpeg_start"  # Restart = start for CLI tools

# Test handlers - resource health validation
CLI_COMMAND_HANDLERS["test::smoke"]="ffmpeg::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="ffmpeg::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="ffmpeg::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="ffmpeg::test::all"

# Content handlers - media processing functionality
CLI_COMMAND_HANDLERS["content::add"]="ffmpeg::content::add"
CLI_COMMAND_HANDLERS["content::list"]="ffmpeg::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="ffmpeg::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="ffmpeg::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="ffmpeg::inject::process_single"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "info" "Show resource runtime information" "ffmpeg::info"
cli::register_command "status" "Show detailed resource status" "ffmpeg::status"
cli::register_command "logs" "Show FFmpeg logs" "ffmpeg::logs"

# ==============================================================================
# FFMPEG-SPECIFIC CUSTOM COMMANDS
# ==============================================================================
# Custom top-level commands for media processing shortcuts
cli::register_command "media-info" "Get media file information" "ffmpeg::media_info"
cli::register_command "transcode" "Convert media format" "ffmpeg::transcode"
cli::register_command "extract" "Extract audio/video/frames" "ffmpeg::extract"

# Custom content subcommands for advanced media operations
cli::register_subcommand "content" "process" "Process media with custom options" "ffmpeg::inject::batch_process" "modifies-system"
cli::register_subcommand "content" "benchmark" "Benchmark encoding performance" "ffmpeg::hardware::benchmark"

# Preset commands group
cli::register_command_group "preset" "Preset conversion management"
cli::register_subcommand "preset" "list" "List available presets" "ffmpeg::preset::list"
cli::register_subcommand "preset" "apply" "Apply a preset to a file" "ffmpeg::preset::apply"

# Stream processing commands group
cli::register_command_group "stream" "Stream processing and transcoding"
cli::register_subcommand "stream" "capture" "Capture stream to file" "ffmpeg::stream::capture"
cli::register_subcommand "stream" "transcode" "Transcode live stream" "ffmpeg::stream::transcode"
cli::register_subcommand "stream" "info" "Get stream information" "ffmpeg::stream::info"

# Web interface and API commands group
cli::register_command_group "web" "Web interface and API server"
cli::register_subcommand "web" "start" "Start web interface server" "ffmpeg::web::start"
cli::register_subcommand "web" "stop" "Stop web interface server" "ffmpeg::web::stop"
cli::register_subcommand "web" "status" "Check web server status" "ffmpeg::web::status"

# Performance monitoring commands group
cli::register_command_group "monitor" "Performance monitoring and metrics"
cli::register_subcommand "monitor" "start" "Start performance monitoring" "ffmpeg::monitor::start"
cli::register_subcommand "monitor" "stop" "Stop performance monitoring" "ffmpeg::monitor::stop"
cli::register_subcommand "monitor" "status" "Get current metrics" "ffmpeg::monitor::status"
cli::register_subcommand "monitor" "report" "Generate performance report" "ffmpeg::monitor::report"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi