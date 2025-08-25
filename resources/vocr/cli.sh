#!/usr/bin/env bash
################################################################################
# VOCR Resource CLI
# 
# Vision OCR and AI-powered screen recognition
#
# Usage:
#   resource-vocr <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    VOCR_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    VOCR_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    VOCR_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${VOCR_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
VOCR_CLI_DIR="${APP_ROOT}/resources/vocr"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source VOCR configuration
# shellcheck disable=SC1091
source "${VOCR_CLI_DIR}/config/defaults.sh"
vocr::export_config

# Source VOCR libraries
for lib in install status common capture ocr vision; do
    lib_file="${VOCR_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file"
    fi
done

# Initialize CLI framework
cli::init "vocr" "Vision OCR and AI-powered screen recognition"

# Register standard commands
cli::register_command "help" "Show help and examples" "vocr_show_help"
cli::register_command "status" "Show VOCR status" "vocr_status"
cli::register_command "start" "Start VOCR service" "vocr_start" "modifies-system"
cli::register_command "stop" "Stop VOCR service" "vocr_stop" "modifies-system"
cli::register_command "install" "Install VOCR" "vocr_install" "modifies-system"
cli::register_command "uninstall" "Uninstall VOCR (requires --force)" "vocr_uninstall" "modifies-system"

# Register VOCR-specific commands
cli::register_command "capture" "Capture screen region" "vocr_capture"
cli::register_command "ocr" "Extract text from screen" "vocr_ocr"
cli::register_command "ask" "Ask AI about screen content" "vocr_ask"
cli::register_command "monitor" "Monitor screen region" "vocr_monitor"
cli::register_command "test-capture" "Test screen capture" "vocr_test_capture"
cli::register_command "configure" "Configure VOCR settings" "vocr_configure" "modifies-system"
cli::register_command "reset-permissions" "Reset screen permissions" "vocr_reset_permissions" "modifies-system"
cli::register_command "diagnose" "Diagnose VOCR issues" "vocr_diagnose"
cli::register_command "inject" "Add OCR templates or models" "vocr_inject" "modifies-system"
cli::register_command "list-models" "List available OCR models" "vocr_list_models"
cli::register_command "pull-model" "Download OCR model" "vocr_pull_model" "modifies-system"
cli::register_command "test" "Run integration tests" "vocr_test"
cli::register_command "logs" "Show VOCR logs" "vocr_logs"
cli::register_command "credentials" "Show n8n credentials" "vocr_credentials"

################################################################################
# Command implementations
################################################################################

# Show status
vocr_status() {
    if command -v vocr::status &>/dev/null; then
        vocr::status "$@"
    else
        # Basic fallback
        log::header "VOCR Status"
        if pgrep -f "vocr" >/dev/null 2>&1; then
            echo "Service: ✅ Running"
        else
            echo "Service: ❌ Not running"
        fi
    fi
}

# Start service
vocr_start() {
    if command -v vocr::start &>/dev/null; then
        vocr::start
    else
        log::error "VOCR start not implemented"
        return 1
    fi
}

# Stop service
vocr_stop() {
    if command -v vocr::stop &>/dev/null; then
        vocr::stop
    else
        log::error "VOCR stop not implemented"
        return 1
    fi
}

# Install VOCR
vocr_install() {
    if command -v vocr::install &>/dev/null; then
        vocr::install
    else
        log::error "VOCR installation not implemented"
        return 1
    fi
}

# Uninstall VOCR
vocr_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove VOCR and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v vocr::uninstall &>/dev/null; then
        vocr::uninstall
    else
        log::error "VOCR uninstall not implemented"
        return 1
    fi
}

# Capture screen
vocr_capture() {
    local region="${1:-}"
    local output="${2:-${VOCR_SCREENSHOTS_DIR}/capture_$(date +%s).png}"
    
    if command -v vocr::capture::screen &>/dev/null; then
        vocr::capture::screen "$region" "$output"
    else
        log::error "Screen capture not implemented"
        return 1
    fi
}

# OCR text extraction
vocr_ocr() {
    local input="${1:-}"
    local language="${2:-${VOCR_OCR_LANGUAGES}}"
    
    if [[ -z "$input" ]]; then
        log::error "Input image or region required"
        echo "Usage: resource-vocr ocr <image|region> [language]"
        return 1
    fi
    
    if command -v vocr::ocr::extract &>/dev/null; then
        vocr::ocr::extract "$input" "$language"
    else
        log::error "OCR extraction not implemented"
        return 1
    fi
}

# Ask AI about screen
vocr_ask() {
    local question="${1:-}"
    local region="${2:-fullscreen}"
    
    if [[ -z "$question" ]]; then
        log::error "Question required"
        echo "Usage: resource-vocr ask \"question\" [region]"
        return 1
    fi
    
    if command -v vocr::vision::ask &>/dev/null; then
        vocr::vision::ask "$question" "$region"
    else
        log::error "Vision AI not implemented"
        return 1
    fi
}

# Monitor screen region
vocr_monitor() {
    local region="${1:-}"
    local interval="${2:-${VOCR_MONITOR_INTERVAL}}"
    local callback="${3:-echo}"
    
    if [[ -z "$region" ]]; then
        log::error "Region required for monitoring"
        echo "Usage: resource-vocr monitor <region> [interval] [callback]"
        return 1
    fi
    
    if command -v vocr::monitor::region &>/dev/null; then
        vocr::monitor::region "$region" "$interval" "$callback"
    else
        log::error "Screen monitoring not implemented"
        return 1
    fi
}

# Test capture
vocr_test_capture() {
    log::header "Testing Screen Capture"
    
    local test_file="${VOCR_SCREENSHOTS_DIR}/test_$(date +%s).png"
    
    if vocr_capture "100,100,400,300" "$test_file"; then
        if [[ -f "$test_file" ]]; then
            log::success "Screen capture successful: $test_file"
            # Try OCR on the capture
            if command -v vocr::ocr::extract &>/dev/null; then
                echo "Testing OCR on captured image..."
                vocr::ocr::extract "$test_file" || true
            fi
            return 0
        fi
    fi
    
    log::error "Screen capture test failed"
    return 1
}

# Configure VOCR
vocr_configure() {
    if command -v vocr::configure &>/dev/null; then
        vocr::configure "$@"
    else
        log::info "VOCR Configuration"
        echo "OCR Engine: $VOCR_OCR_ENGINE"
        echo "Languages: $VOCR_OCR_LANGUAGES"
        echo "Vision Model: $VOCR_VISION_MODEL"
        echo "GPU Enabled: $VOCR_USE_GPU"
    fi
}

# Reset permissions
vocr_reset_permissions() {
    log::header "Resetting Screen Permissions"
    
    case "$(uname -s)" in
        Darwin*)
            echo "On macOS, go to System Preferences > Security & Privacy > Screen Recording"
            echo "Remove and re-add Terminal/VOCR permissions"
            ;;
        Linux*)
            echo "Checking X11 permissions..."
            xhost +local: 2>/dev/null || true
            ;;
        *)
            echo "Platform-specific permission reset not implemented"
            ;;
    esac
}

# Diagnose issues
vocr_diagnose() {
    log::header "VOCR Diagnostics"
    
    # Check dependencies
    echo "Checking dependencies..."
    
    # OCR engines
    if command -v tesseract &>/dev/null; then
        echo "✅ Tesseract: $(tesseract --version 2>&1 | head -1)"
    else
        echo "❌ Tesseract: Not found"
    fi
    
    # Screen capture tools
    case "$(uname -s)" in
        Darwin*)
            if command -v screencapture &>/dev/null; then
                echo "✅ screencapture: Available"
            else
                echo "❌ screencapture: Not found"
            fi
            ;;
        Linux*)
            if command -v scrot &>/dev/null; then
                echo "✅ scrot: Available"
            elif command -v import &>/dev/null; then
                echo "✅ ImageMagick import: Available"
            else
                echo "❌ Screen capture tool: Not found"
            fi
            ;;
    esac
    
    # Python modules
    if python3 -c "import PIL" 2>/dev/null; then
        echo "✅ Python PIL: Available"
    else
        echo "❌ Python PIL: Not found"
    fi
    
    # Permissions
    echo ""
    echo "Checking permissions..."
    vocr_test_capture || echo "⚠️  Screen capture permission may be required"
}

# Inject models or templates
vocr_inject() {
    local type="${1:-}"
    local source="${2:-}"
    
    if [[ -z "$type" || -z "$source" ]]; then
        log::error "Type and source required"
        echo "Usage: resource-vocr inject <model|template> <source>"
        return 1
    fi
    
    case "$type" in
        model)
            cp "$source" "${VOCR_MODELS_DIR}/" || return 1
            log::success "Model injected: $(basename "$source")"
            ;;
        template)
            cp "$source" "${VOCR_CONFIG_DIR}/templates/" || return 1
            log::success "Template injected: $(basename "$source")"
            ;;
        *)
            log::error "Unknown type: $type"
            return 1
            ;;
    esac
}

# List models
vocr_list_models() {
    log::header "Available OCR Models"
    
    if [[ -d "$VOCR_MODELS_DIR" ]]; then
        ls -la "$VOCR_MODELS_DIR" 2>/dev/null || echo "No models found"
    else
        echo "Models directory not initialized"
    fi
}

# Pull model
vocr_pull_model() {
    local model="${1:-}"
    
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-vocr pull-model <model>"
        return 1
    fi
    
    if command -v vocr::models::pull &>/dev/null; then
        vocr::models::pull "$model"
    else
        log::error "Model pulling not implemented"
        return 1
    fi
}

# Run tests
vocr_test() {
    if [[ -f "${VOCR_CLI_DIR}/test/integration-test.sh" ]]; then
        bash "${VOCR_CLI_DIR}/test/integration-test.sh"
    else
        log::error "Integration tests not found"
        return 1
    fi
}

# Show logs
vocr_logs() {
    local lines="${1:-50}"
    
    if [[ -f "${VOCR_LOGS_DIR}/vocr.log" ]]; then
        tail -n "$lines" "${VOCR_LOGS_DIR}/vocr.log"
    else
        echo "No logs available"
    fi
}

# Show credentials for n8n
vocr_credentials() {
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

# Show help
vocr_show_help() {
    cli::_handle_help
    
    echo ""
    echo "👁️ VOCR Vision OCR Examples:"
    echo ""
    echo "Screen Capture:"
    echo "  resource-vocr capture \"100,100,500,400\"        # Capture region"
    echo "  resource-vocr capture fullscreen output.png    # Full screen"
    echo ""
    echo "Text Extraction:"
    echo "  resource-vocr ocr image.png                    # Extract from image"
    echo "  resource-vocr ocr \"100,100,500,400\" eng       # Extract from region"
    echo ""
    echo "AI Vision:"
    echo "  resource-vocr ask \"What's on the screen?\"      # Ask about screen"
    echo "  resource-vocr ask \"Read the error message\"     # Specific query"
    echo ""
    echo "Monitoring:"
    echo "  resource-vocr monitor \"0,0,200,50\" 5          # Monitor region"
    echo "  resource-vocr test-capture                     # Test permissions"
    echo ""
    echo "Configuration:"
    echo "  resource-vocr configure --gpu                  # Enable GPU"
    echo "  resource-vocr configure --engine easyocr       # Change OCR engine"
    echo ""
    echo "Default Port: ${VOCR_PORT}"
    echo "Data Directory: ${VOCR_DATA_DIR}"
}

################################################################################
# Main execution
################################################################################

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi
