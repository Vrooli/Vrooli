#!/usr/bin/env bash
# Speaker Verification - Health Check Functions

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source health framework if available
if [[ -f "${APP_ROOT}/scripts/resources/lib/health-framework.sh" ]]; then
    # shellcheck disable=SC1091
    source "${APP_ROOT}/scripts/resources/lib/health-framework.sh"
fi

#######################################
# Get health configuration for the tiered check framework
# Returns: JSON health config on stdout
#######################################
speaker_verification::health::get_config() {
    cat <<EOF
{
    "container_name": "${SPEAKER_VERIFICATION_CONTAINER_NAME}",
    "checks": {
        "basic": "speaker_verification::health::basic_check",
        "advanced": "speaker_verification::health::advanced_check"
    }
}
EOF
}
export -f speaker_verification::health::get_config

#######################################
# Basic health check: liveness
# Returns: 0 if alive, 1 if not
#######################################
speaker_verification::health::basic_check() {
    if ! common::is_running; then
        echo "UNHEALTHY: Container not running"
        return 1
    fi

    if timeout 5 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/health" &>/dev/null; then
        echo "HEALTHY: Service responding"
        return 0
    else
        echo "UNHEALTHY: Health endpoint not responding"
        return 1
    fi
}
export -f speaker_verification::health::basic_check

#######################################
# Advanced health check: readiness + operational
# Returns: 0 if fully healthy, 1 if degraded/unhealthy
#######################################
speaker_verification::health::advanced_check() {
    # Check readiness (model loaded)
    local ready_response
    if ! ready_response=$(timeout 10 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/ready" 2>/dev/null); then
        echo "DEGRADED: Model not ready"
        return 1
    fi

    local model_loaded
    model_loaded=$(echo "$ready_response" | jq -r '.model_loaded // false' 2>/dev/null)
    if [[ "$model_loaded" != "true" ]]; then
        echo "DEGRADED: Model not loaded"
        return 1
    fi

    # Check profile store is accessible
    local store_ok
    store_ok=$(echo "$ready_response" | jq -r '.profile_store_ok // false' 2>/dev/null)
    if [[ "$store_ok" != "true" ]]; then
        echo "DEGRADED: Profile store not accessible"
        return 1
    fi

    echo "HEALTHY: Model loaded, store accessible"
    return 0
}
export -f speaker_verification::health::advanced_check

#######################################
# Run tiered health check
# Returns: HEALTHY, DEGRADED, or UNHEALTHY
#######################################
speaker_verification::health::check() {
    if command -v health::tiered_check &>/dev/null; then
        local config
        config=$(speaker_verification::health::get_config)
        health::tiered_check "$config"
    else
        # Fallback without framework
        if speaker_verification::health::basic_check &>/dev/null; then
            if speaker_verification::health::advanced_check &>/dev/null; then
                echo "HEALTHY"
                return 0
            else
                echo "DEGRADED"
                return 1
            fi
        else
            echo "UNHEALTHY"
            return 1
        fi
    fi
}
export -f speaker_verification::health::check

#######################################
# Get health status string for status output
# Returns: one of: not_installed, stopped, starting, healthy, degraded, unhealthy
#######################################
speaker_verification::health::get_status() {
    if ! common::container_exists; then
        echo "not_installed"
        return
    fi

    if ! common::is_running; then
        echo "stopped"
        return
    fi

    if ! speaker_verification::is_healthy; then
        echo "unhealthy"
        return
    fi

    if speaker_verification::is_ready; then
        echo "healthy"
    else
        echo "starting"
    fi
}
export -f speaker_verification::health::get_status
