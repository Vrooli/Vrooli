#!/usr/bin/env bash
################################################################################
# KiCad Resource CLI - v2.0 Universal Contract Compliant
# 
# Electronic Design Automation (EDA) for PCB design
#
# Usage:
#   resource-kicad <command> [options]
#   resource-kicad <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    KICAD_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${KICAD_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
KICAD_CLI_DIR="${APP_ROOT}/resources/kicad"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${KICAD_CLI_DIR}/config/defaults.sh"

# Source KiCad libraries
for lib in common core install status inject test content desktop version backup simulation autoroute; do
    lib_file="${KICAD_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "kicad" "Electronic Design Automation for PCB design" "v2"

# Override default handlers for KiCad (desktop application, not Docker-based)
CLI_COMMAND_HANDLERS["manage::install"]="kicad::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="kicad::desktop::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="kicad::desktop::start"
CLI_COMMAND_HANDLERS["manage::stop"]="kicad::desktop::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="kicad::desktop::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="kicad::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="kicad::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="kicad::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="kicad::test::all"

# Content handlers for PCB design functionality
CLI_COMMAND_HANDLERS["content::add"]="kicad::inject"
CLI_COMMAND_HANDLERS["content::list"]="kicad::content::list"
CLI_COMMAND_HANDLERS["content::get"]="kicad::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="kicad::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="kicad::content::execute"

# Add KiCad-specific content subcommands
cli::register_subcommand "content" "export" "Export PCB project to various formats" "kicad::export::project"
cli::register_subcommand "content" "projects" "List all KiCad projects" "kicad::content::list_projects"
cli::register_subcommand "content" "libraries" "List all KiCad libraries" "kicad::content::list_libraries"

# Add version control commands as a regular command with subcommands
cli::register_command "version" "Git version control for KiCad projects" "kicad::version::help"
CLI_COMMAND_HANDLERS["version::init"]="kicad::git::init"
CLI_COMMAND_HANDLERS["version::status"]="kicad::git::status"
CLI_COMMAND_HANDLERS["version::commit"]="kicad::git::commit"
CLI_COMMAND_HANDLERS["version::log"]="kicad::git::log"
CLI_COMMAND_HANDLERS["version::backup"]="kicad::git::backup"

# Version help function
kicad::version::help() {
    echo "KiCad Version Control Commands:"
    echo "  init <project>     - Initialize git repository for project"
    echo "  status <project>   - Show git status for project"
    echo "  commit <project>   - Commit project changes"
    echo "  log <project>      - Show commit history"
    echo "  backup <project>   - Create backup branch"
}

# Add cloud backup commands
cli::register_command "backup" "Cloud backup management for KiCad projects" "kicad::backup::help"
cli::register_subcommand "backup" "cloud" "Backup project to cloud" "kicad::backup::cloud"
cli::register_subcommand "backup" "list" "List available backups" "kicad::backup::list"
cli::register_subcommand "backup" "restore" "Restore from backup" "kicad::backup::restore"
cli::register_subcommand "backup" "schedule" "Schedule automatic backups" "kicad::backup::schedule"

# Backup help function
kicad::backup::help() {
    echo "KiCad Cloud Backup Commands (Minio):"
    echo "  cloud <project>    - Backup project to cloud storage"
    echo "  list [project]     - List available backups"
    echo "  restore <project>  - Restore project from backup"
    echo "  schedule <project> - Schedule automatic backups"
}

# Add SPICE simulation commands
cli::register_command "simulation" "SPICE circuit simulation" "kicad::simulation::help"
cli::register_subcommand "simulation" "extract" "Extract SPICE netlist from schematic" "kicad::simulation::extract_netlist"
cli::register_subcommand "simulation" "run" "Run SPICE simulation" "kicad::simulation::run"
cli::register_subcommand "simulation" "interactive" "Interactive SPICE shell" "kicad::simulation::interactive"
cli::register_subcommand "simulation" "report" "Generate simulation report" "kicad::simulation::report"
cli::register_subcommand "simulation" "models" "Create SPICE models library" "kicad::simulation::create_models"

# Simulation help function
kicad::simulation::help() {
    echo "KiCad SPICE Simulation Commands:"
    echo "  extract <schematic>   - Extract SPICE netlist from schematic"
    echo "  run <netlist> [type]  - Run simulation (tran/dc/ac/op)"
    echo "  interactive <netlist> - Interactive SPICE shell"
    echo "  report <project>      - Generate simulation report"
    echo "  models                - Create SPICE models library"
}

# Add auto-routing commands
cli::register_command "autoroute" "Automated PCB trace routing" "kicad::autoroute::help"
cli::register_subcommand "autoroute" "export" "Export board for auto-routing" "kicad::autoroute::export_dsn"
cli::register_subcommand "autoroute" "run" "Run auto-router" "kicad::autoroute::run"
cli::register_subcommand "autoroute" "import" "Import routed board" "kicad::autoroute::import_ses"
cli::register_subcommand "autoroute" "optimize" "Auto-route with optimization" "kicad::autoroute::optimize"
cli::register_subcommand "autoroute" "assistant" "Interactive routing assistant" "kicad::autoroute::assistant"

# Autoroute help function
kicad::autoroute::help() {
    echo "KiCad Auto-routing Commands:"
    echo "  export <board>        - Export board for auto-routing"
    echo "  run <dsn> [options]   - Run auto-router"
    echo "  import <board> <ses>  - Import routed board"
    echo "  optimize <board>      - Auto-route with optimization"
    echo "  assistant <board>     - Interactive routing assistant"
}

# Information commands
cli::register_command "info" "Show resource runtime information" "kicad::info"
cli::register_command "status" "Show detailed KiCad status" "kicad_status"
cli::register_command "logs" "Show KiCad logs" "kicad::desktop::logs"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi