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
    
    if [[ $failed -eq 0 ]]; then
        log::success "All FFmpeg tests passed"
        return 0
    else
        log::error "$failed test(s) failed"
        return 1
    fi
}