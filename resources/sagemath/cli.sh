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
for lib in common docker install status content test health mathematics gpu; do
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
cli::register_subcommand "test" "parallel" "Test parallel processing capabilities" "sagemath::test::parallel"

# ==============================================================================
# GPU ACCELERATION COMMANDS
# ==============================================================================
cli::register_command "gpu" "GPU acceleration operations" "sagemath::gpu"
cli::register_subcommand "gpu" "check" "Check GPU availability" "sagemath::gpu::check"
cli::register_subcommand "gpu" "enable" "Enable GPU acceleration" "sagemath::gpu::enable"
cli::register_subcommand "gpu" "compute" "Run GPU-accelerated computation" "sagemath::gpu::compute"
cli::register_subcommand "gpu" "benchmark" "Benchmark GPU vs CPU" "sagemath::gpu::benchmark"

# ==============================================================================
# PARALLEL COMPUTING COMMANDS
# ==============================================================================
cli::register_command "parallel" "Parallel computing operations" "sagemath::parallel"
cli::register_subcommand "parallel" "compute" "Run parallel computation" "sagemath::parallel::compute"
cli::register_subcommand "parallel" "status" "Check parallel computing status" "sagemath::parallel::status"

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

# ==============================================================================
# GPU HANDLER FUNCTIONS
# ==============================================================================
sagemath::gpu() {
    echo "GPU acceleration operations:"
    echo "  check     - Check GPU availability"
    echo "  enable    - Enable GPU acceleration in container"
    echo "  compute   - Run GPU-accelerated computation"
    echo "  benchmark - Benchmark GPU vs CPU performance"
}

sagemath::gpu::check() {
    "${SAGEMATH_CLI_DIR}/lib/gpu.sh" check
}

sagemath::gpu::enable() {
    "${SAGEMATH_CLI_DIR}/lib/gpu.sh" enable
}

sagemath::gpu::compute() {
    local code="${1:-}"
    if [[ -z "$code" ]]; then
        error "Usage: resource-sagemath gpu compute \"<code>\""
        exit 1
    fi
    "${SAGEMATH_CLI_DIR}/lib/gpu.sh" compute "$code"
}

sagemath::gpu::benchmark() {
    "${SAGEMATH_CLI_DIR}/lib/gpu.sh" benchmark
}

# ==============================================================================
# PARALLEL HANDLER FUNCTIONS
# ==============================================================================
sagemath::parallel() {
    echo "Parallel computing operations:"
    echo "  compute - Run parallel computation"
    echo "  status  - Check parallel computing capabilities"
}

sagemath::parallel::compute() {
    local code="${1:-}"
    local cores="${2:-4}"
    if [[ -z "$code" ]]; then
        error "Usage: resource-sagemath parallel compute \"<code>\" [num_cores]"
        exit 1
    fi
    "${SAGEMATH_CLI_DIR}/lib/gpu.sh" parallel "$code" "$cores"
}

sagemath::parallel::status() {
    local status=$(docker exec "$SAGEMATH_CONTAINER_NAME" python3 -c 'import multiprocessing; print(multiprocessing.cpu_count())')
    echo "Available CPU cores: ${status}"
    
    # Check if parallel sage is available
    local parallel_check=$(docker exec "$SAGEMATH_CONTAINER_NAME" sage -c 'from sage.parallel.decorate import parallel; print("Parallel computing available")')
    echo "${parallel_check}"
}

# Test parallel processing
sagemath::test::parallel() {
    echo "Testing parallel processing capabilities..."
    
    # Test parallel computation
    local test_code='
@parallel
def compute_prime(n):
    return n, is_prime(n)

results = list(compute_prime([10^6 + i for i in range(10)]))
for r in results:
    print(f"Prime check {r[0][0]}: {r[1]}")
'
    
    "${SAGEMATH_CLI_DIR}/lib/gpu.sh" parallel "$test_code" 4
    echo "Parallel processing test complete!"
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi