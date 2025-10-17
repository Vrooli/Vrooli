#!/bin/bash

# FFmpeg API Server
# Provides RESTful endpoints for media processing

set -e

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

source "$RESOURCE_DIR/config/defaults.sh"
source "$SCRIPT_DIR/core.sh"

# API Configuration - Use port 8097 as default to avoid conflicts
API_PORT="${FFMPEG_API_PORT:-8097}"
WEB_ROOT="$RESOURCE_DIR/web"
UPLOAD_DIR="/tmp/ffmpeg-uploads"
OUTPUT_DIR="/tmp/ffmpeg-output"
STATS_FILE="/tmp/ffmpeg-stats.json"

# Ensure directories exist
mkdir -p "$UPLOAD_DIR" "$OUTPUT_DIR"

# Initialize stats file
if [[ ! -f "$STATS_FILE" ]]; then
    echo '{"activeJobs":0,"completedJobs":0,"totalProcessingTime":0}' > "$STATS_FILE"
fi

# Update stats
update_stats() {
    local field="$1"
    local value="$2"
    local current=$(jq -r ".$field // 0" "$STATS_FILE")
    local new=$((current + value))
    jq ".$field = $new" "$STATS_FILE" > "${STATS_FILE}.tmp" && mv "${STATS_FILE}.tmp" "$STATS_FILE"
}

# Get system stats
get_system_stats() {
    local cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1 2>/dev/null || echo "0")
    local mem_usage=$(free -m | awk 'NR==2{printf "%.0f", $3}' 2>/dev/null || echo "0")
    local gpu_available="false"
    
    if command -v nvidia-smi &>/dev/null; then
        gpu_available="true"
    fi
    
    local completed=$(jq -r '.completedJobs // 0' "$STATS_FILE")
    local total_time=$(jq -r '.totalProcessingTime // 0' "$STATS_FILE")
    local avg_time=0
    if [[ $completed -gt 0 ]]; then
        avg_time=$((total_time / completed))
    fi
    
    jq -n \
        --argjson active "$(jq -r '.activeJobs // 0' "$STATS_FILE")" \
        --argjson completed "$completed" \
        --arg cpu "$cpu_usage" \
        --arg mem "$mem_usage" \
        --argjson gpu "$gpu_available" \
        --argjson avg "$avg_time" \
        '{activeJobs: $active, completedJobs: $completed, cpuUsage: $cpu, memoryUsage: $mem, gpuAvailable: $gpu, avgProcessingTime: $avg}'
}

# Process media conversion
process_convert() {
    local input_file="$1"
    local preset="$2"
    local custom_options="$3"
    local output_file="${input_file%.*}_converted.mp4"
    
    update_stats "activeJobs" 1
    local start_time=$(date +%s)
    
    local ffmpeg_cmd="ffmpeg -i \"$input_file\""
    
    # Apply preset
    case "$preset" in
        "mp4-h264")
            ffmpeg_cmd="$ffmpeg_cmd -c:v libx264 -preset fast -crf 23 -c:a aac"
            output_file="${input_file%.*}_h264.mp4"
            ;;
        "webm-vp9")
            ffmpeg_cmd="$ffmpeg_cmd -c:v libvpx-vp9 -crf 30 -b:v 0"
            output_file="${input_file%.*}_vp9.webm"
            ;;
        "mp3-320")
            ffmpeg_cmd="$ffmpeg_cmd -ab 320k"
            output_file="${input_file%.*}.mp3"
            ;;
        "gif")
            ffmpeg_cmd="$ffmpeg_cmd -vf \"fps=10,scale=320:-1:flags=lanczos\" -c:v gif"
            output_file="${input_file%.*}.gif"
            ;;
        "thumbnail")
            ffmpeg_cmd="$ffmpeg_cmd -vframes 1 -vf \"select=eq(n\\,0)\""
            output_file="${input_file%.*}_thumb.jpg"
            ;;
    esac
    
    # Add custom options if provided
    if [[ -n "$custom_options" ]]; then
        ffmpeg_cmd="$ffmpeg_cmd $custom_options"
    fi
    
    ffmpeg_cmd="$ffmpeg_cmd \"$output_file\" -y"
    
    # Execute conversion
    if eval "$ffmpeg_cmd" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        update_stats "activeJobs" -1
        update_stats "completedJobs" 1
        update_stats "totalProcessingTime" "$duration"
        
        echo "{\"success\": true, \"output\": \"$output_file\", \"duration\": $duration}"
    else
        update_stats "activeJobs" -1
        echo "{\"success\": false, \"error\": \"Conversion failed\"}"
    fi
}

# Process extraction
process_extract() {
    local input_file="$1"
    local extract_type="$2"
    local output_file=""
    
    update_stats "activeJobs" 1
    local start_time=$(date +%s)
    
    case "$extract_type" in
        "audio")
            output_file="${input_file%.*}.mp3"
            ffmpeg -i "$input_file" -vn -ab 192k "$output_file" -y 2>&1
            ;;
        "audio-wav")
            output_file="${input_file%.*}.wav"
            ffmpeg -i "$input_file" -vn "$output_file" -y 2>&1
            ;;
        "frames")
            output_file="${input_file%.*}_frames"
            mkdir -p "$output_file"
            ffmpeg -i "$input_file" -vf fps=1 "$output_file/frame_%04d.jpg" -y 2>&1
            ;;
        "thumbnail")
            output_file="${input_file%.*}_thumb.jpg"
            ffmpeg -i "$input_file" -vframes 1 -vf "select=eq(n\\,0)" "$output_file" -y 2>&1
            ;;
    esac
    
    if [[ $? -eq 0 ]]; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        update_stats "activeJobs" -1
        update_stats "completedJobs" 1
        update_stats "totalProcessingTime" "$duration"
        
        echo "{\"success\": true, \"output\": \"$output_file\", \"duration\": $duration}"
    else
        update_stats "activeJobs" -1
        echo "{\"success\": false, \"error\": \"Extraction failed\"}"
    fi
}

# Get media info
get_media_info() {
    local input_file="$1"
    
    local info=$(ffprobe -v quiet -print_format json -show_format -show_streams "$input_file" 2>/dev/null)
    
    if [[ -n "$info" ]]; then
        echo "$info"
    else
        echo "{\"error\": \"Could not read media file\"}"
    fi
}

# Process stream
process_stream() {
    local url="$1"
    local action="$2"
    local duration="${3:-60}"
    
    update_stats "activeJobs" 1
    
    case "$action" in
        "info")
            local info=$(ffprobe -v quiet -print_format json -show_format -show_streams "$url" 2>/dev/null)
            update_stats "activeJobs" -1
            if [[ -n "$info" ]]; then
                echo "$info"
            else
                echo "{\"error\": \"Could not read stream\"}"
            fi
            ;;
        "capture")
            local output_file="$OUTPUT_DIR/stream_capture_$(date +%s).mp4"
            if timeout "$duration" ffmpeg -i "$url" -t "$duration" -c copy "$output_file" -y 2>&1; then
                update_stats "activeJobs" -1
                update_stats "completedJobs" 1
                echo "{\"success\": true, \"output\": \"$output_file\"}"
            else
                update_stats "activeJobs" -1
                echo "{\"success\": false, \"error\": \"Stream capture failed\"}"
            fi
            ;;
        "transcode")
            local output_file="$OUTPUT_DIR/stream_transcode_$(date +%s).mp4"
            if timeout "$duration" ffmpeg -i "$url" -t "$duration" -c:v libx264 -preset veryfast "$output_file" -y 2>&1; then
                update_stats "activeJobs" -1
                update_stats "completedJobs" 1
                echo "{\"success\": true, \"output\": \"$output_file\"}"
            else
                update_stats "activeJobs" -1
                echo "{\"success\": false, \"error\": \"Stream transcode failed\"}"
            fi
            ;;
    esac
}

# Start API server using the new Python implementation
start_api_server() {
    # Use the improved API server implementation
    local api_script="$SCRIPT_DIR/api_server.py"
    
    if [[ ! -f "$api_script" ]]; then
        echo "[ERROR] API server script not found: $api_script"
        return 1
    fi
    
    # Make executable
    chmod +x "$api_script"
    
    # Set environment variables for the Python server
    export API_PORT="$API_PORT"
    export WEB_ROOT="$WEB_ROOT"
    export UPLOAD_DIR="$UPLOAD_DIR"
    export OUTPUT_DIR="$OUTPUT_DIR"
    export SCRIPT_DIR="$SCRIPT_DIR"
    export MAX_FILE_SIZE="${MAX_FILE_SIZE:-524288000}"  # 500MB default
    
    # Start in background
    python3 "$api_script" &
    local server_pid=$!
    
    # Wait a moment for server to start
    sleep 2
    
    # Check if server started successfully
    if kill -0 $server_pid 2>/dev/null; then
        echo "[SUCCESS] FFmpeg API server started on port $API_PORT (PID: $server_pid)"
        echo $server_pid > /tmp/ffmpeg-api.pid
        return 0
    else
        echo "[ERROR] Failed to start FFmpeg API server"
        return 1
    fi
}

# Stop API server
stop_api_server() {
    if [[ -f /tmp/ffmpeg-api.pid ]]; then
        local pid=$(cat /tmp/ffmpeg-api.pid)
        if kill -0 $pid 2>/dev/null; then
            kill $pid
            echo "[INFO] FFmpeg API server stopped (PID: $pid)"
            rm -f /tmp/ffmpeg-api.pid
        else
            echo "[WARNING] FFmpeg API server not running"
            rm -f /tmp/ffmpeg-api.pid
        fi
    else
        echo "[INFO] No FFmpeg API server PID file found"
    fi
}

# CLI handler
case "${1:-}" in
    "start")
        echo "[INFO] Starting FFmpeg API server on port $API_PORT"
        start_api_server
        ;;
    "stop")
        echo "[INFO] Stopping FFmpeg API server"
        stop_api_server
        ;;
    "restart")
        echo "[INFO] Restarting FFmpeg API server"
        stop_api_server
        sleep 1
        start_api_server
        ;;
    "status")
        if [[ -f /tmp/ffmpeg-api.pid ]]; then
            pid=$(cat /tmp/ffmpeg-api.pid)
            if kill -0 $pid 2>/dev/null; then
                echo "[INFO] FFmpeg API server is running (PID: $pid, Port: $API_PORT)"
            else
                echo "[WARNING] FFmpeg API server PID file exists but process not running"
            fi
        else
            echo "[INFO] FFmpeg API server is not running"
        fi
        ;;
    "get_stats")
        get_system_stats
        ;;
    "convert")
        process_convert "$2" "$3" "$4"
        ;;
    "extract")
        process_extract "$2" "$3"
        ;;
    "info")
        get_media_info "$2"
        ;;
    "stream")
        process_stream "$2" "$3" "$4"
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|get_stats|convert|extract|info|stream}"
        exit 1
        ;;
esac