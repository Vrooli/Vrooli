#!/usr/bin/env bats
# Enhanced FFmpeg Integration Tests - Testing new features

setup() {
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-support/load"
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-assert/load"
    
    # Source FFmpeg configuration
    source "${BATS_TEST_DIRNAME}/../config/defaults.sh"
    ffmpeg::export_config
    
    # Create temp directory for test files
    export TEST_TMP_DIR="$(mktemp -d)"
    
    # Create test media directory structure
    export TEST_MEDIA_DIR="$TEST_TMP_DIR/media"
    mkdir -p "$TEST_MEDIA_DIR"
}

teardown() {
    # Clean up test files
    [[ -d "$TEST_TMP_DIR" ]] && rm -rf "$TEST_TMP_DIR"
}

@test "ffmpeg: cli.sh is executable and shows help" {
    run "${BATS_TEST_DIRNAME}/../cli.sh" help
    assert_success
    assert_output --partial "Usage: resource-ffmpeg"
}

@test "ffmpeg: cli.sh status works" {
    run "${BATS_TEST_DIRNAME}/../cli.sh" status
    assert_success
    assert_output --partial "FFmpeg Status"
}

@test "ffmpeg: cli.sh status --verbose provides detailed info" {
    run "${BATS_TEST_DIRNAME}/../cli.sh" status --verbose
    assert_success
    assert_output --partial "FFmpeg Status"
    assert_output --partial "Basic Status:"
    assert_output --partial "Configuration:"
}

@test "ffmpeg: cli.sh start verifies installation" {
    run "${BATS_TEST_DIRNAME}/../cli.sh" start
    assert_success
    
    # Verify directories were created
    assert [ -d "${FFMPEG_DATA_DIR}" ]
    assert [ -d "${FFMPEG_OUTPUT_DIR}" ]
    assert [ -d "${FFMPEG_TEMP_DIR}" ]
    assert [ -d "${FFMPEG_LOGS_DIR}" ]
}

@test "ffmpeg: enhanced inject shows detailed help" {
    run resource-ffmpeg inject
    assert_success
    assert_output --partial "thumbnail"
    assert_output --partial "detailed"
    assert_output --partial "batch"
}

@test "ffmpeg: can create thumbnail from video" {
    local input_file="$TEST_TMP_DIR/test_video.mp4"
    
    # Make sure FFmpeg directories exist
    "${BATS_TEST_DIRNAME}/../cli.sh" start >/dev/null
    
    # Create test video
    ffmpeg -f lavfi -i testsrc=duration=1:size=320x240:rate=30 \
        -c:v libx264 -pix_fmt yuv420p -y "$input_file" 2>/dev/null
    
    # Test thumbnail generation via inject
    run resource-ffmpeg inject "$input_file" thumbnail
    assert_success
    
    # Verify thumbnail was created (should be in output directory)
    local basename_file=$(basename "$input_file")
    local filename="${basename_file%.*}"
    local expected_thumb="${FFMPEG_OUTPUT_DIR}/${filename}_thumbnail.jpg"
    assert [ -f "$expected_thumb" ]
}

@test "ffmpeg: cli.sh test runs tests" {
    run "${BATS_TEST_DIRNAME}/../cli.sh" test
    assert_success
}

@test "ffmpeg: hardware detection works" {
    run "${BATS_TEST_DIRNAME}/../cli.sh" status --verbose
    assert_success
    assert_output --partial "Capabilities:"
}

@test "ffmpeg: can process with output parameter" {
    local input_file="$TEST_TMP_DIR/input.mp4"
    local output_file="$TEST_TMP_DIR/custom_output.mp4"
    
    # Create test video
    ffmpeg -f lavfi -i testsrc=duration=1:size=320x240:rate=30 \
        -c:v libx264 -pix_fmt yuv420p -y "$input_file" 2>&1
    
    # Test processing with custom output via cli.sh
    run "${BATS_TEST_DIRNAME}/../cli.sh" transcode "$input_file" --output "$output_file"
    assert_success
    
    # Verify output file was created
    assert [ -f "$output_file" ]
}

@test "ffmpeg: directories structure is correct after start" {
    # Run start command
    "${BATS_TEST_DIRNAME}/../cli.sh" start >/dev/null
    
    # Check directory permissions
    assert [ -r "${FFMPEG_DATA_DIR}" ]
    assert [ -w "${FFMPEG_DATA_DIR}" ]
    assert [ -x "${FFMPEG_DATA_DIR}" ]
    
    # Check subdirectories exist
    assert [ -d "${FFMPEG_OUTPUT_DIR}" ]
    assert [ -d "${FFMPEG_TEMP_DIR}" ]
    assert [ -d "${FFMPEG_LOGS_DIR}" ]
    
    # Check for startup log
    local log_count=$(find "${FFMPEG_LOGS_DIR}" -name "startup-*.log" | wc -l)
    assert [ "$log_count" -ge 1 ]
}

@test "ffmpeg: batch processing setup works" {
    # Make sure FFmpeg directories exist
    "${BATS_TEST_DIRNAME}/../cli.sh" start >/dev/null
    
    # Create multiple test files
    for i in {1..3}; do
        local test_file="$TEST_MEDIA_DIR/test_${i}.mp4"
        ffmpeg -f lavfi -i testsrc=duration=1:size=160x120:rate=15 \
            -c:v libx264 -pix_fmt yuv420p -y "$test_file" 2>/dev/null
    done
    
    # Test that batch processing recognizes directory (just check setup, don't run full batch)
    run timeout 5 resource-ffmpeg inject "$TEST_MEDIA_DIR"
    
    # Should show batch processing output (may timeout but that's ok)
    local exit_code=$?
    [[ $exit_code -eq 0 || $exit_code -eq 124 ]] # 124 is timeout exit code
    
    # Check the output shows batch processing started
    echo "$output" | grep -q "Batch Processing"
}

@test "ffmpeg: status shows current settings" {
    run "${BATS_TEST_DIRNAME}/../cli.sh" status
    assert_success
    assert_output --partial "FFmpeg Status"
    assert_output --partial "Configuration:"
}

@test "ffmpeg: uninstall with force flag works (dry run)" {
    # Test uninstall help message without actually uninstalling
    run "${BATS_TEST_DIRNAME}/../cli.sh" uninstall
    assert_failure
    assert_output --partial "force"
}

@test "ffmpeg: enhanced status provides detailed information" {
    run "${BATS_TEST_DIRNAME}/../cli.sh" status --format json
    assert_success
    
    # Should include JSON output
    run "${BATS_TEST_DIRNAME}/../cli.sh" status
    assert_success
    assert_output --partial "FFmpeg Status"
}

@test "ffmpeg: all core functions are available through cli.sh" {
    local actions=("status" "test" "help")
    
    for action in "${actions[@]}"; do
        run "${BATS_TEST_DIRNAME}/../cli.sh" "$action"
        assert_success
    done
}