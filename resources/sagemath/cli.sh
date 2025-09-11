#!/usr/bin/env bash
################################################################################
# SageMath Resource CLI - v2.0 Universal Contract Compliant
# 
# Open-source mathematics software system for symbolic and numerical computation
#
# Usage:
#   resource-sagemath <command> [options]
#   resource-sagemath <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    SAGEMATH_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${SAGEMATH_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
SAGEMATH_CLI_DIR="${APP_ROOT}/resources/sagemath"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${SAGEMATH_CLI_DIR}/config/defaults.sh"

# Source SageMath libraries
for lib in common docker install status content test health mathematics; do
    lib_file="${SAGEMATH_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "sagemath" "SageMath mathematical computation system" "v2"

# ==============================================================================
# REQUIRED HANDLERS - Universal Contract v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="sagemath::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="sagemath::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="sagemath::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="sagemath::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="sagemath::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="sagemath::test::smoke"
CLI_COMMAND_HANDLERS["test::unit"]="sagemath::test::unit"
CLI_COMMAND_HANDLERS["test::integration"]="sagemath::test::integration"
CLI_COMMAND_HANDLERS["test::all"]="sagemath::test::all"

# Content handlers for mathematical computation functionality
CLI_COMMAND_HANDLERS["content::add"]="sagemath::content::add"
CLI_COMMAND_HANDLERS["content::list"]="sagemath::content::list"
CLI_COMMAND_HANDLERS["content::get"]="sagemath::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="sagemath::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="sagemath::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed SageMath status" "sagemath::status"
cli::register_command "logs" "Show SageMath container logs" "sagemath::docker::logs"
cli::register_command "health" "Check SageMath health status" "sagemath::health::check"

# ==============================================================================
# SAGEMATH-SPECIFIC CONTENT COMMANDS
# ==============================================================================
# Mathematical computation specific operations
cli::register_subcommand "content" "calculate" "Calculate mathematical expression" "sagemath::content::calculate"
cli::register_subcommand "content" "notebook" "Open Jupyter notebook interface" "sagemath::content::notebook"

# SageMath-specific test commands
cli::register_subcommand "test" "performance" "Run performance benchmarks" "sagemath::test::performance"

# ==============================================================================
# MATHEMATICAL OPERATIONS AS DIRECT COMMANDS
# ==============================================================================
# Register mathematical operations as direct commands
cli::register_command "solve" "Solve equations" "sagemath::math::solve"
cli::register_command "differentiate" "Differentiate expressions" "sagemath::math::differentiate"
cli::register_command "integrate" "Integrate expressions" "sagemath::math::integrate"
cli::register_command "matrix" "Matrix operations" "sagemath::math::matrix"
cli::register_command "prime" "Prime number operations" "sagemath::math::prime"
cli::register_command "stats" "Statistical operations" "sagemath::math::stats"
cli::register_command "polynomial" "Polynomial operations" "sagemath::math::polynomial"
cli::register_command "complex" "Complex number operations" "sagemath::math::complex"
cli::register_command "limit" "Calculate limits" "sagemath::math::limit"
cli::register_command "series" "Series expansions" "sagemath::math::series"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi