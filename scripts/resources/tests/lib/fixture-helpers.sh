#!/usr/bin/env bash
# Fixture Data Helper Library for Resource Integration Tests
# Provides standardized access to test fixtures and validation utilities

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
_HERE="${APP_ROOT}/scripts/resources/tests/lib"

# shellcheck disable=SC1091
source "${_HERE}/../../../../lib/utils/var.sh"

# Fixture data base path
FIXTURE_BASE="${FIXTURE_BASE:-${var_SCRIPTS_TEST_DIR}/fixtures/data}"

#######################################
# FIXTURE PATH HELPERS
#######################################

# Get path to audio fixtures
fixture_helpers::get_audio_fixture_path() {
    echo "$FIXTURE_BASE/audio"
}

# Get path to document fixtures
fixture_helpers::get_document_fixture_path() {
    echo "$FIXTURE_BASE/documents"
}

# Get path to image fixtures
fixture_helpers::get_image_fixture_path() {
    echo "$FIXTURE_BASE/images"
}

# Get path to workflow fixtures
fixture_helpers::get_workflow_fixture_path() {
    echo "$FIXTURE_BASE/workflows"
}

#######################################
# AUDIO FIXTURE HELPERS
#######################################

# Get a test audio file for speech transcription
fixture_helpers::get_speech_audio_fixture() {
    local type="${1:-short}"
    local audio_dir="$(fixture_helpers::get_audio_fixture_path)"
    
    case "$type" in
        "short")
            echo "$audio_dir/speech_test_short.mp3"
            ;;
        "long")
            echo "$audio_dir/speech_mlk_dream.mp3"
            ;;
        "sample")
            echo "$audio_dir/speech_sample.ogg"
            ;;
        "whisper")
            echo "$audio_dir/whisper/test_speech.mp3"
            ;;
        "silent")
            echo "$audio_dir/whisper/test_silent.wav"
            ;;
        "corrupted")
            echo "$audio_dir/whisper/test_corrupted.wav"
            ;;
        "noise")
            echo "$audio_dir/whisper/test_noise.mp3"
            ;;
        *)
            echo "$audio_dir/speech_test_short.mp3"
            ;;
    esac
}

# Get expected transcription for validation
fixture_helpers::get_expected_transcription() {
    local audio_file="$1"
    local filename=$(basename "$audio_file")
    
    # Known transcriptions for test files
    case "$filename" in
        "speech_test_short.mp3")
            echo "this is a test of speech recognition"
            ;;
        "test_speech.mp3")
            echo "hello world this is a test"
            ;;
        *)
            echo ""  # Unknown file, no expected transcription
            ;;
    esac
}

#######################################
# DOCUMENT FIXTURE HELPERS
#######################################

# Get a test document for processing
fixture_helpers::get_document_fixture() {
    local type="${1:-pdf}"
    local subtype="${2:-simple}"
    local doc_dir="$(fixture_helpers::get_document_fixture_path)"
    
    case "$type" in
        "pdf")
            case "$subtype" in
                "simple")
                    echo "$doc_dir/pdf/simple_text.pdf"
                    ;;
                "multipage")
                    echo "$doc_dir/pdf/multipage.pdf"
                    ;;
                "table")
                    echo "$doc_dir/pdf/table_document.pdf"
                    ;;
                "large")
                    echo "$doc_dir/pdf/large_document.pdf"
                    ;;
                "government")
                    echo "$doc_dir/samples/gao_report_sample.pdf"
                    ;;
                *)
                    echo "$doc_dir/pdf/simple_text.pdf"
                    ;;
            esac
            ;;
        "word")
            echo "$doc_dir/office/word/educational/mtsac_word_template.docx"
            ;;
        "excel")
            echo "$doc_dir/office/excel/educational/ou_sample_data.xlsx"
            ;;
        "json")
            echo "$doc_dir/test-data.json"
            ;;
        "html")
            echo "$doc_dir/web/article.html"
            ;;
        "corrupted")
            echo "$doc_dir/edge_cases/corrupted.json"
            ;;
        "empty")
            echo "$doc_dir/edge_cases/empty.pdf"
            ;;
        "malformed")
            echo "$doc_dir/edge_cases/malformed.csv"
            ;;
        *)
            echo "$doc_dir/pdf/simple_text.pdf"
            ;;
    esac
}

#######################################
# IMAGE FIXTURE HELPERS  
#######################################

# Get a test image for processing
fixture_helpers::get_image_fixture() {
    local type="${1:-synthetic}"
    local subtype="${2:-small}"
    local image_dir="$(fixture_helpers::get_image_fixture_path)"
    
    case "$type" in
        "synthetic")
            case "$subtype" in
                "small")
                    echo "$image_dir/dimensions/small/small-green.jpg"
                    ;;
                "medium")
                    echo "$image_dir/dimensions/medium/medium-blue.png"
                    ;;
                "large")
                    echo "$image_dir/dimensions/large/large-red.png"
                    ;;
                *)
                    echo "$image_dir/dimensions/small/small-green.jpg"
                    ;;
            esac
            ;;
        "real")
            case "$subtype" in
                "nature")
                    echo "$image_dir/real-world/nature/nature-landscape.jpg"
                    ;;
                "people")
                    echo "$image_dir/real-world/people/person-woman.jpg"
                    ;;
                "animals")
                    echo "$image_dir/real-world/animals/animal-cat.jpg"
                    ;;
                *)
                    echo "$image_dir/real-world/nature/nature-landscape.jpg"
                    ;;
            esac
            ;;
        "ocr")
            echo "$image_dir/ocr/images/1_simple_text.png"
            ;;
        *)
            echo "$image_dir/dimensions/small/small-green.jpg"
            ;;
    esac
}

#######################################
# WORKFLOW FIXTURE HELPERS
#######################################

# Get a workflow fixture for testing
fixture_helpers::get_workflow_fixture() {
    local platform="${1:-n8n}"
    local type="${2:-default}"
    local workflow_dir="$(fixture_helpers::get_workflow_fixture_path)"
    
    case "$platform" in
        "n8n")
            case "$type" in
                "whisper")
                    echo "$workflow_dir/n8n/n8n-whisper-transcription.json"
                    ;;
                *)
                    echo "$workflow_dir/n8n/n8n-workflow.json"
                    ;;
            esac
            ;;
        "comfyui")
            case "$type" in
                "ollama")
                    echo "$workflow_dir/comfyui/comfyui-ollama-guided.json"
                    ;;
                *)
                    echo "$workflow_dir/comfyui/comfyui-text-to-image.json"
                    ;;
            esac
            ;;
        "node-red")
            echo "$workflow_dir/node-red/node-red-workflow.json"
            ;;
        "huginn")
            echo "$workflow_dir/huginn/huginn-web-automation.json"
            ;;
        "windmill")
            echo "$workflow_dir/windmill/windmill-secure-processing.json"
            ;;
        "integration")
            echo "$workflow_dir/integration/multi-ai-pipeline.json"
            ;;
        *)
            echo "$workflow_dir/n8n/n8n-workflow.json"
            ;;
    esac
}

#######################################
# PROMPT FIXTURE HELPERS
#######################################

# Get test prompts for LLM testing
fixture_helpers::get_llm_test_prompts() {
    cat <<EOF
[
    {
        "id": "simple_greeting",
        "prompt": "Say hello in one word",
        "expected_pattern": "(?i)hello|hi|hey|greetings",
        "max_tokens": 10
    },
    {
        "id": "math_simple",
        "prompt": "What is 2+2? Answer with just the number.",
        "expected_pattern": "4|four",
        "max_tokens": 5
    },
    {
        "id": "completion",
        "prompt": "Complete this sentence: The sky is",
        "expected_pattern": "(?i)blue|cloudy|clear|grey|gray",
        "max_tokens": 20
    },
    {
        "id": "json_generation",
        "prompt": "Generate a JSON object with a single key 'status' and value 'ok'. Return only valid JSON.",
        "expected_pattern": "\\\\{.*\"status\".*:.*\"ok\".*\\\\}",
        "max_tokens": 30
    },
    {
        "id": "code_generation",
        "prompt": "Write a Python function that returns True. Use minimal code.",
        "expected_pattern": "def.*return.*True",
        "max_tokens": 50
    }
]
EOF
}

# Get a specific test prompt
fixture_helpers::get_llm_test_prompt() {
    local prompt_id="${1:-simple_greeting}"
    fixture_helpers::get_llm_test_prompts | jq -r ".[] | select(.id == \"$prompt_id\") | .prompt"
}

# Get expected pattern for a prompt
fixture_helpers::get_llm_expected_pattern() {
    local prompt_id="$1"
    fixture_helpers::get_llm_test_prompts | jq -r ".[] | select(.id == \"$prompt_id\") | .expected_pattern"
}

#######################################
# VALIDATION HELPERS
#######################################

# Check if a file exists and is readable
fixture_helpers::validate_fixture_file() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "ERROR: Fixture file not found: $file" >&2
        return 1
    fi
    
    if [[ ! -r "$file" ]]; then
        echo "ERROR: Fixture file not readable: $file" >&2
        return 1
    fi
    
    return 0
}

# Validate transcription accuracy (fuzzy match)
fixture_helpers::validate_transcription() {
    local actual="$1"
    local expected="$2"
    local threshold="${3:-0.7}"  # 70% similarity by default
    
    # Convert to lowercase for comparison
    actual=$(echo "$actual" | tr '[:upper:]' '[:lower:]')
    expected=$(echo "$expected" | tr '[:upper:]' '[:lower:]')
    
    # Simple word-based similarity check
    local actual_words=($actual)
    local expected_words=($expected)
    local matches=0
    
    for word in "${expected_words[@]}"; do
        if [[ " ${actual_words[*]} " =~ " $word " ]]; then
            ((matches++))
        fi
    done
    
    local similarity
    if [[ ${#expected_words[@]} -gt 0 ]]; then
        # Calculate similarity as percentage (multiply by 100 for integer math)
        similarity=$((matches * 100 / ${#expected_words[@]}))
    else
        similarity=0
    fi
    
    # Convert threshold to percentage for comparison (e.g., 0.7 -> 70)
    local threshold_percent
    threshold_percent=$(awk "BEGIN{print int($threshold * 100)}")
    
    # Check if similarity meets threshold
    if [[ $similarity -ge $threshold_percent ]]; then
        return 0
    else
        echo "Transcription validation failed. Similarity: ${similarity}% (threshold: ${threshold_percent}%)" >&2
        echo "Expected: $expected" >&2
        echo "Actual: $actual" >&2
        return 1
    fi
}

# Validate JSON structure
fixture_helpers::validate_json_structure() {
    local json="$1"
    local expected_keys="${2:-}"
    
    # Check if valid JSON
    if ! echo "$json" | jq empty 2>/dev/null; then
        echo "ERROR: Invalid JSON structure" >&2
        return 1
    fi
    
    # Check for expected keys if provided
    if [[ -n "$expected_keys" ]]; then
        for key in $expected_keys; do
            if ! echo "$json" | jq -e "has(\"$key\")" >/dev/null 2>&1; then
                echo "ERROR: Missing expected key: $key" >&2
                return 1
            fi
        done
    fi
    
    return 0
}

# Validate LLM response against pattern
fixture_helpers::validate_llm_response() {
    local response="$1"
    local pattern="$2"
    
    if [[ "$response" =~ $pattern ]]; then
        return 0
    else
        echo "ERROR: Response doesn't match expected pattern" >&2
        echo "Pattern: $pattern" >&2
        echo "Response: $response" >&2
        return 1
    fi
}

#######################################
# NEGATIVE TEST HELPERS
#######################################

# Get path to negative test fixtures
fixture_helpers::get_negative_fixture_path() {
    echo "$FIXTURE_BASE/negative-tests"
}

# Get a fixture that should fail processing
fixture_helpers::get_negative_fixture() {
    local type="${1:-document}"
    local subtype="${2:-invalid}"
    local negative_dir="$(fixture_helpers::get_negative_fixture_path)"
    
    case "$type" in
        "audio")
            case "$subtype" in
                "empty")
                    echo "$negative_dir/audio/empty.mp3"
                    ;;
                "invalid")
                    echo "$negative_dir/audio/invalid_header.mp3"
                    ;;
                "truncated")
                    echo "$negative_dir/audio/truncated.mp3"
                    ;;
                "random")
                    echo "$negative_dir/audio/random_binary.wav"
                    ;;
                "text")
                    echo "$negative_dir/audio/text_as_audio.mp3"
                    ;;
                *)
                    echo "$negative_dir/audio/invalid_header.mp3"
                    ;;
            esac
            ;;
        "document")
            case "$subtype" in
                "empty")
                    echo "$negative_dir/documents/empty.pdf"
                    ;;
                "invalid_pdf")
                    echo "$negative_dir/documents/invalid_pdf.pdf"
                    ;;
                "malformed_json")
                    echo "$negative_dir/documents/malformed.json"
                    ;;
                "invalid_xml")
                    echo "$negative_dir/documents/invalid.xml"
                    ;;
                "corrupted_word")
                    echo "$negative_dir/documents/corrupted.docx"
                    ;;
                "circular_json")
                    echo "$negative_dir/documents/circular_reference.json"
                    ;;
                "binary")
                    echo "$negative_dir/documents/binary.pdf"
                    ;;
                *)
                    echo "$negative_dir/documents/malformed.json"
                    ;;
            esac
            ;;
        "image")
            case "$subtype" in
                "empty")
                    echo "$negative_dir/images/empty.jpg"
                    ;;
                "invalid")
                    echo "$negative_dir/images/invalid_jpeg.jpg"
                    ;;
                "truncated_png")
                    echo "$negative_dir/images/partial_png.png"
                    ;;
                "text")
                    echo "$negative_dir/images/text_as_image.png"
                    ;;
                "truncated_gif")
                    echo "$negative_dir/images/truncated.gif"
                    ;;
                "random")
                    echo "$negative_dir/images/random.jpg"
                    ;;
                *)
                    echo "$negative_dir/images/invalid_jpeg.jpg"
                    ;;
            esac
            ;;
        "edge")
            case "$subtype" in
                "zero")
                    echo "$negative_dir/edge-cases/zero_bytes.txt"
                    ;;
                "whitespace")
                    echo "$negative_dir/edge-cases/only_whitespace.txt"
                    ;;
                "null_bytes")
                    echo "$negative_dir/edge-cases/null_bytes.txt"
                    ;;
                "unicode")
                    echo "$negative_dir/edge-cases/unicode_stress.txt"
                    ;;
                "long_line")
                    echo "$negative_dir/edge-cases/long_single_line.txt"
                    ;;
                "nested")
                    echo "$negative_dir/edge-cases/nested_structure.json"
                    ;;
                *)
                    echo "$negative_dir/edge-cases/zero_bytes.txt"
                    ;;
            esac
            ;;
        *)
            echo "$negative_dir/documents/malformed.json"
            ;;
    esac
}

# Test that a service properly handles invalid input
fixture_helpers::test_negative_case() {
    local service_name="$1"
    local fixture_file="$2"
    local expected_behavior="${3:-error}"  # error, skip, or sanitize
    
    echo "Testing negative case for $service_name with $(basename "$fixture_file")"
    
    # Expected behaviors:
    # - error: Should return non-zero exit code
    # - skip: Should skip the file gracefully
    # - sanitize: Should process but sanitize/clean the input
    
    case "$expected_behavior" in
        "error")
            # Service should fail with error code
            return 0
            ;;
        "skip")
            # Service should skip but not crash
            return 0
            ;;
        "sanitize")
            # Service should handle and clean input
            return 0
            ;;
        *)
            echo "Unknown expected behavior: $expected_behavior" >&2
            return 1
            ;;
    esac
}

# Export all functions for use in tests
export -f fixture_helpers::get_audio_fixture_path
export -f fixture_helpers::get_document_fixture_path
export -f fixture_helpers::get_image_fixture_path
export -f fixture_helpers::get_workflow_fixture_path
export -f fixture_helpers::get_negative_fixture_path
export -f fixture_helpers::get_speech_audio_fixture
export -f fixture_helpers::get_expected_transcription
export -f fixture_helpers::get_document_fixture
export -f fixture_helpers::get_image_fixture
export -f fixture_helpers::get_workflow_fixture
export -f fixture_helpers::get_llm_test_prompts
export -f fixture_helpers::get_llm_test_prompt
export -f fixture_helpers::get_llm_expected_pattern
export -f fixture_helpers::validate_fixture_file
export -f fixture_helpers::validate_transcription
export -f fixture_helpers::validate_json_structure
export -f fixture_helpers::validate_llm_response
export -f fixture_helpers::get_negative_fixture
export -f fixture_helpers::test_negative_case