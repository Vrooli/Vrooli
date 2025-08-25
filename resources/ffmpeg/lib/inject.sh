#!/bin/bash

# Source configuration and utilities
ffmpeg::inject::init() {
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
    local FFMPEG_LIB_DIR="${APP_ROOT}/resources/ffmpeg/lib"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/log.sh" || {
        echo "Error: Could not source log.sh"
        return 1
    }
    source "${FFMPEG_LIB_DIR}/../config/defaults.sh" || {
        echo "Error: Could not source defaults.sh"
        return 1
    }
    ffmpeg::export_config
}

# Batch processing function
ffmpeg::inject::batch_process() {
    local directory="${1:-}"
    local action="${2:-process}"
    local pattern="${3:-*}"
    local max_parallel="${4:-4}"
    
    if [[ -z "$directory" ]]; then
        log::error "No directory specified for batch processing"
        return 1
    fi
    
    if [[ ! -d "$directory" ]]; then
        log::error "Directory not found: $directory"
        return 1
    fi
    
    # Create output and temp directories
    mkdir -p "${FFMPEG_OUTPUT_DIR}" "${FFMPEG_TEMP_DIR}"
    
    log::header "🎬 FFmpeg Batch Processing"
    log::info "Directory: $directory"
    log::info "Action: $action"
    log::info "Pattern: $pattern"
    log::info "Max parallel: $max_parallel"
    echo
    
    # Find all media files matching pattern
    local files=()
    while IFS= read -r -d '' file; do
        files+=("$file")
    done < <(find "$directory" -maxdepth 1 -name "$pattern" -type f -print0 2>/dev/null)
    
    if [[ ${#files[@]} -eq 0 ]]; then
        log::warn "No files found matching pattern: $pattern"
        return 0
    fi
    
    log::info "Found ${#files[@]} files to process"
    
    # Track processing results
    local processed=0
    local failed=0
    local skipped=0
    
    # Process files with parallel limit
    local current_jobs=0
    local pids=()
    
    for file in "${files[@]}"; do
        # Wait if we've reached max parallel jobs
        while [[ $current_jobs -ge $max_parallel ]]; do
            for i in "${!pids[@]}"; do
                if ! kill -0 "${pids[$i]}" 2>/dev/null; then
                    wait "${pids[$i]}"
                    local exit_code=$?
                    if [[ $exit_code -eq 0 ]]; then
                        ((processed++))
                    else
                        ((failed++))
                    fi
                    unset pids[$i]
                    ((current_jobs--))
                    break
                fi
            done
            sleep 0.1
        done
        
        # Start processing in background
        (ffmpeg::inject::process_single "$file" "$action") &
        pids+=($!)
        ((current_jobs++))
        
        log::info "Started processing: $(basename "$file")"
    done
    
    # Wait for all remaining jobs
    for pid in "${pids[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            wait "$pid"
            local exit_code=$?
            if [[ $exit_code -eq 0 ]]; then
                ((processed++))
            else
                ((failed++))
            fi
        fi
    done
    
    echo
    log::header "📊 Batch Processing Results"
    log::success "Processed: $processed"
    [[ $failed -gt 0 ]] && log::error "Failed: $failed" || log::info "Failed: $failed"
    [[ $skipped -gt 0 ]] && log::warn "Skipped: $skipped" || log::info "Skipped: $skipped"
    log::info "Total: ${#files[@]}"
}

# Process a single file (used by batch processing)
ffmpeg::inject::process_single() {
    local file_path="$1"
    local action="$2"
    
    local basename_file=$(basename "$file_path")
    local filename="${basename_file%.*}"
    local extension="${basename_file##*.}"
    
    case "$action" in
        info)
            ffmpeg -i "$file_path" 2>&1 | grep -E "Duration:|Stream #" > "${FFMPEG_OUTPUT_DIR}/${filename}_info.txt"
            ;;
        transcode)
            local output="${FFMPEG_OUTPUT_DIR}/${filename}_transcoded.mp4"
            ffmpeg -i "$file_path" -c:v "${FFMPEG_DEFAULT_VIDEO_CODEC}" -crf "${FFMPEG_DEFAULT_QUALITY}" \
                -c:a "${FFMPEG_DEFAULT_AUDIO_CODEC}" -b:a 128k "$output" -y &>/dev/null
            ;;
        extract)
            local audio_out="${FFMPEG_OUTPUT_DIR}/${filename}_audio.mp3"
            ffmpeg -i "$file_path" -vn -ar 44100 -ac 2 -b:a 192k "$audio_out" -y &>/dev/null
            ;;
        thumbnail)
            local thumb_out="${FFMPEG_OUTPUT_DIR}/${filename}_thumbnail.jpg"
            ffmpeg -i "$file_path" -ss 00:00:00.5 -vframes 1 -q:v 2 "$thumb_out" -y &>/dev/null
            ;;
        process|*)
            local output="${FFMPEG_OUTPUT_DIR}/${filename}_processed.mp4"
            ffmpeg -i "$file_path" -c:v "${FFMPEG_DEFAULT_VIDEO_CODEC}" -preset "${FFMPEG_DEFAULT_PRESET}" \
                -crf "${FFMPEG_DEFAULT_QUALITY}" "$output" -y &>/dev/null
            ;;
    esac
}

# Enhanced media information extraction
ffmpeg::inject::get_detailed_info() {
    local file_path="$1"
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    log::header "📊 Detailed Media Information"
    echo "File: $file_path"
    echo
    
    # Get comprehensive info using ffprobe if available
    if command -v ffprobe &>/dev/null; then
        log::info "🎯 Technical Details:"
        ffprobe -v quiet -print_format json -show_format -show_streams "$file_path" | jq . 2>/dev/null || {
            log::warn "jq not available, using basic ffprobe output"
            ffprobe -v quiet -show_format -show_streams "$file_path"
        }
    else
        log::info "🎯 Basic Information:"
        ffmpeg -i "$file_path" 2>&1 | grep -E "Duration:|Stream #|Metadata:"
    fi
    
    echo
    log::info "📁 File Details:"
    local file_size=$(stat -c%s "$file_path" 2>/dev/null || stat -f%z "$file_path" 2>/dev/null)
    echo "  Size: $(numfmt --to=iec "$file_size")"
    echo "  Modified: $(stat -c%y "$file_path" 2>/dev/null || stat -f%Sm "$file_path" 2>/dev/null)"
}

# Advanced transcoding with progress monitoring
ffmpeg::inject::advanced_transcode() {
    local input_file="$1"
    local output_file="$2"
    local codec="${3:-${FFMPEG_DEFAULT_VIDEO_CODEC}}"
    local quality="${4:-${FFMPEG_DEFAULT_QUALITY}}"
    local preset="${5:-${FFMPEG_DEFAULT_PRESET}}"
    
    if [[ ! -f "$input_file" ]]; then
        log::error "Input file not found: $input_file"
        return 1
    fi
    
    # Get duration for progress calculation
    local duration_line=$(ffmpeg -i "$input_file" 2>&1 | grep "Duration:")
    local duration=$(echo "$duration_line" | sed -n 's/.*Duration: \([^,]*\).*/\1/p')
    
    log::info "Transcoding with progress monitoring:"
    log::info "  Input: $input_file"
    log::info "  Output: $output_file" 
    log::info "  Codec: $codec"
    log::info "  Quality: $quality"
    log::info "  Preset: $preset"
    log::info "  Duration: $duration"
    echo
    
    # Create progress file
    local progress_file="${FFMPEG_TEMP_DIR}/progress_$(basename "$output_file").log"
    
    # Run ffmpeg with progress reporting
    ffmpeg -i "$input_file" \
        -c:v "$codec" -crf "$quality" -preset "$preset" \
        -c:a "${FFMPEG_DEFAULT_AUDIO_CODEC}" \
        -progress "$progress_file" \
        -y "$output_file" 2>/dev/null &
    
    local ffmpeg_pid=$!
    
    # Monitor progress
    while kill -0 $ffmpeg_pid 2>/dev/null; do
        if [[ -f "$progress_file" ]]; then
            local current_time=$(tail -n 20 "$progress_file" | grep "out_time=" | tail -n 1 | cut -d= -f2)
            if [[ -n "$current_time" && "$current_time" != "N/A" ]]; then
                printf "\rProgress: %s / %s" "$current_time" "$duration"
            fi
        fi
        sleep 1
    done
    
    wait $ffmpeg_pid
    local exit_code=$?
    
    # Clean up progress file
    [[ -f "$progress_file" ]] && rm -f "$progress_file"
    
    echo  # New line after progress
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "Transcoding completed: $output_file"
        local input_size=$(stat -c%s "$input_file" 2>/dev/null || stat -f%z "$input_file")
        local output_size=$(stat -c%s "$output_file" 2>/dev/null || stat -f%z "$output_file")
        local compression_ratio=$(echo "scale=1; $input_size * 100 / $output_size" | bc -l 2>/dev/null || echo "N/A")
        
        log::info "  Input size: $(numfmt --to=iec "$input_size")"
        log::info "  Output size: $(numfmt --to=iec "$output_size")"
        log::info "  Compression ratio: ${compression_ratio}%"
    else
        log::error "Transcoding failed"
        return 1
    fi
}

ffmpeg_inject() {
    local file_path="${1:-}"
    local action="${2:-process}"
    
    # Initialize
    ffmpeg::inject::init || return 1
    
    if [[ -z "$file_path" ]]; then
        log::header "📥 FFmpeg Injection"
        log::info "Usage: inject <file|directory> [action]"
        log::info ""
        log::info "Actions:"
        log::info "  process     - Process media file with default settings"
        log::info "  transcode   - Transcode to different format"
        log::info "  extract     - Extract audio from video"
        log::info "  thumbnail   - Generate thumbnail image"
        log::info "  info        - Get basic media information"
        log::info "  detailed    - Get detailed media information"
        log::info "  batch       - Process all files in directory"
        log::info ""
        log::info "Examples:"
        log::info "  inject video.mp4 info"
        log::info "  inject input.avi transcode"
        log::info "  inject video.mp4 extract"
        log::info "  inject /path/to/videos batch process"
        log::info "  inject /path/to/videos batch extract *.mp4"
        return 0
    fi
    
    # Handle batch processing for directories
    if [[ -d "$file_path" ]]; then
        local batch_action="${3:-process}"
        local pattern="${4:-*}"
        local max_parallel="${5:-4}"
        ffmpeg::inject::batch_process "$file_path" "$batch_action" "$pattern" "$max_parallel"
        return $?
    fi
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    # Create output directories
    mkdir -p "${FFMPEG_OUTPUT_DIR}" "${FFMPEG_TEMP_DIR}"
    
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
        detailed)
            ffmpeg::inject::get_detailed_info "$file_path"
            ;;
        transcode)
            local output="${file_path%.*}_transcoded.mp4"
            if [[ -n "${OUTPUT_PATH:-}" ]]; then
                output="$OUTPUT_PATH"
            fi
            
            if command -v bc &>/dev/null; then
                # Use advanced transcoding with progress if bc is available
                ffmpeg::inject::advanced_transcode "$file_path" "$output" \
                    "${CODEC:-${FFMPEG_DEFAULT_VIDEO_CODEC}}" \
                    "${QUALITY:-${FFMPEG_DEFAULT_QUALITY}}" \
                    "${PRESET:-${FFMPEG_DEFAULT_PRESET}}"
            else
                # Fallback to simple transcoding
                log::info "Transcoding to: $output"
                ffmpeg -i "$file_path" -c:v "${FFMPEG_DEFAULT_VIDEO_CODEC}" -crf "${FFMPEG_DEFAULT_QUALITY}" \
                    -c:a "${FFMPEG_DEFAULT_AUDIO_CODEC}" -b:a 128k "$output" -y &>/dev/null || {
                    log::error "Transcoding failed"
                    return 1
                }
                log::success "Transcoded to: $output"
            fi
            ;;
        extract)
            local audio_out="${file_path%.*}_audio.mp3"
            if [[ -n "${OUTPUT_PATH:-}" ]]; then
                audio_out="$OUTPUT_PATH"
            fi
            
            log::info "Extracting audio to: $audio_out"
            ffmpeg -i "$file_path" -vn -ar 44100 -ac 2 -b:a 192k "$audio_out" -y &>/dev/null || {
                log::error "Audio extraction failed"
                return 1
            }
            log::success "Audio extracted to: $audio_out"
            ;;
        thumbnail)
            local basename_file=$(basename "$file_path")
            local filename="${basename_file%.*}"
            local thumb_out="${FFMPEG_OUTPUT_DIR}/${filename}_thumbnail.jpg"
            if [[ -n "${OUTPUT_PATH:-}" ]]; then
                thumb_out="$OUTPUT_PATH"
            fi
            
            log::info "Generating thumbnail: $thumb_out"
            ffmpeg -i "$file_path" -ss 00:00:00.5 -vframes 1 -q:v 2 "$thumb_out" -y &>/dev/null || {
                log::error "Thumbnail generation failed"
                return 1
            }
            log::success "Thumbnail generated: $thumb_out"
            ;;
        process|*)
            local output="${file_path%.*}_processed.mp4"
            if [[ -n "${OUTPUT_PATH:-}" ]]; then
                output="$OUTPUT_PATH"
            fi
            
            log::info "Processing to: $output"
            
            local cmd="ffmpeg -i \"$file_path\""
            
            # Add hardware acceleration if configured
            if [[ -n "${HW_ACCEL:-}" && "${HW_ACCEL}" != "none" ]]; then
                cmd="$cmd -hwaccel ${HW_ACCEL}"
            fi
            
            cmd="$cmd -c:v \"${CODEC:-${FFMPEG_DEFAULT_VIDEO_CODEC}}\" -preset \"${PRESET:-${FFMPEG_DEFAULT_PRESET}}\" -crf \"${QUALITY:-${FFMPEG_DEFAULT_QUALITY}}\" \"$output\" -y"
            
            eval "$cmd" &>/dev/null || {
                log::error "Processing failed"
                return 1
            }
            log::success "Processed to: $output"
            ;;
    esac
    
    return 0
}