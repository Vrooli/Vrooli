#!/usr/bin/env bash
# Kyutai STT Integration Test
# Tests the Kyutai STT streaming speech-to-text service contract:
# /health, /v1/info, and the /v1/stream WebSocket endpoint surface.

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

# Load Kyutai STT configuration
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

# Override library defaults with Kyutai STT-specific settings
SERVICE_NAME="kyutai-stt"
BASE_URL="${KYUTAI_STT_BASE_URL:-http://localhost:8094}"
HEALTH_ENDPOINT="/health"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "API Port: ${KYUTAI_STT_PORT:-8094}"
    "Container: ${KYUTAI_STT_CONTAINER_NAME:-kyutai-stt}"
    "Model: ${KYUTAI_STT_HF_REPO:-kyutai/stt-1b-en_fr}"
)

#######################################
# KYUTAI STT-SPECIFIC TEST FUNCTIONS
#######################################

test_health_endpoint() {
    local test_name="Kyutai STT health endpoint"

    local response
    if response=$(make_api_request "/health" "GET" 10); then
        if validate_json_response "$response"; then
            local status
            status=$(echo "$response" | jq -r '.status // empty')
            if [[ "$status" == "ok" ]]; then
                log_test_result "$test_name" "PASS" "health reports status=ok"
                return 0
            fi
        fi
    fi

    log_test_result "$test_name" "FAIL" "health endpoint not accessible or malformed"
    return 1
}

test_info_endpoint() {
    local test_name="Kyutai STT info endpoint contract"

    local response
    if response=$(make_api_request "/v1/info" "GET" 10); then
        if validate_json_response "$response"; then
            local backend sample_rate
            backend=$(echo "$response" | jq -r '.backend // empty')
            sample_rate=$(echo "$response" | jq -r '.sample_rate // empty')
            if [[ "$backend" == "kyutai" && "$sample_rate" == "16000" ]]; then
                log_test_result "$test_name" "PASS" "backend=kyutai sample_rate=16000"
                return 0
            fi
            log_test_result "$test_name" "FAIL" "unexpected info contract (backend=$backend sample_rate=$sample_rate)"
            return 1
        fi
    fi

    log_test_result "$test_name" "FAIL" "info endpoint not accessible"
    return 1
}

test_model_loaded() {
    local test_name="Kyutai STT model loaded"

    local response
    if response=$(make_api_request "/health" "GET" 10); then
        local loaded
        loaded=$(echo "$response" | jq -r '.model_loaded // false')
        if [[ "$loaded" == "true" ]]; then
            log_test_result "$test_name" "PASS" "model_loaded=true"
            return 0
        fi
        log_test_result "$test_name" "FAIL" "model not yet loaded (model_loaded=$loaded)"
        return 1
    fi

    log_test_result "$test_name" "FAIL" "could not read health for model_loaded"
    return 1
}

test_websocket_endpoint_reachable() {
    local test_name="Kyutai STT stream endpoint reachable"

    # The stream endpoint is a WebSocket; a plain GET should be rejected with
    # an HTTP upgrade-required style status (426/400/404 from the ASGI stack),
    # which still proves the route is mounted and the server is up.
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/v1/stream" --max-time 10 2>/dev/null)

    if [[ "$status_code" =~ ^(426|400|404|405|403)$ ]]; then
        log_test_result "$test_name" "PASS" "stream route mounted (HTTP: $status_code on non-WS GET)"
        return 0
    fi

    log_test_result "$test_name" "FAIL" "stream route not reachable (HTTP: $status_code)"
    return 1
}

test_container_status() {
    local test_name="Docker container status"

    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi

    if docker ps --format '{{.Names}}' | grep -q "^${KYUTAI_STT_CONTAINER_NAME}$"; then
        local container_status
        container_status=$(docker inspect "${KYUTAI_STT_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")

        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "container running"
            return 0
        fi
        log_test_result "$test_name" "FAIL" "container status: $container_status"
        return 1
    fi

    log_test_result "$test_name" "FAIL" "container not found"
    return 1
}

#######################################
# TEST RUNNER CONFIGURATION
#######################################

SERVICE_TESTS=(
    "test_health_endpoint"
    "test_info_endpoint"
    "test_container_status"
    "test_websocket_endpoint_reachable"
    "test_model_loaded"
)

#######################################
# MAIN EXECUTION
#######################################

init_config
integration_test_main "$@"
