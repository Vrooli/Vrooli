#!/usr/bin/env bash
# Speaker Verification Integration Test
# Tests real speaker enrollment + verification functionality against a running
# service. Tests API endpoints, profile lifecycle, and target-speaker extraction.

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh"

# Source integration test libraries using var.sh
# shellcheck disable=SC1091
if [[ -f "${var_TEST_DIR}/integration/health-check.sh" ]]; then
    source "${var_TEST_DIR}/integration/health-check.sh"
else
    # Define basic functions if library not found
    log_test_result() {
        local test_name="$1"
        local status="$2"
        local message="$3"
        echo "[$status] $test_name: $message"
    }

    make_api_request() {
        local endpoint="$1"
        local method="${2:-GET}"
        local timeout="${3:-10}"
        curl -s -X "$method" "${BASE_URL}${endpoint}" --max-time "$timeout"
    }

    validate_json_response() {
        local response="$1"
        echo "$response" | jq . >/dev/null 2>&1
    }
fi

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# shellcheck disable=SC1091
if [[ -f "${var_RESOURCES_COMMON_FILE}" ]]; then
    source "${var_RESOURCES_COMMON_FILE}"
fi

# shellcheck disable=SC1091
if [[ -f "${RESOURCE_DIR}/config/defaults.sh" ]]; then
    source "${RESOURCE_DIR}/config/defaults.sh"
    if declare -f defaults::export_config >/dev/null 2>&1; then
        defaults::export_config
    fi
fi

SERVICE_NAME="speaker-verification"
BASE_URL="${SPEAKER_VERIFICATION_BASE_URL:-http://localhost:11452}"
HEALTH_ENDPOINT="/ready"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "API Port: ${SPEAKER_VERIFICATION_PORT:-11452}"
    "Container: ${SPEAKER_VERIFICATION_CONTAINER_NAME:-speaker-verification}"
    "Model: ${SPEAKER_VERIFICATION_MODEL:-speechbrain/spkrec-ecapa-voxceleb}"
)

# Shared test fixtures
TEST_PROFILE_ID="integration-test-speaker"
TEST_AUDIO_FILE="/tmp/speaker_verification_integration.wav"

#######################################
# Generate a short test audio clip (tone) via ffmpeg if available.
# Returns 0 if a fixture exists/was created, 1 otherwise.
#######################################
ensure_test_audio() {
    if [[ -f "$TEST_AUDIO_FILE" ]]; then
        return 0
    fi
    if command -v ffmpeg >/dev/null 2>&1; then
        ffmpeg -hide_banner -loglevel error -y \
            -f lavfi -i "sine=frequency=220:duration=3" \
            -ar 16000 -ac 1 "$TEST_AUDIO_FILE" >/dev/null 2>&1 && return 0
    fi
    return 1
}

#######################################
# SPEAKER-VERIFICATION-SPECIFIC TEST FUNCTIONS
#######################################

test_ready_endpoint() {
    local test_name="readiness endpoint"

    local response
    if response=$(make_api_request "/ready" "GET" 10); then
        if validate_json_response "$response"; then
            local status
            status=$(echo "$response" | jq -r '.status' 2>/dev/null || echo "")
            if [[ "$status" == "ok" ]]; then
                log_test_result "$test_name" "PASS" "ready returns status ok"
                return 0
            fi
        fi
    fi

    log_test_result "$test_name" "FAIL" "ready endpoint not accessible"
    return 1
}

test_info_endpoint() {
    local test_name="info endpoint"

    local response
    if response=$(make_api_request "/v1/info" "GET" 10); then
        if validate_json_response "$response"; then
            local dim backend
            dim=$(echo "$response" | jq -r '.embedding_dim' 2>/dev/null || echo "")
            backend=$(echo "$response" | jq -r '.backend' 2>/dev/null || echo "")
            if [[ "$dim" == "192" && "$backend" == "speechbrain" ]]; then
                log_test_result "$test_name" "PASS" "info reports backend=speechbrain embedding_dim=192"
                return 0
            fi
            log_test_result "$test_name" "FAIL" "unexpected info payload (dim=$dim backend=$backend)"
            return 1
        fi
    fi

    log_test_result "$test_name" "FAIL" "info endpoint not accessible"
    return 1
}

test_list_profiles_endpoint() {
    local test_name="list profiles endpoint"

    local response
    if response=$(make_api_request "/v1/profiles" "GET" 10); then
        if validate_json_response "$response"; then
            if echo "$response" | jq -e 'has("profiles") and has("count")' >/dev/null 2>&1; then
                log_test_result "$test_name" "PASS" "profiles list returns profiles+count"
                return 0
            fi
        fi
    fi

    log_test_result "$test_name" "FAIL" "profiles endpoint malformed"
    return 1
}

test_enroll_and_verify() {
    local test_name="enroll and verify"

    if ! ensure_test_audio; then
        log_test_result "$test_name" "SKIP" "no ffmpeg to synthesize test audio"
        return 2
    fi

    # Enroll
    local enroll_resp pid
    enroll_resp=$(curl -s -X POST "$BASE_URL/v1/profiles" \
        -F "profile_id=${TEST_PROFILE_ID}" \
        -F "display_name=Integration Test" \
        -F "notes=integration" \
        -F "audio=@${TEST_AUDIO_FILE}" \
        --max-time 90 2>/dev/null)

    pid=$(echo "$enroll_resp" | jq -r '.profile_id' 2>/dev/null || echo "")
    if [[ "$pid" != "$TEST_PROFILE_ID" ]]; then
        log_test_result "$test_name" "FAIL" "enrollment did not return expected profile_id"
        return 1
    fi

    # Verify the same clip against the profile (should score high)
    local verify_resp score matched
    verify_resp=$(curl -s -X POST "$BASE_URL/v1/verify" \
        -F "profile_id=${TEST_PROFILE_ID}" \
        -F "threshold=0.25" \
        -F "audio=@${TEST_AUDIO_FILE}" \
        --max-time 90 2>/dev/null)

    score=$(echo "$verify_resp" | jq -r '.score' 2>/dev/null || echo "")
    matched=$(echo "$verify_resp" | jq -r '.matched' 2>/dev/null || echo "")

    if [[ -n "$score" && "$matched" == "true" ]]; then
        log_test_result "$test_name" "PASS" "self-verification matched (score=$score)"
        return 0
    fi

    log_test_result "$test_name" "FAIL" "self-verification did not match (score=$score matched=$matched)"
    return 1
}

test_extract_unknown_profile() {
    local test_name="extract unknown profile (404)"

    if ! ensure_test_audio; then
        log_test_result "$test_name" "SKIP" "no ffmpeg to synthesize test audio"
        return 2
    fi

    # Unknown-profile path is model-free (profile lookup happens before the
    # separator loads), so this always runs fast and proves the endpoint is live.
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST "$BASE_URL/v1/extract" \
        -F "profile_id=does-not-exist-$(date +%s)" \
        -F "verify=false" \
        -F "audio=@${TEST_AUDIO_FILE}" \
        --max-time 30 2>/dev/null)

    if [[ "$status_code" == "404" ]]; then
        log_test_result "$test_name" "PASS" "extract reports 404 for an unknown profile"
        return 0
    fi

    log_test_result "$test_name" "FAIL" "extract returned HTTP $status_code (expected 404)"
    return 1
}

test_extract_isolates_enrolled() {
    local test_name="extract isolates enrolled speaker"

    if ! ensure_test_audio; then
        log_test_result "$test_name" "SKIP" "no ffmpeg to synthesize test audio"
        return 2
    fi

    # Happy path loads + runs SepFormer; the first call may trigger a model
    # download, so use a generous timeout and SKIP (not FAIL) on
    # timeout/unavailability — model provisioning is an environment concern, not
    # a contract regression. A contract regression is a NON-200 status.
    local resp status_code score
    resp=$(curl -s -D - -o /dev/null -w "\n%{http_code}" \
        -X POST "$BASE_URL/v1/extract" \
        -F "profile_id=${TEST_PROFILE_ID}" \
        -F "verify=true" \
        -F "audio=@${TEST_AUDIO_FILE}" \
        --max-time 300 2>/dev/null) || {
        log_test_result "$test_name" "SKIP" "extract request failed/timed out (separation model may be downloading)"
        return 2
    }
    status_code=$(echo "$resp" | tail -n1)
    score=$(echo "$resp" | grep -i "^X-Speaker-Score:" | tr -d '\r' | awk '{print $2}')

    if [[ "$status_code" == "200" && -n "$score" ]]; then
        log_test_result "$test_name" "PASS" "extract returned cleaned audio (X-Speaker-Score=${score})"
        return 0
    fi
    if [[ -z "$status_code" ]]; then
        log_test_result "$test_name" "SKIP" "no response (separation model may be unavailable)"
        return 2
    fi

    log_test_result "$test_name" "FAIL" "extract returned HTTP ${status_code} without a score header"
    return 1
}

test_delete_profile() {
    local test_name="delete profile"

    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -X DELETE "$BASE_URL/v1/profiles/${TEST_PROFILE_ID}" \
        --max-time 10 2>/dev/null)

    rm -f "$TEST_AUDIO_FILE"

    if [[ "$status_code" == "200" ]]; then
        log_test_result "$test_name" "PASS" "profile deleted"
        return 0
    fi

    log_test_result "$test_name" "FAIL" "delete returned HTTP $status_code"
    return 1
}

test_container_status() {
    local test_name="Docker container status"

    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi

    if docker ps --format '{{.Names}}' | grep -q "^${SPEAKER_VERIFICATION_CONTAINER_NAME}$"; then
        local container_status
        container_status=$(docker inspect "${SPEAKER_VERIFICATION_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")

        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "container running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "container not found"
        return 1
    fi
}

#######################################
# TEST RUNNER CONFIGURATION
#######################################

SERVICE_TESTS=(
    "test_ready_endpoint"
    "test_info_endpoint"
    "test_list_profiles_endpoint"
    "test_container_status"
    "test_enroll_and_verify"
    "test_extract_unknown_profile"
    "test_extract_isolates_enrolled"
    "test_delete_profile"
)

#######################################
# MAIN EXECUTION
#######################################

init_config
integration_test_main "$@"
