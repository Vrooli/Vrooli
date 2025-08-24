#!/bin/bash
# Vrooli Audio Fixtures Type-Specific Validator
# Validates audio-specific requirements and tests with available resources

set -e

# Get APP_ROOT using cached value or compute once (3 levels up: __test/fixtures/audio/validate-audio.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test/fixtures/audio"
FIXTURES_DIR="$SCRIPT_DIR"
METADATA_FILE="$FIXTURES_DIR/metadata.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_AUDIO=0
VALID_AUDIO=0
FAILED_AUDIO=0
TESTED_AUDIO=0

print_section() {
    echo -e "${YELLOW}--- $1 ---${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# Check if required tools are available
check_audio_tools() {
    local missing_tools=()
    
    if ! command -v ffprobe >/dev/null 2>&1; then
        missing_tools+=("ffprobe (ffmpeg)")
    fi
    
    if ! command -v file >/dev/null 2>&1; then
        missing_tools+=("file")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        print_warning "Some audio validation tools are missing: ${missing_tools[*]}"
        print_info "Install ffmpeg for complete validation: sudo apt-get install ffmpeg"
        return 1
    fi
    
    return 0
}

# Validate individual audio file
validate_audio_file() {
    local audio_path="$1"
    local expected_format="$2"
    local expected_duration="$3"
    local expected_sample_rate="$4"
    local expected_channels="$5"
    local content_type="$6"
    
    TOTAL_AUDIO=$((TOTAL_AUDIO + 1))
    
    if [[ ! -f "$audio_path" ]]; then
        print_error "Audio file not found: $audio_path"
        FAILED_AUDIO=$((FAILED_AUDIO + 1))
        return 1
    fi
    
    local errors=0
    local filename
    filename=$(basename "$audio_path")
    
    # Check file exists and is readable
    if [[ ! -r "$audio_path" ]]; then
        print_error "$filename: Not readable"
        errors=$((errors + 1))
    fi
    
    # Check file size
    local actual_size
    actual_size=$(stat -c%s "$audio_path" 2>/dev/null || echo "0")
    if [[ $actual_size -eq 0 && "$content_type" != "non-audio" ]]; then
        print_error "$filename: Empty file (expected audio content)"
        errors=$((errors + 1))
    fi
    
    # Handle non-audio files (edge cases)
    if [[ "$content_type" == "non-audio" ]]; then
        if [[ "$expected_format" == "txt" ]]; then
            # This is expected to be a non-audio file for testing
            print_success "$filename: Non-audio edge case validated"
            VALID_AUDIO=$((VALID_AUDIO + 1))
            return 0
        fi
    fi
    
    # Basic file type validation
    if command -v file >/dev/null 2>&1; then
        local file_type=$(file -b --mime-type "$audio_path" 2>/dev/null)
        
        case "$expected_format" in
            "mp3")
                if [[ "$file_type" != "audio/mpeg" ]]; then
                    print_error "$filename: Expected MP3, got $file_type"
                    errors=$((errors + 1))
                fi
                ;;
            "wav")
                if [[ "$file_type" != "audio/x-wav" && "$file_type" != "audio/wav" ]]; then
                    print_error "$filename: Expected WAV, got $file_type"
                    errors=$((errors + 1))
                fi
                ;;
            "ogg")
                if [[ "$file_type" != "audio/ogg" && "$file_type" != "application/ogg" ]]; then
                    print_error "$filename: Expected OGG, got $file_type"
                    errors=$((errors + 1))
                fi
                ;;
            "flac")
                if [[ "$file_type" != "audio/flac" ]]; then
                    print_error "$filename: Expected FLAC, got $file_type"
                    errors=$((errors + 1))
                fi
                ;;
        esac
    fi
    
    # Advanced validation with ffprobe
    if command -v ffprobe >/dev/null 2>&1 && [[ "$content_type" != "non-audio" ]]; then
        local ffprobe_output=$(ffprobe -v quiet -print_format json -show_format -show_streams "$audio_path" 2>/dev/null || echo "")
        
        if [[ -n "$ffprobe_output" ]]; then
            # Extract audio properties using basic parsing (avoid jq dependency)
            local actual_duration=$(echo "$ffprobe_output" | grep -o '"duration": *"[0-9.]*"' | head -1 | grep -o '[0-9.]*')
            local actual_sample_rate=$(echo "$ffprobe_output" | grep -o '"sample_rate": *"[0-9]*"' | head -1 | grep -o '[0-9]*')
            local actual_channels=$(echo "$ffprobe_output" | grep -o '"channels": *[0-9]*' | head -1 | grep -o '[0-9]*')
            
            # Validate duration (allow 10% tolerance)
            if [[ -n "$actual_duration" && "$expected_duration" != "0" ]]; then
                local duration_diff=$(echo "$actual_duration $expected_duration" | awk '{print ($1 > $2) ? $1 - $2 : $2 - $1}')
                local duration_tolerance=$(echo "$expected_duration" | awk '{print $1 * 0.1}')
                
                if (( $(echo "$duration_diff > $duration_tolerance" | bc -l 2>/dev/null || echo "0") )); then
                    print_warning "$filename: Duration mismatch. Expected ~${expected_duration}s, got ${actual_duration}s"
                fi
            fi
            
            # Validate sample rate
            if [[ -n "$actual_sample_rate" && "$expected_sample_rate" != "0" ]]; then
                if [[ "$actual_sample_rate" != "$expected_sample_rate" ]]; then
                    print_warning "$filename: Sample rate mismatch. Expected $expected_sample_rate Hz, got $actual_sample_rate Hz"
                fi
            fi
            
            # Validate channels
            if [[ -n "$actual_channels" && "$expected_channels" != "0" ]]; then
                if [[ "$actual_channels" != "$expected_channels" ]]; then
                    print_warning "$filename: Channel count mismatch. Expected $expected_channels, got $actual_channels"
                fi
            fi
            
            # Check if audio stream exists
            if ! echo "$ffprobe_output" | grep -q '"codec_type": *"audio"'; then
                print_error "$filename: No audio stream found"
                errors=$((errors + 1))
            fi
        else
            print_warning "$filename: Could not analyze with ffprobe"
        fi
    fi
    
    if [[ $errors -gt 0 ]]; then
        print_error "$filename: $errors validation errors"
        FAILED_AUDIO=$((FAILED_AUDIO + 1))
        return 1
    fi
    
    print_success "$filename: Audio validation passed"
    VALID_AUDIO=$((VALID_AUDIO + 1))
    return 0
}

# Test transcription with available speech recognition tools
test_transcription_capabilities() {
    print_section "Testing Transcription Capabilities"
    
    # Check if Whisper API is available (localhost)
    if curl -s http://localhost:8090/health >/dev/null 2>&1; then
        print_info "Whisper API available - testing transcription"
        
        # Find speech test files
        local speech_files=($(find "$FIXTURES_DIR" -name "*speech*.mp3" -o -name "*speech*.wav" | head -2))
        
        for audio in "${speech_files[@]}"; do
            local filename=$(basename "$audio")
            print_info "Testing transcription on $filename"
            
            # Simple transcription test (just check API responds)
            local response=$(curl -s -X POST http://localhost:8090/transcribe \
                -F "audio=@$audio" \
                -F "model=base" 2>/dev/null || echo "")
            
            if [[ -n "$response" ]]; then
                print_success "$filename: Transcription API responded"
                TESTED_AUDIO=$((TESTED_AUDIO + 1))
            else
                print_warning "$filename: Transcription test failed"
            fi
        done
    else
        print_info "Whisper API not available - transcription testing skipped"
    fi
}

# Test silence detection
test_silence_detection() {
    print_section "Testing Silence Detection"
    
    if ! command -v ffprobe >/dev/null 2>&1; then
        print_info "ffprobe not available - silence detection skipped"
        return 0
    fi
    
    # Find silent test files
    local silent_files=($(find "$FIXTURES_DIR" -name "*silent*" | head -1))
    
    for audio in "${silent_files[@]}"; do
        local filename=$(basename "$audio")
        print_info "Testing silence detection on $filename"
        
        # Use ffprobe to detect audio levels
        local max_volume=$(ffprobe -v quiet -show_entries frame=metadata:tags=lavfi.astats.Overall.Max_level \
            -f lavfi -i "amovie='$audio',astats=metadata=1:reset=1" 2>/dev/null | grep "Max_level" | head -1 | cut -d'=' -f2)
        
        if [[ -n "$max_volume" ]]; then
            # Check if volume is very low (indicating silence)
            if (( $(echo "$max_volume < -40" | bc -l 2>/dev/null || echo "1") )); then
                print_success "$filename: Silence detected correctly"
            else
                print_warning "$filename: Expected silence but found audio signal"
            fi
        else
            print_info "$filename: Could not analyze audio levels"
        fi
    done
}

# Validate all audio files from metadata
validate_all_audio() {
    print_section "Validating Audio Files"
    
    if [[ ! -f "$METADATA_FILE" ]]; then
        print_error "metadata.yaml not found"
        return 1
    fi
    
    # Check if yq is available
    if ! command -v yq >/dev/null 2>&1; then
        print_error "yq required for metadata parsing"
        return 1
    fi
    
    # Extract all audio paths and their properties
    local audio_data=$(yq eval '.audio | .. | select(has("path")) | [.path, .format, .duration, .sampleRate, .channels, .contentType] | @csv' "$METADATA_FILE" 2>/dev/null)
    
    if [[ -z "$audio_data" ]]; then
        print_error "No audio data found in metadata.yaml"
        return 1
    fi
    
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            # Parse CSV line (path, format, duration, sampleRate, channels, contentType)
            local path=$(echo "$line" | cut -d',' -f1 | tr -d '"')
            local format=$(echo "$line" | cut -d',' -f2 | tr -d '"')
            local duration=$(echo "$line" | cut -d',' -f3 | tr -d '"')
            local sample_rate=$(echo "$line" | cut -d',' -f4 | tr -d '"')
            local channels=$(echo "$line" | cut -d',' -f5 | tr -d '"')
            local content_type=$(echo "$line" | cut -d',' -f6 | tr -d '"')
            
            if [[ -n "$path" && "$path" != "null" ]]; then
                local full_path="$FIXTURES_DIR/$path"
                validate_audio_file "$full_path" "$format" "$duration" "$sample_rate" "$channels" "$content_type"
            fi
        fi
    done <<< "$audio_data"
}

# Generate validation report
generate_report() {
    print_section "Audio Validation Summary"
    
    echo "Total audio files validated: $TOTAL_AUDIO"
    echo "Valid audio files: $VALID_AUDIO"
    echo "Failed validations: $FAILED_AUDIO"
    echo "Files tested with tools: $TESTED_AUDIO"
    echo
    
    if [[ $FAILED_AUDIO -eq 0 ]]; then
        print_success "All audio fixtures validated successfully!"
        echo -e "${GREEN}✅ Audio formats verified${NC}"
        echo -e "${GREEN}✅ Duration and properties validated${NC}"
        echo -e "${GREEN}✅ Files accessible and readable${NC}"
    else
        print_error "$FAILED_AUDIO audio validation failures"
        echo -e "${RED}❌ Fix audio issues before running tests${NC}"
        return 1
    fi
    
    # Tool availability report
    echo
    print_info "Audio Processing Tool Availability:"
    if command -v ffprobe >/dev/null 2>&1; then
        echo "  - ffprobe: ✅ Available"
    else
        echo "  - ffprobe: ❌ Missing"
    fi
    
    if command -v ffmpeg >/dev/null 2>&1; then
        echo "  - ffmpeg: ✅ Available"
    else
        echo "  - ffmpeg: ❌ Missing"
    fi
    
    if curl -s http://localhost:8090/health >/dev/null 2>&1; then
        echo "  - Whisper API: ✅ Available"
    else
        echo "  - Whisper API: ❌ Not running"
    fi
}

main() {
    print_section "Audio Fixtures Type-Specific Validation"
    
    check_audio_tools || print_info "Continuing with limited validation capabilities"
    validate_all_audio
    
    # Run optional tests if tools/services are available
    test_transcription_capabilities
    test_silence_detection
    
    echo
    generate_report
}

# Show usage if requested
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "Vrooli Audio Fixtures Type-Specific Validator"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help    Show this help message"
    echo ""
    echo "This validator performs audio-specific validation:"
    echo "  - File format verification (MP3, WAV, OGG, FLAC)"
    echo "  - Duration and audio properties validation"
    echo "  - Audio stream detection"
    echo "  - Transcription capability testing (if Whisper available)"
    echo "  - Silence detection testing"
    echo ""
    echo "Required tools for full validation:"
    echo "  - ffmpeg/ffprobe (audio analysis)"
    echo "  - file command"
    echo "  - bc (for floating point calculations)"
    echo ""
    echo "Optional resources:"
    echo "  - Whisper API at localhost:8090 (for transcription testing)"
    exit 0
fi

# Run validation if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi