#!/usr/bin/env bash
set -euo pipefail

# Whisper Speech-to-Text Resource Setup and Management
# This script handles installation, configuration, and management of Whisper using Docker

DESCRIPTION="Install and manage Whisper speech-to-text service using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"

# Source dependencies using var.sh variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"

# Whisper configuration
readonly WHISPER_PORT="${WHISPER_CUSTOM_PORT:-9000}"
readonly WHISPER_BASE_URL="http://localhost:${WHISPER_PORT}"
readonly WHISPER_CONTAINER_NAME="whisper"
readonly WHISPER_DATA_DIR="${HOME}/.whisper"
readonly WHISPER_MODELS_DIR="${WHISPER_DATA_DIR}/models"
readonly WHISPER_UPLOADS_DIR="${WHISPER_DATA_DIR}/uploads"

# Docker image configuration
readonly WHISPER_IMAGE="${WHISPER_IMAGE:-onerahmet/openai-whisper-asr-webservice:latest-gpu}"
readonly WHISPER_CPU_IMAGE="${WHISPER_CPU_IMAGE:-onerahmet/openai-whisper-asr-webservice:latest}"

# Model configuration
readonly WHISPER_MODEL_SIZES=("tiny" "base" "small" "medium" "large" "large-v2" "large-v3")
readonly WHISPER_DEFAULT_MODEL="${WHISPER_DEFAULT_MODEL:-medium}"

# Model size information (approximate in GB)
declare -A MODEL_SIZES=(
    ["tiny"]="0.04"
    ["base"]="0.07"
    ["small"]="0.24"
    ["medium"]="0.77"
    ["large"]="1.55"
    ["large-v2"]="1.55"
    ["large-v3"]="1.55"
)

#######################################
# Parse command line arguments
#######################################
manage::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|transcribe|models|info|test|cleanup" \
        --default ""
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Whisper appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "model" \
        --flag "m" \
        --desc "Whisper model size to use (tiny|base|small|medium|large|large-v2|large-v3)" \
        --type "value" \
        --default "$WHISPER_DEFAULT_MODEL"
    
    args::register \
        --name "gpu" \
        --desc "Use GPU-accelerated image (requires NVIDIA GPU)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "file" \
        --desc "Audio file to transcribe (for transcribe action)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "language" \
        --desc "Language code for transcription (e.g., en, es, fr)" \
        --type "value" \
        --default "auto"
    
    args::register \
        --name "task" \
        --desc "Task to perform (transcribe or translate)" \
        --type "value" \
        --options "transcribe|translate" \
        --default "transcribe"
    
    if args::is_asking_for_help "$@"; then
        manage::usage
        exit 0
    fi
    
    # Handle version request (check arguments manually)
    for arg in "$@"; do
        if [[ "$arg" == "--version" ]]; then
            manage::version
            exit 0
        fi
    done
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export MODEL_SIZE=$(args::get "model")
    export USE_GPU=$(args::get "gpu")
    export AUDIO_FILE=$(args::get "file")
    export LANGUAGE=$(args::get "language")
    export TASK=$(args::get "task")
}

#######################################
# Display usage information
#######################################
manage::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                      # Install Whisper with default large model"
    echo "  $0 --action install --model small        # Install with small model"
    echo "  $0 --action install --gpu yes            # Install with GPU support"
    echo "  $0 --action status                       # Check Whisper status"
    echo "  $0 --action transcribe --file audio.mp3  # Transcribe an audio file"
    echo "  $0 --action models                       # List available models"
    echo "  $0 --action cleanup                      # Clean up old upload files"
    echo "  $0 --action uninstall                    # Remove Whisper"
    echo
    echo "Model Sizes:"
    for model in "${WHISPER_MODEL_SIZES[@]}"; do
        printf "  %-10s : %s GB\n" "$model" "${MODEL_SIZES[$model]}"
    done
}

#######################################
# Display version information
#######################################
manage::version() {
    echo "whisper resource manager v1.0"
    echo "Vrooli AI resource for speech-to-text transcription"
    
    # Show service version if container is running
    if manage::is_healthy; then
        echo "Service status: Running"
        local service_info
        if service_info=$(manage::get_service_info 2>/dev/null); then
            echo "$service_info"
        fi
    else
        echo "Service status: Not running or not installed"
        echo "Run '$0 --action install' to install Whisper"
    fi
}

#######################################
# Check if Whisper container exists
# Returns: 0 if exists, 1 otherwise
#######################################
manage::container_exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${WHISPER_CONTAINER_NAME}$"
}

#######################################
# Check if Whisper is running
# Returns: 0 if running, 1 otherwise
#######################################
manage::is_running() {
    docker ps --format "{{.Names}}" | grep -q "^${WHISPER_CONTAINER_NAME}$"
}

#######################################
# Check if Whisper API is healthy and validate service identity
# Returns: 0 if responsive and valid, 1 otherwise
#######################################
manage::is_healthy() {
    # Check the OpenAPI endpoint which returns 200 when the service is ready
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$WHISPER_BASE_URL/openapi.json")
    
    if [[ "$response_code" != "200" ]]; then
        return 1
    fi
    
    # Validate that this is actually the Whisper service by checking the API spec
    local service_title
    service_title=$(curl -s "$WHISPER_BASE_URL/openapi.json" 2>/dev/null | jq -r '.info.title // empty' 2>/dev/null)
    
    if [[ "$service_title" =~ ^Whisper.*Webservice$ ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Get detailed service information for validation
# Returns: service info or empty string if not available
#######################################
manage::get_service_info() {
    if ! manage::is_healthy; then
        return 1
    fi
    
    local openapi_response
    openapi_response=$(curl -s "$WHISPER_BASE_URL/openapi.json" 2>/dev/null)
    
    if [[ -n "$openapi_response" ]] && echo "$openapi_response" | jq . >/dev/null 2>&1; then
        local title version description
        title=$(echo "$openapi_response" | jq -r '.info.title // "Unknown"')
        version=$(echo "$openapi_response" | jq -r '.info.version // "Unknown"')
        description=$(echo "$openapi_response" | jq -r '.info.description // "Unknown"')
        
        echo "Service: $title"
        echo "Version: $version"
        echo "Description: $description"
        return 0
    else
        return 1
    fi
}

#######################################
# Validate model size
#######################################
manage::validate_model() {
    local model="$1"
    for valid_model in "${WHISPER_MODEL_SIZES[@]}"; do
        if [[ "$model" == "$valid_model" ]]; then
            return 0
        fi
    done
    return 1
}

#######################################
# Comprehensive GPU validation
# Returns: 0 if GPU is available and compatible, 1 otherwise
#######################################
manage::validate_gpu() {
    log::info "Validating GPU support..."
    
    # Check if nvidia-smi is available
    if ! system::is_command "nvidia-smi"; then
        log::error "nvidia-smi command not found"
        log::info "Install NVIDIA drivers: https://developer.nvidia.com/cuda-downloads"
        return 1
    fi
    
    # Check if NVIDIA drivers are loaded
    if ! nvidia-smi >/dev/null 2>&1; then
        log::error "NVIDIA drivers not loaded or GPU not detected"
        log::info "Check: sudo modprobe nvidia"
        return 1
    fi
    
    # Get GPU information
    local gpu_count gpu_memory gpu_driver_version
    gpu_count=$(nvidia-smi --query-gpu=count --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "0")
    gpu_memory=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "0")
    gpu_driver_version=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null | head -1 || echo "unknown")
    
    if [[ "$gpu_count" == "0" ]]; then
        log::error "No NVIDIA GPUs detected"
        return 1
    fi
    
    log::success "✅ Found $gpu_count NVIDIA GPU(s)"
    log::info "GPU memory: ${gpu_memory}MB, Driver: $gpu_driver_version"
    
    # Check minimum memory requirement (2GB recommended for Whisper)
    if [[ "$gpu_memory" -lt 2048 ]]; then
        log::warn "⚠️  GPU has less than 2GB memory (${gpu_memory}MB). Performance may be limited."
    fi
    
    # Check if Docker can access GPU
    if ! docker run --rm --gpus all nvidia/cuda:11.0-base nvidia-smi >/dev/null 2>&1; then
        log::error "Docker cannot access GPU. Check nvidia-docker installation"
        log::info "Install nvidia-docker: https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html"
        return 1
    fi
    
    log::success "✅ Docker GPU access confirmed"
    return 0
}

#######################################
# Get GPU system information for diagnostics
#######################################
manage::get_gpu_info() {
    if ! system::is_command "nvidia-smi"; then
        echo "NVIDIA drivers not installed"
        return 1
    fi
    
    echo "=== GPU Information ==="
    nvidia-smi --query-gpu=name,memory.total,memory.used,memory.free,temperature.gpu,utilization.gpu --format=csv 2>/dev/null || echo "GPU query failed"
    echo
    echo "=== Driver Information ==="
    nvidia-smi --query-gpu=driver_version,cuda_version --format=csv 2>/dev/null || echo "Driver query failed"
}

#######################################
# Create necessary directories
#######################################
manage::create_directories() {
    log::info "Creating Whisper directories..."
    
    mkdir -p "$WHISPER_DATA_DIR" || {
        log::error "Failed to create data directory: $WHISPER_DATA_DIR"
        return 1
    }
    
    mkdir -p "$WHISPER_MODELS_DIR" || {
        log::error "Failed to create models directory: $WHISPER_MODELS_DIR"
        return 1
    }
    
    mkdir -p "$WHISPER_UPLOADS_DIR" || {
        log::error "Failed to create uploads directory: $WHISPER_UPLOADS_DIR"
        return 1
    }
    
    # Set proper permissions
    chmod 755 "$WHISPER_DATA_DIR" "$WHISPER_MODELS_DIR" "$WHISPER_UPLOADS_DIR"
    
    log::success "Directories created successfully"
    return 0
}

#######################################
# Pull Docker image with progress indicators
#######################################
manage::pull_image() {
    local image="$1"
    
    log::info "Pulling Whisper Docker image: $image"
    
    # Check if image already exists locally
    if docker image inspect "$image" >/dev/null 2>&1; then
        log::info "Image already exists locally, checking for updates..."
    fi
    
    # Estimate download size based on image type
    local estimated_size="1-2GB"
    if [[ "$image" == *"gpu"* ]]; then
        estimated_size="2-3GB"
    fi
    
    log::info "Estimated download size: $estimated_size"
    log::info "This may take 5-15 minutes depending on your internet connection..."
    echo "Progress will be shown below:"
    echo
    
    # Pull with progress output
    local pull_start_time=$(date +%s)
    if docker pull "$image" 2>&1 | while IFS= read -r line; do
        # Show progress lines but filter out redundant info
        if [[ "$line" =~ (Pulling|Downloading|Extracting|Pull complete) ]]; then
            echo "  $line"
        elif [[ "$line" =~ (Status:|Digest:|already exists) ]]; then
            echo "  $line"
        fi
    done; then
        local pull_end_time=$(date +%s)
        local pull_duration=$((pull_end_time - pull_start_time))
        echo
        log::success "✅ Docker image pulled successfully in ${pull_duration}s"
        
        # Show final image info
        local image_size
        image_size=$(docker image inspect "$image" --format='{{.Size}}' 2>/dev/null | numfmt --to=iec || echo "unknown")
        log::info "Final image size: $image_size"
        return 0
    else
        echo
        log::error "Failed to pull Docker image"
        log::info "This could be due to:"
        log::info "  - Network connectivity issues"
        log::info "  - Docker Hub rate limiting"
        log::info "  - Insufficient disk space"
        log::info "  - Docker daemon issues"
        log::info "Check: docker system df (for disk space)"
        return 1
    fi
}

#######################################
# Start Whisper container
#######################################
manage::start_container() {
    if manage::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Whisper is already running"
        return 0
    fi
    
    # Stop existing container if force is specified
    if manage::is_running && [[ "$FORCE" == "yes" ]]; then
        log::info "Stopping existing container (force specified)..."
        manage::stop_container
    fi
    
    # Remove existing container if it exists but is not running
    if manage::container_exists && ! manage::is_running; then
        log::info "Removing existing stopped container..."
        docker rm "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Determine which image to use
    local image="$WHISPER_CPU_IMAGE"
    if [[ "$USE_GPU" == "yes" ]]; then
        if manage::validate_gpu; then
            image="$WHISPER_IMAGE"
            log::success "✅ GPU validation passed - using GPU-accelerated image"
        else
            log::warn "⚠️  GPU validation failed - falling back to CPU image"
            log::info "For GPU support, ensure NVIDIA drivers and nvidia-docker are properly installed"
        fi
    fi
    
    log::info "Starting Whisper container..."
    
    # Docker run command with proper volume mounts and environment
    local docker_cmd=(
        docker run -d
        --name "$WHISPER_CONTAINER_NAME"
        --restart unless-stopped
        -p "${WHISPER_PORT}:9000"
        -v "${WHISPER_MODELS_DIR}:/root/.cache/whisper"
        -v "${WHISPER_UPLOADS_DIR}:/tmp/uploads"
        -e "ASR_MODEL=${MODEL_SIZE}"
        -e "ASR_ENGINE=openai_whisper"
    )
    
    # Add GPU support if requested
    if [[ "$USE_GPU" == "yes" && "$image" == "$WHISPER_IMAGE" ]]; then
        docker_cmd+=(--gpus all)
    fi
    
    docker_cmd+=("$image")
    
    if "${docker_cmd[@]}"; then
        log::success "Whisper container started"
        
        # Wait for service to be ready
        log::info "Waiting for Whisper to be ready (model loading can take 1-3 minutes)..."
        if resources::wait_for_service "Whisper" "$WHISPER_PORT" 180; then
            # Additional health check
            sleep 5
            if manage::is_healthy; then
                log::success "✅ Whisper is running and healthy on port $WHISPER_PORT"
                return 0
            else
                log::warn "⚠️  Whisper started but health check failed"
                log::info "The service may still be initializing. Check logs with: $0 --action logs"
                return 0
            fi
        else
            log::error "Whisper failed to start within 3 minutes"
            log::info "Check logs with: docker logs $WHISPER_CONTAINER_NAME"
            return 1
        fi
    else
        log::error "Failed to start Whisper container"
        return 1
    fi
}

#######################################
# Stop Whisper container
#######################################
manage::stop_container() {
    if ! manage::is_running; then
        log::info "Whisper is not running"
        return 0
    fi
    
    log::info "Stopping Whisper container..."
    if docker stop "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "Whisper container stopped"
        return 0
    else
        log::error "Failed to stop Whisper container"
        return 1
    fi
}

#######################################
# Show container logs
#######################################
manage::show_logs() {
    if ! manage::container_exists; then
        log::error "Whisper container does not exist"
        return 1
    fi
    
    docker logs -f "$WHISPER_CONTAINER_NAME"
}

#######################################
# Validate audio file before processing
#######################################
manage::validate_audio_file() {
    local file="$1"
    local max_size_mb=100
    
    # Check file exists
    if [[ ! -f "$file" ]]; then
        log::error "Audio file not found: $file"
        return 1
    fi
    
    # Check file size
    local file_size_mb
    file_size_mb=$(du -m "$file" | cut -f1)
    if [[ $file_size_mb -gt $max_size_mb ]]; then
        log::error "File too large: ${file_size_mb}MB (maximum: ${max_size_mb}MB)"
        log::info "For large files, consider splitting into smaller segments"
        return 1
    fi
    
    # Check file format by extension
    local file_ext="${file##*.}"
    file_ext="${file_ext,,}"  # Convert to lowercase
    local supported_formats=("wav" "mp3" "ogg" "m4a" "flac" "aac" "wma" "opus" "webm")
    if [[ ! " ${supported_formats[@]} " =~ " ${file_ext} " ]]; then
        log::error "Unsupported file format: '$file_ext'"
        log::info "Supported formats: ${supported_formats[*]}"
        log::info "Convert your file using: ffmpeg -i input.$file_ext -acodec pcm_s16le -ar 16000 output.wav"
        return 1
    fi
    
    # Check if file is actually audio using file command (if available)
    if system::is_command "file"; then
        local file_type
        file_type=$(file -b --mime-type "$file" 2>/dev/null)
        if [[ -n "$file_type" && ! "$file_type" =~ ^(audio/|video/) ]]; then
            log::error "File does not appear to be audio/video: $file_type"
            log::info "Expected MIME types: audio/* or video/*"
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Parse API error response for better error messages
#######################################
manage::parse_api_error() {
    local response="$1"
    
    # Try to parse as JSON error
    if echo "$response" | jq . >/dev/null 2>&1; then
        local error_detail
        error_detail=$(echo "$response" | jq -r '.detail // empty' 2>/dev/null)
        if [[ -n "$error_detail" && "$error_detail" != "null" ]]; then
            # Handle FastAPI validation errors
            if echo "$error_detail" | jq -e 'type == "array"' >/dev/null 2>&1; then
                log::error "API validation error:"
                echo "$error_detail" | jq -r '.[] | "  - \(.loc | join(".")): \(.msg)"'
            else
                log::error "API error: $error_detail"
            fi
            return 0
        fi
    fi
    
    # Handle common error patterns
    case "$response" in
        *"Internal Server Error"*)
            log::error "Internal server error - the audio file may be corrupted or in an unsupported format"
            log::info "Try converting to WAV: ffmpeg -i input.audio -acodec pcm_s16le -ar 16000 output.wav"
            ;;
        *"413"*|*"Request Entity Too Large"*)
            log::error "File too large for server processing"
            ;;
        *"timeout"*|*"timed out"*)
            log::error "Processing timeout - file may be too long or server overloaded"
            ;;
        *"503"*|*"Service Unavailable"*)
            log::error "Whisper service temporarily unavailable"
            log::info "Check service status: $0 --action status"
            ;;
        *)
            log::error "Whisper API error"
            log::info "Raw response: $response"
            ;;
    esac
}

#######################################
# Transcribe audio file
#######################################
manage::transcribe() {
    if [[ -z "${AUDIO_FILE:-}" ]]; then
        log::error "No audio file specified. Use --file <path>"
        return 1
    fi
    
    # Validate audio file
    if ! manage::validate_audio_file "$AUDIO_FILE"; then
        return 1
    fi
    
    # Get file info for logging (validation already passed)
    local file_size_mb
    file_size_mb=$(du -m "$AUDIO_FILE" | cut -f1)
    local file_ext="${AUDIO_FILE##*.}"
    file_ext="${file_ext,,}"
    
    if ! manage::is_healthy; then
        log::error "Whisper service is not available"
        log::info "Start it with: $0 --action start"
        return 1
    fi
    
    log::info "Transcribing file: $AUDIO_FILE"
    log::info "Model: $MODEL_SIZE, Task: $TASK, Language: $LANGUAGE"
    log::info "File size: ${file_size_mb}MB, Format: $file_ext"
    
    # Prepare curl command - output must be query parameter
    local curl_cmd=(
        curl -s -X POST
        "${WHISPER_BASE_URL}/asr?output=json"
        -F "audio_file=@${AUDIO_FILE}"
        -F "task=${TASK}"
    )
    
    # Add language if specified
    if [[ "$LANGUAGE" != "auto" ]]; then
        curl_cmd+=(-F "language=${LANGUAGE}")
    fi
    
    # Execute transcription with progress
    log::info "Processing... (this may take a few minutes for longer files)"
    local response
    local start_time=$(date +%s)
    
    if response=$("${curl_cmd[@]}" 2>/dev/null); then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        # Check if response is valid JSON and contains transcription
        if echo "$response" | jq . >/dev/null 2>&1; then
            # Check if this is an error response
            if echo "$response" | jq -e '.detail' >/dev/null 2>&1; then
                manage::parse_api_error "$response"
                return 1
            fi
            
            # Check if response contains expected transcription fields
            if echo "$response" | jq -e '.text' >/dev/null 2>&1; then
                log::success "✅ Transcription completed in ${duration}s"
                echo "$response" | jq .
                
                # Show summary
                local text_length
                text_length=$(echo "$response" | jq -r '.text' | wc -c)
                log::info "Transcribed text length: ${text_length} characters"
                
                # Check for silent audio indication
                local no_speech_prob
                no_speech_prob=$(echo "$response" | jq -r '.segments[0].no_speech_prob // 0' 2>/dev/null)
                if (( $(echo "$no_speech_prob > 0.8" | bc -l 2>/dev/null || echo 0) )); then
                    log::warn "⚠️  High probability of no speech detected (${no_speech_prob})"
                    log::info "This may be a silent file or very quiet audio"
                fi
            else
                log::error "Unexpected response format from Whisper API"
                echo "Raw response: $response"
                return 1
            fi
        else
            manage::parse_api_error "$response"
            return 1
        fi
    else
        log::error "Failed to connect to Whisper service"
        log::info "Check if Whisper is running: $0 --action status"
        return 1
    fi
}

#######################################
# Update Vrooli configuration
#######################################
manage::update_config() {
    local additional_config=$(cat <<EOF
{
    "modelSize": "$MODEL_SIZE",
    "supportedFormats": ["wav", "mp3", "ogg", "m4a", "flac", "aac", "wma"],
    "api": {
        "transcribe": "/asr",
        "detectLanguage": "/detect-language",
        "openapi": "/openapi.json",
        "docs": "/docs"
    },
    "capabilities": {
        "transcription": true,
        "translation": true,
        "timestamps": true,
        "multiLanguage": true,
        "languageDetection": true
    }
}
EOF
)
    
    resources::update_config "ai" "whisper" "$WHISPER_BASE_URL" "$additional_config"
}

#######################################
# Setup automatic cleanup scheduling
#######################################
manage::setup_automatic_cleanup() {
    local script_path="$(realpath "${BASH_SOURCE[0]}")"
    local cleanup_schedule="0 */6 * * *"  # Every 6 hours
    local cron_command="$script_path --action cleanup"
    local cron_entry="$cleanup_schedule $cron_command # Whisper automatic cleanup"
    
    log::info "Setting up automatic cleanup schedule..."
    
    # Check if cron is available
    if ! system::is_command "crontab"; then
        log::warn "crontab not available - skipping automatic cleanup setup"
        log::info "You can manually run cleanup with: $script_path --action cleanup"
        return 1
    fi
    
    # Get current crontab (may be empty)
    local current_crontab
    current_crontab=$(crontab -l 2>/dev/null || echo "")
    
    # Check if cleanup job already exists
    if echo "$current_crontab" | grep -q "Whisper automatic cleanup"; then
        log::info "Automatic cleanup already scheduled"
        return 0
    fi
    
    # Add the new cron job
    {
        echo "$current_crontab"
        echo "$cron_entry"
    } | crontab -
    
    if [[ $? -eq 0 ]]; then
        log::success "✅ Automatic cleanup scheduled (every 6 hours)"
        log::info "Cleanup command: $cron_command"
        log::info "View schedule: crontab -l | grep Whisper"
        return 0
    else
        log::error "Failed to setup automatic cleanup"
        return 1
    fi
}

#######################################
# Remove automatic cleanup scheduling
#######################################
manage::remove_automatic_cleanup() {
    log::info "Removing automatic cleanup schedule..."
    
    if ! system::is_command "crontab"; then
        log::info "crontab not available - nothing to remove"
        return 0
    fi
    
    # Get current crontab
    local current_crontab
    current_crontab=$(crontab -l 2>/dev/null || echo "")
    
    # Remove Whisper cleanup entries
    local new_crontab
    new_crontab=$(echo "$current_crontab" | grep -v "Whisper automatic cleanup" || echo "")
    
    # Update crontab
    if [[ -n "$new_crontab" ]]; then
        echo "$new_crontab" | crontab -
    else
        # If no cron jobs left, remove crontab entirely
        crontab -r 2>/dev/null || true
    fi
    
    log::success "✅ Automatic cleanup schedule removed"
}

#######################################
# Complete Whisper installation
#######################################
manage::install() {
    log::header "🎤 Installing Whisper Speech-to-Text"
    
    # Start rollback context
    resources::start_rollback_context "install_whisper"
    
    # Check if already installed
    if manage::container_exists && manage::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Whisper is already installed and running"
        log::info "Use --force yes to reinstall"
        return 0
    fi
    
    # Validate Docker is available
    if ! resources::ensure_docker; then
        log::error "Docker is required but not available"
        return 1
    fi
    
    # Install ffmpeg if not present
    if ! system::is_command "ffmpeg"; then
        log::info "Installing ffmpeg for audio format conversion..."
        local pm
        pm=$(system::detect_pm)
        
        case "$pm" in
            "apt-get")
                if sudo apt-get update && sudo apt-get install -y ffmpeg; then
                    log::success "ffmpeg installed successfully"
                else
                    log::warn "Failed to install ffmpeg. Audio format conversion may be limited."
                fi
                ;;
            "dnf"|"yum")
                if sudo "$pm" install -y ffmpeg; then
                    log::success "ffmpeg installed successfully"
                else
                    log::warn "Failed to install ffmpeg. Audio format conversion may be limited."
                fi
                ;;
            "pacman")
                if sudo pacman -S --noconfirm ffmpeg; then
                    log::success "ffmpeg installed successfully"
                else
                    log::warn "Failed to install ffmpeg. Audio format conversion may be limited."
                fi
                ;;
            *)
                log::warn "Unable to auto-install ffmpeg on this system (package manager: $pm). Please install manually for full audio format support."
                ;;
        esac
    else
        log::info "ffmpeg is already installed"
    fi
    
    # Install espeak if not present (useful for testing)
    if ! system::is_command "espeak"; then
        log::info "Installing espeak for text-to-speech testing..."
        local pm
        pm=$(system::detect_pm)
        
        case "$pm" in
            "apt-get")
                if sudo apt-get install -y espeak; then
                    log::success "espeak installed successfully"
                else
                    log::warn "Failed to install espeak. Text-to-speech testing will be limited."
                fi
                ;;
            "dnf"|"yum")
                if sudo "$pm" install -y espeak; then
                    log::success "espeak installed successfully"
                else
                    log::warn "Failed to install espeak. Text-to-speech testing will be limited."
                fi
                ;;
            "pacman")
                if sudo pacman -S --noconfirm espeak; then
                    log::success "espeak installed successfully"
                else
                    log::warn "Failed to install espeak. Text-to-speech testing will be limited."
                fi
                ;;
            *)
                log::info "Unable to auto-install espeak on this system. Install manually for text-to-speech testing."
                ;;
        esac
    else
        log::info "espeak is already installed"
    fi
    
    # Validate model size
    if ! manage::validate_model "$MODEL_SIZE"; then
        log::error "Invalid model size: $MODEL_SIZE"
        log::info "Valid sizes: ${WHISPER_MODEL_SIZES[*]}"
        return 1
    fi
    
    # Check available disk space
    local required_gb=2  # Base container size
    required_gb=$(echo "$required_gb + ${MODEL_SIZES[$MODEL_SIZE]}" | bc)
    local available_gb
    if available_gb=$(df "$HOME" --output=avail --block-size=1G | tail -n1 | tr -d ' '); then
        if (( $(echo "$available_gb < $required_gb" | bc -l) )); then
            log::warn "⚠️  Low disk space: ${available_gb}GB available, ~${required_gb}GB recommended"
            log::info "Whisper model '$MODEL_SIZE' requires ~${MODEL_SIZES[$MODEL_SIZE]}GB"
            
            if ! flow::is_yes "$YES"; then
                if ! flow::confirm "Continue anyway?"; then
                    log::info "Installation cancelled"
                    return 0
                fi
            fi
        fi
    fi
    
    # Validate port
    if ! resources::validate_port "whisper" "$WHISPER_PORT" "$FORCE"; then
        log::error "Port validation failed for Whisper"
        return 1
    fi
    
    # Create directories
    if ! manage::create_directories; then
        return 1
    fi
    
    # Add rollback for directories
    resources::add_rollback_action \
        "Remove Whisper directories" \
        "trash::safe_remove '$WHISPER_DATA_DIR' --no-confirm" \
        10
    
    # Determine image to use
    local image="$WHISPER_CPU_IMAGE"
    if [[ "$USE_GPU" == "yes" ]]; then
        image="$WHISPER_IMAGE"
    fi
    
    # Pull Docker image
    if ! manage::pull_image "$image"; then
        return 1
    fi
    
    # Start container
    if ! manage::start_container; then
        return 1
    fi
    
    # Add rollback for container
    resources::add_rollback_action \
        "Stop and remove Whisper container" \
        "docker stop $WHISPER_CONTAINER_NAME 2>/dev/null || true; docker rm $WHISPER_CONTAINER_NAME 2>/dev/null || true" \
        20
    
    # Wait a bit more for model to fully load
    log::info "Waiting for model to load (this can take up to 3 minutes for large models)..."
    sleep 10
    
    # Clear rollback since core installation succeeded
    log::info "Whisper core installation completed successfully"
    ROLLBACK_ACTIONS=()
    OPERATION_ID=""
    
    # Update Vrooli configuration
    if ! manage::update_config; then
        log::warn "Failed to update Vrooli configuration"
        log::info "Whisper is installed but may need manual configuration"
    fi
    
    # Setup automatic cleanup scheduling
    if ! manage::setup_automatic_cleanup; then
        log::warn "Failed to setup automatic cleanup - you may want to run cleanup manually"
    fi
    
    log::success "✅ Whisper installation completed successfully"
    
    # Show status
    echo
    manage::status
}

#######################################
# Uninstall Whisper
#######################################
manage::uninstall() {
    log::header "🗑️  Uninstalling Whisper"
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will remove the Whisper container and optionally its data"
        if ! flow::confirm "Are you sure you want to continue?"; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop container
    if manage::is_running; then
        manage::stop_container
    fi
    
    # Remove container
    if manage::container_exists; then
        log::info "Removing Whisper container..."
        docker rm "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Ask about data removal
    if [[ -d "$WHISPER_DATA_DIR" ]]; then
        if ! flow::is_yes "$YES"; then
            if flow::confirm "Remove Whisper data directory?"; then
                trash::safe_remove "$WHISPER_DATA_DIR" --no-confirm
                log::info "Data directory removed"
            fi
        fi
    fi
    
    # Remove from Vrooli config
    resources::remove_config "ai" "whisper"
    
    # Remove automatic cleanup scheduling
    manage::remove_automatic_cleanup
    
    log::success "✅ Whisper uninstalled successfully"
}

#######################################
# Show Whisper status
#######################################
manage::status() {
    log::header "📊 Whisper Status"
    
    # Check Docker
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        return 1
    fi
    
    # Check container status
    if manage::container_exists; then
        if manage::is_running; then
            log::success "✅ Whisper container is running"
            
            # Get container stats
            local stats
            stats=$(docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" "$WHISPER_CONTAINER_NAME" 2>/dev/null || echo "")
            if [[ -n "$stats" ]]; then
                echo
                echo "$stats"
            fi
            
            # Check health and validate service
            if manage::is_healthy; then
                log::success "✅ API is healthy and responding"
                
                # Show detailed service information
                echo
                log::info "Service Details:"
                if service_info=$(manage::get_service_info 2>/dev/null); then
                    echo "$service_info" | while IFS= read -r line; do
                        log::info "  $line"
                    done
                else
                    log::info "  Service information unavailable"
                fi
            else
                log::warn "⚠️  API health check failed or service identity could not be verified"
            fi
            
            # Show port
            echo
            log::info "Port: $WHISPER_PORT"
            
            # Check if port is actually listening
            if resources::is_service_running "$WHISPER_PORT"; then
                log::success "✅ Service is listening on port $WHISPER_PORT"
            else
                log::warn "⚠️  Port $WHISPER_PORT is not accessible"
            fi
            
            # Additional Whisper-specific info
            echo
            log::info "API Endpoints:"
            log::info "  Base URL: $WHISPER_BASE_URL"
            log::info "  Transcription: $WHISPER_BASE_URL/asr"
            log::info "  Language Detection: $WHISPER_BASE_URL/detect-language"
            log::info "  API Docs: $WHISPER_BASE_URL/docs"
            log::info "  OpenAPI Spec: $WHISPER_BASE_URL/openapi.json"
            
            echo
            log::info "Configuration:"
            log::info "  Model Size: $MODEL_SIZE (~${MODEL_SIZES[$MODEL_SIZE]}GB)"
            log::info "  GPU Enabled: $USE_GPU"
            log::info "  Models Directory: $WHISPER_MODELS_DIR"
            log::info "  Uploads Directory: $WHISPER_UPLOADS_DIR"
            
            # Show GPU information if available
            if [[ "$USE_GPU" == "yes" ]] && system::is_command "nvidia-smi"; then
                echo
                log::info "GPU Information:"
                if gpu_info=$(manage::get_gpu_info 2>/dev/null); then
                    echo "$gpu_info" | while IFS= read -r line; do
                        if [[ "$line" =~ ^=== ]]; then
                            log::info "  $line"
                        elif [[ -n "$line" ]]; then
                            log::info "    $line"
                        fi
                    done
                else
                    log::info "  GPU information unavailable"
                fi
            fi
            
            # Show model files if they exist
            if [[ -d "$WHISPER_MODELS_DIR" ]]; then
                local model_count
                model_count=$(find "$WHISPER_MODELS_DIR" -name "*.pt" 2>/dev/null | wc -l)
                if [[ $model_count -gt 0 ]]; then
                    echo
                    log::info "Downloaded Models:"
                    find "$WHISPER_MODELS_DIR" -name "*.pt" -printf "  - %f (%s bytes)\n" 2>/dev/null | sort
                fi
            fi
            
        else
            log::warn "⚠️  Whisper container exists but is not running"
            log::info "Start it with: $0 --action start"
        fi
    else
        log::info "Whisper is not installed"
        log::info "Install with: $0 --action install"
    fi
}

#######################################
# List available models
#######################################
manage::list_models() {
    log::header "📋 Available Whisper Models"
    
    echo "Model sizes (from smallest to largest):"
    echo
    printf "%-12s %-10s %s\n" "MODEL" "SIZE" "NOTES"
    printf "%-12s %-10s %s\n" "-----" "----" "-----"
    
    for model in "${WHISPER_MODEL_SIZES[@]}"; do
        local size="${MODEL_SIZES[$model]}GB"
        local notes=""
        
        if [[ "$model" == "$WHISPER_DEFAULT_MODEL" ]]; then
            notes="(default)"
        fi
        
        if [[ "$model" == "tiny" ]]; then
            notes="$notes Fast, lower accuracy"
        elif [[ "$model" == "large" || "$model" == "large-v2" || "$model" == "large-v3" ]]; then
            notes="$notes Best accuracy, slower"
        fi
        
        printf "%-12s %-10s %s\n" "$model" "$size" "$notes"
    done
    
    echo
    log::info "Current model: $MODEL_SIZE"
    log::info "To change model, reinstall with: $0 --action install --model <size>"
}

#######################################
# Show Whisper information
#######################################
manage::info() {
    cat << EOF
=== Whisper Resource Information ===

ID: whisper
Category: ai
Display Name: Whisper Speech-to-Text
Description: OpenAI Whisper for audio transcription

Service Details:
- Container Name: $WHISPER_CONTAINER_NAME
- Service Port: $WHISPER_PORT
- Service URL: $WHISPER_BASE_URL
- Data Directory: $WHISPER_DATA_DIR

Docker Images:
- CPU: $WHISPER_CPU_IMAGE
- GPU: $WHISPER_IMAGE

Endpoints:
- Transcription: POST $WHISPER_BASE_URL/asr
- Language Detection: POST $WHISPER_BASE_URL/detect-language
- API Documentation: GET $WHISPER_BASE_URL/docs
- OpenAPI Spec: GET $WHISPER_BASE_URL/openapi.json

Configuration:
- Default Model: $WHISPER_DEFAULT_MODEL
- Available Models: ${WHISPER_MODEL_SIZES[@]}
- Model Storage: $WHISPER_MODELS_DIR
- Upload Directory: $WHISPER_UPLOADS_DIR

Supported Features:
- Audio transcription (speech-to-text)
- Audio translation (to English)
- Timestamp generation
- Multiple languages (100+ languages)
- Various audio formats (WAV, MP3, OGG, M4A, FLAC, etc.)

Example Usage:
# Transcribe audio file
$0 --action transcribe --file recording.mp3

# Transcribe with specific language
$0 --action transcribe --file spanish.mp3 --language es

# Translate audio to English
$0 --action transcribe --file foreign.mp3 --task translate

# Use API directly
curl -X POST "$WHISPER_BASE_URL/asr?output=json" \\
  -F "audio_file=@audio.mp3" \\
  -F "task=transcribe"

For more information, visit: https://github.com/openai/whisper
EOF
}

#######################################
# Run test transcription
#######################################
manage::test() {
    log::header "🧪 Testing Whisper Transcription"
    
    # Check if Whisper is running
    if ! manage::is_healthy; then
        log::error "Whisper is not running. Start it with: $0 --action start"
        return 1
    fi
    
    # Check for test audio files
    local test_audio_dir="${SCRIPT_DIR}/tests/audio"
    if [[ ! -d "$test_audio_dir" ]]; then
        log::error "Test audio directory not found: $test_audio_dir"
        log::info "Please ensure test audio files are available"
        return 1
    fi
    
    # Find test audio files
    local test_files=()
    while IFS= read -r -d '' file; do
        test_files+=("$file")
    done < <(find "$test_audio_dir" -type f \( -name "*.mp3" -o -name "*.wav" \) -print0 | sort -z)
    
    if [[ ${#test_files[@]} -eq 0 ]]; then
        log::error "No test audio files found in $test_audio_dir"
        return 1
    fi
    
    log::info "Found ${#test_files[@]} test audio file(s)"
    echo
    
    # Test each file
    local all_passed=true
    
    for test_file in "${test_files[@]}"; do
        local basename=$(basename "$test_file")
        
        echo
        log::info "Testing transcription of: $basename"
        
        # Run transcription
        local result
        # Temporarily set AUDIO_FILE for the transcribe function
        local original_audio_file="$AUDIO_FILE"
        export AUDIO_FILE="$test_file"
        
        if result=$(manage::transcribe 2>&1); then
            # Extract just the text field from JSON output
            local transcribed_text
            # Filter out ALL log lines (anything starting with [) and extract JSON
            local json_result
            json_result=$(echo "$result" | grep -v "^\[" | jq -s '.[0] // empty' 2>/dev/null)
            if [[ -n "$json_result" ]] && echo "$json_result" | jq -e '.text' >/dev/null 2>&1; then
                transcribed_text=$(echo "$json_result" | jq -r '.text' 2>/dev/null)
            else
                transcribed_text="Failed to parse JSON response"
            fi
            
            log::success "✅ Transcription successful"
            log::info "Result: $transcribed_text"
            
            # Check for empty or minimal transcription
            if [[ -z "$transcribed_text" ]] || [[ "$transcribed_text" =~ ^[[:space:]]*$ ]]; then
                log::warn "⚠️  Transcription returned empty or whitespace only"
            fi
        else
            log::error "❌ Transcription failed for $basename"
            all_passed=false
        fi
        
        # Restore original AUDIO_FILE
        export AUDIO_FILE="$original_audio_file"
    done
    
    # Test API directly
    echo
    log::info "Testing direct API access..."
    if [[ ${#test_files[@]} -gt 0 ]]; then
        local first_test_file="${test_files[0]}"
        local api_response
        api_response=$(curl -s -X POST "${WHISPER_BASE_URL}/asr?output=json" \
            -F "audio_file=@$first_test_file" \
            -F "task=transcribe" 2>/dev/null)
        
        if [[ -n "$api_response" ]] && echo "$api_response" | jq . >/dev/null 2>&1; then
            log::success "✅ Direct API access working"
        else
            log::error "❌ Direct API access failed"
            all_passed=false
        fi
    fi
    
    # Test language detection
    echo
    log::info "Testing language detection..."
    if [[ ${#test_files[@]} -gt 0 ]]; then
        local test_speech_file
        # Try to find a speech file for better language detection
        for f in "${test_files[@]}"; do
            if [[ "$f" =~ speech ]]; then
                test_speech_file="$f"
                break
            fi
        done
        test_speech_file="${test_speech_file:-${test_files[0]}}"
        
        local lang_response
        lang_response=$(curl -s -X POST "${WHISPER_BASE_URL}/detect-language" \
            -F "audio_file=@$test_speech_file" 2>/dev/null)
        
        if [[ -n "$lang_response" ]] && echo "$lang_response" | jq . >/dev/null 2>&1; then
            log::success "✅ Language detection working"
            local detected_lang
            detected_lang=$(echo "$lang_response" | jq -r '.language_code' 2>/dev/null)
            log::info "Detected language: $detected_lang"
        else
            log::error "❌ Language detection failed"
            all_passed=false
        fi
    fi
    
    # Summary
    echo
    if $all_passed; then
        log::success "✅ All tests passed! Whisper is working correctly."
    else
        log::error "❌ Some tests failed. Check the logs above."
        return 1
    fi
    
    echo
    log::info "You can now transcribe your own audio files with:"
    log::info "  $0 --action transcribe --file <your-audio-file>"
}

#######################################
# Clean up old upload files
# Arguments:
#   $1: Age in days (default: 1)
#######################################
manage::cleanup_uploads() {
    local age_days="${1:-1}"
    
    log::info "Cleaning up upload files older than $age_days day(s)..."
    
    if [[ ! -d "$WHISPER_UPLOADS_DIR" ]]; then
        log::info "Upload directory does not exist, nothing to clean"
        return 0
    fi
    
    # Count files before cleanup
    local file_count_before
    file_count_before=$(find "$WHISPER_UPLOADS_DIR" -type f 2>/dev/null | wc -l)
    
    # Find and delete old files
    local deleted_count=0
    while IFS= read -r file; do
        if rm -f "$file" 2>/dev/null; then
            ((deleted_count++))
            log::info "Removed: $(basename "$file")"
        fi
    done < <(find "$WHISPER_UPLOADS_DIR" -type f -mtime +"$age_days" 2>/dev/null)
    
    # Count files after cleanup
    local file_count_after
    file_count_after=$(find "$WHISPER_UPLOADS_DIR" -type f 2>/dev/null | wc -l)
    
    if [[ $deleted_count -gt 0 ]]; then
        log::success "✅ Cleaned up $deleted_count old upload file(s)"
        log::info "Files remaining: $file_count_after (was: $file_count_before)"
    else
        log::info "No old files to clean up"
        log::info "Current files: $file_count_after"
    fi
    
    # Show disk usage
    local disk_usage
    disk_usage=$(du -sh "$WHISPER_UPLOADS_DIR" 2>/dev/null | cut -f1)
    log::info "Upload directory size: $disk_usage"
}

#######################################
# Main execution function
#######################################

# ========================================
# GENERATED FUNCTION STUBS
# Generated by fix-interface-compliance.sh on Tue Aug  5 09:25:24 PM EDT 2025
# ========================================

#######################################
# Start the whisper service
# Arguments:
#   None
# Returns:
#   0 - Success
#   1 - Error
#   2 - Already in desired state (skip)
#######################################
manage::start() {
    # Standard interface function - delegates to existing implementation
    manage::start_container
}

#######################################
# Stop the whisper service
# Arguments:
#   None
# Returns:
#   0 - Success
#   1 - Error
#   2 - Already in desired state (skip)
#######################################
manage::stop() {
    # Standard interface function - delegates to existing implementation
    manage::stop_container
}

#######################################
# Show whisper service logs
# Arguments:
#   None
# Returns:
#   0 - Success
#   1 - Error
#######################################
manage::logs() {
    # Standard interface function - delegates to existing implementation
    manage::show_logs
}
manage::main() {
    manage::parse_arguments "$@"
    
    # If no action specified, show usage
    if [[ -z "$ACTION" ]]; then
        log::error "No action specified"
        manage::usage
        exit 1
    fi
    
    case "$ACTION" in
        install)
            manage::install
            ;;
        uninstall)
            manage::uninstall
            ;;
        start)
            manage::start_container
            ;;
        stop)
            manage::stop_container
            ;;
        restart)
            manage::stop_container
            sleep 2
            manage::start_container
            ;;
        status)
            manage::status
            ;;
        logs)
            manage::show_logs
            ;;
        transcribe)
            manage::transcribe
            ;;
        models)
            manage::list_models
            ;;
        info)
            manage::info
            ;;
        test)
            manage::test
            ;;
        cleanup)
            manage::cleanup_uploads
            ;;
        *)
            log::error "Unknown action: $ACTION"
            manage::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    manage::main "$@"
fi