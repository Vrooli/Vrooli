#!/usr/bin/env bash
# VOCR Common Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VOCR_COMMON_DIR="${APP_ROOT}/resources/vocr/lib"

# Source utilities first
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/flow.sh"

# Source configuration
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/vocr/config/defaults.sh"

# Export configuration
vocr::export_config

#######################################
# Start VOCR service
#######################################
vocr::start() {
    log::header "Starting VOCR"
    
    # Check if already running
    if pgrep -f "vocr-service.py" >/dev/null 2>&1; then
        log::warning "VOCR is already running"
        return 0
    fi
    
    # Check if installed
    if [[ ! -f "${VOCR_DATA_DIR}/vocr-service.py" ]]; then
        log::error "VOCR is not installed"
        log::info "Run: vrooli resource install vocr"
        return 1
    fi
    
    # Check Python environment
    if [[ ! -d "${VOCR_DATA_DIR}/venv" ]]; then
        log::error "Python environment not found"
        log::info "Run: vrooli resource install vocr"
        return 1
    fi
    
    # Start service
    if [[ -f /etc/systemd/system/vocr.service ]] && command -v systemctl &>/dev/null && flow::can_run_sudo "systemd service management"; then
        # Use systemd if available and we have sudo
        log::info "Starting VOCR via systemd..."
        sudo systemctl start vocr || {
            log::error "Failed to start VOCR service"
            return 1
        }
    else
        # Start in background
        log::info "Starting VOCR in background..."
        
        # Export environment variables
        export VOCR_PORT VOCR_HOST VOCR_SCREENSHOTS_DIR VOCR_OCR_ENGINE VOCR_OCR_LANGUAGES
        
        # Start service
        nohup "${VOCR_DATA_DIR}/venv/bin/python" "${VOCR_DATA_DIR}/vocr-service.py" \
            > "${VOCR_LOGS_DIR}/vocr.log" 2>&1 &
        
        local pid=$!
        echo "$pid" > "${VOCR_DATA_DIR}/vocr.pid"
        
        # Wait for service to start
        local retries=10
        while [ $retries -gt 0 ]; do
            if curl -s -f "http://${VOCR_HOST}:${VOCR_PORT}/health" >/dev/null 2>&1; then
                log::success "VOCR started successfully (PID: $pid)"
                return 0
            fi
            sleep 1
            ((retries--))
        done
        
        log::error "VOCR failed to start properly"
        return 1
    fi
    
    # Verify service is running
    sleep 2
    if curl -s -f "http://${VOCR_HOST}:${VOCR_PORT}/health" >/dev/null 2>&1; then
        log::success "VOCR is running and healthy"
        return 0
    else
        log::error "VOCR started but health check failed"
        return 1
    fi
}

#######################################
# Stop VOCR service
#######################################
vocr::stop() {
    log::header "Stopping VOCR"
    
    # Check if running
    if ! pgrep -f "vocr-service.py" >/dev/null 2>&1; then
        log::info "VOCR is not running"
        return 0
    fi
    
    # Stop service
    if [[ -f /etc/systemd/system/vocr.service ]] && command -v systemctl &>/dev/null && flow::can_run_sudo "systemd service management"; then
        # Use systemd if available and we have sudo
        log::info "Stopping VOCR via systemd..."
        sudo systemctl stop vocr || {
            log::error "Failed to stop VOCR service"
            return 1
        }
    else
        # Stop background process
        log::info "Stopping VOCR process..."
        
        # Try graceful shutdown first
        if [[ -f "${VOCR_DATA_DIR}/vocr.pid" ]]; then
            local pid
            pid=$(cat "${VOCR_DATA_DIR}/vocr.pid")
            if kill -TERM "$pid" 2>/dev/null; then
                # Wait for process to stop
                local retries=10
                while [ $retries -gt 0 ] && kill -0 "$pid" 2>/dev/null; do
                    sleep 1
                    ((retries--))
                done
            fi
            rm -f "${VOCR_DATA_DIR}/vocr.pid"
        fi
        
        # Force kill if still running
        pkill -f "vocr-service.py" 2>/dev/null || true
    fi
    
    # Verify stopped
    if pgrep -f "vocr-service.py" >/dev/null 2>&1; then
        log::error "Failed to stop VOCR completely"
        return 1
    else
        log::success "VOCR stopped successfully"
        return 0
    fi
}

#######################################
# Restart VOCR service
#######################################
vocr::restart() {
    log::header "Restarting VOCR"
    
    vocr::stop || true
    sleep 2
    vocr::start
}

#######################################
# Configure VOCR
#######################################
vocr::configure() {
    log::header "Configuring VOCR"
    
    # Platform-specific configuration
    case "$(uname -s)" in
        Darwin*)
            echo "macOS Screen Recording Permission Required"
            echo ""
            echo "Please follow these steps:"
            echo "1. Open System Preferences"
            echo "2. Go to Security & Privacy > Privacy"
            echo "3. Select Screen Recording"
            echo "4. Add Terminal (or your terminal app) to the list"
            echo "5. Restart your terminal"
            echo ""
            echo "Press Enter when ready..."
            read -r
            ;;
            
        Linux*)
            echo "Linux Display Configuration"
            echo ""
            if [[ -z "$DISPLAY" ]]; then
                echo "⚠️  No DISPLAY variable set"
                echo "If running remotely, you may need:"
                echo "  export DISPLAY=:0"
                echo "  xhost +local:"
            else
                echo "✅ DISPLAY is set to: $DISPLAY"
                echo "Testing X11 access..."
                xhost +local: 2>/dev/null || {
                    echo "⚠️  Could not set xhost permissions"
                    echo "You may need to run: xhost +local:"
                }
            fi
            ;;
            
        *)
            echo "Platform-specific configuration not available"
            ;;
    esac
    
    # Test capture
    echo ""
    echo "Testing screen capture..."
    if command -v vocr::test_capture &>/dev/null; then
        vocr::test_capture
    else
        resource-vocr test-capture
    fi
}

#######################################
# Clear captured images
#######################################
vocr::clear_captures() {
    local force="${FORCE:-false}"
    
    if [[ "$force" != "true" ]]; then
        log::warning "This will remove all captured images"
        echo "Use --force to confirm"
        return 1
    fi
    
    if [[ -d "$VOCR_SCREENSHOTS_DIR" ]]; then
        rm -rf "${VOCR_SCREENSHOTS_DIR}"/*.png 2>/dev/null || true
        log::success "Cleared captured images"
    else
        log::info "No captures directory found"
    fi
}

#######################################
# List OCR models
#######################################
vocr::list_models() {
    log::header "Available OCR Models"
    
    # List Tesseract models
    if command -v tesseract &>/dev/null; then
        echo "Tesseract languages:"
        tesseract --list-langs 2>/dev/null | tail -n +2 || echo "  No languages found"
    fi
    
    # List custom models
    if [[ -d "$VOCR_MODELS_DIR" ]]; then
        echo ""
        echo "Custom models:"
        ls -1 "$VOCR_MODELS_DIR" 2>/dev/null || echo "  No custom models"
    fi
}

#######################################
# Show VOCR logs
#######################################
vocr::logs() {
    local lines="${1:-50}"
    
    if [[ -f "${VOCR_LOGS_DIR}/vocr.log" ]]; then
        tail -n "$lines" "${VOCR_LOGS_DIR}/vocr.log"
    else
        log::info "No logs available"
    fi
}

#######################################
# Show VOCR credentials
#######################################
vocr::credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "vocr"; return 0; }
        return 1
    fi
    
    local status="not_running"
    if pgrep -f "vocr" >/dev/null 2>&1; then
        status="running"
    fi
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "$VOCR_HOST" \
            --argjson port "$VOCR_PORT" \
            --arg path "/api" \
            '{host: $host, port: $port, path: $path}')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg engine "$VOCR_OCR_ENGINE" \
            --arg languages "$VOCR_OCR_LANGUAGES" \
            --arg vision "$VOCR_VISION_ENABLED" \
            '{ocr_engine: $engine, languages: $languages, vision_enabled: $vision}')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "VOCR API" \
            "noAuth" \
            "$connection_obj" \
            "{}" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    local response
    response=$(credentials::build_response "vocr" "$status" "$connections_array")
    credentials::format_output "$response"
}