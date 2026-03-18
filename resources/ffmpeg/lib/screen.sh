#!/bin/bash
# Screen Capture Functions for FFmpeg Resource
# Provides desktop/window screen recording via x11grab on Linux.
# macOS and Windows fail gracefully with clear messages.

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/system/system_commands.sh"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# Generate a unique recording ID using timestamp + PID + random bytes.
# The random suffix prevents collisions when the same shell PID starts
# multiple recordings (e.g. in a loop or concurrent subshells).
_screen_generate_id() {
    local rand_hex
    if [[ -r /dev/urandom ]]; then
        rand_hex=$(head -c 4 /dev/urandom | od -An -tx1 | tr -d ' \n')
    else
        rand_hex=$(printf '%04x' "$RANDOM")
    fi
    printf "rec-%s-%s-%s" "$(date +%Y%m%d%H%M%S)" "$$" "$rand_hex"
}

# Path to metadata JSON for a recording
_screen_meta_path() {
    local id="$1"
    echo "${FFMPEG_SCREEN_CAPTURES_DIR}/metadata/${id}.json"
}

# Write metadata JSON
_screen_write_meta() {
    local id="$1" ffmpeg_pid="$2" xvfb_pid="$3" display="$4" output="$5"
    local meta_dir="${FFMPEG_SCREEN_CAPTURES_DIR}/metadata"
    mkdir -p "$meta_dir"
    cat > "$(_screen_meta_path "$id")" <<METAEOF
{
  "recording_id": "${id}",
  "ffmpeg_pid": ${ffmpeg_pid},
  "xvfb_pid": ${xvfb_pid},
  "display": "${display}",
  "output_path": "${output}",
  "start_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "status": "recording"
}
METAEOF
}

# Read a field from metadata JSON using jq (common dependency).
_screen_read_meta_field() {
    local id="$1" field="$2"
    local file
    file="$(_screen_meta_path "$id")"
    [[ -f "$file" ]] || return 1
    jq -r ".${field} // empty" "$file" 2>/dev/null
}

# Update status field in metadata using jq for safe JSON manipulation.
_screen_update_status() {
    local id="$1" new_status="$2"
    local file
    file="$(_screen_meta_path "$id")"
    [[ -f "$file" ]] || return 1
    local tmp="${file}.tmp"
    jq --arg s "$new_status" '.status = $s' "$file" > "$tmp" 2>/dev/null && mv "$tmp" "$file"
}

# Check if a display is already in use
_screen_display_in_use() {
    local display="$1"
    if command -v xdpyinfo &>/dev/null; then
        xdpyinfo -display "$display" &>/dev/null && return 0
    fi
    return 1
}

# Find the most recent recording ID
_screen_most_recent_id() {
    local meta_dir="${FFMPEG_SCREEN_CAPTURES_DIR}/metadata"
    [[ -d "$meta_dir" ]] || return 1
    local latest
    latest=$(ls -t "$meta_dir"/*.json 2>/dev/null | head -1)
    [[ -n "$latest" ]] || return 1
    basename "$latest" .json
}

# ---------------------------------------------------------------------------
# ffmpeg::screen::start
# Start screen recording.
# Options:
#   --display <:N>       X display (default: :99)
#   --resolution <WxH>   Virtual display resolution (default from config)
#   --framerate <fps>    Capture framerate (default from config)
#   --output <path>      Output file path (default: auto-generated in captures dir)
#   --codec <codec>      Video codec (default from config)
#   --color-depth <N>    Color depth for Xvfb (default from config)
# Prints recording_id to stdout on success.
# ---------------------------------------------------------------------------
ffmpeg::screen::start() {
    local platform
    platform=$(system::detect_platform)
    if [[ "$platform" != "linux" ]]; then
        log::error "Screen capture requires Linux with X11. Current platform: ${platform}."
        log::info "On macOS and Windows, screen capture is not yet supported."
        return 1
    fi

    # Verify required tools exist before doing anything
    if ! command -v ffmpeg &>/dev/null; then
        log::error "FFmpeg is not installed. Run 'vrooli setup' or install ffmpeg manually."
        return 1
    fi

    if ! ffmpeg -devices 2>/dev/null | grep -q x11grab; then
        log::error "FFmpeg x11grab device not available. Your FFmpeg build may lack X11 support."
        log::info "On Ubuntu/Debian: apt-get install ffmpeg (the default build includes x11grab)"
        return 1
    fi

    if ! command -v Xvfb &>/dev/null; then
        log::error "Xvfb is not installed. Run 'vrooli setup' to install it automatically."
        log::info "Or install manually: apt-get install xvfb (Ubuntu/Debian)"
        return 1
    fi

    if ! command -v jq &>/dev/null; then
        log::error "jq is required for metadata management. Run 'vrooli setup' to install it."
        return 1
    fi

    # Parse arguments
    local display="${FFMPEG_SCREEN_DEFAULT_DISPLAY:-:99}"
    local resolution="${FFMPEG_SCREEN_DEFAULT_RESOLUTION:-1920x1080}"
    local framerate="${FFMPEG_SCREEN_DEFAULT_FRAMERATE:-30}"
    local codec="${FFMPEG_SCREEN_DEFAULT_CODEC:-libx264}"
    local color_depth="${FFMPEG_SCREEN_DEFAULT_COLOR_DEPTH:-24}"
    local output=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --display)    display="$2"; shift 2 ;;
            --resolution) resolution="$2"; shift 2 ;;
            --framerate)  framerate="$2"; shift 2 ;;
            --output)     output="$2"; shift 2 ;;
            --codec)      codec="$2"; shift 2 ;;
            --color-depth) color_depth="$2"; shift 2 ;;
            *) log::error "Unknown option: $1"; return 1 ;;
        esac
    done

    local id
    id=$(_screen_generate_id)

    mkdir -p "${FFMPEG_SCREEN_CAPTURES_DIR}"

    if [[ -z "$output" ]]; then
        output="${FFMPEG_SCREEN_CAPTURES_DIR}/${id}.mp4"
    fi

    # Start Xvfb if the target display is not already active
    local xvfb_pid=0
    if ! _screen_display_in_use "$display"; then
        log::info "Starting Xvfb on display ${display} at ${resolution}x${color_depth}..."
        Xvfb "$display" -screen 0 "${resolution}x${color_depth}" +extension GLX +render -noreset &>/dev/null &
        xvfb_pid=$!

        # Wait for Xvfb to become ready (up to 5 seconds)
        local retries=50
        while (( retries > 0 )); do
            if _screen_display_in_use "$display"; then
                break
            fi
            sleep 0.1
            (( retries-- ))
        done

        if ! _screen_display_in_use "$display"; then
            log::error "Xvfb failed to start on display ${display}"
            kill "$xvfb_pid" 2>/dev/null || true
            return 1
        fi
        log::info "Xvfb ready on display ${display} (PID ${xvfb_pid})"
    else
        log::info "Display ${display} already active, reusing"
    fi

    # Build FFmpeg capture command
    local width height
    width="${resolution%%x*}"
    height="${resolution##*x}"

    # Try hardware encoding first via detect, fall back to requested codec
    local encoder="$codec"
    if declare -F ffmpeg::hardware::detect &>/dev/null; then
        ffmpeg::hardware::detect &>/dev/null || true
        if [[ -n "${FFMPEG_HW_ENCODER:-}" ]]; then
            encoder="$FFMPEG_HW_ENCODER"
            log::info "Using hardware encoder: ${encoder}"
        fi
    fi

    log::info "Starting screen capture: ${width}x${height}@${framerate}fps → ${output}"

    DISPLAY="$display" ffmpeg \
        -f x11grab \
        -framerate "$framerate" \
        -video_size "${width}x${height}" \
        -i "$display" \
        -c:v "$encoder" \
        -pix_fmt yuv420p \
        -preset ultrafast \
        -crf 23 \
        -y "$output" &>/dev/null &
    local ffmpeg_pid=$!

    # Verify FFmpeg started
    sleep 0.5
    if ! kill -0 "$ffmpeg_pid" 2>/dev/null; then
        log::error "FFmpeg failed to start screen capture"
        [[ "$xvfb_pid" -ne 0 ]] && kill "$xvfb_pid" 2>/dev/null || true
        return 1
    fi

    _screen_write_meta "$id" "$ffmpeg_pid" "$xvfb_pid" "$display" "$output"

    log::success "Recording started (ID: ${id}, FFmpeg PID: ${ffmpeg_pid})"
    echo "$id"
    return 0
}

# ---------------------------------------------------------------------------
# ffmpeg::screen::stop
# Stop a recording by ID (or most recent).
# Usage: ffmpeg::screen::stop [recording_id]
# Prints output path and duration.
# ---------------------------------------------------------------------------
ffmpeg::screen::stop() {
    local id="${1:-}"

    if [[ -z "$id" ]]; then
        id=$(_screen_most_recent_id) || {
            log::error "No recording ID provided and no recent recordings found"
            return 1
        }
    fi

    local meta_file
    meta_file="$(_screen_meta_path "$id")"
    if [[ ! -f "$meta_file" ]]; then
        log::error "Recording not found: ${id}"
        return 1
    fi

    local ffmpeg_pid xvfb_pid output_path
    ffmpeg_pid=$(_screen_read_meta_field "$id" "ffmpeg_pid")
    xvfb_pid=$(_screen_read_meta_field "$id" "xvfb_pid")
    output_path=$(_screen_read_meta_field "$id" "output_path")

    # Stop FFmpeg gracefully with SIGINT (allows proper file finalization)
    if [[ -n "$ffmpeg_pid" ]] && kill -0 "$ffmpeg_pid" 2>/dev/null; then
        log::info "Sending SIGINT to FFmpeg (PID ${ffmpeg_pid})..."
        kill -INT "$ffmpeg_pid" 2>/dev/null || true

        # Wait up to 10 seconds for graceful exit
        local wait_count=0
        while kill -0 "$ffmpeg_pid" 2>/dev/null && (( wait_count < 100 )); do
            sleep 0.1
            (( wait_count++ ))
        done

        # Force kill if still running
        if kill -0 "$ffmpeg_pid" 2>/dev/null; then
            log::warn "FFmpeg did not stop gracefully, sending SIGTERM"
            kill -TERM "$ffmpeg_pid" 2>/dev/null || true
            sleep 1
            kill -9 "$ffmpeg_pid" 2>/dev/null || true
        fi
    fi

    # Stop Xvfb if we started it
    if [[ -n "$xvfb_pid" && "$xvfb_pid" != "0" ]]; then
        if kill -0 "$xvfb_pid" 2>/dev/null; then
            log::info "Stopping Xvfb (PID ${xvfb_pid})..."
            kill "$xvfb_pid" 2>/dev/null || true
        fi
    fi

    _screen_update_status "$id" "stopped"

    # Report results
    if [[ -f "$output_path" ]]; then
        local size duration_str
        size=$(stat -c%s "$output_path" 2>/dev/null || echo "0")
        duration_str="unknown"
        if command -v ffprobe &>/dev/null; then
            duration_str=$(ffprobe -v error -show_entries format=duration \
                -of default=noprint_wrappers=1:nokey=1 "$output_path" 2>/dev/null || echo "unknown")
        fi

        log::success "Recording stopped: ${output_path}"
        log::info "  Duration: ${duration_str}s  Size: ${size} bytes"
        echo "$output_path"
    else
        log::warn "Output file not found: ${output_path}"
        return 1
    fi

    return 0
}

# ---------------------------------------------------------------------------
# ffmpeg::screen::list_displays
# List available X displays.
# ---------------------------------------------------------------------------
ffmpeg::screen::list_displays() {
    local platform
    platform=$(system::detect_platform)
    if [[ "$platform" != "linux" ]]; then
        log::error "Display listing requires Linux with X11. Current platform: ${platform}."
        return 1
    fi

    log::header "Available Displays"

    # Check DISPLAY env var
    if [[ -n "${DISPLAY:-}" ]]; then
        echo "Current DISPLAY: ${DISPLAY}"
    fi

    # List running Xvfb instances
    if ! command -v pgrep &>/dev/null; then
        log::warn "pgrep not available, cannot list Xvfb instances"
        return 0
    fi

    local xvfb_procs
    xvfb_procs=$(pgrep -a Xvfb 2>/dev/null || true)
    if [[ -n "$xvfb_procs" ]]; then
        echo ""
        echo "Xvfb instances:"
        echo "$xvfb_procs" | while read -r line; do
            echo "  $line"
        done
    else
        echo "No Xvfb instances running"
    fi

    return 0
}

# ---------------------------------------------------------------------------
# ffmpeg::screen::status
# Show active recordings — scans metadata, verifies PIDs, marks crashed ones.
# ---------------------------------------------------------------------------
ffmpeg::screen::status() {
    local meta_dir="${FFMPEG_SCREEN_CAPTURES_DIR:-/tmp/ffmpeg-screen-captures}/metadata"

    if [[ ! -d "$meta_dir" ]] || [[ -z "$(ls -A "$meta_dir" 2>/dev/null)" ]]; then
        log::info "No recordings found"
        return 0
    fi

    log::header "Screen Recordings"
    printf "%-30s %-12s %-10s %-8s %s\n" "ID" "STATUS" "FFMPEG PID" "DISPLAY" "OUTPUT"
    printf "%-30s %-12s %-10s %-8s %s\n" "---" "------" "----------" "-------" "------"

    for meta_file in "$meta_dir"/*.json; do
        [[ -f "$meta_file" ]] || continue
        local rec_id
        rec_id=$(basename "$meta_file" .json)

        local status ffmpeg_pid display_val output_val
        status=$(_screen_read_meta_field "$rec_id" "status")
        ffmpeg_pid=$(_screen_read_meta_field "$rec_id" "ffmpeg_pid")
        display_val=$(_screen_read_meta_field "$rec_id" "display")
        output_val=$(_screen_read_meta_field "$rec_id" "output_path")

        # Verify PID is still alive for "recording" status
        if [[ "$status" == "recording" ]]; then
            if ! kill -0 "$ffmpeg_pid" 2>/dev/null; then
                status="crashed"
                _screen_update_status "$rec_id" "crashed"
            fi
        fi

        printf "%-30s %-12s %-10s %-8s %s\n" "$rec_id" "$status" "$ffmpeg_pid" "$display_val" "$output_val"
    done

    return 0
}

# ---------------------------------------------------------------------------
# ffmpeg::screen::info
# Get display information via xdpyinfo.
# Usage: ffmpeg::screen::info [display]
# ---------------------------------------------------------------------------
ffmpeg::screen::info() {
    local platform
    platform=$(system::detect_platform)
    if [[ "$platform" != "linux" ]]; then
        log::error "Display info requires Linux with X11. Current platform: ${platform}."
        return 1
    fi

    local display="${1:-${DISPLAY:-:0}}"

    if ! command -v xdpyinfo &>/dev/null; then
        log::error "xdpyinfo not available. Run 'vrooli setup' or install x11-utils manually."
        return 1
    fi

    log::header "Display Information: ${display}"
    xdpyinfo -display "$display" 2>/dev/null || {
        log::error "Cannot connect to display ${display}"
        return 1
    }

    return 0
}

# Export all functions
export -f ffmpeg::screen::start
export -f ffmpeg::screen::stop
export -f ffmpeg::screen::list_displays
export -f ffmpeg::screen::status
export -f ffmpeg::screen::info
export -f _screen_generate_id
export -f _screen_meta_path
export -f _screen_write_meta
export -f _screen_read_meta_field
export -f _screen_update_status
export -f _screen_display_in_use
export -f _screen_most_recent_id
