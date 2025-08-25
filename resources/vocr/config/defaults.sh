#!/usr/bin/env bash
# VOCR Configuration Defaults

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VOCR_SCRIPT_CONFIG_DIR="${APP_ROOT}/resources/vocr/config"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# Source port registry for dynamic port allocation
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/port_registry.sh"

# Export configuration function
vocr::export_config() {
    # Basic configuration
    export VOCR_NAME="vocr"
    export VOCR_DISPLAY_NAME="VOCR (Vision OCR)"
    export VOCR_DESCRIPTION="Advanced screen recognition and AI-powered image analysis"
    export VOCR_CATEGORY="execution"
    
    # Container configuration
    export VOCR_CONTAINER_NAME="${VOCR_CONTAINER_NAME:-vocr}"
    export VOCR_IMAGE="${VOCR_IMAGE:-vocr/vocr:latest}"
    export VOCR_IMAGE_TAG="${VOCR_IMAGE_TAG:-latest}"
    
    # Port configuration - use port registry
    local registry_port=$(ports::get_resource_port vocr)
    export VOCR_PORT="${VOCR_PORT:-${registry_port:-9420}}"
    export VOCR_HOST="${VOCR_HOST:-localhost}"
    export VOCR_BASE_URL="http://${VOCR_HOST}:${VOCR_PORT}"
    
    # Paths - use project directory structure
    export VOCR_DATA_DIR="${VOCR_DATA_DIR:-${var_DATA_DIR}/resources/vocr}"
    export VOCR_CONFIG_DIR="${VOCR_CONFIG_DIR:-${VOCR_DATA_DIR}/config}"
    export VOCR_SCREENSHOTS_DIR="${VOCR_SCREENSHOTS_DIR:-${VOCR_DATA_DIR}/screenshots}"
    export VOCR_MODELS_DIR="${VOCR_MODELS_DIR:-${VOCR_DATA_DIR}/models}"
    export VOCR_LOGS_DIR="${VOCR_LOGS_DIR:-${VOCR_DATA_DIR}/logs}"
    
    # OCR Configuration
    export VOCR_OCR_ENGINE="${VOCR_OCR_ENGINE:-tesseract}"  # tesseract or easyocr
    export VOCR_OCR_LANGUAGES="${VOCR_OCR_LANGUAGES:-eng}"  # comma-separated language codes
    export VOCR_OCR_QUALITY="${VOCR_OCR_QUALITY:-high}"     # low, medium, high
    export VOCR_OCR_TIMEOUT="${VOCR_OCR_TIMEOUT:-10}"       # seconds
    
    # AI Vision Configuration
    export VOCR_VISION_ENABLED="${VOCR_VISION_ENABLED:-true}"
    export VOCR_VISION_MODEL="${VOCR_VISION_MODEL:-llava}"  # Model for vision queries
    export VOCR_VISION_PROVIDER="${VOCR_VISION_PROVIDER:-ollama}"  # ollama or openrouter
    export VOCR_VISION_TIMEOUT="${VOCR_VISION_TIMEOUT:-30}"  # seconds
    
    # Performance Configuration
    export VOCR_USE_GPU="${VOCR_USE_GPU:-false}"
    export VOCR_MAX_WORKERS="${VOCR_MAX_WORKERS:-4}"
    export VOCR_CACHE_SIZE="${VOCR_CACHE_SIZE:-100}"  # MB
    export VOCR_SCREENSHOT_RETENTION="${VOCR_SCREENSHOT_RETENTION:-3600}"  # seconds
    
    # Monitoring Configuration
    export VOCR_MONITOR_ENABLED="${VOCR_MONITOR_ENABLED:-true}"
    export VOCR_MONITOR_INTERVAL="${VOCR_MONITOR_INTERVAL:-5}"  # seconds
    export VOCR_MONITOR_THRESHOLD="${VOCR_MONITOR_THRESHOLD:-0.1}"  # change threshold
    
    # Security Configuration
    export VOCR_AUTH_ENABLED="${VOCR_AUTH_ENABLED:-false}"
    export VOCR_API_KEY="${VOCR_API_KEY:-}"
    export VOCR_ALLOWED_IPS="${VOCR_ALLOWED_IPS:-127.0.0.1}"
    
    # Docker Configuration
    export VOCR_DOCKER_NETWORK="${VOCR_DOCKER_NETWORK:-vrooli_network}"
    export VOCR_RESTART_POLICY="${VOCR_RESTART_POLICY:-unless-stopped}"
    export VOCR_MEMORY_LIMIT="${VOCR_MEMORY_LIMIT:-2g}"
    export VOCR_CPU_LIMIT="${VOCR_CPU_LIMIT:-2}"
    
    # Platform-specific Configuration
    case "$(uname -s)" in
        Darwin*)
            export VOCR_CAPTURE_METHOD="${VOCR_CAPTURE_METHOD:-screencapture}"
            export VOCR_PERMISSIONS_REQUIRED="Screen Recording"
            ;;
        Linux*)
            export VOCR_CAPTURE_METHOD="${VOCR_CAPTURE_METHOD:-scrot}"
            export VOCR_DISPLAY="${VOCR_DISPLAY:-:0}"
            ;;
        MINGW*|CYGWIN*|MSYS*)
            export VOCR_CAPTURE_METHOD="${VOCR_CAPTURE_METHOD:-powershell}"
            ;;
        *)
            export VOCR_CAPTURE_METHOD="${VOCR_CAPTURE_METHOD:-generic}"
            ;;
    esac
    
    # Development Mode
    export VOCR_DEV_MODE="${VOCR_DEV_MODE:-false}"
    export VOCR_DEBUG="${VOCR_DEBUG:-false}"
    export VOCR_LOG_LEVEL="${VOCR_LOG_LEVEL:-info}"
}

# Validate configuration
vocr::validate_config() {
    local errors=0
    
    # Check required directories exist or can be created
    for dir in "$VOCR_DATA_DIR" "$VOCR_CONFIG_DIR" "$VOCR_SCREENSHOTS_DIR" "$VOCR_MODELS_DIR" "$VOCR_LOGS_DIR"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir" 2>/dev/null || {
                echo "Error: Cannot create directory $dir"
                ((errors++))
            }
        fi
    done
    
    # Validate port
    if ! [[ "$VOCR_PORT" =~ ^[0-9]+$ ]] || [ "$VOCR_PORT" -lt 1 ] || [ "$VOCR_PORT" -gt 65535 ]; then
        echo "Error: Invalid port number $VOCR_PORT"
        ((errors++))
    fi
    
    # Check OCR engine availability
    case "$VOCR_OCR_ENGINE" in
        tesseract)
            if ! command -v tesseract &>/dev/null && [[ "$VOCR_DEV_MODE" != "true" ]]; then
                echo "Warning: Tesseract not found in PATH"
            fi
            ;;
        easyocr)
            if ! python3 -c "import easyocr" 2>/dev/null && [[ "$VOCR_DEV_MODE" != "true" ]]; then
                echo "Warning: EasyOCR Python module not found"
            fi
            ;;
    esac
    
    return $errors
}

# Initialize configuration
vocr::export_config