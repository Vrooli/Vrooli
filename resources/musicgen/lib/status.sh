#!/usr/bin/env bash
# MusicGen Status Module - Using Standard Format

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MUSICGEN_STATUS_DIR="${APP_ROOT}/resources/musicgen/lib"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"
# shellcheck disable=SC1091  
source "${MUSICGEN_STATUS_DIR}/common.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

#######################################
# Main status function using standard framework
#######################################
musicgen::status() {
    status::run_standard "musicgen" "musicgen::status::collect_data" "musicgen::status::display_text" "$@"
}

#######################################
# Collect status data
#######################################
musicgen::status::collect_data() {
    local -a data=()
    
    # Check if installed
    local installed="false"
    if docker images "${MUSICGEN_IMAGE}" --format "{{.Repository}}" 2>/dev/null | grep -q "musicgen"; then
        installed="true"
    fi
    data+=("installed" "$installed")
    
    # Check if running
    local running="false"
    local pid=""
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${MUSICGEN_CONTAINER_NAME}$"; then
        running="true"
        pid=$(docker inspect -f '{{.State.Pid}}' "${MUSICGEN_CONTAINER_NAME}" 2>/dev/null || echo "")
    fi
    data+=("running" "$running")
    data+=("pid" "$pid")
    
    # Check health
    local health="unknown"
    if [[ "$running" == "true" ]]; then
        if timeout 5 curl -s "http://localhost:${MUSICGEN_PORT}/health" > /dev/null 2>&1; then
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
    
    # Service info
    data+=("port" "$MUSICGEN_PORT")
    data+=("host" "localhost")
    data+=("base_url" "http://localhost:${MUSICGEN_PORT}")
    
    # Container info
    data+=("container_name" "$MUSICGEN_CONTAINER_NAME")
    data+=("image" "$MUSICGEN_IMAGE")
    
    # Data directories
    data+=("data_dir" "$MUSICGEN_DATA_DIR")
    data+=("models_dir" "$MUSICGEN_MODELS_DIR")
    data+=("output_dir" "$MUSICGEN_OUTPUT_DIR")
    data+=("config_dir" "$MUSICGEN_CONFIG_DIR")
    data+=("inject_dir" "$MUSICGEN_INJECT_DIR")
    
    # Count models and outputs
    local models_count=0
    local outputs_count=0
    if [[ -d "$MUSICGEN_MODELS_DIR" ]]; then
        models_count=$(find "$MUSICGEN_MODELS_DIR" -type f 2>/dev/null | wc -l)
    fi
    if [[ -d "$MUSICGEN_OUTPUT_DIR" ]]; then
        outputs_count=$(find "$MUSICGEN_OUTPUT_DIR" -type f -name "*.wav" 2>/dev/null | wc -l)
    fi
    data+=("models_count" "$models_count")
    data+=("outputs_count" "$outputs_count")
    
    # Check test results
    local test_status="not_run"
    local test_timestamp="N/A"
    local test_file="${var_DATA_DIR}/test-results/musicgen-test.json"
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
musicgen::status::display_text() {
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
    
    log::header "🎵 MusicGen Status"
    echo ""
    
    # Extract key values
    local installed="${data[installed]:-false}"
    local running="${data[running]:-false}"
    local health="${data[health]:-}"
    local pid="${data[pid]:-}"
    local port="${data[port]:-}"
    
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
        log::info "   🎵 Generate: http://${data[host]:-localhost}:$port/generate"
        log::info "   📋 Models: http://${data[host]:-localhost}:$port/models"
        echo ""
    fi
    
    # Container Info
    log::info "🐳 Container:"
    log::info "   📦 Image: ${data[image]:-}"
    log::info "   🏷️  Name: ${data[container_name]:-}"
    echo ""
    
    # Statistics
    local models_count="${data[models_count]:-0}"
    local outputs_count="${data[outputs_count]:-0}"
    if [[ "$models_count" -gt 0 || "$outputs_count" -gt 0 ]]; then
        log::info "📈 Statistics:"
        log::info "   🤖 Models Cached: $models_count"
        log::info "   🎵 Songs Generated: $outputs_count"
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
    log::info "   🤖 Models: ${data[models_dir]:-}"
    log::info "   🎵 Outputs: ${data[output_dir]:-}"
    log::info "   ⚙️  Config: ${data[config_dir]:-}"
}

# Main entrypoint when script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    musicgen::status "$@"
fi