#!/bin/bash
# ====================================================================
# Audio Test Helper Functions
# ====================================================================
# Provides functions to work with audio metadata in tests
#

# Get the directory of this script
AUDIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
METADATA_FILE="$AUDIO_DIR/audio.yaml"

# Load metadata for a specific audio file
get_audio_metadata() {
    local audio_path="$1"
    
    if [[ ! -f "$METADATA_FILE" ]]; then
        echo "ERROR: Metadata file not found: $METADATA_FILE" >&2
        return 1
    fi
    
    # Use yq to query the metadata (assumes yq is installed)
    if command -v yq >/dev/null 2>&1; then
        yq eval ".audio[][] | select(.path == \"$audio_path\")" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Get all audio files with a specific tag
get_audio_by_tag() {
    local tag="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".audio[][] | select(.tags[] == \"$tag\") | .path" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Get audio files for a specific test suite
get_test_suite_audio() {
    local suite_name="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".testSuites.${suite_name}[]" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Get expected transcription for an audio file
get_expected_transcription() {
    local audio_path="$1"
    
    if command -v yq >/dev/null 2>&1; then
        # First try to get from testData
        local expected=$(yq eval ".audio[][] | select(.path == \"$audio_path\") | .testData.expectedTranscription" "$METADATA_FILE")
        
        # If not found or null, check transcriptionExpectations
        if [[ "$expected" == "null" ]] || [[ -z "$expected" ]]; then
            expected=$(yq eval ".transcriptionExpectations.\"$audio_path\"" "$METADATA_FILE")
        fi
        
        echo "$expected"
    else
        return 1
    fi
}

# Check if audio file is expected to be valid
is_valid_audio() {
    local audio_path="$1"
    
    if command -v yq >/dev/null 2>&1; then
        local is_valid=$(yq eval ".audio[][] | select(.path == \"$audio_path\") | .testData.isValid" "$METADATA_FILE")
        [[ "$is_valid" == "true" ]]
    else
        return 1
    fi
}

# Get audio duration from metadata
get_audio_duration() {
    local audio_path="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".audio[][] | select(.path == \"$audio_path\") | .duration" "$METADATA_FILE"
    else
        return 1
    fi
}

# Get audio content type (speech, music, silence, etc.)
get_audio_content_type() {
    local audio_path="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".audio[][] | select(.path == \"$audio_path\") | .contentType" "$METADATA_FILE"
    else
        return 1
    fi
}

# Example usage in tests:
# 
# # In a BATS test:
# @test "transcription accuracy for speech files" {
#     source "$FIXTURES_DIR/audio/test_helpers.sh"
#     
#     # Get all speech files
#     while IFS= read -r audio_path; do
#         # Check if it's valid audio
#         if is_valid_audio "$audio_path"; then
#             # Get expected transcription details
#             expected=$(get_expected_transcription "$audio_path")
#             
#             # Run transcription
#             run transcribe_audio "$AUDIO_DIR/$audio_path"
#             assert_success
#             
#             # Validate based on expectations
#             if [[ -n "$expected" ]]; then
#                 assert_output --partial "$expected"
#             fi
#         fi
#     done < <(get_audio_by_tag "speech")
# }
#
# @test "format support validation" {
#     source "$FIXTURES_DIR/audio/test_helpers.sh"
#     
#     # Test each format from the test suite
#     while IFS= read -r audio_path; do
#         if is_valid_audio "$audio_path"; then
#             run process_audio "$AUDIO_DIR/$audio_path"
#             assert_success
#         else
#             run process_audio "$AUDIO_DIR/$audio_path"
#             assert_failure
#         fi
#     done < <(get_test_suite_audio "formatSupport")
# }