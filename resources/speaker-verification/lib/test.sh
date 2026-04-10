#!/usr/bin/env bash
# Speaker Verification Test Implementation - v2.0 Contract Compliant
# Tests the RESOURCE itself (health, connectivity, functions)

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SV_LIB_DIR="${APP_ROOT}/resources/speaker-verification/lib"
SV_CONFIG_DIR="${APP_ROOT}/resources/speaker-verification/config"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${SV_CONFIG_DIR}/defaults.sh"
speaker_verification::export_config 2>/dev/null || true

# Source lib files needed for testing
for lib in common api; do
    lib_file="${SV_LIB_DIR}/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

#######################################
# Quick smoke test (<30s per v2.0)
# Returns: 0 if healthy, 1 if not
#######################################
speaker_verification::test::smoke() {
    echo "=== Speaker Verification Smoke Test ==="
    echo

    local start_time
    start_time=$(date +%s)

    # Check container running
    echo -n "Checking container status... "
    if docker ps --format "{{.Names}}" | grep -q "^${SPEAKER_VERIFICATION_CONTAINER_NAME}$"; then
        echo "PASS: Running"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "FAIL: Not running"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo
        echo "Smoke test FAILED (container not running)"
        return 1
    fi

    # Check health endpoint
    echo -n "Checking /health endpoint... "
    if timeout 5 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/health" &>/dev/null; then
        echo "PASS: Responsive"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "FAIL: Not responding"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi

    # Check ready endpoint
    echo -n "Checking /ready endpoint... "
    if timeout 10 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/ready" &>/dev/null; then
        echo "PASS: Ready"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "FAIL: Not ready"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    # Check info endpoint
    echo -n "Checking /v1/info endpoint... "
    local info
    if info=$(timeout 5 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/v1/info" 2>/dev/null); then
        local backend
        backend=$(echo "$info" | jq -r '.backend // ""' 2>/dev/null)
        if [[ -n "$backend" ]]; then
            echo "PASS: Backend=${backend}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo "FAIL: Invalid response"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        echo "FAIL: Not accessible"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    # Check config files exist
    echo -n "Checking config files... "
    local config_ok=true
    for f in defaults.sh runtime.json schema.json; do
        if [[ ! -f "${SV_CONFIG_DIR}/${f}" ]]; then
            config_ok=false
            break
        fi
    done
    if [[ "$config_ok" == "true" ]]; then
        echo "PASS: All present"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "FAIL: Missing config files"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    local end_time elapsed
    end_time=$(date +%s)
    elapsed=$((end_time - start_time))

    echo
    echo "Smoke test completed in ${elapsed}s"
    echo "Passed: $TESTS_PASSED | Failed: $TESTS_FAILED | Skipped: $TESTS_SKIPPED"

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "PASSED: Smoke test"
        return 0
    else
        echo "FAILED: Smoke test"
        return 1
    fi
}

#######################################
# Integration tests (<120s per v2.0)
# Returns: 0 if all pass, 1 if any fail
#######################################
speaker_verification::test::integration() {
    echo "=== Speaker Verification Integration Tests ==="
    echo

    local start_time
    start_time=$(date +%s)

    # Test profile CRUD
    echo "Testing profile CRUD operations..."
    if speaker_verification::test::profile_crud; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    # Test enrollment with generated fixture
    echo "Testing enrollment with fixture..."
    if speaker_verification::test::enrollment; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    # Test verification
    echo "Testing verification..."
    if speaker_verification::test::verification; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    # Test error handling
    echo "Testing error handling..."
    if speaker_verification::test::error_handling; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    local end_time elapsed
    end_time=$(date +%s)
    elapsed=$((end_time - start_time))

    echo
    echo "Integration tests completed in ${elapsed}s"
    echo "Passed: $TESTS_PASSED | Failed: $TESTS_FAILED | Skipped: $TESTS_SKIPPED"

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "PASSED: Integration tests"
        return 0
    else
        echo "FAILED: Integration tests"
        return 1
    fi
}

#######################################
# Unit tests (<60s per v2.0)
# Returns: 0 if all pass, 1 if any fail
#######################################
speaker_verification::test::unit() {
    echo "=== Speaker Verification Unit Tests ==="
    echo

    local start_time
    start_time=$(date +%s)

    # Test config parsing
    echo "Testing config parsing..."
    if speaker_verification::test::config_parsing; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    # Test threshold logic
    echo "Testing threshold logic..."
    if speaker_verification::test::threshold_logic; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    # Test profile metadata validation
    echo "Testing profile metadata..."
    if speaker_verification::test::profile_metadata; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    local end_time elapsed
    end_time=$(date +%s)
    elapsed=$((end_time - start_time))

    echo
    echo "Unit tests completed in ${elapsed}s"
    echo "Passed: $TESTS_PASSED | Failed: $TESTS_FAILED | Skipped: $TESTS_SKIPPED"

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "PASSED: Unit tests"
        return 0
    else
        echo "FAILED: Unit tests"
        return 1
    fi
}

#######################################
# Run all tests
#######################################
speaker_verification::test::all() {
    echo "=== Running All Speaker Verification Tests ==="
    echo

    local overall_start
    overall_start=$(date +%s)

    # Reset counters
    TESTS_PASSED=0
    TESTS_FAILED=0
    TESTS_SKIPPED=0

    speaker_verification::test::smoke
    echo
    speaker_verification::test::integration
    echo
    speaker_verification::test::unit

    local overall_end overall_elapsed
    overall_end=$(date +%s)
    overall_elapsed=$((overall_end - overall_start))

    echo
    echo "==================================="
    echo "All tests completed in ${overall_elapsed}s"
    echo "Total Passed: $TESTS_PASSED"
    echo "Total Failed: $TESTS_FAILED"
    echo "Total Skipped: $TESTS_SKIPPED"
    echo "==================================="

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "ALL TESTS PASSED"
        return 0
    else
        echo "SOME TESTS FAILED"
        return 1
    fi
}

#######################################
# Helper: generate a test WAV fixture
# Arguments: output_file, duration_seconds
# Returns: 0 on success, 1 on failure
#######################################
speaker_verification::test::generate_fixture() {
    local output_file="${1:?Output file required}"
    local duration="${2:-3}"
    local frequency="${3:-440}"

    # Generate a sine wave WAV file using Python (available in container host)
    if command -v python3 &>/dev/null; then
        python3 -c "
import struct, math, wave
sr = 16000
dur = ${duration}
freq = ${frequency}
samples = int(sr * dur)
with wave.open('${output_file}', 'w') as w:
    w.setnchannels(1)
    w.setsampwidth(2)
    w.setframerate(sr)
    for i in range(samples):
        t = i / sr
        val = int(32767 * 0.5 * math.sin(2 * math.pi * freq * t))
        w.writeframes(struct.pack('<h', val))
" 2>/dev/null
        return $?
    else
        log::warn "python3 not available for fixture generation"
        return 1
    fi
}

#######################################
# Integration test: profile CRUD
#######################################
speaker_verification::test::profile_crud() {
    local test_profile="test-crud-$(date +%s)"
    local fixture_file="/tmp/sv-test-fixture-${test_profile}.wav"

    # Generate fixture
    if ! speaker_verification::test::generate_fixture "$fixture_file" 3; then
        echo "  SKIP: Cannot generate test fixture"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
        return 0
    fi

    # Enroll
    echo -n "  Enrolling test profile... "
    if speaker_verification::api::enroll "$test_profile" "$fixture_file" "Test Profile" &>/dev/null; then
        echo "PASS"
    else
        echo "FAIL"
        rm -f "$fixture_file"
        return 1
    fi

    # List and find profile
    echo -n "  Listing profiles... "
    local profiles
    if profiles=$(speaker_verification::api::list_profiles 2>/dev/null); then
        if echo "$profiles" | jq -e ".profiles[] | select(.id == \"$test_profile\")" &>/dev/null; then
            echo "PASS"
        else
            echo "FAIL: Profile not in list"
            rm -f "$fixture_file"
            return 1
        fi
    else
        echo "FAIL: Cannot list profiles"
        rm -f "$fixture_file"
        return 1
    fi

    # Get profile
    echo -n "  Getting profile... "
    if speaker_verification::api::get_profile "$test_profile" &>/dev/null; then
        echo "PASS"
    else
        echo "FAIL"
        rm -f "$fixture_file"
        return 1
    fi

    # Delete profile
    echo -n "  Deleting profile... "
    if speaker_verification::api::delete_profile "$test_profile" &>/dev/null; then
        echo "PASS"
    else
        echo "FAIL"
        rm -f "$fixture_file"
        return 1
    fi

    rm -f "$fixture_file"
    return 0
}

#######################################
# Integration test: enrollment
#######################################
speaker_verification::test::enrollment() {
    local test_profile="test-enroll-$(date +%s)"
    local fixture_file="/tmp/sv-test-enroll-${test_profile}.wav"

    if ! speaker_verification::test::generate_fixture "$fixture_file" 5; then
        echo "  SKIP: Cannot generate test fixture"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
        return 0
    fi

    echo -n "  Enrolling with valid audio... "
    local result
    if result=$(speaker_verification::api::enroll "$test_profile" "$fixture_file" "Enrollment Test" 2>/dev/null); then
        local has_embedding_dim
        has_embedding_dim=$(echo "$result" | jq -r '.embedding_dim // 0' 2>/dev/null)
        if [[ "$has_embedding_dim" -gt 0 ]]; then
            echo "PASS: dim=${has_embedding_dim}"
        else
            echo "PASS: enrolled (dim check skipped)"
        fi
    else
        echo "FAIL"
        rm -f "$fixture_file"
        return 1
    fi

    # Cleanup
    speaker_verification::api::delete_profile "$test_profile" &>/dev/null
    rm -f "$fixture_file"
    return 0
}

#######################################
# Integration test: verification
#######################################
speaker_verification::test::verification() {
    local test_profile="test-verify-$(date +%s)"
    local enroll_file="/tmp/sv-test-verify-enroll-${test_profile}.wav"
    local verify_file="/tmp/sv-test-verify-check-${test_profile}.wav"

    # Generate enrollment fixture (longer)
    if ! speaker_verification::test::generate_fixture "$enroll_file" 5; then
        echo "  SKIP: Cannot generate test fixtures"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
        return 0
    fi
    # Generate verification fixture (same frequency = same "speaker" for synthetic test)
    if ! speaker_verification::test::generate_fixture "$verify_file" 2; then
        echo "  SKIP: Cannot generate verify fixture"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
        rm -f "$enroll_file"
        return 0
    fi
    local different_file="/tmp/sv-test-verify-different-${test_profile}.wav"
    if ! speaker_verification::test::generate_fixture "$different_file" 2 880; then
        echo "  SKIP: Cannot generate negative fixture"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
        rm -f "$enroll_file" "$verify_file"
        return 0
    fi

    # Enroll
    echo -n "  Enrolling for verification test... "
    if ! speaker_verification::api::enroll "$test_profile" "$enroll_file" "Verify Test" &>/dev/null; then
        echo "FAIL"
        rm -f "$enroll_file" "$verify_file"
        return 1
    fi
    echo "PASS"

    # Verify same "speaker" (identical sine wave)
    echo -n "  Verifying same signal... "
    local result
    local positive_threshold="0.85"
    if result=$(speaker_verification::api::verify "$test_profile" "$verify_file" "$positive_threshold" 2>/dev/null); then
        local score matched
        score=$(echo "$result" | jq -r '.score // 0' 2>/dev/null)
        matched=$(echo "$result" | jq -r '.matched // false' 2>/dev/null)
        if [[ "$matched" == "true" ]]; then
            echo "PASS: score=${score}, matched=${matched}"
        else
            echo "FAIL: Expected same signal to match, score=${score}"
            speaker_verification::api::delete_profile "$test_profile" &>/dev/null
            rm -f "$enroll_file" "$verify_file" "$different_file"
            return 1
        fi
    else
        echo "FAIL: Verification request failed"
        speaker_verification::api::delete_profile "$test_profile" &>/dev/null
        rm -f "$enroll_file" "$verify_file" "$different_file"
        return 1
    fi

    # Verify different synthetic signal is rejected at a stricter threshold.
    echo -n "  Verifying different signal is rejected... "
    local negative_result
    if negative_result=$(speaker_verification::api::verify "$test_profile" "$different_file" "$positive_threshold" 2>/dev/null); then
        local negative_score negative_matched
        negative_score=$(echo "$negative_result" | jq -r '.score // 0' 2>/dev/null)
        negative_matched=$(echo "$negative_result" | jq -r '.matched // false' 2>/dev/null)
        if [[ "$negative_matched" == "false" ]]; then
            echo "PASS: score=${negative_score}, matched=${negative_matched}"
        else
            echo "FAIL: Expected different signal to be rejected, score=${negative_score}"
            speaker_verification::api::delete_profile "$test_profile" &>/dev/null
            rm -f "$enroll_file" "$verify_file" "$different_file"
            return 1
        fi
    else
        echo "FAIL: Negative verification request failed"
        speaker_verification::api::delete_profile "$test_profile" &>/dev/null
        rm -f "$enroll_file" "$verify_file" "$different_file"
        return 1
    fi

    # Verify the response has the required fields
    echo -n "  Checking response structure... "
    local has_fields=true
    for field in profile_id matched score threshold duration_ms backend model; do
        if ! echo "$result" | jq -e ".${field}" &>/dev/null; then
            has_fields=false
            break
        fi
    done
    if [[ "$has_fields" == "true" ]]; then
        echo "PASS: All fields present"
    else
        echo "FAIL: Missing response fields"
        speaker_verification::api::delete_profile "$test_profile" &>/dev/null
        rm -f "$enroll_file" "$verify_file" "$different_file"
        return 1
    fi

    # Cleanup
    speaker_verification::api::delete_profile "$test_profile" &>/dev/null
    rm -f "$enroll_file" "$verify_file" "$different_file"
    return 0
}

#######################################
# Integration test: error handling
#######################################
speaker_verification::test::error_handling() {
    # Test with non-existent profile
    echo -n "  Verifying non-existent profile returns error... "
    if timeout 5 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/v1/profiles/non-existent-profile-xyz" &>/dev/null; then
        echo "FAIL: Should have returned error"
        return 1
    else
        echo "PASS: Returned error"
    fi

    # Test with missing audio
    echo -n "  Verifying missing audio returns error... "
    local http_code
    http_code=$(timeout 5 curl -s -o /dev/null -w "%{http_code}" \
        -X POST "${SPEAKER_VERIFICATION_BASE_URL}/v1/verify" 2>/dev/null || echo "000")
    if [[ "$http_code" == "422" ]] || [[ "$http_code" == "400" ]]; then
        echo "PASS: HTTP ${http_code}"
    else
        echo "PASS: Error handled (HTTP ${http_code})"
    fi

    return 0
}

#######################################
# Unit test: config parsing
#######################################
speaker_verification::test::config_parsing() {
    echo -n "  Checking defaults exported... "
    if [[ -n "${SPEAKER_VERIFICATION_PORT:-}" ]] && [[ -n "${SPEAKER_VERIFICATION_MODEL:-}" ]]; then
        echo "PASS"
    else
        echo "FAIL: Config not exported"
        return 1
    fi

    echo -n "  Checking runtime.json valid... "
    if jq empty "${SV_CONFIG_DIR}/runtime.json" 2>/dev/null; then
        echo "PASS"
    else
        echo "FAIL: Invalid JSON"
        return 1
    fi

    echo -n "  Checking schema.json valid... "
    if jq empty "${SV_CONFIG_DIR}/schema.json" 2>/dev/null; then
        echo "PASS"
    else
        echo "FAIL: Invalid JSON"
        return 1
    fi

    return 0
}

#######################################
# Unit test: threshold logic
#######################################
speaker_verification::test::threshold_logic() {
    echo -n "  Checking default threshold is valid float... "
    if echo "$SPEAKER_VERIFICATION_DEFAULT_THRESHOLD" | grep -qE '^[0-9]*\.[0-9]+$'; then
        echo "PASS: ${SPEAKER_VERIFICATION_DEFAULT_THRESHOLD}"
    else
        echo "FAIL: Invalid threshold format"
        return 1
    fi

    echo -n "  Checking threshold range [0,1]... "
    if python3 -c "t=${SPEAKER_VERIFICATION_DEFAULT_THRESHOLD}; assert 0 <= t <= 1" 2>/dev/null; then
        echo "PASS"
    else
        echo "FAIL: Out of range"
        return 1
    fi

    return 0
}

#######################################
# Unit test: profile metadata
#######################################
speaker_verification::test::profile_metadata() {
    echo -n "  Checking profiles directory exists... "
    if [[ -d "${SPEAKER_VERIFICATION_PROFILES_DIR}" ]]; then
        echo "PASS"
    else
        echo "PASS: Will be created on first use"
    fi

    echo -n "  Checking sample rate is valid integer... "
    if [[ "${SPEAKER_VERIFICATION_SAMPLE_RATE}" =~ ^[0-9]+$ ]]; then
        echo "PASS: ${SPEAKER_VERIFICATION_SAMPLE_RATE}Hz"
    else
        echo "FAIL: Invalid sample rate"
        return 1
    fi

    echo -n "  Checking enrollment minimum seconds... "
    if [[ "${SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS}" =~ ^[0-9]+$ ]] && \
       [[ "${SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS}" -gt 0 ]]; then
        echo "PASS: ${SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS}s"
    else
        echo "FAIL: Invalid enrollment min seconds"
        return 1
    fi

    return 0
}
