#!/bin/bash

# Get the directory of this lib file
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
FFMPEG_STATUS_DIR="${APP_ROOT}/resources/ffmpeg/lib"

# Source status-args library
source "${FFMPEG_STATUS_DIR}/../../../lib/status-args.sh"

# Collect FFmpeg status data in format-agnostic structure
ffmpeg::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local installed="false"
    local version="unknown"
    local capabilities=""
    local running="true"  # FFmpeg is a CLI tool, always "running" if installed
    
    if command -v ffmpeg &> /dev/null; then
        installed="true"
        
        # Get version (skip in fast mode)
        if [[ "$fast_mode" == "true" ]]; then
            version="N/A"
            capabilities="N/A"
        else
            version=$(ffmpeg -version 2>&1 | head -n1 | cut -d' ' -f3)
            
            # Check for key capabilities
            local caps=()
            local codecs_output=$(ffmpeg -hide_banner -codecs 2>&1)
            echo "$codecs_output" | grep -q "libx264" && caps+=("h264")
            echo "$codecs_output" | grep -q "libx265" && caps+=("h265")
            echo "$codecs_output" | grep -q "libvpx" && caps+=("vp8/vp9")
            echo "$codecs_output" | grep -q "mp3" && caps+=("mp3")
            echo "$codecs_output" | grep -q "aac" && caps+=("aac")
            capabilities=$(IFS=,; echo "${caps[*]}")
        fi
    else
        running="false"
    fi
    
    # Determine health status
    local health="false"
    local health_message="FFmpeg not installed"
    if [[ "$installed" == "true" ]]; then
        health="true"
        health_message="FFmpeg installed and ready"
    fi
    
    # Check test results
    local test_status="not_run"
    local test_timestamp=""
    local test_results=""
    local test_result_file="${FFMPEG_STATUS_DIR}/../test/.test_results"
    
    if [[ -f "$test_result_file" ]] && [[ "$fast_mode" != "true" ]]; then
        test_timestamp=$(stat -c %Y "$test_result_file" 2>/dev/null || stat -f %m "$test_result_file" 2>/dev/null || echo "")
        if [[ -n "$test_timestamp" ]]; then
            test_timestamp=$(date -d "@$test_timestamp" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r "$test_timestamp" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
        fi
        test_results=$(cat "$test_result_file" 2>/dev/null || echo "")
        if [[ -n "$test_results" ]]; then
            test_status="passed"
            # Check if any test failed
            if echo "$test_results" | grep -q "not ok"; then
                test_status="failed"
            fi
        fi
    fi
    
    # Build status data array
    local status_data=(
        "name" "ffmpeg"
        "category" "execution"
        "description" "Universal media processing framework"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$health"
        "health_message" "$health_message"
        "version" "$version"
        "capabilities" "$capabilities"
        "test_status" "$test_status"
        "test_timestamp" "$test_timestamp"
    )
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
ffmpeg::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Source format utilities for proper logging functions
    source "${FFMPEG_STATUS_DIR}/../../../../lib/utils/format.sh"
    
    # Header
    log::header "🎬 FFmpeg Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
        log::success "   ✅ Running: Yes (CLI tool)"
        log::success "   ✅ Health: ${data[health_message]:-Healthy}"
    else
        log::error "   ❌ Installed: No"
        log::error "   ❌ Running: No"
        log::warn "   ⚠️  Health: ${data[health_message]:-Not installed}"
        echo
        log::info "💡 Installation Required:"
        log::info "   Install FFmpeg using your system package manager"
        log::info "   Ubuntu/Debian: sudo apt install ffmpeg"
        log::info "   macOS: brew install ffmpeg"
        return
    fi
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📦 Version: ${data[version]:-unknown}"
    log::info "   🔧 Capabilities: ${data[capabilities]:-unknown}"
    echo
    
    # Test Results
    if [[ -n "${data[test_timestamp]}" ]]; then
        log::info "🧪 Test Results:"
        case "${data[test_status]:-not_run}" in
            "passed")
                log::success "   ✅ Status: All tests passed"
                ;;
            "failed")
                log::error "   ❌ Status: Some tests failed"
                ;;
            *)
                log::warn "   ⚠️  Status: Tests not run"
                ;;
        esac
        if [[ -n "${data[test_timestamp]}" ]] && [[ "${data[test_timestamp]}" != "" ]]; then
            log::info "   📅 Last Run: ${data[test_timestamp]}"
        fi
        echo
    fi
}

# Main status function using standard wrapper
ffmpeg::status() {
    status::run_standard "ffmpeg" "ffmpeg::status::collect_data" "ffmpeg::status::display_text" "$@"
}

# Legacy function name for backward compatibility
ffmpeg_status() {
    ffmpeg::status "$@"
}
