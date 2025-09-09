#!/usr/bin/env bash
# VOCR Status Module - Using Standard Format

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VOCR_STATUS_DIR="${APP_ROOT}/resources/vocr/lib"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/vocr/config/defaults.sh"

# Always export configuration
vocr::export_config

#######################################
# Main status function using standard framework
#######################################
vocr::status() {
    status::run_standard "vocr" "vocr::status::collect_data" "vocr::status::display_text" "$@"
}

#######################################
# Collect status data
#######################################
vocr::status::collect_data() {
    local -a data=()
    
    # Check if installed
    local installed="false"
    # Get data directory from config or use default
    local data_dir="${VOCR_DATA_DIR:-${APP_ROOT}/data/vocr}"
    if [[ -d "$data_dir" ]] && [[ -f "${data_dir}/vocr-service.py" ]]; then
        installed="true"
    fi
    data+=("installed" "$installed")
    
    # Check if running
    local running="false"
    local pid=""
    if pgrep -f "vocr-service.py" >/dev/null 2>&1; then
        running="true"
        pid=$(pgrep -f "vocr-service.py" | head -1)
    fi
    data+=("running" "$running")
    data+=("pid" "$pid")
    
    # Check health
    local health="unknown"
    if [[ "$running" == "true" ]]; then
        if timeout 5s curl -s -f "http://${VOCR_HOST}:${VOCR_PORT}/health" >/dev/null 2>&1; then
            # Service is responding, check if essential tools are available
            local missing_deps=""
            local fallback_script="${APP_ROOT}/data/vocr/scrot-fallback"
            
            # Check OCR capability
            if [[ "$VOCR_OCR_ENGINE" == "tesseract" ]]; then
                # Check for tesseract in various forms
                if ! command -v tesseract &>/dev/null && \
                   ! [[ -x "${VOCR_DATA_DIR}/tesseract-wrapper.sh" ]] && \
                   ! timeout 5s docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^tesseract-vocr$"; then
                    missing_deps="tesseract"
                fi
            fi
            
            # Check screen capture capability
            if ! command -v scrot &>/dev/null && ! command -v import &>/dev/null && ! [[ -x "$fallback_script" ]] && [[ "$(uname -s)" != "Darwin" ]]; then
                missing_deps="${missing_deps:+$missing_deps,}capture_tool"
            fi
            
            if [[ -n "$missing_deps" ]]; then
                health="degraded"
            else
                health="healthy"
            fi
        else
            health="unhealthy"
        fi
    elif [[ "$installed" == "true" ]]; then
        health="stopped"
    else
        health="not_installed"
    fi
    # Convert health status to boolean for JSON compatibility
    local healthy_bool="false"
    if [[ "$health" == "healthy" ]]; then
        healthy_bool="true"
    fi
    data+=("healthy" "$healthy_bool")
    data+=("health" "$health")  # Keep original health status for text display
    
    # Service info
    data+=("port" "$VOCR_PORT")
    data+=("host" "$VOCR_HOST")
    data+=("base_url" "$VOCR_BASE_URL")
    
    # Configuration
    data+=("ocr_engine" "$VOCR_OCR_ENGINE")
    data+=("languages" "$VOCR_OCR_LANGUAGES")
    data+=("vision_enabled" "$VOCR_VISION_ENABLED")
    data+=("gpu_enabled" "$VOCR_USE_GPU")
    data+=("capture_method" "$VOCR_CAPTURE_METHOD")
    
    # Directories
    data+=("data_dir" "$VOCR_DATA_DIR")
    data+=("screenshots_dir" "$VOCR_SCREENSHOTS_DIR")
    data+=("models_dir" "$VOCR_MODELS_DIR")
    
    # Dependencies check
    local tesseract_installed="false"
    # Check for system tesseract or Docker wrapper
    if command -v tesseract &>/dev/null; then
        tesseract_installed="true"
    elif [[ -x "${VOCR_DATA_DIR}/tesseract-wrapper.sh" ]]; then
        tesseract_installed="true"
    elif timeout 5s docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^tesseract-vocr$"; then
        tesseract_installed="true"
    fi
    data+=("tesseract_installed" "$tesseract_installed")
    
    # Check for screen capture tools
    local capture_tool=""
    local fallback_script="${APP_ROOT}/data/vocr/scrot-fallback"
    if command -v scrot &>/dev/null; then
        capture_tool="scrot"
    elif command -v import &>/dev/null; then
        capture_tool="imagemagick"
    elif [[ "$(uname -s)" == "Darwin" ]] && command -v screencapture &>/dev/null; then
        capture_tool="screencapture"
    elif [[ -x "$fallback_script" ]]; then
        capture_tool="python-fallback"
    fi
    data+=("capture_tool" "$capture_tool")
    
    local python_env="false"
    if [[ -d "${VOCR_DATA_DIR}/venv" ]]; then
        python_env="true"
    fi
    data+=("python_env" "$python_env")
    
    # Permissions check (platform specific)
    local permissions="unknown"
    case "$(uname -s)" in
        Darwin*)
            # Check if Terminal has screen recording permission
            # This is a simplified check - full check would require system API
            permissions="check_required"
            ;;
        Linux*)
            # Check X11 access
            if [[ -n "$DISPLAY" ]]; then
                permissions="likely_ok"
            else
                permissions="no_display"
            fi
            ;;
        *)
            permissions="platform_unsupported"
            ;;
    esac
    data+=("permissions" "$permissions")
    
    # Test results
    local test_status="not_run"
    local test_timestamp=""
    if [[ -f "${VOCR_DATA_DIR}/test-results.json" ]]; then
        test_status=$(jq -r '.status // "unknown"' "${VOCR_DATA_DIR}/test-results.json" 2>/dev/null || echo "error")
        test_timestamp=$(jq -r '.timestamp // ""' "${VOCR_DATA_DIR}/test-results.json" 2>/dev/null || echo "")
    fi
    data+=("test_status" "$test_status")
    data+=("test_timestamp" "$test_timestamp")
    
    # Return data array - each key/value on separate line for framework compatibility
    for ((i=0; i<${#data[@]}; i++)); do
        echo "${data[i]}"
    done
}

#######################################
# Display status in text format
#######################################
vocr::status::display_text() {
    local -A data
    
    # Convert array to associative array
    local args=("$@")
    for ((i=0; i<${#args[@]}; i+=2)); do
        local key="${args[i]}"
        local value="${args[i+1]:-}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "👁️ VOCR Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "To install: vrooli resource install vocr"
        return 1
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes (PID: ${data[pid]:-unknown})"
    else
        log::warning "   ⚠️  Running: No"
    fi
    
    case "${data[health]:-unknown}" in
        healthy)
            log::success "   ✅ Health: Healthy"
            ;;
        degraded)
            log::warning "   ⚠️  Health: Degraded (missing dependencies)"
            ;;
        unhealthy)
            log::error "   ❌ Health: Unhealthy"
            ;;
        stopped)
            log::warning "   ⚠️  Health: Stopped"
            ;;
        *)
            log::info "   ❔ Health: ${data[health]:-unknown}"
            ;;
    esac
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🔗 Base URL: ${data[base_url]:-http://localhost:9420}"
    log::info "   🏥 Health: ${data[base_url]:-http://localhost:9420}/health"
    log::info "   📸 Capture: ${data[base_url]:-http://localhost:9420}/capture"
    log::info "   📖 OCR: ${data[base_url]:-http://localhost:9420}/ocr"
    if [[ "${data[vision_enabled]:-false}" == "true" ]]; then
        log::info "   👁️  Vision: ${data[base_url]:-http://localhost:9420}/ask"
    fi
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   🔧 OCR Engine: ${data[ocr_engine]:-tesseract}"
    log::info "   🌐 Languages: ${data[languages]:-eng}"
    log::info "   👁️  Vision AI: ${data[vision_enabled]:-false}"
    log::info "   🎮 GPU Acceleration: ${data[gpu_enabled]:-false}"
    log::info "   📸 Capture Method: ${data[capture_method]:-scrot}"
    echo
    
    # Dependencies
    log::info "📦 Dependencies:"
    if [[ "${data[tesseract_installed]:-false}" == "true" ]]; then
        log::success "   ✅ Tesseract: Installed"
    else
        log::warning "   ⚠️  Tesseract: Not found (required for OCR)"
    fi
    
    if [[ -n "${data[capture_tool]:-}" ]]; then
        log::success "   ✅ Screen Capture: ${data[capture_tool]}"
    else
        log::warning "   ⚠️  Screen Capture: No tool found (install scrot or imagemagick)"
    fi
    
    if [[ "${data[python_env]:-false}" == "true" ]]; then
        log::success "   ✅ Python Environment: Ready"
    else
        log::warning "   ⚠️  Python Environment: Not setup"
    fi
    echo
    
    # Permissions
    log::info "🔐 Permissions:"
    case "${data[permissions]:-unknown}" in
        check_required)
            log::warning "   ⚠️  Screen Recording: Permission check required"
            log::info "   Run: resource-vocr configure"
            ;;
        likely_ok)
            log::success "   ✅ Display Access: Available"
            ;;
        no_display)
            log::error "   ❌ Display: No DISPLAY variable set"
            ;;
        platform_unsupported)
            log::warning "   ⚠️  Platform: Not fully supported"
            ;;
        *)
            log::info "   ❔ Status: ${data[permissions]:-unknown}"
            ;;
    esac
    echo
    
    # Test results
    if [[ -n "${data[test_timestamp]:-}" ]]; then
        log::info "🧪 Test Results:"
        case "${data[test_status]:-unknown}" in
            passed)
                log::success "   ✅ Status: All tests passed"
                ;;
            failed)
                log::error "   ❌ Status: Tests failed"
                ;;
            partial)
                log::warning "   ⚠️  Status: Some tests failed"
                ;;
            *)
                log::info "   Status: ${data[test_status]:-unknown}"
                ;;
        esac
        log::info "   📅 Last Run: ${data[test_timestamp]:-never}"
    fi
    
    # Directories
    log::info "📁 Data Directories:"
    log::info "   📂 Data: ${data[data_dir]}"
    log::info "   📸 Screenshots: ${data[screenshots_dir]}"
    log::info "   🧠 Models: ${data[models_dir]}"
    
    return 0
}

#######################################
# Get simple status
#######################################
vocr::get_status() {
    local health
    health=$(vocr::status::collect_data | while read -r key value; do
        [[ "$key" == "health" ]] && echo "$value" && break
    done)
    echo "${health:-unknown}"
}

#######################################
# Check if healthy
#######################################
vocr::is_healthy() {
    [[ "$(vocr::get_status)" == "healthy" ]]
}