#!/usr/bin/env bats
# FFmpeg Integration Tests

setup() {
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-support/load"
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-assert/load"
    
    # Source FFmpeg configuration
    source "${BATS_TEST_DIRNAME}/../config/defaults.sh"
    ffmpeg::export_config
    
    # Source test utilities - comment out for now as path doesn't exist
    # source "${BATS_TEST_DIRNAME}/../../../../resources/tests/lib/base.sh"
    
    # Create temp directory for test files
    export TEST_TMP_DIR="$(mktemp -d)"
}

teardown() {
    # Clean up test files
    [[ -d "$TEST_TMP_DIR" ]] && rm -rf "$TEST_TMP_DIR"
}

@test "ffmpeg: CLI is available" {
    run command -v resource-ffmpeg
    assert_success
}

@test "ffmpeg: Status check works" {
    run resource-ffmpeg status
    assert_success
    assert_output --partial "FFmpeg Status"
}

@test "ffmpeg: Can check version" {
    run ffmpeg -version
    assert_success
    assert_output --partial "ffmpeg version"
}

@test "ffmpeg: Can list formats" {
    # Just check that ffmpeg -formats runs without error
    run ffmpeg -formats
    assert_success
}

@test "ffmpeg: Can list codecs" {
    # Just check that ffmpeg -codecs runs without error
    run ffmpeg -codecs
    assert_success
}

@test "ffmpeg: Can create test video" {
    local output_file="$TEST_TMP_DIR/test.mp4"
    
    # Generate a 1-second test video with color bars
    run ffmpeg -f lavfi -i testsrc=duration=1:size=320x240:rate=30 \
        -c:v libx264 -pix_fmt yuv420p -y "$output_file" 2>&1
    assert_success
    
    # Verify file was created
    assert [ -f "$output_file" ]
    
    # Verify file size is reasonable (>1KB)
    local size=$(stat -c%s "$output_file" 2>/dev/null || stat -f%z "$output_file" 2>/dev/null)
    assert [ "$size" -gt 1000 ]
}

@test "ffmpeg: Can extract audio from video" {
    local input_file="$TEST_TMP_DIR/input.mp4"
    local output_file="$TEST_TMP_DIR/output.mp3"
    
    # Create test video with audio
    ffmpeg -f lavfi -i testsrc=duration=1:size=320x240:rate=30 \
        -f lavfi -i sine=frequency=1000:duration=1 \
        -c:v libx264 -c:a aac -y "$input_file" 2>&1
    
    # Extract audio
    run ffmpeg -i "$input_file" -vn -acodec mp3 -y "$output_file" 2>&1
    assert_success
    
    # Verify audio file was created
    assert [ -f "$output_file" ]
}

@test "ffmpeg: Can resize video" {
    local input_file="$TEST_TMP_DIR/input.mp4"
    local output_file="$TEST_TMP_DIR/resized.mp4"
    
    # Create test video
    ffmpeg -f lavfi -i testsrc=duration=1:size=640x480:rate=30 \
        -c:v libx264 -pix_fmt yuv420p -y "$input_file" 2>&1
    
    # Resize to 320x240
    run ffmpeg -i "$input_file" -vf scale=320:240 -y "$output_file" 2>&1
    assert_success
    
    # Verify output file was created
    assert [ -f "$output_file" ]
    
    # Check dimensions using ffprobe
    if command -v ffprobe &>/dev/null; then
        local width=$(ffprobe -v error -select_streams v:0 -show_entries stream=width -of csv=s=x:p=0 "$output_file")
        assert [ "$width" = "320" ]
    fi
}

@test "ffmpeg: Can convert between formats" {
    local input_file="$TEST_TMP_DIR/input.mp4"
    local output_file="$TEST_TMP_DIR/output.webm"
    
    # Create test video
    ffmpeg -f lavfi -i testsrc=duration=1:size=320x240:rate=30 \
        -c:v libx264 -pix_fmt yuv420p -y "$input_file" 2>&1
    
    # Convert to WebM
    run ffmpeg -i "$input_file" -c:v libvpx -crf 10 -b:v 1M -y "$output_file" 2>&1
    assert_success
    
    # Verify WebM file was created
    assert [ -f "$output_file" ]
}

@test "ffmpeg: Can generate thumbnail from video" {
    local input_file="$TEST_TMP_DIR/input.mp4"
    local output_file="$TEST_TMP_DIR/thumbnail.jpg"
    
    # Create test video
    ffmpeg -f lavfi -i testsrc=duration=2:size=640x480:rate=30 \
        -c:v libx264 -pix_fmt yuv420p -y "$input_file" 2>&1
    
    # Extract frame at 1 second
    run ffmpeg -i "$input_file" -ss 00:00:01 -vframes 1 -y "$output_file" 2>&1
    assert_success
    
    # Verify thumbnail was created
    assert [ -f "$output_file" ]
    
    # Verify it's a valid image (has reasonable size)
    local size=$(stat -c%s "$output_file" 2>/dev/null || stat -f%z "$output_file" 2>/dev/null)
    assert [ "$size" -gt 1000 ]
}

@test "ffmpeg: Can concatenate videos" {
    local input1="$TEST_TMP_DIR/part1.mp4"
    local input2="$TEST_TMP_DIR/part2.mp4"
    local list_file="$TEST_TMP_DIR/list.txt"
    local output_file="$TEST_TMP_DIR/combined.mp4"
    
    # Create two test videos
    ffmpeg -f lavfi -i testsrc=duration=1:size=320x240:rate=30 \
        -c:v libx264 -pix_fmt yuv420p -y "$input1" 2>&1
    
    ffmpeg -f lavfi -i testsrc2=duration=1:size=320x240:rate=30 \
        -c:v libx264 -pix_fmt yuv420p -y "$input2" 2>&1
    
    # Create concat list
    echo "file '$input1'" > "$list_file"
    echo "file '$input2'" >> "$list_file"
    
    # Concatenate
    run ffmpeg -f concat -safe 0 -i "$list_file" -c copy -y "$output_file" 2>&1
    assert_success
    
    # Verify combined file was created
    assert [ -f "$output_file" ]
}

@test "ffmpeg: Resource shows as healthy in vrooli status" {
    run vrooli status --verbose
    assert_success
    assert_output --partial "ffmpeg"
    assert_output --partial "healthy"
}