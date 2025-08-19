#!/bin/bash

ffmpeg_status() {
    local format="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            json)
                format="json"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Get the directory of this lib file
    local FFMPEG_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/format.sh" || {
        echo "Error: Could not source format.sh"
        return 1
    }
    
    local installed="false"
    local version="unknown"
    local capabilities=""
    local running="true"  # FFmpeg is a CLI tool, always "running" if installed
    
    if command -v ffmpeg &> /dev/null; then
        installed="true"
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
    else
        running="false"
    fi
    
    # Use format::output for automatic JSON support with kv data type
    format::output "$format" "kv" \
        "installed" "$installed" \
        "running" "$running" \
        "version" "$version" \
        "capabilities" "$capabilities" \
        "description" "Universal media processing framework" \
        "category" "execution"
}
