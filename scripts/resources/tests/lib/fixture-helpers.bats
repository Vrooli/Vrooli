#!/usr/bin/env bats
# Test suite for fixture-helpers.sh

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/tests/lib"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Source the library being tested
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/fixture-helpers.sh"
    
    # Setup test environment
    export FIXTURE_BASE="${var_TEST_DIR}/fixtures"
}

teardown() {
    vrooli_cleanup_test
    unset FIXTURE_BASE
}

@test "fixture_helpers::get_audio_fixture_path - returns correct path" {
    run fixture_helpers::get_audio_fixture_path
    
    assert_success
    assert_output "$FIXTURE_BASE/audio"
}

@test "fixture_helpers::get_document_fixture_path - returns correct path" {
    run fixture_helpers::get_document_fixture_path
    
    assert_success
    assert_output "$FIXTURE_BASE/documents"
}

@test "fixture_helpers::get_image_fixture_path - returns correct path" {
    run fixture_helpers::get_image_fixture_path
    
    assert_success
    assert_output "$FIXTURE_BASE/images"
}

@test "fixture_helpers::get_workflow_fixture_path - returns correct path" {
    run fixture_helpers::get_workflow_fixture_path
    
    assert_success
    assert_output "$FIXTURE_BASE/workflows"
}

@test "fixture_helpers::get_speech_audio_fixture - returns correct file for type" {
    run fixture_helpers::get_speech_audio_fixture "short"
    
    assert_success
    assert_output "$FIXTURE_BASE/audio/speech_test_short.mp3"
    
    run fixture_helpers::get_speech_audio_fixture "whisper"
    
    assert_success
    assert_output "$FIXTURE_BASE/audio/whisper/test_speech.mp3"
}

@test "fixture_helpers::get_expected_transcription - returns known transcriptions" {
    run fixture_helpers::get_expected_transcription "speech_test_short.mp3"
    
    assert_success
    assert_output "this is a test of speech recognition"
    
    run fixture_helpers::get_expected_transcription "unknown.mp3"
    
    assert_success
    assert_output ""
}

@test "fixture_helpers::get_document_fixture - returns correct document paths" {
    run fixture_helpers::get_document_fixture "pdf" "simple"
    
    assert_success
    assert_output "$FIXTURE_BASE/documents/pdf/simple_text.pdf"
    
    run fixture_helpers::get_document_fixture "word"
    
    assert_success
    assert_output "$FIXTURE_BASE/documents/office/word/educational/mtsac_word_template.docx"
}

@test "fixture_helpers::get_image_fixture - returns correct image paths" {
    run fixture_helpers::get_image_fixture "synthetic" "small"
    
    assert_success
    assert_output "$FIXTURE_BASE/images/dimensions/small/small-green.jpg"
    
    run fixture_helpers::get_image_fixture "real" "nature"
    
    assert_success
    assert_output "$FIXTURE_BASE/images/real-world/nature/nature-landscape.jpg"
}

@test "fixture_helpers::get_workflow_fixture - returns correct workflow paths" {
    run fixture_helpers::get_workflow_fixture "node-red"

    assert_success
    assert_output "$FIXTURE_BASE/workflows/node-red/node-red-workflow.json"
    
    run fixture_helpers::get_workflow_fixture "comfyui"
    
    assert_success
    assert_output "$FIXTURE_BASE/workflows/comfyui/comfyui-text-to-image.json"
}

@test "fixture_helpers::get_llm_test_prompts - returns valid JSON" {
    run fixture_helpers::get_llm_test_prompts
    
    assert_success
    # Check if output is valid JSON
    echo "$output" | jq . >/dev/null 2>&1
}

@test "fixture_helpers::validate_fixture_file - validates existing files" {
    # Create a temporary test file
    local test_file="${BATS_TMPDIR}/test_file.txt"
    echo "test" > "$test_file"
    
    run fixture_helpers::validate_fixture_file "$test_file"
    
    assert_success
    
    # Test with non-existent file
    run fixture_helpers::validate_fixture_file "/nonexistent/file.txt"
    
    assert_failure
    assert_output --partial "not found"
    
    trash::safe_remove "$test_file" --test-cleanup
}

@test "fixture_helpers::validate_transcription - validates transcription accuracy" {
    run fixture_helpers::validate_transcription "hello world" "hello world" "1.0"
    
    assert_success
    
    run fixture_helpers::validate_transcription "hello world" "goodbye mars" "0.9"
    
    assert_failure
    assert_output --partial "validation failed"
}

@test "fixture_helpers::validate_json_structure - validates JSON" {
    local valid_json='{"key": "value", "number": 42}'
    
    run fixture_helpers::validate_json_structure "$valid_json" "key number"
    
    assert_success
    
    run fixture_helpers::validate_json_structure "$valid_json" "missing_key"
    
    assert_failure
    assert_output --partial "Missing expected key"
    
    run fixture_helpers::validate_json_structure "not json"
    
    assert_failure
    assert_output --partial "Invalid JSON"
}

@test "fixture_helpers::validate_llm_response - validates against pattern" {
    run fixture_helpers::validate_llm_response "hello world" "hello.*"
    
    assert_success
    
    run fixture_helpers::validate_llm_response "hello world" "goodbye.*"
    
    assert_failure
    assert_output --partial "doesn't match expected pattern"
}

@test "fixture_helpers::get_negative_fixture_path - returns correct path" {
    run fixture_helpers::get_negative_fixture_path
    
    assert_success
    assert_output "$FIXTURE_BASE/negative-tests"
}

@test "fixture_helpers::get_negative_fixture - returns negative test fixtures" {
    run fixture_helpers::get_negative_fixture "audio" "empty"
    
    assert_success
    assert_output "$FIXTURE_BASE/negative-tests/audio/empty.mp3"
    
    run fixture_helpers::get_negative_fixture "document" "malformed_json"
    
    assert_success
    assert_output "$FIXTURE_BASE/negative-tests/documents/malformed.json"
}

@test "fixture_helpers::test_negative_case - handles negative test cases" {
    run fixture_helpers::test_negative_case "test-service" "test.file" "error"
    
    assert_success
    
    run fixture_helpers::test_negative_case "test-service" "test.file" "unknown_behavior"
    
    assert_failure
    assert_output --partial "Unknown expected behavior"
}
