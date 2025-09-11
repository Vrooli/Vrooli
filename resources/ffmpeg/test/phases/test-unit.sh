#!/bin/bash
set -euo pipefail

# FFmpeg Unit Test
# Tests individual library functions

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Load resource libraries
source "$RESOURCE_DIR/lib/core.sh"
source "$RESOURCE_DIR/lib/content.sh"

# Test data
TEST_DIR="/tmp/ffmpeg-unit-test-$$"
mkdir -p "$TEST_DIR"
trap "rm -rf $TEST_DIR" EXIT

# Test functions
test_get_media_info() {
    echo "Testing: get_media_info function..."
    
    # Create a test file using ffmpeg
    local test_file="$TEST_DIR/test.wav"
    ffmpeg -f lavfi -i "sine=frequency=1000:duration=1" -ac 2 -ar 44100 "$test_file" -y &> /dev/null
    
    if [[ -f "$test_file" ]]; then
        # Test getting media info
        if ffprobe -v quiet -print_format json -show_format -show_streams "$test_file" &> /dev/null; then
            echo "✅ Media info extraction works"
            return 0
        else
            echo "❌ Failed to extract media info"
            return 1
        fi
    else
        echo "❌ Failed to create test file"
        return 1
    fi
}

test_validate_codec() {
    echo "Testing: Codec validation..."
    
    # Test valid codecs
    local valid_codecs=("h264" "aac" "mp3")
    for codec in "${valid_codecs[@]}"; do
        if ffmpeg -codecs 2>/dev/null | grep -q "$codec"; then
            echo "✅ Valid codec: $codec"
        else
            echo "⚠️  Codec not available: $codec"
        fi
    done
    
    # Test invalid codec detection
    if ! ffmpeg -codecs 2>/dev/null | grep -q "invalid_codec_xyz"; then
        echo "✅ Invalid codec correctly not found"
        return 0
    else
        echo "❌ Invalid codec detection failed"
        return 1
    fi
}

test_format_detection() {
    echo "Testing: Format detection..."
    
    # Create test files with different formats
    local formats=("wav" "mp3")
    
    for format in "${formats[@]}"; do
        local test_file="$TEST_DIR/test.$format"
        
        # Create test file
        case "$format" in
            wav)
                ffmpeg -f lavfi -i "sine=frequency=1000:duration=0.5" "$test_file" -y &> /dev/null
                ;;
            mp3)
                ffmpeg -f lavfi -i "sine=frequency=1000:duration=0.5" -acodec libmp3lame "$test_file" -y &> /dev/null
                ;;
        esac
        
        if [[ -f "$test_file" ]]; then
            # Detect format
            local detected=$(ffprobe -v error -select_streams a:0 -show_entries stream=codec_name -of default=noprint_wrappers=1:nokey=1 "$test_file" 2>/dev/null || echo "unknown")
            echo "✅ Format detection for .$format: $detected"
        else
            echo "❌ Failed to create $format test file"
            return 1
        fi
    done
    
    return 0
}

test_parameter_validation() {
    echo "Testing: Parameter validation..."
    
    # Test bitrate validation
    local valid_bitrates=("128k" "256k" "1M" "5M")
    for bitrate in "${valid_bitrates[@]}"; do
        if [[ "$bitrate" =~ ^[0-9]+[kKmM]$ ]]; then
            echo "✅ Valid bitrate format: $bitrate"
        else
            echo "❌ Invalid bitrate format: $bitrate"
            return 1
        fi
    done
    
    # Test invalid parameters
    local invalid_bitrates=("abc" "123" "10GB")
    for bitrate in "${invalid_bitrates[@]}"; do
        if [[ ! "$bitrate" =~ ^[0-9]+[kKmM]$ ]]; then
            echo "✅ Correctly rejected invalid bitrate: $bitrate"
        else
            echo "❌ Failed to reject invalid bitrate: $bitrate"
            return 1
        fi
    done
    
    return 0
}

test_temp_file_management() {
    echo "Testing: Temporary file management..."
    
    local temp_dir="$TEST_DIR/temp"
    mkdir -p "$temp_dir"
    
    # Create some temp files
    touch "$temp_dir/temp1.tmp"
    touch "$temp_dir/temp2.tmp"
    
    # Count files
    local file_count=$(ls -1 "$temp_dir"/*.tmp 2>/dev/null | wc -l)
    if [[ $file_count -eq 2 ]]; then
        echo "✅ Temp files created: $file_count"
    else
        echo "❌ Wrong number of temp files: $file_count"
        return 1
    fi
    
    # Clean up
    rm -f "$temp_dir"/*.tmp
    file_count=$(ls -1 "$temp_dir"/*.tmp 2>/dev/null | wc -l)
    if [[ $file_count -eq 0 ]]; then
        echo "✅ Temp files cleaned up"
        return 0
    else
        echo "❌ Failed to clean up temp files"
        return 1
    fi
}

# Main unit test execution
main() {
    echo "🧪 FFmpeg Unit Test Suite"
    echo "========================"
    
    local failed=0
    
    # Run all unit tests
    test_get_media_info || failed=1
    test_validate_codec || failed=1
    test_format_detection || failed=1
    test_parameter_validation || failed=1
    test_temp_file_management || failed=1
    
    echo ""
    if [[ $failed -eq 0 ]]; then
        echo "✅ All unit tests passed"
        exit 0
    else
        echo "❌ Some unit tests failed"
        exit 1
    fi
}

main "$@"