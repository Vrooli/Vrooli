#!/usr/bin/env bash
set -euo pipefail

# Whisper Speech-to-Text Resource Setup and Management
# This script handles installation, configuration, and management of Whisper using Docker

DESCRIPTION="Install and manage Whisper speech-to-text service using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Whisper configuration
readonly WHISPER_PORT="${WHISPER_CUSTOM_PORT:-$(resources::get_default_port "whisper")}"
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
readonly WHISPER_DEFAULT_MODEL="${WHISPER_DEFAULT_MODEL:-large}"

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
whisper::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|transcribe|models|info|test" \
        --default "install"
    
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
        whisper::usage
        exit 0
    fi
    
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
whisper::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                      # Install Whisper with default large model"
    echo "  $0 --action install --model small        # Install with small model"
    echo "  $0 --action install --gpu yes            # Install with GPU support"
    echo "  $0 --action status                       # Check Whisper status"
    echo "  $0 --action transcribe --file audio.mp3  # Transcribe an audio file"
    echo "  $0 --action models                       # List available models"
    echo "  $0 --action uninstall                    # Remove Whisper"
    echo
    echo "Model Sizes:"
    for model in "${WHISPER_MODEL_SIZES[@]}"; do
        printf "  %-10s : %s GB\n" "$model" "${MODEL_SIZES[$model]}"
    done
}

#######################################
# Check if Whisper container exists
# Returns: 0 if exists, 1 otherwise
#######################################
whisper::container_exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${WHISPER_CONTAINER_NAME}$"
}

#######################################
# Check if Whisper is running
# Returns: 0 if running, 1 otherwise
#######################################
whisper::is_running() {
    docker ps --format "{{.Names}}" | grep -q "^${WHISPER_CONTAINER_NAME}$"
}

#######################################
# Check if Whisper API is healthy
# Returns: 0 if responsive, 1 otherwise
#######################################
whisper::is_healthy() {
    # The Whisper API doesn't have a /health endpoint, so we check the root
    # which returns a 307 redirect to /docs when healthy
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$WHISPER_BASE_URL/")
    [[ "$response_code" == "307" ]] || [[ "$response_code" == "200" ]]
}

#######################################
# Validate model size
#######################################
whisper::validate_model() {
    local model="$1"
    for valid_model in "${WHISPER_MODEL_SIZES[@]}"; do
        if [[ "$model" == "$valid_model" ]]; then
            return 0
        fi
    done
    return 1
}

#######################################
# Create necessary directories
#######################################
whisper::create_directories() {
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
# Pull Docker image
#######################################
whisper::pull_image() {
    local image="$1"
    
    log::info "Pulling Whisper Docker image: $image"
    log::info "This may take several minutes..."
    
    if docker pull "$image"; then
        log::success "Docker image pulled successfully"
        return 0
    else
        log::error "Failed to pull Docker image"
        return 1
    fi
}

#######################################
# Start Whisper container
#######################################
whisper::start_container() {
    if whisper::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Whisper is already running"
        return 0
    fi
    
    # Stop existing container if force is specified
    if whisper::is_running && [[ "$FORCE" == "yes" ]]; then
        log::info "Stopping existing container (force specified)..."
        whisper::stop_container
    fi
    
    # Remove existing container if it exists but is not running
    if whisper::container_exists && ! whisper::is_running; then
        log::info "Removing existing stopped container..."
        docker rm "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Determine which image to use
    local image="$WHISPER_CPU_IMAGE"
    if [[ "$USE_GPU" == "yes" ]]; then
        if ! system::is_command "nvidia-smi"; then
            log::warn "GPU requested but nvidia-smi not found. Using CPU image instead."
        else
            image="$WHISPER_IMAGE"
            log::info "Using GPU-accelerated image"
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
            if whisper::is_healthy; then
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
whisper::stop_container() {
    if ! whisper::is_running; then
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
whisper::show_logs() {
    if ! whisper::container_exists; then
        log::error "Whisper container does not exist"
        return 1
    fi
    
    docker logs -f "$WHISPER_CONTAINER_NAME"
}

#######################################
# Transcribe audio file
#######################################
whisper::transcribe() {
    if [[ -z "${AUDIO_FILE:-}" ]]; then
        log::error "No audio file specified. Use --file <path>"
        return 1
    fi
    
    if [[ ! -f "$AUDIO_FILE" ]]; then
        log::error "Audio file not found: $AUDIO_FILE"
        return 1
    fi
    
    if ! whisper::is_healthy; then
        log::error "Whisper service is not available"
        return 1
    fi
    
    log::info "Transcribing file: $AUDIO_FILE"
    log::info "Model: $MODEL_SIZE, Task: $TASK"
    
    # Prepare curl command - output must be query parameter
    local curl_cmd=(
        curl -X POST
        "${WHISPER_BASE_URL}/asr?output=json"
        -F "audio_file=@${AUDIO_FILE}"
        -F "task=${TASK}"
    )
    
    # Add language if specified
    if [[ "$LANGUAGE" != "auto" ]]; then
        curl_cmd+=(-F "language=${LANGUAGE}")
    fi
    
    # Execute transcription
    local response
    if response=$("${curl_cmd[@]}" 2>/dev/null); then
        # Check if response is valid JSON
        if echo "$response" | jq . >/dev/null 2>&1; then
            echo "$response" | jq .
        else
            echo "$response"
        fi
    else
        log::error "Transcription failed"
        return 1
    fi
}

#######################################
# Update Vrooli configuration
#######################################
whisper::update_config() {
    local additional_config=$(cat <<EOF
{
    "modelSize": "$MODEL_SIZE",
    "supportedFormats": ["wav", "mp3", "ogg", "m4a", "flac", "aac", "wma"],
    "api": {
        "transcribe": "/asr",
        "health": "/health"
    },
    "capabilities": {
        "transcription": true,
        "translation": true,
        "timestamps": true,
        "multiLanguage": true
    }
}
EOF
)
    
    resources::update_config "ai" "whisper" "$WHISPER_BASE_URL" "$additional_config"
}

#######################################
# Complete Whisper installation
#######################################
whisper::install() {
    log::header "🎤 Installing Whisper Speech-to-Text"
    
    # Start rollback context
    resources::start_rollback_context "install_whisper"
    
    # Check if already installed
    if whisper::container_exists && whisper::is_running && [[ "$FORCE" != "yes" ]]; then
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
    if ! whisper::validate_model "$MODEL_SIZE"; then
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
                read -p "Continue anyway? (y/N): " -r
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
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
    if ! whisper::create_directories; then
        return 1
    fi
    
    # Add rollback for directories
    resources::add_rollback_action \
        "Remove Whisper directories" \
        "rm -rf \"$WHISPER_DATA_DIR\"" \
        10
    
    # Determine image to use
    local image="$WHISPER_CPU_IMAGE"
    if [[ "$USE_GPU" == "yes" ]]; then
        image="$WHISPER_IMAGE"
    fi
    
    # Pull Docker image
    if ! whisper::pull_image "$image"; then
        return 1
    fi
    
    # Start container
    if ! whisper::start_container; then
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
    if ! whisper::update_config; then
        log::warn "Failed to update Vrooli configuration"
        log::info "Whisper is installed but may need manual configuration"
    fi
    
    log::success "✅ Whisper installation completed successfully"
    
    # Show status
    echo
    whisper::status
}

#######################################
# Uninstall Whisper
#######################################
whisper::uninstall() {
    log::header "🗑️  Uninstalling Whisper"
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will remove the Whisper container and optionally its data"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop container
    if whisper::is_running; then
        whisper::stop_container
    fi
    
    # Remove container
    if whisper::container_exists; then
        log::info "Removing Whisper container..."
        docker rm "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Ask about data removal
    if [[ -d "$WHISPER_DATA_DIR" ]]; then
        if ! flow::is_yes "$YES"; then
            read -p "Remove Whisper data directory? (y/N): " -r
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                rm -rf "$WHISPER_DATA_DIR"
                log::info "Data directory removed"
            fi
        fi
    fi
    
    # Remove from Vrooli config
    resources::remove_config "ai" "whisper"
    
    log::success "✅ Whisper uninstalled successfully"
}

#######################################
# Show Whisper status
#######################################
whisper::status() {
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
    if whisper::container_exists; then
        if whisper::is_running; then
            log::success "✅ Whisper container is running"
            
            # Get container stats
            local stats
            stats=$(docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" "$WHISPER_CONTAINER_NAME" 2>/dev/null || echo "")
            if [[ -n "$stats" ]]; then
                echo
                echo "$stats"
            fi
            
            # Check health
            if whisper::is_healthy; then
                log::success "✅ API is healthy and responding"
            else
                log::warn "⚠️  API health check failed"
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
            log::info "  API Docs: $WHISPER_BASE_URL/docs"
            
            echo
            log::info "Configuration:"
            log::info "  Model Size: $MODEL_SIZE (~${MODEL_SIZES[$MODEL_SIZE]}GB)"
            log::info "  GPU Enabled: $USE_GPU"
            log::info "  Models Directory: $WHISPER_MODELS_DIR"
            log::info "  Uploads Directory: $WHISPER_UPLOADS_DIR"
            
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
whisper::list_models() {
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
whisper::info() {
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
- Health Check: GET $WHISPER_BASE_URL/health

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
whisper::test() {
    log::header "🧪 Testing Whisper Transcription"
    
    # Check if Whisper is running
    if ! whisper::is_healthy; then
        log::error "Whisper is not running. Start it with: $0 --action start"
        return 1
    fi
    
    # Check if espeak is available
    if ! system::is_command "espeak"; then
        log::warn "espeak not found. Installing it..."
        local pm
        pm=$(system::detect_pm)
        
        case "$pm" in
            "apt-get")
                sudo apt-get update && sudo apt-get install -y espeak || {
                    log::error "Failed to install espeak"
                    return 1
                }
                ;;
            *)
                log::error "Please install espeak manually to run tests"
                return 1
                ;;
        esac
    fi
    
    # Create test directory
    local test_dir="/tmp/whisper-test-$$"
    mkdir -p "$test_dir"
    
    log::info "Creating test audio files..."
    
    # Create various test files
    echo "Hello, this is a test of the Whisper speech recognition system." | \
        espeak --stdout > "$test_dir/test1.wav" 2>/dev/null
    
    echo "The quick brown fox jumps over the lazy dog." | \
        espeak --stdout -s 150 > "$test_dir/test2.wav" 2>/dev/null
    
    echo "Testing numbers: one two three four five six seven eight nine ten." | \
        espeak --stdout > "$test_dir/test3.wav" 2>/dev/null
    
    # Test each file
    local test_files=("test1.wav" "test2.wav" "test3.wav")
    local all_passed=true
    
    for test_file in "${test_files[@]}"; do
        local full_path="$test_dir/$test_file"
        
        echo
        log::info "Testing transcription of: $test_file"
        log::info "Expected content varies slightly from espeak pronunciation"
        
        # Run transcription
        local result
        # Temporarily set AUDIO_FILE for the transcribe function
        local original_audio_file="$AUDIO_FILE"
        export AUDIO_FILE="$full_path"
        
        if result=$(whisper::transcribe 2>/dev/null); then
            # Extract just the text field
            local transcribed_text
            transcribed_text=$(echo "$result" | jq -r '.text' 2>/dev/null || echo "$result")
            
            log::success "✅ Transcription successful"
            log::info "Result: $transcribed_text"
        else
            log::error "❌ Transcription failed for $test_file"
            all_passed=false
        fi
        
        # Restore original AUDIO_FILE
        export AUDIO_FILE="$original_audio_file"
    done
    
    # Test API directly
    echo
    log::info "Testing direct API access..."
    local api_response
    api_response=$(curl -s -X POST "${WHISPER_BASE_URL}/asr?output=json" \
        -F "audio_file=@$test_dir/test1.wav" \
        -F "task=transcribe" 2>/dev/null)
    
    if [[ -n "$api_response" ]] && echo "$api_response" | jq . >/dev/null 2>&1; then
        log::success "✅ Direct API access working"
    else
        log::error "❌ Direct API access failed"
        all_passed=false
    fi
    
    # Cleanup
    rm -rf "$test_dir"
    
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
# Main execution function
#######################################
whisper::main() {
    whisper::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            whisper::install
            ;;
        "uninstall")
            whisper::uninstall
            ;;
        "start")
            whisper::start_container
            ;;
        "stop")
            whisper::stop_container
            ;;
        "restart")
            whisper::stop_container
            sleep 2
            whisper::start_container
            ;;
        "status")
            whisper::status
            ;;
        "logs")
            whisper::show_logs
            ;;
        "transcribe")
            whisper::transcribe
            ;;
        "models")
            whisper::list_models
            ;;
        "info")
            whisper::info
            ;;
        "test")
            whisper::test
            ;;
        *)
            log::error "Unknown action: $ACTION"
            whisper::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    whisper::main "$@"
fi