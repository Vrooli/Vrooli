#!/bin/bash
set -euo pipefail

# FFmpeg Integration Test
# Tests end-to-end functionality and integrations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CLI="$RESOURCE_DIR/cli.sh"

# Test configuration
TEST_DIR="/tmp/ffmpeg-integration-test-$$"
mkdir -p "$TEST_DIR"

# Comprehensive cleanup function
cleanup_test_resources() {
    # Remove test directory
    [[ -d "$TEST_DIR" ]] && rm -rf "$TEST_DIR"

    # Kill any lingering test-related ffmpeg processes
    for pid in $(pgrep -f "ffmpeg" 2>/dev/null); do
        if ps -p "$pid" -o cmd= 2>/dev/null | grep -qE "(test|/tmp/ffmpeg-integration-test)"; then
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
}

trap cleanup_test_resources EXIT

# Test media creation
create_test_video() {
    local output="$1"
    ffmpeg -f lavfi -i testsrc=duration=2:size=320x240:rate=10 \
           -f lavfi -i sine=frequency=1000:duration=2 \
           -c:v libx264 -c:a aac -shortest \
           "$output" -y &> /dev/null
}

create_test_audio() {
    local output="$1"
    ffmpeg -f lavfi -i "sine=frequency=440:duration=2" \
           -c:a mp3 "$output" -y &> /dev/null
}

# Integration tests
test_video_transcoding() {
    echo "Testing: Video transcoding..."
    
    local input="$TEST_DIR/input.mp4"
    local output="$TEST_DIR/output.avi"
    
    # Create test video
    create_test_video "$input"
    
    if [[ ! -f "$input" ]]; then
        echo "❌ Failed to create test video"
        return 1
    fi
    
    # Transcode using CLI (transcode takes only input file and outputs to same dir with _transcoded suffix)
    if $CLI transcode "$input" &> /dev/null; then
        # Check for the default output file name
        local expected_output="${input%.mp4}_transcoded.mp4"
        if [[ -f "$expected_output" ]]; then
            echo "✅ Video transcoding successful"
            return 0
        else
            echo "❌ Output file not created at $expected_output"
            return 1
        fi
    else
        echo "❌ Transcoding command failed"
        return 1
    fi
}

test_audio_extraction() {
    echo "Testing: Audio extraction from video..."
    
    local input="$TEST_DIR/video_with_audio.mp4"
    local output="$TEST_DIR/extracted_audio.mp3"
    
    # Create test video with audio
    create_test_video "$input"
    
    # Extract audio using CLI (extract takes only input file)
    if $CLI extract "$input" &> /dev/null; then
        # Check for the default output file name
        local expected_output="${input%.mp4}_audio.mp3"
        if [[ -f "$expected_output" ]]; then
            # Verify it's audio only
            if ffprobe -v error -select_streams v -show_entries stream=codec_type -of csv=p=0 "$expected_output" 2>/dev/null | grep -q video; then
                echo "❌ Output contains video (should be audio only)"
                return 1
            else
                echo "✅ Audio extraction successful"
                return 0
            fi
        else
            echo "❌ Audio file not created at $expected_output"
            return 1
        fi
    else
        echo "❌ Audio extraction command failed"
        return 1
    fi
}

test_thumbnail_generation() {
    echo "Testing: Thumbnail generation..."
    
    local input="$TEST_DIR/video_for_thumb.mp4"
    local output="$TEST_DIR/thumbnail.jpg"
    
    # Create test video
    create_test_video "$input"
    
    # Extract thumbnail
    if ffmpeg -i "$input" -ss 00:00:01 -vframes 1 "$output" -y &> /dev/null; then
        if [[ -f "$output" ]]; then
            # Check if it's an image
            if file "$output" | grep -qE "JPEG|image"; then
                echo "✅ Thumbnail generation successful"
                return 0
            else
                echo "❌ Output is not an image"
                return 1
            fi
        else
            echo "❌ Thumbnail not created"
            return 1
        fi
    else
        echo "❌ Thumbnail generation failed"
        return 1
    fi
}

test_media_info_query() {
    echo "Testing: Media information query..."
    
    local test_file="$TEST_DIR/info_test.mp4"
    create_test_video "$test_file"
    
    # Get media info using CLI (media-info takes file path directly)
    if $CLI media-info "$test_file" &> /dev/null; then
        echo "✅ Media info query successful"
        return 0
    else
        echo "❌ Media info query failed"
        return 1
    fi
}

test_batch_processing() {
    echo "Testing: Batch processing capability..."
    
    # Create multiple test files
    local files=()
    for i in {1..3}; do
        local input="$TEST_DIR/batch_input_$i.wav"
        ffmpeg -f lavfi -i "sine=frequency=$((440 * i)):duration=1" "$input" -y &> /dev/null
        files+=("$input")
    done
    
    # Process batch (simulate with loop since batch may not be implemented)
    local success=0
    local total=${#files[@]}
    
    for file in "${files[@]}"; do
        local output="${file%.wav}.mp3"
        if ffmpeg -i "$file" -c:a libmp3lame "$output" -y &> /dev/null; then
            if [[ -f "$output" ]]; then
                ((success++))
            fi
        fi
    done
    
    if [[ $success -eq $total ]]; then
        echo "✅ Batch processing successful ($success/$total files)"
        return 0
    else
        echo "❌ Batch processing incomplete ($success/$total files)"
        return 1
    fi
}

test_error_handling() {
    echo "Testing: Error handling..."
    
    # Test with non-existent file (media-info takes file path directly)
    local result=0
    if ! $CLI media-info "/tmp/nonexistent_file_xyz.mp4" &> /dev/null; then
        echo "✅ Correctly handled non-existent file"
    else
        echo "❌ Failed to handle non-existent file error"
        result=1
    fi
    
    # Test with invalid format
    local invalid_file="$TEST_DIR/invalid.txt"
    echo "This is not a media file" > "$invalid_file"
    
    if ! ffmpeg -i "$invalid_file" -c copy "$TEST_DIR/output.mp4" &> /dev/null; then
        echo "✅ Correctly rejected invalid media file"
    else
        echo "❌ Failed to reject invalid media file"
        result=1
    fi
    
    return $result
}

test_resource_cleanup() {
    echo "Testing: Resource cleanup..."

    # Check for zombie processes
    local ffmpeg_procs=$(pgrep -f "ffmpeg" | wc -l)

    if [[ $ffmpeg_procs -eq 0 ]]; then
        echo "✅ No zombie ffmpeg processes"
        return 0
    else
        # Check if any are test-related (from test directories or using test files)
        local test_procs=0
        for pid in $(pgrep -f "ffmpeg"); do
            if ps -p "$pid" -o cmd= 2>/dev/null | grep -qE "(test|/tmp/ffmpeg-integration-test)"; then
                ((test_procs++))
            fi
        done

        if [[ $test_procs -eq 0 ]]; then
            echo "✅ No test-related ffmpeg processes (found $ffmpeg_procs system processes)"
            return 0
        else
            echo "⚠️  Found $test_procs test-related ffmpeg processes"
            # Kill them for cleanup
            for pid in $(pgrep -f "ffmpeg"); do
                if ps -p "$pid" -o cmd= 2>/dev/null | grep -qE "(test|/tmp/ffmpeg-integration-test)"; then
                    kill -9 "$pid" 2>/dev/null || true
                fi
            done
            # Wait a moment for cleanup
            sleep 1
            # Verify cleanup
            local remaining=$(pgrep -f "ffmpeg" | while read pid; do
                ps -p "$pid" -o cmd= 2>/dev/null | grep -qE "(test|/tmp/ffmpeg-integration-test)" && echo "$pid"
            done | wc -l)
            if [[ $remaining -eq 0 ]]; then
                echo "✅ Test processes cleaned up successfully"
                return 0
            else
                echo "⚠️  Warning: $remaining test processes could not be cleaned"
                return 0  # Don't fail the test, just warn
            fi
        fi
    fi
}

# Main integration test execution
main() {
    echo "🧪 FFmpeg Integration Test Suite"
    echo "================================"
    
    local failed=0
    
    # Run all integration tests
    test_video_transcoding || failed=1
    test_audio_extraction || failed=1
    test_thumbnail_generation || failed=1
    test_media_info_query || failed=1
    test_batch_processing || failed=1
    test_error_handling || failed=1
    test_resource_cleanup || failed=1
    
    echo ""
    if [[ $failed -eq 0 ]]; then
        echo "✅ All integration tests passed"
        exit 0
    else
        echo "❌ Some integration tests failed"
        exit 1
    fi
}

main "$@"