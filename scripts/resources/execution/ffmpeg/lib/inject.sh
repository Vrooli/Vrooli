#!/bin/bash

ffmpeg_inject() {
    local file_path="${1:-}"
    local action="${2:-process}"
    
    # Get the directory of this lib file
    local FFMPEG_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/log.sh" || {
        echo "Error: Could not source log.sh"
        return 1
    }
    
    if [[ -z "$file_path" ]]; then
        log::header "📥 FFmpeg Injection"
        log::info "Usage: inject <file> [action]"
        log::info ""
        log::info "Actions:"
        log::info "  process   - Process media file with default settings"
        log::info "  transcode - Transcode to different format"
        log::info "  extract   - Extract audio/video/frames"
        log::info "  info      - Get media information"
        log::info ""
        log::info "Examples:"
        log::info "  inject video.mp4 info"
        log::info "  inject input.avi transcode"
        log::info "  inject video.mp4 extract"
        return 1
    fi
    
    if [ ! -f "$file_path" ]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    case "$action" in
        info)
            log::header "📊 Media Information"
            local info=$(ffmpeg -i "$file_path" 2>&1 | grep -E "Duration:|Stream #")
            if [[ -n "$info" ]]; then
                echo "$info"
                log::success "Media info retrieved successfully"
            else
                log::error "Failed to get media info"
                return 1
            fi
            ;;
        transcode)
            local output="${file_path%.*}_transcoded.mp4"
            log::info "Transcoding to: $output"
            ffmpeg -i "$file_path" -c:v libx264 -crf 23 -c:a aac -b:a 128k "$output" -y &>/dev/null || {
                log::error "Transcoding failed"
                return 1
            }
            log::success "Transcoded to: $output"
            ;;
        extract)
            local audio_out="${file_path%.*}_audio.mp3"
            log::info "Extracting audio to: $audio_out"
            ffmpeg -i "$file_path" -vn -ar 44100 -ac 2 -b:a 192k "$audio_out" -y &>/dev/null || {
                log::error "Audio extraction failed"
                return 1
            }
            log::success "Audio extracted to: $audio_out"
            ;;
        process|*)
            local output="${file_path%.*}_processed.mp4"
            log::info "Processing to: $output"
            ffmpeg -i "$file_path" -c:v libx264 -preset fast -crf 28 "$output" -y &>/dev/null || {
                log::error "Processing failed"
                return 1
            }
            log::success "Processed to: $output"
            ;;
    esac
    
    return 0
}