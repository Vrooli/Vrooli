#!/usr/bin/env bash
# Speaker Verification Core Functions - v2.0 Contract Compliant
# Combines docker, api, and status functionality

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SV_LIB_DIR="${APP_ROOT}/resources/speaker-verification/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source shared frameworks (conditional - some may not exist in all environments)
for framework in docker-utils.sh http-utils.sh status-engine.sh health-framework.sh init-framework.sh wait-utils.sh; do
    if [[ -f "${var_SCRIPTS_RESOURCES_LIB_DIR}/${framework}" ]]; then
        # shellcheck disable=SC1090
        source "${var_SCRIPTS_RESOURCES_LIB_DIR}/${framework}"
    fi
done

#######################################
# Get initialization configuration
# Returns: JSON configuration for init framework
#######################################
speaker_verification::get_init_config() {
    local env_config='{
        "SPEAKER_VERIFICATION_DEVICE": "'"${SPEAKER_VERIFICATION_DEVICE}"'",
        "SPEAKER_VERIFICATION_MODEL": "'"${SPEAKER_VERIFICATION_MODEL}"'",
        "SPEAKER_VERIFICATION_DEFAULT_THRESHOLD": "'"${SPEAKER_VERIFICATION_DEFAULT_THRESHOLD}"'",
        "SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS": "'"${SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS}"'",
        "SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS": "'"${SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS}"'",
        "SPEAKER_VERIFICATION_SAMPLE_RATE": "'"${SPEAKER_VERIFICATION_SAMPLE_RATE}"'",
        "SPEAKER_VERIFICATION_MAX_UPLOAD_MB": "'"${SPEAKER_VERIFICATION_MAX_UPLOAD_MB}"'"
    }'

    local volumes_array="[\"${SPEAKER_VERIFICATION_PROFILES_DIR}:/data/profiles:z\",\"${SPEAKER_VERIFICATION_CACHE_DIR}:/data/cache:z\",\"${SPEAKER_VERIFICATION_LOG_DIR}:/data/logs:z\"]"

    local config='{
        "resource_name": "speaker-verification",
        "container_name": "'"${SPEAKER_VERIFICATION_CONTAINER_NAME}"'",
        "data_dir": "'"${SPEAKER_VERIFICATION_DATA_DIR}"'",
        "port": '"${SPEAKER_VERIFICATION_PORT}"',
        "image": "'"${SPEAKER_VERIFICATION_IMAGE}"'",
        "env_vars": '"${env_config}"',
        "volumes": '"${volumes_array}"',
        "networks": ["'"${SPEAKER_VERIFICATION_NETWORK_NAME}"'"],
        "healthcheck": {
            "test": ["CMD", "curl", "-f", "http://localhost:8891/health"],
            "interval": "30s",
            "timeout": "10s",
            "retries": 5,
            "start_period": "120s"
        },
        "restart_policy": "unless-stopped"
    }'

    echo "$config"
}
export -f speaker_verification::get_init_config

#######################################
# Check basic health for smoke testing
# Returns: 0 if healthy, 1 if not
#######################################
speaker_verification::check_basic_health() {
    if ! common::is_running; then
        log::error "Container is not running"
        return 1
    fi
    if ! speaker_verification::is_healthy; then
        log::error "Health check failed"
        return 1
    fi
    return 0
}
export -f speaker_verification::check_basic_health

#######################################
# Display connection information
#######################################
speaker_verification::display_connection_info() {
    echo
    echo "Speaker Verification Connection Info:"
    echo "  API:    ${SPEAKER_VERIFICATION_BASE_URL}"
    echo "  Health: ${SPEAKER_VERIFICATION_BASE_URL}/health"
    echo "  Ready:  ${SPEAKER_VERIFICATION_BASE_URL}/ready"
    echo "  Info:   ${SPEAKER_VERIFICATION_BASE_URL}/v1/info"
    echo
}
export -f speaker_verification::display_connection_info
