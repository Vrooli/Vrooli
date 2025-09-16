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
for lib in common docker install status content test health mathematics gpu export; do
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
cli::register_subcommand "content" "create" "Create new Jupyter notebook from template" "sagemath::content::create"

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

# Register GPU handlers
CLI_COMMAND_HANDLERS["gpu::check"]="sagemath::gpu::check"
CLI_COMMAND_HANDLERS["gpu::enable"]="sagemath::gpu::enable"
CLI_COMMAND_HANDLERS["gpu::compute"]="sagemath::gpu::compute"
CLI_COMMAND_HANDLERS["gpu::benchmark"]="sagemath::gpu::benchmark"

# ==============================================================================
# PARALLEL COMPUTING COMMANDS
# ==============================================================================
cli::register_command "parallel" "Parallel computing operations" "sagemath::parallel"
cli::register_subcommand "parallel" "compute" "Run parallel computation" "sagemath::parallel::compute"
cli::register_subcommand "parallel" "status" "Check parallel computing status" "sagemath::parallel::status"

# Register parallel handlers
CLI_COMMAND_HANDLERS["parallel::compute"]="sagemath::parallel::compute"
CLI_COMMAND_HANDLERS["parallel::status"]="sagemath::parallel::status"

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
# CACHE MANAGEMENT COMMANDS
# ==============================================================================
cli::register_command "cache" "Cache management operations" "sagemath::cache"
cli::register_subcommand "cache" "clear" "Clear all cached results" "sagemath::cache::clear"
cli::register_subcommand "cache" "stats" "Show cache statistics" "sagemath::cache::stats"

# Register cache handlers
CLI_COMMAND_HANDLERS["cache::clear"]="sagemath::cache::clear"
CLI_COMMAND_HANDLERS["cache::stats"]="sagemath::cache::stats"

# ==============================================================================
# VISUALIZATION COMMANDS
# ==============================================================================
cli::register_command "plot" "Create mathematical plots" "sagemath::plot"
cli::register_subcommand "plot" "2d" "Create 2D plot" "sagemath::plot::2d"
cli::register_subcommand "plot" "3d" "Create 3D plot" "sagemath::plot::3d"
cli::register_subcommand "plot" "parametric" "Create parametric plot" "sagemath::plot::parametric"
cli::register_subcommand "plot" "polar" "Create polar plot" "sagemath::plot::polar"

# Register plot handlers
CLI_COMMAND_HANDLERS["plot::2d"]="sagemath::plot::2d"
CLI_COMMAND_HANDLERS["plot::3d"]="sagemath::plot::3d"
CLI_COMMAND_HANDLERS["plot::parametric"]="sagemath::plot::parametric"
CLI_COMMAND_HANDLERS["plot::polar"]="sagemath::plot::polar"

# Register export command and subcommands
cli::register_command "export" "Export mathematical expressions" "sagemath::export"
cli::register_subcommand "export" "latex" "Export to LaTeX format" "sagemath::export::latex"
cli::register_subcommand "export" "mathml" "Export to MathML format" "sagemath::export::mathml"
cli::register_subcommand "export" "image" "Render equation to PNG image" "sagemath::export::image"
cli::register_subcommand "export" "all" "Export to all formats" "sagemath::export::all"
cli::register_subcommand "export" "formats" "List available formats" "sagemath::export::formats"

# Register export handlers
CLI_COMMAND_HANDLERS["export::latex"]="sagemath::export::latex"
CLI_COMMAND_HANDLERS["export::mathml"]="sagemath::export::mathml"
CLI_COMMAND_HANDLERS["export::image"]="sagemath::export::image"
CLI_COMMAND_HANDLERS["export::all"]="sagemath::export::all"
CLI_COMMAND_HANDLERS["export::formats"]="sagemath::export::formats"

# ==============================================================================
# EXPORT HANDLER FUNCTIONS
# ==============================================================================
sagemath::export() {
    echo "Export operations:"
    echo "  latex   - Export to LaTeX format"
    echo "  mathml  - Export to MathML format"
    echo "  image   - Render equation to PNG image"
    echo "  all     - Export to all formats"
    echo "  formats - List available export formats"
}

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
# CACHE HANDLER FUNCTIONS
# ==============================================================================
sagemath::cache() {
    echo "Cache management operations:"
    echo "  clear  - Clear all cached results"
    echo "  stats  - Show cache statistics"
}

sagemath::cache::clear() {
    echo "Clearing SageMath calculation cache..."
    local cache_dir="/home/matthalloran8/Vrooli/data/resources/sagemath/cache"
    if [ -d "$cache_dir" ]; then
        local count=$(find "$cache_dir" -name "*.cache" -type f 2>/dev/null | wc -l)
        rm -f "$cache_dir"/*.cache 2>/dev/null || true
        echo "✅ Cleared $count cached results"
    else
        echo "✅ No cache to clear"
    fi
}

sagemath::cache::stats() {
    local cache_dir="/home/matthalloran8/Vrooli/data/resources/sagemath/cache"
    if [ -d "$cache_dir" ]; then
        local count=$(find "$cache_dir" -name "*.cache" -type f 2>/dev/null | wc -l)
        local size=$(du -sh "$cache_dir" 2>/dev/null | cut -f1)
        local fresh=$(find "$cache_dir" -name "*.cache" -type f -mmin -60 2>/dev/null | wc -l)
        local stale=$(find "$cache_dir" -name "*.cache" -type f -mmin +60 2>/dev/null | wc -l)
        echo "Cache Statistics:"
        echo "  Total entries: $count"
        echo "  Fresh (<1hr): $fresh"
        echo "  Stale (>1hr): $stale"
        echo "  Total size: $size"
    else
        echo "Cache not initialized"
    fi
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

# ==============================================================================
# PLOT HANDLER FUNCTIONS
# ==============================================================================
sagemath::plot() {
    echo "Plot creation operations:"
    echo "  2d         - Create 2D plot"
    echo "  3d         - Create 3D plot"
    echo "  parametric - Create parametric plot"
    echo "  polar      - Create polar plot"
}

sagemath::plot::2d() {
    local expression="${1:-}"
    local xmin="${2:--5}"
    local xmax="${3:-5}"
    local title="${4:-2D Plot}"
    
    if [[ -z "$expression" ]]; then
        echo "Usage: resource-sagemath plot 2d \"expression\" [xmin] [xmax] [title]"
        echo "Example: resource-sagemath plot 2d \"sin(x) + cos(2*x)\" -pi pi \"Trig Functions\""
        return 1
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_file="/home/matthalloran8/Vrooli/data/resources/sagemath/outputs/plot_2d_${timestamp}.png"
    
    local code="
import matplotlib
matplotlib.use('Agg')
p = plot($expression, (x, $xmin, $xmax), title='$title', legend_label='f(x)')
p.save('$output_file')
print('Plot saved to: $output_file')
"
    sagemath::content::calculate "$code"
    echo "📊 Plot saved: $output_file"
}

sagemath::plot::3d() {
    local expression="${1:-}"
    local xrange="${2:-(-5,5)}"
    local yrange="${3:-(-5,5)}"
    local title="${4:-3D Plot}"
    
    if [[ -z "$expression" ]]; then
        echo "Usage: resource-sagemath plot 3d \"expression\" [xrange] [yrange] [title]"
        echo "Example: resource-sagemath plot 3d \"x^2 + y^2\" \"(-3,3)\" \"(-3,3)\" \"Paraboloid\""
        return 1
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_file="/home/matthalloran8/Vrooli/data/resources/sagemath/outputs/plot_3d_${timestamp}.png"
    
    local code="
import matplotlib
matplotlib.use('Agg')
var('x,y')
p = plot3d($expression, (x, $xrange), (y, $yrange))
p.save('$output_file')
print('3D plot saved to: $output_file')
"
    sagemath::content::calculate "$code"
    echo "📊 3D plot saved: $output_file"
}

sagemath::plot::parametric() {
    local x_expr="${1:-}"
    local y_expr="${2:-}"
    local tmin="${3:-0}"
    local tmax="${4:-2*pi}"
    local title="${5:-Parametric Plot}"
    
    if [[ -z "$x_expr" ]] || [[ -z "$y_expr" ]]; then
        echo "Usage: resource-sagemath plot parametric \"x(t)\" \"y(t)\" [tmin] [tmax] [title]"
        echo "Example: resource-sagemath plot parametric \"cos(t)\" \"sin(t)\" 0 \"2*pi\" \"Circle\""
        return 1
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_file="/home/matthalloran8/Vrooli/data/resources/sagemath/outputs/plot_parametric_${timestamp}.png"
    
    local code="
import matplotlib
matplotlib.use('Agg')
var('t')
p = parametric_plot(($x_expr, $y_expr), (t, $tmin, $tmax), title='$title')
p.save('$output_file')
print('Parametric plot saved to: $output_file')
"
    sagemath::content::calculate "$code"
    echo "📊 Parametric plot saved: $output_file"
}

sagemath::plot::polar() {
    local r_expr="${1:-}"
    local theta_min="${2:-0}"
    local theta_max="${3:-2*pi}"
    local title="${4:-Polar Plot}"
    
    if [[ -z "$r_expr" ]]; then
        echo "Usage: resource-sagemath plot polar \"r(theta)\" [theta_min] [theta_max] [title]"
        echo "Example: resource-sagemath plot polar \"1 + cos(theta)\" 0 \"2*pi\" \"Cardioid\""
        return 1
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_file="/home/matthalloran8/Vrooli/data/resources/sagemath/outputs/plot_polar_${timestamp}.png"
    
    local code="
import matplotlib
matplotlib.use('Agg')
var('theta')
p = polar_plot($r_expr, (theta, $theta_min, $theta_max), title='$title')
p.save('$output_file')
print('Polar plot saved to: $output_file')
"
    sagemath::content::calculate "$code"
    echo "📊 Polar plot saved: $output_file"
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi