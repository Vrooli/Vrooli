#!/usr/bin/env bash
# Fixture Loading Helper for BATS Tests
# Provides utilities to load and use test fixtures in BATS tests with mocked responses

# Get the fixtures directory path
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
FIXTURES_DIR="${APP_ROOT}/__test/fixtures"

#######################################
# Load fixture file path by category and name
# Arguments:
#   $1 - category (audio, documents, images, workflows)
#   $2 - fixture name or path within category
# Returns:
#   Full path to fixture file
# Example:
#   audio_file=$(fixture_get_path "audio" "whisper/test_speech.mp3")
#######################################
fixture_get_path() {
    local category="$1"
    local fixture_name="$2"
    
    local fixture_path="$FIXTURES_DIR/$category/$fixture_name"
    
    if [[ ! -f "$fixture_path" ]]; then
        echo "ERROR: Fixture not found: $fixture_path" >&2
        return 1
    fi
    
    echo "$fixture_path"
}

#######################################
# Get all fixtures of a specific category
# Arguments:
#   $1 - category (audio, documents, images, workflows)
#   $2 - optional pattern (e.g., "*.mp3")
# Returns:
#   Array of fixture paths
#######################################
fixture_get_all() {
    local category="$1"
    local pattern="${2:-*}"
    
    local category_dir="$FIXTURES_DIR/$category"
    
    if [[ ! -d "$category_dir" ]]; then
        echo "ERROR: Category not found: $category_dir" >&2
        return 1
    fi
    
    find "$category_dir" -name "$pattern" -type f 2>/dev/null | sort
}

#######################################
# Create a mock API response for fixture processing
# This allows BATS tests to use real fixture files with expected responses
# Arguments:
#   $1 - fixture_type (transcription, ocr, document_parse, etc.)
#   $2 - fixture_file_path
#   $3 - expected_response_json (optional, uses default if not provided)
#######################################
fixture_mock_response() {
    local fixture_type="$1"
    local fixture_file="$2"
    local expected_response="${3:-}"
    
    local fixture_basename
    fixture_basename=$(basename "$fixture_file")
    
    case "$fixture_type" in
        "transcription")
            if [[ -z "$expected_response" ]]; then
                case "$fixture_basename" in
                    "test_speech.mp3")
                        expected_response='{"text":"I have a dream that one day this nation will rise up"}'
                        ;;
                    "test_silent.wav")
                        expected_response='{"text":""}'
                        ;;
                    "test_noise.mp3")
                        expected_response='{"text":"[Music]"}'
                        ;;
                    "test_corrupted.wav")
                        expected_response='{"error":"Invalid audio format"}'
                        ;;
                    *)
                        expected_response='{"text":"Sample transcription"}'
                        ;;
                esac
            fi
            ;;
        "ocr")
            if [[ -z "$expected_response" ]]; then
                case "$fixture_basename" in
                    "1_simple_text.png")
                        expected_response='{"text":"Hello World"}'
                        ;;
                    "2_multi_font_text.png")
                        expected_response='{"text":"Multiple fonts and sizes"}'
                        ;;
                    *)
                        expected_response='{"text":"Sample OCR text"}'
                        ;;
                esac
            fi
            ;;
        "document_parse")
            if [[ -z "$expected_response" ]]; then
                expected_response='{"elements":[{"type":"title","text":"Document Title"},{"type":"paragraph","text":"Document content"}]}'
            fi
            ;;
        *)
            if [[ -z "$expected_response" ]]; then
                expected_response='{"result":"success","data":"processed"}'
            fi
            ;;
    esac
    
    # Set up mock curl response
    mock_curl_response "$expected_response"
}

#######################################
# Assert that a fixture file was processed correctly
# Arguments:
#   $1 - fixture_file_path
#   $2 - expected_curl_args (e.g., "--data-binary")
#######################################
fixture_assert_processed() {
    local fixture_file="$1"
    local expected_args="$2"
    
    # Verify curl was called with the fixture file
    assert_curl_called_with "$expected_args @$fixture_file"
}

#######################################
# Load fixture metadata for validation
# Arguments:
#   $1 - category (audio, documents, images)
# Returns:
#   Outputs metadata yaml content
#######################################
fixture_load_metadata() {
    local category="$1"
    local metadata_file="$FIXTURES_DIR/$category/metadata.yaml"
    
    if [[ ! -f "$metadata_file" ]]; then
        echo "ERROR: Metadata not found: $metadata_file" >&2
        return 1
    fi
    
    cat "$metadata_file"
}

# Export functions for use in BATS tests
export -f fixture_get_path
export -f fixture_get_all 
export -f fixture_mock_response
export -f fixture_assert_processed
export -f fixture_load_metadata
export FIXTURES_DIR