#!/bin/bash
set -euo pipefail

# FFmpeg Smoke Test
# Quick validation that FFmpeg is installed and operational

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Load resource libraries
source "$RESOURCE_DIR/lib/core.sh"

# Test functions
test_ffmpeg_installed() {
    echo "Testing: FFmpeg installation..."
    if command -v ffmpeg &> /dev/null; then
        echo "✅ FFmpeg is installed"
        return 0
    else
        echo "❌ FFmpeg is not installed"
        return 1
    fi
}

test_ffmpeg_version() {
    echo "Testing: FFmpeg version..."
    if ffmpeg -version &> /dev/null; then
        local version=$(ffmpeg -version 2>&1 | head -n1 | grep -oP 'version \K[0-9.]+' || echo "unknown")
        echo "✅ FFmpeg version: $version"
        return 0
    else
        echo "❌ Failed to get FFmpeg version"
        return 1
    fi
}

test_basic_codecs() {
    echo "Testing: Basic codec support..."
    # Just verify ffmpeg can handle basic formats
    local test_file="/tmp/ffmpeg_codec_test_$$.wav"
    
    # Create a simple test file
    if ffmpeg -f lavfi -i "sine=frequency=1000:duration=0.1" "$test_file" -y &> /dev/null; then
        echo "✅ Can create audio files"
        rm -f "$test_file"
    else
        echo "❌ Failed to create test audio"
        return 1
    fi
    
    # Test if we can query codecs at all
    if ffmpeg -hide_banner -codecs &> /dev/null; then
        echo "✅ Codec querying works"
    else
        echo "❌ Cannot query codecs"
        return 1
    fi
    
    # Test basic format support (create an actual mp4 to verify)
    local test_mp4="/tmp/ffmpeg_format_test_$$.mp4"
    if ffmpeg -f lavfi -i "testsrc=duration=0.1:size=320x240:rate=1" -c:v mpeg4 "$test_mp4" -y &> /dev/null; then
        if [[ -f "$test_mp4" ]]; then
            echo "✅ MP4 format supported"
            rm -f "$test_mp4"
        else
            echo "❌ MP4 file creation failed"
            return 1
        fi
    else
        echo "❌ MP4 format not supported"
        return 1
    fi
    
    return 0
}

test_cli_commands() {
    echo "Testing: CLI command availability..."
    local cli="$RESOURCE_DIR/cli.sh"
    
    if [[ ! -x "$cli" ]]; then
        echo "❌ CLI script not executable: $cli"
        return 1
    fi
    
    # Test help command
    if $cli help &> /dev/null; then
        echo "✅ CLI help command works"
    else
        echo "❌ CLI help command failed"
        return 1
    fi
    
    # Test status command
    if $cli status &> /dev/null; then
        echo "✅ CLI status command works"
    else
        echo "❌ CLI status command failed"
        return 1
    fi
    
    return 0
}

# Main smoke test execution
main() {
    echo "🧪 FFmpeg Smoke Test Suite"
    echo "=========================="
    
    local failed=0
    
    # Run all smoke tests
    test_ffmpeg_installed || failed=1
    test_ffmpeg_version || failed=1
    test_basic_codecs || failed=1
    test_cli_commands || failed=1
    
    echo ""
    if [[ $failed -eq 0 ]]; then
        echo "✅ All smoke tests passed"
        exit 0
    else
        echo "❌ Some smoke tests failed"
        exit 1
    fi
}

main "$@"