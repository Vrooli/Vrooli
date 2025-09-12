#!/bin/bash
# Earthly Resource - Core Functionality
# Implements v2.0 contract requirements

set -euo pipefail

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Show resource information from runtime.json
show_info() {
    local arg="${1:-}"
    local runtime_file="${SCRIPT_DIR}/config/runtime.json"
    
    if [[ ! -f "${runtime_file}" ]]; then
        log_error "Runtime configuration not found: ${runtime_file}"
        return 1
    fi
    
    # Check for invalid flags
    if [[ -n "${arg}" && "${arg}" != "--json" ]]; then
        log_error "Invalid option: ${arg}"
        echo "Usage: info [--json]"
        return 1
    fi
    
    if [[ "${arg}" == "--json" ]]; then
        cat "${runtime_file}"
    else
        echo "🏗️ Earthly Resource Information"
        echo "================================="
        jq -r '
            "Name: \(.name)",
            "Version: \(.version)",
            "Description: \(.description)",
            "Category: \(.category)",
            "Priority: \(.priority)",
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Startup Timeout: \(.startup_timeout)s",
            "Startup Time Estimate: \(.startup_time_estimate)",
            "Recovery Attempts: \(.recovery_attempts)"
        ' "${runtime_file}"
    fi
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        install)
            install_earthly "$@"
            ;;
        uninstall)
            uninstall_earthly "$@"
            ;;
        start)
            start_earthly "$@"
            ;;
        stop)
            stop_earthly "$@"
            ;;
        restart)
            stop_earthly "$@" && start_earthly "$@"
            ;;
        *)
            log_error "Unknown manage subcommand: ${subcommand}"
            echo "Available subcommands: install, uninstall, start, stop, restart"
            return 1
            ;;
    esac
}

# Install Earthly
install_earthly() {
    log_info "Installing Earthly v${EARTHLY_VERSION}..."
    
    # Check if already installed
    if command -v earthly &> /dev/null; then
        local installed_version=$(earthly --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        if [[ "${installed_version}" == "${EARTHLY_VERSION}" ]]; then
            log_info "Earthly v${EARTHLY_VERSION} already installed"
            return 2  # Already installed
        fi
    fi
    
    # Check Docker dependency
    if ! command -v docker &> /dev/null; then
        log_error "Docker is required but not installed"
        return 1
    fi
    
    # Create directories
    mkdir -p "${EARTHLY_HOME}" "${EARTHLY_CONFIG_DIR}" "${EARTHLY_CACHE_DIR}" \
             "${EARTHLY_ARTIFACTS_DIR}" "${EARTHLY_LOGS_DIR}"
    
    # Download and install Earthly
    log_info "Downloading Earthly from ${EARTHLY_DOWNLOAD_URL}..."
    
    # Try to install in user directory if system install fails
    local install_dir="${EARTHLY_INSTALL_DIR}"
    if [[ ! -w "${install_dir}" ]]; then
        install_dir="${HOME}/.local/bin"
        mkdir -p "${install_dir}"
        log_info "Installing to user directory: ${install_dir}"
    fi
    
    if ! wget -q "${EARTHLY_DOWNLOAD_URL}" -O "${install_dir}/earthly"; then
        log_error "Failed to download Earthly"
        return 1
    fi
    
    chmod +x "${install_dir}/earthly"
    
    # Add to PATH if needed
    if [[ "${install_dir}" == "${HOME}/.local/bin" ]] && ! echo "$PATH" | grep -q "${install_dir}"; then
        export PATH="${install_dir}:${PATH}"
        log_info "Added ${install_dir} to PATH for this session"
    fi
    
    # Bootstrap Earthly (skip if it fails due to permissions)
    log_info "Bootstrapping Earthly..."
    if ! "${install_dir}/earthly" bootstrap --with-autocomplete 2>/dev/null; then
        log_warn "Earthly bootstrap skipped (requires Docker access)"
    fi
    
    # Create default configuration
    create_default_config
    
    log_info "Earthly v${EARTHLY_VERSION} installed successfully"
    return 0
}

# Uninstall Earthly
uninstall_earthly() {
    local keep_data="${1:-}"
    
    log_info "Uninstalling Earthly..."
    
    # Stop any running builds
    stop_earthly
    
    # Remove binary from system or user directory
    if [[ -f "${EARTHLY_INSTALL_DIR}/earthly" ]]; then
        rm -f "${EARTHLY_INSTALL_DIR}/earthly" 2>/dev/null || sudo rm -f "${EARTHLY_INSTALL_DIR}/earthly"
    fi
    if [[ -f "${HOME}/.local/bin/earthly" ]]; then
        rm -f "${HOME}/.local/bin/earthly"
    fi
    
    # Remove data if not keeping
    if [[ "${keep_data}" != "--keep-data" ]]; then
        log_info "Removing Earthly data..."
        rm -rf "${EARTHLY_HOME}"
    else
        log_info "Keeping Earthly data in ${EARTHLY_HOME}"
    fi
    
    log_info "Earthly uninstalled successfully"
    return 0
}

# Start Earthly daemon
start_earthly() {
    log_info "Starting Earthly daemon..."
    
    # Earthly runs on-demand, no persistent daemon needed
    # Just verify it's ready
    if ! earthly --version &>/dev/null; then
        log_error "Earthly not installed"
        return 1
    fi
    
    # Ensure Docker is running
    if ! docker ps &>/dev/null; then
        log_error "Docker daemon is not running"
        return 1
    fi
    
    log_info "Earthly is ready"
    return 0
}

# Stop Earthly daemon
stop_earthly() {
    log_info "Stopping Earthly builds..."
    
    # Prune BuildKit cache if needed
    if command -v earthly &> /dev/null; then
        earthly prune --reset || true
    fi
    
    log_info "Earthly stopped"
    return 0
}

# Handle content subcommands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        add)
            add_content "$@"
            ;;
        list)
            list_content "$@"
            ;;
        get)
            get_content "$@"
            ;;
        remove)
            remove_content "$@"
            ;;
        execute)
            execute_build "$@"
            ;;
        clear)
            clear_cache "$@"
            ;;
        configure)
            configure_earthly "$@"
            ;;
        *)
            log_error "Unknown content subcommand: ${subcommand}"
            echo "Available subcommands: add, list, get, remove, execute, clear, configure"
            return 1
            ;;
    esac
}

# Add content (Earthfile or artifact)
add_content() {
    local file=""
    local type="earthfile"
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "${file}" ]]; then
        log_error "File path required: --file <path>"
        return 1
    fi
    
    if [[ ! -f "${file}" ]]; then
        log_error "File not found: ${file}"
        return 1
    fi
    
    if [[ "${type}" == "earthfile" ]]; then
        local dest="${EARTHLY_HOME}/Earthfile"
        if [[ -n "${name}" ]]; then
            dest="${EARTHLY_HOME}/${name}"
        fi
        cp "${file}" "${dest}"
        log_info "Earthfile added successfully to ${dest}"
    elif [[ "${type}" == "secret" && -n "${name}" ]]; then
        # Store secret securely
        mkdir -p "${EARTHLY_CONFIG_DIR}/secrets"
        echo "${file}" > "${EARTHLY_CONFIG_DIR}/secrets/${name}"
        chmod 600 "${EARTHLY_CONFIG_DIR}/secrets/${name}"
        log_info "Secret '${name}' stored securely"
    else
        cp "${file}" "${EARTHLY_ARTIFACTS_DIR}/"
        log_info "Artifact added: $(basename "${file}")"
    fi
    
    return 0
}

# List content
list_content() {
    local type="${1:-all}"
    local json_output="${2:-}"
    
    case "${type}" in
        artifacts)
            if [[ "${json_output}" == "--json" ]]; then
                echo "{"
                echo "  \"artifacts\": ["
                local first=true
                for file in "${EARTHLY_ARTIFACTS_DIR}"/*; do
                    if [[ -f "${file}" ]]; then
                        if [[ "${first}" != "true" ]]; then echo ","; fi
                        echo -n "    \"$(basename "${file}")\""
                        first=false
                    fi
                done
                echo ""
                echo "  ]"
                echo "}"
            else
                log_info "Build artifacts:"
                ls -la "${EARTHLY_ARTIFACTS_DIR}/" 2>/dev/null || echo "No artifacts found"
            fi
            ;;
        earthfiles)
            if [[ "${json_output}" == "--json" ]]; then
                echo "{"
                echo "  \"earthfiles\": ["
                local first=true
                for file in $(find "${EARTHLY_HOME}" -name "Earthfile*" -type f 2>/dev/null); do
                    if [[ "${first}" != "true" ]]; then echo ","; fi
                    echo -n "    \"${file}\""
                    first=false
                done
                echo ""
                echo "  ]"
                echo "}"
            else
                log_info "Earthfiles:"
                find "${EARTHLY_HOME}" -name "Earthfile*" -type f 2>/dev/null || echo "No Earthfiles found"
            fi
            ;;
        *)
            if [[ "${json_output}" == "--json" ]]; then
                echo "{"
                list_content earthfiles --json | sed 's/^{//' | sed 's/^}/,/'
                list_content artifacts --json | sed 's/^{//' | sed 's/^}//'
                echo "}"
            else
                list_content earthfiles
                echo ""
                list_content artifacts
            fi
            ;;
    esac
}

# Remove content
remove_content() {
    local type="${1:-}"
    local name="${2:-}"
    
    if [[ -z "${type}" || -z "${name}" ]]; then
        log_error "Usage: remove <type> <name>"
        echo "Types: artifact, earthfile"
        return 1
    fi
    
    case "${type}" in
        artifact)
            local artifact_path="${EARTHLY_ARTIFACTS_DIR}/${name}"
            if [[ -f "${artifact_path}" ]]; then
                rm -f "${artifact_path}"
                log_info "Removed artifact: ${name}"
            else
                log_error "Artifact not found: ${name}"
                return 1
            fi
            ;;
        earthfile)
            local earthfile_path="${EARTHLY_HOME}/${name}"
            if [[ -f "${earthfile_path}" ]]; then
                rm -f "${earthfile_path}"
                log_info "Removed Earthfile: ${name}"
            else
                log_error "Earthfile not found: ${name}"
                return 1
            fi
            ;;
        *)
            log_error "Unknown type: ${type}"
            echo "Valid types: artifact, earthfile"
            return 1
            ;;
    esac
    
    return 0
}

# Get specific content
get_content() {
    local type="${1:-}"
    local name="${2:-}"
    local output="${3:-}"
    
    if [[ "${type}" == "artifact" && -n "${name}" ]]; then
        local artifact_path="${EARTHLY_ARTIFACTS_DIR}/${name}"
        if [[ -f "${artifact_path}" ]]; then
            if [[ -n "${output}" ]]; then
                cp "${artifact_path}" "${output}"
                log_info "Artifact saved to: ${output}"
            else
                cat "${artifact_path}"
            fi
        else
            log_error "Artifact not found: ${name}"
            return 1
        fi
    elif [[ "${type}" == "metrics" ]]; then
        if [[ -f "${EARTHLY_LOGS_DIR}/metrics.csv" ]]; then
            echo "Build Performance Metrics:"
            echo "========================="
            echo "Target,Duration(s),Timestamp,Status"
            tail -20 "${EARTHLY_LOGS_DIR}/metrics.csv"
        else
            log_info "No metrics collected yet. Run builds to generate metrics."
        fi
    fi
}

# Execute build
execute_build() {
    local target="+build"
    local earthfile=""
    local platform=""
    local cache_opt=""
    local parallel_opt=""
    local metrics_opt=""
    local satellite_opt=""
    local no_cache_opt=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --target)
                target="$2"
                shift 2
                ;;
            --file)
                earthfile="$2"
                shift 2
                ;;
            --platform)
                platform="--platform=$2"
                shift 2
                ;;
            --cache)
                cache_opt="--use-inline-cache"
                shift
                ;;
            --parallel)
                parallel_opt="--max-parallel=${EARTHLY_PARALLEL_LIMIT}"
                shift
                ;;
            --metrics)
                metrics_opt="--verbose"
                shift
                ;;
            --satellite)
                satellite_opt="--sat=${EARTHLY_SATELLITE_NAME}"
                shift
                ;;
            --no-cache)
                no_cache_opt="--no-cache"
                shift
                ;;
            --dry-run)
                log_info "Dry run mode - would execute: earthly ${target}"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # If no file specified, try to find one
    if [[ -z "${earthfile}" ]]; then
        if [[ -f "./Earthfile" ]]; then
            earthfile="./Earthfile"
        elif [[ -f "${EARTHLY_HOME}/Earthfile" ]]; then
            earthfile="${EARTHLY_HOME}/Earthfile"
        else
            earthfile="${EARTHLY_DEFAULT_EARTHFILE}"
        fi
    fi
    
    if [[ ! -f "${earthfile}" ]]; then
        log_error "Earthfile not found: ${earthfile}"
        log_info "Use --file to specify the Earthfile location"
        return 1
    fi
    
    log_info "Executing build target: ${target} from ${earthfile}"
    
    # Ensure earthly is in PATH
    if ! command -v earthly &> /dev/null; then
        if [[ -f "${HOME}/.local/bin/earthly" ]]; then
            export PATH="${HOME}/.local/bin:${PATH}"
        else
            log_error "Earthly binary not found in PATH"
            return 1
        fi
    fi
    
    # Create build log directory
    mkdir -p "${EARTHLY_LOGS_DIR}"
    local build_log="${EARTHLY_LOGS_DIR}/build-$(date +%Y%m%d-%H%M%S).log"
    
    # Performance optimization: Set BuildKit settings
    export BUILDKIT_PROGRESS=plain
    export EARTHLY_BUILDKIT_CACHE_SIZE_MB="${EARTHLY_CACHE_SIZE_MB}"
    export EARTHLY_BUILDKIT_MAX_PARALLELISM="${EARTHLY_PARALLEL_LIMIT}"
    
    # Run the build with performance tracking
    local start_time=$(date +%s)
    
    # Run with optimized settings
    # Copy Earthfile to current directory as earthly expects it there
    local temp_earthfile=""
    if [[ "${earthfile}" != "./Earthfile" ]]; then
        cp "${earthfile}" ./Earthfile
        temp_earthfile="./Earthfile"
    fi
    
    if earthly ${platform} ${cache_opt} ${no_cache_opt} ${parallel_opt} ${metrics_opt} ${satellite_opt} \
            --config "${EARTHLY_CONFIG_DIR}/config.yml" \
            --buildkit-cache-size-mb "${EARTHLY_CACHE_SIZE_MB}" \
            --conversion-parallelism "${EARTHLY_PARALLEL_LIMIT}" \
            "${target}" 2>&1 | tee "${build_log}"; then
        
        local end_time=$(date +%s)
        local build_time=$((end_time - start_time))
        log_info "Build completed in ${build_time} seconds"
        log_info "Build log saved to: ${build_log}"
        
        # Cleanup temporary file
        if [[ -n "${temp_earthfile}" ]]; then
            rm -f "${temp_earthfile}"
        fi
        
        # Track metrics
        echo "${target},${build_time},$(date +%Y-%m-%d_%H:%M:%S),success" >> "${EARTHLY_LOGS_DIR}/metrics.csv"
        return 0
    else
        local end_time=$(date +%s)
        local build_time=$((end_time - start_time))
        log_error "Build failed after ${build_time} seconds"
        log_info "Build log saved to: ${build_log}"
        
        # Cleanup temporary file
        if [[ -n "${temp_earthfile}" ]]; then
            rm -f "${temp_earthfile}"
        fi
        
        # Track failure
        echo "${target},${build_time},$(date +%Y-%m-%d_%H:%M:%S),failed" >> "${EARTHLY_LOGS_DIR}/metrics.csv"
        return 1
    fi
}

# Clear cache
clear_cache() {
    log_info "Clearing Earthly cache..."
    earthly prune -a
    log_info "Cache cleared"
}

# Configure Earthly
configure_earthly() {
    local option="${1:-}"
    local value="${2:-}"
    
    case "${option}" in
        --remote-cache)
            log_info "Configuring remote cache..."
            if [[ -n "${value}" ]]; then
                export EARTHLY_REMOTE_CACHE_ENDPOINT="${value}"
                # Update config file
                cat >> "${EARTHLY_CONFIG_DIR}/config.yml" << EOF

remote_cache:
  address: ${value}
  push: true
  pull: true
EOF
                log_info "Remote cache configured: ${value}"
            else
                log_error "Remote cache endpoint required"
                return 1
            fi
            ;;
        --webhook)
            log_info "Configuring webhooks..."
            if [[ -n "${value}" ]]; then
                echo "${value}" >> "${EARTHLY_CONFIG_DIR}/webhooks.txt"
                log_info "Webhook added: ${value}"
            else
                log_error "Webhook URL required"
                return 1
            fi
            ;;
        --optimize-cache)
            log_info "Optimizing cache settings for CI/CD..."
            # Set aggressive caching for CI/CD
            export EARTHLY_CACHE_SIZE_MB=20480  # 20GB
            export EARTHLY_USE_INLINE_CACHE=true
            export EARTHLY_CACHE_INLINE_SIZE_MB=500
            export EARTHLY_MAX_PARALLEL_STEPS=30
            export EARTHLY_BUILDKIT_MAX_PARALLELISM=$(nproc)
            
            # Create optimized config with advanced settings
            cat > "${EARTHLY_CONFIG_DIR}/config.yml" << EOF
global:
  disable_analytics: true
  disable_log_upload: true
  cache_size_mb: 20480
  conversion_parallelism: $(nproc)
  buildkit_additional_args: [
    "--oci-worker-gc",
    "--oci-worker-gc-keepstorage", "10000",
    "--oci-max-parallelism", "$(nproc)"
  ]
  
buildkit:
  max_parallelism: $(nproc)
  cache_inline_size_mb: 500
  buildkit_max_parallelism: $(nproc)
  local_registry_host: "tcp://127.0.0.1:8371"
  ip_tables: "auto"
  additional_config: |
    [worker.oci]
      max-parallelism = $(nproc)
    [worker.containerd]
      max-parallelism = $(nproc)
  
git:
  global:
    url_instead_of: ""
    ssh_command: "ssh -o StrictHostKeyChecking=no"

output:
  color: "auto"
  show_images: true
  
cache:
  mode: "max"
  import_parallelism: $(nproc)
  export_parallelism: $(nproc)
EOF
            log_info "Cache optimized for CI/CD performance (20GB cache, $(nproc) parallel workers)"
            ;;
        --optimize-development)
            log_info "Optimizing for local development..."
            export EARTHLY_CACHE_SIZE_MB=10240  # 10GB
            export EARTHLY_USE_INLINE_CACHE=true
            export EARTHLY_CACHE_INLINE_SIZE_MB=200
            export EARTHLY_MAX_PARALLEL_STEPS=10
            create_default_config
            log_info "Cache optimized for development (10GB cache, balanced performance)"
            ;;
        --list)
            log_info "Current configuration:"
            if [[ -f "${EARTHLY_CONFIG_DIR}/config.yml" ]]; then
                cat "${EARTHLY_CONFIG_DIR}/config.yml"
            else
                log_warn "No configuration file found"
            fi
            ;;
        *)
            log_info "Creating default configuration..."
            create_default_config
            ;;
    esac
}

# Create default configuration
create_default_config() {
    cat > "${EARTHLY_CONFIG_DIR}/config.yml" << EOF
global:
  disable_analytics: ${EARTHLY_DISABLE_ANALYTICS}
  disable_log_upload: true
  cache_size_mb: ${EARTHLY_CACHE_SIZE_MB}
  conversion_parallelism: ${EARTHLY_PARALLEL_LIMIT}
  buildkit_additional_args: [
    "--oci-worker-gc",
    "--oci-worker-gc-keepstorage", "50000"
  ]
  
buildkit:
  max_parallelism: ${EARTHLY_PARALLEL_LIMIT}
  cache_inline_size_mb: ${EARTHLY_CACHE_INLINE_SIZE_MB}
  buildkit_max_parallelism: ${EARTHLY_PARALLEL_LIMIT}
  local_registry_host: "tcp://127.0.0.1:8371"
  ip_tables: "auto"
  
git:
  global:
    url_instead_of: ""
    ssh_command: "ssh -o StrictHostKeyChecking=no"

output:
  color: "auto"
  show_images: true
EOF
    
    log_info "Configuration created at ${EARTHLY_CONFIG_DIR}/config.yml"
}

# Show status
show_status() {
    local format="${1:-}"
    
    if [[ "${format}" == "--json" ]]; then
        local earthly_installed=false
        local earthly_version=""
        local docker_running=false
        local cache_size="0"
        local artifact_count=0
        
        if command -v earthly &> /dev/null; then
            earthly_installed=true
            earthly_version=$(earthly --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        fi
        
        if docker ps &>/dev/null; then
            docker_running=true
        fi
        
        if [[ -d "${EARTHLY_CACHE_DIR}" ]]; then
            cache_size=$(du -sh "${EARTHLY_CACHE_DIR}" 2>/dev/null | cut -f1 || echo "0")
        fi
        
        if [[ -d "${EARTHLY_ARTIFACTS_DIR}" ]]; then
            artifact_count=$(find "${EARTHLY_ARTIFACTS_DIR}" -type f 2>/dev/null | wc -l)
        fi
        
        cat << EOF
{
  "earthly_installed": ${earthly_installed},
  "earthly_version": "${earthly_version}",
  "docker_running": ${docker_running},
  "cache_size": "${cache_size}",
  "artifact_count": ${artifact_count},
  "parallel_limit": ${EARTHLY_PARALLEL_LIMIT},
  "cache_size_limit_mb": ${EARTHLY_CACHE_SIZE_MB}
}
EOF
        return 0
    fi
    
    echo "🏗️ Earthly Status"
    echo "=================="
    
    # Check Earthly installation
    if command -v earthly &> /dev/null; then
        local version=$(earthly --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        echo "✅ Earthly installed: v${version}"
    else
        echo "❌ Earthly not installed"
        return 1
    fi
    
    # Check Docker
    if docker ps &>/dev/null; then
        echo "✅ Docker daemon running"
    else
        echo "❌ Docker daemon not running"
    fi
    
    # Check cache usage
    if [[ -d "${EARTHLY_CACHE_DIR}" ]]; then
        local cache_size=$(du -sh "${EARTHLY_CACHE_DIR}" 2>/dev/null | cut -f1)
        echo "📊 Cache usage: ${cache_size:-0}"
    fi
    
    # Check artifacts
    if [[ -d "${EARTHLY_ARTIFACTS_DIR}" ]]; then
        local artifact_count=$(find "${EARTHLY_ARTIFACTS_DIR}" -type f 2>/dev/null | wc -l)
        echo "📦 Artifacts: ${artifact_count}"
    fi
    
    if [[ "${format}" == "--metrics" ]]; then
        echo ""
        echo "Performance Metrics:"
        echo "==================="
        echo "Parallel Limit: ${EARTHLY_PARALLEL_LIMIT}"
        echo "Cache Size Limit: ${EARTHLY_CACHE_SIZE_MB}MB"
        echo "Cache Inline Size: ${EARTHLY_CACHE_INLINE_SIZE_MB}MB"
        
        # Show recent build metrics
        if [[ -f "${EARTHLY_LOGS_DIR}/metrics.csv" ]]; then
            echo ""
            echo "Recent Builds:"
            echo "-------------"
            tail -5 "${EARTHLY_LOGS_DIR}/metrics.csv" | while IFS=',' read -r target duration timestamp status; do
                echo "  ${target}: ${duration}s (${status})"
            done
            
            # Calculate average build time
            local avg_time=$(awk -F',' '$4=="success" {sum+=$2; count++} END {if(count>0) printf "%.1f", sum/count}' "${EARTHLY_LOGS_DIR}/metrics.csv")
            local success_rate=$(awk -F',' '{total++} $4=="success" {success++} END {if(total>0) printf "%.1f", success*100/total}' "${EARTHLY_LOGS_DIR}/metrics.csv")
            
            echo ""
            echo "Statistics:"
            echo "-----------"
            echo "  Average Build Time: ${avg_time:-N/A}s"
            echo "  Success Rate: ${success_rate:-N/A}%"
        else
            echo "No build metrics available yet"
        fi
    fi
    
    return 0
}

# Show logs
show_logs() {
    local lines="${1:-50}"
    
    if [[ -f "${EARTHLY_LOG_FILE}" ]]; then
        tail -n "${lines}" "${EARTHLY_LOG_FILE}"
    else
        log_info "No logs available yet"
    fi
}

# Show credentials
show_credentials() {
    echo "🔑 Earthly Configuration"
    echo "======================="
    echo "Config Dir: ${EARTHLY_CONFIG_DIR}"
    echo "Cache Dir: ${EARTHLY_CACHE_DIR}"
    echo "Artifacts Dir: ${EARTHLY_ARTIFACTS_DIR}"
    echo ""
    
    if [[ -f "${EARTHLY_CONFIG_DIR}/config.yml" ]]; then
        echo "Configuration File:"
        cat "${EARTHLY_CONFIG_DIR}/config.yml"
    else
        echo "No configuration file found"
    fi
}

# Run performance benchmark
run_benchmark() {
    log_info "Running performance benchmark..."
    
    # Create benchmark Earthfile
    local benchmark_file="${EARTHLY_HOME}/benchmark.earth"
    cat > "${benchmark_file}" << 'EOF'
VERSION 0.8
FROM alpine:3.18

benchmark-serial:
    RUN echo "Task 1" && sleep 1
    RUN echo "Task 2" && sleep 1
    RUN echo "Task 3" && sleep 1
    RUN echo "Serial complete"

benchmark-parallel:
    BUILD +worker1
    BUILD +worker2
    BUILD +worker3
    RUN echo "Parallel complete"

worker1:
    RUN echo "Worker 1" && sleep 1

worker2:
    RUN echo "Worker 2" && sleep 1

worker3:
    RUN echo "Worker 3" && sleep 1

benchmark-cache:
    RUN apk add --no-cache curl jq git
    RUN echo "Cache test $(date)" > test.txt
    SAVE ARTIFACT test.txt

all:
    BUILD +benchmark-serial
    BUILD +benchmark-parallel
    BUILD +benchmark-cache
EOF
    
    # Run benchmarks
    log_info "Testing serial execution..."
    local serial_start=$(date +%s)
    earthly --config "${EARTHLY_CONFIG_DIR}/config.yml" \
            -f "${benchmark_file}" +benchmark-serial &>/dev/null
    local serial_end=$(date +%s)
    local serial_time=$((serial_end - serial_start))
    
    log_info "Testing parallel execution..."
    local parallel_start=$(date +%s)
    earthly --config "${EARTHLY_CONFIG_DIR}/config.yml" \
            -f "${benchmark_file}" +benchmark-parallel &>/dev/null
    local parallel_end=$(date +%s)
    local parallel_time=$((parallel_end - parallel_start))
    
    log_info "Testing cache performance..."
    # First build (cold cache)
    earthly prune -a &>/dev/null
    local cold_start=$(date +%s)
    earthly --config "${EARTHLY_CONFIG_DIR}/config.yml" \
            -f "${benchmark_file}" +benchmark-cache &>/dev/null
    local cold_end=$(date +%s)
    local cold_time=$((cold_end - cold_start))
    
    # Second build (warm cache)
    local warm_start=$(date +%s)
    earthly --config "${EARTHLY_CONFIG_DIR}/config.yml" \
            -f "${benchmark_file}" +benchmark-cache &>/dev/null
    local warm_end=$(date +%s)
    local warm_time=$((warm_end - warm_start))
    
    # Calculate metrics
    local speedup=$(echo "scale=2; $serial_time / $parallel_time" | bc 2>/dev/null || echo "N/A")
    local cache_improvement=$(echo "scale=2; ($cold_time - $warm_time) * 100 / $cold_time" | bc 2>/dev/null || echo "N/A")
    
    echo ""
    echo "🏁 Performance Benchmark Results"
    echo "================================="
    echo "Serial Execution: ${serial_time}s"
    echo "Parallel Execution: ${parallel_time}s"
    echo "Parallel Speedup: ${speedup}x"
    echo ""
    echo "Cold Cache Build: ${cold_time}s"
    echo "Warm Cache Build: ${warm_time}s"
    echo "Cache Improvement: ${cache_improvement}%"
    echo ""
    echo "Configuration:"
    echo "- Parallel Limit: ${EARTHLY_PARALLEL_LIMIT}"
    echo "- Cache Size: ${EARTHLY_CACHE_SIZE_MB}MB"
    echo "- Inline Cache: ${EARTHLY_USE_INLINE_CACHE}"
    
    # Store benchmark results
    echo "$(date +%Y-%m-%d_%H:%M:%S),${serial_time},${parallel_time},${speedup},${cold_time},${warm_time},${cache_improvement}" >> "${EARTHLY_LOGS_DIR}/benchmarks.csv"
}