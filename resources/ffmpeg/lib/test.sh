#!/bin/bash
# FFmpeg Test Functions - v2.0 Universal Contract Compliant

# Initialize test environment
ffmpeg::test::init() {
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
    source "${APP_ROOT}/scripts/lib/utils/log.sh"
    source "${APP_ROOT}/resources/ffmpeg/config/defaults.sh"
    ffmpeg::export_config
}

# Smoke test - quick health check (required)
ffmpeg::test::smoke() {
    ffmpeg::test::init
    
    log::header "🧪 FFmpeg Smoke Test"
    
    # Test if ffmpeg is installed
    if ! command -v ffmpeg &> /dev/null; then
        log::error "FFmpeg is not installed"
        return 1
    fi
    
    # Test basic ffmpeg functionality
    if ! ffmpeg -version &> /dev/null; then
        log::error "FFmpeg is installed but not working properly"
        return 1
    fi
    
    log::success "FFmpeg smoke test passed"
    return 0
}

# Integration test - test ffmpeg functionality (required)
ffmpeg::test::integration() {
    ffmpeg::test::init
    
    log::header "🧪 FFmpeg Integration Test"
    
    # Ensure directories exist
    mkdir -p "${FFMPEG_DATA_DIR}" "${FFMPEG_OUTPUT_DIR}" "${FFMPEG_TEMP_DIR}"
    
    # Create a test input file (1 second of silence)
    local test_input="${FFMPEG_TEMP_DIR}/test_input.wav"
    local test_output="${FFMPEG_TEMP_DIR}/test_output.mp3"
    
    log::info "Creating test audio file..."
    if ! ffmpeg -f lavfi -i "sine=frequency=440:duration=1" -ac 1 -ar 22050 "$test_input" -y &>/dev/null; then
        log::error "Failed to create test input file"
        return 1
    fi
    
    log::info "Testing basic transcoding..."
    if ! ffmpeg -i "$test_input" -codec:a libmp3lame -b:a 64k "$test_output" -y &>/dev/null; then
        log::error "Failed to transcode test file"
        rm -f "$test_input"
        return 1
    fi
    
    # Verify output file exists and has content
    if [[ ! -f "$test_output" ]] || [[ ! -s "$test_output" ]]; then
        log::error "Output file is missing or empty"
        rm -f "$test_input" "$test_output"
        return 1
    fi
    
    # Clean up test files
    rm -f "$test_input" "$test_output"
    
    log::success "FFmpeg integration test passed"
    return 0
}

# Unit test - test library functions (optional)
ffmpeg::test::unit() {
    ffmpeg::test::init
    
    log::header "🧪 FFmpeg Unit Test"
    
    # Test configuration export
    if ! ffmpeg::export_config; then
        log::error "Configuration export failed"
        return 1
    fi
    
    # Test version detection
    local version=$(ffmpeg::get_version)
    if [[ "$version" == "not_installed" ]]; then
        log::error "Version detection failed"
        return 1
    fi
    
    log::info "Version detected: $version"
    
    # Test installation check
    if ! ffmpeg::test_installation; then
        log::error "Installation test failed"
        return 1
    fi
    
    log::success "FFmpeg unit test passed"
    return 0
}

# Screen capture smoke test — verify functions defined and config set
ffmpeg::test::screen_smoke() {
    ffmpeg::test::init

    log::header "🧪 FFmpeg Screen Capture Smoke Test"

    # Verify functions are defined
    local funcs=(
        "ffmpeg::screen::start"
        "ffmpeg::screen::stop"
        "ffmpeg::screen::list_displays"
        "ffmpeg::screen::status"
        "ffmpeg::screen::info"
    )
    for fn in "${funcs[@]}"; do
        if ! declare -F "$fn" &>/dev/null; then
            log::error "Function not defined: $fn"
            return 1
        fi
    done

    # Verify config variables
    if [[ -z "${FFMPEG_SCREEN_CAPTURES_DIR:-}" ]]; then
        log::error "FFMPEG_SCREEN_CAPTURES_DIR not set"
        return 1
    fi
    if [[ -z "${FFMPEG_SCREEN_DEFAULT_FRAMERATE:-}" ]]; then
        log::error "FFMPEG_SCREEN_DEFAULT_FRAMERATE not set"
        return 1
    fi

    # On non-Linux, verify stubs return 1
    local platform
    platform=$(system::detect_platform)
    if [[ "$platform" != "linux" ]]; then
        if ffmpeg::screen::start 2>/dev/null; then
            log::error "screen::start should return 1 on non-Linux"
            return 1
        fi
        log::info "Non-Linux platform: stubs correctly return 1"
    fi

    log::success "Screen capture smoke test passed"
    return 0
}

# Screen capture integration test — Linux only: record and verify video
ffmpeg::test::screen_integration() {
    ffmpeg::test::init

    log::header "🧪 FFmpeg Screen Capture Integration Test"

    local platform
    platform=$(system::detect_platform)
    if [[ "$platform" != "linux" ]]; then
        log::info "Skipping screen capture integration test on ${platform}"
        return 0
    fi

    # Check for Xvfb
    if ! command -v Xvfb &>/dev/null; then
        log::warn "Xvfb not installed, skipping screen capture integration test"
        return 0
    fi

    # Check for x11grab
    if ! ffmpeg -devices 2>/dev/null | grep -q x11grab; then
        log::warn "FFmpeg x11grab not available, skipping"
        return 0
    fi

    local test_display=":99"
    local test_output="${FFMPEG_TEMP_DIR}/screen_test_${$}.mp4"
    mkdir -p "${FFMPEG_TEMP_DIR}"

    # Start recording at low resolution/fps to keep test fast
    local rec_id
    rec_id=$(ffmpeg::screen::start \
        --display "$test_display" \
        --resolution "640x480" \
        --framerate 10 \
        --output "$test_output") || {
        log::error "Failed to start screen capture"
        return 1
    }

    log::info "Recording started: ${rec_id}"
    sleep 2

    # Stop recording
    ffmpeg::screen::stop "$rec_id" || {
        log::error "Failed to stop screen capture"
        rm -f "$test_output"
        return 1
    }

    # Verify output file
    if [[ ! -f "$test_output" ]] || [[ ! -s "$test_output" ]]; then
        log::error "Output file missing or empty: ${test_output}"
        rm -f "$test_output"
        return 1
    fi

    # Verify it's a valid video with ffprobe
    if command -v ffprobe &>/dev/null; then
        if ! ffprobe -v error -select_streams v:0 \
            -show_entries stream=codec_type \
            -of default=noprint_wrappers=1:nokey=1 \
            "$test_output" 2>/dev/null | grep -q "video"; then
            log::error "Output file is not a valid video"
            rm -f "$test_output"
            return 1
        fi
        log::info "ffprobe confirms valid video stream"
    fi

    rm -f "$test_output"
    log::success "Screen capture integration test passed"
    return 0
}

# Run all tests (required)
ffmpeg::test::all() {
    ffmpeg::test::init
    
    log::header "🧪 FFmpeg All Tests"
    
    local failed=0
    
    # Run smoke test
    if ! ffmpeg::test::smoke; then
        ((failed++))
    fi
    
    # Run integration test
    if ! ffmpeg::test::integration; then
        ((failed++))
    fi
    
    # Run unit test
    if ! ffmpeg::test::unit; then
        ((failed++))
    fi

    # Run screen capture tests
    if ! ffmpeg::test::screen_smoke; then
        ((failed++))
    fi
    if ! ffmpeg::test::screen_integration; then
        ((failed++))
    fi

    if [[ $failed -eq 0 ]]; then
        log::success "All FFmpeg tests passed"
        return 0
    else
        log::error "$failed test(s) failed"
        return 1
    fi
}