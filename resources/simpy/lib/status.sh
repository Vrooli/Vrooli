#!/usr/bin/env bash
# SimPy Status Module - Using Standard Format

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SIMPY_STATUS_DIR="${APP_ROOT}/resources/simpy/lib"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/simpy/config/defaults.sh"
# shellcheck disable=SC1091
source "${SIMPY_STATUS_DIR}/core.sh"

# Always export configuration
simpy::export_config

#######################################
# Main status function using standard framework
#######################################
simpy::status() {
    status::run_standard "simpy" "simpy::status::collect_data" "simpy::status::display_text" "$@"
}

#######################################
# Collect status data
#######################################
simpy::status::collect_data() {
    local -a data=()
    
    # Check if installed
    local installed="false"
    if simpy::is_installed; then
        installed="true"
    fi
    data+=("installed" "$installed")
    
    # Check if running
    local running="false"
    local pid=""
    if simpy::is_running; then
        running="true"
        pid=$(simpy::get_pid)
    fi
    data+=("running" "$running")
    data+=("pid" "$pid")
    
    # Check health
    local health="unknown"
    if [[ "$running" == "true" ]]; then
        if simpy::test_connection; then
            health="healthy"
        else
            health="unhealthy"
        fi
    elif [[ "$installed" == "true" ]]; then
        health="stopped"
    else
        health="not_installed"
    fi
    data+=("health" "$health")
    
    # Add healthy boolean for standard status handling
    local healthy="false"
    [[ "$health" == "healthy" ]] && healthy="true"
    data+=("healthy" "$healthy")
    
    # Service info
    data+=("port" "$SIMPY_PORT")
    data+=("host" "$SIMPY_HOST")
    data+=("base_url" "$SIMPY_BASE_URL")
    
    # Configuration - ensure all values are strings
    data+=("python_version" "${SIMPY_PYTHON_VERSION}")
    data+=("packages" "${SIMPY_PACKAGES}")
    data+=("visualization_enabled" "${SIMPY_ENABLE_VISUALIZATION}")
    data+=("logging_enabled" "${SIMPY_ENABLE_LOGGING}")
    data+=("default_timeout" "${SIMPY_DEFAULT_TIMEOUT}")
    data+=("default_seed" "${SIMPY_DEFAULT_SEED}")
    
    # Directories
    data+=("data_dir" "$SIMPY_DATA_DIR")
    data+=("venv_dir" "$SIMPY_VENV_DIR")
    data+=("examples_dir" "$SIMPY_EXAMPLES_DIR")
    data+=("models_dir" "$SIMPY_MODELS_DIR")
    data+=("results_dir" "$SIMPY_RESULTS_DIR")
    
    # Version info
    local simpy_version="not_installed"
    if [[ "$installed" == "true" ]]; then
        simpy_version=$(simpy::get_version)
    fi
    data+=("simpy_version" "$simpy_version")
    
    # Examples available
    local examples=""
    if [[ "$installed" == "true" ]]; then
        examples=$(simpy::list_examples | tr '\n' ',' | sed 's/,$//')
    fi
    data+=("available_examples" "$examples")
    
    # Check test results
    local test_status="not_run"
    local test_timestamp="N/A"
    local test_file="${var_ROOT_DIR}/data/test-results/simpy-test.json"
    if [[ -f "$test_file" ]]; then
        test_status=$(jq -r '.status // "unknown"' "$test_file" 2>/dev/null || echo "unknown")
        test_timestamp=$(jq -r '.timestamp // "N/A"' "$test_file" 2>/dev/null || echo "N/A")
    fi
    data+=("test_status" "$test_status")
    data+=("test_timestamp" "$test_timestamp")
    
    # Return data array - each key/value on separate line for framework compatibility
    for ((i=0; i<${#data[@]}; i++)); do
        echo "${data[i]}"
    done
}

#######################################
# Display text format status
#######################################
simpy::status::display_text() {
    local -A data
    
    # Convert space-separated string to associative array
    local input="$*"
    local -a pairs
    read -ra pairs <<< "$input"
    
    for ((i=0; i<${#pairs[@]}; i+=2)); do
        local key="${pairs[i]}"
        local value="${pairs[i+1]:-}"
        [[ -n "$key" ]] && data["$key"]="$value"
    done
    
    log::header "🔬 SimPy Status"
    echo ""
    
    # Extract key values
    local installed="${data[installed]:-false}"
    local running="${data[running]:-false}"
    local health="${data[health]:-}"
    local pid="${data[pid]:-}"
    local port="${data[port]:-}"
    local simpy_version="${data[simpy_version]:-}"
    
    # Basic Status
    log::info "📊 Basic Status:"
    if [[ "$installed" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
    fi
    
    if [[ "$running" == "true" ]]; then
        log::success "   ✅ Running: Yes (PID: $pid)"
    else
        log::warning "   ⚠️  Running: No"
    fi
    
    case "$health" in
        healthy)
            log::success "   ✅ Health: Healthy"
            ;;
        unhealthy)
            log::error "   ❌ Health: Unhealthy"
            ;;
        stopped)
            log::warning "   ⚠️  Health: Stopped"
            ;;
        *)
            log::info "   ℹ️  Health: $health"
            ;;
    esac
    
    echo ""
    
    # Service Endpoints
    if [[ "$installed" == "true" ]]; then
        log::info "🌐 Service Endpoints:"
        log::info "   🔗 Base URL: ${data[base_url]:-}"
        log::info "   🏥 Health: http://${data[host]:-localhost}:$port/health"
        log::info "   📊 Simulate: http://${data[host]:-localhost}:$port/simulate"
        log::info "   📚 Examples: http://${data[host]:-localhost}:$port/examples"
        echo ""
    fi
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   🐍 Python Version: ${data[python_version]:-}"
    log::info "   📦 SimPy Version: $simpy_version"
    log::info "   📊 Visualization: ${data[visualization_enabled]:-false}"
    log::info "   📝 Logging: ${data[logging_enabled]:-true}"
    log::info "   ⏱️  Default Timeout: ${data[default_timeout]:-30}s"
    echo ""
    
    # Available Examples
    local examples="${data[available_examples]:-}"
    if [[ -n "$examples" ]]; then
        log::info "📚 Available Examples:"
        echo "$examples" | tr ',' '\n' | while read -r example; do
            [[ -n "$example" ]] && log::info "   • $example"
        done
        echo ""
    fi
    
    # Test Results
    local test_status="${data[test_status]:-not_run}"
    local test_timestamp="${data[test_timestamp]:-}"
    if [[ "$test_status" != "not_run" ]]; then
        log::info "🧪 Test Results:"
        if [[ "$test_status" == "passed" ]]; then
            log::success "   ✅ Status: All tests passed"
        else
            log::warning "   ⚠️  Status: $test_status"
        fi
        log::info "   📅 Last Run: $test_timestamp"
        echo ""
    fi
    
    # Data Directories
    log::info "📁 Data Directories:"
    log::info "   📂 Data: ${data[data_dir]:-}"
    log::info "   🐍 Virtual Env: ${data[venv_dir]:-}"
    log::info "   📚 Examples: ${data[examples_dir]:-}"
    log::info "   📊 Results: ${data[results_dir]:-}"
}

# Main entrypoint when script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    simpy::status "$@"
fi