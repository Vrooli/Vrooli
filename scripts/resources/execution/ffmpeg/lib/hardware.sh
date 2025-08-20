#!/bin/bash

# Hardware acceleration detection and configuration for FFmpeg

# Get the directory of this lib file
FFMPEG_HW_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "${FFMPEG_HW_LIB_DIR}/../../../../lib/utils/log.sh"

#######################################
# Detect available hardware acceleration
#######################################
ffmpeg::hardware::detect() {
    local available_hw=()
    local recommended_hw=""
    
    log::info "🔍 Detecting hardware acceleration capabilities..."
    
    # Check for NVIDIA GPU
    if command -v nvidia-smi &>/dev/null; then
        if nvidia-smi &>/dev/null; then
            available_hw+=("nvenc")
            if [[ -z "$recommended_hw" ]]; then
                recommended_hw="nvenc"
            fi
            log::info "  ✅ NVIDIA GPU detected - NVENC available"
        fi
    fi
    
    # Check for Intel GPU/QSV
    if [[ -d "/sys/class/drm" ]]; then
        for card in /sys/class/drm/card*; do
            if [[ -f "$card/device/vendor" ]]; then
                local vendor=$(cat "$card/device/vendor" 2>/dev/null)
                if [[ "$vendor" == "0x8086" ]]; then
                    available_hw+=("qsv")
                    if [[ -z "$recommended_hw" ]]; then
                        recommended_hw="qsv"
                    fi
                    log::info "  ✅ Intel GPU detected - QuickSync (QSV) available"
                    break
                fi
            fi
        done
    fi
    
    # Check for VAAPI (Video Acceleration API)
    if [[ -e "/dev/dri/renderD128" ]]; then
        available_hw+=("vaapi")
        if [[ -z "$recommended_hw" ]]; then
            recommended_hw="vaapi"
        fi
        log::info "  ✅ VAAPI device found - VAAPI acceleration available"
    fi
    
    # Check for OpenCL
    if command -v clinfo &>/dev/null && clinfo &>/dev/null; then
        available_hw+=("opencl")
        log::info "  ✅ OpenCL detected"
    fi
    
    # Test hardware acceleration support in FFmpeg
    local ffmpeg_hw=()
    if command -v ffmpeg &>/dev/null; then
        local hw_accel_list=$(ffmpeg -hwaccels 2>/dev/null | tail -n +2)
        
        for hw in "${available_hw[@]}"; do
            if echo "$hw_accel_list" | grep -q "$hw"; then
                ffmpeg_hw+=("$hw")
            fi
        done
    fi
    
    if [[ ${#ffmpeg_hw[@]} -eq 0 ]]; then
        log::warn "  ⚠️  No hardware acceleration available in FFmpeg"
        recommended_hw="none"
    else
        log::success "  🚀 Available in FFmpeg: ${ffmpeg_hw[*]}"
    fi
    
    # Export results
    export FFMPEG_DETECTED_HW_ACCEL="${ffmpeg_hw[*]}"
    export FFMPEG_RECOMMENDED_HW_ACCEL="$recommended_hw"
    
    return 0
}

#######################################
# Test hardware acceleration performance
#######################################
ffmpeg::hardware::benchmark() {
    local hw_method="${1:-auto}"
    
    if [[ "$hw_method" == "auto" ]]; then
        ffmpeg::hardware::detect
        hw_method="$FFMPEG_RECOMMENDED_HW_ACCEL"
    fi
    
    if [[ "$hw_method" == "none" ]]; then
        log::info "No hardware acceleration to benchmark"
        return 0
    fi
    
    log::info "🔬 Benchmarking hardware acceleration: $hw_method"
    
    # Create temporary test file
    local temp_dir=$(mktemp -d)
    local test_input="${temp_dir}/hw_test_input.mp4"
    local test_output_hw="${temp_dir}/hw_test_output.mp4"
    local test_output_sw="${temp_dir}/sw_test_output.mp4"
    
    trap "rm -rf $temp_dir" EXIT
    
    # Generate test video
    log::info "Creating test video..."
    ffmpeg -f lavfi -i testsrc=duration=3:size=1920x1080:rate=30 \
        -c:v libx264 -pix_fmt yuv420p -y "$test_input" 2>/dev/null || {
        log::error "Failed to create test video"
        return 1
    }
    
    # Test hardware acceleration
    log::info "Testing hardware acceleration ($hw_method)..."
    local hw_start=$(date +%s.%N)
    
    case "$hw_method" in
        "nvenc")
            ffmpeg -hwaccel cuda -i "$test_input" -c:v h264_nvenc -preset fast \
                -y "$test_output_hw" 2>/dev/null
            ;;
        "qsv")
            ffmpeg -hwaccel qsv -i "$test_input" -c:v h264_qsv -preset fast \
                -y "$test_output_hw" 2>/dev/null
            ;;
        "vaapi")
            ffmpeg -hwaccel vaapi -hwaccel_device /dev/dri/renderD128 \
                -i "$test_input" -vf 'format=nv12,hwupload' -c:v h264_vaapi \
                -y "$test_output_hw" 2>/dev/null
            ;;
        *)
            log::warn "Unsupported hardware acceleration method: $hw_method"
            return 1
            ;;
    esac
    
    local hw_end=$(date +%s.%N)
    local hw_duration=$(echo "$hw_end - $hw_start" | bc -l)
    local hw_success=$?
    
    # Test software encoding
    log::info "Testing software encoding..."
    local sw_start=$(date +%s.%N)
    
    ffmpeg -i "$test_input" -c:v libx264 -preset fast \
        -y "$test_output_sw" 2>/dev/null
    
    local sw_end=$(date +%s.%N)
    local sw_duration=$(echo "$sw_end - $sw_start" | bc -l)
    local sw_success=$?
    
    # Report results
    log::header "🏁 Benchmark Results"
    
    if [[ $hw_success -eq 0 ]]; then
        local hw_size=$(stat -c%s "$test_output_hw" 2>/dev/null || stat -f%z "$test_output_hw")
        printf "Hardware (%s): %.2fs (%.2fx)\n" "$hw_method" "$hw_duration" "$(echo "scale=2; 30 / $hw_duration" | bc -l)"
        echo "  Output size: $(numfmt --to=iec "$hw_size")"
    else
        log::error "Hardware encoding failed"
    fi
    
    if [[ $sw_success -eq 0 ]]; then
        local sw_size=$(stat -c%s "$test_output_sw" 2>/dev/null || stat -f%z "$test_output_sw")
        printf "Software: %.2fs (%.2fx)\n" "$sw_duration" "$(echo "scale=2; 30 / $sw_duration" | bc -l)"
        echo "  Output size: $(numfmt --to=iec "$sw_size")"
        
        if [[ $hw_success -eq 0 ]]; then
            local speedup=$(echo "scale=2; $sw_duration / $hw_duration" | bc -l)
            log::success "Hardware acceleration speedup: ${speedup}x"
        fi
    else
        log::error "Software encoding failed"
    fi
}

#######################################
# Configure optimal hardware settings
#######################################
ffmpeg::hardware::configure() {
    local force_hw="${1:-}"
    
    if [[ -n "$force_hw" ]]; then
        export FFMPEG_HW_ACCEL="$force_hw"
        log::info "Hardware acceleration forced to: $force_hw"
        return 0
    fi
    
    # Auto-detect and configure
    ffmpeg::hardware::detect
    
    local recommended="$FFMPEG_RECOMMENDED_HW_ACCEL"
    
    if [[ "$recommended" != "none" ]]; then
        export FFMPEG_HW_ACCEL="$recommended"
        log::success "Configured hardware acceleration: $recommended"
        
        # Set hardware-specific defaults
        case "$recommended" in
            "nvenc")
                export FFMPEG_HW_ENCODER="h264_nvenc"
                export FFMPEG_HW_PRESET="fast"
                ;;
            "qsv")
                export FFMPEG_HW_ENCODER="h264_qsv"
                export FFMPEG_HW_PRESET="fast"
                ;;
            "vaapi")
                export FFMPEG_HW_ENCODER="h264_vaapi"
                export FFMPEG_HW_DEVICE="/dev/dri/renderD128"
                ;;
        esac
    else
        export FFMPEG_HW_ACCEL="none"
        log::info "No hardware acceleration available - using software encoding"
    fi
}

#######################################
# Get hardware acceleration info
#######################################
ffmpeg::hardware::info() {
    log::header "🖥️  Hardware Acceleration Information"
    
    ffmpeg::hardware::detect
    
    echo "System Hardware:"
    
    # CPU info
    local cpu_model=$(grep "model name" /proc/cpuinfo | head -1 | cut -d: -f2 | xargs)
    local cpu_cores=$(nproc)
    echo "  CPU: $cpu_model ($cpu_cores cores)"
    
    # Memory info
    local mem_total=$(free -h | grep '^Mem:' | awk '{print $2}')
    echo "  RAM: $mem_total"
    
    # GPU info
    if command -v nvidia-smi &>/dev/null; then
        local gpu_info=$(nvidia-smi --query-gpu=name,memory.total --format=csv,noheader,nounits 2>/dev/null | head -1)
        if [[ -n "$gpu_info" ]]; then
            local gpu_name=$(echo "$gpu_info" | cut -d, -f1 | xargs)
            local gpu_mem=$(echo "$gpu_info" | cut -d, -f2 | xargs)
            echo "  GPU: $gpu_name (${gpu_mem}MB VRAM)"
        fi
    fi
    
    echo
    echo "Hardware Acceleration:"
    echo "  Available: ${FFMPEG_DETECTED_HW_ACCEL:-none}"
    echo "  Recommended: ${FFMPEG_RECOMMENDED_HW_ACCEL:-none}"
    echo "  Current: ${FFMPEG_HW_ACCEL:-none}"
    
    if command -v ffmpeg &>/dev/null; then
        echo
        echo "FFmpeg Hardware Support:"
        ffmpeg -hwaccels 2>/dev/null | tail -n +2 | sed 's/^/  /'
    fi
}

#######################################
# Generate hardware-optimized FFmpeg command
#######################################
ffmpeg::hardware::build_command() {
    local input_file="$1"
    local output_file="$2"
    local hw_accel="${3:-${FFMPEG_HW_ACCEL:-none}}"
    
    local cmd="ffmpeg -i \"$input_file\""
    
    case "$hw_accel" in
        "nvenc")
            cmd="$cmd -hwaccel cuda -c:v h264_nvenc"
            if [[ -n "${FFMPEG_HW_PRESET:-}" ]]; then
                cmd="$cmd -preset ${FFMPEG_HW_PRESET}"
            fi
            ;;
        "qsv")
            cmd="$cmd -hwaccel qsv -c:v h264_qsv"
            if [[ -n "${FFMPEG_HW_PRESET:-}" ]]; then
                cmd="$cmd -preset ${FFMPEG_HW_PRESET}"
            fi
            ;;
        "vaapi")
            local device="${FFMPEG_HW_DEVICE:-/dev/dri/renderD128}"
            cmd="$cmd -hwaccel vaapi -hwaccel_device $device"
            cmd="$cmd -vf 'format=nv12,hwupload' -c:v h264_vaapi"
            ;;
        "none"|*)
            cmd="$cmd -c:v ${FFMPEG_DEFAULT_VIDEO_CODEC:-libx264}"
            cmd="$cmd -preset ${FFMPEG_DEFAULT_PRESET:-medium}"
            cmd="$cmd -crf ${FFMPEG_DEFAULT_QUALITY:-23}"
            ;;
    esac
    
    cmd="$cmd -c:a ${FFMPEG_DEFAULT_AUDIO_CODEC:-aac} \"$output_file\" -y"
    
    echo "$cmd"
}