#!/bin/bash
# FFmpeg core functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
FFMPEG_CORE_DIR="${APP_ROOT}/resources/ffmpeg/lib"

# Source shared utilities (using relative path from lib to scripts)
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Configuration
export FFMPEG_HOME="${FFMPEG_HOME:-$HOME/.ffmpeg}"
export FFMPEG_BIN="${FFMPEG_HOME}/bin/ffmpeg"
export FFMPEG_PROBE_BIN="${FFMPEG_HOME}/bin/ffprobe"
export FFMPEG_VERSION="${FFMPEG_VERSION:-6.1.1}"
# Data directories are now managed by defaults.sh

# Initialize FFmpeg environment
ffmpeg::init() {
    # Create necessary directories
    mkdir -p "$FFMPEG_HOME" "$FFMPEG_DATA_DIR" "$FFMPEG_SCRIPTS_DIR" "$FFMPEG_OUTPUT_DIR"
    
    # Add FFmpeg to PATH if not already there
    if [[ ":$PATH:" != *":$FFMPEG_HOME/bin:"* ]]; then
        export PATH="$FFMPEG_HOME/bin:$PATH"
    fi
    
    return 0
}

# Test FFmpeg installation
ffmpeg::test_installation() {
    if [[ -x "$FFMPEG_BIN" ]]; then
        "$FFMPEG_BIN" -version >/dev/null 2>&1
        return $?
    elif command -v ffmpeg >/dev/null 2>&1; then
        ffmpeg -version >/dev/null 2>&1
        return $?
    fi
    return 1
}

# Get FFmpeg version
ffmpeg::get_version() {
    if [[ -x "$FFMPEG_BIN" ]]; then
        "$FFMPEG_BIN" -version 2>/dev/null | head -n1 | awk '{print $3}'
    elif command -v ffmpeg >/dev/null 2>&1; then
        ffmpeg -version 2>/dev/null | head -n1 | awk '{print $3}'
    else
        echo "not_installed"
    fi
}

# Show resource runtime information (v2.0 contract requirement)
ffmpeg::info() {
    local json_flag="${1:-}"
    local runtime_file="${FFMPEG_CLI_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        log::error "Runtime configuration not found at $runtime_file"
        return 1
    fi
    
    if [[ "$json_flag" == "--json" ]]; then
        cat "$runtime_file"
    else
        log::header "FFmpeg Resource Information"
        log::info "Runtime Configuration:"
        jq -r '
            "  Startup Order: \(.startup_order)",
            "  Startup Timeout: \(.startup_timeout)s",
            "  Startup Time Estimate: \(.startup_time_estimate)",
            "  Dependencies: \(.dependencies | join(", "))",
            "  Recovery Attempts: \(.recovery_attempts)",
            "  Priority: \(.priority)"
        ' "$runtime_file"
    fi
    return 0
}

# Get media file information (renamed from info to avoid conflict)
ffmpeg::media_info() {
    local file_path="$1"
    if [[ -z "$file_path" ]]; then
        log::error "Usage: ffmpeg media-info <file_path>"
        return 1
    fi
    ffmpeg::inject::get_detailed_info "$file_path"
}

ffmpeg::transcode() {
    local file_path="$1"
    if [[ -z "$file_path" ]]; then
        log::error "Usage: ffmpeg transcode <file_path>"
        return 1
    fi
    # Initialize inject system
    ffmpeg::inject::init || return 1
    ffmpeg_inject "$file_path" "transcode"
}

ffmpeg::extract() {
    local file_path="$1"
    if [[ -z "$file_path" ]]; then
        log::error "Usage: ffmpeg extract <file_path>"
        return 1
    fi
    # Initialize inject system
    ffmpeg::inject::init || return 1
    ffmpeg_inject "$file_path" "extract"
}

ffmpeg::logs() {
    local tail_lines="${1:-50}"
    local follow="${2:-false}"
    
    # Initialize config
    ffmpeg::export_config
    
    if [[ ! -d "${FFMPEG_LOGS_DIR}" ]]; then
        log::info "No logs directory found. Run 'resource-ffmpeg manage start' first."
        return 0
    fi
    
    local latest_log=$(find "${FFMPEG_LOGS_DIR}" -name "*.log" -type f -printf '%T@ %p\n' 2>/dev/null | sort -n | tail -1 | cut -d' ' -f2-)
    
    if [[ -z "$latest_log" || ! -f "$latest_log" ]]; then
        log::info "No log files found in ${FFMPEG_LOGS_DIR}"
        return 0
    fi
    
    log::info "Showing logs from: $(basename "$latest_log")"
    
    if [[ "$follow" == "true" || "$follow" == "--follow" ]]; then
        tail -f -n "$tail_lines" "$latest_log"
    else
        tail -n "$tail_lines" "$latest_log"
    fi
}

# Preset library for common conversions
ffmpeg::preset::list() {
    log::header "📚 Available FFmpeg Presets"
    echo ""
    echo "Video Presets:"
    echo "  • web-1080p     - H.264 1080p optimized for web streaming"
    echo "  • web-720p      - H.264 720p optimized for web streaming"
    echo "  • mobile-high   - H.264 720p optimized for mobile devices"
    echo "  • mobile-low    - H.264 480p for low bandwidth"
    echo "  • social-square - 1:1 aspect ratio for Instagram/social media"
    echo "  • social-vertical - 9:16 aspect ratio for TikTok/Reels"
    echo ""
    echo "Audio Presets:"
    echo "  • podcast       - MP3 128kbps mono optimized for speech"
    echo "  • music-high    - MP3 320kbps stereo high quality"
    echo "  • music-standard - MP3 192kbps stereo standard quality"
    echo "  • audiobook     - MP3 64kbps mono for audiobooks"
    echo ""
    echo "Conversion Presets:"
    echo "  • gif-from-video - Convert video to animated GIF"
    echo "  • extract-audio  - Extract audio track from video"
    echo "  • remove-audio   - Remove audio track from video"
    echo "  • compress-50    - Reduce file size by ~50%"
    echo ""
    echo "Usage: resource-ffmpeg preset apply <preset-name> <input-file> [output-file]"
}

# Stream processing capabilities
ffmpeg::stream::capture() {
    local input_url="$1"
    local output_file="${2:-stream_capture_$(date +%Y%m%d_%H%M%S).mp4}"
    local duration="${3:-60}"  # Default 60 seconds
    
    # Initialize config
    ffmpeg::export_config
    
    if [[ -z "$input_url" ]]; then
        log::error "Usage: resource-ffmpeg stream capture <url> [output_file] [duration_seconds]"
        return 1
    fi
    
    log::info "Capturing stream from: $input_url"
    log::info "Duration: ${duration} seconds"
    log::info "Output: $output_file"
    
    # Create output directory if needed
    mkdir -p "$(dirname "$output_file")"
    
    # Capture stream with timeout
    ffmpeg -i "$input_url" -t "$duration" -c copy -bsf:a aac_adtstoasc "$output_file" 2>&1 | \
    while IFS= read -r line; do
        if [[ "$line" =~ frame=.*fps=.*time=.*speed= ]]; then
            printf "\r%s" "$line"
        elif [[ "$line" =~ ^(Error|Invalid|Unable) ]]; then
            log::error "$line"
        fi
    done
    
    if [[ -f "$output_file" ]]; then
        local size=$(du -h "$output_file" | cut -f1)
        log::success "Stream captured: $output_file ($size)"
        return 0
    else
        log::error "Stream capture failed"
        return 1
    fi
}

ffmpeg::stream::transcode() {
    local input_url="$1"
    local output_url="$2"
    local preset="${3:-web-720p}"
    
    # Initialize config
    ffmpeg::export_config
    
    if [[ -z "$input_url" || -z "$output_url" ]]; then
        log::error "Usage: resource-ffmpeg stream transcode <input_url> <output_url> [preset]"
        log::info "Presets: web-1080p, web-720p, mobile-high, mobile-low"
        return 1
    fi
    
    log::info "Transcoding stream:"
    log::info "  Input: $input_url"
    log::info "  Output: $output_url"
    log::info "  Preset: $preset"
    
    # Build FFmpeg command based on preset
    local ffmpeg_cmd="ffmpeg -i \"$input_url\""
    
    case "$preset" in
        web-1080p)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset veryfast -crf 23 -vf scale=1920:1080 -c:a aac -b:a 128k"
            ;;
        web-720p)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset veryfast -crf 23 -vf scale=1280:720 -c:a aac -b:a 128k"
            ;;
        mobile-high)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset veryfast -crf 25 -vf scale=1280:720 -c:a aac -b:a 96k"
            ;;
        mobile-low)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset veryfast -crf 28 -vf scale=854:480 -c:a aac -b:a 64k"
            ;;
        *)
            log::error "Unknown preset: $preset"
            return 1
            ;;
    esac
    
    # Add streaming flags
    ffmpeg_cmd="$ffmpeg_cmd -f flv \"$output_url\""
    
    log::info "Starting stream transcoding (press Ctrl+C to stop)..."
    log::info "Command: $ffmpeg_cmd"
    
    # Execute transcoding
    eval $ffmpeg_cmd
}

ffmpeg::stream::info() {
    local stream_url="$1"
    
    if [[ -z "$stream_url" ]]; then
        log::error "Usage: resource-ffmpeg stream info <url>"
        return 1
    fi
    
    log::info "Analyzing stream: $stream_url"
    
    # Use ffprobe to get stream information
    ffprobe -v quiet -print_format json -show_format -show_streams "$stream_url" 2>/dev/null | \
    jq -r '
        "Stream Information:",
        "==================",
        if .format then
            "Format: \(.format.format_name)",
            "Duration: \(.format.duration // "N/A") seconds",
            "Bitrate: \(.format.bit_rate // "N/A") bps",
            ""
        else empty end,
        if .streams then
            "Streams:",
            (.streams[] | 
                "  [\(.index)] Type: \(.codec_type)",
                "      Codec: \(.codec_name)",
                if .codec_type == "video" then
                    "      Resolution: \(.width)x\(.height)",
                    "      FPS: \(.r_frame_rate)"
                elif .codec_type == "audio" then
                    "      Channels: \(.channels)",
                    "      Sample Rate: \(.sample_rate)"
                else empty end,
                ""
            )
        else empty end
    ' || {
        log::error "Failed to analyze stream. Check if the URL is valid and accessible."
        return 1
    }
}

ffmpeg::preset::apply() {
    local preset_name="$1"
    local input_file="$2"
    local output_file="${3:-}"
    
    # Initialize config
    ffmpeg::export_config
    
    if [[ -z "$preset_name" || -z "$input_file" ]]; then
        log::error "Usage: resource-ffmpeg preset apply <preset-name> <input-file> [output-file]"
        return 1
    fi
    
    if [[ ! -f "$input_file" ]]; then
        log::error "Input file not found: $input_file"
        return 1
    fi
    
    # Auto-generate output filename if not provided
    if [[ -z "$output_file" ]]; then
        local base_name=$(basename "$input_file" | sed 's/\.[^.]*$//')
        local extension=""
        
        case "$preset_name" in
            web-*|mobile-*|social-*|compress-*|remove-audio)
                extension="mp4"
                ;;
            podcast|music-*|audiobook|extract-audio)
                extension="mp3"
                ;;
            gif-from-video)
                extension="gif"
                ;;
            *)
                extension="mp4"
                ;;
        esac
        
        output_file="${FFMPEG_OUTPUT_DIR}/${base_name}_${preset_name}.${extension}"
    fi
    
    # Create output directory if needed
    mkdir -p "$(dirname "$output_file")"
    
    local ffmpeg_cmd="ffmpeg -i \"$input_file\""
    
    case "$preset_name" in
        # Video presets
        web-1080p)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset fast -crf 23 -vf scale=1920:1080 -c:a aac -b:a 128k"
            ;;
        web-720p)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset fast -crf 23 -vf scale=1280:720 -c:a aac -b:a 128k"
            ;;
        mobile-high)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset fast -crf 25 -vf scale=1280:720 -c:a aac -b:a 96k"
            ;;
        mobile-low)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset fast -crf 28 -vf scale=854:480 -c:a aac -b:a 64k"
            ;;
        social-square)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset fast -crf 23 -vf \"scale=1080:1080:force_original_aspect_ratio=increase,crop=1080:1080\" -c:a aac -b:a 128k"
            ;;
        social-vertical)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset fast -crf 23 -vf \"scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920\" -c:a aac -b:a 128k"
            ;;
        
        # Audio presets
        podcast)
            ffmpeg_cmd="$ffmpeg_cmd -c:a mp3 -b:a 128k -ac 1 -ar 44100"
            ;;
        music-high)
            ffmpeg_cmd="$ffmpeg_cmd -c:a mp3 -b:a 320k -ac 2 -ar 44100"
            ;;
        music-standard)
            ffmpeg_cmd="$ffmpeg_cmd -c:a mp3 -b:a 192k -ac 2 -ar 44100"
            ;;
        audiobook)
            ffmpeg_cmd="$ffmpeg_cmd -c:a mp3 -b:a 64k -ac 1 -ar 22050"
            ;;
        
        # Conversion presets
        gif-from-video)
            ffmpeg_cmd="$ffmpeg_cmd -vf \"fps=10,scale=480:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse\" -loop 0"
            ;;
        extract-audio)
            ffmpeg_cmd="$ffmpeg_cmd -vn -c:a mp3 -b:a 192k"
            ;;
        remove-audio)
            ffmpeg_cmd="$ffmpeg_cmd -c:v copy -an"
            ;;
        compress-50)
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset slow -crf 28 -c:a aac -b:a 96k"
            ;;
        
        *)
            log::error "Unknown preset: $preset_name"
            log::info "Run 'resource-ffmpeg preset list' to see available presets"
            return 1
            ;;
    esac
    
    ffmpeg_cmd="$ffmpeg_cmd -y \"$output_file\""
    
    log::info "Applying preset: $preset_name"
    log::info "Input: $input_file"
    log::info "Output: $output_file"
    log::info "Command: $ffmpeg_cmd"
    
    # Execute the command
    eval $ffmpeg_cmd 2>&1 | while IFS= read -r line; do
        if [[ "$line" =~ frame=.*fps=.*time=.*speed= ]]; then
            # Progress line - show it on same line
            printf "\r%s" "$line"
        elif [[ "$line" =~ ^(Error|Invalid|Unknown) ]]; then
            log::error "$line"
        fi
    done
    
    if [[ -f "$output_file" ]]; then
        local output_size=$(du -h "$output_file" | cut -f1)
        log::success "Conversion complete! Output: $output_file ($output_size)"
        return 0
    else
        log::error "Conversion failed - output file not created"
        return 1
    fi
}

################################################################################
# START: Web Interface Functions
################################################################################

# Web interface PID file
WEB_PID_FILE="/tmp/ffmpeg-web.pid"

# Start web interface server
ffmpeg::web::start() {
    local port="${1:-${FFMPEG_API_PORT:-8080}}"
    
    if [[ -f "$WEB_PID_FILE" ]]; then
        local pid=$(cat "$WEB_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            log::info "Web interface already running on port $port (PID: $pid)"
            return 0
        fi
    fi
    
    log::info "Starting FFmpeg web interface on port $port"
    
    # Start the API server in background
    nohup "${FFMPEG_CLI_DIR}/lib/api.sh" start > /tmp/ffmpeg-web.log 2>&1 &
    local pid=$!
    echo "$pid" > "$WEB_PID_FILE"
    
    # Wait for server to start
    local max_wait=10
    local waited=0
    while [[ $waited -lt $max_wait ]]; do
        if timeout 1 curl -sf "http://localhost:$port" >/dev/null 2>&1; then
            log::success "Web interface started successfully on http://localhost:$port"
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
    done
    
    log::error "Failed to start web interface"
    return 1
}

# Stop web interface server
ffmpeg::web::stop() {
    if [[ -f "$WEB_PID_FILE" ]]; then
        local pid=$(cat "$WEB_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f "$WEB_PID_FILE"
            log::success "Web interface stopped"
        else
            log::warn "Web interface not running"
        fi
    else
        log::warn "No web interface PID file found"
    fi
}

# Check web interface status
ffmpeg::web::status() {
    local port="${FFMPEG_API_PORT:-8080}"
    
    if [[ -f "$WEB_PID_FILE" ]]; then
        local pid=$(cat "$WEB_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            log::success "Web interface running on port $port (PID: $pid)"
            echo "URL: http://localhost:$port"
            return 0
        fi
    fi
    
    log::info "Web interface not running"
    return 1
}

################################################################################
# END: Web Interface Functions
################################################################################

################################################################################
# START: Performance Monitoring Functions
################################################################################

# Start performance monitoring
ffmpeg::monitor::start() {
    "${FFMPEG_CLI_DIR}/lib/monitor.sh" start
}

# Stop performance monitoring
ffmpeg::monitor::stop() {
    "${FFMPEG_CLI_DIR}/lib/monitor.sh" stop
}

# Get current metrics
ffmpeg::monitor::status() {
    "${FFMPEG_CLI_DIR}/lib/monitor.sh" status
}

# Generate performance report
ffmpeg::monitor::report() {
    "${FFMPEG_CLI_DIR}/lib/monitor.sh" report
}

################################################################################
# END: Performance Monitoring Functions
################################################################################

# Export functions
export -f ffmpeg::init
export -f ffmpeg::test_installation
export -f ffmpeg::get_version
export -f ffmpeg::info
export -f ffmpeg::transcode
export -f ffmpeg::extract
export -f ffmpeg::logs
export -f ffmpeg::preset::list
export -f ffmpeg::preset::apply
export -f ffmpeg::stream::capture
export -f ffmpeg::stream::transcode
export -f ffmpeg::stream::info
export -f ffmpeg::web::start
export -f ffmpeg::web::stop
export -f ffmpeg::web::status
export -f ffmpeg::monitor::start
export -f ffmpeg::monitor::stop
export -f ffmpeg::monitor::status
export -f ffmpeg::monitor::report