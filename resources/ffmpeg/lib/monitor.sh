#!/bin/bash

# FFmpeg Performance Monitoring
# Tracks conversion speed, resource usage, and performance metrics

set -e

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

source "$RESOURCE_DIR/config/defaults.sh"
source "$SCRIPT_DIR/core.sh"

# Monitoring configuration
METRICS_FILE="/tmp/ffmpeg-metrics.json"
MONITOR_INTERVAL="${FFMPEG_MONITOR_INTERVAL:-5}"
MONITOR_PID_FILE="/tmp/ffmpeg-monitor.pid"

# Initialize metrics file
init_metrics() {
    cat > "$METRICS_FILE" << EOF
{
    "startTime": $(date +%s),
    "conversions": {
        "total": 0,
        "successful": 0,
        "failed": 0,
        "avgDuration": 0,
        "avgSpeed": 0
    },
    "resources": {
        "cpuUsage": [],
        "memoryUsage": [],
        "gpuUsage": []
    },
    "performance": {
        "framesProcessed": 0,
        "bytesProcessed": 0,
        "avgFps": 0,
        "avgBitrate": 0
    }
}
EOF
}

# Update conversion metrics
update_conversion_metrics() {
    local status="$1"  # success or failed
    local duration="$2"
    local speed="$3"
    local input_size="$4"
    local output_size="$5"
    
    local total=$(jq -r '.conversions.total' "$METRICS_FILE")
    local successful=$(jq -r '.conversions.successful' "$METRICS_FILE")
    local failed=$(jq -r '.conversions.failed' "$METRICS_FILE")
    local avg_duration=$(jq -r '.conversions.avgDuration' "$METRICS_FILE")
    local avg_speed=$(jq -r '.conversions.avgSpeed' "$METRICS_FILE")
    
    total=$((total + 1))
    
    if [[ "$status" == "success" ]]; then
        successful=$((successful + 1))
        
        # Update average duration
        if [[ $successful -eq 1 ]]; then
            avg_duration=$duration
            avg_speed=$speed
        else
            avg_duration=$(echo "scale=2; ($avg_duration * ($successful - 1) + $duration) / $successful" | bc)
            avg_speed=$(echo "scale=2; ($avg_speed * ($successful - 1) + $speed) / $successful" | bc)
        fi
        
        # Update bytes processed
        local bytes_processed=$(jq -r '.performance.bytesProcessed' "$METRICS_FILE")
        bytes_processed=$((bytes_processed + input_size + output_size))
        
        jq ".performance.bytesProcessed = $bytes_processed" "$METRICS_FILE" > "${METRICS_FILE}.tmp"
        mv "${METRICS_FILE}.tmp" "$METRICS_FILE"
    else
        failed=$((failed + 1))
    fi
    
    # Update metrics file
    jq ".conversions.total = $total | 
        .conversions.successful = $successful | 
        .conversions.failed = $failed | 
        .conversions.avgDuration = $avg_duration | 
        .conversions.avgSpeed = $avg_speed" "$METRICS_FILE" > "${METRICS_FILE}.tmp"
    mv "${METRICS_FILE}.tmp" "$METRICS_FILE"
}

# Monitor system resources
monitor_resources() {
    while true; do
        # CPU usage
        local cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1 2>/dev/null || echo "0")
        
        # Memory usage
        local mem_usage=$(free -m | awk 'NR==2{printf "%.2f", $3/$2*100}' 2>/dev/null || echo "0")
        
        # GPU usage (if available)
        local gpu_usage="0"
        if command -v nvidia-smi &>/dev/null; then
            gpu_usage=$(nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "0")
        fi
        
        # Update metrics file
        jq ".resources.cpuUsage += [$cpu_usage] | 
            .resources.memoryUsage += [$mem_usage] | 
            .resources.gpuUsage += [$gpu_usage]" "$METRICS_FILE" > "${METRICS_FILE}.tmp"
        mv "${METRICS_FILE}.tmp" "$METRICS_FILE"
        
        # Keep only last 100 samples
        jq '.resources.cpuUsage = .resources.cpuUsage[-100:] | 
            .resources.memoryUsage = .resources.memoryUsage[-100:] | 
            .resources.gpuUsage = .resources.gpuUsage[-100:]' "$METRICS_FILE" > "${METRICS_FILE}.tmp"
        mv "${METRICS_FILE}.tmp" "$METRICS_FILE"
        
        sleep "$MONITOR_INTERVAL"
    done
}

# Analyze FFmpeg progress output
analyze_ffmpeg_output() {
    local log_file="$1"
    local start_time="$2"
    
    # Extract progress information
    local fps=$(grep -oP 'fps=\s*\K[0-9.]+' "$log_file" | tail -1 || echo "0")
    local bitrate=$(grep -oP 'bitrate=\s*\K[0-9.]+' "$log_file" | tail -1 || echo "0")
    local speed=$(grep -oP 'speed=\s*\K[0-9.]+x' "$log_file" | tail -1 | sed 's/x//' || echo "0")
    local frames=$(grep -oP 'frame=\s*\K[0-9]+' "$log_file" | tail -1 || echo "0")
    
    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Update performance metrics
    local total_frames=$(jq -r '.performance.framesProcessed' "$METRICS_FILE")
    total_frames=$((total_frames + frames))
    
    local avg_fps=$(jq -r '.performance.avgFps' "$METRICS_FILE")
    local conversions=$(jq -r '.conversions.successful' "$METRICS_FILE")
    
    if [[ $conversions -gt 0 ]]; then
        avg_fps=$(echo "scale=2; ($avg_fps * ($conversions - 1) + $fps) / $conversions" | bc)
    else
        avg_fps=$fps
    fi
    
    jq ".performance.framesProcessed = $total_frames | 
        .performance.avgFps = $avg_fps | 
        .performance.avgBitrate = $bitrate" "$METRICS_FILE" > "${METRICS_FILE}.tmp"
    mv "${METRICS_FILE}.tmp" "$METRICS_FILE"
    
    echo "$duration $speed"
}

# Get current metrics
get_metrics() {
    if [[ -f "$METRICS_FILE" ]]; then
        # Calculate uptime
        local start_time=$(jq -r '.startTime' "$METRICS_FILE")
        local current_time=$(date +%s)
        local uptime=$((current_time - start_time))
        
        # Add calculated fields
        jq ".uptime = $uptime" "$METRICS_FILE"
    else
        echo '{"error": "No metrics available"}'
    fi
}

# Get performance report
get_performance_report() {
    if [[ ! -f "$METRICS_FILE" ]]; then
        echo "No metrics available"
        return 1
    fi
    
    local total=$(jq -r '.conversions.total' "$METRICS_FILE")
    local successful=$(jq -r '.conversions.successful' "$METRICS_FILE")
    local failed=$(jq -r '.conversions.failed' "$METRICS_FILE")
    local avg_duration=$(jq -r '.conversions.avgDuration' "$METRICS_FILE")
    local avg_speed=$(jq -r '.conversions.avgSpeed' "$METRICS_FILE")
    local frames=$(jq -r '.performance.framesProcessed' "$METRICS_FILE")
    local bytes=$(jq -r '.performance.bytesProcessed' "$METRICS_FILE")
    local avg_fps=$(jq -r '.performance.avgFps' "$METRICS_FILE")
    
    # Calculate average resource usage
    local avg_cpu=$(jq '[.resources.cpuUsage[]] | add/length' "$METRICS_FILE" 2>/dev/null || echo "0")
    local avg_mem=$(jq '[.resources.memoryUsage[]] | add/length' "$METRICS_FILE" 2>/dev/null || echo "0")
    local avg_gpu=$(jq '[.resources.gpuUsage[]] | add/length' "$METRICS_FILE" 2>/dev/null || echo "0")
    
    # Format bytes
    local bytes_mb=$((bytes / 1048576))
    
    cat << EOF
=== FFmpeg Performance Report ===

Conversion Statistics:
  Total Conversions: $total
  Successful: $successful
  Failed: $failed
  Success Rate: $(echo "scale=1; $successful * 100 / $total" | bc 2>/dev/null || echo "0")%
  
Performance Metrics:
  Average Duration: ${avg_duration}s
  Average Speed: ${avg_speed}x realtime
  Frames Processed: $frames
  Data Processed: ${bytes_mb}MB
  Average FPS: $avg_fps
  
Resource Usage (Average):
  CPU: ${avg_cpu}%
  Memory: ${avg_mem}%
  GPU: ${avg_gpu}%

EOF
}

# Start monitoring
start_monitoring() {
    if [[ -f "$MONITOR_PID_FILE" ]]; then
        local pid=$(cat "$MONITOR_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            echo "[INFO] Monitor already running with PID $pid"
            return 0
        fi
    fi
    
    init_metrics
    ( monitor_resources ) &
    local pid=$!
    echo "$pid" > "$MONITOR_PID_FILE"
    echo "[INFO] Started performance monitor with PID $pid"
    
    # Return immediately
    return 0
}

# Stop monitoring
stop_monitoring() {
    if [[ -f "$MONITOR_PID_FILE" ]]; then
        local pid=$(cat "$MONITOR_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f "$MONITOR_PID_FILE"
            echo "[INFO] Stopped performance monitor"
        else
            echo "[WARN] Monitor not running"
        fi
    else
        echo "[WARN] No monitor PID file found"
    fi
}

# Monitor a specific conversion
monitor_conversion() {
    local input_file="$1"
    local output_file="$2"
    local ffmpeg_cmd="$3"
    local log_file="/tmp/ffmpeg_$(date +%s).log"
    
    local start_time=$(date +%s)
    local input_size=$(stat -c%s "$input_file" 2>/dev/null || echo "0")
    
    # Run FFmpeg with progress output
    eval "$ffmpeg_cmd" 2>&1 | tee "$log_file"
    local exit_code=${PIPESTATUS[0]}
    
    if [[ $exit_code -eq 0 ]]; then
        local output_size=$(stat -c%s "$output_file" 2>/dev/null || echo "0")
        local metrics=($(analyze_ffmpeg_output "$log_file" "$start_time"))
        update_conversion_metrics "success" "${metrics[0]}" "${metrics[1]}" "$input_size" "$output_size"
        echo "[SUCCESS] Conversion completed in ${metrics[0]}s at ${metrics[1]}x speed"
    else
        update_conversion_metrics "failed" 0 0 0 0
        echo "[ERROR] Conversion failed"
    fi
    
    rm -f "$log_file"
    return $exit_code
}

# CLI handler
case "${1:-}" in
    "start")
        start_monitoring
        ;;
    "stop")
        stop_monitoring
        ;;
    "status")
        get_metrics
        ;;
    "report")
        get_performance_report
        ;;
    "monitor")
        shift
        monitor_conversion "$@"
        ;;
    *)
        echo "Usage: $0 {start|stop|status|report|monitor <input> <output> <command>}"
        exit 1
        ;;
esac